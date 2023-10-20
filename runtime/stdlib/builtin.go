/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package stdlib

import "github.com/onflow/cadence/runtime/sema"

var UInt8Type = sema.UInt8Type

type StandardLibraryHandler interface {
	Logger
	UnsafeRandomGenerator
	BlockAtHeightProvider
	CurrentBlockProvider
	AccountCreator
	PublicKeyValidator
	PublicKeySignatureVerifier
	BLSPoPVerifier
	BLSPublicKeyAggregator
	BLSSignatureAggregator
	Hasher
}

func DefaultStandardLibraryValues(handler StandardLibraryHandler) []StandardLibraryValue {
	return []StandardLibraryValue{
		AssertFunction,
		PanicFunction,
		SignatureAlgorithmConstructor,
		RLPContract,
		NewLogFunction(handler),
		NewUnsafeRandomFunction(handler),
		NewGetBlockFunction(handler),
		NewGetCurrentBlockFunction(handler),
		NewGetAccountFunction(handler),
		NewAccountConstructor(handler),
		NewPublicKeyConstructor(handler, handler, handler),
		NewBLSContract(nil, handler),
		NewHashAlgorithmConstructor(handler),
	}
}

func DefaultScriptStandardLibraryValues(handler StandardLibraryHandler) []StandardLibraryValue {
	return append(
		DefaultStandardLibraryValues(handler),
		NewGetAuthAccountFunction(handler),
	)
}
