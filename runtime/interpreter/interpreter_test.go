/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpreterOptionalBoxing(t *testing.T) {

	t.Parallel()

	checker, err := sema.NewChecker(nil, utils.TestLocation)
	require.NoError(t, err)

	program := ProgramFromChecker(checker)

	inter, err := NewInterpreter(program, checker.Location)
	require.NoError(t, err)

	t.Run("Bool to Bool?", func(t *testing.T) {
		value := inter.BoxOptional(
			BoolValue(true),
			sema.BoolType,
			&sema.OptionalType{Type: sema.BoolType},
		)
		assert.Equal(t,
			NewSomeValueNonCopying(BoolValue(true)),
			value,
		)
	})

	t.Run("Bool? to Bool?", func(t *testing.T) {
		value := inter.BoxOptional(
			NewSomeValueNonCopying(BoolValue(true)),
			&sema.OptionalType{Type: sema.BoolType},
			&sema.OptionalType{Type: sema.BoolType},
		)
		assert.Equal(t,
			NewSomeValueNonCopying(BoolValue(true)),
			value,
		)
	})

	t.Run("Bool? to Bool??", func(t *testing.T) {
		value := inter.BoxOptional(
			NewSomeValueNonCopying(BoolValue(true)),
			&sema.OptionalType{Type: sema.BoolType},
			&sema.OptionalType{Type: &sema.OptionalType{Type: sema.BoolType}},
		)
		assert.Equal(t,
			NewSomeValueNonCopying(
				NewSomeValueNonCopying(BoolValue(true)),
			),
			value,
		)
	})

	t.Run("nil (Never?) to Bool??", func(t *testing.T) {
		// NOTE:
		value := inter.BoxOptional(
			NilValue{},
			&sema.OptionalType{Type: sema.NeverType},
			&sema.OptionalType{Type: &sema.OptionalType{Type: sema.BoolType}},
		)
		assert.Equal(t,
			NilValue{},
			value,
		)
	})

	t.Run("nil (Some(nil): Never??) to Bool??", func(t *testing.T) {
		// NOTE:
		value := inter.BoxOptional(
			NewSomeValueNonCopying(NilValue{}),
			&sema.OptionalType{Type: &sema.OptionalType{Type: sema.NeverType}},
			&sema.OptionalType{Type: &sema.OptionalType{Type: sema.BoolType}},
		)
		assert.Equal(t,
			NilValue{},
			value,
		)
	})
}

func TestInterpreterBoxing(t *testing.T) {

	t.Parallel()

	checker, err := sema.NewChecker(nil, utils.TestLocation)
	require.NoError(t, err)

	program := ProgramFromChecker(checker)

	inter, err := NewInterpreter(program, checker.Location)
	require.NoError(t, err)

	for _, anyType := range []sema.Type{
		sema.AnyStructType,
		sema.AnyResourceType,
	} {

		t.Run(anyType.String(), func(t *testing.T) {

			t.Run(fmt.Sprintf("Bool to %s?", anyType), func(t *testing.T) {

				assert.Equal(t,
					NewSomeValueNonCopying(
						BoolValue(true),
					),
					inter.ConvertAndBox(
						BoolValue(true),
						sema.BoolType,
						&sema.OptionalType{Type: anyType},
					),
				)

			})

			t.Run(fmt.Sprintf("Bool? to %s?", anyType), func(t *testing.T) {

				assert.Equal(t,
					NewSomeValueNonCopying(
						BoolValue(true),
					),
					inter.ConvertAndBox(
						NewSomeValueNonCopying(BoolValue(true)),
						&sema.OptionalType{Type: sema.BoolType},
						&sema.OptionalType{Type: anyType},
					),
				)
			})
		})
	}
}

func BenchmarkTransfer(b *testing.B) {

	b.ReportAllocs()

	const size = 1000
	values := make([]Value, 0, size)

	for i := 0; i < size; i++ {
		value := NewStringValue(fmt.Sprintf("value%d", i))
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

	array := NewArrayValue(inter, typ, owner, values...)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ok := inter.CheckValueTransferTargetType(array, semaType)
		assert.True(b, ok)
	}
}
