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

package opcode

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintRecursionFib(t *testing.T) {
	t.Parallel()

	code := []byte{
		// if n < 2
		byte(GetLocal), 0, 0,
		byte(GetConstant), 0, 0,
		byte(IntLess),
		byte(JumpIfFalse), 0, 14,
		// then return n
		byte(GetLocal), 0, 0,
		byte(ReturnValue),
		// fib(n - 1)
		byte(GetLocal), 0, 0,
		byte(GetConstant), 0, 1,
		byte(IntSubtract),
		byte(Transfer), 0, 0,
		byte(GetGlobal), 0, 0,
		byte(Invoke), 0, 0,
		// fib(n - 2)
		byte(GetLocal), 0, 0,
		byte(GetConstant), 0, 0,
		byte(IntSubtract),
		byte(Transfer), 0, 0,
		byte(GetGlobal), 0, 0,
		byte(Invoke), 0, 0,
		// return sum
		byte(IntAdd),
		byte(ReturnValue),
	}

	const expected = `GetLocal 0
GetConstant 0
IntLess
JumpIfFalse 14
GetLocal 0
ReturnValue
GetLocal 0
GetConstant 1
IntSubtract
Transfer 0
GetGlobal 0
Invoke typeParamCount:0 typeParams:[]
GetLocal 0
GetConstant 0
IntSubtract
Transfer 0
GetGlobal 0
Invoke typeParamCount:0 typeParams:[]
IntAdd
ReturnValue
`

	var builder strings.Builder
	reader := bytes.NewReader(code)
	err := PrintInstructions(&builder, reader)
	require.NoError(t, err)

	assert.Equal(t, expected, builder.String())
}

func TestPrintInstruction(t *testing.T) {
	t.Parallel()

	instructions := map[string][]byte{
		"GetConstant 258": {byte(GetConstant), 1, 2},
		"GetLocal 258":    {byte(GetLocal), 1, 2},
		"SetLocal 258":    {byte(SetLocal), 1, 2},
		"GetGlobal 258":   {byte(GetGlobal), 1, 2},
		"SetGlobal 258":   {byte(SetGlobal), 1, 2},
		"Jump 258":        {byte(Jump), 1, 2},
		"JumpIfFalse 258": {byte(JumpIfFalse), 1, 2},
		"Transfer 258":    {byte(Transfer), 1, 2},

		"New kind:258 typeIndex:772": {byte(New), 1, 2, 3, 4},

		"Cast typeIndex:258 castKind:3": {byte(Cast), 1, 2, 3},

		`Path domain:1 identifier:"hello"`: {byte(Path), 1, 0, 5, 'h', 'e', 'l', 'l', 'o'},

		`InvokeDynamic funcName:"abc" typeParamCount:2 typeParams:[772, 1286] argsCount:1800`: {
			byte(InvokeDynamic), 0, 3, 'a', 'b', 'c', 0, 2, 3, 4, 5, 6, 7, 8,
		},

		"Invoke typeParamCount:2 typeParams:[772, 1286]": {
			byte(Invoke), 0, 2, 3, 4, 5, 6,
		},

		"Unknown":           {byte(Unknown)},
		"Return":            {byte(Return)},
		"ReturnValue":       {byte(ReturnValue)},
		"IntAdd":            {byte(IntAdd)},
		"IntSubtract":       {byte(IntSubtract)},
		"IntMultiply":       {byte(IntMultiply)},
		"IntDivide":         {byte(IntDivide)},
		"IntMod":            {byte(IntMod)},
		"IntLess":           {byte(IntLess)},
		"IntGreater":        {byte(IntGreater)},
		"IntLessOrEqual":    {byte(IntLessOrEqual)},
		"IntGreaterOrEqual": {byte(IntGreaterOrEqual)},
		"Equal":             {byte(Equal)},
		"NotEqual":          {byte(NotEqual)},
		"Unwrap":            {byte(Unwrap)},
		"Destroy":           {byte(Destroy)},
		"True":              {byte(True)},
		"False":             {byte(False)},
		"Nil":               {byte(Nil)},
		"NewArray":          {byte(NewArray)},
		"NewDictionary":     {byte(NewDictionary)},
		"NewRef":            {byte(NewRef)},
		"GetField":          {byte(GetField)},
		"SetField":          {byte(SetField)},
		"SetIndex":          {byte(SetIndex)},
		"GetIndex":          {byte(GetIndex)},
		"Drop":              {byte(Drop)},
		"Dup":               {byte(Dup)},
	}

	for expected, instruction := range instructions {
		t.Run(expected, func(t *testing.T) {

			var builder strings.Builder
			reader := bytes.NewReader(instruction)
			err := PrintInstruction(&builder, reader)
			require.NoError(t, err)
			assert.Equal(t, 0, reader.Len())
			assert.Equal(t, expected, builder.String())
		})
	}
}
