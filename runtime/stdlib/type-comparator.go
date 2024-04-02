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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

var _ ast.TypeEqualityChecker = &TypeComparator{}

type TypeComparator struct {
	RootDeclIdentifier                *ast.Identifier
	expectedIdentifierImportLocations map[string]common.Location
	foundIdentifierImportLocations    map[string]common.Location
}

func (c *TypeComparator) CheckNominalTypeEquality(expected *ast.NominalType, found ast.Type) error {
	foundNominalType, ok := found.(*ast.NominalType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	// First check whether the names are equal.
	ok = c.checkNameEquality(expected, foundNominalType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	return nil
}

func (c *TypeComparator) CheckOptionalTypeEquality(expected *ast.OptionalType, found ast.Type) error {
	foundOptionalType, ok := found.(*ast.OptionalType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(foundOptionalType.Type, c)
}

func (c *TypeComparator) CheckVariableSizedTypeEquality(expected *ast.VariableSizedType, found ast.Type) error {
	foundVarSizedType, ok := found.(*ast.VariableSizedType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(foundVarSizedType.Type, c)
}

func (c *TypeComparator) CheckConstantSizedTypeEquality(expected *ast.ConstantSizedType, found ast.Type) error {
	foundConstSizedType, ok := found.(*ast.ConstantSizedType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	// Check size
	if foundConstSizedType.Size.Value.Cmp(expected.Size.Value) != 0 ||
		foundConstSizedType.Size.Base != expected.Size.Base {
		return newTypeMismatchError(expected, found)
	}

	// Check type
	return expected.Type.CheckEqual(foundConstSizedType.Type, c)
}

func (c *TypeComparator) CheckDictionaryTypeEquality(expected *ast.DictionaryType, found ast.Type) error {
	foundDictionaryType, ok := found.(*ast.DictionaryType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	err := expected.KeyType.CheckEqual(foundDictionaryType.KeyType, c)
	if err != nil {
		return err
	}

	return expected.ValueType.CheckEqual(foundDictionaryType.ValueType, c)
}

func (c *TypeComparator) CheckIntersectionTypeEquality(expected *ast.IntersectionType, found ast.Type) error {
	foundIntersectionType, ok := found.(*ast.IntersectionType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	if len(expected.Types) != len(foundIntersectionType.Types) {
		return newTypeMismatchError(expected, found)
	}

	for index, expectedIntersectedType := range expected.Types {
		foundType := foundIntersectionType.Types[index]
		err := expectedIntersectedType.CheckEqual(foundType, c)
		if err != nil {
			return newTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (c *TypeComparator) CheckInstantiationTypeEquality(expected *ast.InstantiationType, found ast.Type) error {
	foundInstType, ok := found.(*ast.InstantiationType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	err := expected.Type.CheckEqual(foundInstType.Type, c)
	if err != nil || len(expected.TypeArguments) != len(foundInstType.TypeArguments) {
		return newTypeMismatchError(expected, found)
	}

	for index, typeArgs := range expected.TypeArguments {
		otherTypeArgs := foundInstType.TypeArguments[index]
		err := typeArgs.Type.CheckEqual(otherTypeArgs.Type, c)
		if err != nil {
			return newTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (c *TypeComparator) CheckFunctionTypeEquality(expected *ast.FunctionType, found ast.Type) error {
	foundFuncType, ok := found.(*ast.FunctionType)
	if !ok || len(expected.ParameterTypeAnnotations) != len(foundFuncType.ParameterTypeAnnotations) {
		return newTypeMismatchError(expected, found)
	}

	for index, expectedParamType := range expected.ParameterTypeAnnotations {
		foundParamType := foundFuncType.ParameterTypeAnnotations[index]
		err := expectedParamType.Type.CheckEqual(foundParamType.Type, c)
		if err != nil {
			return newTypeMismatchError(expected, found)
		}
	}

	return expected.ReturnTypeAnnotation.Type.CheckEqual(foundFuncType.ReturnTypeAnnotation.Type, c)
}

func (c *TypeComparator) CheckReferenceTypeEquality(expected *ast.ReferenceType, found ast.Type) error {
	refType, ok := found.(*ast.ReferenceType)
	if !ok {
		return newTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(refType.Type, c)
}

func (c *TypeComparator) checkNameEquality(expectedType *ast.NominalType, foundType *ast.NominalType) bool {
	isExpectedQualifiedName := expectedType.IsQualifiedName()
	isFoundQualifiedName := foundType.IsQualifiedName()

	// A field with a composite type can be defined in two ways:
	// 	- Using type name (var x @ResourceName)
	//	- Using qualified type name (var x @ContractName.ResourceName)

	if isExpectedQualifiedName && !isFoundQualifiedName {
		return c.checkIdentifierEquality(expectedType, foundType)
	}

	if isFoundQualifiedName && !isExpectedQualifiedName {
		return c.checkIdentifierEquality(foundType, expectedType)
	}

	// At this point, either both are qualified names, or both are simple names.
	// Thus, do a one-to-one match.
	expectedIdentifier := expectedType.Identifier.Identifier
	foundIdentifier := foundType.Identifier.Identifier

	if expectedIdentifier != foundIdentifier {
		return false
	}

	// if the identifier is imported, then it must be imported from the same location in each type
	if c.expectedIdentifierImportLocations[expectedIdentifier] != c.foundIdentifierImportLocations[foundIdentifier] {
		return false
	}

	return identifiersEqual(expectedType.NestedIdentifiers, foundType.NestedIdentifiers)
}

func (c *TypeComparator) checkIdentifierEquality(
	qualifiedNominalType *ast.NominalType,
	simpleNominalType *ast.NominalType,
) bool {

	// Situation:
	// qualifiedNominalType -> identifier: A, nestedIdentifiers: [foo, bar, ...]
	// simpleNominalType -> identifier: foo,  nestedIdentifiers: [bar, ...]

	// If the first identifier (i.e: 'A') refers to a composite decl that is not the enclosing contract,
	// then it must be referring to an imported contract. That means the two types are no longer the same.
	if c.RootDeclIdentifier != nil &&
		qualifiedNominalType.Identifier.Identifier != c.RootDeclIdentifier.Identifier {
		return false
	}

	if qualifiedNominalType.NestedIdentifiers[0].Identifier != simpleNominalType.Identifier.Identifier {
		return false
	}

	return identifiersEqual(simpleNominalType.NestedIdentifiers, qualifiedNominalType.NestedIdentifiers[1:])
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

func newTypeMismatchError(expectedType ast.Type, foundType ast.Type) *TypeMismatchError {
	return &TypeMismatchError{
		ExpectedType: expectedType,
		FoundType:    foundType,
		Range:        ast.NewUnmeteredRangeFromPositioned(foundType),
	}
}
