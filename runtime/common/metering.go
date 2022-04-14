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

package common

import (
	"math/big"
	"unsafe"
)

type MemoryUsage struct {
	Kind   MemoryKind
	Amount uint64
}

type MemoryGauge interface {
	MeterMemory(usage MemoryUsage) error
}

var (
	ProgramMemoryUsage        = NewConstantMemoryUsage(MemoryKindProgram)
	IdentifierMemoryUsage     = NewConstantMemoryUsage(MemoryKindIdentifier)
	ArgumentMemoryUsage       = NewConstantMemoryUsage(MemoryKindArgument)
	BlockMemoryUsage          = NewConstantMemoryUsage(MemoryKindBlock)
	FunctionBlockMemoryUsage  = NewConstantMemoryUsage(MemoryKindFunctionBlock)
	ParameterMemoryUsage      = NewConstantMemoryUsage(MemoryKindParameter)
	ParameterListMemoryUsage  = NewConstantMemoryUsage(MemoryKindParameterList)
	TransferMemoryUsage       = NewConstantMemoryUsage(MemoryKindTransfer)
	TypeAnnotationMemoryUsage = NewConstantMemoryUsage(MemoryKindTypeAnnotation)

	FunctionDeclarationMemoryUsage        = NewConstantMemoryUsage(MemoryKindFunctionDeclaration)
	CompositeDeclarationMemoryUsage       = NewConstantMemoryUsage(MemoryKindCompositeDeclaration)
	InterfaceDeclarationMemoryUsage       = NewConstantMemoryUsage(MemoryKindInterfaceDeclaration)
	ImportDeclarationMemoryUsage          = NewConstantMemoryUsage(MemoryKindImportDeclaration)
	TransactionDeclarationMemoryUsage     = NewConstantMemoryUsage(MemoryKindTransactionDeclaration)
	FieldDeclarationMemoryUsage           = NewConstantMemoryUsage(MemoryKindFieldDeclaration)
	EnumCaseDeclarationMemoryUsage        = NewConstantMemoryUsage(MemoryKindEnumCaseDeclaration)
	VariableDeclarationMemoryUsage        = NewConstantMemoryUsage(MemoryKindVariableDeclaration)
	SpecialFunctionDeclarationMemoryUsage = NewConstantMemoryUsage(MemoryKindSpecialFunctionDeclaration)
	PragmaDeclarationMemoryUsage          = NewConstantMemoryUsage(MemoryKindPragmaDeclaration)

	AssignmentStatementMemoryUsage = NewConstantMemoryUsage(MemoryKindAssignmentStatement)
	BreakStatementMemoryUsage      = NewConstantMemoryUsage(MemoryKindBreakStatement)
	ContinueStatementMemoryUsage   = NewConstantMemoryUsage(MemoryKindContinueStatement)
	EmitStatementMemoryUsage       = NewConstantMemoryUsage(MemoryKindEmitStatement)
	ExpressionStatementMemoryUsage = NewConstantMemoryUsage(MemoryKindExpressionStatement)
	ForStatementMemoryUsage        = NewConstantMemoryUsage(MemoryKindForStatement)
	IfStatementMemoryUsage         = NewConstantMemoryUsage(MemoryKindIfStatement)
	ReturnStatementMemoryUsage     = NewConstantMemoryUsage(MemoryKindReturnStatement)
	SwapStatementMemoryUsage       = NewConstantMemoryUsage(MemoryKindSwapStatement)
	SwitchStatementMemoryUsage     = NewConstantMemoryUsage(MemoryKindSwitchStatement)
	WhileStatementMemoryUsage      = NewConstantMemoryUsage(MemoryKindWhileStatement)

	BooleanExpressionMemoryUsage     = NewConstantMemoryUsage(MemoryKindBooleanExpression)
	NilExpressionMemoryUsage         = NewConstantMemoryUsage(MemoryKindNilExpression)
	StringExpressionMemoryUsage      = NewConstantMemoryUsage(MemoryKindStringExpression)
	IntegerExpressionMemoryUsage     = NewConstantMemoryUsage(MemoryKindIntegerExpression)
	FixedPointExpressionMemoryUsage  = NewConstantMemoryUsage(MemoryKindFixedPointExpression)
	IdentifierExpressionMemoryUsage  = NewConstantMemoryUsage(MemoryKindIdentifierExpression)
	InvocationExpressionMemoryUsage  = NewConstantMemoryUsage(MemoryKindInvocationExpression)
	MemberExpressionMemoryUsage      = NewConstantMemoryUsage(MemoryKindMemberExpression)
	IndexExpressionMemoryUsage       = NewConstantMemoryUsage(MemoryKindIndexExpression)
	ConditionalExpressionMemoryUsage = NewConstantMemoryUsage(MemoryKindConditionalExpression)
	UnaryExpressionMemoryUsage       = NewConstantMemoryUsage(MemoryKindUnaryExpression)
	BinaryExpressionMemoryUsage      = NewConstantMemoryUsage(MemoryKindBinaryExpression)
	FunctionExpressionMemoryUsage    = NewConstantMemoryUsage(MemoryKindFunctionExpression)
	CastingExpressionMemoryUsage     = NewConstantMemoryUsage(MemoryKindCastingExpression)
	CreateExpressionMemoryUsage      = NewConstantMemoryUsage(MemoryKindCreateExpression)
	DestroyExpressionMemoryUsage     = NewConstantMemoryUsage(MemoryKindDestroyExpression)
	ReferenceExpressionMemoryUsage   = NewConstantMemoryUsage(MemoryKindReferenceExpression)
	ForceExpressionMemoryUsage       = NewConstantMemoryUsage(MemoryKindForceExpression)
	PathExpressionMemoryUsage        = NewConstantMemoryUsage(MemoryKindPathExpression)

	ConstantSizedTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindConstantSizedType)
	DictionaryTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindDictionaryType)
	FunctionTypeMemoryUsage      = NewConstantMemoryUsage(MemoryKindFunctionType)
	InstantiationTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindInstantiationType)
	NominalTypeMemoryUsage       = NewConstantMemoryUsage(MemoryKindNominalType)
	OptionalTypeMemoryUsage      = NewConstantMemoryUsage(MemoryKindOptionalType)
	ReferenceTypeMemoryUsage     = NewConstantMemoryUsage(MemoryKindReferenceType)
	RestrictedTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindRestrictedType)
	VariableSizedTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindVariableSizedType)
)

func UseMemory(gauge MemoryGauge, usage MemoryUsage) {
	if gauge == nil {
		return
	}

	err := gauge.MeterMemory(usage)
	if err != nil {
		panic(err)
	}
}

func NewConstantMemoryUsage(kind MemoryKind) MemoryUsage {
	return MemoryUsage{
		Kind:   kind,
		Amount: 1,
	}
}

func NewStringMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindString,
		Amount: uint64(length) + 1, // +1 to account for empty strings
	}
}

func NewRawStringMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindRawString,
		Amount: uint64(length) + 1, // +1 to account for empty strings
	}
}

func NewCadenceStringMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceString,
		Amount: uint64(length) + 1, // +1 to account for empty strings
	}
}

func NewCadenceCharacterMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceCharacter,
		Amount: uint64(length),
	}
}

func NewCadenceIntMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceInt,
		Amount: uint64(bytes),
	}
}

func NewBytesMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindBytes,
		Amount: uint64(length) + 1, // +1 to account for empty arrays
	}
}

func NewBigIntMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindBigInt,
		Amount: uint64(bytes),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const bigIntWordSize = int(unsafe.Sizeof(big.Word(0)))

func BigIntByteLength(v *big.Int) int {
	// NOTE: big.Int.Bits() actually returns bytes:
	// []big.Word, where big.Word = uint
	return len(v.Bits()) * bigIntWordSize
}

func NewPlusBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		) + bigIntWordSize,
	)
}

func NewMinusBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewMulBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		BigIntByteLength(a) +
			BigIntByteLength(b),
	)
}

func NewTypeMemoryUsage(staticTypeAsString string) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindTypeValue,
		Amount: uint64(len(staticTypeAsString)),
	}
}

func NewCharacterMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCharacter,
		Amount: uint64(length),
	}
}

func NewModBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewDivBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewBitwiseOrBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewBitwiseXorBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewBitwiseAndBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewBitwiseLeftShiftBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		BigIntByteLength(a) +
			BigIntByteLength(b),
	)
}

func NewBitwiseRightShiftBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewNumberMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindNumber,
		Amount: uint64(bytes),
	}
}

func NewCommentTokenMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindTokenComment,
		Amount: uint64(length),
	}
}

func NewIdentifierTokenMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindTokenIdentifier,
		Amount: uint64(length),
	}
}

func NewNumericLiteralTokenMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindTokenNumericLiteral,
		Amount: uint64(length),
	}
}

func NewSyntaxTokenMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindTokenSyntax,
		Amount: uint64(length),
	}
}

func NewArrayExpressionMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind: MemoryKindArrayExpression,
		// +1 to account for empty arrays
		Amount: uint64(length) + 1,
	}
}

func NewDictionaryExpressionMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind: MemoryKindDictionaryExpression,
		// +1 to account for empty dictionaries
		Amount: uint64(length) + 1,
	}
}
func NewMembersMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind: MemoryKindMembers,
		// +1 to account for empty members
		Amount: uint64(length) + 1,
	}
}

func NewCadenceNumberMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceNumber,
		Amount: uint64(bytes),
	}
}

func NewCadenceBigIntMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCadenceNumber,
		Amount: uint64(bytes),
	}
}

// UseConstantMemory uses a pre-determined amount of memory
//
func UseConstantMemory(memoryGauge MemoryGauge, kind MemoryKind) {
	UseMemory(memoryGauge, MemoryUsage{
		Kind:   kind,
		Amount: 1,
	})
}
