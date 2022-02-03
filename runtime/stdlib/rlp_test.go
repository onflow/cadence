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

package stdlib

import (
	"testing"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRLPReadString(t *testing.T) {

	t.Parallel()

	checker, err := sema.NewChecker(
		&ast.Program{},
		utils.TestLocation,
		sema.WithPredeclaredValues(BuiltinFunctions.ToSemaValueDeclarations()),
	)
	require.Nil(t, err)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreter.WithStorage(interpreter.NewInMemoryStorage()),
		interpreter.WithPredeclaredValues(
			BuiltinFunctions.ToInterpreterValueDeclarations(),
		),
	)
	require.Nil(t, err)

	tests := []struct {
		input       interpreter.Value
		output      interpreter.Value
		expectedErr error
	}{
		{ // empty string
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(128),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
			),
			nil,
		},
		{ // single char
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(47),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(47),
			),
			nil,
		},
		{ // dog
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(131), // 0x83
				interpreter.UInt8Value(100), // 0x64
				interpreter.UInt8Value(111), // 0x6f
				interpreter.UInt8Value(103), // 0x67
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value('d'),
				interpreter.UInt8Value('o'),
				interpreter.UInt8Value('g'),
			),
			nil,
		},

		// TODO add some cases with errors
	}

	for _, test := range tests {
		output, err := inter.Invoke(
			"RLPDecodeString",
			test.input,
		)
		outputArray := output.(*interpreter.ArrayValue)
		expectedOutputArray := test.output.(*interpreter.ArrayValue)
		assert.Equal(t, test.expectedErr, err)
		assert.Equal(t, expectedOutputArray.Count(), outputArray.Count())
		for i := 0; i < expectedOutputArray.Count(); i++ {
			assert.Equal(t,
				expectedOutputArray.Get(inter, interpreter.ReturnEmptyLocationRange, i),
				outputArray.Get(inter, interpreter.ReturnEmptyLocationRange, i))
		}

	}
}
