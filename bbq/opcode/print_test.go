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
		byte(Less),
		byte(JumpIfFalse), 0, 14,
		// then return n
		byte(GetLocal), 0, 0,
		byte(ReturnValue),
		// fib(n - 1)
		byte(GetLocal), 0, 0,
		byte(GetConstant), 0, 1,
		byte(Subtract),
		byte(Transfer), 0, 0,
		byte(GetGlobal), 0, 0,
		byte(Invoke), 0, 0,
		// fib(n - 2)
		byte(GetLocal), 0, 0,
		byte(GetConstant), 0, 0,
		byte(Subtract),
		byte(Transfer), 0, 0,
		byte(GetGlobal), 0, 0,
		byte(Invoke), 0, 0,
		// return sum
		byte(Add),
		byte(ReturnValue),
	}

	const expected = `GetLocal localIndex:0
GetConstant constantIndex:0
Less
JumpIfFalse target:14
GetLocal localIndex:0
ReturnValue
GetLocal localIndex:0
GetConstant constantIndex:1
Subtract
Transfer typeIndex:0
GetGlobal globalIndex:0
Invoke typeArgs:[]
GetLocal localIndex:0
GetConstant constantIndex:0
Subtract
Transfer typeIndex:0
GetGlobal globalIndex:0
Invoke typeArgs:[]
Add
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

		"Jump target:258":        {byte(Jump), 1, 2},
		"JumpIfFalse target:258": {byte(JumpIfFalse), 1, 2},
		"JumpIfTrue target:258":  {byte(JumpIfTrue), 1, 2},
		"JumpIfNil target:258":   {byte(JumpIfNil), 1, 2},

		"Transfer typeIndex:258": {byte(Transfer), 1, 2},

		"New kind:CompositeKind(258) typeIndex:772": {byte(New), 1, 2, 3, 4},

		"SimpleCast typeIndex:258":   {byte(SimpleCast), 1, 2, 3},
		"FailableCast typeIndex:258": {byte(FailableCast), 1, 2, 3},
		"ForceCast typeIndex:258":    {byte(ForceCast), 1, 2, 3},

		`Path domain:PathDomainStorage identifierIndex:5`: {byte(Path), 1, 0, 5},

		`InvokeDynamic nameIndex:1 typeArgs:[772, 1286] argCount:1800`: {
			byte(InvokeDynamic), 0, 1, 0, 2, 3, 4, 5, 6, 7, 8,
		},

		"Invoke typeArgs:[772, 1286]": {
			byte(Invoke), 0, 2, 3, 4, 5, 6,
		},

		"NewRef typeIndex:258": {byte(NewRef), 1, 2},
		"Deref":                {byte(Deref)},

		"NewArray typeIndex:258 size:772 isResource:true":      {byte(NewArray), 1, 2, 3, 4, 1},
		"NewDictionary typeIndex:258 size:772 isResource:true": {byte(NewDictionary), 1, 2, 3, 4, 1},

		"NewClosure functionIndex:258": {byte(NewClosure), 1, 2},

		"Unknown":     {byte(Unknown)},
		"Return":      {byte(Return)},
		"ReturnValue": {byte(ReturnValue)},

		"Add":      {byte(Add)},
		"Subtract": {byte(Subtract)},
		"Multiply": {byte(Multiply)},
		"Divide":   {byte(Divide)},
		"Mod":      {byte(Mod)},
		"Negate":   {byte(Negate)},

		"Less":           {byte(Less)},
		"Greater":        {byte(Greater)},
		"LessOrEqual":    {byte(LessOrEqual)},
		"GreaterOrEqual": {byte(GreaterOrEqual)},

		"Equal":    {byte(Equal)},
		"NotEqual": {byte(NotEqual)},

		"Unwrap":                      {byte(Unwrap)},
		"Destroy":                     {byte(Destroy)},
		"True":                        {byte(True)},
		"False":                       {byte(False)},
		"Nil":                         {byte(Nil)},
		"GetField fieldNameIndex:258": {byte(GetField), 1, 2},
		"SetField fieldNameIndex:258": {byte(SetField), 1, 2},
		"SetIndex":                    {byte(SetIndex)},
		"GetIndex":                    {byte(GetIndex)},
		"Drop":                        {byte(Drop)},
		"Dup":                         {byte(Dup)},
		"Not":                         {byte(Not)},

		"BitwiseOr":         {byte(BitwiseOr)},
		"BitwiseAnd":        {byte(BitwiseAnd)},
		"BitwiseXor":        {byte(BitwiseXor)},
		"BitwiseLeftShift":  {byte(BitwiseLeftShift)},
		"BitwiseRightShift": {byte(BitwiseRightShift)},

		"Iterator":        {byte(Iterator)},
		"IteratorHasNext": {byte(IteratorHasNext)},
		"IteratorNext":    {byte(IteratorNext)},

		"EmitEvent typeIndex:258": {byte(EmitEvent), 1, 2},
	}

	// Check if there is any opcode that is not tested

	tested := map[string]struct{}{}
	for expected := range instructions {
		name := strings.SplitN(expected, " ", 2)[0]
		tested[name] = struct{}{}
	}

	for opcode := range OpcodeMax {
		name := opcode.String()
		if !strings.HasPrefix(name, "Opcode(") {
			assert.Contains(t, tested, name, "missing test for opcode %s", name)
		}
	}

	// Run tests

	for expected, code := range instructions {
		t.Run(expected, func(t *testing.T) {

			var ip uint16
			instruction := DecodeInstruction(&ip, code)
			assert.Equal(t, expected, instruction.String())
		})
	}
}
