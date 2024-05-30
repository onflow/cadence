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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/migrations/statictypes"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	checkerUtils "github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

// TODO: improve
func TestConvertToEntitledType(t *testing.T) {

	t.Parallel()

	inter := NewTestInterpreter(t)
	migration := NewEntitlementsMigration(inter)

	elaboration := sema.NewElaboration(nil)

	inter.Program = &interpreter.Program{
		Elaboration: elaboration,
	}

	testLocation := inter.Location

	// E, F, G

	entitlementE := sema.NewEntitlementType(nil, testLocation, "E")
	elaboration.SetEntitlementType(
		entitlementE.ID(),
		entitlementE,
	)

	entitlementF := sema.NewEntitlementType(nil, testLocation, "F")
	elaboration.SetEntitlementType(
		entitlementF.ID(),
		entitlementF,
	)

	entitlementG := sema.NewEntitlementType(nil, testLocation, "G")
	elaboration.SetEntitlementType(
		entitlementG.ID(),
		entitlementG,
	)

	// auth(E)

	eAccess := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{entitlementE},
		sema.Conjunction,
	)

	// auth(F)

	fAccess := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{entitlementF},
		sema.Conjunction,
	)

	// auth(E | F)

	eOrFAccess := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{entitlementE, entitlementF},
		sema.Disjunction,
	)

	// auth(E, F)

	eAndFAccess := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{entitlementE, entitlementF},
		sema.Conjunction,
	)

	// auth(E, G)

	eAndGAccess := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{entitlementE, entitlementG},
		sema.Conjunction,
	)

	// auth(E, F, G)

	eFAndGAccess := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{entitlementE, entitlementF, entitlementG},
		sema.Conjunction,
	)

	// M (map)

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
	elaboration.SetEntitlementMapType(
		mapM.ID(),
		mapM,
	)

	// S (compositeStructWithOnlyE)

	compositeStructWithOnlyE := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "S",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeStructWithOnlyE.Members.Set(
		"foo",
		sema.NewFieldMember(
			nil,
			compositeStructWithOnlyE,
			eAccess,
			ast.VariableKindConstant,
			"foo",
			sema.IntType,
			"",
		),
	)
	elaboration.SetCompositeType(
		compositeStructWithOnlyE.ID(),
		compositeStructWithOnlyE,
	)

	// R (compositeResourceWithOnlyF)

	compositeResourceWithOnlyF := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "R",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeResourceWithOnlyF.Members.Set(
		"bar",
		sema.NewFieldMember(
			nil,
			compositeResourceWithOnlyF,
			fAccess,
			ast.VariableKindConstant,
			"bar",
			sema.IntType,
			"",
		),
	)
	compositeResourceWithOnlyF.Members.Set(
		"baz",
		sema.NewFieldMember(
			nil,
			compositeResourceWithOnlyF,
			fAccess,
			ast.VariableKindConstant,
			"baz",
			compositeStructWithOnlyE,
			"",
		),
	)
	elaboration.SetCompositeType(
		compositeResourceWithOnlyF.ID(),
		compositeResourceWithOnlyF,
	)

	// R2 (compositeResourceWithEOrF)

	compositeResourceWithEOrF := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "R2",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeResourceWithEOrF.Members.Set(
		"qux",
		sema.NewFieldMember(
			nil,
			compositeResourceWithEOrF,
			eOrFAccess,
			ast.VariableKindConstant,
			"qux",
			sema.IntType,
			"",
		),
	)
	elaboration.SetCompositeType(
		compositeResourceWithEOrF.ID(),
		compositeResourceWithEOrF,
	)

	// S2 (compositeTwoFields)

	compositeTwoFields := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "S2",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeTwoFields.Members.Set(
		"foo",
		sema.NewFieldMember(
			nil,
			compositeTwoFields,
			eAccess,
			ast.VariableKindConstant,
			"foo",
			sema.IntType,
			"",
		),
	)
	compositeTwoFields.Members.Set(
		"bar",
		sema.NewFieldMember(
			nil,
			compositeTwoFields,
			fAccess,
			ast.VariableKindConstant,
			"bar",
			sema.IntType,
			"",
		),
	)
	elaboration.SetCompositeType(
		compositeTwoFields.ID(),
		compositeTwoFields,
	)

	// I (interfaceTypeWithEAndG)

	interfaceTypeWithEAndG := &sema.InterfaceType{
		Location:      testLocation,
		Identifier:    "I",
		CompositeKind: common.CompositeKindResource,
		Members:       &sema.StringMemberOrderedMap{},
	}
	interfaceTypeWithEAndG.Members.Set(
		"foo",
		sema.NewFunctionMember(
			nil,
			interfaceTypeWithEAndG,
			eAndGAccess,
			"foo",
			&sema.FunctionType{},
			"",
		),
	)
	elaboration.SetInterfaceType(
		interfaceTypeWithEAndG.ID(),
		interfaceTypeWithEAndG,
	)

	// J (interfaceTypeInheriting)

	interfaceTypeInheriting := &sema.InterfaceType{
		Location:                      testLocation,
		Identifier:                    "J",
		CompositeKind:                 common.CompositeKindResource,
		Members:                       &sema.StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceTypeWithEAndG},
	}
	elaboration.SetInterfaceType(
		interfaceTypeInheriting.ID(),
		interfaceTypeInheriting,
	)

	// RI (compositeTypeInheriting)

	compositeTypeInheriting := &sema.CompositeType{
		Location:                      testLocation,
		Identifier:                    "RI",
		Kind:                          common.CompositeKindResource,
		Members:                       &sema.StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceTypeInheriting},
	}
	elaboration.SetCompositeType(
		compositeTypeInheriting.ID(),
		compositeTypeInheriting,
	)

	// RI2 (compositeTypeWithMap)

	compositeTypeWithMap := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "RI2",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeTypeWithMap.Members.Set(
		"foo",
		sema.NewFunctionMember(
			nil,
			compositeTypeWithMap,
			mapAccess,
			"foo",
			&sema.FunctionType{},
			"",
		),
	)
	elaboration.SetCompositeType(
		compositeTypeWithMap.ID(),
		compositeTypeWithMap,
	)

	// RI3 (interfaceTypeWithMap)

	interfaceTypeWithMap := &sema.InterfaceType{
		Location:      testLocation,
		Identifier:    "RI3",
		CompositeKind: common.CompositeKindResource,
		Members:       &sema.StringMemberOrderedMap{},
	}
	interfaceTypeWithMap.Members.Set(
		"foo",
		sema.NewFunctionMember(
			nil,
			interfaceTypeWithMap,
			mapAccess,
			"foo",
			&sema.FunctionType{},
			"",
		),
	)
	elaboration.SetInterfaceType(
		interfaceTypeWithMap.ID(),
		interfaceTypeWithMap,
	)

	// RI4 (compositeTypeWithCapField)

	compositeTypeWithCapField := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "RI4",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
	}
	compositeTypeWithCapField.Members.Set(
		"foo",
		sema.NewFieldMember(
			nil,
			compositeTypeWithCapField,
			sema.UnauthorizedAccess,
			ast.VariableKindConstant,
			"foo",
			sema.NewCapabilityType(nil,
				sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeInheriting),
			),
			"",
		),
	)
	elaboration.SetCompositeType(
		compositeTypeWithCapField.ID(),
		compositeTypeWithCapField,
	)

	// RI5 (interfaceTypeWithCapField)

	interfaceTypeWithCapField := &sema.InterfaceType{
		Location:      testLocation,
		Identifier:    "RI5",
		CompositeKind: common.CompositeKindResource,
		Members:       &sema.StringMemberOrderedMap{},
	}
	interfaceTypeWithCapField.Members.Set(
		"foo",
		sema.NewFieldMember(
			nil,
			interfaceTypeWithCapField,
			sema.UnauthorizedAccess,
			ast.VariableKindConstant,
			"foo",
			sema.NewCapabilityType(nil,
				sema.NewReferenceType(nil, sema.UnauthorizedAccess, interfaceTypeInheriting),
			),
			"",
		),
	)
	elaboration.SetInterfaceType(
		interfaceTypeWithCapField.ID(),
		interfaceTypeWithCapField,
	)

	// J2 (interfaceTypeInheritingCapField)

	interfaceTypeInheritingCapField := &sema.InterfaceType{
		Location:                      testLocation,
		Identifier:                    "J2",
		CompositeKind:                 common.CompositeKindResource,
		Members:                       &sema.StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*sema.InterfaceType{interfaceTypeWithCapField},
	}
	elaboration.SetInterfaceType(
		interfaceTypeInheritingCapField.ID(),
		interfaceTypeInheritingCapField,
	)

	// RI6 (compositeTypeInheritingCapField)

	compositeTypeInheritingCapField := &sema.CompositeType{
		Location:   testLocation,
		Identifier: "RI6",
		Kind:       common.CompositeKindResource,
		Members:    &sema.StringMemberOrderedMap{},
		ExplicitInterfaceConformances: []*sema.InterfaceType{
			interfaceTypeInheritingCapField,
		},
	}
	elaboration.SetCompositeType(
		compositeTypeInheritingCapField.ID(),
		compositeTypeInheritingCapField,
	)

	tests := []struct {
		Input  sema.Type
		Output sema.Type
		Name   string
	}{
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, sema.IntType),
			Output: nil,
			Name:   "int",
		},
		{
			Input:  sema.NewReferenceType(nil, sema.UnauthorizedAccess, &sema.FunctionType{ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType)}),
			Output: nil,
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
			Output: sema.NewReferenceType(nil, eOrFAccess, compositeResourceWithEOrF),
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
			Input: sema.NewReferenceType(
				nil,
				sema.UnauthorizedAccess,
				sema.NewCapabilityType(
					nil,
					sema.NewReferenceType(nil, sema.UnauthorizedAccess, compositeTypeWithMap),
				),
			),
			Output: sema.NewReferenceType(
				nil,
				sema.UnauthorizedAccess,
				sema.NewCapabilityType(
					nil,
					sema.NewReferenceType(nil, eAndFAccess, compositeTypeWithMap),
				),
			),
			Name: "reference to capability",
		},
		{
			Input: sema.NewReferenceType(
				nil,
				sema.UnauthorizedAccess,
				sema.NewIntersectionType(
					nil,
					nil,
					[]*sema.InterfaceType{
						interfaceTypeInheriting,
						interfaceTypeWithMap,
					},
				),
			),
			Output: sema.NewReferenceType(
				nil,
				eFAndGAccess,
				sema.NewIntersectionType(
					nil,
					nil,
					[]*sema.InterfaceType{
						interfaceTypeInheriting,
						interfaceTypeWithMap,
					}),
			),
			Name: "intersection",
		},
		{
			Input: sema.NewReferenceType(
				nil,
				sema.UnauthorizedAccess,
				sema.NewOptionalType(
					nil,
					sema.NewIntersectionType(
						nil,
						nil,
						[]*sema.InterfaceType{
							interfaceTypeInheriting,
							interfaceTypeWithMap,
						},
					),
				),
			),
			Output: sema.NewReferenceType(
				nil,
				eFAndGAccess,
				sema.NewOptionalType(
					nil,
					sema.NewIntersectionType(
						nil,
						nil,
						[]*sema.InterfaceType{
							interfaceTypeInheriting,
							interfaceTypeWithMap,
						}),
				),
			),
			Name: "reference to optional",
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
		if test.Output != nil {
			capabilityTest.Output = sema.NewCapabilityType(nil, test.Output)
		}
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
		if test.Output != nil {
			optionalTest.Output = sema.NewOptionalType(nil, test.Output)
		}
		optionalTest.Name = "optional " + test.Name

		tests = append(tests, optionalTest)
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {

			inputStaticType := interpreter.ConvertSemaToStaticType(nil, test.Input)
			convertedType, _ := migration.ConvertToEntitledType(inputStaticType)

			expectedType := interpreter.ConvertSemaToStaticType(nil, test.Output)

			compareTypesRecursively(t, convertedType, expectedType)
		})
	}

}

func compareTypesRecursively(t *testing.T, expected, actual interpreter.StaticType) {
	require.IsType(t, expected, actual)

	switch expected := expected.(type) {
	case *interpreter.ReferenceStaticType:
		actual := actual.(*interpreter.ReferenceStaticType)
		require.IsType(t, expected.Authorization, actual.Authorization)
		require.True(t, expected.Authorization.Equal(actual.Authorization))
		compareTypesRecursively(t, expected.ReferencedType, actual.ReferencedType)
	case *interpreter.OptionalStaticType:
		actual := actual.(*interpreter.OptionalStaticType)
		compareTypesRecursively(t, expected.Type, actual.Type)
	case *interpreter.CapabilityStaticType:
		actual := actual.(*interpreter.CapabilityStaticType)
		compareTypesRecursively(t, expected.BorrowType, actual.BorrowType)
	}
}

type testEntitlementsMigration struct {
	inter *interpreter.Interpreter
}

var _ migrations.ValueMigration = testEntitlementsMigration{}

func (testEntitlementsMigration) Name() string {
	return "Test Entitlements Migration"
}

func (m testEntitlementsMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (
	interpreter.Value,
	error,
) {
	migration := NewEntitlementsMigration(m.inter)
	return migration.ConvertValueToEntitlements(value)
}

func (m testEntitlementsMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testEntitlementsMigration) Domains() map[string]struct{} {
	return nil
}

func convertEntireTestValue(
	t *testing.T,
	inter *interpreter.Interpreter,
	storage *runtime.Storage,
	address common.Address,
	v interpreter.Value,
) interpreter.Value {

	reporter := newTestReporter()

	migration, err := migrations.NewStorageMigration(inter, storage, "test", address)
	require.NoError(t, err)

	migratedValue := migration.MigrateNestedValue(
		interpreter.StorageKey{
			Key:     common.PathDomainStorage.Identifier(),
			Address: address,
		},
		interpreter.StringStorageMapKey("test"),
		v,
		[]migrations.ValueMigration{
			testEntitlementsMigration{inter: inter},
		},
		reporter,
		true,
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	if migratedValue == nil {
		return v
	} else {
		return migratedValue
	}
}

func newIntersectionStaticTypeWithLegacyType(
	legacyType interpreter.StaticType,
	interfaceTypes []*interpreter.InterfaceStaticType,
) *interpreter.IntersectionStaticType {
	intersectionType := interpreter.NewIntersectionStaticType(nil, interfaceTypes)
	intersectionType.LegacyType = legacyType
	return intersectionType
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

        access(all) struct S {
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
    `
	checker, err := checkerUtils.ParseAndCheckWithOptions(t,
		code,
		checkerUtils.ParseAndCheckOptions{},
	)
	require.NoError(t, err)

	location := checker.Location

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		location,
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

	// E, F, G

	eTypeID := location.TypeID(nil, "E")
	fTypeID := location.TypeID(nil, "F")
	gTypeID := location.TypeID(nil, "G")

	// S

	const sQualifiedIdentifier = "S"
	sTypeID := location.TypeID(nil, sQualifiedIdentifier)
	sStaticType := &interpreter.CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: sQualifiedIdentifier,
		TypeID:              sTypeID,
	}

	// R

	const rQualifiedIdentifier = "R"
	rTypeID := location.TypeID(nil, rQualifiedIdentifier)
	rStaticType := &interpreter.CompositeStaticType{
		Location:            location,
		QualifiedIdentifier: rQualifiedIdentifier,
		TypeID:              rTypeID,
	}

	// I

	iTypeID := location.TypeID(nil, "I")
	iStaticType := &interpreter.InterfaceStaticType{
		Location:            location,
		QualifiedIdentifier: "I",
		TypeID:              iTypeID,
	}

	// J

	jTypeID := location.TypeID(nil, "J")
	jStaticType := &interpreter.InterfaceStaticType{
		Location:            location,
		QualifiedIdentifier: "J",
		TypeID:              jTypeID,
	}

	type testCase struct {
		Input  interpreter.StaticType
		Output interpreter.StaticType
		Name   string
	}

	tests := []testCase{
		{
			Name:   "R --> R",
			Input:  rStaticType,
			Output: rStaticType,
		},
		{
			Name:   "S --> S",
			Input:  sStaticType,
			Output: sStaticType,
		},
		{
			Name: "&S --> auth(E, F) &S",
			Input: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				sStaticType,
			),
			Output: interpreter.NewReferenceStaticType(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID {
						return []common.TypeID{
							eTypeID,
							fTypeID,
						}
					},
					2,
					sema.Conjunction,
				),
				sStaticType,
			),
		},
		{
			Name: "&R --> auth(E, G) &R",
			Input: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				rStaticType,
			),
			Output: interpreter.NewReferenceStaticType(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID {
						return []common.TypeID{
							eTypeID,
							gTypeID,
						}
					},
					2,
					sema.Conjunction,
				),
				rStaticType,
			),
		},
		{
			Name: "&{I} --> auth(E) &{I}",
			Input: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				interpreter.NewIntersectionStaticType(
					inter,
					[]*interpreter.InterfaceStaticType{
						iStaticType,
					},
				),
			),
			Output: interpreter.NewReferenceStaticType(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID {
						return []common.TypeID{
							eTypeID,
						}
					},
					1,
					sema.Conjunction,
				),
				interpreter.NewIntersectionStaticType(
					inter,
					[]*interpreter.InterfaceStaticType{
						iStaticType,
					},
				),
			),
		},
		{
			Name: "&{I, J} --> auth(E, G) &{I, J}",
			Input: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				interpreter.NewIntersectionStaticType(
					inter,
					[]*interpreter.InterfaceStaticType{
						iStaticType,
						jStaticType,
					},
				),
			),
			Output: interpreter.NewReferenceStaticType(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID {
						return []common.TypeID{
							eTypeID,
							gTypeID,
						}
					},
					2,
					sema.Conjunction,
				),
				interpreter.NewIntersectionStaticType(
					inter,
					[]*interpreter.InterfaceStaticType{
						iStaticType,
						jStaticType,
					},
				),
			),
		},
		{
			Name: "&AnyStruct{I} --> auth(E) &{I}",
			Input: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				newIntersectionStaticTypeWithLegacyType(
					interpreter.PrimitiveStaticTypeAnyStruct,
					[]*interpreter.InterfaceStaticType{
						iStaticType,
					},
				),
			),
			Output: interpreter.NewReferenceStaticType(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID {
						return []common.TypeID{
							eTypeID,
						}
					},
					1,
					sema.Conjunction,
				),
				interpreter.NewIntersectionStaticType(
					inter,
					[]*interpreter.InterfaceStaticType{
						iStaticType,
					},
				),
			),
		},
		{
			Name: "&AnyStruct{} --> &AnyStruct",
			Input: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				newIntersectionStaticTypeWithLegacyType(
					interpreter.PrimitiveStaticTypeAnyStruct,
					nil,
				),
			),
			Output: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				interpreter.PrimitiveStaticTypeAnyStruct,
			),
		},
		{
			Name: "&R{I} --> auth(E) &R",
			Input: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				newIntersectionStaticTypeWithLegacyType(
					rStaticType,
					[]*interpreter.InterfaceStaticType{
						iStaticType,
					},
				),
			),
			Output: interpreter.NewReferenceStaticType(
				inter,
				interpreter.NewEntitlementSetAuthorization(
					inter,
					func() []common.TypeID {
						return []common.TypeID{
							eTypeID,
						}
					},
					1,
					sema.Conjunction,
				),
				rStaticType,
			),
		},
		{
			// NOTE: NOT auth(E, G) &R!
			Name: "&R{} --> &R",
			Input: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				newIntersectionStaticTypeWithLegacyType(rStaticType, nil),
			),
			Output: interpreter.NewReferenceStaticType(
				inter,
				interpreter.UnauthorizedAccess,
				rStaticType,
			),
		},
	}

	var referencePeekingEqual func(interpreter.EquatableValue, interpreter.Value) bool

	// equality that peeks inside references to use structural equality for their values
	referencePeekingEqual = func(input interpreter.EquatableValue, output interpreter.Value) bool {
		switch v := input.(type) {

		// TODO: support more types (e.g. dictionaries)

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

		case *interpreter.ArrayValue:
			otherArray, ok := output.(*interpreter.ArrayValue)
			if !ok {
				return false
			}

			if v.Count() != otherArray.Count() {
				return false
			}

			for i := 0; i < v.Count(); i++ {
				innerValue := v.Get(inter, interpreter.EmptyLocationRange, i)
				otherInnerValue := otherArray.Get(inter, interpreter.EmptyLocationRange, i)

				switch innerValue := innerValue.(type) {
				case interpreter.EquatableValue:
					if !referencePeekingEqual(
						innerValue,
						otherInnerValue,
					) {
						return false
					}
				default:
					if innerValue != otherInnerValue {
						return false
					}
				}
			}
			return true

		case interpreter.TypeValue:
			// TypeValue considers missing type "unknown"/"invalid",
			// and "unknown"/"invalid" type values unequal.
			// However, we want to consider those equal here for testing/asserting purposes
			other, ok := output.(interpreter.TypeValue)
			if !ok {
				return false
			}

			if other.Type == nil {
				return v.Type == nil
			} else {
				return other.Type.Equal(v.Type)
			}
		}

		return input.Equal(inter, interpreter.EmptyLocationRange, output)
	}

	type valueGenerator struct {
		name string
		wrap func(interpreter.StaticType) interpreter.Value
	}

	valueGenerators := []valueGenerator{
		{
			name: "runtime type value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				return interpreter.NewTypeValue(nil, staticType)
			},
		},
		{
			name: "variable-sized array value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(nil, staticType),
					common.ZeroAddress,
				)
			},
		},
		{
			name: "constant-sized array value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewConstantSizedStaticType(nil, staticType, 1),
					common.ZeroAddress,
				)
			},
		},
		{
			name: "dictionary value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				return interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewDictionaryStaticType(nil, interpreter.PrimitiveStaticTypeInt, staticType),
				)
			},
		},
		{
			name: "ID capability value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				return interpreter.NewCapabilityValue(
					nil,
					1,
					interpreter.AddressValue{},
					staticType,
				)
			},
		},
		{
			name: "path capability value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				return &interpreter.PathCapabilityValue{ //nolint:staticcheck
					BorrowType: staticType,
					Address:    interpreter.AddressValue{},
					Path:       interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "test"),
				}
			},
		},
		{
			name: "published capability value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				return interpreter.NewPublishedValue(
					nil,
					interpreter.AddressValue{},
					interpreter.NewCapabilityValue(
						nil,
						1,
						interpreter.AddressValue{},
						staticType,
					),
				)
			},
		},
		{
			name: "path-link value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				return interpreter.PathLinkValue{ //nolint:staticcheck
					Type: staticType,
					TargetPath: interpreter.NewUnmeteredPathValue(
						common.PathDomainStorage,
						"test",
					),
				}
			},
		},
		{
			name: "storage capability controller value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				referenceStaticType, ok := staticType.(*interpreter.ReferenceStaticType)
				if !ok {
					return nil
				}
				return &interpreter.StorageCapabilityControllerValue{
					BorrowType: referenceStaticType,
				}
			},
		},
		{
			name: "account capability controller value",
			wrap: func(staticType interpreter.StaticType) interpreter.Value {
				referenceStaticType, ok := staticType.(*interpreter.ReferenceStaticType)
				if !ok {
					return nil
				}
				return &interpreter.AccountCapabilityControllerValue{
					BorrowType: referenceStaticType,
				}
			},
		},
	}

	type typeGenerator struct {
		name string
		wrap func(staticType interpreter.StaticType) interpreter.StaticType
	}

	typeGenerators := []typeGenerator{
		{
			name: "as-is",
			wrap: func(staticType interpreter.StaticType) interpreter.StaticType {
				return staticType
			},
		},
		{
			name: "variable-sized array type",
			wrap: func(staticType interpreter.StaticType) interpreter.StaticType {
				return interpreter.NewVariableSizedStaticType(nil, staticType)
			},
		},
		{
			name: "constant-sized array type",
			wrap: func(staticType interpreter.StaticType) interpreter.StaticType {
				return interpreter.NewConstantSizedStaticType(nil, staticType, 1)
			},
		},
		{
			name: "dictionary type",
			wrap: func(staticType interpreter.StaticType) interpreter.StaticType {
				return interpreter.NewDictionaryStaticType(nil, interpreter.PrimitiveStaticTypeInt, staticType)
			},
		},
		{
			name: "optional type",
			wrap: func(staticType interpreter.StaticType) interpreter.StaticType {
				return interpreter.NewOptionalStaticType(nil, staticType)
			},
		},
		{
			name: "capability type",
			wrap: func(staticType interpreter.StaticType) interpreter.StaticType {
				return interpreter.NewCapabilityStaticType(nil, staticType)
			},
		},
	}

	test := func(
		t *testing.T,
		testCase testCase,
		valueGenerator valueGenerator,
		typeGenerator typeGenerator,
	) {
		input := valueGenerator.wrap(typeGenerator.wrap(testCase.Input))
		if input == nil {
			return
		}

		expectedValue := valueGenerator.wrap(typeGenerator.wrap(testCase.Output))

		convertedValue := convertEntireTestValue(t, inter, storage, testAddress, input)

		err := storage.CheckHealth()
		require.NoError(t, err)

		switch convertedValue := convertedValue.(type) {
		case interpreter.EquatableValue:
			require.True(t,
				referencePeekingEqual(convertedValue, expectedValue),
				"expected: %s\nactual: %s",
				expectedValue,
				convertedValue,
			)
		default:
			require.Equal(t, convertedValue, expectedValue)
		}
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {

			for _, valueGenerator := range valueGenerators {
				t.Run(valueGenerator.name, func(t *testing.T) {

					for _, typeGenerator := range typeGenerators {
						t.Run(typeGenerator.name, func(t *testing.T) {

							test(t, testCase, valueGenerator, typeGenerator)
						})
					}
				})
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

	checker, err := checkerUtils.ParseAndCheckWithOptions(t,
		`
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
        `,
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

	unentitledRRef := interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.UnauthorizedAccess,
		rValue,
		inter.MustSemaTypeOfValue(rValue),
		interpreter.EmptyLocationRange,
	)
	unentitledRRefStaticType := unentitledRRef.StaticType(inter)

	unentitledRCap := interpreter.NewCapabilityValue(
		inter,
		1,
		interpreter.NewAddressValue(inter, account),
		unentitledRRefStaticType,
	)

	entitledRRef := interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID {
				return []common.TypeID{"S.test.E"}
			},
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
		1,
		interpreter.NewAddressValue(inter, account),
		entitledRRefStaticType,
	)

	tValue.(*interpreter.CompositeValue).
		SetMember(inter, interpreter.EmptyLocationRange, "cap", unentitledRCap.Clone(inter))

	expectedTValue := tValue.Clone(inter)
	expectedTValue.(*interpreter.CompositeValue).
		SetMember(inter, interpreter.EmptyLocationRange, "cap", entitledRCap.Clone(inter))

	testCases := map[string]testCase{
		"rCap": {
			storedValue: unentitledRCap.Clone(inter),
			expectedValue: interpreter.NewCapabilityValue(
				inter,
				1,
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
			expectedValue: expectedTValue.Clone(inter),
		},
	}

	for name, testCase := range testCases {
		transferredValue := testCase.storedValue.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(account),
			false,
			nil,
			nil,
			true, // standalone values doesn't have a parent container.
		)

		inter.WriteStored(
			account,
			storageIdentifier,
			interpreter.StringStorageMapKey(name),
			transferredValue,
		)
	}

	err = storage.Commit(inter, true)
	require.NoError(t, err)

	// Migrate

	migration, err := migrations.NewStorageMigration(inter, storage, "test", account)
	require.NoError(t, err)

	reporter := newTestReporter()

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			NewEntitlementsMigration(inter),
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

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

func TestNilTypeValue(t *testing.T) {
	t.Parallel()

	migration := NewEntitlementsMigration(nil)
	result, err := migration.ConvertValueToEntitlements(
		interpreter.NewTypeValue(nil, nil),
	)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestNilPathCapabilityValue(t *testing.T) {
	t.Parallel()

	migration := NewEntitlementsMigration(NewTestInterpreter(t))
	result, err := migration.ConvertValueToEntitlements(
		&interpreter.PathCapabilityValue{ //nolint:staticcheck
			Address:    interpreter.NewAddressValue(nil, common.MustBytesToAddress([]byte{0x1})),
			Path:       interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "test"),
			BorrowType: nil,
		},
	)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestMigratePublishedValue(t *testing.T) {
	t.Parallel()

	testAddress := common.Address{0, 0, 0, 0, 0, 0, 0, 1}

	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{testAddress}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
	}

	// Prepare

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
            prepare(signer: auth(Inbox, Storage, Capabilities) &Account) {
                let cap = signer.capabilities.storage.issue<&C.R>(/storage/r)
                signer.storage.save(cap, to: /storage/cap)
                signer.inbox.publish(cap, name: "r_cap", recipient: 0x2)
            }
        }
    `)

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract to 0x1

	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x1

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Update contract on 0x1

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Important: invalidate the loaded program, as it was updated
	runtimeInterface.InvalidateUpdatedPrograms()

	// Migrate

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := newTestReporter()

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			NewEntitlementsMigration(inter),
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	assert.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}]struct{}{
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(1),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     common.PathDomainStorage.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("cap"),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     stdlib.InboxStorageDomain,
				},
				StorageMapKey: interpreter.StringStorageMapKey("r_cap"),
			}: {},
		},
		reporter.migrated,
	)

	inboxStorageIdentifier := stdlib.InboxStorageDomain
	inboxStorageMap := storage.GetStorageMap(
		testAddress,
		inboxStorageIdentifier,
		false,
	)
	require.NotNil(t, inboxStorageMap)
	require.Equal(t, inboxStorageMap.Count(), uint64(1))

	storageMap := storage.GetStorageMap(
		testAddress,
		common.PathDomainStorage.Identifier(),
		false,
	)
	require.NotNil(t, storageMap)
	require.Equal(t, inboxStorageMap.Count(), uint64(1))

	cap1 := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("cap"))
	capValue := cap1.(*interpreter.IDCapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref := capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID {
				return []common.TypeID{"A.0000000000000001.C.E"}
			},
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	publishedValue := inboxStorageMap.ReadValue(nil, interpreter.StringStorageMapKey("r_cap"))

	require.IsType(t, &interpreter.PublishedValue{}, publishedValue)
	publishedValueValue := publishedValue.(*interpreter.PublishedValue).Value

	require.IsType(t, &interpreter.IDCapabilityValue{}, publishedValueValue)
	capabilityValue := publishedValueValue.(*interpreter.IDCapabilityValue)

	require.IsType(t, &interpreter.ReferenceStaticType{}, capabilityValue.BorrowType)
	ref = capabilityValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID {
				return []common.TypeID{"A.0000000000000001.C.E"}
			},
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)
}

func TestMigratePublishedValueAcrossTwoAccounts(t *testing.T) {
	t.Parallel()

	testAddress1 := common.Address{0, 0, 0, 0, 0, 0, 0, 1}
	testAddress2 := common.Address{0, 0, 0, 0, 0, 0, 0, 2}

	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var signingAddress common.Address

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{signingAddress}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
	}

	// Prepare

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
           prepare(signer: auth(Inbox, Storage, Capabilities) &Account) {
               let cap = signer.capabilities.storage.issue<&C.R>(/storage/r)
               signer.storage.save(cap, to: /storage/cap)
               signer.inbox.publish(cap, name: "r_cap", recipient: 0x2)
           }
       }
   `)

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract to 0x1

	signingAddress = testAddress1

	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2

	signingAddress = testAddress2

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Update contract on 0x1

	signingAddress = testAddress1

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	// Important: invalidate the loaded program, as it was updated
	runtimeInterface.InvalidateUpdatedPrograms()

	inboxStorageIdentifier := stdlib.InboxStorageDomain
	inboxStorageMap := storage.GetStorageMap(
		testAddress2,
		inboxStorageIdentifier,
		false,
	)
	require.NotNil(t, inboxStorageMap)
	require.Equal(t, inboxStorageMap.Count(), uint64(1))

	storageIdentifier := common.PathDomainStorage.Identifier()
	storageMap := storage.GetStorageMap(
		testAddress2,
		storageIdentifier,
		false,
	)
	require.NotNil(t, storageMap)
	require.Equal(t, inboxStorageMap.Count(), uint64(1))

	// Migrate

	reporter := newTestReporter()

	for _, address := range []common.Address{
		testAddress1,
		testAddress2,
	} {

		migration, err := migrations.NewStorageMigration(inter, storage, "test", address)
		require.NoError(t, err)

		migrator := migration.NewValueMigrationsPathMigrator(
			reporter,
			NewEntitlementsMigration(inter),
		)
		migration.Migrate(migrator)

		err = migration.Commit()
		require.NoError(t, err)
	}

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	assert.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}]struct{}{
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(1),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     common.PathDomainStorage.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("cap"),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     stdlib.InboxStorageDomain,
				},
				StorageMapKey: interpreter.StringStorageMapKey("r_cap"),
			}: {},
		},
		reporter.migrated,
	)

	cap1 := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("cap"))
	capValue := cap1.(*interpreter.IDCapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref := capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID {
				return []common.TypeID{"A.0000000000000001.C.E"}
			},
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	publishedValue := inboxStorageMap.ReadValue(nil, interpreter.StringStorageMapKey("r_cap"))

	require.IsType(t, &interpreter.PublishedValue{}, publishedValue)
	publishedValueValue := publishedValue.(*interpreter.PublishedValue).Value

	require.IsType(t, &interpreter.IDCapabilityValue{}, publishedValueValue)
	capabilityValue := publishedValueValue.(*interpreter.IDCapabilityValue)

	require.IsType(t, &interpreter.ReferenceStaticType{}, capabilityValue.BorrowType)
	ref = capabilityValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID {
				return []common.TypeID{"A.0000000000000001.C.E"}
			},
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)
}

func TestMigrateAcrossContracts(t *testing.T) {
	t.Parallel()

	testAddress1 := common.Address{0, 0, 0, 0, 0, 0, 0, 1}
	testAddress2 := common.Address{0, 0, 0, 0, 0, 0, 0, 2}

	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var signingAddress common.Address

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{signingAddress}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
	}

	// Prepare

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

	updatedContract := []byte(`
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

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract to 0x1

	signingAddress = testAddress1

	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2

	signingAddress = testAddress2

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Update contract on 0x1

	signingAddress = testAddress1

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", updatedContract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	// Important: invalidate the loaded program, as it was updated
	runtimeInterface.InvalidateUpdatedPrograms()

	storageIdentifier := common.PathDomainStorage.Identifier()
	storageMap := storage.GetStorageMap(testAddress2, storageIdentifier, false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	// Migrate

	reporter := newTestReporter()

	for _, address := range []common.Address{
		testAddress1,
		testAddress2,
	} {
		migration, err := migrations.NewStorageMigration(inter, storage, "test", address)
		require.NoError(t, err)

		migrator := migration.NewValueMigrationsPathMigrator(
			reporter,
			NewEntitlementsMigration(inter),
		)
		migration.Migrate(migrator)

		err = migration.Commit()
		require.NoError(t, err)
	}

	// Assert

	assert.Len(t, reporter.errors, 0)
	assert.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}]struct{}{
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(1),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     common.PathDomainStorage.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("bar"),
			}: {},
		},
		reporter.migrated,
	)

	value := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("bar"))

	require.IsType(t, &interpreter.CompositeValue{}, value)
	tValue := value.(*interpreter.CompositeValue)
	require.Equal(t, "C.T", tValue.QualifiedIdentifier)

	field := tValue.GetMember(inter, interpreter.EmptyLocationRange, "cap")

	require.IsType(t, &interpreter.IDCapabilityValue{}, field)
	cap := field.(*interpreter.IDCapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, cap.BorrowType)
	ref := cap.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID {
				return []common.TypeID{"A.0000000000000001.C.E"}
			},
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)
}

func TestMigrateArrayOfValues(t *testing.T) {
	t.Parallel()

	testAddress1 := common.Address{0, 0, 0, 0, 0, 0, 0, 1}
	testAddress2 := common.Address{0, 0, 0, 0, 0, 0, 0, 2}

	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var signingAddress common.Address

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{signingAddress}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
	}

	// Prepare

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

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract to 0x1

	signingAddress = testAddress1

	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2

	signingAddress = testAddress2

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// update contract on 0x1

	signingAddress = testAddress1

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	// Important: invalidate the loaded program, as it was updated
	runtimeInterface.InvalidateUpdatedPrograms()

	storageIdentifier := common.PathDomainStorage.Identifier()
	storageMap := storage.GetStorageMap(testAddress2, storageIdentifier, false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	// Migrate

	reporter := newTestReporter()

	for _, address := range []common.Address{
		testAddress1,
		testAddress2,
	} {
		migration, err := migrations.NewStorageMigration(inter, storage, "test", address)
		require.NoError(t, err)

		migrator := migration.NewValueMigrationsPathMigrator(
			reporter,
			NewEntitlementsMigration(inter),
		)
		migration.Migrate(migrator)

		err = migration.Commit()
		require.NoError(t, err)
	}

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	assert.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}]struct{}{
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(1),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(2),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     common.PathDomainStorage.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("caps"),
			}: {},
		},
		reporter.migrated,
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
			func() []common.TypeID {
				return []common.TypeID{"A.0000000000000001.C.E"}
			},
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	cap1 := arrValue.Get(inter, interpreter.EmptyLocationRange, 0)
	require.IsType(t, &interpreter.IDCapabilityValue{}, cap1)
	capValue := cap1.(*interpreter.IDCapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref = capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID {
				return []common.TypeID{"A.0000000000000001.C.E"}
			},
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)

	cap2 := arrValue.Get(inter, interpreter.EmptyLocationRange, 1)
	require.IsType(t, &interpreter.IDCapabilityValue{}, cap2)
	capValue = cap1.(*interpreter.IDCapabilityValue)
	require.IsType(t, &interpreter.ReferenceStaticType{}, capValue.BorrowType)
	ref = capValue.BorrowType.(*interpreter.ReferenceStaticType)
	require.Equal(t,
		interpreter.NewEntitlementSetAuthorization(
			inter,
			func() []common.TypeID {
				return []common.TypeID{"A.0000000000000001.C.E"}
			},
			1,
			sema.Conjunction,
		),
		ref.Authorization,
	)
}

func TestMigrateDictOfValues(t *testing.T) {
	t.Parallel()

	testAddress1 := common.Address{0, 0, 0, 0, 0, 0, 0, 1}
	testAddress2 := common.Address{0, 0, 0, 0, 0, 0, 0, 2}

	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var signingAddress common.Address

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{signingAddress}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
	}

	// Prepare

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

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract to 0x1

	signingAddress = testAddress1

	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2

	signingAddress = testAddress2

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// update contract on 0x1

	signingAddress = testAddress1

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	// Important: invalidate the loaded program, as it was updated
	runtimeInterface.InvalidateUpdatedPrograms()

	storageIdentifier := common.PathDomainStorage.Identifier()
	storageMap := storage.GetStorageMap(testAddress2, storageIdentifier, false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	// Migrate

	reporter := newTestReporter()

	for _, address := range []common.Address{
		testAddress1,
		testAddress2,
	} {
		migration, err := migrations.NewStorageMigration(inter, storage, "test", address)
		require.NoError(t, err)

		migrator := migration.NewValueMigrationsPathMigrator(
			reporter,
			NewEntitlementsMigration(inter),
		)

		migration.Migrate(migrator)

		err = migration.Commit()
		require.NoError(t, err)
	}

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	assert.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}]struct{}{
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(1),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(2),
			}: {},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress2,
					Key:     common.PathDomainStorage.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("caps"),
			}: {},
		},
		reporter.migrated,
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

	cap1, present := dictionaryValue.Get(
		inter,
		interpreter.EmptyLocationRange,
		interpreter.NewUnmeteredStringValue("a"),
	)
	require.True(t, present)
	require.IsType(t, &interpreter.IDCapabilityValue{}, cap1)
	capValue := cap1.(*interpreter.IDCapabilityValue)
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

	cap2, present := dictionaryValue.Get(
		inter,
		interpreter.EmptyLocationRange,
		interpreter.NewUnmeteredStringValue("b"),
	)
	require.True(t, present)
	require.IsType(t, &interpreter.IDCapabilityValue{}, cap2)
	capValue = cap1.(*interpreter.IDCapabilityValue)
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

func TestConvertDeprecatedStaticTypes(t *testing.T) {

	t.Parallel()

	test := func(ty interpreter.PrimitiveStaticType) {

		t.Run(ty.String(), func(t *testing.T) {
			t.Parallel()

			inter := NewTestInterpreter(t)
			migration := NewEntitlementsMigration(inter)
			value := interpreter.NewUnmeteredCapabilityValue(
				1,
				interpreter.AddressValue(common.ZeroAddress),
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					ty,
				),
			)

			result, err := migration.ConvertValueToEntitlements(value)
			require.Error(t, err)
			assert.ErrorContains(t, err, "cannot migrate deprecated type")
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

func TestConvertMigratedAccountTypes(t *testing.T) {

	t.Parallel()

	test := func(ty interpreter.PrimitiveStaticType) {

		t.Run(ty.String(), func(t *testing.T) {
			t.Parallel()

			inter := NewTestInterpreter(t)
			migration := NewEntitlementsMigration(inter)
			value := interpreter.NewUnmeteredCapabilityValue(
				1,
				interpreter.AddressValue(common.ZeroAddress),
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					ty,
				),
			)

			newValue, err := statictypes.NewStaticTypeMigration().
				Migrate(
					interpreter.StorageKey{},
					nil,
					value,
					inter,
				)
			require.NoError(t, err)
			require.NotNil(t, newValue)

			result, err := migration.ConvertValueToEntitlements(newValue)
			require.NoError(t, err)
			require.Nilf(t, result, "expected no migration, but got %s", result)
		})
	}

	for ty := interpreter.PrimitiveStaticType(1); ty < interpreter.PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() || !ty.IsDeprecated() { //nolint:staticcheck
			continue
		}

		test(ty)
	}
}

func TestMigrateCapConsAcrossTwoAccounts(t *testing.T) {
	t.Parallel()

	testAddress1 := common.Address{0, 0, 0, 0, 0, 0, 0, 1}
	testAddress2 := common.Address{0, 0, 0, 0, 0, 0, 0, 2}

	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var signingAddress common.Address

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{signingAddress}, nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
	}

	// Prepare

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
           prepare(signer: auth(Inbox, Storage, Capabilities) &Account) {
               signer.capabilities.storage.issue<&C.R>(/storage/r)
           }
       }
   `)

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract to 0x1

	signingAddress = testAddress1

	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("C", oldContract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Execute transaction on 0x2

	signingAddress = testAddress2

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: saveValues,
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Update contract on 0x1

	signingAddress = testAddress1

	err = rt.ExecuteTransaction(
		runtime.Script{
			Source: UpdateTransaction("C", contract),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	// Important: invalidate the loaded program, as it was updated
	runtimeInterface.InvalidateUpdatedPrograms()

	// Migrate

	reporter := newTestReporter()

	for _, address := range []common.Address{
		testAddress1,
		testAddress2,
	} {
		migration, err := migrations.NewStorageMigration(inter, storage, "test", address)
		require.NoError(t, err)

		migrator := migration.NewValueMigrationsPathMigrator(
			reporter,
			NewEntitlementsMigration(inter),
		)

		migration.Migrate(migrator)

		err = migration.Commit()
		require.NoError(t, err)
	}

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	assert.Len(t, reporter.migrated, 1)

	// TODO: assert
}

var _ migrations.Reporter = &testReporter{}

type testReporter struct {
	migrated map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}]struct{}
	errors []error
}

func newTestReporter() *testReporter {
	return &testReporter{
		migrated: map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}]struct{}{},
	}
}

func (t *testReporter) Migrated(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	_ string,
) {
	t.migrated[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}{
		StorageKey:    storageKey,
		StorageMapKey: storageMapKey,
	}] = struct{}{}
}

func (t *testReporter) Error(err error) {
	t.errors = append(t.errors, err)
}

func (t *testReporter) DictionaryKeyConflict(addressPath interpreter.AddressPath) {
	// For testing purposes, record the conflict as an error
	t.errors = append(t.errors, fmt.Errorf("dictionary key conflict: %s", addressPath))
}

func TestRehash(t *testing.T) {

	t.Parallel()

	locationRange := interpreter.EmptyLocationRange

	ledger := NewTestLedger(nil, nil)

	storageMapKey := interpreter.StringStorageMapKey("dict")
	newTestValue := func() interpreter.Value {
		return interpreter.NewUnmeteredStringValue("test")
	}

	const fooBarQualifiedIdentifier = "Foo.Bar"
	testAddress := common.Address{0x42}
	fooAddressLocation := common.NewAddressLocation(nil, testAddress, "Foo")

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
		storage := runtime.NewStorage(ledger, nil)
		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage: storage,
				// NOTE: disabled, because encoded and decoded values are expected to not match
				AtreeValueValidationEnabled:   false,
				AtreeStorageValidationEnabled: true,
			},
		)
		require.NoError(t, err)

		return storage, inter
	}

	newCompositeType := func() *interpreter.CompositeStaticType {
		return interpreter.NewCompositeStaticType(
			nil,
			fooAddressLocation,
			fooBarQualifiedIdentifier,
			common.NewTypeIDFromQualifiedName(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
			),
		)
	}

	entitlementSetAuthorization := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{
			sema.NewEntitlementType(
				nil,
				fooAddressLocation,
				"E",
			),
		},
		sema.Conjunction,
	)

	// Prepare
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		dictionaryStaticType := interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeMetaType,
			interpreter.PrimitiveStaticTypeString,
		)
		dictValue := interpreter.NewDictionaryValue(inter, locationRange, dictionaryStaticType)

		refType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.UnauthorizedAccess,
			newCompositeType(),
		)
		refType.HasLegacyIsAuthorized = true
		refType.LegacyIsAuthorized = true

		legacyRefType := &migrations.LegacyReferenceType{
			ReferenceStaticType: refType,
		}

		optType := interpreter.NewOptionalStaticType(
			nil,
			legacyRefType,
		)

		legacyOptType := &migrations.LegacyOptionalType{
			OptionalStaticType: optType,
		}

		typeValue := interpreter.NewUnmeteredTypeValue(legacyOptType)

		dictValue.Insert(
			inter,
			locationRange,
			typeValue,
			newTestValue(),
		)

		// Note: ID is in the old format
		assert.Equal(t,
			common.TypeID("auth&A.4200000000000000.Foo.Bar?"),
			legacyOptType.ID(),
		)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			true,
		)

		storageMap.SetValue(inter,
			storageMapKey,
			dictValue.Transfer(
				inter,
				locationRange,
				atree.Address(testAddress),
				false,
				nil,
				nil,
				true, // dictValue is standalone
			),
		)

		err := storage.Commit(inter, false)
		require.NoError(t, err)

		err = storage.CheckHealth()
		require.NoError(t, err)
	})()

	// Migrate
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		inter.SharedState.Config.CompositeTypeHandler = func(
			location common.Location,
			typeID interpreter.TypeID,
		) *sema.CompositeType {

			compositeType := &sema.CompositeType{
				Location:   fooAddressLocation,
				Identifier: fooBarQualifiedIdentifier,
				Kind:       common.CompositeKindStructure,
			}

			compositeType.Members = sema.MembersAsMap([]*sema.Member{
				sema.NewUnmeteredFunctionMember(
					compositeType,
					entitlementSetAuthorization,
					"sayHello",
					&sema.FunctionType{},
					"",
				),
			})

			return compositeType
		}

		migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
			migration.NewValueMigrationsPathMigrator(
				reporter,
				NewEntitlementsMigration(inter),
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		err = storage.CheckHealth()
		require.NoError(t, err)

		assert.Empty(t, reporter.errors)

		require.Equal(t,
			map[struct {
				interpreter.StorageKey
				interpreter.StorageMapKey
			}]struct{}{
				{
					StorageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainStorage.Identifier(),
					},
					StorageMapKey: storageMapKey,
				}: {},
			},
			reporter.migrated,
		)
	})()

	// Load
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		err := storage.CheckHealth()
		require.NoError(t, err)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			false,
		)
		storedValue := storageMap.ReadValue(inter, storageMapKey)

		require.IsType(t, &interpreter.DictionaryValue{}, storedValue)

		dictValue := storedValue.(*interpreter.DictionaryValue)

		refType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.ConvertSemaAccessToStaticAuthorization(nil, entitlementSetAuthorization),
			newCompositeType(),
		)

		optType := interpreter.NewOptionalStaticType(
			nil,
			refType,
		)

		typeValue := interpreter.NewUnmeteredTypeValue(optType)

		// Note: ID is in the new format
		assert.Equal(t,
			common.TypeID("(auth(A.4200000000000000.E)&A.4200000000000000.Foo.Bar)?"),
			optType.ID(),
		)

		assert.Equal(t, 1, dictValue.Count())

		value, ok := dictValue.Get(inter, locationRange, typeValue)
		require.True(t, ok)

		require.IsType(t, &interpreter.StringValue{}, value)
		require.Equal(t,
			newTestValue(),
			value.(*interpreter.StringValue),
		)
	})()
}

func TestIntersectionTypeWithIntersectionLegacyType(t *testing.T) {

	t.Parallel()

	testAddress := common.Address{0x42}

	const interface1QualifiedIdentifier = "SI1"
	interfaceType1 := interpreter.NewInterfaceStaticType(
		nil,
		utils.TestLocation,
		interface1QualifiedIdentifier,
		utils.TestLocation.TypeID(nil, interface1QualifiedIdentifier),
	)

	const interface2QualifiedIdentifier = "SI2"
	interfaceType2 := interpreter.NewInterfaceStaticType(
		nil,
		utils.TestLocation,
		interface2QualifiedIdentifier,
		utils.TestLocation.TypeID(nil, interface2QualifiedIdentifier),
	)

	ledger := NewTestLedger(nil, nil)

	storageMapKey := interpreter.StringStorageMapKey("dict")

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
		storage := runtime.NewStorage(ledger, nil)
		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage: storage,
				// NOTE: disabled, because encoded and decoded values are expected to not match
				AtreeValueValidationEnabled:   false,
				AtreeStorageValidationEnabled: true,
			},
		)
		require.NoError(t, err)

		return storage, inter
	}

	// Prepare
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		expectedIntersection := interpreter.NewIntersectionStaticType(
			nil,
			[]*interpreter.InterfaceStaticType{
				interfaceType1,
			},
		)
		// NOTE: setting the legacy type to an intersection type
		expectedIntersection.LegacyType = interpreter.NewIntersectionStaticType(
			nil,
			[]*interpreter.InterfaceStaticType{
				interfaceType2,
			},
		)

		storedValue := interpreter.NewTypeValue(
			nil,
			interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				expectedIntersection,
			),
		)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			true,
		)

		storageMap.SetValue(inter,
			storageMapKey,
			storedValue,
		)

		err := storage.Commit(inter, false)
		require.NoError(t, err)

		err = storage.CheckHealth()
		require.NoError(t, err)
	})()

	// Migrate
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		inter.SharedState.Config.InterfaceTypeHandler = func(
			location common.Location,
			typeID interpreter.TypeID,
		) *sema.InterfaceType {

			_, qualifiedIdentifier, err := common.DecodeTypeID(nil, string(typeID))
			require.NoError(t, err)

			return &sema.InterfaceType{
				Location:      TestLocation,
				Identifier:    qualifiedIdentifier,
				CompositeKind: common.CompositeKindStructure,
				Members:       &sema.StringMemberOrderedMap{},
			}
		}

		migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
			migration.NewValueMigrationsPathMigrator(
				reporter,
				NewEntitlementsMigration(inter),
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		err = storage.CheckHealth()
		require.NoError(t, err)

		assert.Empty(t, reporter.errors)

		require.Equal(t,
			map[struct {
				interpreter.StorageKey
				interpreter.StorageMapKey
			}]struct{}{
				{
					StorageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainStorage.Identifier(),
					},
					StorageMapKey: storageMapKey,
				}: {},
			},
			reporter.migrated,
		)
	})()

	// Load
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		err := storage.CheckHealth()
		require.NoError(t, err)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			false,
		)

		storedValue := storageMap.ReadValue(inter, storageMapKey)

		require.IsType(t, interpreter.TypeValue{}, storedValue)

		typeValue := storedValue.(interpreter.TypeValue)

		expectedType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.UnauthorizedAccess,
			interpreter.NewIntersectionStaticType(
				nil,
				[]*interpreter.InterfaceStaticType{
					// NOTE: this is the legacy type
					interfaceType2,
				},
			),
		)

		require.Equal(t, expectedType, typeValue.Type)
	})()
}

func TestUseAfterMigrationFailure(t *testing.T) {

	t.Parallel()

	locationRange := interpreter.EmptyLocationRange

	ledger := NewTestLedger(nil, nil)

	storageMapKey := interpreter.StringStorageMapKey("dict")
	newTestValue := func() interpreter.Value {
		return interpreter.NewUnmeteredStringValue("test")
	}

	const fooBarQualifiedIdentifier = "Foo.Bar"
	testAddress := common.Address{0x42}
	fooAddressLocation := common.NewAddressLocation(nil, testAddress, "Foo")

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
		storage := runtime.NewStorage(ledger, nil)
		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage: storage,
				// NOTE: disabled, because encoded and decoded values are expected to not match
				AtreeValueValidationEnabled:   false,
				AtreeStorageValidationEnabled: true,
			},
		)
		require.NoError(t, err)

		return storage, inter
	}

	newCompositeType := func() *interpreter.CompositeStaticType {
		return interpreter.NewCompositeStaticType(
			nil,
			fooAddressLocation,
			fooBarQualifiedIdentifier,
			common.NewTypeIDFromQualifiedName(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
			),
		)
	}

	// Prepare
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		dictionaryStaticType := interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeMetaType,
			interpreter.PrimitiveStaticTypeString,
		)
		dictValue := interpreter.NewDictionaryValue(inter, locationRange, dictionaryStaticType)

		refType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.UnauthorizedAccess,
			newCompositeType(),
		)
		refType.HasLegacyIsAuthorized = true
		refType.LegacyIsAuthorized = true

		legacyRefType := &migrations.LegacyReferenceType{
			ReferenceStaticType: refType,
		}

		optType := interpreter.NewOptionalStaticType(
			nil,
			legacyRefType,
		)

		legacyOptType := &migrations.LegacyOptionalType{
			OptionalStaticType: optType,
		}

		typeValue := interpreter.NewUnmeteredTypeValue(legacyOptType)

		dictValue.Insert(
			inter,
			locationRange,
			typeValue,
			newTestValue(),
		)

		// Note: ID is in the old format
		assert.Equal(t,
			common.TypeID("auth&A.4200000000000000.Foo.Bar?"),
			legacyOptType.ID(),
		)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			true,
		)

		storageMap.SetValue(inter,
			storageMapKey,
			dictValue.Transfer(
				inter,
				locationRange,
				atree.Address(testAddress),
				false,
				nil,
				nil,
				true, // dictValue is standalone
			),
		)

		err := storage.Commit(inter, false)
		require.NoError(t, err)

		err = storage.CheckHealth()
		require.NoError(t, err)
	})()

	// Migrate
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		const importErrorMessage = "cannot import"

		inter.SharedState.Config.ImportLocationHandler =
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				panic(importErrorMessage)
			}

		migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
			migration.NewValueMigrationsPathMigrator(
				reporter,
				NewEntitlementsMigration(inter),
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		err = storage.CheckHealth()
		require.NoError(t, err)

		require.Len(t, reporter.errors, 1)

		assert.ErrorContains(t, reporter.errors[0], importErrorMessage)

		require.Empty(t, reporter.migrated)
	})()

	// Load
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		err := storage.CheckHealth()
		require.NoError(t, err)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			false,
		)
		storedValue := storageMap.ReadValue(inter, storageMapKey)

		require.IsType(t, &interpreter.DictionaryValue{}, storedValue)

		dictValue := storedValue.(*interpreter.DictionaryValue)

		refType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.UnauthorizedAccess,
			newCompositeType(),
		)
		refType.HasLegacyIsAuthorized = true
		refType.LegacyIsAuthorized = true

		optType := interpreter.NewOptionalStaticType(
			nil,
			refType,
		)

		typeValue := interpreter.NewUnmeteredTypeValue(optType)

		// Note: ID is in the new format
		assert.Equal(t,
			common.TypeID("(&A.4200000000000000.Foo.Bar)?"),
			optType.ID(),
		)

		assert.Equal(t, 1, dictValue.Count())

		// Key did not get migrated, so is inaccessible using the "new" type value
		_, ok := dictValue.Get(inter, locationRange, typeValue)
		require.False(t, ok)

		// But the key is still accessible using the "old" type value
		legacyKey := migrations.LegacyKey(typeValue)

		value, ok := dictValue.Get(inter, locationRange, legacyKey)
		require.True(t, ok)
		require.Equal(t, newTestValue(), value)
	})()
}
