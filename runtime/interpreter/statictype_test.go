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

package interpreter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
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

		require.True(t, CapabilityStaticType{}.Equal(CapabilityStaticType{}))
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
					Type: PrimitiveStaticTypeString,
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
				Authorized: false,
				Type:       PrimitiveStaticTypeString,
			}.Equal(
				ReferenceStaticType{
					Authorized: false,
					Type:       PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ReferenceStaticType{
				Authorized: false,
				Type:       PrimitiveStaticTypeInt,
			}.Equal(
				ReferenceStaticType{
					Authorized: false,
					Type:       PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different auth", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ReferenceStaticType{
				Authorized: false,
				Type:       PrimitiveStaticTypeInt,
			}.Equal(
				ReferenceStaticType{
					Authorized: true,
					Type:       PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			ReferenceStaticType{
				Type: PrimitiveStaticTypeString,
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
			CompositeStaticType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "X",
			}.Equal(
				CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("different name", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CompositeStaticType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "X",
			}.Equal(
				CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "Y",
				},
			),
		)
	})

	t.Run("different locations, different identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CompositeStaticType{
				Location:            common.IdentifierLocation("A"),
				QualifiedIdentifier: "X",
			}.Equal(
				CompositeStaticType{
					Location:            common.IdentifierLocation("B"),
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("different locations, different identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CompositeStaticType{
				Location:            common.IdentifierLocation("A"),
				QualifiedIdentifier: "X",
			}.Equal(
				CompositeStaticType{
					Location:            common.StringLocation("A"),
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("no location", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			CompositeStaticType{
				Location:            nil,
				QualifiedIdentifier: "X",
			}.Equal(
				CompositeStaticType{
					Location:            nil,
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("no location, different identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CompositeStaticType{
				Location:            nil,
				QualifiedIdentifier: "X",
			}.Equal(
				CompositeStaticType{
					Location:            nil,
					QualifiedIdentifier: "Y",
				},
			),
		)
	})

	t.Run("one location, same identifier", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CompositeStaticType{
				Location:            nil,
				QualifiedIdentifier: "X",
			}.Equal(
				CompositeStaticType{
					Location:            common.StringLocation("B"),
					QualifiedIdentifier: "X",
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			CompositeStaticType{
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
				CompositeStaticType{
					Location:            nil,
					QualifiedIdentifier: "X",
				},
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

		require.True(t,
			PrimitiveStaticTypeString.
				Equal(PrimitiveStaticTypeString),
		)
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

func TestRestrictedStaticType_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&RestrictedStaticType{
				Type: PrimitiveStaticTypeInt,
				Restrictions: []InterfaceStaticType{
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
				&RestrictedStaticType{
					Type: PrimitiveStaticTypeInt,
					Restrictions: []InterfaceStaticType{
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

	t.Run("equal, no restrictions", func(t *testing.T) {

		t.Parallel()

		require.True(t,
			(&RestrictedStaticType{
				Type:         PrimitiveStaticTypeInt,
				Restrictions: []InterfaceStaticType{},
			}).Equal(
				&RestrictedStaticType{
					Type:         PrimitiveStaticTypeInt,
					Restrictions: []InterfaceStaticType{},
				},
			),
		)
	})

	t.Run("different restricted type", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&RestrictedStaticType{
				Type: PrimitiveStaticTypeString,
				Restrictions: []InterfaceStaticType{
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
				&RestrictedStaticType{
					Type: PrimitiveStaticTypeInt,
					Restrictions: []InterfaceStaticType{
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

	t.Run("fewer restrictions", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&RestrictedStaticType{
				Type: PrimitiveStaticTypeInt,
				Restrictions: []InterfaceStaticType{
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
				&RestrictedStaticType{
					Type: PrimitiveStaticTypeInt,
					Restrictions: []InterfaceStaticType{
						{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "Y",
						},
					},
				},
			),
		)
	})

	t.Run("more restrictions", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&RestrictedStaticType{
				Type: PrimitiveStaticTypeInt,
				Restrictions: []InterfaceStaticType{
					{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "X",
					},
				},
			}).Equal(
				&RestrictedStaticType{
					Type: PrimitiveStaticTypeInt,
					Restrictions: []InterfaceStaticType{
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

	t.Run("different restrictions", func(t *testing.T) {

		t.Parallel()

		require.False(t,
			(&RestrictedStaticType{
				Type: PrimitiveStaticTypeInt,
				Restrictions: []InterfaceStaticType{
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
				&RestrictedStaticType{
					Type: PrimitiveStaticTypeInt,
					Restrictions: []InterfaceStaticType{
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
			(&RestrictedStaticType{
				Type: PrimitiveStaticTypeInt,
				Restrictions: []InterfaceStaticType{
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
					Type: PrimitiveStaticTypeInt,
				},
			),
		)
	})
}
