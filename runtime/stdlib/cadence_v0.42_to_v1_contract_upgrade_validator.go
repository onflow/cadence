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
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type CadenceV042ToV1ContractUpdateValidator struct {
	*TypeComparator

	newElaborations                          map[common.Location]*sema.Elaboration
	currentRestrictedTypeUpgradeRestrictions []*ast.NominalType

	underlyingUpdateValidator *ContractUpdateValidator

	checkUserDefinedType func(oldTypeID common.TypeID, newTypeID common.TypeID) (checked, valid bool)
}

// NewCadenceV042ToV1ContractUpdateValidator initializes and returns a validator, without performing any validation.
// Invoke the `Validate()` method of the validator returned, to start validating the contract.
func NewCadenceV042ToV1ContractUpdateValidator(
	location common.Location,
	contractName string,
	provider AccountContractNamesProvider,
	oldProgram *ast.Program,
	newProgram *interpreter.Program,
	newElaborations map[common.Location]*sema.Elaboration,
) *CadenceV042ToV1ContractUpdateValidator {

	underlyingValidator := NewContractUpdateValidator(
		location,
		contractName,
		provider,
		oldProgram,
		newProgram.Program,
	)

	// Also add the elaboration of the current program.
	newElaborations[location] = newProgram.Elaboration

	return &CadenceV042ToV1ContractUpdateValidator{
		underlyingUpdateValidator: underlyingValidator,
		newElaborations:           newElaborations,
		TypeComparator:            underlyingValidator.TypeComparator,
	}
}

var _ UpdateValidator = &CadenceV042ToV1ContractUpdateValidator{}

func (validator *CadenceV042ToV1ContractUpdateValidator) WithUserDefinedTypeChangeChecker(
	typeChangeCheckFunc func(oldTypeID common.TypeID, newTypeID common.TypeID) (checked, valid bool),
) *CadenceV042ToV1ContractUpdateValidator {
	validator.checkUserDefinedType = typeChangeCheckFunc
	return validator
}

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

	checkDeclarationUpdatability(
		validator,
		oldRootDecl,
		newRootDecl,
		validator.checkConformanceV1,
	)

	if underlyingValidator.hasErrors() {
		return underlyingValidator.getContractUpdateError()
	}

	return nil
}

func (validator *CadenceV042ToV1ContractUpdateValidator) report(err error) {
	validator.underlyingUpdateValidator.report(err)
}

func (validator *CadenceV042ToV1ContractUpdateValidator) typeIDFromType(typ ast.Type) (
	common.TypeID,
	error,
) {
	switch typ := typ.(type) {
	case *ast.NominalType:
		id, _ := validator.idAndLocationOfQualifiedType(typ)
		return id, nil
	case *ast.IntersectionType:
		var interfaceTypeIDs []common.TypeID
		for _, typ := range typ.Types {
			typeID, err := validator.typeIDFromType(typ)
			if err != nil {
				return "", err
			}
			interfaceTypeIDs = append(interfaceTypeIDs, typeID)
		}

		return sema.FormatIntersectionTypeID[common.TypeID](interfaceTypeIDs), nil
	default:
		// For now, only needs to support nominal types and intersection types.
		return "", errors.NewDefaultUserError("Unsupported type")
	}
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

	newImportLocations := validator.TypeComparator.foundIdentifierImportLocations
	oldImportLocations := validator.TypeComparator.expectedIdentifierImportLocations

	// Here we only need to find the qualified type ID.
	// So check in both old imports as well as in new imports.
	location, wasImported := newImportLocations[typIdentifier]
	if !wasImported {
		location, wasImported = oldImportLocations[typIdentifier]
	}

	if !wasImported {
		location = validator.underlyingUpdateValidator.location
	}

	if typIdentifier != rootIdentifier && !wasImported {
		qualifiedString = fmt.Sprintf("%s.%s", rootIdentifier, qualifiedString)
		return common.NewTypeIDFromQualifiedName(nil, location, qualifiedString), location
	}

	return common.NewTypeIDFromQualifiedName(nil, location, qualifiedString), location
}

func (validator *CadenceV042ToV1ContractUpdateValidator) getEntitlementType(
	entitlement *ast.NominalType,
) *sema.EntitlementType {
	typeID, location := validator.idAndLocationOfQualifiedType(entitlement)
	elaboration, ok := validator.newElaborations[location]
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return elaboration.EntitlementType(typeID)
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

func (validator *CadenceV042ToV1ContractUpdateValidator) requireEqualAccess(
	expected sema.Access,
	found sema.EntitlementSetAccess,
	foundType ast.Type,
) error {
	if !found.Equal(expected) {
		return &AuthorizationMismatchError{
			FoundAuthorization:    found,
			ExpectedAuthorization: expected,
			Range:                 ast.NewUnmeteredRangeFromPositioned(foundType),
		}
	}
	return nil
}

func (validator *CadenceV042ToV1ContractUpdateValidator) expectedAuthorizationOfComposite(composite *ast.NominalType) sema.Access {
	// If this field is set, we are currently upgrading a former legacy restricted type into a reference to a composite
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

func (validator *CadenceV042ToV1ContractUpdateValidator) checkEntitlementsUpgrade(newType *ast.ReferenceType) error {
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
		return validator.requireEqualAccess(expectedAccess, foundEntitlementSet, newReferencedType)

	case *ast.IntersectionType:
		expectedAccess := validator.expectedAuthorizationOfIntersection(newReferencedType.Types)
		return validator.requireEqualAccess(expectedAccess, foundEntitlementSet, newReferencedType)
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
				return validator.checkEntitlementsUpgrade(newReference)

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
		// and if there are atleast one restriction, require them to drop the "restricted type".
		// e.g-1: `AnyStruct{I} -> {I}`
		// e.g-2: `AnyResource{I} -> {I}`
		// See: https://github.com/onflow/cadence/issues/3112
		if restrictedNominalType, isNominal := oldType.LegacyRestrictedType.(*ast.NominalType); isNominal {
			switch restrictedNominalType.Identifier.Identifier {
			case "AnyStruct", "AnyResource":
				if len(oldType.Types) > 0 {
					break typeSwitch
				}
			}
		}

		// Otherwise require them to drop the "restrictions".
		// e.g-1: `T{I} -> T`
		// e.g-2: `AnyStruct{} -> AnyStruct`
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

	case *ast.NominalType:
		if validator.checkUserDefinedType == nil {
			break
		}

		if _, isbuiltinType := builtinTypes[oldType.String()]; !isbuiltinType {
			checked, valid := validator.checkUserDefinedTypeCustomRules(oldType, newType)

			// If there are no custom rules for this type,
			// do the default type comparison.
			if !checked {
				break
			}

			if valid {
				return nil
			}

			return newTypeMismatchError(oldType, newType)
		}
	}

	// If the new/old type is non-storable,
	// then changing the type of this field has no impact to the storage.
	if isNonStorableType(oldType) || isNonStorableType(newType) {
		return nil
	}

	return oldType.CheckEqual(newType, validator)

}

func (validator *CadenceV042ToV1ContractUpdateValidator) checkUserDefinedTypeCustomRules(
	oldType ast.Type,
	newType ast.Type,
) (checked, valid bool) {

	if validator.checkUserDefinedType == nil {
		return false, false
	}

	oldTypeID, err := validator.typeIDFromType(oldType)
	if err != nil {
		return false, false
	}

	newTypeID, err := validator.typeIDFromType(newType)
	if err != nil {
		return false, false
	}

	return validator.checkUserDefinedType(oldTypeID, newTypeID)
}

func isNonStorableType(typ ast.Type) bool {
	switch typ := typ.(type) {
	case *ast.ReferenceType, *ast.FunctionType:
		return true
	case *ast.OptionalType:
		return isNonStorableType(typ.Type)
	default:
		return false
	}
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
	if parent != nil &&
		parent.DeclarationKind() == common.DeclarationKindContractInterface {

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

func (validator *CadenceV042ToV1ContractUpdateValidator) checkNestedDeclarationRemoval(
	nestedDeclaration ast.Declaration,
	oldContainingDeclaration ast.Declaration,
	newContainingDeclaration ast.Declaration,
) {

	// enums can be removed from contract interfaces, as they have no interface equivalent and are not
	// actually used in field type annotations in any contracts
	if oldContainingDeclaration.DeclarationKind() == common.DeclarationKindContractInterface &&
		newContainingDeclaration.DeclarationKind() == common.DeclarationKindContractInterface &&
		nestedDeclaration.DeclarationKind() == common.DeclarationKindEnum {
		return
	}

	validator.underlyingUpdateValidator.checkNestedDeclarationRemoval(
		nestedDeclaration,
		oldContainingDeclaration,
		newContainingDeclaration,
	)
}

func (validator *CadenceV042ToV1ContractUpdateValidator) checkConformanceV1(
	oldDecl *ast.CompositeDeclaration,
	newDecl *ast.CompositeDeclaration,
) {

	oldConformances := oldDecl.Conformances

	// NOTE 1: Here it is assumed enums will always have one and only one conformance.
	// This is enforced by the checker.
	//
	// NOTE 2: If one declaration is an enum, then other is also an enum at this stage.
	// This is enforced by the validator (in `checkDeclarationUpdatability`), before calling this function.
	if newDecl.Kind() == common.CompositeKindEnum {
		oldConformance := oldConformances[0]
		newConformance := newDecl.Conformances[0]

		err := oldConformance.CheckEqual(newConformance, validator)
		if err != nil {
			validator.report(&ConformanceMismatchError{
				DeclName:           newDecl.Identifier.Identifier,
				MissingConformance: oldConformance.String(),
				Range:              ast.NewUnmeteredRangeFromPositioned(newDecl.Identifier),
			})
		}

		return
	}

	// Below check for multiple conformances is only applicable
	// for non-enum type composite declarations. i.e: structs, resources, etc.

	location := validator.underlyingUpdateValidator.location

	elaboration := validator.newElaborations[location]
	newDeclType := elaboration.CompositeDeclarationType(newDecl)

	// A conformance may not be explicitly defined in the current declaration,
	// but they could be available via inheritance.
	newConformances := newDeclType.EffectiveInterfaceConformances()

	// All the existing conformances must have a match. Order is not important.
	// Having extra new conformance is OK. See: https://github.com/onflow/cadence/issues/1394

	// Note: Removing a conformance is NOT OK. That could lead to type-safety issues.
	// e.g:
	//  - Someone stores an array of type `[{I}]` with `T:I` objects inside.
	//  - Later T’s conformance to `I` is removed.
	//  - Now `[{I}]` contains objects if `T` that does not conform to `I`.

	for _, oldConformance := range oldConformances {
		found := false
		for index, newConformance := range newConformances {
			newConformanceNominalType := semaConformanceToASTNominalType(newConformance)

			// First check whether there are any custom type-change rules.
			customRuleChecked, customRuleValid :=
				validator.checkUserDefinedTypeCustomRules(oldConformance, newConformanceNominalType)

			if customRuleChecked {
				// If exists, take its result.
				// DO NOT fall back to the default type equality check, even if the rule did not satisfy.
				found = customRuleValid
			} else {
				// If no custom rule exist, then use the default type equality check.
				err := oldConformance.CheckEqual(newConformanceNominalType, validator)
				found = err == nil
			}

			if found {
				// Remove the matched conformance, so we don't have to check it again.
				// i.e: optimization
				newConformances = append(newConformances[:index], newConformances[index+1:]...)
				break
			}
		}

		if !found {
			oldConformanceID := validator.underlyingUpdateValidator.oldTypeID(oldConformance)

			validator.report(&ConformanceMismatchError{
				DeclName:           newDecl.Identifier.Identifier,
				MissingConformance: string(oldConformanceID),
				Range:              ast.NewUnmeteredRangeFromPositioned(newDecl.Identifier),
			})

			return
		}
	}
}

func semaConformanceToASTNominalType(newConformance sema.Conformance) *ast.NominalType {
	interfaceType := newConformance.InterfaceType
	containerType := interfaceType.GetContainerType()

	identifier := ast.Identifier{
		Identifier: interfaceType.Identifier,
	}

	if containerType == nil {
		return ast.NewNominalType(nil, identifier, nil)
	}

	return ast.NewNominalType(
		nil,
		ast.Identifier{
			Identifier: containerType.String(),
		},
		[]ast.Identifier{identifier},
	)

}

var builtinTypes = map[string]struct{}{}

func init() {
	err := sema.BaseTypeActivation.ForEach(func(s string, _ *sema.Variable) error {
		builtinTypes[s] = struct{}{}
		return nil
	})

	if err != nil {
		panic(err)
	}
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

func (*AuthorizationMismatchError) IsUserError() {}

func (e *AuthorizationMismatchError) Error() string {
	if e.ExpectedAuthorization == sema.PrimitiveAccess(ast.AccessAll) {
		return fmt.Sprintf(
			"mismatching authorization: the entitlements migration would not grant this value any entitlements, but the annotation present is `%s`",
			e.FoundAuthorization.QualifiedString(),
		)
	}

	return fmt.Sprintf(
		"mismatching authorization: the entitlements migration would only grant this value `%s`, but the annotation present is `%s`",
		e.ExpectedAuthorization.QualifiedString(),
		e.FoundAuthorization.QualifiedString(),
	)
}
