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
	"fmt"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
)

type ContractUpdateValidator struct {
	oldDeclaration          ast.Declaration
	newDeclaration          ast.Declaration
	oldNestedCompositeDecls map[string]*ast.CompositeDeclaration
	newNestedCompositeDecls map[string]*ast.CompositeDeclaration
	entryPointName          string
	currentDeclName         string
	currentFieldName        string
	visitedDecls            map[string]bool
}

// ContractUpdateValidator should implement ast.TypeEqualityChecker
var _ ast.TypeEqualityChecker = &ContractUpdateValidator{}

func NewContractUpdateValidator(
	runtime *interpreterRuntime,
	context Context,
	existingCode []byte,
	newProgram *ast.Program) *ContractUpdateValidator {

	oldProgram, err := runtime.parse(existingCode, context)
	if err != nil {
		// Not a user error. Hence panic
		panic(err)
	}

	oldCompDecls := oldProgram.CompositeDeclarations()
	oldInterfaceDecls := oldProgram.InterfaceDeclarations()
	var oldDecl ast.Declaration

	if len(oldCompDecls) == 1 {
		oldDecl = oldCompDecls[0]
	} else if len(oldInterfaceDecls) == 1 {
		oldDecl = oldInterfaceDecls[0]
	} else {
		panic(errors.NewUnreachableError())
	}

	newCompDecls := newProgram.CompositeDeclarations()
	newInterfaceDecls := newProgram.InterfaceDeclarations()
	var newDecl ast.Declaration

	if len(newCompDecls) == 1 {
		newDecl = newCompDecls[0]
	} else if len(newInterfaceDecls) == 1 {
		newDecl = newInterfaceDecls[0]
	} else {
		panic(errors.NewUnreachableError())
	}

	return &ContractUpdateValidator{
		oldDeclaration: oldDecl,
		newDeclaration: newDecl,
		visitedDecls:   map[string]bool{}}
}

func (validator *ContractUpdateValidator) Validate() error {
	// Do not allow converting between 'contracts' and 'contract-interfaces'.
	if validator.oldDeclaration.DeclarationKind() != validator.newDeclaration.DeclarationKind() {
		return fmt.Errorf("trying to convert a %s to a %s",
			validator.oldDeclaration.DeclarationKind().Name(),
			validator.newDeclaration.DeclarationKind().Name())
	}

	validator.entryPointName = getDeclarationName(validator.newDeclaration)
	return validator.checkDeclarationUpdatability(validator.oldDeclaration, validator.newDeclaration)
}

func (validator *ContractUpdateValidator) checkDeclarationUpdatability(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration) error {

	parentDeclName := validator.currentDeclName
	validator.currentDeclName = getDeclarationName(newDeclaration)
	defer func() {
		validator.currentDeclName = parentDeclName
	}()

	// If the same same decl is already visited, then do not check again.
	// This also avoids getting stuck on circular dependencies between composite decls.
	if validator.visitedDecls[validator.currentDeclName] {
		return nil
	}

	validator.visitedDecls[validator.currentDeclName] = true

	oldFields := getDeclarationMembers(oldDeclaration).Fields()
	newFields := getDeclarationMembers(newDeclaration).Fields()

	// Updated contract has to have at-most the same number of field as the old contract.
	// Any additional field may cause crashes/garbage-values when deserializing the
	// already-stored data. However, having less number of fields is fine for now. It will
	// only leave some unused-data, and will not do any harm to the programs that are running.

	// This is a fail-fast check.
	if len(oldFields) < len(newFields) {
		if validator.isInNestedDecl() {
			return fmt.Errorf("too many fields in `%s`. expected %d, found %d",
				validator.currentDeclName, len(oldFields), len(newFields))
		}

		return fmt.Errorf("too many fields. expected %d, found %d",
			len(oldFields), len(newFields))
	}

	// Put the old field-types against their field-names to a map, for faster lookup.
	oldFiledTypes := map[string]ast.Type{}
	for _, field := range oldFields {
		oldFiledTypes[field.Identifier.Identifier] = field.TypeAnnotation.Type
	}

	for _, newField := range newFields {
		err := validator.visitField(newField, oldFiledTypes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) visitField(newField *ast.FieldDeclaration, oldFiledTypes map[string]ast.Type) error {
	prevFieldName := validator.currentFieldName
	validator.currentFieldName = newField.Identifier.Identifier
	defer func() {
		validator.currentFieldName = prevFieldName
	}()

	oldType := oldFiledTypes[validator.currentFieldName]
	if oldType == nil {
		if validator.isInNestedDecl() {
			return fmt.Errorf("found new field `%s` in `%s",
				validator.currentDeclName,
				validator.currentFieldName)
		}

		return fmt.Errorf("found new field `%s`", validator.currentFieldName)
	}

	return oldType.Equal(newField.TypeAnnotation.Type, validator)
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

	// Then, check whether the fields of the two composite declarations
	// referred by this nominal type, are also compatible.

	var compositeDeclName string
	if isQualifiedName(foundNominalType) {
		compositeDeclName = foundNominalType.NestedIdentifiers[0].Identifier
	} else {
		compositeDeclName = foundNominalType.Identifier.Identifier
	}

	validator.loadCompositeDecls()

	oldCompositeDecl := validator.oldNestedCompositeDecls[compositeDeclName]
	if oldCompositeDecl == nil {
		// If the declaration is not available, that means this is an imported contract.
		// Thus, no need to validate anymore.
		return nil
	}

	newCompositeDecl := validator.newNestedCompositeDecls[compositeDeclName]
	if newCompositeDecl == nil {
		// If the declaration is not available, that means this is an imported contract.
		// Thus, no need to validate anymore. Ideally shouldn't reach here.
		return nil
	}

	return validator.checkDeclarationUpdatability(oldCompositeDecl, newCompositeDecl)
}

func (validator *ContractUpdateValidator) CheckOptionalTypeEquality(expected *ast.OptionalType, found ast.Type) error {
	foundOptionalType, ok := found.(*ast.OptionalType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	return expected.Type.Equal(foundOptionalType.Type, validator)
}

func (validator *ContractUpdateValidator) CheckVariableSizedTypeEquality(expected *ast.VariableSizedType, found ast.Type) error {
	foundVarSizedType, ok := found.(*ast.VariableSizedType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	return expected.Type.Equal(foundVarSizedType.Type, validator)
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
	return expected.Type.Equal(foundConstSizedType.Type, validator)
}

func (validator *ContractUpdateValidator) CheckDictionaryTypeEquality(expected *ast.DictionaryType, found ast.Type) error {
	foundDictionaryType, ok := found.(*ast.DictionaryType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	err := expected.KeyType.Equal(foundDictionaryType.KeyType, validator)
	if err != nil {
		return err
	}

	return expected.ValueType.Equal(foundDictionaryType.ValueType, validator)
}

func (validator *ContractUpdateValidator) CheckRestrictedTypeEquality(expected *ast.RestrictedType, found ast.Type) error {
	foundRestrictedType, ok := found.(*ast.RestrictedType)
	if !ok {
		return validator.getTypeMismatchError(expected, found)
	}

	err := expected.Type.Equal(foundRestrictedType.Type, validator)
	if err != nil || len(expected.Restrictions) != len(foundRestrictedType.Restrictions) {
		return validator.getTypeMismatchError(expected, found)
	}

	for index, expectedRestriction := range expected.Restrictions {
		foundRestriction := foundRestrictedType.Restrictions[index]
		err := expectedRestriction.Equal(foundRestriction, validator)
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

	err := expected.Type.Equal(foundInstType.Type, validator)
	if err != nil || len(expected.TypeArguments) != len(foundInstType.TypeArguments) {
		return validator.getTypeMismatchError(expected, found)
	}

	for index, typeArgs := range expected.TypeArguments {
		otherTypeArgs := foundInstType.TypeArguments[index]
		err := typeArgs.Type.Equal(otherTypeArgs.Type, validator)
		if err != nil {
			return validator.getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (validator *ContractUpdateValidator) CheckFunctionTypeEquality(_ *ast.FunctionType, _ ast.Type) error {
	// Non storable type
	panic(errors.NewUnreachableError())
}

func (validator *ContractUpdateValidator) CheckReferenceTypeEquality(_ *ast.ReferenceType, _ ast.Type) error {
	// Non storable type
	panic(errors.NewUnreachableError())
}

func (validator *ContractUpdateValidator) loadCompositeDecls() {
	if validator.oldNestedCompositeDecls == nil {
		validator.oldNestedCompositeDecls = map[string]*ast.CompositeDeclaration{}
		for _, nestedComposite := range getDeclarationMembers(validator.oldDeclaration).Composites() {
			validator.oldNestedCompositeDecls[nestedComposite.Identifier.Identifier] = nestedComposite
		}
	}

	if validator.newNestedCompositeDecls == nil {
		validator.newNestedCompositeDecls = map[string]*ast.CompositeDeclaration{}
		for _, nestedComposite := range getDeclarationMembers(validator.newDeclaration).Composites() {
			validator.newNestedCompositeDecls[nestedComposite.Identifier.Identifier] = nestedComposite
		}
	}
}

func (validator *ContractUpdateValidator) checkNameEquality(expectedType *ast.NominalType, foundType *ast.NominalType) bool {
	isExpectedQualifiedName := isQualifiedName(expectedType)
	isFoundQualifiedName := isQualifiedName(foundType)

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

	return checkSliceEquality(expectedType.NestedIdentifiers, foundType.NestedIdentifiers)
}

func (validator *ContractUpdateValidator) checkIdentifierEquality(
	qualifiedNominalType *ast.NominalType,
	simpleNominalType *ast.NominalType) bool {

	// Situation:
	// qualifiedNominalType -> identifier: A, nestedIdentifiers: [foo, bar, ...]
	// simpleNominalType -> identifier: foo,  nestedIdentifiers: [bar, ...]

	// If the first identifier (i.e: 'A') refers to a composite decl that is not the enclosing contract,
	// then it must be referring to an imported contract. That means the two types are no longer the same.
	if qualifiedNominalType.Identifier.Identifier != validator.entryPointName {
		return false
	}

	if qualifiedNominalType.NestedIdentifiers[0].Identifier != simpleNominalType.Identifier.Identifier {
		return false
	}

	return checkSliceEquality(simpleNominalType.NestedIdentifiers, qualifiedNominalType.NestedIdentifiers[1:])
}

func (validator *ContractUpdateValidator) getTypeMismatchError(
	expectedType ast.Type,
	foundType ast.Type) error {

	if validator.isInNestedDecl() {
		return fmt.Errorf("type annotation does not match for field `%s` in `%s`. expected `%s`, found `%s`",
			validator.currentFieldName,
			validator.currentDeclName,
			expectedType,
			foundType)
	}

	return fmt.Errorf("type annotation does not match for field `%s`. expected `%s`, found `%s`",
		validator.currentFieldName,
		expectedType,
		foundType)
}

// isInNestedDecl checks whether the currently visiting declaration is a nested one.
func (validator *ContractUpdateValidator) isInNestedDecl() bool {
	return validator.entryPointName != validator.currentDeclName
}

func getDeclarationName(declaration ast.Declaration) string {
	switch decl := declaration.(type) {
	case *ast.CompositeDeclaration:
		return decl.Identifier.Identifier
	case *ast.InterfaceDeclaration:
		return decl.Identifier.Identifier
	default:
		panic(errors.NewUnreachableError())
	}
}

func getDeclarationMembers(declaration ast.Declaration) *ast.Members {
	switch decl := declaration.(type) {
	case *ast.CompositeDeclaration:
		return decl.Members
	case *ast.InterfaceDeclaration:
		return decl.Members
	default:
		panic(errors.NewUnreachableError())
	}
}

func checkSliceEquality(expected []ast.Identifier, found []ast.Identifier) bool {
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

func isQualifiedName(nominalType *ast.NominalType) bool {
	return len(nominalType.NestedIdentifiers) > 0
}
