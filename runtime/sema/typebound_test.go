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

package sema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeBound_Satisfies(t *testing.T) {
	t.Parallel()

	t.Run("subtype", func(t *testing.T) {

		t.Parallel()

		typeBound := NewSubtypeTypeBound(IntegerType)

		assert.True(t, typeBound.Satisfies(IntegerType))
		assert.True(t, typeBound.Satisfies(NeverType))

		for _, integerType := range AllLeafIntegerTypes {
			assert.True(t, typeBound.Satisfies(integerType))
		}
	})

	t.Run("strict subtype", func(t *testing.T) {

		t.Parallel()

		typeBound := NewStrictSubtypeTypeBound(IntegerType)

		assert.False(t, typeBound.Satisfies(IntegerType))
		assert.True(t, typeBound.Satisfies(NeverType))

		for _, integerType := range AllLeafIntegerTypes {
			assert.Truef(t, typeBound.Satisfies(integerType), "%s should satisfy", integerType)
		}
	})

	t.Run("supertype", func(t *testing.T) {

		t.Parallel()

		typeBound := NewSupertypeTypeBound(NeverType)

		assert.True(t, typeBound.Satisfies(NeverType))
		assert.True(t, typeBound.Satisfies(IntegerType))

		for _, integerType := range AllLeafIntegerTypes {
			assert.True(t, typeBound.Satisfies(integerType))
		}
	})

	t.Run("strict supertype", func(t *testing.T) {

		t.Parallel()

		typeBound := NewStrictSupertypeTypeBound(NeverType)

		assert.False(t, typeBound.Satisfies(NeverType))
		assert.True(t, typeBound.Satisfies(IntegerType))

		for _, integerType := range AllLeafIntegerTypes {
			assert.True(t, typeBound.Satisfies(integerType))
		}
	})

	t.Run("conjunction", func(t *testing.T) {

		t.Parallel()

		typeBound := NewConjunctionTypeBound([]TypeBound{
			NewStrictSupertypeTypeBound(NeverType),
			NewStrictSubtypeTypeBound(FixedSizeUnsignedIntegerType),
		})

		assert.False(t, typeBound.Satisfies(FixedSizeUnsignedIntegerType))
		assert.False(t, typeBound.Satisfies(NeverType))

		for _, integerType := range AllLeafFixedSizeUnsignedIntegerTypes {
			assert.True(t, typeBound.Satisfies(integerType))
		}
	})

	t.Run("disjunction", func(t *testing.T) {

		t.Parallel()

		typeBound := NewDisjunctionTypeBound([]TypeBound{
			NewStrictSupertypeTypeBound(NeverType),
			NewStrictSubtypeTypeBound(NeverType),
		})

		assert.True(t, typeBound.Satisfies(FixedSizeUnsignedIntegerType))
		assert.True(t, typeBound.Satisfies(AnyStructType))
		assert.False(t, typeBound.Satisfies(NeverType))

		for _, integerType := range AllLeafFixedSizeUnsignedIntegerTypes {
			assert.True(t, typeBound.Satisfies(integerType))
		}
	})
}

func TestTypeBoundSerialization(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		bound    TypeBound
		expected string
	}

	tests := []testCase{
		{
			name:     "basic subtype",
			bound:    NewSubtypeTypeBound(IntType),
			expected: "<=: Int",
		},
		{
			name:     "basic equal",
			bound:    NewEqualTypeBound(IntType),
			expected: "= Int",
		},
		{
			name:     "basic not subtype",
			bound:    NewSubtypeTypeBound(IntType).Not(),
			expected: "!(<=: Int)",
		},
		{
			name:     "basic not equal",
			bound:    NewEqualTypeBound(IntType).Not(),
			expected: "!(= Int)",
		},
		{
			name:     "basic conjunction",
			bound:    NewSubtypeTypeBound(IntType).And(NewEqualTypeBound(StringType)),
			expected: "(<=: Int) && (= String)",
		},
		{
			name:     "conjunction of negations",
			bound:    NewSubtypeTypeBound(IntType).Not().And(NewEqualTypeBound(StringType).Not()),
			expected: "(!(<=: Int)) && (!(= String))",
		},
		{
			name:     "flattened double negative",
			bound:    NewSubtypeTypeBound(IntType).Not().Not(),
			expected: "<=: Int",
		},
		{
			name:     "strict subtype",
			bound:    NewStrictSubtypeTypeBound(IntType),
			expected: "<: Int",
		},
		{
			name:     "strict supertype",
			bound:    NewStrictSupertypeTypeBound(IntType),
			expected: ">: Int",
		},
		{
			name:     "supertype",
			bound:    NewSupertypeTypeBound(IntType),
			expected: ">=: Int",
		},
		{
			name:     "basic disjunction",
			bound:    NewSubtypeTypeBound(IntType).Or(NewEqualTypeBound(StringType)),
			expected: "(<=: Int) || (= String)",
		},
		{
			name: "triple conjunction",
			bound: NewSubtypeTypeBound(IntType).
				And(NewEqualTypeBound(StringType)).
				And(NewStrictSupertypeTypeBound(BoolType)),
			expected: "(<=: Int) && (= String) && (>: Bool)",
		},
		{
			name:     "Capabilities.get",
			bound:    Account_CapabilitiesTypeGetFunctionTypeParameterT.TypeBound,
			expected: "(<=: &Any) && (>: Never)",
		},
		{
			name:     "InclusiveRange",
			bound:    InclusiveRangeConstructorFunctionTypeParameter.TypeBound,
			expected: "(((= UInt) || (= Int)) || (<: FixedSizeUnsignedInteger)) || (<: SignedInteger)",
		},
		{
			name: "triple disjunction",
			bound: NewSubtypeTypeBound(IntType).
				Or(NewEqualTypeBound(StringType)).
				Or(NewStrictSupertypeTypeBound(BoolType)),
			expected: "((<=: Int) || (= String)) || (>: Bool)",
		},
		{
			name: "conjunction of disjunctions",
			bound: NewSubtypeTypeBound(IntType).
				Or(NewEqualTypeBound(StringType)).
				And(
					NewStrictSupertypeTypeBound(BoolType).
						Or(NewSupertypeTypeBound(AnyStructType)),
				),
			expected: "((<=: Int) || (= String)) && ((>: Bool) || (>=: AnyStruct))",
		},
	}

	test := func(test testCase) {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.bound.String(), test.expected)
		})
	}

	for _, testCase := range tests {
		test(testCase)
	}
}

func TestTypeBound_HasInvalid(t *testing.T) {
	t.Parallel()

	t.Run("subtype", func(t *testing.T) {

		t.Parallel()

		assert.False(t, NewSubtypeTypeBound(IntegerType).HasInvalidType())
		assert.True(t, NewSubtypeTypeBound(InvalidType).HasInvalidType())
	})

	t.Run("strict subtype", func(t *testing.T) {

		t.Parallel()

		assert.False(t, NewStrictSubtypeTypeBound(IntegerType).HasInvalidType())
		assert.True(t, NewStrictSubtypeTypeBound(InvalidType).HasInvalidType())
	})

	t.Run("supertype", func(t *testing.T) {

		t.Parallel()

		assert.False(t, NewSupertypeTypeBound(IntegerType).HasInvalidType())
		assert.True(t, NewSupertypeTypeBound(InvalidType).HasInvalidType())
	})

	t.Run("strict supertype", func(t *testing.T) {

		t.Parallel()

		assert.False(t, NewStrictSupertypeTypeBound(IntegerType).HasInvalidType())
		assert.True(t, NewStrictSupertypeTypeBound(InvalidType).HasInvalidType())
	})

	t.Run("conjunction", func(t *testing.T) {

		t.Parallel()

		assert.False(t,
			(&ConjunctionTypeBound{
				TypeBounds: []TypeBound{
					SubtypeTypeBound{Type: IntegerType},
				},
			}).HasInvalidType(),
		)

		assert.True(t,
			(&ConjunctionTypeBound{
				TypeBounds: []TypeBound{
					SubtypeTypeBound{Type: InvalidType},
				},
			}).HasInvalidType(),
		)
	})
}
