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

package entitlements

import (
	"testing"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/stretchr/testify/require"
)

func TestConvertToEntitledType(t *testing.T) {

	t.Parallel()

	testLocation := common.StringLocation("test")

	entitlementE := NewEntitlementType(nil, testLocation, "E")
	entitlementF := NewEntitlementType(nil, testLocation, "F")
	entitlementG := NewEntitlementType(nil, testLocation, "G")

	eAccess := NewEntitlementSetAccess([]*EntitlementType{entitlementE}, Conjunction)
	fAccess := NewEntitlementSetAccess([]*EntitlementType{entitlementF}, Conjunction)
	eOrFAccess := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementF}, Disjunction)
	eAndFAccess := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementF}, Conjunction)
	eAndGAccess := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementG}, Conjunction)
	eFAndGAccess := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementF, entitlementG}, Conjunction)

	mapM := NewEntitlementMapType(nil, testLocation, "M")
	mapM.Relations = []EntitlementRelation{
		{
			Input:  entitlementE,
			Output: entitlementF,
		},
		{
			Input:  entitlementF,
			Output: entitlementG,
		},
	}
	mapAccess := NewEntitlementMapAccess(mapM)

	compositeStructWithOnlyE := &CompositeType{
		Location:   testLocation,
		Identifier: "S",
		Kind:       common.CompositeKindStructure,
		Members:    &StringMemberOrderedMap{},
	}
	compositeStructWithOnlyE.Members.Set(
		"foo",
		NewFieldMember(nil, compositeStructWithOnlyE, eAccess, ast.VariableKindConstant, "foo", IntType, ""),
	)

	compositeResourceWithOnlyF := &CompositeType{
		Location:   testLocation,
		Identifier: "R",
		Kind:       common.CompositeKindResource,
		Members:    &StringMemberOrderedMap{},
	}
	compositeResourceWithOnlyF.Members.Set(
		"bar",
		NewFieldMember(nil, compositeResourceWithOnlyF, fAccess, ast.VariableKindConstant, "bar", IntType, ""),
	)
	compositeResourceWithOnlyF.Members.Set(
		"baz",
		NewFieldMember(nil, compositeResourceWithOnlyF, fAccess, ast.VariableKindConstant, "baz", compositeStructWithOnlyE, ""),
	)

	compositeResourceWithEOrF := &CompositeType{
		Location:   testLocation,
		Identifier: "R",
		Kind:       common.CompositeKindResource,
		Members:    &StringMemberOrderedMap{},
	}
	compositeResourceWithEOrF.Members.Set(
		"qux",
		NewFieldMember(nil, compositeResourceWithEOrF, eOrFAccess, ast.VariableKindConstant, "qux", IntType, ""),
	)

	compositeTwoFields := &CompositeType{
		Location:   testLocation,
		Identifier: "S",
		Kind:       common.CompositeKindStructure,
		Members:    &StringMemberOrderedMap{},
	}
	compositeTwoFields.Members.Set(
		"foo",
		NewFieldMember(nil, compositeTwoFields, eAccess, ast.VariableKindConstant, "foo", IntType, ""),
	)
	compositeTwoFields.Members.Set(
		"bar",
		NewFieldMember(nil, compositeTwoFields, fAccess, ast.VariableKindConstant, "bar", IntType, ""),
	)

	interfaceTypeWithEAndG := &InterfaceType{
		Location:      testLocation,
		Identifier:    "I",
		CompositeKind: common.CompositeKindResource,
		Members:       &StringMemberOrderedMap{},
	}
	interfaceTypeWithEAndG.Members.Set(
		"foo",
		NewFunctionMember(nil, interfaceTypeWithEAndG, eAndGAccess, "foo", &FunctionType{}, ""),
	)

	interfaceTypeInheriting := &InterfaceType{
		Location:                      testLocation,
		Identifier:                    "J",
		CompositeKind:                 common.CompositeKindResource,
		Members:                       &StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*InterfaceType{interfaceTypeWithEAndG},
	}

	compositeTypeInheriting := &CompositeType{
		Location:                      testLocation,
		Identifier:                    "RI",
		Kind:                          common.CompositeKindResource,
		Members:                       &StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*InterfaceType{interfaceTypeInheriting},
	}

	compositeTypeWithMap := &CompositeType{
		Location:   testLocation,
		Identifier: "RI",
		Kind:       common.CompositeKindResource,
		Members:    &StringMemberOrderedMap{},
	}
	compositeTypeWithMap.Members.Set(
		"foo",
		NewFunctionMember(nil, compositeTypeWithMap, mapAccess, "foo", &FunctionType{}, ""),
	)

	interfaceTypeWithMap := &InterfaceType{
		Location:      testLocation,
		Identifier:    "RI",
		CompositeKind: common.CompositeKindResource,
		Members:       &StringMemberOrderedMap{},
	}
	interfaceTypeWithMap.Members.Set(
		"foo",
		NewFunctionMember(nil, interfaceTypeWithMap, mapAccess, "foo", &FunctionType{}, ""),
	)

	compositeTypeWithCapField := &CompositeType{
		Location:   testLocation,
		Identifier: "RI",
		Kind:       common.CompositeKindResource,
		Members:    &StringMemberOrderedMap{},
	}
	compositeTypeWithCapField.Members.Set(
		"foo",
		NewFieldMember(
			nil, compositeTypeWithCapField, UnauthorizedAccess, ast.VariableKindConstant, "foo",
			NewCapabilityType(nil,
				NewReferenceType(nil, interfaceTypeInheriting, UnauthorizedAccess),
			),
			"",
		),
	)

	interfaceTypeWithCapField := &InterfaceType{
		Location:      testLocation,
		Identifier:    "RI",
		CompositeKind: common.CompositeKindResource,
		Members:       &StringMemberOrderedMap{},
	}
	interfaceTypeWithCapField.Members.Set(
		"foo",
		NewFieldMember(
			nil, interfaceTypeWithCapField, UnauthorizedAccess, ast.VariableKindConstant, "foo",
			NewCapabilityType(nil,
				NewReferenceType(nil, interfaceTypeInheriting, UnauthorizedAccess),
			),
			"",
		),
	)

	interfaceTypeInheritingCapField := &InterfaceType{
		Location:                      testLocation,
		Identifier:                    "J",
		CompositeKind:                 common.CompositeKindResource,
		Members:                       &StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*InterfaceType{interfaceTypeWithCapField},
	}

	compositeTypeInheritingCapField := &CompositeType{
		Location:                      testLocation,
		Identifier:                    "RI",
		Kind:                          common.CompositeKindResource,
		Members:                       &StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*InterfaceType{interfaceTypeInheritingCapField},
	}

	tests := []struct {
		Input  Type
		Output Type
		Name   string
	}{
		{
			Input:  NewReferenceType(nil, IntType, UnauthorizedAccess),
			Output: NewReferenceType(nil, IntType, UnauthorizedAccess),
			Name:   "int",
		},
		{
			Input:  NewReferenceType(nil, &FunctionType{}, UnauthorizedAccess),
			Output: NewReferenceType(nil, &FunctionType{}, UnauthorizedAccess),
			Name:   "function",
		},
		{
			Input:  NewReferenceType(nil, compositeStructWithOnlyE, UnauthorizedAccess),
			Output: NewReferenceType(nil, compositeStructWithOnlyE, eAccess),
			Name:   "composite E",
		},
		{
			Input:  NewReferenceType(nil, compositeResourceWithOnlyF, UnauthorizedAccess),
			Output: NewReferenceType(nil, compositeResourceWithOnlyF, fAccess),
			Name:   "composite F",
		},
		{
			Input:  NewReferenceType(nil, compositeResourceWithEOrF, UnauthorizedAccess),
			Output: NewReferenceType(nil, compositeResourceWithEOrF, eAndFAccess),
			Name:   "composite E or F",
		},
		{
			Input:  NewReferenceType(nil, compositeTwoFields, UnauthorizedAccess),
			Output: NewReferenceType(nil, compositeTwoFields, eAndFAccess),
			Name:   "composite E and F",
		},
		{
			Input:  NewReferenceType(nil, interfaceTypeWithEAndG, UnauthorizedAccess),
			Output: NewReferenceType(nil, interfaceTypeWithEAndG, eAndGAccess),
			Name:   "interface E and G",
		},
		{
			Input:  NewReferenceType(nil, interfaceTypeInheriting, UnauthorizedAccess),
			Output: NewReferenceType(nil, interfaceTypeInheriting, eAndGAccess),
			Name:   "interface inheritance",
		},
		{
			Input:  NewReferenceType(nil, compositeTypeInheriting, UnauthorizedAccess),
			Output: NewReferenceType(nil, compositeTypeInheriting, eAndGAccess),
			Name:   "composite inheritance",
		},
		{
			Input:  NewReferenceType(nil, compositeTypeWithMap, UnauthorizedAccess),
			Output: NewReferenceType(nil, compositeTypeWithMap, eAndFAccess),
			Name:   "composite map",
		},
		{
			Input:  NewReferenceType(nil, interfaceTypeWithMap, UnauthorizedAccess),
			Output: NewReferenceType(nil, interfaceTypeWithMap, eAndFAccess),
			Name:   "interface map",
		},
		{
			Input:  NewReferenceType(nil, NewCapabilityType(nil, NewReferenceType(nil, compositeTypeWithMap, UnauthorizedAccess)), UnauthorizedAccess),
			Output: NewReferenceType(nil, NewCapabilityType(nil, NewReferenceType(nil, compositeTypeWithMap, eAndFAccess)), UnauthorizedAccess),
			Name:   "reference to capability",
		},
		{
			Input:  NewReferenceType(nil, NewIntersectionType(nil, []*InterfaceType{interfaceTypeInheriting, interfaceTypeWithMap}), UnauthorizedAccess),
			Output: NewReferenceType(nil, NewIntersectionType(nil, []*InterfaceType{interfaceTypeInheriting, interfaceTypeWithMap}), eFAndGAccess),
			Name:   "intersection",
		},
		// no change
		{
			Input:  NewReferenceType(nil, compositeTypeWithCapField, UnauthorizedAccess),
			Output: NewReferenceType(nil, compositeTypeWithCapField, UnauthorizedAccess),
			Name:   "composite with capability field",
		},
		// no change
		{
			Input:  NewReferenceType(nil, interfaceTypeWithCapField, UnauthorizedAccess),
			Output: NewReferenceType(nil, interfaceTypeWithCapField, UnauthorizedAccess),
			Name:   "interface with capability field",
		},
		// no change
		{
			Input:  NewReferenceType(nil, compositeTypeInheritingCapField, UnauthorizedAccess),
			Output: NewReferenceType(nil, compositeTypeInheritingCapField, UnauthorizedAccess),
			Name:   "composite inheriting capability field",
		},
		// no change
		{
			Input:  NewReferenceType(nil, interfaceTypeInheritingCapField, UnauthorizedAccess),
			Output: NewReferenceType(nil, interfaceTypeInheritingCapField, UnauthorizedAccess),
			Name:   "interface inheriting capability field",
		},
		// TODO: add tests for array and dictionary entitlements once the mutability changes are merged
	}

	// create capability versions of all the existing tests
	for _, test := range tests {
		var capabilityTest struct {
			Input  Type
			Output Type
			Name   string
		}
		capabilityTest.Input = NewCapabilityType(nil, test.Input)
		capabilityTest.Output = NewCapabilityType(nil, test.Output)
		capabilityTest.Name = "capability " + test.Name

		tests = append(tests, capabilityTest)
	}

	// create optional versions of all the existing tests
	for _, test := range tests {
		var optionalTest struct {
			Input  Type
			Output Type
			Name   string
		}
		optionalTest.Input = NewOptionalType(nil, test.Input)
		optionalTest.Output = NewOptionalType(nil, test.Output)
		optionalTest.Name = "optional " + test.Name

		tests = append(tests, optionalTest)
	}

	var compareTypesRecursively func(t *testing.T, expected Type, actual Type)
	compareTypesRecursively = func(t *testing.T, expected Type, actual Type) {
		require.IsType(t, expected, actual)

		switch expected := expected.(type) {
		case *ReferenceType:
			actual := actual.(*ReferenceType)
			require.IsType(t, expected.Authorization, actual.Authorization)
			require.True(t, expected.Authorization.Equal(actual.Authorization))
			compareTypesRecursively(t, expected.Type, actual.Type)
		case *OptionalType:
			actual := actual.(*OptionalType)
			compareTypesRecursively(t, expected.Type, actual.Type)
		case *CapabilityType:
			actual := actual.(*CapabilityType)
			compareTypesRecursively(t, expected.BorrowType, actual.BorrowType)
		}
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			compareTypesRecursively(t, ConvertToEntitledType(nil, test.Input), test.Output)
		})
	}

}
