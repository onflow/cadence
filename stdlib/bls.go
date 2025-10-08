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

//go:generate go run ../sema/gen -p stdlib bls.cdc bls.gen.go

import (
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type BLSPublicKeyAggregator interface {
	PublicKeySignatureVerifier
	BLSPoPVerifier
	// BLSAggregatePublicKeys aggregate multiple BLS public keys into one.
	BLSAggregatePublicKeys(publicKeys []*PublicKey) (*PublicKey, error)
}

func NativeBLSAggregatePublicKeysFunction(
	aggregator BLSPublicKeyAggregator,
) interpreter.NativeFunction {
	return interpreter.NativeFunction(
		func(
			context interpreter.NativeFunctionContext,
			_ interpreter.LocationRange,
			_ interpreter.TypeParameterGetter,
			_ interpreter.Value,
			args ...interpreter.Value,
		) interpreter.Value {
			publicKeysValue := interpreter.AssertValueOfType[*interpreter.ArrayValue](args[0])
			return BLSAggregatePublicKeys(
				context,
				publicKeysValue,
				aggregator,
			)
		},
	)
}

func newInterpreterBLSAggregatePublicKeysFunction(
	gauge common.MemoryGauge,
	aggregator BLSPublicKeyAggregator,
) *interpreter.HostFunctionValue {
	// TODO: Should create a bound-host function here, but interpreter is not available at this point.
	// However, this is not a problem for now, since underlying contract doesn't get moved.
	return interpreter.NewStaticHostFunctionValueFromNativeFunction(
		gauge,
		BLSTypeAggregatePublicKeysFunctionType,
		NativeBLSAggregatePublicKeysFunction(aggregator),
	)
}

func NewVMBLSAggregatePublicKeysFunction(
	aggregator BLSPublicKeyAggregator,
) VMFunction {
	return VMFunction{
		BaseType: BLSType,
		FunctionValue: vm.NewNativeFunctionValue(
			BLSTypeAggregatePublicKeysFunctionName,
			BLSTypeAggregatePublicKeysFunctionType,
			NativeBLSAggregatePublicKeysFunction(aggregator),
		),
	}
}

func BLSAggregatePublicKeys(
	context interpreter.InvocationContext,
	publicKeysValue *interpreter.ArrayValue,
	aggregator BLSPublicKeyAggregator,
) interpreter.Value {

	interpreter.ExpectType(
		context,
		publicKeysValue,
		sema.PublicKeyArrayType,
	)

	publicKeys := make([]*PublicKey, 0, publicKeysValue.Count())
	publicKeysValue.Iterate(
		context,
		func(element interpreter.Value) (resume bool) {
			publicKeyValue, ok := element.(*interpreter.CompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			publicKey, err := NewPublicKeyFromValue(context, publicKeyValue)
			if err != nil {
				panic(err)
			}

			publicKeys = append(publicKeys, publicKey)

			// Continue iteration
			return true
		},
		false,
	)

	aggregatedPublicKey, err := aggregator.BLSAggregatePublicKeys(publicKeys)

	// If the crypto layer produces an error, we have invalid input, return nil
	if err != nil {
		return interpreter.NilOptionalValue
	}

	aggregatedPublicKeyValue := NewPublicKeyValue(
		context,
		aggregatedPublicKey,
	)

	return interpreter.NewSomeValueNonCopying(
		context,
		aggregatedPublicKeyValue,
	)
}

type BLSSignatureAggregator interface {
	// BLSAggregateSignatures aggregate multiple BLS signatures into one.
	BLSAggregateSignatures(signatures [][]byte) ([]byte, error)
}

func NativeBLSAggregateSignaturesFunction(
	aggregator BLSSignatureAggregator,
) interpreter.NativeFunction {
	return interpreter.NativeFunction(
		func(
			context interpreter.NativeFunctionContext,
			_ interpreter.LocationRange,
			_ interpreter.TypeParameterGetter,
			_ interpreter.Value,
			args ...interpreter.Value,
		) interpreter.Value {
			signaturesValue := interpreter.AssertValueOfType[*interpreter.ArrayValue](args[0])
			return BLSAggregateSignatures(
				context,
				signaturesValue,
				aggregator,
			)
		},
	)
}

func newInterpreterBLSAggregateSignaturesFunction(
	gauge common.MemoryGauge,
	aggregator BLSSignatureAggregator,
) *interpreter.HostFunctionValue {
	// TODO: Should create a bound-host function here, but interpreter is not available at this point.
	// However, this is not a problem for now, since underlying contract doesn't get moved.
	return interpreter.NewStaticHostFunctionValueFromNativeFunction(
		gauge,
		BLSTypeAggregateSignaturesFunctionType,
		NativeBLSAggregateSignaturesFunction(aggregator),
	)
}

func NewVMBLSAggregateSignaturesFunction(
	aggregator BLSSignatureAggregator,
) VMFunction {
	return VMFunction{
		BaseType: BLSType,
		FunctionValue: vm.NewNativeFunctionValue(
			BLSTypeAggregateSignaturesFunctionName,
			BLSTypeAggregateSignaturesFunctionType,
			NativeBLSAggregateSignaturesFunction(aggregator),
		),
	}
}

func BLSAggregateSignatures(
	context interpreter.InvocationContext,
	signaturesValue *interpreter.ArrayValue,
	aggregator BLSSignatureAggregator,
) interpreter.Value {

	interpreter.ExpectType(
		context,
		signaturesValue,
		sema.ByteArrayArrayType,
	)

	bytesArray := make([][]byte, 0, signaturesValue.Count())
	signaturesValue.Iterate(
		context,
		func(element interpreter.Value) (resume bool) {
			signature, ok := element.(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			bytes, err := interpreter.ByteArrayValueToByteSlice(context, signature)
			if err != nil {
				panic(err)
			}

			bytesArray = append(bytesArray, bytes)

			// Continue iteration
			return true
		},
		false,
	)

	aggregatedSignature, err := aggregator.BLSAggregateSignatures(bytesArray)

	// If the crypto layer produces an error, we have invalid input, return nil
	if err != nil {
		return interpreter.NilOptionalValue
	}

	aggregatedSignatureValue := interpreter.ByteSliceToByteArrayValue(context, aggregatedSignature)

	return interpreter.NewSomeValueNonCopying(
		context,
		aggregatedSignatureValue,
	)
}

type BLSContractHandler interface {
	PublicKeyValidator
	PublicKeySignatureVerifier
	BLSPoPVerifier
	BLSPublicKeyAggregator
	BLSSignatureAggregator
}

var BLSTypeStaticType = interpreter.ConvertSemaToStaticType(nil, BLSType)

func NewBLSContract(
	gauge common.MemoryGauge,
	handler BLSContractHandler,
) StandardLibraryValue {
	methods := map[string]interpreter.FunctionValue{}

	computeLazyStoredMethod := func(name string) interpreter.FunctionValue {
		switch name {
		case BLSTypeAggregatePublicKeysFunctionName:
			return newInterpreterBLSAggregatePublicKeysFunction(gauge, handler)
		case BLSTypeAggregateSignaturesFunctionName:
			return newInterpreterBLSAggregateSignaturesFunction(gauge, handler)
		default:
			return nil
		}
	}

	blsContractMethodsGetter := func(name string, _ interpreter.MemberAccessibleContext) interpreter.FunctionValue {
		method, ok := methods[name]
		if !ok {
			method = computeLazyStoredMethod(name)
			if method != nil {
				methods[name] = method
			}
		}

		return method
	}

	blsContractValue := interpreter.NewSimpleCompositeValue(
		gauge,
		BLSType.ID(),
		BLSTypeStaticType,
		nil,
		nil,
		nil,
		blsContractMethodsGetter,
		nil,
		nil,
	)

	return StandardLibraryValue{
		Name:  BLSTypeName,
		Type:  BLSType,
		Value: blsContractValue,
		Kind:  common.DeclarationKindContract,
	}
}
