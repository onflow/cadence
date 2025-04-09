/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/test_utils"
)

var compile = flag.Bool("compile", false, "Run tests using the compiler")

func parseCheckAndPrepare(tb testing.TB, code string) test_utils.Invokable {
	tb.Helper()
	return test_utils.ParseCheckAndPrepare(tb, code, *compile)
}

func TestInterpreterOptionalBoxing(t *testing.T) {

	t.Parallel()

	t.Run("Bool to Bool?", func(t *testing.T) {
		inter := newTestInterpreter(t)

		value := BoxOptional(
			inter,
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

		value := BoxOptional(
			inter,
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

		value := BoxOptional(
			inter,
			NewUnmeteredSomeValueNonCopying(TrueValue),
			&sema.OptionalType{
				Type: &sema.OptionalType{
					Type: sema.BoolType,
				},
			},
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
		value := BoxOptional(
			inter,
			Nil,
			&sema.OptionalType{
				Type: &sema.OptionalType{
					Type: sema.BoolType,
				},
			},
		)
		assert.Equal(t,
			Nil,
			value,
		)
	})

	t.Run("nil (Some(nil): Never??) to Bool??", func(t *testing.T) {
		inter := newTestInterpreter(t)

		// NOTE:
		value := BoxOptional(
			inter,
			NewUnmeteredSomeValueNonCopying(Nil),
			&sema.OptionalType{
				Type: &sema.OptionalType{
					Type: sema.BoolType,
				},
			},
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
					ConvertAndBox(
						inter,
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
					ConvertAndBox(
						inter,
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
	typ := &ConstantSizedStaticType{
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
