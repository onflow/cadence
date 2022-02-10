/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRLPDecodeString(t *testing.T) {

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
		input          interpreter.Value
		output         interpreter.Value
		expectedErrMsg string
	}{
		{ // empty input
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
			),
			nil,
			"failed to RLP-decode string: input data is empty",
		},
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
			"",
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
			"",
		},
		{ //  single char with an extra trailing byte
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(65),
				interpreter.UInt8Value(1),
			),
			nil,
			"failed to RLP-decode list: input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes.",
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
			"",
		},
		{ // dog str with an extra trailing byte
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(131), // 0x83
				interpreter.UInt8Value(100), // 0x64
				interpreter.UInt8Value(111), // 0x6f
				interpreter.UInt8Value(103), // 0x67
				interpreter.UInt8Value(1),   // extra byte
			),
			nil,
			"failed to RLP-decode list: input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes.",
		},
		{ // handling lower level errors - incomplete data case
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(131),
			),
			nil,
			"failed to RLP-decode string: incomplete input! not enough bytes to read",
		},
		// { // wrong input type
		// 	interpreter.NewArrayValue(
		// 		inter,
		// 		interpreter.VariableSizedStaticType{
		// 			Type: interpreter.ByteArrayStaticType,
		// 		},
		// 		common.Address{},
		// 		interpreter.UInt8Value(128),
		// 	),
		// 	interpreter.NewArrayValue(
		// 		inter,
		// 		interpreter.ByteArrayStaticType,
		// 		common.Address{},
		// 	),
		// 	"",
		// },
	}

	for _, test := range tests {
		output, err := inter.Invoke(
			"DecodeRLPString",
			test.input,
		)
		if len(test.expectedErrMsg) > 0 {
			require.Error(t, err)
			assert.Equal(t, test.expectedErrMsg, err.Error())
			continue
		}
		require.NoError(t, err)
		utils.AssertValuesEqual(t, inter, test.output, output)
	}
}

func TestRLPDecodeList(t *testing.T) {

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
		input          interpreter.Value
		output         interpreter.Value
		expectedErrMsg string
	}{
		{ // empty input
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
			),
			nil,
			"failed to RLP-decode list: input data is empty",
		},
		{ // empty list
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(192),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.ByteArrayStaticType,
				},
				common.Address{},
			),
			"",
		},
		{ // single element list
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(193),
				interpreter.UInt8Value(65),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.ByteArrayStaticType,
				},
				common.Address{},
				interpreter.NewArrayValue(
					inter,
					interpreter.ByteArrayStaticType,
					common.Address{},
					interpreter.UInt8Value('A'),
				),
			),
			"",
		},
		{ // single element list with trailing extra bytes
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(193),
				interpreter.UInt8Value(65),
				interpreter.UInt8Value(65), // extra byte
			),
			nil,
			"failed to RLP-decode list: input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes.",
		},
		{ // multiple member list
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(200),
				interpreter.UInt8Value(131),
				interpreter.UInt8Value(65),
				interpreter.UInt8Value(66),
				interpreter.UInt8Value(67),
				interpreter.UInt8Value(131),
				interpreter.UInt8Value(69),
				interpreter.UInt8Value(70),
				interpreter.UInt8Value(71),
			),
			interpreter.NewArrayValue(
				inter,
				interpreter.VariableSizedStaticType{
					Type: interpreter.ByteArrayStaticType,
				},
				common.Address{},
				interpreter.NewArrayValue(
					inter,
					interpreter.ByteArrayStaticType,
					common.Address{},
					interpreter.UInt8Value(131),
					interpreter.UInt8Value(65),
					interpreter.UInt8Value(66),
					interpreter.UInt8Value(67),
				),
				interpreter.NewArrayValue(
					inter,
					interpreter.ByteArrayStaticType,
					common.Address{},
					interpreter.UInt8Value(131),
					interpreter.UInt8Value(69),
					interpreter.UInt8Value(70),
					interpreter.UInt8Value(71),
				),
			),
			"",
		},
		{ // multiple member list with an extra trailing byte
			interpreter.NewArrayValue(
				inter,
				interpreter.ByteArrayStaticType,
				common.Address{},
				interpreter.UInt8Value(200),
				interpreter.UInt8Value(131),
				interpreter.UInt8Value(65),
				interpreter.UInt8Value(66),
				interpreter.UInt8Value(67),
				interpreter.UInt8Value(131),
				interpreter.UInt8Value(69),
				interpreter.UInt8Value(70),
				interpreter.UInt8Value(71),
				interpreter.UInt8Value(55),
			),
			nil,
			"failed to RLP-decode list: input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes.",
		},
	}

	for _, test := range tests {
		decoded, err := inter.Invoke(
			"DecodeRLPList",
			test.input,
		)
		if len(test.expectedErrMsg) > 0 {
			require.Error(t, err)
			assert.Equal(t, test.expectedErrMsg, err.Error())
			continue
		}
		require.NoError(t, err)
		utils.AssertValuesEqual(t, inter, test.output, decoded)
	}
}
