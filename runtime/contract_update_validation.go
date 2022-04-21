/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package runtime

import (
	"sort"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

type ContractUpdateValidator struct {
	location     Location
	contractName string
	oldProgram   *ast.Program
	newProgram   *ast.Program
	rootDecl     ast.Declaration
	currentDecl  ast.Declaration
	errors       []error
}

// ContractUpdateValidator should implement ast.TypeEqualityChecker
var _ ast.TypeEqualityChecker = &ContractUpdateValidator{}

// NewContractUpdateValidator initializes and returns a validator, without performing any validation.
// Invoke the `Validate()` method of the validator returned, to start validating the contract.
func NewContractUpdateValidator(
	location Location,
	contractName string,
	oldProgram *ast.Program,
	newProgram *ast.Program,
) *ContractUpdateValidator {

	return &ContractUpdateValidator{
		location:     location,
		oldProgram:   oldProgram,
		newProgram:   newProgram,
		contractName: contractName,
	}
}

// Validate validates the contract update, and returns an error if it is an invalid update.
func (validator *ContractUpdateValidator) Validate() error {
	oldRootDecl := validator.getRootDeclaration(validator.oldProgram)
	if validator.hasErrors() {
		return validator.getContractUpdateError()
	}

	newRootDecl := validator.getRootDeclaration(validator.newProgram)
	if validator.hasErrors() {
		return validator.getContractUpdateError()
	}

	validator.rootDecl = newRootDecl
	validator.checkDeclarationUpdatability(oldRootDecl, newRootDecl)

	if validator.hasErrors() {
		return validator.getContractUpdateError()
	}

	return nil
}

func (validator *ContractUpdateValidator) getRootDeclaration(program *ast.Program) ast.Declaration {
	decl, err := getRootDeclaration(program)

	if err != nil {
		validator.report(&ContractNotFoundError{
			Range: ast.NewUnmeteredRangeFromPositioned(program),
		})
	}

	return decl
}

func getRootDeclaration(program *ast.Program) (ast.Declaration, error) {
	compositeDecl := program.SoleContractDeclaration()
	if compositeDecl != nil {
		return compositeDecl, nil
	}

	interfaceDecl := program.SoleContractInterfaceDeclaration()
	if interfaceDecl != nil {
		return interfaceDecl, nil
	}

	return nil, &ContractNotFoundError{
		Range: ast.NewUnmeteredRangeFromPositioned(program),
	}

}

func (validator *ContractUpdateValidator) hasErrors() bool {
	return len(validator.errors) > 0
}

func (validator *ContractUpdateValidator) checkDeclarationUpdatability(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) {

	// Do not allow converting between different types of composite declarations:
	// e.g: - 'contracts' and 'contract-interfaces',
	//      - 'structs' and 'enums'
	if oldDeclaration.DeclarationKind() != newDeclaration.DeclarationKind() {
		validator.report(&InvalidDeclarationKindChangeError{
			Name:    oldDeclaration.DeclarationIdentifier().Identifier,
			OldKind: oldDeclaration.DeclarationKind(),
			NewKind: newDeclaration.DeclarationKind(),
			Range:   ast.NewUnmeteredRangeFromPositioned(newDeclaration.DeclarationIdentifier()),
		})

		return
	}

	parentDecl := validator.currentDecl
	validator.currentDecl = newDeclaration
	defer func() {
		validator.currentDecl = parentDecl
	}()

	validator.checkFields(oldDeclaration, newDeclaration)

	validator.checkNestedDeclarations(oldDeclaration, newDeclaration)

	if newDecl, ok := newDeclaration.(*ast.CompositeDeclaration); ok {
		if oldDecl, ok := oldDeclaration.(*ast.CompositeDeclaration); ok {
			validator.checkConformances(oldDecl, newDecl)
		}
	}
}

func (validator *ContractUpdateValidator) checkFields(oldDeclaration ast.Declaration, newDeclaration ast.Declaration) {

	oldFields := oldDeclaration.DeclarationMembers().FieldsByIdentifier()
	newFields := newDeclaration.DeclarationMembers().Fields()

	// Updated contract has to have at-most the same number of field as the old contract.
	// Any additional field may cause crashes/garbage-values when deserializing the already-stored data.
	// However, having fewer fields is fine for now. It will only leave some unused data,
	// and will not do any harm to the programs that are running.

	for _, newField := range newFields {
		oldField := oldFields[newField.Identifier.Identifier]
		if oldField == nil {
			validator.report(&ExtraneousFieldError{
				DeclName:  newDeclaration.DeclarationIdentifier().Identifier,
				FieldName: newField.Identifier.Identifier,
				Range:     ast.NewUnmeteredRangeFromPositioned(newField.Identifier),
			})

			continue
		}

		validator.checkField(oldField, newField)
	}
}

func (validator *ContractUpdateValidator) checkField(oldField *ast.FieldDeclaration, newField *ast.FieldDeclaration) {
	err := oldField.TypeAnnotation.Type.CheckEqual(newField.TypeAnnotation.Type, validator)
	if err != nil {
		validator.report(&FieldMismatchError{
			DeclName:  validator.currentDecl.DeclarationIdentifier().Identifier,
			FieldName: newField.Identifier.Identifier,
			Err:       err,
			Range:     ast.NewUnmeteredRangeFromPositioned(newField.TypeAnnotation),
		})
	}
}

func (validator *ContractUpdateValidator) checkNestedDeclarations(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) {

	oldCompositeAndInterfaceDecls := getNestedCompositeAndInterfaceDecls(oldDeclaration)

	// Check nested structs, enums, etc.
	newNestedCompositeDecls := newDeclaration.DeclarationMembers().Composites()
	for _, newNestedDecl := range newNestedCompositeDecls {
		oldNestedDecl, found := oldCompositeAndInterfaceDecls[newNestedDecl.Identifier.Identifier]
		if !found {
			// Then it's a new declaration
			continue
		}

		validator.checkDeclarationUpdatability(oldNestedDecl, newNestedDecl)

		// If there's a matching new decl, then remove the old one from the map.
		delete(oldCompositeAndInterfaceDecls, newNestedDecl.Identifier.Identifier)
	}

	// Check nested interfaces.
	newNestedInterfaces := newDeclaration.DeclarationMembers().Interfaces()
	for _, newNestedDecl := range newNestedInterfaces {
		oldNestedDecl, found := oldCompositeAndInterfaceDecls[newNestedDecl.Identifier.Identifier]
		if !found {
			// Then this is a new declaration.
			continue
		}

		validator.checkDeclarationUpdatability(oldNestedDecl, newNestedDecl)

		// If there's a matching new decl, then remove the old one from the map.
		delete(oldCompositeAndInterfaceDecls, newNestedDecl.Identifier.Identifier)
	}

	// The remaining old declarations don't have a corresponding new declaration,
	// i.e., an existing declaration was removed.
	// Hence, report an error.

	missingDeclarations := make([]ast.Declaration, 0, len(oldCompositeAndInterfaceDecls))

	for _, declaration := range oldCompositeAndInterfaceDecls { //nolint:maprangecheck
		missingDeclarations = append(missingDeclarations, declaration)
	}

	sort.Slice(missingDeclarations, func(i, j int) bool {
		return missingDeclarations[i].DeclarationIdentifier().Identifier <
			missingDeclarations[j].DeclarationIdentifier().Identifier
	})

	for _, declaration := range missingDeclarations {
		validator.report(&MissingDeclarationError{
			Name: declaration.DeclarationIdentifier().Identifier,
			Kind: declaration.DeclarationKind(),
			Range: ast.NewUnmeteredRangeFromPositioned(
				newDeclaration.DeclarationIdentifier(),
			),
		})
	}

	// Check enum-cases, if there are any.
	validator.checkEnumCases(oldDeclaration, newDeclaration)
}

func getNestedCompositeAndInterfaceDecls(declaration ast.Declaration) map[string]ast.Declaration {
	compositeAndInterfaceDecls := map[string]ast.Declaration{}

	nestedCompositeDecls := declaration.DeclarationMembers().CompositesByIdentifier()
	for identifier, nestedDecl := range nestedCompositeDecls { //nolint:maprangecheck
		compositeAndInterfaceDecls[identifier] = nestedDecl
	}

	nestedInterfaceDecls := declaration.DeclarationMembers().InterfacesByIdentifier()
	for identifier, nestedDecl := range nestedInterfaceDecls { //nolint:maprangecheck
		compositeAndInterfaceDecls[identifier] = nestedDecl
	}

	return compositeAndInterfaceDecls
}

// checkEnumCases validates updating enum cases. Updated enum must:
//   - Have at-least the same number of enum-cases as the old enum (Adding is allowed, but no removals).
//   - Preserve the order of the old enum-cases (Adding to top/middle is not allowed, swapping is not allowed).
func (validator *ContractUpdateValidator) checkEnumCases(oldDeclaration ast.Declaration, newDeclaration ast.Declaration) {
	newEnumCases := newDeclaration.DeclarationMembers().EnumCases()
	oldEnumCases := oldDeclaration.DeclarationMembers().EnumCases()

	newEnumCaseCount := len(newEnumCases)
	oldEnumCaseCount := len(oldEnumCases)

	if newEnumCaseCount < oldEnumCaseCount {
		validator.report(&MissingEnumCasesError{
			DeclName: newDeclaration.DeclarationIdentifier().Identifier,
			Expected: oldEnumCaseCount,
			Found:    newEnumCaseCount,
			Range:    ast.NewUnmeteredRangeFromPositioned(newDeclaration.DeclarationIdentifier()),
		})

		// If some enum cases are removed, trying to match each enum case
		// may result in too many regression errors.
		// Hence, return.
		return
	}

	// Check whether the enum cases matches the old enum cases.
	for index, newEnumCase := range newEnumCases {
		// If there are no more old enum-cases, then these are newly added enum-cases,
		// which should be fine.
		if index >= oldEnumCaseCount {
			continue
		}

		oldEnumCase := oldEnumCases[index]
		if oldEnumCase.Identifier.Identifier != newEnumCase.Identifier.Identifier {
			validator.report(&EnumCaseMismatchError{
				ExpectedName: oldEnumCase.Identifier.Identifier,
				FoundName:    newEnumCase.Identifier.Identifier,
				Range:        ast.NewUnmeteredRangeFromPositioned(newEnumCase),
			})
		}
	}
}

func (validator *ContractUpdateValidator) CheckNominalTypeEquality(expected *ast.NominalType, found ast.Type) error {
	foundNominalType, ok := found.(*ast.NominalType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	// First check whether the names are equal.
	ok = validator.checkNameEquality(expected, foundNominalType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return nil
}

func (validator *ContractUpdateValidator) CheckOptionalTypeEquality(expected *ast.OptionalType, found ast.Type) error {
	foundOptionalType, ok := found.(*ast.OptionalType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(foundOptionalType.Type, validator)
}

func (validator *ContractUpdateValidator) CheckVariableSizedTypeEquality(expected *ast.VariableSizedType, found ast.Type) error {
	foundVarSizedType, ok := found.(*ast.VariableSizedType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(foundVarSizedType.Type, validator)
}

func (validator *ContractUpdateValidator) CheckConstantSizedTypeEquality(expected *ast.ConstantSizedType, found ast.Type) error {
	foundConstSizedType, ok := found.(*ast.ConstantSizedType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	// Check size
	if foundConstSizedType.Size.Value.Cmp(expected.Size.Value) != 0 ||
		foundConstSizedType.Size.Base != expected.Size.Base {
		return getTypeMismatchError(expected, found)
	}

	// Check type
	return expected.Type.CheckEqual(foundConstSizedType.Type, validator)
}

func (validator *ContractUpdateValidator) CheckDictionaryTypeEquality(expected *ast.DictionaryType, found ast.Type) error {
	foundDictionaryType, ok := found.(*ast.DictionaryType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	err := expected.KeyType.CheckEqual(foundDictionaryType.KeyType, validator)
	if err != nil {
		return err
	}

	return expected.ValueType.CheckEqual(foundDictionaryType.ValueType, validator)
}

func (validator *ContractUpdateValidator) CheckRestrictedTypeEquality(expected *ast.RestrictedType, found ast.Type) error {
	foundRestrictedType, ok := found.(*ast.RestrictedType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	if expected.Type == nil {
		if !isAnyStructOrAnyResourceType(foundRestrictedType.Type) {
			return getTypeMismatchError(expected, found)
		}
		// else go on to check type restrictions
	} else if foundRestrictedType.Type == nil {
		if !isAnyStructOrAnyResourceType(expected.Type) {
			return getTypeMismatchError(expected, found)
		}
		// else go on to check type restrictions
	} else {
		// both are not nil
		err := expected.Type.CheckEqual(foundRestrictedType.Type, validator)
		if err != nil {
			return getTypeMismatchError(expected, found)
		}
	}

	if len(expected.Restrictions) != len(foundRestrictedType.Restrictions) {
		return getTypeMismatchError(expected, found)
	}

	for index, expectedRestriction := range expected.Restrictions {
		foundRestriction := foundRestrictedType.Restrictions[index]
		err := expectedRestriction.CheckEqual(foundRestriction, validator)
		if err != nil {
			return getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) CheckInstantiationTypeEquality(expected *ast.InstantiationType, found ast.Type) error {
	foundInstType, ok := found.(*ast.InstantiationType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	err := expected.Type.CheckEqual(foundInstType.Type, validator)
	if err != nil || len(expected.TypeArguments) != len(foundInstType.TypeArguments) {
		return getTypeMismatchError(expected, found)
	}

	for index, typeArgs := range expected.TypeArguments {
		otherTypeArgs := foundInstType.TypeArguments[index]
		err := typeArgs.Type.CheckEqual(otherTypeArgs.Type, validator)
		if err != nil {
			return getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) CheckFunctionTypeEquality(expected *ast.FunctionType, found ast.Type) error {
	foundFuncType, ok := found.(*ast.FunctionType)
	if !ok || len(expected.ParameterTypeAnnotations) != len(foundFuncType.ParameterTypeAnnotations) {
		return getTypeMismatchError(expected, found)
	}

	for index, expectedParamType := range expected.ParameterTypeAnnotations {
		foundParamType := foundFuncType.ParameterTypeAnnotations[index]
		err := expectedParamType.Type.CheckEqual(foundParamType.Type, validator)
		if err != nil {
			return getTypeMismatchError(expected, found)
		}
	}

	return expected.ReturnTypeAnnotation.Type.CheckEqual(foundFuncType.ReturnTypeAnnotation.Type, validator)
}

func (validator *ContractUpdateValidator) CheckReferenceTypeEquality(expected *ast.ReferenceType, found ast.Type) error {
	refType, ok := found.(*ast.ReferenceType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(refType.Type, validator)
}

func (validator *ContractUpdateValidator) checkNameEquality(expectedType *ast.NominalType, foundType *ast.NominalType) bool {
	isExpectedQualifiedName := expectedType.IsQualifiedName()
	isFoundQualifiedName := foundType.IsQualifiedName()

	// A field with a composite type can be defined in two ways:
	// 	- Using type name (var x @ResourceName)
	//	- Using qualified type name (var x @ContractName.ResourceName)

	if isExpectedQualifiedName && !isFoundQualifiedName {
		return validator.checkIdentifierEquality(expectedType, foundType)
	}

	if isFoundQualifiedName && !isExpectedQualifiedName {
		return validator.checkIdentifierEquality(foundType, expectedType)
	}

	// At this point, either both are qualified names, or both are simple names.
	// Thus, do a one-to-one match.
	if expectedType.Identifier.Identifier != foundType.Identifier.Identifier {
		return false
	}

	return identifiersEqual(expectedType.NestedIdentifiers, foundType.NestedIdentifiers)
}

func (validator *ContractUpdateValidator) checkIdentifierEquality(
	qualifiedNominalType *ast.NominalType,
	simpleNominalType *ast.NominalType,
) bool {

	// Situation:
	// qualifiedNominalType -> identifier: A, nestedIdentifiers: [foo, bar, ...]
	// simpleNominalType -> identifier: foo,  nestedIdentifiers: [bar, ...]

	// If the first identifier (i.e: 'A') refers to a composite decl that is not the enclosing contract,
	// then it must be referring to an imported contract. That means the two types are no longer the same.
	if qualifiedNominalType.Identifier.Identifier != validator.rootDecl.DeclarationIdentifier().Identifier {
		return false
	}

	if qualifiedNominalType.NestedIdentifiers[0].Identifier != simpleNominalType.Identifier.Identifier {
		return false
	}

	return identifiersEqual(simpleNominalType.NestedIdentifiers, qualifiedNominalType.NestedIdentifiers[1:])
}

func (validator *ContractUpdateValidator) checkConformances(
	oldDecl *ast.CompositeDeclaration,
	newDecl *ast.CompositeDeclaration,
) {

	// Here it is assumed enums will always have one and only one conformance.
	// This is enforced by the checker.
	// Therefore, below check for multiple conformances is only applicable
	// for non-enum type composite declarations. i.e: structs, resources, etc.

	oldConformances := oldDecl.Conformances
	newConformances := newDecl.Conformances

	// All the existing conformances must have a match. Order is not important.
	// Having extra new conformance is OK. See: https://github.com/onflow/cadence/issues/1394
	for _, oldConformance := range oldConformances {
		found := false
		for index, newConformance := range newConformances {
			err := oldConformance.CheckEqual(newConformance, validator)
			if err == nil {
				found = true

				// Remove the matched conformance, so we don't have to check it again.
				// i.e: optimization
				newConformances = append(newConformances[:index], newConformances[index+1:]...)
				break
			}
		}

		if !found {
			validator.report(&ConformanceMismatchError{
				DeclName: newDecl.Identifier.Identifier,
				Range:    ast.NewUnmeteredRangeFromPositioned(newDecl.Identifier),
			})

			return
		}
	}
}

func (validator *ContractUpdateValidator) report(err error) {
	if err == nil {
		return
	}
	validator.errors = append(validator.errors, err)
}

func (validator *ContractUpdateValidator) getContractUpdateError() error {
	return &ContractUpdateError{
		ContractName: validator.contractName,
		Errors:       validator.errors,
		Location:     validator.location,
	}
}

func getTypeMismatchError(expectedType ast.Type, foundType ast.Type) *TypeMismatchError {
	return &TypeMismatchError{
		ExpectedType: expectedType,
		FoundType:    foundType,
		Range:        ast.NewUnmeteredRangeFromPositioned(foundType),
	}
}

func identifiersEqual(expected []ast.Identifier, found []ast.Identifier) bool {
	if len(expected) != len(found) {
		return false
	}

	for index, element := range found {
		if expected[index].Identifier != element.Identifier {
			return false
		}
	}
	return true
}

func isAnyStructOrAnyResourceType(astType ast.Type) bool {
	// If the restricted type is not stated, then it is either AnyStruct or AnyResource
	if astType == nil {
		return true
	}

	nominalType, ok := astType.(*ast.NominalType)
	if !ok {
		return false
	}

	switch nominalType.Identifier.Identifier {
	case sema.AnyStructType.Name, sema.AnyResourceType.Name:
		return true
	default:
		return false
	}
}

func containsEnumsInProgram(program *ast.Program) bool {
	declaration, err := getRootDeclaration(program)

	if err != nil {
		return false
	}

	return containsEnums(declaration)
}

func containsEnums(declaration ast.Declaration) bool {
	if declaration.DeclarationKind() == common.DeclarationKindEnum {
		return true
	}

	nestedCompositeDecls := declaration.DeclarationMembers().Composites()
	for _, nestedDecl := range nestedCompositeDecls {
		if containsEnums(nestedDecl) {
			return true
		}
	}

	return false
}
