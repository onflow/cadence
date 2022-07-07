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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

var _ ast.TypeEqualityChecker = &TypeComparator{}

type TypeComparator struct {
	RootDeclIdentifier *Identifier
}

func (c *TypeComparator) CheckNominalTypeEquality(expected *ast.NominalType, found ast.Type) error {
	foundNominalType, ok := found.(*ast.NominalType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	// First check whether the names are equal.
	ok = c.checkNameEquality(expected, foundNominalType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return nil
}

func (c *TypeComparator) CheckOptionalTypeEquality(expected *ast.OptionalType, found ast.Type) error {
	foundOptionalType, ok := found.(*ast.OptionalType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(foundOptionalType.Type, c)
}

func (c *TypeComparator) CheckVariableSizedTypeEquality(expected *ast.VariableSizedType, found ast.Type) error {
	foundVarSizedType, ok := found.(*ast.VariableSizedType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	return expected.Type.CheckEqual(foundVarSizedType.Type, c)
}

func (c *TypeComparator) CheckConstantSizedTypeEquality(expected *ast.ConstantSizedType, found ast.Type) error {
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
	return expected.Type.CheckEqual(foundConstSizedType.Type, c)
}

func (c *TypeComparator) CheckDictionaryTypeEquality(expected *ast.DictionaryType, found ast.Type) error {
	foundDictionaryType, ok := found.(*ast.DictionaryType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	err := expected.KeyType.CheckEqual(foundDictionaryType.KeyType, c)
	if err != nil {
		return err
	}

	return expected.ValueType.CheckEqual(foundDictionaryType.ValueType, c)
}

func (c *TypeComparator) CheckRestrictedTypeEquality(expected *ast.RestrictedType, found ast.Type) error {
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
		err := expected.Type.CheckEqual(foundRestrictedType.Type, c)
		if err != nil {
			return getTypeMismatchError(expected, found)
		}
	}

	if len(expected.Restrictions) != len(foundRestrictedType.Restrictions) {
		return getTypeMismatchError(expected, found)
	}

	for index, expectedRestriction := range expected.Restrictions {
		foundRestriction := foundRestrictedType.Restrictions[index]
		err := expectedRestriction.CheckEqual(foundRestriction, c)
		if err != nil {
			return getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (c *TypeComparator) CheckInstantiationTypeEquality(expected *ast.InstantiationType, found ast.Type) error {
	foundInstType, ok := found.(*ast.InstantiationType)
	if !ok {
		return getTypeMismatchError(expected, found)
	}

	err := expected.Type.CheckEqual(foundInstType.Type, c)
	if err != nil || len(expected.TypeArguments) != len(foundInstType.TypeArguments) {
		return getTypeMismatchError(expected, found)
	}

	for index, typeArgs := range expected.TypeArguments {
		otherTypeArgs := foundInstType.TypeArguments[index]
		err := typeArgs.Type.CheckEqual(otherTypeArgs.Type, c)
		if err != nil {
			return getTypeMismatchError(expected, found)
		}
	}

	return nil
}

func (c *TypeComparator) CheckFunctionTypeEquality(expected *ast.FunctionType, found ast.Type) error {
	foundFuncType, ok := found.(*ast.FunctionType)
	if !ok || len(expected.ParameterTypeAnnotations) != len(foundFuncType.ParameterTypeAnnotations) {
		return getTypeMismatchError(expected, found)
	}

	for index, expectedParamType := range expected.ParameterTypeAnnotations {
		foundParamType := foundFuncType.ParameterTypeAnnotations[index]
		err := expectedParamType.Type.CheckEqual(foundParamType.Type, c)
		if err != nil {
			return getTypeMismatchError(expected, found)
		}
	}

	return expected.ReturnTypeAnnotation.Type.CheckEqual(foundFuncType.ReturnTypeAnnotation.Type, c)
}

func (c *TypeComparator) CheckReferenceTypeEquality(expected *ast.ReferenceType, found ast.Type) error {
	refType, ok := found.(*ast.ReferenceType)
	if !ok {
		return getTypeMismatchError(expected, found)
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
	if expectedType.Identifier.Identifier != foundType.Identifier.Identifier {
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

func getTypeMismatchError(expectedType ast.Type, foundType ast.Type) *TypeMismatchError {
	return &TypeMismatchError{
		ExpectedType: expectedType,
		FoundType:    foundType,
		Range:        ast.NewUnmeteredRangeFromPositioned(foundType),
	}
}
