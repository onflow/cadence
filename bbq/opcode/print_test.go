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

	const expected = `GetLocal localIndex:0
GetConstant constantIndex:0
IntLess
JumpIfFalse target:14
GetLocal localIndex:0
ReturnValue
GetLocal localIndex:0
GetConstant constantIndex:1
IntSubtract
Transfer typeIndex:0
GetGlobal globalIndex:0
Invoke typeArgs:[]
GetLocal localIndex:0
GetConstant constantIndex:0
IntSubtract
Transfer typeIndex:0
GetGlobal globalIndex:0
Invoke typeArgs:[]
IntAdd
ReturnValue
`

	var builder strings.Builder
	err := PrintBytecode(&builder, code)
	require.NoError(t, err)

	assert.Equal(t, expected, builder.String())
}

func TestPrintInstruction(t *testing.T) {
	t.Parallel()

	instructions := map[string][]byte{
		"GetConstant constantIndex:258": {byte(GetConstant), 1, 2},
		"GetLocal localIndex:258":       {byte(GetLocal), 1, 2},
		"SetLocal localIndex:258":       {byte(SetLocal), 1, 2},
		"GetGlobal globalIndex:258":     {byte(GetGlobal), 1, 2},
		"SetGlobal globalIndex:258":     {byte(SetGlobal), 1, 2},
		"Jump target:258":               {byte(Jump), 1, 2},
		"JumpIfFalse target:258":        {byte(JumpIfFalse), 1, 2},
		"Transfer typeIndex:258":        {byte(Transfer), 1, 2},

		"New kind:258 typeIndex:772": {byte(New), 1, 2, 3, 4},

		"Cast typeIndex:258 kind:3": {byte(Cast), 1, 2, 3},

		`Path domain:1 identifier:"hello"`: {byte(Path), 1, 0, 5, 'h', 'e', 'l', 'l', 'o'},

		`InvokeDynamic name:"abc" typeArgs:[772 1286] argCount:1800`: {
			byte(InvokeDynamic), 0, 3, 'a', 'b', 'c', 0, 2, 3, 4, 5, 6, 7, 8,
		},

		"Invoke typeArgs:[772 1286]": {
			byte(Invoke), 0, 2, 3, 4, 5, 6,
		},

		"NewRef typeIndex:258": {byte(NewRef), 1, 2},

		"NewArray typeIndex:258 size:772 isResource:true": {byte(NewArray), 1, 2, 3, 4, 1},

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
		"GetField":          {byte(GetField)},
		"SetField":          {byte(SetField)},
		"SetIndex":          {byte(SetIndex)},
		"GetIndex":          {byte(GetIndex)},
		"Drop":              {byte(Drop)},
		"Dup":               {byte(Dup)},
	}

	for expected, code := range instructions {
		t.Run(expected, func(t *testing.T) {

			var ip uint16
			instruction := DecodeInstruction(&ip, code)
			assert.Equal(t, expected, instruction.String())
		})
	}
}
