/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func TestInterpreterOptionalBoxing(t *testing.T) {

	t.Parallel()

	t.Run("Bool to Bool?", func(t *testing.T) {
		inter := newTestInterpreter(t)

		value := inter.BoxOptional(
			EmptyLocationRange,
			TrueValue,
			&sema.OptionalType{Type: sema.BoolType},
		)
		assert.Equal(t,
			NewUnmeteredSomeValueNonCopying(TrueValue),
			value,
		)
	})

	t.Run("Bool? to Bool?", func(t *testing.T) {
		inter := newTestInterpreter(t)

		value := inter.BoxOptional(
			EmptyLocationRange,
			NewUnmeteredSomeValueNonCopying(TrueValue),
			&sema.OptionalType{Type: sema.BoolType},
		)
		assert.Equal(t,
			NewUnmeteredSomeValueNonCopying(TrueValue),
			value,
		)
	})

	t.Run("Bool? to Bool??", func(t *testing.T) {
		inter := newTestInterpreter(t)

		value := inter.BoxOptional(
			EmptyLocationRange,
			NewUnmeteredSomeValueNonCopying(TrueValue),
			&sema.OptionalType{Type: &sema.OptionalType{Type: sema.BoolType}},
		)
		assert.Equal(t,
			NewUnmeteredSomeValueNonCopying(
				NewUnmeteredSomeValueNonCopying(TrueValue),
			),
			value,
		)
	})

	t.Run("nil (Never?) to Bool??", func(t *testing.T) {
		inter := newTestInterpreter(t)

		// NOTE:
		value := inter.BoxOptional(
			EmptyLocationRange,
			Nil,
			&sema.OptionalType{Type: &sema.OptionalType{Type: sema.BoolType}},
		)
		assert.Equal(t,
			Nil,
			value,
		)
	})

	t.Run("nil (Some(nil): Never??) to Bool??", func(t *testing.T) {
		inter := newTestInterpreter(t)

		// NOTE:
		value := inter.BoxOptional(
			EmptyLocationRange,
			NewUnmeteredSomeValueNonCopying(Nil),
			&sema.OptionalType{Type: &sema.OptionalType{Type: sema.BoolType}},
		)
		assert.Equal(t,
			Nil,
			value,
		)
	})
}

func TestInterpreterBoxing(t *testing.T) {

	t.Parallel()

	for _, anyType := range []sema.Type{
		sema.AnyStructType,
		sema.AnyResourceType,
	} {

		t.Run(anyType.String(), func(t *testing.T) {

			t.Run(fmt.Sprintf("Bool to %s?", anyType), func(t *testing.T) {
				inter := newTestInterpreter(t)

				assert.Equal(t,
					NewUnmeteredSomeValueNonCopying(
						TrueValue,
					),
					inter.ConvertAndBox(
						EmptyLocationRange,
						TrueValue,
						sema.BoolType,
						&sema.OptionalType{Type: anyType},
					),
				)

			})

			t.Run(fmt.Sprintf("Bool? to %s?", anyType), func(t *testing.T) {
				inter := newTestInterpreter(t)

				assert.Equal(t,
					NewUnmeteredSomeValueNonCopying(
						TrueValue,
					),
					inter.ConvertAndBox(
						EmptyLocationRange,
						NewUnmeteredSomeValueNonCopying(TrueValue),
						&sema.OptionalType{Type: sema.BoolType},
						&sema.OptionalType{Type: anyType},
					),
				)
			})
		})
	}
}

func BenchmarkValueIsSubtypeOfSemaType(b *testing.B) {

	b.ReportAllocs()

	const size = 1000
	values := make([]Value, 0, size)

	for i := 0; i < size; i++ {
		value := NewUnmeteredStringValue(fmt.Sprintf("value%d", i))
		values = append(values, value)
	}

	inter := newTestInterpreter(b)
	owner := common.Address{'A'}
	typ := ConstantSizedStaticType{
		Type: PrimitiveStaticTypeString,
		Size: size,
	}

	semaType := &sema.ConstantSizedType{
		Type: sema.StringType,
		Size: size,
	}

	array := NewArrayValue(
		inter,
		EmptyLocationRange,
		typ,
		owner,
		values...,
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ok := inter.ValueIsSubtypeOfSemaType(array, semaType)
		assert.True(b, ok)
	}
}
