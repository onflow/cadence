/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

//go:generate go run golang.org/x/tools/cmd/stringer -type=ComputationKind -trimprefix=ComputationKind

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
	// stdlibs computation kinds
	//
	ComputationKindSTDLIBPanic
	ComputationKindSTDLIBAssert
	ComputationKindSTDLIBUnsafeRandom
	_
	_
	_
	_
	_
	// RLP
	ComputationKindSTDLIBRLPDecodeString
	ComputationKindSTDLIBRLPDecodeList
)
