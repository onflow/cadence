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

//go:generate go run golang.org/x/tools/cmd/stringer -type=Opcode

type Opcode byte

const (
	Unknown Opcode = iota

	// Control flow

	Return
	ReturnValue
	Jump
	JumpIfFalse
	JumpIfTrue
	JumpIfNil
	_
	_
	_
	_

	// Number operations

	Add
	Subtract
	Multiply
	Divide
	Mod
	Negate
	_
	_
	_

	// Bitwise operations

	BitwiseOr
	BitwiseAnd
	BitwiseXor
	BitwiseLeftShift
	BitwiseRightShift
	_

	// Comparison

	Less
	Greater
	LessOrEqual
	GreaterOrEqual

	// Equality

	Equal
	NotEqual

	// Unary/Binary operators

	Not
	_
	_
	_
	Unwrap
	Destroy
	TransferAndConvert
	SimpleCast
	FailableCast
	ForceCast
	Deref
	Transfer
	_
	_
	_
	_
	_

	// Value/Constant loading

	True
	False
	Void
	Nil
	NewComposite
	NewCompositeAt
	NewPath
	NewArray
	NewDictionary
	NewRef
	NewClosure
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	GetConstant
	GetLocal
	SetLocal
	GetUpvalue
	SetUpvalue
	CloseUpvalue
	GetGlobal
	SetGlobal
	GetField
	RemoveField
	SetField
	SetIndex
	GetIndex
	RemoveIndex
	GetMethod
	_
	_
	_
	_
	_
	_

	// Invocations

	Invoke
	InvokeDynamic
	_
	_
	_
	_
	_
	_
	_
	_

	// Stack operations

	Drop
	Dup
	_
	_
	_
	_
	_
	_

	// Iterator related

	Iterator
	IteratorHasNext
	IteratorNext
	IteratorEnd

	// Other

	EmitEvent
	Loop
	Statement
	TemplateString

	// NOTE: not an actual opcode, must be last item
	OpcodeMax
)

func (i Opcode) IsControlFlow() bool {
	switch i {
	case Return,
		ReturnValue,
		Jump,
		JumpIfFalse,
		JumpIfTrue,
		JumpIfNil,
		Invoke,
		InvokeDynamic:

		return true

	default:
		return false
	}
}
