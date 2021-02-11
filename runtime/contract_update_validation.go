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
	oldDeclaration ast.Declaration
	newDeclaration ast.Declaration
	currentDecl    ast.Declaration
	currentField   *ast.FieldDeclaration
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

	compositeDecl := validator.oldProgram.SoleContractDeclaration()
	if compositeDecl != nil {
		validator.oldDeclaration = compositeDecl
	} else {
		interfaceDecl := validator.oldProgram.SoleContractInterfaceDeclaration()
		if interfaceDecl == nil {
			return &ContractNotFoundError{
				Range: ast.NewRangeFromPositioned(validator.oldProgram),
			}
		}

		validator.oldDeclaration = interfaceDecl
	}

	compositeDecl = validator.newProgram.SoleContractDeclaration()
	if compositeDecl != nil {
		validator.newDeclaration = compositeDecl
	} else {
		interfaceDecl := validator.newProgram.SoleContractInterfaceDeclaration()
		if interfaceDecl == nil {
			return &ContractNotFoundError{
				Range: ast.NewRangeFromPositioned(validator.newProgram),
			}
		}
		validator.newDeclaration = interfaceDecl
	}

	// Do not allow converting between 'contracts' and 'contract-interfaces'.
	if validator.oldDeclaration.DeclarationKind() != validator.newDeclaration.DeclarationKind() {
		return &InvalidContractKindChangeError{
			oldKind: validator.oldDeclaration.DeclarationKind().Name(),
			newKind: validator.newDeclaration.DeclarationKind().Name(),
			Range:   ast.NewRangeFromPositioned(validator.newDeclaration),
		}
	}

	return validator.checkDeclarationUpdatability(validator.oldDeclaration, validator.newDeclaration)
}

func (validator *ContractUpdateValidator) checkDeclarationUpdatability(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) error {

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

	return validator.checkNestedCompositeDeclarations(oldDeclaration, newDeclaration)
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
		currentDeclName := validator.currentDecl.DeclarationIdentifier()
		return &TooManyFieldsError{
			declName:       currentDeclName.Identifier,
			expectedFields: len(oldFields),
			foundFields:    len(newFields),
			Range:          ast.NewRangeFromPositioned(currentDeclName),
		}
	}

	for _, newField := range newFields {
		err := validator.checkField(newField, oldFields[newField.Identifier.Identifier])
		if err != nil {
			return err
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) checkField(
	newField *ast.FieldDeclaration,
	oldField *ast.FieldDeclaration,
) error {

	prevField := validator.currentField
	validator.currentField = newField
	defer func() {
		validator.currentField = prevField
	}()

	oldType := oldField.TypeAnnotation.Type
	if oldType == nil {
		return &ExtraneousFieldError{
			declName:  validator.currentDecl.DeclarationIdentifier().Identifier,
			fieldName: validator.currentField.Identifier.Identifier,
			Range:     ast.NewRangeFromPositioned(validator.currentField.Identifier),
		}
	}

	return oldType.CheckEqual(newField.TypeAnnotation.Type, validator)
}

func (validator *ContractUpdateValidator) checkNestedCompositeDeclarations(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) error {

	oldNestedDecls := oldDeclaration.DeclarationMembers().CompositesByIdentifier()
	newNestedDecls := newDeclaration.DeclarationMembers().Composites()

	for _, nestedDecl := range newNestedDecls {
		err := validator.checkDeclarationUpdatability(oldNestedDecls[nestedDecl.Identifier.Identifier], nestedDecl)
		if err != nil {
			return err
		}
	}
	return nil
}

func (validator *ContractUpdateValidator) CheckNominalTypeEquality(expected *ast.NominalType, found ast.Type) error {
	foundNominalType, ok := found.(*ast.NominalType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	// First check whether the names are equal.
	ok = validator.checkNameEquality(expected, foundNominalType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	return nil
}

func (validator *ContractUpdateValidator) CheckOptionalTypeEquality(expected *ast.OptionalType, found ast.Type) error {
	foundOptionalType, ok := found.(*ast.OptionalType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(foundOptionalType.Type, validator)
}

func (validator *ContractUpdateValidator) CheckVariableSizedTypeEquality(expected *ast.VariableSizedType, found ast.Type) error {
	foundVarSizedType, ok := found.(*ast.VariableSizedType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(foundVarSizedType.Type, validator)
}

func (validator *ContractUpdateValidator) CheckConstantSizedTypeEquality(expected *ast.ConstantSizedType, found ast.Type) error {
	foundConstSizedType, ok := found.(*ast.ConstantSizedType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	// Check size
	if foundConstSizedType.Size.Value.Cmp(expected.Size.Value) != 0 ||
		foundConstSizedType.Size.Base != expected.Size.Base {
		return validator.getTypeMismatchError(expected, found)
	}

	// Check type
	return expected.Type.CheckEqual(foundConstSizedType.Type, validator)
}

func (validator *ContractUpdateValidator) CheckDictionaryTypeEquality(expected *ast.DictionaryType, found ast.Type) error {
	foundDictionaryType, ok := found.(*ast.DictionaryType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
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
		return validator.getTypeMismatchError(expected, found)
	}

	err := expected.Type.CheckEqual(foundRestrictedType.Type, validator)
	if err != nil || len(expected.Restrictions) != len(foundRestrictedType.Restrictions) {
		return validator.getTypeMismatchError(expected, found)
	}

	for index, expectedRestriction := range expected.Restrictions {
		foundRestriction := foundRestrictedType.Restrictions[index]
		err := expectedRestriction.CheckEqual(foundRestriction, validator)
		if err != nil {
			return validator.getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) CheckInstantiationTypeEquality(expected *ast.InstantiationType, found ast.Type) error {
	foundInstType, ok := found.(*ast.InstantiationType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	err := expected.Type.CheckEqual(foundInstType.Type, validator)
	if err != nil || len(expected.TypeArguments) != len(foundInstType.TypeArguments) {
		return validator.getTypeMismatchError(expected, found)
	}

	for index, typeArgs := range expected.TypeArguments {
		otherTypeArgs := foundInstType.TypeArguments[index]
		err := typeArgs.Type.CheckEqual(otherTypeArgs.Type, validator)
		if err != nil {
			return validator.getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) CheckFunctionTypeEquality(funcType *ast.FunctionType, _ ast.Type) error {
	return &InvalidNonStorableTypeUsageError{
		nonStorableType: funcType,
		Range:           ast.NewRangeFromPositioned(funcType),
	}
}

func (validator *ContractUpdateValidator) CheckReferenceTypeEquality(refType *ast.ReferenceType, _ ast.Type) error {
	return &InvalidNonStorableTypeUsageError{
		nonStorableType: refType,
		Range:           ast.NewRangeFromPositioned(refType),
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

func (validator *ContractUpdateValidator) getTypeMismatchError(expectedType ast.Type, foundType ast.Type) *FieldTypeMismatchError {
	return &FieldTypeMismatchError{
		declName:     validator.currentDecl.DeclarationIdentifier().Identifier,
		fieldName:    validator.currentField.Identifier.Identifier,
		expectedType: expectedType,
		foundType:    foundType,
		Range:        ast.NewRangeFromPositioned(validator.currentField),
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
