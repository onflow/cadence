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

package stdlib

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

var blsContractType = func() *sema.CompositeType {
	ty := &sema.CompositeType{
		Identifier: "BLS",
		Kind:       common.CompositeKindContract,
	}

	ty.Members = sema.GetMembersAsMap([]*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			blsAggregatePublicKeysFunctionName,
			blsAggregatePublicKeysFunctionType,
			blsAggregatePublicKeysFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			blsAggregateSignaturesFunctionName,
			blsAggregateSignaturesFunctionType,
			blsAggregateSignaturesFunctionDocString,
		),
	})
	return ty
}()

var blsContractTypeID = blsContractType.ID()
var blsContractStaticType interpreter.StaticType = interpreter.CompositeStaticType{
	QualifiedIdentifier: blsContractType.Identifier,
	TypeID:              blsContractTypeID,
}

const blsAggregateSignaturesFunctionDocString = `
Aggregates multiple BLS signatures into one,
considering the proof of possession as a defense against rogue attacks.

Signatures could be generated from the same or distinct messages,
they could also be the aggregation of other signatures.
The order of the signatures in the slice does not matter since the aggregation is commutative.
No subgroup membership check is performed on the input signatures.
The function returns nil if the array is empty or if decoding one of the signature fails.
`

const blsAggregateSignaturesFunctionName = "aggregateSignatures"

var blsAggregateSignaturesFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "signatures",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.ByteArrayArrayType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.ByteArrayType,
		},
	),
}

const blsAggregatePublicKeysFunctionDocString = `
Aggregates multiple BLS public keys into one.

The order of the public keys in the slice does not matter since the aggregation is commutative.
No subgroup membership check is performed on the input keys.
The function returns nil if the array is empty or any of the input keys is not a BLS key.
`

const blsAggregatePublicKeysFunctionName = "aggregatePublicKeys"

var blsAggregatePublicKeysFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "keys",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.PublicKeyArrayType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.PublicKeyType,
		},
	),
}

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
	return interpreter.NewHostFunctionValue(
		gauge,
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
			publicKeysValue.Iterate(inter, func(element interpreter.Value) (resume bool) {
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
			})

			var err error
			var aggregatedPublicKey *PublicKey
			wrapPanic(func() {
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
				aggregator,
				aggregator,
			)

			return interpreter.NewSomeValueNonCopying(
				inter,
				aggregatedPublicKeyValue,
			)
		},
		blsAggregatePublicKeysFunctionType,
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
	return interpreter.NewHostFunctionValue(
		gauge,
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
			signaturesValue.Iterate(inter, func(element interpreter.Value) (resume bool) {
				signature, ok := element.(*interpreter.ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				bytes, err := interpreter.ByteArrayValueToByteSlice(inter, signature)
				if err != nil {
					panic(err)
				}

				bytesArray = append(bytesArray, bytes)

				// Continue iteration
				return true
			})

			var err error
			var aggregatedSignature []byte
			wrapPanic(func() {
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
		blsAggregateSignaturesFunctionType,
	)
}

type BLSContractHandler interface {
	PublicKeyValidator
	PublicKeySignatureVerifier
	BLSPoPVerifier
	BLSPublicKeyAggregator
	BLSSignatureAggregator
}

func NewBLSContract(
	gauge common.MemoryGauge,
	handler BLSContractHandler,
) StandardLibraryValue {
	var blsContractFields = map[string]interpreter.Value{
		blsAggregatePublicKeysFunctionName: newBLSAggregatePublicKeysFunction(gauge, handler),
		blsAggregateSignaturesFunctionName: newBLSAggregateSignaturesFunction(gauge, handler),
	}

	var blsContractValue = interpreter.NewSimpleCompositeValue(
		nil,
		blsContractType.ID(),
		blsContractStaticType,
		nil,
		blsContractFields,
		nil,
		nil,
		nil,
	)

	return StandardLibraryValue{
		Name:  "BLS",
		Type:  blsContractType,
		Value: blsContractValue,
		Kind:  common.DeclarationKindContract,
	}
}
