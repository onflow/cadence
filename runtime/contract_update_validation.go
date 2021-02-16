/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/ast"
)

type ContractUpdateValidator struct {
	oldProgram     *ast.Program
	newProgram     *ast.Program
	newDeclaration ast.Declaration
	currentDecl    ast.Declaration
	visited        map[ast.Declaration]bool
}

// ContractUpdateValidator should implement ast.TypeEqualityChecker
var _ ast.TypeEqualityChecker = &ContractUpdateValidator{}

func NewContractUpdateValidator(oldProgram *ast.Program, newProgram *ast.Program) *ContractUpdateValidator {
	return &ContractUpdateValidator{
		oldProgram: oldProgram,
		newProgram: newProgram,
		visited:    map[ast.Declaration]bool{},
	}
}

func (validator *ContractUpdateValidator) Validate() error {
	oldRootDecl, err := getRootDeclaration(validator.oldProgram)
	if err != nil {
		return err
	}

	newRootDecl, err := getRootDeclaration(validator.newProgram)
	if err != nil {
		return err
	}

	validator.newDeclaration = newRootDecl

	return validator.checkDeclarationUpdatability(oldRootDecl, newRootDecl)
}

func (validator *ContractUpdateValidator) checkDeclarationUpdatability(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) error {

	// Do not allow converting between different types of composite declarations:
	// e.g: - 'contracts' and 'contract-interfaces',
	//      - 'structs' and 'enums'
	if oldDeclaration.DeclarationKind() != newDeclaration.DeclarationKind() {
		return &InvalidDeclarationKindChangeError{
			name:    oldDeclaration.DeclarationIdentifier().Identifier,
			oldKind: oldDeclaration.DeclarationKind().Name(),
			newKind: newDeclaration.DeclarationKind().Name(),
			Range:   ast.NewRangeFromPositioned(newDeclaration),
		}
	}

	parentDecl := validator.currentDecl
	validator.currentDecl = newDeclaration
	defer func() {
		validator.currentDecl = parentDecl
	}()

	// If the same decl is already visited, then do not check again.
	// This also avoids getting stuck on circular dependencies between composite decls.
	if validator.visited[newDeclaration] {
		return nil
	}
	validator.visited[newDeclaration] = true

	err := validator.checkFields(oldDeclaration, newDeclaration)
	if err != nil {
		return err
	}

	err = validator.checkNestedDeclarations(oldDeclaration, newDeclaration)
	if err != nil {
		return err
	}

	if newDecl, ok := newDeclaration.(*ast.CompositeDeclaration); ok {
		if oldDecl, ok := oldDeclaration.(*ast.CompositeDeclaration); ok {
			err := validator.checkConformances(oldDecl, newDecl)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) checkFields(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) error {

	oldFields := oldDeclaration.DeclarationMembers().FieldsByIdentifier()
	newFields := newDeclaration.DeclarationMembers().Fields()

	// Updated contract has to have at-most the same number of field as the old contract.
	// Any additional field may cause crashes/garbage-values when deserializing the
	// already-stored data. However, having less number of fields is fine for now. It will
	// only leave some unused-data, and will not do any harm to the programs that are running.

	// This is a fail-fast check.
	if len(oldFields) < len(newFields) {
		currentDeclName := newDeclaration.DeclarationIdentifier()
		return &TooManyFieldsError{
			declName:       currentDeclName.Identifier,
			expectedFields: len(oldFields),
			foundFields:    len(newFields),
			Range:          ast.NewRangeFromPositioned(currentDeclName),
		}
	}

	for _, newField := range newFields {
		oldField := oldFields[newField.Identifier.Identifier]
		if oldField == nil {
			return &ExtraneousFieldError{
				declName:  newDeclaration.DeclarationIdentifier().Identifier,
				fieldName: newField.Identifier.Identifier,
				Range:     ast.NewRangeFromPositioned(newField.Identifier),
			}
		}

		err := validator.checkField(oldField, newField)
		if err != nil {
			return err
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) checkField(
	oldField *ast.FieldDeclaration,
	newField *ast.FieldDeclaration,
) error {

	oldType := oldField.TypeAnnotation.Type

	err := oldType.CheckEqual(newField.TypeAnnotation.Type, validator)
	if err != nil {
		return &FieldMismatchError{
			declName:  validator.currentDecl.DeclarationIdentifier().Identifier,
			fieldName: newField.Identifier.Identifier,
			err:       err,
			Range:     ast.NewRangeFromPositioned(newField),
		}
	}
	return nil
}

func (validator *ContractUpdateValidator) checkNestedDeclarations(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) error {

	oldNestedCompositeDecls := oldDeclaration.DeclarationMembers().CompositesByIdentifier()
	oldNestedInterfaceDecls := oldDeclaration.DeclarationMembers().InterfacesByIdentifier()

	getOldCompositeOrInterfaceDecl := func(name string) (ast.Declaration, bool) {
		oldCompositeDecl := oldNestedCompositeDecls[name]
		if oldCompositeDecl != nil {
			return oldCompositeDecl, true
		}

		oldInterfaceDecl := oldNestedInterfaceDecls[name]
		if oldInterfaceDecl != nil {
			return oldInterfaceDecl, true
		}

		return nil, false
	}

	newNestedCompositeDecls := newDeclaration.DeclarationMembers().Composites()
	for _, newNestedDecl := range newNestedCompositeDecls {
		oldNestedDecl, found := getOldCompositeOrInterfaceDecl(newNestedDecl.Identifier.Identifier)
		if !found {
			// Then its a new declaration
			continue
		}

		err := validator.checkDeclarationUpdatability(oldNestedDecl, newNestedDecl)
		if err != nil {
			return err
		}
	}

	newNestedInterfaces := newDeclaration.DeclarationMembers().Interfaces()
	for _, newNestedDecl := range newNestedInterfaces {
		oldNestedDecl, found := getOldCompositeOrInterfaceDecl(newNestedDecl.Identifier.Identifier)
		if !found {
			// Then this is a new declaration.
			continue
		}

		err := validator.checkDeclarationUpdatability(oldNestedDecl, newNestedDecl)
		if err != nil {
			return err
		}
	}

	return nil
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

	err := expected.Type.CheckEqual(foundRestrictedType.Type, validator)
	if err != nil || len(expected.Restrictions) != len(foundRestrictedType.Restrictions) {
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
	_, ok := found.(*ast.FunctionType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return &InvalidNonStorableTypeUsageError{
		nonStorableType: found,
		Range:           ast.NewRangeFromPositioned(found),
	}
}

func (validator *ContractUpdateValidator) CheckReferenceTypeEquality(expected *ast.ReferenceType, found ast.Type) error {
	_, ok := found.(*ast.ReferenceType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return &InvalidNonStorableTypeUsageError{
		nonStorableType: found,
		Range:           ast.NewRangeFromPositioned(found),
	}
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
	if qualifiedNominalType.Identifier.Identifier != validator.newDeclaration.DeclarationIdentifier().Identifier {
		return false
	}

	if qualifiedNominalType.NestedIdentifiers[0].Identifier != simpleNominalType.Identifier.Identifier {
		return false
	}

	return identifiersEqual(simpleNominalType.NestedIdentifiers, qualifiedNominalType.NestedIdentifiers[1:])
}

func (validator *ContractUpdateValidator) checkConformances(
	oldEnum *ast.CompositeDeclaration,
	newEnum *ast.CompositeDeclaration,
) error {

	oldConformances := oldEnum.Conformances
	newConformances := newEnum.Conformances

	if len(oldConformances) != len(newConformances) {
		return &ConformanceCountMismatchError{
			expected: len(oldConformances),
			found:    len(newConformances),
			Range:    ast.NewRangeFromPositioned(newEnum.Identifier),
		}
	}

	for index, conformance := range oldConformances {
		newConformance := newConformances[index]
		err := conformance.CheckEqual(newConformance, validator)
		if err != nil {
			return &ConformanceMismatchError{
				err:   err,
				Range: ast.NewRangeFromPositioned(newConformance),
			}
		}
	}

	return nil
}

func getTypeMismatchError(expectedType ast.Type, foundType ast.Type) *TypeMismatchError {
	return &TypeMismatchError{
		expectedType: expectedType,
		foundType:    foundType,
		Range:        ast.NewRangeFromPositioned(foundType),
	}
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
		Range: ast.NewRangeFromPositioned(program),
	}
}

func identifiersEqual(expected []ast.Identifier, found []ast.Identifier) bool {
	if len(expected) != len(found) {
		return false
	}

	for index, element := range found {
		if expected[index] != element {
			return false
		}
	}
	return true
}
