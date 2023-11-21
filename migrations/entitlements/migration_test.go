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
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	checkerUtils "github.com/onflow/cadence/runtime/tests/checker"
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

func TestConvertToEntitledValue(t *testing.T) {
	t.Parallel()

	var uuid uint64

	storage := interpreter.NewInMemoryStorage(nil)

	testAddress := common.MustBytesToAddress([]byte{0x1})

	code := `
		access(all) entitlement E
		access(all) entitlement F
		access(all) entitlement G

		access(all) entitlement mapping M {
			E -> F
			F -> G
		}

		access(all)  struct S {
			access(E) let eField: Int
			access(F) let fField: String
			init() {
				self.eField = 0
				self.fField = ""
			}
		}

		access(all) resource interface I {
			access(E) let eField: Int
		}

		access(all) resource interface J {
			access(G) let gField: Int
		}

		access(all) resource R: I, J {
			access(E) let eField: Int
			access(G) let gField: Int
			access(E, G) let egField: Int
			init() {
				self.egField = 0
				self.eField = 1
				self.gField = 2
			}
		}

		access(all) resource Nested {
			access(E | F) let efField: @R
			init() {
				self.efField <- create R()
			}
		}

		access(all) fun makeS(): S {
			return S()
		}

		access(all) fun makeR(): @R {
			return <- create R()
		}

		access(all) fun makeNested(): @Nested {
			return <- create Nested()
		}
	`
	checker, err := checkerUtils.ParseAndCheckWithOptions(t,
		code,
		checkerUtils.ParseAndCheckOptions{},
	)

	require.NoError(t, err)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage: storage,
			UUIDHandler: func() (uint64, error) {
				uuid++
				return uuid, nil
			},
		},
	)

	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	rValue, err := inter.Invoke("makeR")
	require.NoError(t, err)
	sValue, err := inter.Invoke("makeS")
	require.NoError(t, err)
	nestedValue, err := inter.Invoke("makeNested")
	require.NoError(t, err)

	// &S

	unentitledSRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, sValue, inter.MustSemaTypeOfValue(sValue))
	unentitledSRefStaticType := unentitledSRef.StaticType(inter)

	entitledSRef := interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"S.test.E", "S.test.F"} },
			2,
			sema.Conjunction,
		),
		sValue,
		inter.MustSemaTypeOfValue(sValue),
	)
	entitledSRefStaticType := entitledSRef.StaticType(inter)

	// &R

	unentitledRRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, rValue, inter.MustSemaTypeOfValue(rValue))
	unentitledRRefStaticType := unentitledRRef.StaticType(inter)

	entitledRRef := interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"S.test.E", "S.test.G"} },
			2,
			sema.Conjunction,
		),
		rValue,
		inter.MustSemaTypeOfValue(rValue),
	)
	entitledRRefStaticType := entitledRRef.StaticType(inter)

	// &{I}

	intersectionIType := sema.NewIntersectionType(inter, []*sema.InterfaceType{checker.Elaboration.InterfaceType("S.test.I")})
	unentitledIRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, rValue, intersectionIType)

	entitledIRef := interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"S.test.E"} },
			1,
			sema.Conjunction,
		),
		rValue,
		intersectionIType,
	)

	// legacy Capability<&R{I}>

	legacyIntersectionType := interpreter.ConvertSemaToStaticType(inter, intersectionIType).(*interpreter.IntersectionStaticType)
	legacyIntersectionType.LegacyType = rValue.StaticType(inter)
	unentitledLegacyReferenceStaticType := interpreter.NewReferenceStaticType(
		inter,
		interpreter.UnauthorizedAccess,
		legacyIntersectionType,
	)

	unentitledLegacyCapability := interpreter.NewCapabilityValue(
		inter,
		0,
		interpreter.NewAddressValue(inter, testAddress),
		unentitledLegacyReferenceStaticType,
	)

	entitledConvertedLegacyReferenceStaticType := interpreter.NewReferenceStaticType(
		inter,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"S.test.E"} },
			1,
			sema.Conjunction,
		),
		rValue.StaticType(inter),
	)

	entitledLegacyConvertedCapability := interpreter.NewCapabilityValue(
		inter,
		0,
		interpreter.NewAddressValue(inter, testAddress),
		entitledConvertedLegacyReferenceStaticType,
	)

	// &{I, J}

	intersectionIJType := sema.NewIntersectionType(
		inter,
		[]*sema.InterfaceType{
			checker.Elaboration.InterfaceType("S.test.I"),
			checker.Elaboration.InterfaceType("S.test.J"),
		},
	)
	unentitledIJRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, rValue, intersectionIJType)

	entitledIJRef := interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"S.test.E", "S.test.G"} },
			2,
			sema.Conjunction,
		),
		rValue,
		intersectionIJType,
	)

	// &Nested

	unentitledNestedRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, nestedValue, inter.MustSemaTypeOfValue(nestedValue))
	unentitledNestedRefStaticType := unentitledNestedRef.StaticType(inter)

	entitledNestedRef := interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"S.test.E", "S.test.F"} },
			2,
			sema.Conjunction,
		),
		nestedValue,
		inter.MustSemaTypeOfValue(nestedValue),
	)
	entitledNestedRefStaticType := entitledNestedRef.StaticType(inter)

	tests := []struct {
		Input  interpreter.Value
		Output interpreter.Value
		Name   string
	}{
		{
			Input:  rValue,
			Output: rValue,
			Name:   "R",
		},
		{
			Input:  sValue,
			Output: sValue,
			Name:   "S",
		},
		{
			Input:  nestedValue,
			Output: nestedValue,
			Name:   "Nested",
		},
		{
			Input:  unentitledSRef,
			Output: entitledSRef,
			Name:   "&S",
		},
		{
			Input: interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(inter, unentitledSRefStaticType),
				testAddress,
				unentitledSRef,
			),
			Output: interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(inter, entitledSRefStaticType),
				testAddress,
				entitledSRef,
			),
			Name: "[&S]",
		},
		{
			Input: interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(inter, interpreter.PrimitiveStaticTypeMetaType),
				testAddress,
				interpreter.NewTypeValue(inter, unentitledSRefStaticType),
			),
			Output: interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(inter, interpreter.PrimitiveStaticTypeMetaType),
				testAddress,
				interpreter.NewTypeValue(inter, entitledSRefStaticType),
			),
			Name: "[Type]",
		},
		{
			Input: interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewDictionaryStaticType(inter, interpreter.PrimitiveStaticTypeInt, unentitledSRefStaticType),
				interpreter.NewIntValueFromInt64(inter, 0),
				unentitledSRef,
			),
			Output: interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewDictionaryStaticType(inter, interpreter.PrimitiveStaticTypeInt, entitledSRefStaticType),
				interpreter.NewIntValueFromInt64(inter, 0),
				entitledSRef,
			),
			Name: "{Int: &S}",
		},
		{
			Input: interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewDictionaryStaticType(inter, interpreter.PrimitiveStaticTypeInt, interpreter.PrimitiveStaticTypeMetaType),
				interpreter.NewIntValueFromInt64(inter, 0),
				interpreter.NewTypeValue(inter, unentitledSRefStaticType),
			),
			Output: interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewDictionaryStaticType(inter, interpreter.PrimitiveStaticTypeInt, interpreter.PrimitiveStaticTypeMetaType),
				interpreter.NewIntValueFromInt64(inter, 0),
				interpreter.NewTypeValue(inter, entitledSRefStaticType),
			),
			Name: "{Int: Type}",
		},
		{
			Input:  unentitledRRef,
			Output: entitledRRef,
			Name:   "&R",
		},
		{
			Input:  unentitledIRef,
			Output: entitledIRef,
			Name:   "&{I}",
		},
		{
			Input:  unentitledIJRef,
			Output: entitledIJRef,
			Name:   "&{I, J}",
		},
		{
			Input: interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(inter, unentitledRRefStaticType),
				testAddress,
				unentitledRRef,
			),
			Output: interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(inter, entitledRRefStaticType),
				testAddress,
				entitledRRef,
			),
			Name: "[&R]",
		},
		{
			Input:  unentitledNestedRef,
			Output: entitledNestedRef,
			Name:   "&Nested",
		},
		{
			Input: interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(inter, unentitledNestedRefStaticType),
				testAddress,
				unentitledNestedRef,
			),
			Output: interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewVariableSizedStaticType(inter, entitledNestedRefStaticType),
				testAddress,
				entitledNestedRef,
			),
			Name: "[&Nested]",
		},
		{
			Input: interpreter.NewCapabilityValue(
				inter,
				0,
				interpreter.NewAddressValue(inter, testAddress),
				unentitledSRefStaticType,
			),
			Output: interpreter.NewCapabilityValue(
				inter,
				0,
				interpreter.NewAddressValue(inter, testAddress),
				entitledSRefStaticType,
			),
			Name: "Capability<&S>",
		},
		{
			Input: interpreter.NewCapabilityValue(
				inter,
				0,
				interpreter.NewAddressValue(inter, testAddress),
				unentitledRRefStaticType,
			),
			Output: interpreter.NewCapabilityValue(
				inter,
				0,
				interpreter.NewAddressValue(inter, testAddress),
				entitledRRefStaticType,
			),
			Name: "Capability<&R>",
		},
		{
			Input:  unentitledLegacyCapability,
			Output: entitledLegacyConvertedCapability,
			Name:   "Capability<&R{I}>",
		},
		{
			Input: interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.UnauthorizedAccess,
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(inter, rValue.StaticType(inter)),
					testAddress,
					rValue.Clone(inter),
				),
				sema.NewVariableSizedType(inter, inter.MustSemaTypeOfValue(rValue)),
			),
			Output: interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID { return []common.TypeID{"Mutate", "Insert", "Remove"} },
					3,
					sema.Conjunction,
				),
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(inter, rValue.StaticType(inter)),
					testAddress,
					rValue.Clone(inter),
				),
				sema.NewVariableSizedType(inter, inter.MustSemaTypeOfValue(rValue)),
			),
			Name: "&[R]",
		},
		{
			Input: interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.UnauthorizedAccess,
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(inter, unentitledRRefStaticType),
					testAddress,
					unentitledRRef,
				),
				sema.NewVariableSizedType(
					inter,
					sema.NewReferenceType(
						inter,
						sema.UnauthorizedAccess,
						inter.MustSemaTypeOfValue(rValue),
					),
				),
			),
			Output: interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID { return []common.TypeID{"Mutate", "Insert", "Remove"} },
					3,
					sema.Conjunction,
				),
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(inter, entitledRRefStaticType),
					testAddress,
					entitledRRef,
				),
				sema.NewVariableSizedType(
					inter,
					sema.NewReferenceType(
						inter,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								checker.Elaboration.EntitlementType("S.test.E"),
								checker.Elaboration.EntitlementType("S.test.G"),
							},
							sema.Conjunction,
						),
						inter.MustSemaTypeOfValue(rValue),
					),
				),
			),
			Name: "&[&R]",
		},
		{
			Input: interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.UnauthorizedAccess,
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewDictionaryStaticType(inter, interpreter.PrimitiveStaticTypeInt, rValue.StaticType(inter)),
					interpreter.NewIntValueFromInt64(inter, 0),
					rValue.Clone(inter),
				),
				sema.NewDictionaryType(inter, sema.IntType, inter.MustSemaTypeOfValue(rValue)),
			),
			Output: interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID { return []common.TypeID{"Mutate", "Insert", "Remove"} },
					3,
					sema.Conjunction,
				),
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewDictionaryStaticType(inter, interpreter.PrimitiveStaticTypeInt, rValue.StaticType(inter)),
					interpreter.NewIntValueFromInt64(inter, 0),
					rValue.Clone(inter),
				),
				sema.NewDictionaryType(inter, sema.IntType, inter.MustSemaTypeOfValue(rValue)),
			),
			Name: "&{Int: R}",
		},
		{
			Input: interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.UnauthorizedAccess,
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewDictionaryStaticType(inter, interpreter.PrimitiveStaticTypeInt, unentitledRRefStaticType),
					interpreter.NewIntValueFromInt64(inter, 0),
					unentitledRRef,
				),
				sema.NewDictionaryType(inter, sema.IntType,
					sema.NewReferenceType(
						inter,
						sema.UnauthorizedAccess,
						inter.MustSemaTypeOfValue(rValue),
					),
				),
			),
			Output: interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID { return []common.TypeID{"Mutate", "Insert", "Remove"} },
					3,
					sema.Conjunction,
				),
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewDictionaryStaticType(inter, interpreter.PrimitiveStaticTypeInt, entitledRRefStaticType),
					interpreter.NewIntValueFromInt64(inter, 0),
					entitledRRef,
				),
				sema.NewDictionaryType(inter, sema.IntType,
					sema.NewReferenceType(
						inter,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								checker.Elaboration.EntitlementType("S.test.E"),
								checker.Elaboration.EntitlementType("S.test.G"),
							},
							sema.Conjunction,
						),
						inter.MustSemaTypeOfValue(rValue),
					),
				),
			),
			Name: "&{Int: &R}",
		},
	}

	getStaticType := func(v interpreter.Value) interpreter.StaticType {
		// for reference types, we want to use the borrow type, rather than the type of the referenced value
		if referenceValue, isReferenceValue := v.(*interpreter.EphemeralReferenceValue); isReferenceValue {
			return interpreter.NewReferenceStaticType(
				inter,
				referenceValue.Authorization,
				interpreter.ConvertSemaToStaticType(inter, referenceValue.BorrowedType),
			)
		} else {
			return v.StaticType(inter)
		}
	}

	for _, test := range tests {
		var runtimeTypeTest struct {
			Input  interpreter.Value
			Output interpreter.Value
			Name   string
		}
		runtimeTypeTest.Input = interpreter.NewTypeValue(inter, getStaticType(test.Input.Clone(inter)))
		runtimeTypeTest.Output = interpreter.NewTypeValue(inter, getStaticType(test.Output.Clone(inter)))
		runtimeTypeTest.Name = "runtime type " + test.Name

		tests = append(tests, runtimeTypeTest)
	}

	for _, test := range tests {
		var optionalValueTest struct {
			Input  interpreter.Value
			Output interpreter.Value
			Name   string
		}
		optionalValueTest.Input = interpreter.NewSomeValueNonCopying(inter, test.Input.Clone(inter))
		optionalValueTest.Output = interpreter.NewSomeValueNonCopying(inter, test.Output.Clone(inter))
		optionalValueTest.Name = "optional " + test.Name

		tests = append(tests, optionalValueTest)
	}

	var referencePeekingEqual func(interpreter.EquatableValue, interpreter.Value) bool

	// equality that peeks inside referneces to use structural equality for their values
	referencePeekingEqual = func(input interpreter.EquatableValue, output interpreter.Value) bool {
		switch v := input.(type) {
		case *interpreter.SomeValue:
			otherSome, ok := output.(*interpreter.SomeValue)
			if !ok {
				return false
			}

			switch innerValue := v.InnerValue(inter, interpreter.EmptyLocationRange).(type) {
			case interpreter.EquatableValue:
				return referencePeekingEqual(
					innerValue,
					otherSome.InnerValue(inter, interpreter.EmptyLocationRange),
				)
			default:
				return innerValue == otherSome.InnerValue(inter, interpreter.EmptyLocationRange)
			}
		case *interpreter.EphemeralReferenceValue:
			otherReference, ok := output.(*interpreter.EphemeralReferenceValue)
			if !ok || !v.Authorization.Equal(otherReference.Authorization) {
				return false
			}

			if v.BorrowedType == nil && otherReference.BorrowedType != nil {
				return false
			} else if !v.BorrowedType.Equal(otherReference.BorrowedType) {
				return false
			}

			switch innerValue := v.Value.(type) {
			case interpreter.EquatableValue:
				return innerValue.Equal(inter, interpreter.EmptyLocationRange, otherReference.Value)
			default:
				return innerValue == otherReference.Value
			}
		}

		return input.Equal(inter, interpreter.EmptyLocationRange, output)
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			inter.ConvertValueToEntitlements(test.Input, ConvertToEntitledType)
			switch input := test.Input.(type) {
			case interpreter.EquatableValue:
				require.True(t, referencePeekingEqual(input, test.Output))
			default:
				require.Equal(t, input, test.Output)
			}
		})
	}
}
