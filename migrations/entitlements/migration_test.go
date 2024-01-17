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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	checkerUtils "github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
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
			converedType, _ := ConvertToEntitledType(test.Input)
			compareTypesRecursively(t, converedType, test.Output)
		})
	}

}

type testEntitlementsMigration struct {
	inter *interpreter.Interpreter
}

func (testEntitlementsMigration) Name() string {
	return "Test Entitlements Migration"
}

func (m testEntitlementsMigration) Migrate(_ interpreter.AddressPath, value interpreter.Value, _ *interpreter.Interpreter) (interpreter.Value, error) {
	return ConvertValueToEntitlements(m.inter, value), nil
}

func convertEntireTestValue(
	inter *interpreter.Interpreter,
	storage *runtime.Storage,
	address common.Address,
	v interpreter.Value,
) interpreter.Value {
	testMig := testEntitlementsMigration{inter: inter}
	storageMig := migrations.NewStorageMigration(inter, storage)

	migratedValue := storageMig.MigrateNestedValue(
		interpreter.AddressPath{
			Address: address,
			Path:    interpreter.NewPathValue(nil, common.PathDomainStorage, ""),
		},
		v,
		[]migrations.ValueMigration{testMig},
		nil,
	)

	if migratedValue == nil {
		return v
	} else {
		return migratedValue
	}
}

func TestConvertToEntitledValue(t *testing.T) {
	t.Parallel()

	var uuid uint64

	ledger := runtime_utils.NewTestLedger(nil, nil)
	storage := runtime.NewStorage(ledger, nil)

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

	unentitledSRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, sValue, inter.MustSemaTypeOfValue(sValue), interpreter.EmptyLocationRange)
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
		interpreter.EmptyLocationRange,
	)
	entitledSRefStaticType := entitledSRef.StaticType(inter)

	// &R

	unentitledRRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, rValue, inter.MustSemaTypeOfValue(rValue), interpreter.EmptyLocationRange)
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
		interpreter.EmptyLocationRange,
	)
	entitledRRefStaticType := entitledRRef.StaticType(inter)

	// &{I}

	intersectionIType := sema.NewIntersectionType(inter, []*sema.InterfaceType{checker.Elaboration.InterfaceType("S.test.I")})
	unentitledIRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, rValue, intersectionIType, interpreter.EmptyLocationRange)

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
		interpreter.EmptyLocationRange,
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

	unentitledLegacyCapabilityArray := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		interpreter.NewVariableSizedStaticType(inter, unentitledLegacyCapability.StaticType(inter)),
		testAddress,
		unentitledLegacyCapability,
	)

	unentitledLegacyCapabilityOptionalArray := interpreter.NewSomeValueNonCopying(inter, unentitledLegacyCapabilityArray)

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

	entitledLegacyConvertedCapabilityArray := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		interpreter.NewVariableSizedStaticType(inter, entitledLegacyConvertedCapability.StaticType(inter)),
		testAddress,
		entitledLegacyConvertedCapability,
	)

	entitledLegacyConvertedCapabilityOptionalArray := interpreter.NewSomeValueNonCopying(inter, entitledLegacyConvertedCapabilityArray)

	// &{I, J}

	intersectionIJType := sema.NewIntersectionType(
		inter,
		[]*sema.InterfaceType{
			checker.Elaboration.InterfaceType("S.test.I"),
			checker.Elaboration.InterfaceType("S.test.J"),
		},
	)
	unentitledIJRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, rValue, intersectionIJType, interpreter.EmptyLocationRange)

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
		interpreter.EmptyLocationRange,
	)

	// &Nested

	unentitledNestedRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, nestedValue, inter.MustSemaTypeOfValue(nestedValue), interpreter.EmptyLocationRange)
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
		interpreter.EmptyLocationRange,
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
			Input:  unentitledLegacyCapabilityOptionalArray.Clone(inter),
			Output: entitledLegacyConvertedCapabilityOptionalArray.Clone(inter),
			Name:   "[Capability<&R{I}>]? -> [Capability<auth(E) &R>]?",
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
				interpreter.EmptyLocationRange,
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
				interpreter.EmptyLocationRange,
			),
			Name: "&[R]",
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
				interpreter.EmptyLocationRange,
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
				interpreter.EmptyLocationRange,
			),
			Name: "&{Int: R}",
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

	// equality that peeks inside references to use structural equality for their values
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
			convertedValue := convertEntireTestValue(inter, storage, testAddress, test.Input)
			switch convertedValue := convertedValue.(type) {
			case interpreter.EquatableValue:
				require.True(t, referencePeekingEqual(convertedValue, test.Output))
			default:
				require.Equal(t, convertedValue, test.Output)
			}
		})
	}
}

func TestMigrateSimpleContract(t *testing.T) {
	t.Parallel()

	var uuid uint64

	account := common.Address{0x42}
	ledger := NewTestLedger(nil, nil)

	type testCase struct {
		storedValue   interpreter.Value
		expectedValue interpreter.Value
	}

	storage := runtime.NewStorage(ledger, nil)

	code := `
		access(all) entitlement E
		access(all) resource R {
			access(E) fun foo() {}
		}
		access(all) resource T {
			access(all) let cap: Capability<auth(E) &R>?
			init() {
				self.cap = nil
			}
		}
		access(all) fun makeR(): @R {
			return <- create R()
		}
		access(all) fun makeT(): @T {
			return <- create T()
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

	storageIdentifier := common.PathDomainStorage.Identifier()

	err = inter.Interpret()
	require.NoError(t, err)

	rValue, err := inter.Invoke("makeR")
	require.NoError(t, err)

	tValue, err := inter.Invoke("makeT")
	require.NoError(t, err)

	unentitledRRef := interpreter.NewEphemeralReferenceValue(inter, interpreter.UnauthorizedAccess, rValue, inter.MustSemaTypeOfValue(rValue), interpreter.EmptyLocationRange)
	unentitledRRefStaticType := unentitledRRef.StaticType(inter)

	unentitledRCap := interpreter.NewCapabilityValue(
		inter,
		0,
		interpreter.NewAddressValue(inter, account),
		unentitledRRefStaticType,
	)

	entitledRRef := interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"S.test.E"} },
			1,
			sema.Conjunction,
		),
		rValue,
		inter.MustSemaTypeOfValue(rValue),
		interpreter.EmptyLocationRange,
	)
	entitledRRefStaticType := entitledRRef.StaticType(inter)
	entitledRCap := interpreter.NewCapabilityValue(
		inter,
		0,
		interpreter.NewAddressValue(inter, account),
		entitledRRefStaticType,
	)

	tValue.(*interpreter.CompositeValue).SetMember(inter, interpreter.EmptyLocationRange, "cap", unentitledRCap.Clone(inter))

	expeectedTValue := tValue.Clone(inter)
	expeectedTValue.(*interpreter.CompositeValue).SetMember(inter, interpreter.EmptyLocationRange, "cap", entitledRCap.Clone(inter))

	testCases := map[string]testCase{
		"rCap": {
			storedValue: unentitledRCap.Clone(inter),
			expectedValue: interpreter.NewCapabilityValue(
				inter,
				0,
				interpreter.NewAddressValue(inter, account),
				entitledRRefStaticType,
			),
		},
		"rValue": {
			storedValue:   rValue.Clone(inter),
			expectedValue: rValue.Clone(inter),
		},
		"tValue": {
			storedValue:   tValue.Clone(inter),
			expectedValue: expeectedTValue.Clone(inter),
		},
	}

	for name, testCase := range testCases {
		inter.WriteStored(
			account,
			storageIdentifier,
			interpreter.StringStorageMapKey(name),
			testCase.storedValue,
		)
	}

	err = storage.Commit(inter, true)
	require.NoError(t, err)

	// Migrate

	migration := migrations.NewStorageMigration(inter, storage)
	pathMigrator := migration.NewValueMigrationsPathMigrator(nil, NewEntitlementsMigration(inter))
	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		pathMigrator,
	)

	storageMap := storage.GetStorageMap(account, storageIdentifier, false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	iterator := storageMap.Iterator(inter)

	for key, value := iterator.Next(); key != nil; key, value = iterator.Next() {
		identifier := string(key.(interpreter.StringAtreeValue))

		t.Run(identifier, func(t *testing.T) {
			testCase, ok := testCases[identifier]
			require.True(t, ok)

			expectedStoredValue := testCase.expectedValue

			AssertValuesEqual(t, inter, expectedStoredValue, value)
		})
	}
}

func TestMigrateAcrossContracts(t *testing.T) {
	t.Parallel()

	address1 := [8]byte{0, 0, 0, 0, 0, 0, 0, 1}
	address2 := [8]byte{0, 0, 0, 0, 0, 0, 0, 2}

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	interfaces := map[common.Location]*TestRuntimeInterface{}

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{address1}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}
	runtimeInterface1.OnUpdateAccountContractCode = func(location common.AddressLocation, code []byte) error {
		accountCodes[location] = code
		interfaces[location] = runtimeInterface1
		return nil
	}

	runtimeInterface2 := &TestRuntimeInterface{
		Storage: storage,
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{address2}, nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}
	runtimeInterface2.OnUpdateAccountContractCode = func(location common.AddressLocation, code []byte) error {
		accountCodes[location] = code
		interfaces[location] = runtimeInterface2
		return nil
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	oldContract := []byte(`
		access(all) contract C {
			access(all) resource R {
				access(all) fun foo() {}
			}
			access(all) resource T {
				access(all) let cap: Capability<&R>
				init(_ cap: Capability<&R>) {
					self.cap = cap
				}
			}
			access(all) fun makeR(): @R {
				return <- create R()
			}
			access(all) fun makeT(_ cap: Capability<&R>): @T {
				return <- create T(cap)
			}
		}
	`)

	contract := []byte(`
		access(all) contract C {
			access(all) entitlement E
			access(all) resource R {
				access(E) fun foo() {}
			}
			access(all) resource T {
				access(all) let cap: Capability<auth(E) &R>
				init(_ cap: Capability<auth(E) &R>) {
					self.cap = cap
				}
			}
			access(all) fun makeR(): @R {
				return <- create R()
			}
			access(all) fun makeT(_ cap: Capability<auth(E) &R>): @T {
				return <- create T(cap)
			}
		}
	`)

	saveValues := []byte(`
		import C from 0x1	

		transaction {
			prepare(signer: auth(Storage, Capabilities) &Account) {
				let r <- C.makeR()
				signer.storage.save(<-r, to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&C.R>(/storage/foo)
				let t <- C.makeT(cap)
				signer.storage.save(<-t, to: /storage/bar)
			}
		}
	`)

	// Deploy contract to 0x1
	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2
	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// update contract on 0x1
	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	runtimeStorage := runtime.NewStorage(storage, nil)

	inter, err := interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		&interpreter.Config{
			Storage:                       runtimeStorage,
			AtreeValueValidationEnabled:   false,
			AtreeStorageValidationEnabled: false,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				program, err := rt.ParseAndCheckProgram(
					accountCodes[location],
					runtime.Context{
						Interface: interfaces[location],
						Location:  location,
					},
				)
				require.NoError(t, err)

				subInterpreter, err := inter.NewSubInterpreter(program, location)
				require.NoError(t, err)

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		},
	)
	require.NoError(t, err)

	storageIdentifier := common.PathDomainStorage.Identifier()
	storageMap := runtimeStorage.GetStorageMap(address2, storageIdentifier, false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	// Migrate

	migration := migrations.NewStorageMigration(inter, runtimeStorage)
	pathMigrator := migration.NewValueMigrationsPathMigrator(nil, NewEntitlementsMigration(inter))
	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				address1,
				address2,
			},
		},
		pathMigrator,
	)

	value := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("bar"))

	require.IsType(t, &interpreter.CompositeValue{}, value)
	tValue := value.(*interpreter.CompositeValue)
	require.Equal(t, "C.T", tValue.QualifiedIdentifier)

	field := tValue.GetMember(inter, interpreter.EmptyLocationRange, "cap")

	require.IsType(t, &interpreter.CapabilityValue{}, field)
	cap := field.(*interpreter.CapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, cap.BorrowType)
	ref := cap.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

}

func TestMigrateArrayOfValues(t *testing.T) {
	t.Parallel()

	address1 := [8]byte{0, 0, 0, 0, 0, 0, 0, 1}
	address2 := [8]byte{0, 0, 0, 0, 0, 0, 0, 2}

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	interfaces := map[common.Location]*TestRuntimeInterface{}

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{address1}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}
	runtimeInterface1.OnUpdateAccountContractCode = func(location common.AddressLocation, code []byte) error {
		accountCodes[location] = code
		interfaces[location] = runtimeInterface1
		return nil
	}

	runtimeInterface2 := &TestRuntimeInterface{
		Storage: storage,
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{address2}, nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}
	runtimeInterface2.OnUpdateAccountContractCode = func(location common.AddressLocation, code []byte) error {
		accountCodes[location] = code
		interfaces[location] = runtimeInterface2
		return nil
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	oldContract := []byte(`
		access(all) contract C {
			access(all) resource R {
				access(all) fun foo() {}
			}
			access(all) fun makeR(): @R {
				return <- create R()
			}
		}
	`)

	contract := []byte(`
		access(all) contract C {
			access(all) entitlement E
			access(all) resource R {
				access(E) fun foo() {}
			}
			access(all) fun makeR(): @R {
				return <- create R()
			}
		}
	`)

	saveValues := []byte(`
		import C from 0x1	

		transaction {
			prepare(signer: auth(Storage, Capabilities) &Account) {
				let r1 <- C.makeR()
				let r2 <- C.makeR()
				signer.storage.save(<-r1, to: /storage/foo)
				signer.storage.save(<-r2, to: /storage/bar)
				let cap1 = signer.capabilities.storage.issue<&C.R>(/storage/foo)
				let cap2 = signer.capabilities.storage.issue<&C.R>(/storage/bar)
				let arr = [cap1, cap2]
				signer.storage.save(arr, to: /storage/caps)
			}
		}
	`)

	// Deploy contract to 0x1
	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2
	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// update contract on 0x1
	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	runtimeStorage := runtime.NewStorage(storage, nil)

	inter, err := interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		&interpreter.Config{
			Storage:                       runtimeStorage,
			AtreeValueValidationEnabled:   false,
			AtreeStorageValidationEnabled: false,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				program, err := rt.ParseAndCheckProgram(
					accountCodes[location],
					runtime.Context{
						Interface: interfaces[location],
						Location:  location,
					},
				)
				require.NoError(t, err)

				subInterpreter, err := inter.NewSubInterpreter(program, location)
				require.NoError(t, err)

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		},
	)
	require.NoError(t, err)

	storageIdentifier := common.PathDomainStorage.Identifier()
	storageMap := runtimeStorage.GetStorageMap(address2, storageIdentifier, false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	// Migrate

	migration := migrations.NewStorageMigration(inter, runtimeStorage)
	pathMigrator := migration.NewValueMigrationsPathMigrator(nil, NewEntitlementsMigration(inter))
	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				address1,
				address2,
			},
		},
		pathMigrator,
	)

	arrayValue := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("caps"))
	require.IsType(t, &interpreter.ArrayValue{}, arrayValue)
	arrValue := arrayValue.(*interpreter.ArrayValue)
	require.Equal(t, 2, arrValue.Count())

	elementType := arrValue.Type.ElementType()
	require.IsType(t, &interpreter.CapabilityStaticType{}, elementType)
	capElementType := elementType.(*interpreter.CapabilityStaticType)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capElementType.BorrowType)
	ref := capElementType.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	cap1 := arrValue.Get(inter, interpreter.EmptyLocationRange, 0)
	require.IsType(t, &interpreter.CapabilityValue{}, cap1)
	capValue := cap1.(*interpreter.CapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref = capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	cap2 := arrValue.Get(inter, interpreter.EmptyLocationRange, 1)
	require.IsType(t, &interpreter.CapabilityValue{}, cap2)
	capValue = cap1.(*interpreter.CapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref = capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)
}

func TestMigrateDictOfValues(t *testing.T) {
	t.Parallel()

	address1 := [8]byte{0, 0, 0, 0, 0, 0, 0, 1}
	address2 := [8]byte{0, 0, 0, 0, 0, 0, 0, 2}

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	interfaces := map[common.Location]*TestRuntimeInterface{}

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{address1}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}
	runtimeInterface1.OnUpdateAccountContractCode = func(location common.AddressLocation, code []byte) error {
		accountCodes[location] = code
		interfaces[location] = runtimeInterface1
		return nil
	}

	runtimeInterface2 := &TestRuntimeInterface{
		Storage: storage,
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{address2}, nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}
	runtimeInterface2.OnUpdateAccountContractCode = func(location common.AddressLocation, code []byte) error {
		accountCodes[location] = code
		interfaces[location] = runtimeInterface2
		return nil
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	oldContract := []byte(`
		access(all) contract C {
			access(all) resource R {
				access(all) fun foo() {}
			}
			access(all) fun makeR(): @R {
				return <- create R()
			}
		}
	`)

	contract := []byte(`
		access(all) contract C {
			access(all) entitlement E
			access(all) resource R {
				access(E) fun foo() {}
			}
			access(all) fun makeR(): @R {
				return <- create R()
			}
		}
	`)

	saveValues := []byte(`
		import C from 0x1	

		transaction {
			prepare(signer: auth(Storage, Capabilities) &Account) {
				let r1 <- C.makeR()
				let r2 <- C.makeR()
				signer.storage.save(<-r1, to: /storage/foo)
				signer.storage.save(<-r2, to: /storage/bar)
				let cap1 = signer.capabilities.storage.issue<&C.R>(/storage/foo)
				let cap2 = signer.capabilities.storage.issue<&C.R>(/storage/bar)
				let arr = {"a": cap1, "b": cap2}
				signer.storage.save(arr, to: /storage/caps)
			}
		}
	`)

	// Deploy contract to 0x1
	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2
	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// update contract on 0x1
	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	runtimeStorage := runtime.NewStorage(storage, nil)

	inter, err := interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		&interpreter.Config{
			Storage:                       runtimeStorage,
			AtreeValueValidationEnabled:   false,
			AtreeStorageValidationEnabled: false,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				program, err := rt.ParseAndCheckProgram(
					accountCodes[location],
					runtime.Context{
						Interface: interfaces[location],
						Location:  location,
					},
				)
				require.NoError(t, err)

				subInterpreter, err := inter.NewSubInterpreter(program, location)
				require.NoError(t, err)

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		},
	)
	require.NoError(t, err)

	storageIdentifier := common.PathDomainStorage.Identifier()
	storageMap := runtimeStorage.GetStorageMap(address2, storageIdentifier, false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	// Migrate

	migration := migrations.NewStorageMigration(inter, runtimeStorage)
	pathMigrator := migration.NewValueMigrationsPathMigrator(nil, NewEntitlementsMigration(inter))
	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				address1,
				address2,
			},
		},
		pathMigrator,
	)

	dictValue := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("caps"))
	require.IsType(t, &interpreter.DictionaryValue{}, dictValue)
	dictionaryValue := dictValue.(*interpreter.DictionaryValue)

	valueType := dictionaryValue.Type.ValueType
	require.IsType(t, &interpreter.CapabilityStaticType{}, valueType)
	capElementType := valueType.(*interpreter.CapabilityStaticType)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capElementType.BorrowType)
	ref := capElementType.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	cap1, present := dictionaryValue.Get(inter, interpreter.EmptyLocationRange, interpreter.NewUnmeteredStringValue("a"))
	require.True(t, present)
	require.IsType(t, &interpreter.CapabilityValue{}, cap1)
	capValue := cap1.(*interpreter.CapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref = capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	cap2, present := dictionaryValue.Get(inter, interpreter.EmptyLocationRange, interpreter.NewUnmeteredStringValue("b"))
	require.True(t, present)
	require.IsType(t, &interpreter.CapabilityValue{}, cap2)
	capValue = cap1.(*interpreter.CapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref = capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)
}

func TestMigrateDictOfWithTypeValueKey(t *testing.T) {
	t.Parallel()

	address1 := [8]byte{0, 0, 0, 0, 0, 0, 0, 1}
	address2 := [8]byte{0, 0, 0, 0, 0, 0, 0, 2}

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	interfaces := map[common.Location]*TestRuntimeInterface{}

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{address1}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}
	runtimeInterface1.OnUpdateAccountContractCode = func(location common.AddressLocation, code []byte) error {
		accountCodes[location] = code
		interfaces[location] = runtimeInterface1
		return nil
	}

	runtimeInterface2 := &TestRuntimeInterface{
		Storage: storage,
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{address2}, nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}
	runtimeInterface2.OnUpdateAccountContractCode = func(location common.AddressLocation, code []byte) error {
		accountCodes[location] = code
		interfaces[location] = runtimeInterface2
		return nil
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	oldContract := []byte(`
		access(all) contract C {
			access(all) resource R {
				access(all) fun foo() {}
			}
			access(all) fun makeR(): @R {
				return <- create R()
			}
		}
	`)

	contract := []byte(`
		access(all) contract C {
			access(all) entitlement E
			access(all) resource R {
				access(E) fun foo() {}
			}
			access(all) fun makeR(): @R {
				return <- create R()
			}
		}
	`)

	saveValues := []byte(`
		import C from 0x1	

		transaction {
			prepare(signer: auth(Storage, Capabilities) &Account) {
				let r1 <- C.makeR()
				let r2 <- C.makeR()
				let rType = ReferenceType(entitlements: [], type: r1.getType())!
				signer.storage.save(<-r1, to: /storage/foo)
				signer.storage.save(<-r2, to: /storage/bar)
				let cap1 = signer.capabilities.storage.issue<&C.R>(/storage/foo)
				let cap2 = signer.capabilities.storage.issue<&C.R>(/storage/bar)
				let arr = {rType: cap1, Type<Int>(): cap2}
				signer.storage.save(arr, to: /storage/caps)
			}
		}
	`)

	// Deploy contract to 0x1
	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2
	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// update contract on 0x1
	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	runtimeStorage := runtime.NewStorage(storage, nil)

	inter, err := interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		&interpreter.Config{
			Storage:                       runtimeStorage,
			AtreeValueValidationEnabled:   false,
			AtreeStorageValidationEnabled: false,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				program, err := rt.ParseAndCheckProgram(
					accountCodes[location],
					runtime.Context{
						Interface: interfaces[location],
						Location:  location,
					},
				)
				require.NoError(t, err)

				subInterpreter, err := inter.NewSubInterpreter(program, location)
				require.NoError(t, err)

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		},
	)
	require.NoError(t, err)

	storageIdentifier := common.PathDomainStorage.Identifier()
	storageMap := runtimeStorage.GetStorageMap(address2, storageIdentifier, false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	// Migrate

	migration := migrations.NewStorageMigration(inter, runtimeStorage)
	pathMigrator := migration.NewValueMigrationsPathMigrator(nil, NewEntitlementsMigration(inter))
	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				address1,
				address2,
			},
		},
		pathMigrator,
	)

	dictValue := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("caps"))
	require.IsType(t, &interpreter.DictionaryValue{}, dictValue)
	dictionaryValue := dictValue.(*interpreter.DictionaryValue)

	valueType := dictionaryValue.Type.ValueType
	require.IsType(t, &interpreter.CapabilityStaticType{}, valueType)
	capElementType := valueType.(*interpreter.CapabilityStaticType)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capElementType.BorrowType)
	ref := capElementType.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	rTypeKey := interpreter.NewTypeValue(nil, ref)
	intTypeKey := interpreter.NewTypeValue(nil, interpreter.PrimitiveStaticTypeInt)

	cap1, present := dictionaryValue.Get(inter, interpreter.EmptyLocationRange, rTypeKey)
	require.True(t, present)
	require.IsType(t, &interpreter.CapabilityValue{}, cap1)
	capValue := cap1.(*interpreter.CapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref = capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	cap2, present := dictionaryValue.Get(inter, interpreter.EmptyLocationRange, intTypeKey)
	require.True(t, present)
	require.IsType(t, &interpreter.CapabilityValue{}, cap2)
	capValue = cap1.(*interpreter.CapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref = capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID { return []common.TypeID{"A.0000000000000001.C.E"} },
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)
}

func TestConvertDeprecatedTypes(t *testing.T) {

	t.Parallel()

	test := func(ty interpreter.PrimitiveStaticType) {

		t.Run(ty.String(), func(t *testing.T) {
			t.Parallel()

			inter := NewTestInterpreter(t)
			typeValue := interpreter.NewUnmeteredCapabilityValue(
				1,
				interpreter.AddressValue(common.ZeroAddress),
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					ty,
				),
			)

			result := ConvertValueToEntitlements(inter, typeValue)

			require.Nil(t, result)
		})
	}

	for ty := interpreter.PrimitiveStaticType(1); ty < interpreter.PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() || !ty.IsDeprecated() { //nolint:staticcheck
			continue
		}

		test(ty)
	}
}
