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

const publicKeyConstructorFunctionDocString = `
Constructs a new public key
`

var publicKeyConstructorFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]*sema.Parameter{
		{
			Identifier:     sema.PublicKeyPublicKeyField,
			TypeAnnotation: sema.NewTypeAnnotation(sema.ByteArrayType),
		},
		{
			Identifier:     sema.PublicKeySignAlgoField,
			TypeAnnotation: sema.NewTypeAnnotation(sema.SignatureAlgorithmType),
		},
	},
	sema.NewTypeAnnotation(sema.PublicKeyType),
)

type PublicKey struct {
	PublicKey []byte
	SignAlgo  sema.SignatureAlgorithm
}

type PublicKeyValidator interface {
	// ValidatePublicKey verifies the validity of a public key.
	ValidatePublicKey(key *PublicKey) error
}

func newPublicKeyValidationHandler(validator PublicKeyValidator) interpreter.PublicKeyValidationHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		publicKeyValue *interpreter.CompositeValue,
	) error {
		publicKey, err := NewPublicKeyFromValue(inter, locationRange, publicKeyValue)
		if err != nil {
			return err
		}

		wrapPanic(func() {
			err = validator.ValidatePublicKey(publicKey)
		})
		return err
	}
}

func NewPublicKeyConstructor(
	publicKeyValidator PublicKeyValidator,
	publicKeySignatureVerifier PublicKeySignatureVerifier,
	blsPoPVerifier BLSPoPVerifier,
) StandardLibraryValue {
	return NewStandardLibraryFunction(
		sema.PublicKeyTypeName,
		publicKeyConstructorFunctionType,
		publicKeyConstructorFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {

			publicKey, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			signAlgo, ok := invocation.Arguments[1].(*interpreter.SimpleCompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			return NewPublicKeyFromFields(
				inter,
				locationRange,
				publicKey,
				signAlgo,
				publicKeyValidator,
				publicKeySignatureVerifier,
				blsPoPVerifier,
			)
		},
	)
}

func NewPublicKeyFromFields(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	publicKey *interpreter.ArrayValue,
	signAlgo *interpreter.SimpleCompositeValue,
	publicKeyValidator PublicKeyValidator,
	publicKeySignatureVerifier PublicKeySignatureVerifier,
	blsPoPVerifier BLSPoPVerifier,
) *interpreter.CompositeValue {
	return interpreter.NewPublicKeyValue(
		inter,
		locationRange,
		publicKey,
		signAlgo,
		newPublicKeyValidationHandler(publicKeyValidator),
		newPublicKeyVerifySignatureFunction(inter, publicKeySignatureVerifier),
		newPublicKeyVerifyPoPFunction(inter, blsPoPVerifier),
	)
}

func assumePublicKeyIsValid(_ *interpreter.Interpreter, _ interpreter.LocationRange, _ *interpreter.CompositeValue) error {
	return nil
}

func NewPublicKeyValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	publicKey *PublicKey,
	publicKeySignatureVerifier PublicKeySignatureVerifier,
	blsPoPVerifier BLSPoPVerifier,
) *interpreter.CompositeValue {
	return interpreter.NewPublicKeyValue(
		inter,
		locationRange,
		interpreter.ByteSliceToByteArrayValue(
			inter,
			publicKey.PublicKey,
		),
		NewSignatureAlgorithmCase(
			interpreter.UInt8Value(publicKey.SignAlgo.RawValue()),
		),
		// public keys converted from "native" (non-interpreter) keys are assumed to be already validated
		assumePublicKeyIsValid,
		newPublicKeyVerifySignatureFunction(inter, publicKeySignatureVerifier),
		newPublicKeyVerifyPoPFunction(inter, blsPoPVerifier),
	)
}

func NewPublicKeyFromValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	publicKey interpreter.MemberAccessibleValue,
) (
	*PublicKey,
	error,
) {
	// publicKey field
	key := publicKey.GetMember(inter, locationRange, sema.PublicKeyPublicKeyField)

	byteArray, err := interpreter.ByteArrayValueToByteSlice(inter, key)
	if err != nil {
		return nil, errors.NewUnexpectedError("public key needs to be a byte array. %w", err)
	}

	// sign algo field
	signAlgoField := publicKey.GetMember(inter, locationRange, sema.PublicKeySignAlgoField)
	if signAlgoField == nil {
		return nil, errors.NewUnexpectedError("sign algorithm is not set")
	}

	signAlgoValue, ok := signAlgoField.(*interpreter.SimpleCompositeValue)
	if !ok {
		return nil, errors.NewUnexpectedError(
			"sign algorithm does not belong to type: %s",
			sema.SignatureAlgorithmType.QualifiedString(),
		)
	}

	rawValue := signAlgoValue.GetMember(inter, locationRange, sema.EnumRawValueFieldName)
	if rawValue == nil {
		return nil, errors.NewDefaultUserError("sign algorithm raw value is not set")
	}

	signAlgoRawValue, ok := rawValue.(interpreter.UInt8Value)
	if !ok {
		return nil, errors.NewUnexpectedError(
			"sign algorithm raw-value does not belong to type: %s",
			sema.UInt8Type.QualifiedString(),
		)
	}

	return &PublicKey{
		PublicKey: byteArray,
		SignAlgo:  sema.SignatureAlgorithm(signAlgoRawValue.ToInt()),
	}, nil
}

type PublicKeySignatureVerifier interface {
	// VerifySignature returns true if the given signature was produced by signing the given tag + data
	// using the given public key, signature algorithm, and hash algorithm.
	VerifySignature(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm sema.SignatureAlgorithm,
		hashAlgorithm sema.HashAlgorithm,
	) (bool, error)
}

func newPublicKeyVerifySignatureFunction(
	gauge common.MemoryGauge,
	verififier PublicKeySignatureVerifier,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		func(invocation interpreter.Invocation) interpreter.Value {
			signatureValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			signedDataValue, ok := invocation.Arguments[1].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domainSeparationTagValue, ok := invocation.Arguments[2].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			hashAlgorithmValue, ok := invocation.Arguments[3].(*interpreter.SimpleCompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			publicKeyValue := *invocation.Self

			inter := invocation.Interpreter

			locationRange := invocation.LocationRange

			inter.ExpectType(
				publicKeyValue,
				sema.PublicKeyType,
				locationRange,
			)

			signature, err := interpreter.ByteArrayValueToByteSlice(inter, signatureValue)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to get signature. %w", err))
			}

			signedData, err := interpreter.ByteArrayValueToByteSlice(inter, signedDataValue)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to get signed data. %w", err))
			}

			domainSeparationTag := domainSeparationTagValue.Str

			hashAlgorithm := NewHashAlgorithmFromValue(inter, locationRange, hashAlgorithmValue)

			publicKey, err := NewPublicKeyFromValue(inter, locationRange, publicKeyValue)
			if err != nil {
				return interpreter.FalseValue
			}

			var valid bool
			wrapPanic(func() {
				valid, err = verififier.VerifySignature(
					signature,
					domainSeparationTag,
					signedData,
					publicKey.PublicKey,
					publicKey.SignAlgo,
					hashAlgorithm,
				)
			})

			if err != nil {
				panic(err)
			}

			return interpreter.AsBoolValue(valid)
		},
		sema.PublicKeyVerifyFunctionType,
	)
}

type BLSPoPVerifier interface {
	// BLSVerifyPOP verifies a proof of possession (PoP) for the receiver public key.
	BLSVerifyPOP(publicKey *PublicKey, signature []byte) (bool, error)
}

func newPublicKeyVerifyPoPFunction(
	gauge common.MemoryGauge,
	verifier BLSPoPVerifier,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		func(invocation interpreter.Invocation) interpreter.Value {
			signatureValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			publicKeyValue := *invocation.Self

			inter := invocation.Interpreter

			locationRange := invocation.LocationRange

			inter.ExpectType(
				publicKeyValue,
				sema.PublicKeyType,
				locationRange,
			)

			publicKey, err := NewPublicKeyFromValue(inter, locationRange, publicKeyValue)
			if err != nil {
				panic(err)
			}

			signature, err := interpreter.ByteArrayValueToByteSlice(inter, signatureValue)
			if err != nil {
				panic(err)
			}

			var valid bool
			wrapPanic(func() {
				valid, err = verifier.BLSVerifyPOP(publicKey, signature)
			})
			if err != nil {
				panic(err)
			}
			return interpreter.AsBoolValue(valid)
		},
		sema.PublicKeyVerifyPoPFunctionType,
	)
}
