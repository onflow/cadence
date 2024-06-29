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

package stdlib

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type StandardLibraryHandler interface {
	Logger
	RandomGenerator
	BlockAtHeightProvider
	CurrentBlockProvider
	AccountCreator
	PublicKeyValidator
	PublicKeySignatureVerifier
	BLSPoPVerifier
	BLSPublicKeyAggregator
	BLSSignatureAggregator
	Hasher
	WebAssemblyContractHandler
}

type StandardLibraryOptions struct {
	WebAssemblyEnabled bool
}

var DefaultStandardLibraryOptions = StandardLibraryOptions{
	WebAssemblyEnabled: false,
}

func DefaultStandardLibraryValues(
	handler StandardLibraryHandler,
	options StandardLibraryOptions,
) []StandardLibraryValue {
	var values []StandardLibraryValue

	if options.WebAssemblyEnabled {
		values = append(
			values,
			NewWebAssemblyContract(nil, handler),
		)
	}

	return append(
		values,
		AssertFunction,
		PanicFunction,
		SignatureAlgorithmConstructor,
		RLPContract,
		InclusiveRangeConstructorFunction,
		NewLogFunction(handler),
		NewRevertibleRandomFunction(handler),
		NewGetBlockFunction(handler),
		NewGetCurrentBlockFunction(handler),
		NewGetAccountFunction(handler),
		NewAccountConstructor(handler),
		NewPublicKeyConstructor(handler),
		NewBLSContract(nil, handler),
		NewHashAlgorithmConstructor(handler),
	)
}

func DefaultScriptStandardLibraryValues(
	handler StandardLibraryHandler,
	options StandardLibraryOptions,
) []StandardLibraryValue {
	return append(
		DefaultStandardLibraryValues(handler, options),
		NewGetAuthAccountFunction(handler),
	)
}

func DefaultStandardLibraryTypes(
	options StandardLibraryOptions,
) []StandardLibraryType {
	var types []StandardLibraryType

	if options.WebAssemblyEnabled {
		types = append(
			types,
			WebAssemblyContractType,
		)
	}

	return types
}

type CompositeValueFunctionsHandler func(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	compositeValue *interpreter.CompositeValue,
) *interpreter.FunctionOrderedMap

type CompositeValueFunctionsHandlers map[common.TypeID]CompositeValueFunctionsHandler

func DefaultStandardLibraryCompositeValueFunctionHandlers(
	handler StandardLibraryHandler,
) CompositeValueFunctionsHandlers {
	return CompositeValueFunctionsHandlers{
		sema.PublicKeyType.ID(): func(
			inter *interpreter.Interpreter,
			_ interpreter.LocationRange,
			publicKeyValue *interpreter.CompositeValue,
		) *interpreter.FunctionOrderedMap {
			return PublicKeyFunctions(inter, publicKeyValue, handler)
		},
	}
}
