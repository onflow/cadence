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

package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestLegacyEquality(t *testing.T) {

	t.Parallel()

	t.Run("Character value", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(&LegacyCharacterValue{
				CharacterValue: interpreter.NewUnmeteredCharacterValue("foo"),
			}).Equal(nil, emptyLocationRange, &LegacyCharacterValue{
				CharacterValue: interpreter.NewUnmeteredCharacterValue("foo"),
			}),
		)
	})

	t.Run("String value", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(&LegacyStringValue{
				StringValue: interpreter.NewUnmeteredStringValue("foo"),
			}).Equal(nil, emptyLocationRange, &LegacyStringValue{
				StringValue: interpreter.NewUnmeteredStringValue("foo"),
			}),
		)
	})

	t.Run("Intersection type", func(t *testing.T) {
		t.Parallel()

		fooType := interpreter.NewInterfaceStaticTypeComputeTypeID(
			nil,
			utils.TestLocation,
			"Test.Foo",
		)

		require.True(t,
			(&LegacyIntersectionType{
				IntersectionStaticType: interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						fooType,
					},
				),
			}).Equal(&LegacyIntersectionType{
				IntersectionStaticType: interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						fooType,
					},
				),
			}),
		)
	})

	t.Run("Primitive type", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(LegacyPrimitiveStaticType{
				PrimitiveStaticType: interpreter.PrimitiveStaticTypeInt,
			}).Equal(LegacyPrimitiveStaticType{
				PrimitiveStaticType: interpreter.PrimitiveStaticTypeInt,
			}),
		)
	})

	t.Run("Reference type", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(&LegacyReferenceType{
				ReferenceStaticType: &interpreter.ReferenceStaticType{
					Authorization:  interpreter.UnauthorizedAccess,
					ReferencedType: interpreter.PrimitiveStaticTypeInt,
				},
			}).Equal(&LegacyReferenceType{
				ReferenceStaticType: &interpreter.ReferenceStaticType{
					Authorization:  interpreter.UnauthorizedAccess,
					ReferencedType: interpreter.PrimitiveStaticTypeInt,
				},
			}),
		)
	})

	t.Run("Optional type", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(&LegacyOptionalType{
				OptionalStaticType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			}).Equal(&LegacyOptionalType{
				OptionalStaticType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			}),
		)
	})
}
