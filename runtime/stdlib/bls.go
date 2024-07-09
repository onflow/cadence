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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type BLSPublicKeyAggregator interface {
	PublicKeySignatureVerifier
	BLSPoPVerifier
	// BLSAggregatePublicKeys aggregate multiple BLS public keys into one.
	BLSAggregatePublicKeys(publicKeys []*PublicKey) (*PublicKey, error)
}

func newBLSAggregatePublicKeysFunction(
	gauge common.MemoryGauge,
	aggregator BLSPublicKeyAggregator,
) *interpreter.HostFunctionValue {
	// TODO: Should create a bound-host function here, but interpreter is not available at this point.
	// However, this is not a problem for now, since underlying contract doesn't get moved.
	return interpreter.NewStaticHostFunctionValue(
		gauge,
		BLSTypeAggregatePublicKeysFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			publicKeysValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			inter.ExpectType(
				publicKeysValue,
				sema.PublicKeyArrayType,
				locationRange,
			)

			publicKeys := make([]*PublicKey, 0, publicKeysValue.Count())
			publicKeysValue.Iterate(
				inter,
				func(element interpreter.Value) (resume bool) {
					publicKeyValue, ok := element.(*interpreter.CompositeValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					publicKey, err := NewPublicKeyFromValue(inter, locationRange, publicKeyValue)
					if err != nil {
						panic(err)
					}

					publicKeys = append(publicKeys, publicKey)

					// Continue iteration
					return true
				},
				false,
				locationRange,
			)

			var err error
			var aggregatedPublicKey *PublicKey
			errors.WrapPanic(func() {
				aggregatedPublicKey, err = aggregator.BLSAggregatePublicKeys(publicKeys)
			})

			// If the crypto layer produces an error, we have invalid input, return nil
			if err != nil {
				return interpreter.NilOptionalValue
			}

			aggregatedPublicKeyValue := NewPublicKeyValue(
				inter,
				locationRange,
				aggregatedPublicKey,
			)

			return interpreter.NewSomeValueNonCopying(
				inter,
				aggregatedPublicKeyValue,
			)
		},
	)
}

type BLSSignatureAggregator interface {
	// BLSAggregateSignatures aggregate multiple BLS signatures into one.
	BLSAggregateSignatures(signatures [][]byte) ([]byte, error)
}

func newBLSAggregateSignaturesFunction(
	gauge common.MemoryGauge,
	aggregator BLSSignatureAggregator,
) *interpreter.HostFunctionValue {
	// TODO: Should create a bound-host function here, but interpreter is not available at this point.
	// However, this is not a problem for now, since underlying contract doesn't get moved.
	return interpreter.NewStaticHostFunctionValue(
		gauge,
		BLSTypeAggregateSignaturesFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			signaturesValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			inter.ExpectType(
				signaturesValue,
				sema.ByteArrayArrayType,
				locationRange,
			)

			bytesArray := make([][]byte, 0, signaturesValue.Count())
			signaturesValue.Iterate(
				inter,
				func(element interpreter.Value) (resume bool) {
					signature, ok := element.(*interpreter.ArrayValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					bytes, err := interpreter.ByteArrayValueToByteSlice(inter, signature, invocation.LocationRange)
					if err != nil {
						panic(err)
					}

					bytesArray = append(bytesArray, bytes)

					// Continue iteration
					return true
				},
				false,
				locationRange,
			)

			var err error
			var aggregatedSignature []byte
			errors.WrapPanic(func() {
				aggregatedSignature, err = aggregator.BLSAggregateSignatures(bytesArray)
			})

			// If the crypto layer produces an error, we have invalid input, return nil
			if err != nil {
				return interpreter.NilOptionalValue
			}

			aggregatedSignatureValue := interpreter.ByteSliceToByteArrayValue(inter, aggregatedSignature)

			return interpreter.NewSomeValueNonCopying(
				inter,
				aggregatedSignatureValue,
			)
		},
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
	blsContractFields := map[string]interpreter.Value{
		BLSTypeAggregatePublicKeysFunctionName: newBLSAggregatePublicKeysFunction(gauge, handler),
		BLSTypeAggregateSignaturesFunctionName: newBLSAggregateSignaturesFunction(gauge, handler),
	}

	blsContractValue := interpreter.NewSimpleCompositeValue(
		gauge,
		BLSType.ID(),
		BLSTypeStaticType,
		nil,
		blsContractFields,
		nil,
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
