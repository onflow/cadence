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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCapabilityStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal, borrow type", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeString,
			}.Equal(
				CapabilityStaticType{
					BorrowType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("equal, no borrow type", func(t *testing.T) {

		t.Parallel()

		a := CapabilityStaticType{}
		b := CapabilityStaticType{}
		require.True(t, a.Equal(b))
	})

	t.Run("unequal, self no borrow type", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CapabilityStaticType{}.Equal(
				CapabilityStaticType{
					BorrowType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("unequal, other no borrow type", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeString,
			}.Equal(
				CapabilityStaticType{},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeString,
			}.Equal(
				ReferenceStaticType{
					ReferencedType: PrimitiveStaticTypeString,
				},
			),
		)
	})
}

func TestReferenceStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			ReferenceStaticType{
				Authorization:  UnauthorizedAccess,
				ReferencedType: PrimitiveStaticTypeString,
			}.Equal(
				ReferenceStaticType{
					Authorization:  UnauthorizedAccess,
					ReferencedType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ReferenceStaticType{
				Authorization:  UnauthorizedAccess,
				ReferencedType: PrimitiveStaticTypeInt,
			}.Equal(
				ReferenceStaticType{
					Authorization:  UnauthorizedAccess,
					ReferencedType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different auth", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ReferenceStaticType{
				Authorization:  UnauthorizedAccess,
				ReferencedType: PrimitiveStaticTypeInt,
			}.Equal(
				ReferenceStaticType{
					Authorization:  EntitlementMapAuthorization{TypeID: "Foo"},
					ReferencedType: PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ReferenceStaticType{
				ReferencedType: PrimitiveStaticTypeString,
			}.Equal(
				CapabilityStaticType{
					BorrowType: PrimitiveStaticTypeString,
				},
			),
		)
	})
}

func TestCompositeStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			NewCompositeStaticTypeComputeTypeID(
				nil,
				utils.TestLocation,
				"X",
			).Equal(
				NewCompositeStaticTypeComputeTypeID(
					nil,
					utils.TestLocation,
					"X",
				),
			),
		)
	})

	t.Run("different qualified identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewCompositeStaticTypeComputeTypeID(
				nil,
				utils.TestLocation,
				"X",
			).Equal(
				NewCompositeStaticTypeComputeTypeID(
					nil,
					utils.TestLocation,
					"Y",
				),
			),
		)
	})

	t.Run("different locations of same kind, same qualified identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewCompositeStaticTypeComputeTypeID(
				nil,
				common.IdentifierLocation("A"),
				"X",
			).Equal(
				NewCompositeStaticTypeComputeTypeID(
					nil,
					common.IdentifierLocation("B"),
					"X",
				),
			),
		)
	})

	t.Run("different locations of different kinds, same qualified identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewCompositeStaticTypeComputeTypeID(
				nil,
				common.IdentifierLocation("A"),
				"X",
			).Equal(
				NewCompositeStaticTypeComputeTypeID(
					nil,
					common.StringLocation("A"),
					"X",
				),
			),
		)
	})

	t.Run("no locations, same qualified identifier", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			NewCompositeStaticTypeComputeTypeID(
				nil,
				nil,
				"X",
			).Equal(
				NewCompositeStaticTypeComputeTypeID(
					nil,
					nil,
					"X",
				),
			),
		)
	})

	t.Run("no locations, different qualified identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewCompositeStaticTypeComputeTypeID(
				nil,
				nil,
				"X",
			).Equal(
				NewCompositeStaticTypeComputeTypeID(
					nil,
					nil,
					"Y",
				),
			),
		)
	})

	t.Run("one location, same qualified identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewCompositeStaticTypeComputeTypeID(
				nil,
				nil,
				"X",
			).Equal(
				NewCompositeStaticTypeComputeTypeID(
					nil,
					common.StringLocation("B"),
					"X",
				),
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewCompositeStaticTypeComputeTypeID(
				nil,
				nil,
				"X",
			).Equal(
				InterfaceStaticType{
					Location:            nil,
					QualifiedIdentifier: "X",
				},
			),
		)
	})
}

func TestInterfaceStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			InterfaceStaticType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "X",
			}.Equal(
				InterfaceStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("different name", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			InterfaceStaticType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "X",
			}.Equal(
				InterfaceStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "Y",
				},
			),
		)
	})

	t.Run("different locations, different identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			InterfaceStaticType{
				Location:            common.IdentifierLocation("A"),
				QualifiedIdentifier: "X",
			}.Equal(
				InterfaceStaticType{
					Location:            common.IdentifierLocation("B"),
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("different locations, different identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			InterfaceStaticType{
				Location:            common.IdentifierLocation("A"),
				QualifiedIdentifier: "X",
			}.Equal(
				InterfaceStaticType{
					Location:            common.StringLocation("A"),
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("no location", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			InterfaceStaticType{
				Location:            nil,
				QualifiedIdentifier: "X",
			}.Equal(
				InterfaceStaticType{
					Location:            nil,
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("no location, different identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			InterfaceStaticType{
				Location:            nil,
				QualifiedIdentifier: "X",
			}.Equal(
				InterfaceStaticType{
					Location:            nil,
					QualifiedIdentifier: "Y",
				},
			),
		)
	})

	t.Run("one location, same identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			InterfaceStaticType{
				Location:            nil,
				QualifiedIdentifier: "X",
			}.Equal(
				InterfaceStaticType{
					Location:            common.StringLocation("B"),
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			InterfaceStaticType{
				Location:            nil,
				QualifiedIdentifier: "X",
			}.Equal(
				NewCompositeStaticTypeComputeTypeID(
					nil,
					nil,
					"X",
				),
			),
		)
	})
}

func TestConstantSizedStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			ConstantSizedStaticType{
				Type: PrimitiveStaticTypeString,
				Size: 10,
			}.Equal(
				ConstantSizedStaticType{
					Type: PrimitiveStaticTypeString,
					Size: 10,
				},
			),
		)
	})

	t.Run("different sizes", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ConstantSizedStaticType{
				Type: PrimitiveStaticTypeString,
				Size: 20,
			}.Equal(
				ConstantSizedStaticType{
					Type: PrimitiveStaticTypeString,
					Size: 10,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ConstantSizedStaticType{
				Type: PrimitiveStaticTypeInt,
				Size: 10,
			}.Equal(
				ConstantSizedStaticType{
					Type: PrimitiveStaticTypeString,
					Size: 10,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ConstantSizedStaticType{
				Type: PrimitiveStaticTypeInt,
				Size: 10,
			}.Equal(
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInt,
				},
			),
		)
	})
}

func TestVariableSizedStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			VariableSizedStaticType{
				Type: PrimitiveStaticTypeString,
			}.Equal(
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			}.Equal(
				ConstantSizedStaticType{
					Type: PrimitiveStaticTypeInt,
					Size: 10,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			}.Equal(
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})
}

func TestPrimitiveStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		a := PrimitiveStaticTypeString
		b := PrimitiveStaticTypeString
		require.True(t, a.Equal(b))
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			PrimitiveStaticTypeInt.
				Equal(PrimitiveStaticTypeString),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			PrimitiveStaticTypeInt.
				Equal(CapabilityStaticType{}),
		)
	})
}

func TestOptionalStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			OptionalStaticType{
				Type: PrimitiveStaticTypeString,
			}.Equal(
				OptionalStaticType{
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			OptionalStaticType{
				Type: PrimitiveStaticTypeInt,
			}.Equal(
				OptionalStaticType{
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			OptionalStaticType{
				Type: PrimitiveStaticTypeInt,
			}.Equal(
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInt,
				},
			),
		)
	})
}

func TestDictionaryStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeString,
			}.Equal(
				DictionaryStaticType{
					KeyType:   PrimitiveStaticTypeInt,
					ValueType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different key types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeString,
			}.Equal(
				DictionaryStaticType{
					KeyType:   PrimitiveStaticTypeVoid,
					ValueType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different value types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeVoid,
			}.Equal(
				DictionaryStaticType{
					KeyType:   PrimitiveStaticTypeInt,
					ValueType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeVoid,
			}.Equal(
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeInt,
				},
			),
		)
	})
}

func TestIntersectionStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&IntersectionStaticType{
				Type: PrimitiveStaticTypeInt,
				Types: []InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "X",
					},
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "Y",
					},
				},
			}).Equal(
				&IntersectionStaticType{
					Type: PrimitiveStaticTypeInt,
					Types: []InterfaceStaticType{
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "Y",
						},
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "X",
						},
					},
				},
			),
		)
	})

	t.Run("equal, no intersections", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&IntersectionStaticType{
				Type:  PrimitiveStaticTypeInt,
				Types: []InterfaceStaticType{},
			}).Equal(
				&IntersectionStaticType{
					Type:  PrimitiveStaticTypeInt,
					Types: []InterfaceStaticType{},
				},
			),
		)
	})

	t.Run("different intersection type", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Type: PrimitiveStaticTypeString,
				Types: []InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "X",
					},
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "Y",
					},
				},
			}).Equal(
				&IntersectionStaticType{
					Type: PrimitiveStaticTypeInt,
					Types: []InterfaceStaticType{
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "Y",
						},
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "X",
						},
					},
				},
			),
		)
	})

	t.Run("fewer intersections", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Type: PrimitiveStaticTypeInt,
				Types: []InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "X",
					},
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "Y",
					},
				},
			}).Equal(
				&IntersectionStaticType{
					Type: PrimitiveStaticTypeInt,
					Types: []InterfaceStaticType{
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "Y",
						},
					},
				},
			),
		)
	})

	t.Run("more intersections", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Type: PrimitiveStaticTypeInt,
				Types: []InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "X",
					},
				},
			}).Equal(
				&IntersectionStaticType{
					Type: PrimitiveStaticTypeInt,
					Types: []InterfaceStaticType{
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "Y",
						},
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "X",
						},
					},
				},
			),
		)
	})

	t.Run("different intersections", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Type: PrimitiveStaticTypeInt,
				Types: []InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "X",
					},
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "Y",
					},
				},
			}).Equal(
				&IntersectionStaticType{
					Type: PrimitiveStaticTypeInt,
					Types: []InterfaceStaticType{
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "X",
						},
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "Z",
						},
					},
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Type: PrimitiveStaticTypeInt,
				Types: []InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "X",
					},
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "Y",
					},
				},
			}).Equal(
				ReferenceStaticType{
					ReferencedType: PrimitiveStaticTypeInt,
				},
			),
		)
	})
}

func TestPrimitiveStaticTypeCount(t *testing.T) {
	t.Parallel()

	// This asserts that the total number of types in the PrimitiveStaticType enum has not changed,
	// in order to prevent adding new types into the enum in the middle.
	// However, it is possible to safely change the size of this enum by only appending new types the end,
	// (before the PrimitiveStaticType_Count of course).
	// Only update this test if you are certain your change to this enum was to append new types to the end.
	t.Run("No new types added in between", func(t *testing.T) {
		require.Equal(t, byte(105), byte(PrimitiveStaticType_Count))
	})
}

func TestStaticTypeConversion(t *testing.T) {

	t.Parallel()

	const testLocation = common.StringLocation("test")

	const testInterfaceQualifiedIdentifier = "TestInterface"

	testInterfaceSemaType := &sema.InterfaceType{
		Location:   testLocation,
		Identifier: testInterfaceQualifiedIdentifier,
	}

	testInterfaceStaticType := InterfaceStaticType{
		Location:            testLocation,
		QualifiedIdentifier: testInterfaceQualifiedIdentifier,
	}

	const testCompositeQualifiedIdentifier = "TestComposite"

	testCompositeSemaType := &sema.CompositeType{
		Location:   testLocation,
		Identifier: testCompositeQualifiedIdentifier,
	}

	testCompositeStaticType := CompositeStaticType{
		Location:            testLocation,
		QualifiedIdentifier: testCompositeQualifiedIdentifier,
		TypeID:              "S.test.TestComposite",
	}

	testFunctionType := &sema.FunctionType{}

	tests := []struct {
		name         string
		semaType     sema.Type
		staticType   StaticType
		getInterface func(
			location common.Location,
			qualifiedIdentifier string,
		) (
			*sema.InterfaceType,
			error,
		)
		getComposite func(
			location common.Location,
			qualifiedIdentifier string,
			typeID common.TypeID,
		) (
			*sema.CompositeType,
			error,
		)
	}{
		{
			name:       "Void",
			semaType:   sema.VoidType,
			staticType: PrimitiveStaticTypeVoid,
		},
		{
			name:       "Any",
			semaType:   sema.AnyType,
			staticType: PrimitiveStaticTypeAny,
		},
		{
			name:       "Never",
			semaType:   sema.NeverType,
			staticType: PrimitiveStaticTypeNever,
		},
		{
			name:       "AnyStruct",
			semaType:   sema.AnyStructType,
			staticType: PrimitiveStaticTypeAnyStruct,
		},
		{
			name:       "AnyResource",
			semaType:   sema.AnyResourceType,
			staticType: PrimitiveStaticTypeAnyResource,
		},
		{
			name:       "Bool",
			semaType:   sema.BoolType,
			staticType: PrimitiveStaticTypeBool,
		},
		{
			name:       "Address",
			semaType:   sema.TheAddressType,
			staticType: PrimitiveStaticTypeAddress,
		},
		{
			name:       "String",
			semaType:   sema.StringType,
			staticType: PrimitiveStaticTypeString,
		},
		{
			name:       "Character",
			semaType:   sema.CharacterType,
			staticType: PrimitiveStaticTypeCharacter,
		},
		{
			name:       "MetaType",
			semaType:   sema.MetaType,
			staticType: PrimitiveStaticTypeMetaType,
		},
		{
			name:       "Block",
			semaType:   sema.BlockType,
			staticType: PrimitiveStaticTypeBlock,
		},

		{
			name:       "Number",
			semaType:   sema.NumberType,
			staticType: PrimitiveStaticTypeNumber,
		},
		{
			name:       "SignedNumber",
			semaType:   sema.SignedNumberType,
			staticType: PrimitiveStaticTypeSignedNumber,
		},

		{
			name:       "Integer",
			semaType:   sema.IntegerType,
			staticType: PrimitiveStaticTypeInteger,
		},
		{
			name:       "SignedInteger",
			semaType:   sema.SignedIntegerType,
			staticType: PrimitiveStaticTypeSignedInteger,
		},

		{
			name:       "FixedPoint",
			semaType:   sema.FixedPointType,
			staticType: PrimitiveStaticTypeFixedPoint,
		},
		{
			name:       "SignedFixedPoint",
			semaType:   sema.SignedFixedPointType,
			staticType: PrimitiveStaticTypeSignedFixedPoint,
		},

		{
			name:       "Int",
			semaType:   sema.IntType,
			staticType: PrimitiveStaticTypeInt,
		},
		{
			name:       "Int8",
			semaType:   sema.Int8Type,
			staticType: PrimitiveStaticTypeInt8,
		},
		{
			name:       "Int16",
			semaType:   sema.Int16Type,
			staticType: PrimitiveStaticTypeInt16,
		},
		{
			name:       "Int32",
			semaType:   sema.Int32Type,
			staticType: PrimitiveStaticTypeInt32,
		},
		{
			name:       "Int64",
			semaType:   sema.Int64Type,
			staticType: PrimitiveStaticTypeInt64,
		},
		{
			name:       "Int128",
			semaType:   sema.Int128Type,
			staticType: PrimitiveStaticTypeInt128,
		},
		{
			name:       "Int256",
			semaType:   sema.Int256Type,
			staticType: PrimitiveStaticTypeInt256,
		},

		{
			name:       "UInt",
			semaType:   sema.UIntType,
			staticType: PrimitiveStaticTypeUInt,
		},
		{
			name:       "UInt8",
			semaType:   sema.UInt8Type,
			staticType: PrimitiveStaticTypeUInt8,
		},
		{
			name:       "UInt16",
			semaType:   sema.UInt16Type,
			staticType: PrimitiveStaticTypeUInt16,
		},
		{
			name:       "UInt32",
			semaType:   sema.UInt32Type,
			staticType: PrimitiveStaticTypeUInt32,
		},
		{
			name:       "UInt64",
			semaType:   sema.UInt64Type,
			staticType: PrimitiveStaticTypeUInt64,
		},
		{
			name:       "UInt128",
			semaType:   sema.UInt128Type,
			staticType: PrimitiveStaticTypeUInt128,
		},
		{
			name:       "UInt256",
			semaType:   sema.UInt256Type,
			staticType: PrimitiveStaticTypeUInt256,
		},

		{
			name:       "Word8",
			semaType:   sema.Word8Type,
			staticType: PrimitiveStaticTypeWord8,
		},
		{
			name:       "Word16",
			semaType:   sema.Word16Type,
			staticType: PrimitiveStaticTypeWord16,
		},
		{
			name:       "Word32",
			semaType:   sema.Word32Type,
			staticType: PrimitiveStaticTypeWord32,
		},
		{
			name:       "Word64",
			semaType:   sema.Word64Type,
			staticType: PrimitiveStaticTypeWord64,
		},
		{
			name:       "Word128",
			semaType:   sema.Word128Type,
			staticType: PrimitiveStaticTypeWord128,
		},

		{
			name:       "Fix64",
			semaType:   sema.Fix64Type,
			staticType: PrimitiveStaticTypeFix64,
		},

		{
			name:       "UFix64",
			semaType:   sema.UFix64Type,
			staticType: PrimitiveStaticTypeUFix64,
		},

		{
			name:       "Path",
			semaType:   sema.PathType,
			staticType: PrimitiveStaticTypePath,
		},
		{
			name:       "Capability",
			semaType:   &sema.CapabilityType{},
			staticType: PrimitiveStaticTypeCapability,
		},
		{
			name:       "StoragePath",
			semaType:   sema.StoragePathType,
			staticType: PrimitiveStaticTypeStoragePath,
		},
		{
			name:       "CapabilityPath",
			semaType:   sema.CapabilityPathType,
			staticType: PrimitiveStaticTypeCapabilityPath,
		},
		{
			name:       "PublicPath",
			semaType:   sema.PublicPathType,
			staticType: PrimitiveStaticTypePublicPath,
		},
		{
			name:       "PrivatePath",
			semaType:   sema.PrivatePathType,
			staticType: PrimitiveStaticTypePrivatePath,
		},

		{
			name:       "AuthAccount",
			semaType:   sema.AuthAccountType,
			staticType: PrimitiveStaticTypeAuthAccount,
		},
		{
			name:       "PublicAccount",
			semaType:   sema.PublicAccountType,
			staticType: PrimitiveStaticTypePublicAccount,
		},

		{
			name:       "DeployedContract",
			semaType:   sema.DeployedContractType,
			staticType: PrimitiveStaticTypeDeployedContract,
		},
		{
			name:       "AuthAccount.Contracts",
			semaType:   sema.AuthAccountContractsType,
			staticType: PrimitiveStaticTypeAuthAccountContracts,
		},
		{
			name:       "PublicAccount.Contracts",
			semaType:   sema.PublicAccountContractsType,
			staticType: PrimitiveStaticTypePublicAccountContracts,
		},
		{
			name:       "AuthAccount.Keys",
			semaType:   sema.AuthAccountKeysType,
			staticType: PrimitiveStaticTypeAuthAccountKeys,
		},
		{
			name:       "PublicAccount.Keys",
			semaType:   sema.PublicAccountKeysType,
			staticType: PrimitiveStaticTypePublicAccountKeys,
		},
		{
			name:       "AccountKey",
			semaType:   sema.AccountKeyType,
			staticType: PrimitiveStaticTypeAccountKey,
		},
		{
			name:       "AuthAccount.Inbox",
			semaType:   sema.AuthAccountInboxType,
			staticType: PrimitiveStaticTypeAuthAccountInbox,
		},

		{
			name:       "Unparameterized Capability",
			semaType:   &sema.CapabilityType{},
			staticType: PrimitiveStaticTypeCapability,
		},
		{
			name: "Parameterized  Capability",
			semaType: &sema.CapabilityType{
				BorrowType: sema.IntType,
			},
			staticType: CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeInt,
			},
		},

		{
			name: "Variable-sized array",
			semaType: &sema.VariableSizedType{
				Type: sema.IntType,
			},
			staticType: VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			},
		},
		{
			name: "Constant-sized array",
			semaType: &sema.ConstantSizedType{
				Type: sema.IntType,
				Size: 42,
			},
			staticType: ConstantSizedStaticType{
				Type: PrimitiveStaticTypeInt,
				Size: 42,
			},
		},
		{
			name: "Optional",
			semaType: &sema.OptionalType{
				Type: sema.IntType,
			},
			staticType: OptionalStaticType{
				Type: PrimitiveStaticTypeInt,
			},
		},
		{
			name: "Reference",
			semaType: &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: sema.UnauthorizedAccess,
			},
			staticType: ReferenceStaticType{
				ReferencedType: PrimitiveStaticTypeInt,
				Authorization:  UnauthorizedAccess,
			},
		},
		{
			name: "Dictionary",
			semaType: &sema.DictionaryType{
				KeyType:   sema.IntType,
				ValueType: sema.StringType,
			},
			staticType: DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeString,
			},
		},
		{
			name: "Intersection",
			semaType: &sema.IntersectionType{
				Type: sema.IntType,
				Types: []*sema.InterfaceType{
					testInterfaceSemaType,
				},
			},
			staticType: &IntersectionStaticType{
				Type: PrimitiveStaticTypeInt,
				Types: []InterfaceStaticType{
					testInterfaceStaticType,
				},
			},
			getInterface: func(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error) {
				require.Equal(t, testLocation, location)
				require.Equal(t, testInterfaceQualifiedIdentifier, qualifiedIdentifier)
				return testInterfaceSemaType, nil
			},
		},
		{
			name:       "Interface",
			semaType:   testInterfaceSemaType,
			staticType: testInterfaceStaticType,
			getInterface: func(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error) {
				require.Equal(t, testLocation, location)
				require.Equal(t, testInterfaceQualifiedIdentifier, qualifiedIdentifier)
				return testInterfaceSemaType, nil
			},
		},
		{
			name:       "Composite",
			semaType:   testCompositeSemaType,
			staticType: testCompositeStaticType,
			getComposite: func(location common.Location, qualifiedIdentifier string, typeID common.TypeID) (*sema.CompositeType, error) {
				require.Equal(t, testLocation, location)
				require.Equal(t, testCompositeQualifiedIdentifier, qualifiedIdentifier)
				return testCompositeSemaType, nil
			},
		},
		{
			name:     "Function",
			semaType: testFunctionType,
			staticType: FunctionStaticType{
				Type: testFunctionType,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Test sema to static

			convertedStaticType := ConvertSemaToStaticType(nil, test.semaType)
			require.Equal(t,
				test.staticType,
				convertedStaticType,
			)

			// Test static to sema

			getInterface := test.getInterface
			if getInterface == nil {
				getInterface = func(_ common.Location, _ string) (*sema.InterfaceType, error) {
					require.FailNow(t, "getInterface should not be called")
					return nil, nil
				}
			}

			getComposite := test.getComposite
			if getComposite == nil {
				getComposite = func(_ common.Location, _ string, _ common.TypeID) (*sema.CompositeType, error) {
					require.FailNow(t, "getComposite should not be called")
					return nil, nil
				}
			}

			getEntitlement := func(_ common.TypeID) (*sema.EntitlementType, error) {
				require.FailNow(t, "getComposite should not be called")
				return nil, nil
			}

			getEntitlementMap := func(_ common.TypeID) (*sema.EntitlementMapType, error) {
				require.FailNow(t, "getComposite should not be called")
				return nil, nil
			}

			convertedSemaType, err := ConvertStaticToSemaType(
				nil,
				test.staticType,
				getInterface,
				getComposite,
				getEntitlement,
				getEntitlementMap,
			)
			require.NoError(t, err)
			require.Equal(t,
				test.semaType,
				convertedSemaType,
			)
		})
	}
}
