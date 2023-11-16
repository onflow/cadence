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
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/require"
)

func TestConvertToEntitledType(t *testing.T) {

	t.Parallel()

	testLocation := common.StringLocation("test")

	entitlementE := sema.NewEntitlementType(nil, testLocation, "E")
	entitlementF := sema.NewEntitlementType(nil, testLocation, "F")
	entitlementG := sema.NewEntitlementType(nil, testLocation, "G")

	eAccess := sema.NewEntitlementSetAccess([]*sema.EntitlementType{entitlementE}, sema.Conjunction)
	fAccess := sema.NewEntitlementSetAccess([]*sema.EntitlementType{entitlementF}, sema.Conjunction)
	eOrFAccess := sema.NewEntitlementSetAccess([]*sema.EntitlementType{entitlementE, entitlementF}, sema.Disjunction)
	eAndFAccess := sema.NewEntitlementSetAccess([]*sema.EntitlementType{entitlementE, entitlementF}, sema.Conjunction)
	eAndGAccess := sema.NewEntitlementSetAccess([]*sema.EntitlementType{entitlementE, entitlementG}, sema.Conjunction)
	eFAndGAccess := sema.NewEntitlementSetAccess([]*sema.EntitlementType{entitlementE, entitlementF, entitlementG}, sema.Conjunction)

	mapM := sema.NewEntitlementMapType(nil, testLocation, "M")
	mapM.Relations = []sema.EntitlementRelation{
		{
			Input:  entitlementE,
			Output: entitlementF,
		},
		{
			Input:  entitlementF,
			Output: entitlementG,
		},
	}
	mapAccess := sema.NewEntitlementMapAccess(mapM)

	compositeStructWithOnlyE := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "S",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeStructWithOnlyE.Members.Set(
		"foo",
		sema.NewFieldMember(nil, compositeStructWithOnlyE, eAccess, ast.VariableKindConstant, "foo", sema.IntType, ""),
	)

	compositeResourceWithOnlyF := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "R",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeResourceWithOnlyF.Members.Set(
		"bar",
		sema.NewFieldMember(nil, compositeResourceWithOnlyF, fAccess, ast.VariableKindConstant, "bar", sema.IntType, ""),
	)
	compositeResourceWithOnlyF.Members.Set(
		"baz",
		sema.NewFieldMember(nil, compositeResourceWithOnlyF, fAccess, ast.VariableKindConstant, "baz", compositeStructWithOnlyE, ""),
	)

	compositeResourceWithEOrF := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "R",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeResourceWithEOrF.Members.Set(
		"qux",
		sema.NewFieldMember(nil, compositeResourceWithEOrF, eOrFAccess, ast.VariableKindConstant, "qux", sema.IntType, ""),
	)

	compositeTwoFields := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "S",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeTwoFields.Members.Set(
		"foo",
		sema.NewFieldMember(nil, compositeTwoFields, eAccess, ast.VariableKindConstant, "foo", sema.IntType, ""),
	)
	compositeTwoFields.Members.Set(
		"bar",
		sema.NewFieldMember(nil, compositeTwoFields, fAccess, ast.VariableKindConstant, "bar", sema.IntType, ""),
	)

	interfaceTypeWithEAndG := &sema.InterfaceType{
		Location:      testLocation,
		Identifier:    "I",
		CompositeKind: common.CompositeKindResource,
		Members:       &sema.StringMemberOrderedMap{},
	}
	interfaceTypeWithEAndG.Members.Set(
		"foo",
		sema.NewFunctionMember(nil, interfaceTypeWithEAndG, eAndGAccess, "foo", &sema.FunctionType{}, ""),
	)

	interfaceTypeInheriting := &sema.InterfaceType{
		Location:                      testLocation,
		Identifier:                    "J",
		CompositeKind:                 common.CompositeKindResource,
		Members:                       &sema.StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceTypeWithEAndG},
	}

	compositeTypeInheriting := &sema.CompositeType{
		Location:                      testLocation,
		Identifier:                    "RI",
		Kind:                          common.CompositeKindResource,
		Members:                       &sema.StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceTypeInheriting},
	}

	compositeTypeWithMap := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "RI",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeTypeWithMap.Members.Set(
		"foo",
		sema.NewFunctionMember(nil, compositeTypeWithMap, mapAccess, "foo", &sema.FunctionType{}, ""),
	)

	interfaceTypeWithMap := &sema.InterfaceType{
		Location:      testLocation,
		Identifier:    "RI",
		CompositeKind: common.CompositeKindResource,
		Members:       &sema.StringMemberOrderedMap{},
	}
	interfaceTypeWithMap.Members.Set(
		"foo",
		sema.NewFunctionMember(nil, interfaceTypeWithMap, mapAccess, "foo", &sema.FunctionType{}, ""),
	)

	compositeTypeWithCapField := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "RI",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeTypeWithCapField.Members.Set(
		"foo",
		sema.NewFieldMember(
			nil, compositeTypeWithCapField, sema.UnauthorizedAccess, ast.VariableKindConstant, "foo",
			sema.NewCapabilityType(nil,
				sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeInheriting),
			),
			"",
		),
	)

	interfaceTypeWithCapField := &sema.InterfaceType{
		Location:      testLocation,
		Identifier:    "RI",
		CompositeKind: common.CompositeKindResource,
		Members:       &sema.StringMemberOrderedMap{},
	}
	interfaceTypeWithCapField.Members.Set(
		"foo",
		sema.NewFieldMember(
			nil, interfaceTypeWithCapField, sema.UnauthorizedAccess, ast.VariableKindConstant, "foo",
			sema.NewCapabilityType(nil,
				sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeInheriting),
			),
			"",
		),
	)

	interfaceTypeInheritingCapField := &sema.InterfaceType{
		Location:                      testLocation,
		Identifier:                    "J",
		CompositeKind:                 common.CompositeKindResource,
		Members:                       &sema.StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceTypeWithCapField},
	}

	compositeTypeInheritingCapField := &sema.CompositeType{
		Location:                      testLocation,
		Identifier:                    "RI",
		Kind:                          common.CompositeKindResource,
		Members:                       &sema.StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceTypeInheritingCapField},
	}

	tests := []struct {
		Input  sema.Type
		Output sema.Type
		Name   string
	}{
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, sema.IntType),
			Output: sema.NewReferenceType(nil, sema.UnauthorizedAccess, sema.IntType),
			Name:   "int",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, &sema.FunctionType{}),
			Output: sema.NewReferenceType(nil, sema.UnauthorizedAccess, &sema.FunctionType{}),
			Name:   "function",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeStructWithOnlyE),
			Output: sema.NewReferenceType(nil, eAccess, compositeStructWithOnlyE),
			Name:   "composite E",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeResourceWithOnlyF),
			Output: sema.NewReferenceType(nil, fAccess, compositeResourceWithOnlyF),
			Name:   "composite F",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeResourceWithEOrF),
			Output: sema.NewReferenceType(nil, eAndFAccess, compositeResourceWithEOrF),
			Name:   "composite E or F",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTwoFields),
			Output: sema.NewReferenceType(nil, eAndFAccess, compositeTwoFields),
			Name:   "composite E and F",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeWithEAndG),
			Output: sema.NewReferenceType(nil, eAndGAccess, interfaceTypeWithEAndG),
			Name:   "interface E and G",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeInheriting),
			Output: sema.NewReferenceType(nil, eAndGAccess, interfaceTypeInheriting),
			Name:   "interface inheritance",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTypeInheriting),
			Output: sema.NewReferenceType(nil, eAndGAccess, compositeTypeInheriting),
			Name:   "composite inheritance",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTypeWithMap),
			Output: sema.NewReferenceType(nil, eAndFAccess, compositeTypeWithMap),
			Name:   "composite map",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeWithMap),
			Output: sema.NewReferenceType(nil, eAndFAccess, interfaceTypeWithMap),
			Name:   "interface map",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, sema.NewCapabilityType(nil, sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTypeWithMap))),
			Output: sema.NewReferenceType(nil, sema.UnauthorizedAccess, sema.NewCapabilityType(nil, sema.NewReferenceType(nil, eAndFAccess, compositeTypeWithMap))),
			Name:   "reference to capability",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, sema.NewIntersectionType(nil, []*sema.InterfaceType{interfaceTypeInheriting, interfaceTypeWithMap})),
			Output: sema.NewReferenceType(nil, eFAndGAccess, sema.NewIntersectionType(nil, []*sema.InterfaceType{interfaceTypeInheriting, interfaceTypeWithMap})),
			Name:   "intersection",
		},
		// no change
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTypeWithCapField),
			Output: sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTypeWithCapField),
			Name:   "composite with capability field",
		},
		// no change
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeWithCapField),
			Output: sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeWithCapField),
			Name:   "interface with capability field",
		},
		// no change
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTypeInheritingCapField),
			Output: sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTypeInheritingCapField),
			Name:   "composite inheriting capability field",
		},
		// no change
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeInheritingCapField),
			Output: sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeInheritingCapField),
			Name:   "interface inheriting capability field",
		},
		// TODO: add tests for array and dictionary entitlements once the mutability changes are merged
	}

	// create capability versions of all the existing tests
	for _, test := range tests {
		var capabilityTest struct {
			Input  sema.Type
			Output sema.Type
			Name   string
		}
		capabilityTest.Input = sema.NewCapabilityType(nil, test.Input)
		capabilityTest.Output = sema.NewCapabilityType(nil, test.Output)
		capabilityTest.Name = "capability " + test.Name

		tests = append(tests, capabilityTest)
	}

	// create optional versions of all the existing tests
	for _, test := range tests {
		var optionalTest struct {
			Input  sema.Type
			Output sema.Type
			Name   string
		}
		optionalTest.Input = sema.NewOptionalType(nil, test.Input)
		optionalTest.Output = sema.NewOptionalType(nil, test.Output)
		optionalTest.Name = "optional " + test.Name

		tests = append(tests, optionalTest)
	}

	var compareTypesRecursively func(t *testing.T, expected sema.Type, actual sema.Type)
	compareTypesRecursively = func(t *testing.T, expected sema.Type, actual sema.Type) {
		require.IsType(t, expected, actual)

		switch expected := expected.(type) {
		case *sema.ReferenceType:
			actual := actual.(*sema.ReferenceType)
			require.IsType(t, expected.Authorization, actual.Authorization)
			require.True(t, expected.Authorization.Equal(actual.Authorization))
			compareTypesRecursively(t, expected.Type, actual.Type)
		case *sema.OptionalType:
			actual := actual.(*sema.OptionalType)
			compareTypesRecursively(t, expected.Type, actual.Type)
		case *sema.CapabilityType:
			actual := actual.(*sema.CapabilityType)
			compareTypesRecursively(t, expected.BorrowType, actual.BorrowType)
		}
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			compareTypesRecursively(t, ConvertToEntitledType(test.Input), test.Output)
		})
	}

}
