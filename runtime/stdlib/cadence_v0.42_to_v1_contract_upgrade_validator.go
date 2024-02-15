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

type CadenceV042ToV1ContractUpdateValidator struct {
	TypeComparator

	newElaborations                          map[common.Location]*sema.Elaboration
	currentRestrictedTypeUpgradeRestrictions []*ast.NominalType

	underlyingUpdateValidator *ContractUpdateValidator
}

// NewCadenceV042ToV1ContractUpdateValidator initializes and returns a validator, without performing any validation.
// Invoke the `Validate()` method of the validator returned, to start validating the contract.
func NewCadenceV042ToV1ContractUpdateValidator(
	location common.Location,
	contractName string,
	provider AccountContractNamesProvider,
	oldProgram *ast.Program,
	newProgram *ast.Program,
	newElaborations map[common.Location]*sema.Elaboration,
) *CadenceV042ToV1ContractUpdateValidator {

	underlyingValidator := NewContractUpdateValidator(location, contractName, provider, oldProgram, newProgram)

	return &CadenceV042ToV1ContractUpdateValidator{
		underlyingUpdateValidator: underlyingValidator,
		newElaborations:           newElaborations,
	}
}

var _ UpdateValidator = &CadenceV042ToV1ContractUpdateValidator{}

func (validator *CadenceV042ToV1ContractUpdateValidator) getCurrentDeclaration() ast.Declaration {
	return validator.underlyingUpdateValidator.getCurrentDeclaration()
}

func (validator *CadenceV042ToV1ContractUpdateValidator) setCurrentDeclaration(decl ast.Declaration) {
	validator.underlyingUpdateValidator.setCurrentDeclaration(decl)
}

func (validator *CadenceV042ToV1ContractUpdateValidator) getAccountContractNames(address common.Address) ([]string, error) {
	return validator.underlyingUpdateValidator.accountContractNamesProvider.GetAccountContractNames(address)
}

// Validate validates the contract update, and returns an error if it is an invalid update.
func (validator *CadenceV042ToV1ContractUpdateValidator) Validate() error {
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
	validator.TypeComparator.expectedIdentifierImportLocations = collectImports(validator, underlyingValidator.oldProgram)
	validator.TypeComparator.foundIdentifierImportLocations = collectImports(validator, underlyingValidator.newProgram)

	checkDeclarationUpdatability(validator, oldRootDecl, newRootDecl)

	if underlyingValidator.hasErrors() {
		return underlyingValidator.getContractUpdateError()
	}

	return nil
}

func (validator *CadenceV042ToV1ContractUpdateValidator) report(err error) {
	validator.underlyingUpdateValidator.report(err)
}

func (validator *CadenceV042ToV1ContractUpdateValidator) idAndLocationOfQualifiedType(typ *ast.NominalType) (
	common.TypeID,
	common.Location,
) {

	qualifiedString := typ.String()

	// working under the assumption that any new program we are validating already typechecks,
	// any nominal type must fall into one of three cases:
	// 1) type qualified by an import (e.g `C.R` where `C` is an imported type)
	// 2) type qualified by the root declaration (e.g `C.R` where `C` is the root contract or contract interface of the new contract)
	// 3) unqualified type (e.g. `R`, but declared inside `C`)
	//
	// in case 3, we prepend the root declaration identifier with a `.` to the type's string to get its qualified name,
	// and in 1 and 2 we don't need to do anything
	typIdentifier := typ.Identifier.Identifier
	rootIdentifier := validator.TypeComparator.RootDeclIdentifier.Identifier
	location := validator.underlyingUpdateValidator.location

	if typIdentifier != rootIdentifier &&
		validator.TypeComparator.foundIdentifierImportLocations[typ.Identifier.Identifier] == nil {
		qualifiedString = fmt.Sprintf("%s.%s", rootIdentifier, qualifiedString)
		return common.NewTypeIDFromQualifiedName(nil, location, qualifiedString), location

	}

	if loc := validator.TypeComparator.foundIdentifierImportLocations[typ.Identifier.Identifier]; loc != nil {
		location = loc
	}

	return common.NewTypeIDFromQualifiedName(nil, location, qualifiedString), location
}

func (validator *CadenceV042ToV1ContractUpdateValidator) getEntitlementType(
	entitlement *ast.NominalType,
) *sema.EntitlementType {
	typeID, location := validator.idAndLocationOfQualifiedType(entitlement)
	return validator.newElaborations[location].EntitlementType(typeID)
}

func (validator *CadenceV042ToV1ContractUpdateValidator) getEntitlementSetAccess(
	entitlementSet ast.EntitlementSet,
) sema.EntitlementSetAccess {
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

func (validator *CadenceV042ToV1ContractUpdateValidator) getCompositeType(composite *ast.NominalType) *sema.CompositeType {
	typeID, location := validator.idAndLocationOfQualifiedType(composite)
	return validator.newElaborations[location].CompositeType(typeID)
}

func (validator *CadenceV042ToV1ContractUpdateValidator) getInterfaceType(intf *ast.NominalType) *sema.InterfaceType {
	typeID, location := validator.idAndLocationOfQualifiedType(intf)
	return validator.newElaborations[location].InterfaceType(typeID)
}

func (validator *CadenceV042ToV1ContractUpdateValidator) getIntersectedInterfaces(
	intersection []*ast.NominalType,
) (interfaceTypes []*sema.InterfaceType) {
	for _, interfaceType := range intersection {
		interfaceTypes = append(interfaceTypes, validator.getInterfaceType(interfaceType))
	}
	return
}

func (validator *CadenceV042ToV1ContractUpdateValidator) requirePermitsAccess(
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

func (validator *CadenceV042ToV1ContractUpdateValidator) expectedAuthorizationOfComposite(composite *ast.NominalType) sema.Access {
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

func (validator *CadenceV042ToV1ContractUpdateValidator) expectedAuthorizationOfIntersection(
	intersectionTypes []*ast.NominalType,
) sema.Access {

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

func (validator *CadenceV042ToV1ContractUpdateValidator) checkEntitlementsUpgrade(
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

func (validator *CadenceV042ToV1ContractUpdateValidator) checkTypeUpgradability(oldType ast.Type, newType ast.Type) error {

typeSwitch:
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

		// If the old restricted type is for AnyStruct/AnyResource,
		// require them to drop the "restricted type".
		// e.g: `T{I} -> {I}`
		if restrictedNominalType, isNominal := oldType.LegacyRestrictedType.(*ast.NominalType); isNominal {
			switch restrictedNominalType.Identifier.Identifier {
			case "AnyStruct", "AnyResource":
				break typeSwitch
			}
		}

		// Otherwise require them to drop the "restriction".
		// e.g: `T{I} -> T`
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

func (validator *CadenceV042ToV1ContractUpdateValidator) checkField(oldField *ast.FieldDeclaration, newField *ast.FieldDeclaration) {
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

func (validator *CadenceV042ToV1ContractUpdateValidator) checkDeclarationKindChange(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) bool {
	// Do not allow converting between different types of composite declarations:
	// e.g: - 'contracts' and 'contract-interfaces',
	//      - 'structs' and 'enums'
	//
	// However, with the removal of type requirements, it is OK to convert a
	// concrete type (Struct or Resource) to an interface type (StructInterface or ResourceInterface).
	// However, resource should stay a resource interface, and cannot be a struct interface.

	oldDeclKind := oldDeclaration.DeclarationKind()
	newDeclKind := newDeclaration.DeclarationKind()
	if oldDeclKind == newDeclKind {
		return true
	}

	parent := validator.getCurrentDeclaration()

	// If the parent is an interface, and the child is a concrete type,
	// then it is a type requirement.
	if parent.DeclarationKind() == common.DeclarationKindContractInterface {
		// A struct is OK to be converted to a struct-interface
		if oldDeclKind == common.DeclarationKindStructure &&
			newDeclKind == common.DeclarationKindStructureInterface {
			return true
		}

		// A resource is OK to be converted to a resource-interface
		if oldDeclKind == common.DeclarationKindResource &&
			newDeclKind == common.DeclarationKindResourceInterface {
			return true
		}
	}

	validator.report(&InvalidDeclarationKindChangeError{
		Name:    oldDeclaration.DeclarationIdentifier().Identifier,
		OldKind: oldDeclaration.DeclarationKind(),
		NewKind: newDeclaration.DeclarationKind(),
		Range:   ast.NewUnmeteredRangeFromPositioned(newDeclaration.DeclarationIdentifier()),
	})
	return false
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
