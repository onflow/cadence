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
	"errors"
	"math"
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
	ValueTokenMemoryUsage  = NewConstantMemoryUsage(MemoryKindValueToken)
	SyntaxTokenMemoryUsage = NewConstantMemoryUsage(MemoryKindSyntaxToken)
	SpaceTokenMemoryUsage  = NewConstantMemoryUsage(MemoryKindSpaceToken)

	ProgramMemoryUsage         = NewConstantMemoryUsage(MemoryKindProgram)
	IdentifierMemoryUsage      = NewConstantMemoryUsage(MemoryKindIdentifier)
	ArgumentMemoryUsage        = NewConstantMemoryUsage(MemoryKindArgument)
	BlockMemoryUsage           = NewConstantMemoryUsage(MemoryKindBlock)
	FunctionBlockMemoryUsage   = NewConstantMemoryUsage(MemoryKindFunctionBlock)
	ParameterMemoryUsage       = NewConstantMemoryUsage(MemoryKindParameter)
	ParameterListMemoryUsage   = NewConstantMemoryUsage(MemoryKindParameterList)
	TransferMemoryUsage        = NewConstantMemoryUsage(MemoryKindTransfer)
	TypeAnnotationMemoryUsage  = NewConstantMemoryUsage(MemoryKindTypeAnnotation)
	DictionaryEntryMemoryUsage = NewConstantMemoryUsage(MemoryKindDictionaryEntry)

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

	PositionMemoryUsage = NewConstantMemoryUsage(MemoryKindPosition)
	RangeMemoryUsage    = NewConstantMemoryUsage(MemoryKindRange)

	ElaborationMemoryUsage = NewConstantMemoryUsage(MemoryKindElaboration)

	// Following are the known memory usage amounts for string representation of interpreter values.
	// Same as `len(format.X)`. However, values are hard-coded to avoid the circular dependency.

	VoidStringMemoryUsage                  = NewRawStringMemoryUsage(len("()"))
	TrueStringMemoryUsage                  = NewRawStringMemoryUsage(len("true"))
	FalseStringMemoryUsage                 = NewRawStringMemoryUsage(len("false"))
	TypeValueStringMemoryUsage             = NewRawStringMemoryUsage(len("Type<>()"))
	NilValueStringMemoryUsage              = NewRawStringMemoryUsage(len("nil"))
	StorageReferenceValueStringMemoryUsage = NewRawStringMemoryUsage(len("StorageReference()"))
	SeenReferenceStringMemoryUsage         = NewRawStringMemoryUsage(3)                   // len(ellipsis)
	AddressValueStringMemoryUsage          = NewRawStringMemoryUsage(AddressLength*2 + 2) // len(bytes-to-hex + prefix)
	HostFunctionValueStringMemoryUsage     = NewRawStringMemoryUsage(len("Function(...)"))

	// Static types string representations

	VariableSizedStaticTypeMemoryUsage = NewRawStringMemoryUsage(2)  // []
	DictionaryStaticTypeMemoryUsage    = NewRawStringMemoryUsage(4)  // {: }
	OptionalStaticTypeMemoryUsage      = NewRawStringMemoryUsage(1)  // ?
	AuthReferenceStaticTypeMemoryUsage = NewRawStringMemoryUsage(5)  // auth&
	ReferenceStaticTypeMemoryUsage     = NewRawStringMemoryUsage(1)  // &
	CapabilityStaticTypeMemoryUsage    = NewRawStringMemoryUsage(12) // Capability<>
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

func NewArrayMemoryUsages(length int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindArrayBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindArrayLength,
			Amount: uint64(length),
		}
}

func NewArrayAdditionalLengthUsage(originalLength, additionalLength int) MemoryUsage {
	var newAmount uint64
	if originalLength <= 1 {
		newAmount = uint64(originalLength + additionalLength)
	} else {
		// size of b+ tree grows logarithmically with the size of the tree
		newAmount = uint64(math.Log2(float64(originalLength)) + float64(additionalLength))
	}
	return MemoryUsage{
		Kind:   MemoryKindArrayLength,
		Amount: newAmount,
	}
}

func NewDictionaryMemoryUsages(length int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindDictionaryBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindDictionarySize,
			Amount: uint64(length),
		}
}

func NewDictionaryAdditionalSizeUsage(originalSize, additionalSize int) MemoryUsage {
	var newAmount uint64
	if originalSize <= 1 {
		newAmount = uint64(originalSize + additionalSize)
	} else {
		// size of b+ tree grows logarithmically with the size of the tree
		newAmount = uint64(math.Log2(float64(originalSize)) + float64(additionalSize))
	}
	return MemoryUsage{
		Kind:   MemoryKindDictionarySize,
		Amount: newAmount,
	}
}

func NewCompositeMemoryUsages(length int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindCompositeBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindCompositeSize,
			Amount: uint64(length),
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

func NewBytesMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindBytes,
		Amount: uint64(length) + 1, // +1 to account for empty arrays
	}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const bigIntWordSize = int(unsafe.Sizeof(big.Word(0)))

var bigIntWordSizeAsBig = big.NewInt(int64(bigIntWordSize))

func BigIntByteLength(v *big.Int) int {
	// NOTE: big.Int.Bits() actually returns a slice of words,
	// []big.Word, where big.Word = uint,
	// NOT a slice of bytes!
	return len(v.Bits()) * bigIntWordSize
}

// big.Int memory metering:
// - |x| is len(x.Bits()), which is the length in words
//

func NewPlusBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// max(|a|, |b|) + 5

	maxWordLength := max(
		len(a.Bits()),
		len(b.Bits()),
	)
	return NewBigIntMemoryUsage(
		(maxWordLength + 5) *
			bigIntWordSize,
	)
}

func NewMinusBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// max(|a|, |b|) + 4

	maxWordLength := max(
		len(a.Bits()),
		len(b.Bits()),
	)
	return NewBigIntMemoryUsage(
		(maxWordLength + 4) *
			bigIntWordSize,
	)
}

func NewMulBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if min(|a|, |b|) <= 40:
	//     |a| + |b| + 4
	// else:
	//     n = min(|a|, |b|)
	//     3 * n + max(6 * n, |a| + |b|) + 8

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())
	minWordLength := min(
		aWordLength,
		bWordLength,
	)
	wordLengthSum := aWordLength + bWordLength

	var resultWordLength int
	if minWordLength <= 40 {
		resultWordLength = wordLengthSum + 4
	} else {
		resultWordLength = 3*minWordLength + max(6*minWordLength, wordLengthSum) + 8
	}
	return NewBigIntMemoryUsage(
		resultWordLength * bigIntWordSize,
	)
}

var bigOne = big.NewInt(1)
var bigOneHundred = big.NewInt(100)

func NewModBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a < b or |b| == 1:
	//     |a| + 4
	// else if |b| < 100:
	//     |a| - |b| + 5
	// else:
	//     recursion_cost = pointer_size + 9 * |b| + floor(|a| / |b|) + 12
	//     recursion_depth = 2 * BitLen(b)
	//     3 * |b| + 4 + recursion_cost * recursion_depth

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	var resultWordLength int
	if a.Cmp(b) < 0 || b.Cmp(bigOne) == 0 {
		resultWordLength = aWordLength + 4
	} else if b.Cmp(bigOneHundred) < 0 {
		resultWordLength = aWordLength - bWordLength + 5
	} else {
		recursionCost := int(unsafe.Sizeof(uintptr(0))) +
			9*bWordLength +
			int(math.Floor(float64(aWordLength)/float64(bWordLength))) + 12
		recursionDepth := 2 * b.BitLen()
		resultWordLength = 3*bWordLength + 4 + recursionCost*recursionDepth
	}
	return NewBigIntMemoryUsage(
		resultWordLength * bigIntWordSize,
	)
}

func NewDivBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewModBigIntMemoryUsage(a, b)
}

func NewBitwiseOrBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a >= 0 and b >= 0:
	//     max(|a|, |b|) + 4
	// else if a <= 0 and b <= 0:
	//     |a| + |b| + min(|a|, |b|) + 13
	// else:
	//     2 * max(|a|, |b|) + 9

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	var resultWordLength int
	if a.Sign() >= 0 && b.Sign() >= 0 {
		resultWordLength = max(aWordLength, bWordLength) + 4
	} else if a.Sign() <= 0 && b.Sign() <= 0 {
		resultWordLength = aWordLength + bWordLength + min(aWordLength, bWordLength) + 13
	} else {
		resultWordLength = 2*max(aWordLength, bWordLength) + 9
	}
	return NewBigIntMemoryUsage(
		resultWordLength * bigIntWordSize,
	)
}

func NewBitwiseXorBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a >= 0 and b >= 0:
	//     max(|a|, |b|) + 4
	// else if a <= 0 and b <= 0:
	//     |a| + |b| + min(|a|, |b|) + 12
	// else:
	//     2 * max(|a|, |b|) + 9

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	var resultWordLength int
	if a.Sign() >= 0 && b.Sign() >= 0 {
		resultWordLength = max(aWordLength, bWordLength) + 4
	} else if a.Sign() <= 0 && b.Sign() <= 0 {
		resultWordLength = aWordLength + bWordLength + min(aWordLength, bWordLength) + 12
	} else {
		resultWordLength = 2*max(aWordLength, bWordLength) + 9
	}
	return NewBigIntMemoryUsage(
		resultWordLength * bigIntWordSize,
	)
}

func NewBitwiseAndBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a >= 0 and b >= 0:
	//     max(|a|, |b|) + 4
	// else if a <= 0 and b <= 0:
	//     |a| + |b| + max(|a|, |b|) + 13
	// else:
	//     2 * max(|a|, |b|) + 8

	aWordLength := len(a.Bits())
	bWordLength := len(b.Bits())

	var resultWordLength int
	if a.Sign() >= 0 && b.Sign() >= 0 {
		resultWordLength = max(aWordLength, bWordLength) + 4
	} else if a.Sign() <= 0 && b.Sign() <= 0 {
		resultWordLength = aWordLength + bWordLength + max(aWordLength, bWordLength) + 13
	} else {
		resultWordLength = 2*max(aWordLength, bWordLength) + 8
	}
	return NewBigIntMemoryUsage(
		resultWordLength * bigIntWordSize,
	)
}

var invalidLeftShift = errors.New("invalid left shift of non-Int64")

func NewBitwiseLeftShiftBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if b == 0:
	//     |a| + 4
	// else:
	//     |a| + b/word_size + 5

	aWordLength := len(a.Bits())

	var resultWordLength int
	if b.Sign() == 0 {
		resultWordLength = aWordLength + 4
	} else {
		// TODO: meter the allocation of the metering itself
		shiftByteLengthBig := new(big.Int).Div(b, bigIntWordSizeAsBig)
		// TODO: handle big int shifts
		if !shiftByteLengthBig.IsInt64() {
			panic(invalidLeftShift)
		}
		shiftByteLength := int(shiftByteLengthBig.Int64())
		resultWordLength = aWordLength + shiftByteLength + 5
	}
	return NewBigIntMemoryUsage(
		resultWordLength * bigIntWordSize,
	)
}

func NewBitwiseRightShiftBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	// if a >= 0:
	//     if b == 0:
	//         |a| + 4
	//     else:
	//         |a| - b/word_size + 4
	// else:
	//     |a| + 4

	aWordLength := len(a.Bits())

	var resultWordLength int
	if a.Sign() >= 0 {
		if b.Sign() == 0 {
			resultWordLength = aWordLength + 4
		} else {
			// TODO: meter the allocation of the metering itself
			shiftByteLengthBig := new(big.Int).Div(b, bigIntWordSizeAsBig)
			// TODO: handle big int shifts
			if !shiftByteLengthBig.IsInt64() {
				panic(invalidLeftShift)
			}
			shiftByteLength := int(shiftByteLengthBig.Int64())
			resultWordLength = aWordLength - shiftByteLength + 4
		}
	} else {
		resultWordLength = aWordLength + 4
	}
	return NewBigIntMemoryUsage(
		resultWordLength * bigIntWordSize,
	)
}

func NewNegateBigIntMemoryUsage(b *big.Int) MemoryUsage {
	// |a| + 4

	return NewBigIntMemoryUsage(
		(len(b.Bits()) + 4) * bigIntWordSize,
	)
}

func NewNumberMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindNumber,
		Amount: uint64(bytes),
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

// UseConstantMemory uses a pre-determined amount of memory
//
func UseConstantMemory(memoryGauge MemoryGauge, kind MemoryKind) {
	UseMemory(memoryGauge, MemoryUsage{
		Kind:   kind,
		Amount: 1,
	})
}
