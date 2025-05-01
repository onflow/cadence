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

	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/interpreter"
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
		byte(Invoke), 0, 0, 0, 0,
		// fib(n - 2)
		byte(GetLocal), 0, 0,
		byte(GetConstant), 0, 0,
		byte(Subtract),
		byte(Transfer), 0, 0,
		byte(GetGlobal), 0, 0,
		byte(Invoke), 0, 0, 0, 0,
		// return sum
		byte(Add),
		byte(ReturnValue),
	}

	const expected = `  0 |    GetLocal | local:0
  1 | GetConstant | constant:0
  2 |        Less |
  3 | JumpIfFalse | target:14
  4 |    GetLocal | local:0
  5 | ReturnValue |
  6 |    GetLocal | local:0
  7 | GetConstant | constant:1
  8 |    Subtract |
  9 |    Transfer | type:0
 10 |   GetGlobal | global:0
 11 |      Invoke | typeArgs:[] argCount:0
 12 |    GetLocal | local:0
 13 | GetConstant | constant:0
 14 |    Subtract |
 15 |    Transfer | type:0
 16 |   GetGlobal | global:0
 17 |      Invoke | typeArgs:[] argCount:0
 18 |         Add |
 19 | ReturnValue |

`

	var builder strings.Builder
	const resolve = false
	const colorize = false
	err := PrintBytecode(&builder, code, resolve, nil, nil, nil, colorize)
	require.NoError(t, err)

	assert.Equal(t, expected, builder.String())
}

func TestPrintResolved(t *testing.T) {
	t.Parallel()

	instructions := []Instruction{
		InstructionGetConstant{Constant: 0},
		InstructionGetConstant{Constant: 1},

		InstructionEmitEvent{Type: 0, ArgCount: 1},
		InstructionEmitEvent{Type: 1, ArgCount: 2},

		InstructionNewClosure{
			Function: 0,
			Upvalues: nil,
		},
		InstructionNewClosure{
			Function: 1,
			Upvalues: nil,
		},
	}

	const expected = ` 0 | GetConstant | constant:"foo"
 1 | GetConstant | constant:1(Int)
 2 |   EmitEvent | type:"Int" argCount:1
 3 |   EmitEvent | type:"[String]" argCount:2
 4 |  NewClosure | function:bar upvalues:[]
 5 |  NewClosure | function:baz upvalues:[]

`

	var builder strings.Builder
	const resolve = true
	const colorize = false
	err := PrintInstructions(
		&builder,
		instructions,
		resolve,
		[]constant.Constant{
			{
				Data: []byte("foo"),
				Kind: constant.String,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
		},
		[]interpreter.StaticType{
			interpreter.PrimitiveStaticTypeInt,
			interpreter.NewVariableSizedStaticType(
				nil,
				interpreter.PrimitiveStaticTypeString,
			),
		},
		[]string{
			"bar",
			"baz",
		},
		colorize,
	)
	require.NoError(t, err)

	assert.Equal(t, expected, builder.String())
}

func TestPrintInstruction(t *testing.T) {
	t.Parallel()

	instructions := map[string][]byte{
		"GetConstant constant:258": {byte(GetConstant), 1, 2},
		"GetLocal local:258":       {byte(GetLocal), 1, 2},
		"SetLocal local:258":       {byte(SetLocal), 1, 2},
		"GetUpvalue upvalue:258":   {byte(GetUpvalue), 1, 2},
		"SetUpvalue upvalue:258":   {byte(SetUpvalue), 1, 2},
		"GetGlobal global:258":     {byte(GetGlobal), 1, 2},
		"SetGlobal global:258":     {byte(SetGlobal), 1, 2},

		"Jump target:258":        {byte(Jump), 1, 2},
		"JumpIfFalse target:258": {byte(JumpIfFalse), 1, 2},
		"JumpIfTrue target:258":  {byte(JumpIfTrue), 1, 2},
		"JumpIfNil target:258":   {byte(JumpIfNil), 1, 2},

		"Transfer type:258": {byte(Transfer), 1, 2},

		"New kind:CompositeKind(258) type:772": {byte(New), 1, 2, 3, 4},

		"SimpleCast type:258":   {byte(SimpleCast), 1, 2, 3},
		"FailableCast type:258": {byte(FailableCast), 1, 2, 3},
		"ForceCast type:258":    {byte(ForceCast), 1, 2, 3},

		`NewPath domain:PathDomainStorage identifier:5`: {byte(NewPath), 1, 0, 5},

		"Invoke typeArgs:[772, 1286] argCount:1": {
			byte(Invoke), 0, 2, 3, 4, 5, 6, 0, 1,
		},
		"InvokeMethodStatic typeArgs:[772, 1286] argCount:1": {
			byte(InvokeMethodStatic), 0, 2, 3, 4, 5, 6, 0, 1,
		},
		`InvokeMethodDynamic name:1 typeArgs:[772, 1286] argCount:1800`: {
			byte(InvokeMethodDynamic), 0, 1, 0, 2, 3, 4, 5, 6, 7, 8,
		},

		"NewRef type:258 isImplicit:true": {byte(NewRef), 1, 2, 1},
		"Deref":                           {byte(Deref)},

		"NewArray type:258 size:772 isResource:true":      {byte(NewArray), 1, 2, 3, 4, 1},
		"NewDictionary type:258 size:772 isResource:true": {byte(NewDictionary), 1, 2, 3, 4, 1},

		"NewClosure function:258 upvalues:[targetIndex:772 isLocal:false, targetIndex:1543 isLocal:true]": {
			byte(NewClosure), 1, 2, 0, 2, 3, 4, 0, 6, 7, 1,
		},

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

		"Unwrap":                 {byte(Unwrap)},
		"Destroy":                {byte(Destroy)},
		"True":                   {byte(True)},
		"False":                  {byte(False)},
		"Nil":                    {byte(Nil)},
		"GetField fieldName:258": {byte(GetField), 1, 2},
		"SetField fieldName:258": {byte(SetField), 1, 2},
		"SetIndex":               {byte(SetIndex)},
		"GetIndex":               {byte(GetIndex)},
		"Drop":                   {byte(Drop)},
		"Dup":                    {byte(Dup)},
		"Not":                    {byte(Not)},

		"BitwiseOr":         {byte(BitwiseOr)},
		"BitwiseAnd":        {byte(BitwiseAnd)},
		"BitwiseXor":        {byte(BitwiseXor)},
		"BitwiseLeftShift":  {byte(BitwiseLeftShift)},
		"BitwiseRightShift": {byte(BitwiseRightShift)},

		"Iterator":        {byte(Iterator)},
		"IteratorHasNext": {byte(IteratorHasNext)},
		"IteratorNext":    {byte(IteratorNext)},

		"EmitEvent type:258 argCount:772": {byte(EmitEvent), 1, 2, 3, 4},
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
