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

	ElaborationMemoryUsage       = NewConstantMemoryUsage(MemoryKindElaboration)
	ActivationMemoryUsage        = NewConstantMemoryUsage(MemoryKindActivation)
	ActivationEntriesMemoryUsage = NewConstantMemoryUsage(MemoryKindActivationEntries)

	SimpleCompositeBaseMemoryUsage = NewConstantMemoryUsage(MemoryKindSimpleCompositeBase)
	AtreeMapElementOverhead        = NewConstantMemoryUsage(MemoryKindAtreeMapElementOverhead)
	AtreeArrayElementOverhead      = NewConstantMemoryUsage(MemoryKindAtreeArrayElementOverhead)

	VariableSizedSemaTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindVariableSizedSemaType)
	ConstantSizedSemaTypeMemoryUsage = NewConstantMemoryUsage(MemoryKindConstantSizedSemaType)
	DictionarySemaTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindDictionarySemaType)
	OptionalSemaTypeMemoryUsage      = NewConstantMemoryUsage(MemoryKindOptionalSemaType)
	RestrictedSemaTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindRestrictedSemaType)
	ReferenceSemaTypeMemoryUsage     = NewConstantMemoryUsage(MemoryKindReferenceSemaType)
	CapabilitySemaTypeMemoryUsage    = NewConstantMemoryUsage(MemoryKindCapabilitySemaType)

	OrderedMapMemoryUsage = NewConstantMemoryUsage(MemoryKindOrderedMap)
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

func atreeNodes(count uint64, elementSize uint) (leafNodeCount uint64, branchNodeCount uint64) {
	if elementSize != 0 {
		// If we know how large each element is, we can compute the number of
		// atree leaf nodes using this formula:
		// size * elementSize / default_slab_size
		leafNodeCount = uint64(math.Ceil(float64(count) * float64(elementSize) / 1024))
	} else {
		// If we don't know how large each element is, we can overestimate
		// the number of atree leaf nodes this way, since every non-root leaf node
		// will always contain at least two elements.
		leafNodeCount = uint64(math.Ceil(float64(count) / 2))
	}
	if leafNodeCount < 1 {
		leafNodeCount = 1 // there will always be at least one data slab
	}

	branchNodeCount = 0
	if leafNodeCount >= 2 {
		const n = 20 // n-ary tree
		// Compute atree height from number of leaf nodes and n-ary
		height := 1 + int(math.Ceil(math.Log(float64(leafNodeCount))/math.Log(float64(n))))

		// Compute number of branch nodes from leaf node count and height
		for i := 1; i < height; i++ {
			branchNodeCount += uint64(math.Ceil(float64(leafNodeCount) / math.Pow(float64(n), float64(i))))
		}
	}
	return
}

func newAtreeMemoryUsage(count uint64, elementSize uint, array bool) (MemoryUsage, MemoryUsage) {
	newLeafNodes, newBranchNodes := atreeNodes(count, elementSize)
	if array {
		return MemoryUsage{
				Kind:   MemoryKindAtreeArrayDataSlab,
				Amount: newLeafNodes,
			}, MemoryUsage{
				Kind:   MemoryKindAtreeArrayMetaDataSlab,
				Amount: newBranchNodes,
			}
	} else {
		return MemoryUsage{
				Kind:   MemoryKindAtreeMapDataSlab,
				Amount: newLeafNodes,
			}, MemoryUsage{
				Kind:   MemoryKindAtreeMapMetaDataSlab,
				Amount: newBranchNodes,
			}
	}
}

func NewCadenceArrayMemoryUsages(length int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindCadenceArrayBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindCadenceArrayLength,
			Amount: uint64(length),
		}
}

func AdditionalAtreeMemoryUsage(originalCount uint64, elementSize uint, array bool) (MemoryUsage, MemoryUsage) {
	originalLeafNodes, originalBranchNodes := atreeNodes(originalCount, elementSize)
	newLeafNodes, newBranchNodes := atreeNodes(originalCount+1, elementSize)
	if array {
		return MemoryUsage{
				Kind:   MemoryKindAtreeArrayDataSlab,
				Amount: newLeafNodes - originalLeafNodes,
			}, MemoryUsage{
				Kind:   MemoryKindAtreeArrayMetaDataSlab,
				Amount: newBranchNodes - originalBranchNodes,
			}
	} else {
		return MemoryUsage{
				Kind:   MemoryKindAtreeMapDataSlab,
				Amount: newLeafNodes - originalLeafNodes,
			}, MemoryUsage{
				Kind:   MemoryKindAtreeMapMetaDataSlab,
				Amount: newBranchNodes - originalBranchNodes,
			}
	}
}

func NewArrayMemoryUsages(count uint64, elementSize uint) (MemoryUsage, MemoryUsage, MemoryUsage, MemoryUsage) {
	leaves, branches := newAtreeMemoryUsage(count, elementSize, true)
	return MemoryUsage{
			Kind:   MemoryKindArrayBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindAtreeArrayElementOverhead,
			Amount: count,
		}, leaves, branches
}

func NewDictionaryMemoryUsages(count uint64, elementSize uint) (MemoryUsage, MemoryUsage, MemoryUsage, MemoryUsage) {
	leaves, branches := newAtreeMemoryUsage(count, elementSize, false)
	return MemoryUsage{
			Kind:   MemoryKindDictionaryBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindAtreeMapElementOverhead,
			Amount: count,
		}, leaves, branches
}

func NewCadenceDictionaryMemoryUsages(length int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindCadenceDictionaryBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindCadenceDictionarySize,
			Amount: uint64(length),
		}
}

func NewCompositeMemoryUsages(count uint64, elementSize uint) (MemoryUsage, MemoryUsage, MemoryUsage, MemoryUsage) {
	leaves, branches := newAtreeMemoryUsage(count, elementSize, false)
	return MemoryUsage{
			Kind:   MemoryKindCompositeBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindAtreeMapElementOverhead,
			Amount: count,
		}, leaves, branches
}

func NewAtreeMapPreAllocatedElementsMemoryUsage(count uint64, elementSize uint) MemoryUsage {
	leafNodesCount, _ := atreeNodes(count, elementSize)

	const preAllocatedElementCountPerLeafNode uint64 = 32

	var amount uint64 = 0
	if count < preAllocatedElementCountPerLeafNode*leafNodesCount {
		amount = preAllocatedElementCountPerLeafNode*leafNodesCount - count
	}

	return MemoryUsage{
		Kind:   MemoryKindAtreeMapPreAllocatedElement,
		Amount: amount,
	}
}

func NewSimpleCompositeMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindSimpleComposite,
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

func NewCadenceStructMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindCadenceStructBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindCadenceStructSize,
			Amount: uint64(fields),
		}
}

func NewCadenceResourceMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindCadenceResourceBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindCadenceResourceSize,
			Amount: uint64(fields),
		}
}

func NewCadenceEventMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindCadenceEventBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindCadenceEventSize,
			Amount: uint64(fields),
		}
}

func NewCadenceContractMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindCadenceContractBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindCadenceContractSize,
			Amount: uint64(fields),
		}
}

func NewCadenceEnumMemoryUsages(fields int) (MemoryUsage, MemoryUsage) {
	return MemoryUsage{
			Kind:   MemoryKindCadenceEnumBase,
			Amount: 1,
		}, MemoryUsage{
			Kind:   MemoryKindCadenceEnumSize,
			Amount: uint64(fields),
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

func NewOrderedMapMemoryUsages(size uint64) (MemoryUsage, MemoryUsage, MemoryUsage) {
	return OrderedMapMemoryUsage,
		MemoryUsage{
			Kind:   MemoryKindOrderedMapEntryList,
			Amount: size,
		},
		MemoryUsage{
			Kind:   MemoryKindOrderedMapEntry,
			Amount: size,
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
