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
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
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
}

func InterpreterDefaultStandardLibraryValues(handler StandardLibraryHandler) []StandardLibraryValue {
	return []StandardLibraryValue{
		InterpreterAssertFunction,
		InterpreterPanicFunction,
		InterpreterSignatureAlgorithmConstructor,
		InterpreterInclusiveRangeConstructor,
		NewInterpreterLogFunction(handler),
		NewInterpreterRevertibleRandomFunction(handler),
		NewInterpreterGetBlockFunction(handler),
		NewInterpreterGetCurrentBlockFunction(handler),
		NewInterpreterGetAccountFunction(handler),
		NewInterpreterAccountConstructor(handler),
		NewInterpreterPublicKeyConstructor(handler),
		NewInterpreterHashAlgorithmConstructor(handler),
		RLPContract,
		NewBLSContract(nil, handler),
	}
}

func VMDefaultStandardLibraryValues(handler StandardLibraryHandler) []StandardLibraryValue {
	return []StandardLibraryValue{
		VMAssertFunction,
		VMPanicFunction,
		VMSignatureAlgorithmConstructor,
		// TODO: InclusiveRangeConstructor
		NewVMLogFunction(handler),
		NewVMRevertibleRandomFunction(handler),
		NewVMGetBlockFunction(handler),
		NewVMGetCurrentBlockFunction(handler),
		NewVMGetAccountFunction(handler),
		NewVMAccountConstructor(handler),
		NewVMPublicKeyConstructor(handler),
		NewVMHashAlgorithmConstructor(handler),
		RLPContract,
		NewBLSContract(nil, handler),
	}
}

type VMFunction struct {
	BaseType      sema.Type
	FunctionValue *vm.NativeFunctionValue
}

func VMFunctions(handler StandardLibraryHandler) []VMFunction {
	return []VMFunction{
		VMAccountCapabilitiesExistsFunction,
		NewVMAccountCapabilitiesGetFunction(handler, false),
		NewVMAccountCapabilitiesGetFunction(handler, true),
		NewVMAccountCapabilitiesPublishFunction(handler),
		NewVMAccountCapabilitiesUnpublishFunction(handler),

		NewVMAccountKeysAddFunction(handler),
		NewVMAccountKeysGetFunction(handler),
		NewVMAccountKeysRevokeFunction(handler),
		NewVMAccountKeysForEachFunction(handler),

		NewVMAccountInboxPublishFunction(handler),
		NewVMAccountInboxUnpublishFunction(handler),
		NewVMAccountInboxClaimFunction(handler),

		NewVMAccountStorageCapabilitiesGetControllersFunction(handler),
		NewVMAccountStorageCapabilitiesGetControllerFunction(handler),
		NewVMAccountStorageCapabilitiesForEachControllerFunction(handler),
		NewVMAccountStorageCapabilitiesIssueFunction(handler),
		NewVMAccountStorageCapabilitiesIssueWithTypeFunction(handler),

		NewVMAccountAccountCapabilitiesGetControllerFunction(handler),
		NewVMAccountAccountCapabilitiesGetControllersFunction(handler),
		NewVMAccountAccountCapabilitiesForEachControllerFunction(handler),
		NewVMAccountAccountCapabilitiesIssueFunction(handler),
		NewVMAccountAccountCapabilitiesIssueWithTypeFunction(handler),

		VMRLPDecodeStringFunction,
		VMRLPDecodeListFunction,

		NewVMBLSAggregatePublicKeysFunction(handler),
		NewVMBLSAggregateSignaturesFunction(handler),

		NewVMHashAlgorithmHashFunction(handler),
		NewVMHashAlgorithmHashWithTagFunction(handler),

		NewVMPublicKeyVerifySignatureFunction(handler),
		NewVMPublicKeyVerifyPoPFunction(handler),
	}
}

type VMValue struct {
	Name  string
	Value vm.Value
}

func VMValues(handler StandardLibraryHandler) []VMValue {
	return common.Concat(
		VMSignatureAlgorithmCaseValues,
		NewVMHashAlgorithmCaseValues(handler),
	)
}

func InterpreterDefaultScriptStandardLibraryValues(handler StandardLibraryHandler) []StandardLibraryValue {
	return append(
		InterpreterDefaultStandardLibraryValues(handler),
		NewInterpreterGetAuthAccountFunction(handler),
	)
}

func VMDefaultScriptStandardLibraryValues(handler StandardLibraryHandler) []StandardLibraryValue {
	return append(
		VMDefaultStandardLibraryValues(handler),
		NewVMGetAuthAccountFunction(handler),
	)
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
