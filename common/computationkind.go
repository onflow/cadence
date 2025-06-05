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

package common

//go:generate stringer -type=ComputationKind -trimprefix=ComputationKind

// ComputationKind captures kind of computation that would be used for metring computation
type ComputationKind uint

// [1000,2000) is reserved for Cadence interpreter and runtime
const ComputationKindRangeStart = 1000

const (
	ComputationKindUnknown ComputationKind = 0
	// interpreter - base
	ComputationKindStatement ComputationKind = ComputationKindRangeStart + iota
	ComputationKindLoop
	ComputationKindFunctionInvocation
	_
	_
	_
	_
	_
	_
	// interpreter value operations
	ComputationKindCreateCompositeValue
	ComputationKindTransferCompositeValue
	ComputationKindDestroyCompositeValue
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
	_
	ComputationKindCreateArrayValue
	ComputationKindTransferArrayValue
	ComputationKindDestroyArrayValue
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
	_
	ComputationKindCreateDictionaryValue
	ComputationKindTransferDictionaryValue
	ComputationKindDestroyDictionaryValue
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
	_
	_
	_
	_
	ComputationKindEncodeValue
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
	_
	_
	_
	_
	_
	_
	_
	_
	// stdlibs computation kinds
	//
	ComputationKindSTDLIBPanic
	ComputationKindSTDLIBAssert
	ComputationKindSTDLIBRevertibleRandom
	_
	_
	_
	_
	_
	// RLP
	ComputationKindSTDLIBRLPDecodeString
	ComputationKindSTDLIBRLPDecodeList

	// VM Instructions

	// Control flow

	ComputationKindInstructionReturn
	ComputationKindInstructionReturnValue
	ComputationKindInstructionJump
	ComputationKindInstructionJumpIfFalse
	ComputationKindInstructionJumpIfTrue
	ComputationKindInstructionJumpIfNil
	_
	_
	_
	_

	// Number operations

	ComputationKindInstructionAdd
	ComputationKindInstructionSubtract
	ComputationKindInstructionMultiply
	ComputationKindInstructionDivide
	ComputationKindInstructionMod
	ComputationKindInstructionNegate
	_
	_
	_

	// Bitwise operations

	ComputationKindInstructionBitwiseOr
	ComputationKindInstructionBitwiseAnd
	ComputationKindInstructionBitwiseXor
	ComputationKindInstructionBitwiseLeftShift
	ComputationKindInstructionBitwiseRightShift
	_

	// Comparison

	ComputationKindInstructionLess
	ComputationKindInstructionGreater
	ComputationKindInstructionLessOrEqual
	ComputationKindInstructionGreaterOrEqual

	// Equality

	ComputationKindInstructionEqual
	ComputationKindInstructionNotEqual

	// Unary/Binary operators

	ComputationKindInstructionNot
	_
	_
	_
	ComputationKindInstructionUnwrap
	ComputationKindInstructionDestroy
	ComputationKindInstructionTransferAndConvert
	ComputationKindInstructionSimpleCast
	ComputationKindInstructionFailableCast
	ComputationKindInstructionForceCast
	ComputationKindInstructionDeref
	ComputationKindInstructionTransfer
	_
	_
	_
	_
	_

	// Value/Constant loading

	ComputationKindInstructionTrue
	ComputationKindInstructionFalse
	ComputationKindInstructionNil
	ComputationKindInstructionNew
	ComputationKindInstructionNewPath
	ComputationKindInstructionNewArray
	ComputationKindInstructionNewDictionary
	ComputationKindInstructionNewRef
	ComputationKindInstructionNewClosure
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

	ComputationKindInstructionGetConstant
	ComputationKindInstructionGetLocal
	ComputationKindInstructionSetLocal
	ComputationKindInstructionGetUpvalue
	ComputationKindInstructionSetUpvalue
	ComputationKindInstructionGetGlobal
	ComputationKindInstructionSetGlobal
	ComputationKindInstructionGetField
	ComputationKindInstructionRemoveField
	ComputationKindInstructionSetField
	ComputationKindInstructionSetIndex
	ComputationKindInstructionGetIndex
	ComputationKindInstructionRemoveIndex
	_
	_
	_
	_
	_
	_
	_

	// Invocations

	ComputationKindInstructionInvoke
	ComputationKindInstructionInvokeMethodStatic
	ComputationKindInstructionInvokeMethodDynamic
	_
	_
	_
	_
	_
	_
	_

	// Stack operations

	ComputationKindInstructionDrop
	ComputationKindInstructionDup
	_
	_
	_
	_
	_
	_

	// Iterator related

	ComputationKindInstructionIterator
	ComputationKindInstructionIteratorHasNext
	ComputationKindInstructionIteratorNext

	// Other

	ComputationKindInstructionEmitEvent
)
