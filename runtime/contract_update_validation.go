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
	compositeDecl := program.SoleContractDeclaration()
	if compositeDecl != nil {
		return compositeDecl
	}

	interfaceDecl := program.SoleContractInterfaceDeclaration()
	if interfaceDecl != nil {
		return interfaceDecl
	}

	validator.report(&ContractNotFoundError{
		Range: ast.NewRangeFromPositioned(program),
	})

	return nil
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
			name:    oldDeclaration.DeclarationIdentifier().Identifier,
			oldKind: oldDeclaration.DeclarationKind(),
			newKind: newDeclaration.DeclarationKind(),
			Range:   ast.NewRangeFromPositioned(newDeclaration.DeclarationIdentifier()),
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
	// Any additional field may cause crashes/garbage-values when deserializing the
	// already-stored data. However, having less number of fields is fine for now. It will
	// only leave some unused-data, and will not do any harm to the programs that are running.

	for _, newField := range newFields {
		oldField := oldFields[newField.Identifier.Identifier]
		if oldField == nil {
			validator.report(&ExtraneousFieldError{
				declName:  newDeclaration.DeclarationIdentifier().Identifier,
				fieldName: newField.Identifier.Identifier,
				Range:     ast.NewRangeFromPositioned(newField.Identifier),
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
			declName:  validator.currentDecl.DeclarationIdentifier().Identifier,
			fieldName: newField.Identifier.Identifier,
			err:       err,
			Range:     ast.NewRangeFromPositioned(newField.TypeAnnotation),
		})
	}
}

func (validator *ContractUpdateValidator) checkNestedDeclarations(
	oldDeclaration ast.Declaration,
	newDeclaration ast.Declaration,
) {

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

	// Check nested structs, enums, etc.
	newNestedCompositeDecls := newDeclaration.DeclarationMembers().Composites()
	for _, newNestedDecl := range newNestedCompositeDecls {
		oldNestedDecl, found := getOldCompositeOrInterfaceDecl(newNestedDecl.Identifier.Identifier)
		if !found {
			// Then its a new declaration
			continue
		}

		validator.checkDeclarationUpdatability(oldNestedDecl, newNestedDecl)
	}

	// Check nested interfaces.
	newNestedInterfaces := newDeclaration.DeclarationMembers().Interfaces()
	for _, newNestedDecl := range newNestedInterfaces {
		oldNestedDecl, found := getOldCompositeOrInterfaceDecl(newNestedDecl.Identifier.Identifier)
		if !found {
			// Then this is a new declaration.
			continue
		}

		validator.checkDeclarationUpdatability(oldNestedDecl, newNestedDecl)
	}

	// Check enum-cases, if theres any.
	validator.checkEnumCases(oldDeclaration, newDeclaration)
}

// checkEnumCases validates updating enum cases. Updated enum must:
//   - Have at-most the same number of enum-cases as the old enum (Removing is allowed, but no additions).
//   - Preserve the order of the old enum-cases (Removals from middle is not allowed, swapping is not allowed).
func (validator *ContractUpdateValidator) checkEnumCases(oldDeclaration ast.Declaration, newDeclaration ast.Declaration) {
	newEnumCases := newDeclaration.DeclarationMembers().EnumCases()
	oldEnumCases := oldDeclaration.DeclarationMembers().EnumCases()

	oldEnumCaseCount := len(oldEnumCases)

	// Validate the the new enum cases.
	for index, newEnumCase := range newEnumCases {
		// If there are no more old enum-cases, then these are newly added enum-cases.
		// Thus report an error.
		if index >= oldEnumCaseCount {
			validator.report(&ExtraneousFieldError{
				declName:  newDeclaration.DeclarationIdentifier().Identifier,
				fieldName: newEnumCase.Identifier.Identifier,
				Range:     ast.NewRangeFromPositioned(newEnumCase),
			})

			continue
		}

		oldEnumCase := oldEnumCases[index]
		if oldEnumCase.Identifier.Identifier != newEnumCase.Identifier.Identifier {
			validator.report(&EnumCaseMismatchError{
				expectedName: oldEnumCase.Identifier.Identifier,
				foundName:    newEnumCase.Identifier.Identifier,
				Range:        ast.NewRangeFromPositioned(newEnumCase),
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

	oldConformances := oldDecl.Conformances
	newConformances := newDecl.Conformances

	if len(oldConformances) != len(newConformances) {
		validator.report(&ConformanceCountMismatchError{
			expected: len(oldConformances),
			found:    len(newConformances),
			Range:    ast.NewRangeFromPositioned(newDecl.Identifier),
		})

		// If the lengths are not the same, trying to match the conformance
		// may result in too many regression errors. hence return.
		return
	}

	for index, oldConformance := range oldConformances {
		newConformance := newConformances[index]
		err := oldConformance.CheckEqual(newConformance, validator)
		if err != nil {
			validator.report(&ConformanceMismatchError{
				declName: newDecl.Identifier.Identifier,
				err:      err,
				Range:    ast.NewRangeFromPositioned(newConformance),
			})
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
		contractName: validator.contractName,
		errors:       validator.errors,
		location:     validator.location,
	}
}

func getTypeMismatchError(expectedType ast.Type, foundType ast.Type) *TypeMismatchError {
	return &TypeMismatchError{
		expectedType: expectedType,
		foundType:    foundType,
		Range:        ast.NewRangeFromPositioned(foundType),
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
