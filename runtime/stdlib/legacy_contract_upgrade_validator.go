/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package stdlib

import (
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

type LegacyContractUpdateValidator struct {
	TypeComparator

	newElaboration                           *sema.Elaboration
	currentRestrictedTypeUpgradeRestrictions []*ast.NominalType

	underlyingUpdateValidator *ContractUpdateValidator
}

// NewContractUpdateValidator initializes and returns a validator, without performing any validation.
// Invoke the `Validate()` method of the validator returned, to start validating the contract.
func NewLegacyContractUpdateValidator(
	location common.Location,
	contractName string,
	oldProgram *ast.Program,
	newProgram *ast.Program,
	newElaboration *sema.Elaboration,
) *LegacyContractUpdateValidator {

	underlyingValidator := NewContractUpdateValidator(location, contractName, oldProgram, newProgram)

	return &LegacyContractUpdateValidator{
		underlyingUpdateValidator: underlyingValidator,
		newElaboration:            newElaboration,
	}
}

var _ UpdateValidator = &LegacyContractUpdateValidator{}

func (validator *LegacyContractUpdateValidator) getCurrentDeclaration() ast.Declaration {
	return validator.underlyingUpdateValidator.getCurrentDeclaration()
}

func (validator *LegacyContractUpdateValidator) setCurrentDeclaration(decl ast.Declaration) {
	validator.underlyingUpdateValidator.setCurrentDeclaration(decl)
}

// Validate validates the contract update, and returns an error if it is an invalid update.
func (validator *LegacyContractUpdateValidator) Validate() error {
	underlyingValidator := validator.underlyingUpdateValidator

	oldRootDecl := getRootDeclaration(validator, underlyingValidator.oldProgram)
	if underlyingValidator.hasErrors() {
		return underlyingValidator.getContractUpdateError()
	}

	newRootDecl := getRootDeclaration(validator, underlyingValidator.newProgram)
	if underlyingValidator.hasErrors() {
		return underlyingValidator.getContractUpdateError()
	}

	validator.TypeComparator.RootDeclIdentifier = newRootDecl.DeclarationIdentifier()

	checkDeclarationUpdatability(validator, oldRootDecl, newRootDecl)

	if underlyingValidator.hasErrors() {
		return underlyingValidator.getContractUpdateError()
	}

	return nil
}

func (validator *LegacyContractUpdateValidator) report(err error) {
	validator.underlyingUpdateValidator.report(err)
}

func (validator *LegacyContractUpdateValidator) idOfQualifiedType(typ *ast.NominalType) common.TypeID {

	qualifiedString := typ.String()

	// working under the assumption that any new program we are validating already typechecks,
	// any nominal type must fall into one of three cases:
	// 1) type qualified by an import (e.g `C.R` where `C` is an imported type)
	// 2) type qualified by the root declaration (e.g `C.R` where `C` is the root contract or contract interface of the new contract)
	// 3) unqualified type (e.g. `R`, but declared inside `C`)
	//
	// in case 3, we prepend the root declaration identifer with a `.` to the type's string to get its qualified name,
	// and in 1 and 2 we don't need to do anything
	typIdentifier := typ.Identifier.Identifier
	rootIdentifier := validator.TypeComparator.RootDeclIdentifier.Identifier

	if typIdentifier != rootIdentifier { // &&
		// && validator.TypeComparator.foundIdentifierImportLocations[typ.Identifier.Identifier] == nil
		qualifiedString = fmt.Sprintf("%s.%s", rootIdentifier, qualifiedString)

	}
	return common.NewTypeIDFromQualifiedName(nil, validator.underlyingUpdateValidator.location, qualifiedString)
}

func (validator *LegacyContractUpdateValidator) getEntitlementType(entitlement *ast.NominalType) *sema.EntitlementType {
	typeID := validator.idOfQualifiedType(entitlement)
	return validator.newElaboration.EntitlementType(typeID)
}

func (validator *LegacyContractUpdateValidator) getEntitlementSetAccess(entitlementSet ast.EntitlementSet) sema.EntitlementSetAccess {
	var entitlements []*sema.EntitlementType

	for _, entitlement := range entitlementSet.Entitlements() {
		entitlements = append(entitlements, validator.getEntitlementType(entitlement))
	}

	entitlementSetKind := sema.Conjunction
	if entitlementSet.Separator() == ast.Disjunction {
		entitlementSetKind = sema.Disjunction
	}

	return sema.NewEntitlementSetAccess(entitlements, entitlementSetKind)
}

func (validator *LegacyContractUpdateValidator) getCompositeType(composite *ast.NominalType) *sema.CompositeType {
	typeID := validator.idOfQualifiedType(composite)
	return validator.newElaboration.CompositeType(typeID)
}

func (validator *LegacyContractUpdateValidator) getInterfaceType(intf *ast.NominalType) *sema.InterfaceType {
	typeID := validator.idOfQualifiedType(intf)
	return validator.newElaboration.InterfaceType(typeID)
}

func (validator *LegacyContractUpdateValidator) getIntersectedInterfaces(intersection []*ast.NominalType) (intfs []*sema.InterfaceType) {
	for _, intf := range intersection {
		intfs = append(intfs, validator.getInterfaceType(intf))
	}
	return
}

func (validator *LegacyContractUpdateValidator) requirePermitsAccess(
	expected sema.Access,
	found sema.EntitlementSetAccess,
	foundType ast.Type,
) error {
	if !found.PermitsAccess(expected) {
		return &AuthorizationMismatchError{
			FoundAuthorization:    found,
			ExpectedAuthorization: expected,
			Range:                 ast.NewUnmeteredRangeFromPositioned(foundType),
		}
	}
	return nil
}

func (validator *LegacyContractUpdateValidator) expectedAuthorizationOfComposite(composite *ast.NominalType) sema.Access {
	// if this field is set, we are currently upgrading a formerly legacy restricted type into a reference to a composite
	// in this case, the expected entitlements are based not on the underlying composite type,
	// but instead the types previously in the restriction set
	if validator.currentRestrictedTypeUpgradeRestrictions != nil {
		return validator.expectedAuthorizationOfIntersection(validator.currentRestrictedTypeUpgradeRestrictions)
	}

	compositeType := validator.getCompositeType(composite)

	if compositeType == nil {
		return sema.UnauthorizedAccess
	}

	supportedEntitlements := compositeType.SupportedEntitlements()
	return sema.NewAccessFromEntitlementSet(supportedEntitlements, sema.Conjunction)
}

func (validator *LegacyContractUpdateValidator) expectedAuthorizationOfIntersection(intersectionTypes []*ast.NominalType) sema.Access {

	// a reference to an intersection (or restricted) type is granted entitlements based on the intersected interfaces,
	// ignoring the legacy restricted type, as an intersection type appearing in the new contract means it must have originally
	// been a restricted type with no legacy type
	interfaces := validator.getIntersectedInterfaces(intersectionTypes)

	supportedEntitlements := orderedmap.New[sema.EntitlementOrderedSet](0)

	for _, interfaceType := range interfaces {
		supportedEntitlements.SetAll(interfaceType.SupportedEntitlements())
	}

	return sema.NewAccessFromEntitlementSet(supportedEntitlements, sema.Conjunction)
}

func (validator *LegacyContractUpdateValidator) checkEntitlementsUpgrade(
	oldType *ast.ReferenceType,
	newType *ast.ReferenceType,
) error {
	newAuthorization := newType.Authorization
	newEntitlementSet, isEntitlementsSet := newAuthorization.(ast.EntitlementSet)
	foundEntitlementSet := validator.getEntitlementSetAccess(newEntitlementSet)

	// if the new authorization is not an entitlements set, there's nothing to check here
	if !isEntitlementsSet {
		return nil
	}

	switch newReferencedType := newType.Type.(type) {
	// a lone nominal type must be a composite
	case *ast.NominalType:
		expectedAccess := validator.expectedAuthorizationOfComposite(newReferencedType)
		return validator.requirePermitsAccess(expectedAccess, foundEntitlementSet, newReferencedType)

	case *ast.IntersectionType:
		expectedAccess := validator.expectedAuthorizationOfIntersection(newReferencedType.Types)
		return validator.requirePermitsAccess(expectedAccess, foundEntitlementSet, newReferencedType)
	}

	return nil
}

func (validator *LegacyContractUpdateValidator) checkTypeUpgradability(oldType ast.Type, newType ast.Type) error {

	switch oldType := oldType.(type) {
	case *ast.OptionalType:
		if newOptional, isOptional := newType.(*ast.OptionalType); isOptional {
			return validator.checkTypeUpgradability(oldType.Type, newOptional.Type)
		}
	case *ast.ReferenceType:
		if newReference, isReference := newType.(*ast.ReferenceType); isReference {
			err := validator.checkTypeUpgradability(oldType.Type, newReference.Type)
			if err != nil {
				return err
			}

			if newReference.Authorization != nil {
				return validator.checkEntitlementsUpgrade(oldType, newReference)

			}
			return nil
		}
	case *ast.IntersectionType:
		// intersection types cannot be upgraded unless they have a legacy restricted type,
		// in which case they must be upgraded according to the migration rules: i.e. R{I} -> R
		if oldType.LegacyRestrictedType == nil {
			break
		}
		validator.currentRestrictedTypeUpgradeRestrictions = oldType.Types
		return validator.checkTypeUpgradability(oldType.LegacyRestrictedType, newType)

	case *ast.VariableSizedType:
		if newVariableSizedType, isVariableSizedType := newType.(*ast.VariableSizedType); isVariableSizedType {
			return validator.checkTypeUpgradability(oldType.Type, newVariableSizedType.Type)
		}
	case *ast.ConstantSizedType:
		if newConstantSizedType, isConstantSizedType := newType.(*ast.ConstantSizedType); isConstantSizedType {
			if oldType.Size.Value.Cmp(newConstantSizedType.Size.Value) != 0 ||
				oldType.Size.Base != newConstantSizedType.Size.Base {
				return newTypeMismatchError(oldType, newConstantSizedType)
			}
			return validator.checkTypeUpgradability(oldType.Type, newConstantSizedType.Type)
		}
	case *ast.DictionaryType:
		if newDictionaryType, isDictionaryType := newType.(*ast.DictionaryType); isDictionaryType {
			err := validator.checkTypeUpgradability(oldType.KeyType, newDictionaryType.KeyType)
			if err != nil {
				return err
			}
			return validator.checkTypeUpgradability(oldType.ValueType, newDictionaryType.ValueType)
		}
	case *ast.InstantiationType:
		// if the type is a Capability, allow the borrow type to change according to the normal upgrade rules
		if oldNominalType, isNominal := oldType.Type.(*ast.NominalType); isNominal &&
			oldNominalType.Identifier.Identifier == "Capability" {

			if instantiationType, isInstantiation := newType.(*ast.InstantiationType); isInstantiation {
				if newNominalType, isNominal := oldType.Type.(*ast.NominalType); isNominal &&
					newNominalType.Identifier.Identifier == "Capability" {

					// Capability insantiation types must have exactly 1 type argument
					if len(oldType.TypeArguments) != 1 || len(instantiationType.TypeArguments) != 1 {
						break
					}

					oldTypeArg := oldType.TypeArguments[0]
					newTypeArg := instantiationType.TypeArguments[0]

					return validator.checkTypeUpgradability(oldTypeArg.Type, newTypeArg.Type)
				}
			}
		}
	}

	return oldType.CheckEqual(newType, validator)

}

func (validator *LegacyContractUpdateValidator) checkField(oldField *ast.FieldDeclaration, newField *ast.FieldDeclaration) {
	oldType := oldField.TypeAnnotation.Type
	newType := newField.TypeAnnotation.Type

	validator.currentRestrictedTypeUpgradeRestrictions = nil
	err := validator.checkTypeUpgradability(oldType, newType)
	if err == nil {
		return
	}

	validator.report(&FieldMismatchError{
		DeclName:  validator.getCurrentDeclaration().DeclarationIdentifier().Identifier,
		FieldName: newField.Identifier.Identifier,
		Err:       err,
		Range:     ast.NewUnmeteredRangeFromPositioned(newField.TypeAnnotation),
	})
}

// AuthorizationMismatchError is reported during a contract upgrade,
// when a field value is given authorization that is more powerful
// than that which the migration would grant it
type AuthorizationMismatchError struct {
	ExpectedAuthorization sema.Access
	FoundAuthorization    sema.Access
	ast.Range
}

var _ errors.UserError = &AuthorizationMismatchError{}
var _ errors.SecondaryError = &AuthorizationMismatchError{}

func (*AuthorizationMismatchError) IsUserError() {}

func (e *AuthorizationMismatchError) Error() string {
	return "mismatching authorization"
}

func (e *AuthorizationMismatchError) SecondaryError() string {
	if e.ExpectedAuthorization == sema.PrimitiveAccess(ast.AccessAll) {
		return fmt.Sprintf(
			"The entitlements migration would not grant this value any entitlements, but the annotation present is `%s`",
			e.FoundAuthorization.QualifiedString(),
		)
	}

	return fmt.Sprintf(
		"The entitlements migration would only grant this value `%s`, but the annotation present is `%s`",
		e.ExpectedAuthorization.QualifiedString(),
		e.FoundAuthorization.QualifiedString(),
	)
}
