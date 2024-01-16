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

	"github.com/stretchr/testify/assert"
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
			(&CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeString,
			}).Equal(
				&CapabilityStaticType{
					BorrowType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("equal, no borrow type", func(t *testing.T) {

		t.Parallel()

		a := &CapabilityStaticType{}
		b := &CapabilityStaticType{}
		require.True(t, a.Equal(b))
	})

	t.Run("unequal, self no borrow type", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&CapabilityStaticType{}).Equal(
				&CapabilityStaticType{
					BorrowType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("unequal, other no borrow type", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeString,
			}).Equal(
				&CapabilityStaticType{},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeString,
			}).Equal(
				&ReferenceStaticType{
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
			(&ReferenceStaticType{
				Authorization:  UnauthorizedAccess,
				ReferencedType: PrimitiveStaticTypeString,
			}).Equal(
				&ReferenceStaticType{
					Authorization:  UnauthorizedAccess,
					ReferencedType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&ReferenceStaticType{
				Authorization:  UnauthorizedAccess,
				ReferencedType: PrimitiveStaticTypeInt,
			}).Equal(
				&ReferenceStaticType{
					Authorization:  UnauthorizedAccess,
					ReferencedType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different auth", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&ReferenceStaticType{
				Authorization:  UnauthorizedAccess,
				ReferencedType: PrimitiveStaticTypeInt,
			}).Equal(
				&ReferenceStaticType{
					Authorization:  EntitlementMapAuthorization{TypeID: "Foo"},
					ReferencedType: PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&ReferenceStaticType{
				ReferencedType: PrimitiveStaticTypeString,
			}).Equal(
				(&CapabilityStaticType{
					BorrowType: PrimitiveStaticTypeString,
				}),
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
				NewInterfaceStaticTypeComputeTypeID(nil, nil, "X"),
			),
		)
	})
}

func TestInterfaceStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X").
				Equal(NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X")),
		)
	})

	t.Run("different name", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X").
				Equal(NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y")),
		)
	})

	t.Run("different locations of same kind, same qualified identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewInterfaceStaticTypeComputeTypeID(nil, common.IdentifierLocation("A"), "X").
				Equal(NewInterfaceStaticTypeComputeTypeID(nil, common.IdentifierLocation("B"), "X")),
		)
	})

	t.Run("different locations of different kinds, same qualified identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewInterfaceStaticTypeComputeTypeID(nil, common.IdentifierLocation("A"), "X").
				Equal(NewInterfaceStaticTypeComputeTypeID(nil, common.StringLocation("A"), "X")),
		)
	})

	t.Run("no location", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			NewInterfaceStaticTypeComputeTypeID(nil, nil, "X").
				Equal(NewInterfaceStaticTypeComputeTypeID(nil, nil, "X")),
		)
	})

	t.Run("no location, different identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewInterfaceStaticTypeComputeTypeID(nil, nil, "X").
				Equal(NewInterfaceStaticTypeComputeTypeID(nil, nil, "Y")),
		)
	})

	t.Run("one location, same identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewInterfaceStaticTypeComputeTypeID(nil, nil, "X").
				Equal(NewInterfaceStaticTypeComputeTypeID(nil, common.StringLocation("B"), "X")),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			NewInterfaceStaticTypeComputeTypeID(nil, nil, "X").
				Equal(NewCompositeStaticTypeComputeTypeID(nil, nil, "X")),
		)
	})
}

func TestConstantSizedStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&ConstantSizedStaticType{
				Type: PrimitiveStaticTypeString,
				Size: 10,
			}).Equal(
				&ConstantSizedStaticType{
					Type: PrimitiveStaticTypeString,
					Size: 10,
				},
			),
		)
	})

	t.Run("different sizes", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&ConstantSizedStaticType{
				Type: PrimitiveStaticTypeString,
				Size: 20,
			}).Equal(
				&ConstantSizedStaticType{
					Type: PrimitiveStaticTypeString,
					Size: 10,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&ConstantSizedStaticType{
				Type: PrimitiveStaticTypeInt,
				Size: 10,
			}).Equal(
				&ConstantSizedStaticType{
					Type: PrimitiveStaticTypeString,
					Size: 10,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&ConstantSizedStaticType{
				Type: PrimitiveStaticTypeInt,
				Size: 10,
			}).Equal(
				&VariableSizedStaticType{
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
			(&VariableSizedStaticType{
				Type: PrimitiveStaticTypeString,
			}).Equal(
				&VariableSizedStaticType{
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			}).Equal(
				&ConstantSizedStaticType{
					Type: PrimitiveStaticTypeInt,
					Size: 10,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			}).Equal(
				&VariableSizedStaticType{
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
				Equal(&CapabilityStaticType{}),
		)
	})
}

func TestOptionalStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&OptionalStaticType{
				Type: PrimitiveStaticTypeString,
			}).Equal(
				&OptionalStaticType{
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&OptionalStaticType{
				Type: PrimitiveStaticTypeInt,
			}).Equal(
				&OptionalStaticType{
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&OptionalStaticType{
				Type: PrimitiveStaticTypeInt,
			}).Equal(
				&VariableSizedStaticType{
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
			(&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeString,
			}).Equal(
				&DictionaryStaticType{
					KeyType:   PrimitiveStaticTypeInt,
					ValueType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different key types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeString,
			}).Equal(
				&DictionaryStaticType{
					KeyType:   PrimitiveStaticTypeVoid,
					ValueType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different value types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeVoid,
			}).Equal(
				&DictionaryStaticType{
					KeyType:   PrimitiveStaticTypeInt,
					ValueType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeVoid,
			}).Equal(
				&VariableSizedStaticType{
					Type: PrimitiveStaticTypeInt,
				},
			),
		)
	})
}

func TestInclusiveRangeStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			InclusiveRangeStaticType{
				ElementType: PrimitiveStaticTypeInt256,
			}.Equal(
				InclusiveRangeStaticType{
					ElementType: PrimitiveStaticTypeInt256,
				},
			),
		)
	})

	t.Run("different member types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			InclusiveRangeStaticType{
				ElementType: PrimitiveStaticTypeInt,
			}.Equal(
				InclusiveRangeStaticType{
					ElementType: PrimitiveStaticTypeWord256,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			InclusiveRangeStaticType{
				ElementType: PrimitiveStaticTypeInt,
			}.Equal(
				&VariableSizedStaticType{
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
				Types: []*InterfaceStaticType{
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
				},
			}).Equal(
				&IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					},
				},
			),
		)
	})

	t.Run("equal, no intersections", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&IntersectionStaticType{
				Types: []*InterfaceStaticType{},
			}).Equal(
				&IntersectionStaticType{
					Types: []*InterfaceStaticType{},
				},
			),
		)
	})

	t.Run("fewer intersections", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Types: []*InterfaceStaticType{
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
				},
			}).Equal(
				&IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					},
				},
			),
		)
	})

	t.Run("same, restrictions in different order", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&IntersectionStaticType{
				Types: []*InterfaceStaticType{
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
				},
			}).Equal(
				&IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					},
				},
			),
		)
	})

	t.Run("same, restrictions in same order", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&IntersectionStaticType{
				Types: []*InterfaceStaticType{
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
				},
			}).Equal(
				&IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
					},
				},
			),
		)
	})

	t.Run("different intersections", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Types: []*InterfaceStaticType{
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
				},
			}).Equal(
				&IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Z"),
					},
				},
			),
		)
	})

	t.Run("more restrictions", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Types: []*InterfaceStaticType{
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
				},
			}).Equal(
				&IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
					},
				},
			),
		)
	})

	t.Run("different restrictions", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Types: []*InterfaceStaticType{
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
				},
			}).Equal(
				&IntersectionStaticType{
					Types: []*InterfaceStaticType{
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
						NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Z"),
					},
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&IntersectionStaticType{
				Types: []*InterfaceStaticType{
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "X"),
					NewInterfaceStaticTypeComputeTypeID(nil, utils.TestLocation, "Y"),
				},
			}).Equal(
				&ReferenceStaticType{
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
		require.Equal(t, byte(152), byte(PrimitiveStaticType_Count))
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

	testInterfaceStaticType := NewInterfaceStaticTypeComputeTypeID(
		nil,
		testLocation,
		testInterfaceQualifiedIdentifier,
	)

	const testCompositeQualifiedIdentifier = "TestComposite"

	testCompositeSemaType := &sema.CompositeType{
		Location:   testLocation,
		Identifier: testCompositeQualifiedIdentifier,
	}

	testCompositeStaticType := NewCompositeStaticTypeComputeTypeID(
		nil,
		testLocation,
		testCompositeQualifiedIdentifier,
	)

	testFunctionType := &sema.FunctionType{}

	type testCase struct {
		name         string
		semaType     sema.Type
		staticType   StaticType
		getInterface func(
			t *testing.T,
			location common.Location,
			qualifiedIdentifier string,
			typeID TypeID,
		) (
			*sema.InterfaceType,
			error,
		)
		getComposite func(
			t *testing.T,
			location common.Location,
			qualifiedIdentifier string,
			typeID TypeID,
		) (
			*sema.CompositeType,
			error,
		)
	}

	tests := []testCase{
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
			name:       "FixedSizeUnsignedInteger",
			semaType:   sema.FixedSizeUnsignedIntegerType,
			staticType: PrimitiveStaticTypeFixedSizeUnsignedInteger,
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
			name:       "Word256",
			semaType:   sema.Word256Type,
			staticType: PrimitiveStaticTypeWord256,
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
			name:       "Account",
			semaType:   sema.AccountType,
			staticType: PrimitiveStaticTypeAccount,
		},
		{
			name:       "DeployedContract",
			semaType:   sema.DeployedContractType,
			staticType: PrimitiveStaticTypeDeployedContract,
		},
		{
			name:       "Account.Storage",
			semaType:   sema.Account_StorageType,
			staticType: PrimitiveStaticTypeAccount_Storage,
		},
		{
			name:       "Account.Contracts",
			semaType:   sema.Account_ContractsType,
			staticType: PrimitiveStaticTypeAccount_Contracts,
		},
		{
			name:       "Account.Keys",
			semaType:   sema.Account_KeysType,
			staticType: PrimitiveStaticTypeAccount_Keys,
		},
		{
			name:       "Account.Inbox",
			semaType:   sema.Account_InboxType,
			staticType: PrimitiveStaticTypeAccount_Inbox,
		},
		{
			name:       "Account.Capabilities",
			semaType:   sema.Account_CapabilitiesType,
			staticType: PrimitiveStaticTypeAccount_Capabilities,
		},
		{
			name:       "Account.StorageCapabilities",
			semaType:   sema.Account_StorageCapabilitiesType,
			staticType: PrimitiveStaticTypeAccount_StorageCapabilities,
		},
		{
			name:       "Account.AccountCapabilities",
			semaType:   sema.Account_AccountCapabilitiesType,
			staticType: PrimitiveStaticTypeAccount_AccountCapabilities,
		},
		{
			name:       "StorageCapabilityController",
			semaType:   sema.StorageCapabilityControllerType,
			staticType: PrimitiveStaticTypeStorageCapabilityController,
		},
		{
			name:       "AccountCapabilityController",
			semaType:   sema.AccountCapabilityControllerType,
			staticType: PrimitiveStaticTypeAccountCapabilityController,
		},

		{
			name:       "AnyResourceAttachment",
			semaType:   sema.AnyResourceAttachmentType,
			staticType: PrimitiveStaticTypeAnyResourceAttachment,
		},

		{
			name:       "AnyStructAttachment",
			semaType:   sema.AnyStructAttachmentType,
			staticType: PrimitiveStaticTypeAnyStructAttachment,
		},
		{
			name:       "AccountKey",
			semaType:   sema.AccountKeyType,
			staticType: AccountKeyStaticType,
			getComposite: func(
				t *testing.T,
				location common.Location,
				qualifiedIdentifier string,
				_ TypeID,
			) (*sema.CompositeType, error) {
				require.Nil(t, location)
				require.Equal(t, "AccountKey", qualifiedIdentifier)
				return sema.AccountKeyType, nil
			},
		},
		{
			name:       "Mutate",
			semaType:   sema.MutateType,
			staticType: PrimitiveStaticTypeMutate,
		},
		{
			name:       "Insert",
			semaType:   sema.InsertType,
			staticType: PrimitiveStaticTypeInsert,
		},
		{
			name:       "Remove",
			semaType:   sema.RemoveType,
			staticType: PrimitiveStaticTypeRemove,
		},
		{
			name:       "Storage",
			semaType:   sema.StorageType,
			staticType: PrimitiveStaticTypeStorage,
		},
		{
			name:       "SaveValue",
			semaType:   sema.SaveValueType,
			staticType: PrimitiveStaticTypeSaveValue,
		},
		{
			name:       "LoadValue",
			semaType:   sema.LoadValueType,
			staticType: PrimitiveStaticTypeLoadValue,
		},
		{
			name:       "CopyValue",
			semaType:   sema.CopyValueType,
			staticType: PrimitiveStaticTypeCopyValue,
		},
		{
			name:       "BorrowValue",
			semaType:   sema.BorrowValueType,
			staticType: PrimitiveStaticTypeBorrowValue,
		},
		{
			name:       "Contracts",
			semaType:   sema.ContractsType,
			staticType: PrimitiveStaticTypeContracts,
		},
		{
			name:       "AddContract",
			semaType:   sema.AddContractType,
			staticType: PrimitiveStaticTypeAddContract,
		},
		{
			name:       "UpdateContract",
			semaType:   sema.UpdateContractType,
			staticType: PrimitiveStaticTypeUpdateContract,
		},
		{
			name:       "RemoveContract",
			semaType:   sema.RemoveContractType,
			staticType: PrimitiveStaticTypeRemoveContract,
		},
		{
			name:       "Keys",
			semaType:   sema.KeysType,
			staticType: PrimitiveStaticTypeKeys,
		},
		{
			name:       "AddKey",
			semaType:   sema.AddKeyType,
			staticType: PrimitiveStaticTypeAddKey,
		},
		{
			name:       "RevokeKey",
			semaType:   sema.RevokeKeyType,
			staticType: PrimitiveStaticTypeRevokeKey,
		},
		{
			name:       "Inbox",
			semaType:   sema.InboxType,
			staticType: PrimitiveStaticTypeInbox,
		},
		{
			name:       "PublishInboxCapability",
			semaType:   sema.PublishInboxCapabilityType,
			staticType: PrimitiveStaticTypePublishInboxCapability,
		},
		{
			name:       "UnpublishInboxCapability",
			semaType:   sema.UnpublishInboxCapabilityType,
			staticType: PrimitiveStaticTypeUnpublishInboxCapability,
		},
		{
			name:       "ClaimInboxCapability",
			semaType:   sema.ClaimInboxCapabilityType,
			staticType: PrimitiveStaticTypeClaimInboxCapability,
		},
		{
			name:       "Capabilities",
			semaType:   sema.CapabilitiesType,
			staticType: PrimitiveStaticTypeCapabilities,
		},
		{
			name:       "StorageCapabilities",
			semaType:   sema.StorageCapabilitiesType,
			staticType: PrimitiveStaticTypeStorageCapabilities,
		},
		{
			name:       "AccountCapabilities",
			semaType:   sema.AccountCapabilitiesType,
			staticType: PrimitiveStaticTypeAccountCapabilities,
		},
		{
			name:       "PublishCapability",
			semaType:   sema.PublishCapabilityType,
			staticType: PrimitiveStaticTypePublishCapability,
		},
		{
			name:       "UnpublishCapability",
			semaType:   sema.UnpublishCapabilityType,
			staticType: PrimitiveStaticTypeUnpublishCapability,
		},
		{
			name:       "GetStorageCapabilityController",
			semaType:   sema.GetStorageCapabilityControllerType,
			staticType: PrimitiveStaticTypeGetStorageCapabilityController,
		},
		{
			name:       "IssueStorageCapabilityController",
			semaType:   sema.IssueStorageCapabilityControllerType,
			staticType: PrimitiveStaticTypeIssueStorageCapabilityController,
		},
		{
			name:       "GetAccountCapabilityController",
			semaType:   sema.GetAccountCapabilityControllerType,
			staticType: PrimitiveStaticTypeGetAccountCapabilityController,
		},
		{
			name:       "IssueAccountCapabilityController",
			semaType:   sema.IssueAccountCapabilityControllerType,
			staticType: PrimitiveStaticTypeIssueAccountCapabilityController,
		},
		{
			name:       "CapabilitiesMapping",
			semaType:   sema.CapabilitiesMappingType,
			staticType: PrimitiveStaticTypeCapabilitiesMapping,
		},
		{
			name:       "AccountMapping",
			semaType:   sema.AccountMappingType,
			staticType: PrimitiveStaticTypeAccountMapping,
		},
		{
			name:       "Identity",
			semaType:   sema.IdentityType,
			staticType: PrimitiveStaticTypeIdentity,
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
			staticType: &CapabilityStaticType{
				BorrowType: PrimitiveStaticTypeInt,
			},
		},

		{
			name: "Variable-sized array",
			semaType: &sema.VariableSizedType{
				Type: sema.IntType,
			},
			staticType: &VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			},
		},
		{
			name: "Constant-sized array",
			semaType: &sema.ConstantSizedType{
				Type: sema.IntType,
				Size: 42,
			},
			staticType: &ConstantSizedStaticType{
				Type: PrimitiveStaticTypeInt,
				Size: 42,
			},
		},
		{
			name: "Optional",
			semaType: &sema.OptionalType{
				Type: sema.IntType,
			},
			staticType: &OptionalStaticType{
				Type: PrimitiveStaticTypeInt,
			},
		},
		{
			name: "Reference",
			semaType: &sema.ReferenceType{
				Type:          sema.IntType,
				Authorization: sema.UnauthorizedAccess,
			},
			staticType: &ReferenceStaticType{
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
			staticType: &DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeInt,
				ValueType: PrimitiveStaticTypeString,
			},
		},
		{
			name: "Intersection",
			semaType: &sema.IntersectionType{
				Types: []*sema.InterfaceType{
					testInterfaceSemaType,
				},
			},
			staticType: &IntersectionStaticType{
				Types: []*InterfaceStaticType{
					testInterfaceStaticType,
				},
			},
			getInterface: func(
				t *testing.T,
				location common.Location,
				qualifiedIdentifier string,
				typeID TypeID,
			) (*sema.InterfaceType, error) {
				require.Equal(t, testLocation, location)
				require.Equal(t, testInterfaceQualifiedIdentifier, qualifiedIdentifier)
				return testInterfaceSemaType, nil
			},
		},
		{
			name:       "Interface",
			semaType:   testInterfaceSemaType,
			staticType: testInterfaceStaticType,
			getInterface: func(
				t *testing.T,
				location common.Location,
				qualifiedIdentifier string,
				typeID TypeID,
			) (*sema.InterfaceType, error) {
				require.Equal(t, testLocation, location)
				require.Equal(t, testInterfaceQualifiedIdentifier, qualifiedIdentifier)
				return testInterfaceSemaType, nil
			},
		},
		{
			name:       "Composite",
			semaType:   testCompositeSemaType,
			staticType: testCompositeStaticType,
			getComposite: func(
				t *testing.T,
				location common.Location,
				qualifiedIdentifier string,
				typeID TypeID,
			) (*sema.CompositeType, error) {
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
		{
			name:       "HashableStruct",
			semaType:   sema.HashableStructType,
			staticType: PrimitiveStaticTypeHashableStruct,
		},
		{
			name: "InclusiveRange",
			semaType: &sema.InclusiveRangeType{
				MemberType: sema.IntType,
			},
			staticType: InclusiveRangeStaticType{
				ElementType: PrimitiveStaticTypeInt,
			},
		},
	}

	test := func(test testCase) {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			// Test sema to static

			convertedStaticType := ConvertSemaToStaticType(nil, test.semaType)
			require.Equal(t,
				test.staticType,
				convertedStaticType,
			)

			// Test static to sema

			getInterface := test.getInterface
			if getInterface == nil {
				getInterface = func(
					_ *testing.T,
					_ common.Location,
					_ string,
					_ TypeID,
				) (*sema.InterfaceType, error) {
					require.FailNow(t, "getInterface should not be called")
					return nil, nil
				}
			}

			getComposite := test.getComposite
			if getComposite == nil {
				getComposite = func(
					_ *testing.T,
					_ common.Location,
					_ string,
					_ TypeID,
				) (*sema.CompositeType, error) {
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
				func(location common.Location, qualifiedIdentifier string, typeID TypeID) (*sema.InterfaceType, error) {
					return getInterface(t, location, qualifiedIdentifier, typeID)
				},
				func(location common.Location, qualifiedIdentifier string, typeID TypeID) (*sema.CompositeType, error) {
					return getComposite(t, location, qualifiedIdentifier, typeID)
				},
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

	testedStaticTypes := map[StaticType]struct{}{}

	for _, testCase := range tests {
		testedStaticTypes[testCase.staticType] = struct{}{}
		test(testCase)
	}

	for ty := PrimitiveStaticType(1); ty < PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() || ty.IsDeprecated() { //nolint:staticcheck
			continue
		}
		if _, ok := testedStaticTypes[ty]; !ok {
			t.Errorf("missing test case for primitive static type %s", ty)
		}
	}

}

func TestIntersectionStaticType_ID(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("top-level, single", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionStaticType(
			nil,
			[]*InterfaceStaticType{
				NewInterfaceStaticTypeComputeTypeID(
					nil,
					testLocation,
					"I",
				),
			},
		)
		assert.Equal(t,
			TypeID("{S.test.I}"),
			intersectionType.ID(),
		)
	})

	t.Run("top-level, two", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionStaticType(
			nil,
			[]*InterfaceStaticType{
				// NOTE: order
				NewInterfaceStaticTypeComputeTypeID(
					nil,
					testLocation,
					"I2",
				),
				NewInterfaceStaticTypeComputeTypeID(
					nil,
					testLocation,
					"I1",
				),
			},
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("{S.test.I1,S.test.I2}"),
			intersectionType.ID(),
		)
	})

	t.Run("nested, two", func(t *testing.T) {
		t.Parallel()

		interfaceType1 := NewInterfaceStaticTypeComputeTypeID(
			nil,
			testLocation,
			"C.I1",
		)

		interfaceType2 := NewInterfaceStaticTypeComputeTypeID(
			nil,
			testLocation,
			"C.I2",
		)

		intersectionType := NewIntersectionStaticType(
			nil,
			[]*InterfaceStaticType{
				// NOTE: order
				interfaceType2,
				interfaceType1,
			},
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("{S.test.C.I1,S.test.C.I2}"),
			intersectionType.ID(),
		)
	})
}

func TestIntersectionStaticType_String(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("top-level, single", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionStaticType(
			nil,
			[]*InterfaceStaticType{
				NewInterfaceStaticTypeComputeTypeID(
					nil,
					testLocation,
					"I",
				),
			},
		)
		assert.Equal(t,
			"{S.test.I}",
			intersectionType.String(),
		)
	})

	t.Run("top-level, two", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionStaticType(
			nil,
			[]*InterfaceStaticType{
				// NOTE: order
				NewInterfaceStaticTypeComputeTypeID(
					nil,
					testLocation,
					"I2",
				),
				NewInterfaceStaticTypeComputeTypeID(
					nil,
					testLocation,
					"I1",
				),
			},
		)
		// NOTE: order
		assert.Equal(t,
			"{S.test.I2, S.test.I1}",
			intersectionType.String(),
		)
	})
}

func TestEntitlementMapAuthorization_ID(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("top-level", func(t *testing.T) {
		t.Parallel()

		mapTypeID := testLocation.TypeID(nil, "M")
		authorization := NewEntitlementMapAuthorization(nil, mapTypeID)
		assert.Equal(t, TypeID("S.test.M"), authorization.ID())
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		mapTypeID := testLocation.TypeID(nil, "C.M")
		authorization := NewEntitlementMapAuthorization(nil, mapTypeID)
		assert.Equal(t, TypeID("S.test.C.M"), authorization.ID())
	})
}

func TestEntitlementMapAuthorization_String(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("top-level", func(t *testing.T) {
		t.Parallel()

		mapTypeID := testLocation.TypeID(nil, "M")
		authorization := NewEntitlementMapAuthorization(nil, mapTypeID)
		assert.Equal(t, "auth(S.test.M) ", authorization.String())
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		mapTypeID := testLocation.TypeID(nil, "C.M")
		authorization := NewEntitlementMapAuthorization(nil, mapTypeID)
		assert.Equal(t, "auth(S.test.C.M) ", authorization.String())
	})
}

func TestEntitlementSetAuthorization_ID(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		authorization := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					testLocation.TypeID(nil, "E"),
				}
			},
			1,
			sema.Conjunction,
		)
		assert.Equal(t,
			TypeID("S.test.E"),
			authorization.ID(),
		)
	})

	t.Run("two, conjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "E2"),
					testLocation.TypeID(nil, "E1"),
				}
			},
			2,
			sema.Conjunction,
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("S.test.E1,S.test.E2"),
			access.ID(),
		)
	})

	t.Run("two, disjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "E2"),
					testLocation.TypeID(nil, "E1"),
				}
			},
			2,
			sema.Disjunction,
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("S.test.E1|S.test.E2"),
			access.ID(),
		)
	})

	t.Run("three, nested, conjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "C.E3"),
					testLocation.TypeID(nil, "C.E2"),
					testLocation.TypeID(nil, "C.E1"),
				}
			},
			3,
			sema.Conjunction,
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("S.test.C.E1,S.test.C.E2,S.test.C.E3"),
			access.ID(),
		)
	})

	t.Run("three, nested, disjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "C.E3"),
					testLocation.TypeID(nil, "C.E2"),
					testLocation.TypeID(nil, "C.E1"),
				}
			},
			3,
			sema.Disjunction,
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("S.test.C.E1|S.test.C.E2|S.test.C.E3"),
			access.ID(),
		)
	})
}

func TestEntitlementSetAuthorization_String(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		authorization := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					testLocation.TypeID(nil, "E"),
				}
			},
			1,
			sema.Conjunction,
		)
		assert.Equal(t,
			"auth(S.test.E) ",
			authorization.String(),
		)
	})

	t.Run("two, conjunction", func(t *testing.T) {
		t.Parallel()

		authorization := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "E2"),
					testLocation.TypeID(nil, "E1"),
				}
			},
			2,
			sema.Conjunction,
		)
		// NOTE: order
		assert.Equal(t,
			"auth(S.test.E2, S.test.E1) ",
			authorization.String(),
		)
	})

	t.Run("two, disjunction", func(t *testing.T) {
		t.Parallel()

		authorization := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "E2"),
					testLocation.TypeID(nil, "E1"),
				}
			},
			2,
			sema.Disjunction,
		)
		// NOTE: order
		assert.Equal(
			t,
			"auth(S.test.E2 | S.test.E1) ",
			authorization.String(),
		)
	})

	t.Run("three, nested, conjunction", func(t *testing.T) {
		t.Parallel()

		authorization := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "C.E3"),
					testLocation.TypeID(nil, "C.E2"),
					testLocation.TypeID(nil, "C.E1"),
				}
			},
			3,
			sema.Conjunction,
		)
		// NOTE: order
		assert.Equal(
			t,
			"auth(S.test.C.E3, S.test.C.E2, S.test.C.E1) ",
			authorization.String(),
		)
	})

	t.Run("three, nested, disjunction", func(t *testing.T) {
		t.Parallel()
		authorization := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "C.E3"),
					testLocation.TypeID(nil, "C.E2"),
					testLocation.TypeID(nil, "C.E1"),
				}
			},
			3,
			sema.Disjunction,
		)
		// NOTE: order
		assert.Equal(
			t,
			"auth(S.test.C.E3 | S.test.C.E2 | S.test.C.E1) ",
			authorization.String(),
		)
	})
}

func TestReferenceStaticType_ID(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("top-level, unauthorized", func(t *testing.T) {
		t.Parallel()

		referenceType := NewReferenceStaticType(nil, UnauthorizedAccess, PrimitiveStaticTypeInt)
		assert.Equal(t,
			TypeID("&Int"),
			referenceType.ID(),
		)
	})

	t.Run("top-level, authorized, map", func(t *testing.T) {
		t.Parallel()

		mapTypeID := testLocation.TypeID(nil, "M")
		access := NewEntitlementMapAuthorization(nil, mapTypeID)

		referenceType := NewReferenceStaticType(nil, access, PrimitiveStaticTypeInt)
		assert.Equal(t,
			TypeID("auth(S.test.M)&Int"),
			referenceType.ID(),
		)
	})

	t.Run("top-level, authorized, set", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "E2"),
					testLocation.TypeID(nil, "E1"),
				}
			},
			2,
			sema.Conjunction,
		)

		referenceType := NewReferenceStaticType(nil, access, PrimitiveStaticTypeInt)

		// NOTE: sorted
		assert.Equal(t,
			TypeID("auth(S.test.E1,S.test.E2)&Int"),
			referenceType.ID(),
		)
	})

	t.Run("nested, authorized, map", func(t *testing.T) {
		t.Parallel()

		mapTypeID := testLocation.TypeID(nil, "C.M")
		access := NewEntitlementMapAuthorization(nil, mapTypeID)

		referenceType := NewReferenceStaticType(nil, access, PrimitiveStaticTypeInt)
		assert.Equal(t,
			TypeID("auth(S.test.C.M)&Int"),
			referenceType.ID(),
		)
	})

	t.Run("nested, authorized, set", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "C.E2"),
					testLocation.TypeID(nil, "C.E1"),
				}
			},
			2,
			sema.Conjunction,
		)

		referenceType := NewReferenceStaticType(nil, access, PrimitiveStaticTypeInt)

		// NOTE: sorted
		assert.Equal(t,
			TypeID("auth(S.test.C.E1,S.test.C.E2)&Int"),
			referenceType.ID(),
		)
	})
}

func TestReferenceStaticType_String(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		referenceType := NewReferenceStaticType(nil, UnauthorizedAccess, PrimitiveStaticTypeInt)
		assert.Equal(t, "&Int", referenceType.String())
	})

	t.Run("top-level, authorized, map", func(t *testing.T) {
		t.Parallel()

		mapTypeID := testLocation.TypeID(nil, "M")
		access := NewEntitlementMapAuthorization(nil, mapTypeID)

		referenceType := NewReferenceStaticType(nil, access, PrimitiveStaticTypeInt)

		assert.Equal(t,
			"auth(S.test.M) &Int",
			referenceType.String(),
		)
	})

	t.Run("top-level, authorized, set", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "E2"),
					testLocation.TypeID(nil, "E1"),
				}
			},
			2,
			sema.Conjunction,
		)

		referenceType := NewReferenceStaticType(nil, access, PrimitiveStaticTypeInt)

		// NOTE: order
		assert.Equal(t,
			"auth(S.test.E2, S.test.E1) &Int",
			referenceType.String(),
		)
	})

	t.Run("nested, authorized, map", func(t *testing.T) {
		t.Parallel()

		mapTypeID := testLocation.TypeID(nil, "C.M")
		access := NewEntitlementMapAuthorization(nil, mapTypeID)

		referenceType := NewReferenceStaticType(nil, access, PrimitiveStaticTypeInt)

		assert.Equal(t,
			"auth(S.test.C.M) &Int",
			referenceType.String(),
		)
	})

	t.Run("nested, authorized, set", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAuthorization(
			nil,
			func() []TypeID {
				return []TypeID{
					// NOTE: order
					testLocation.TypeID(nil, "C.E2"),
					testLocation.TypeID(nil, "C.E1"),
				}
			},
			2,
			sema.Conjunction,
		)

		referenceType := NewReferenceStaticType(nil, access, PrimitiveStaticTypeInt)

		// NOTE: order
		assert.Equal(t,
			"auth(S.test.C.E2, S.test.C.E1) &Int",
			referenceType.String(),
		)
	})
}
