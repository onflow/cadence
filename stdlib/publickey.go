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
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

const publicKeyConstructorFunctionDocString = `
Constructs a new public key
`

var publicKeyConstructorFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]sema.Parameter{
		{
			Identifier:     sema.PublicKeyTypePublicKeyFieldName,
			TypeAnnotation: sema.ByteArrayTypeAnnotation,
		},
		{
			Identifier:     sema.PublicKeyTypeSignAlgoFieldName,
			TypeAnnotation: sema.SignatureAlgorithmTypeAnnotation,
		},
	},
	sema.PublicKeyTypeAnnotation,
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
		context interpreter.PublicKeyValidationContext,
		locationRange interpreter.LocationRange,
		publicKeyValue *interpreter.CompositeValue,
	) error {
		publicKey, err := NewPublicKeyFromValue(context, locationRange, publicKeyValue)
		if err != nil {
			return err
		}

		errors.WrapPanic(func() {
			err = validator.ValidatePublicKey(publicKey)
		})
		if err != nil {
			err = interpreter.WrappedExternalError(err)
		}
		return err
	}
}

func NewPublicKeyConstructor(
	publicKeyValidator PublicKeyValidator,
) StandardLibraryValue {
	return NewStandardLibraryStaticFunction(
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

			inter := invocation.InvocationContext
			locationRange := invocation.LocationRange

			return NewPublicKeyFromFields(
				inter,
				locationRange,
				publicKey,
				signAlgo,
				publicKeyValidator,
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
) *interpreter.CompositeValue {
	return interpreter.NewPublicKeyValue(
		inter,
		locationRange,
		publicKey,
		signAlgo,
		newPublicKeyValidationHandler(publicKeyValidator),
	)
}

func assumePublicKeyIsValid(_ interpreter.PublicKeyValidationContext, _ interpreter.LocationRange, _ *interpreter.CompositeValue) error {
	return nil
}

func NewPublicKeyValue(
	context interpreter.PublicKeyCreationContext,
	locationRange interpreter.LocationRange,
	publicKey *PublicKey,
) *interpreter.CompositeValue {
	return interpreter.NewPublicKeyValue(
		context,
		locationRange,
		interpreter.ByteSliceToByteArrayValue(
			context,
			publicKey.PublicKey,
		),
		NewSignatureAlgorithmCase(
			interpreter.UInt8Value(publicKey.SignAlgo.RawValue()),
		),
		// public keys converted from "native" (non-interpreter) keys are assumed to be already validated
		assumePublicKeyIsValid,
	)
}

func NewPublicKeyFromValue(
	context interpreter.PublicKeyCreationContext,
	locationRange interpreter.LocationRange,
	publicKey interpreter.MemberAccessibleValue,
) (
	*PublicKey,
	error,
) {
	// publicKey field
	key := publicKey.GetMember(context, locationRange, sema.PublicKeyTypePublicKeyFieldName)

	byteArray, err := interpreter.ByteArrayValueToByteSlice(context, key, locationRange)
	if err != nil {
		return nil, errors.NewUnexpectedError("public key needs to be a byte array. %w", err)
	}

	// sign algo field
	signAlgoField := publicKey.GetMember(context, locationRange, sema.PublicKeyTypeSignAlgoFieldName)
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

	rawValue := signAlgoValue.GetMember(context, locationRange, sema.EnumRawValueFieldName)
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
		SignAlgo:  sema.SignatureAlgorithm(signAlgoRawValue.ToInt(locationRange)),
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
	inter *interpreter.Interpreter,
	publicKeyValue *interpreter.CompositeValue,
	verifier PublicKeySignatureVerifier,
) interpreter.BoundFunctionValue {
	return interpreter.NewBoundHostFunctionValue(
		inter,
		publicKeyValue,
		sema.PublicKeyVerifyFunctionType,
		func(publicKeyValue *interpreter.CompositeValue, invocation interpreter.Invocation) interpreter.Value {
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

			inter := invocation.InvocationContext

			locationRange := invocation.LocationRange

			interpreter.ExpectType(
				inter,
				publicKeyValue,
				sema.PublicKeyType,
				locationRange,
			)

			signature, err := interpreter.ByteArrayValueToByteSlice(inter, signatureValue, locationRange)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to get signature. %w", err))
			}

			signedData, err := interpreter.ByteArrayValueToByteSlice(inter, signedDataValue, locationRange)
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
			errors.WrapPanic(func() {
				valid, err = verifier.VerifySignature(
					signature,
					domainSeparationTag,
					signedData,
					publicKey.PublicKey,
					publicKey.SignAlgo,
					hashAlgorithm,
				)
			})

			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}

			return interpreter.BoolValue(valid)
		},
	)
}

type BLSPoPVerifier interface {
	// BLSVerifyPOP verifies a proof of possession (PoP) for the receiver public key.
	BLSVerifyPOP(publicKey *PublicKey, signature []byte) (bool, error)
}

func newPublicKeyVerifyPoPFunction(
	inter *interpreter.Interpreter,
	publicKeyValue *interpreter.CompositeValue,
	verifier BLSPoPVerifier,
) interpreter.BoundFunctionValue {
	return interpreter.NewBoundHostFunctionValue(
		inter,
		publicKeyValue,
		sema.PublicKeyVerifyPoPFunctionType,
		func(publicKeyValue *interpreter.CompositeValue, invocation interpreter.Invocation) interpreter.Value {
			signatureValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.InvocationContext

			locationRange := invocation.LocationRange

			interpreter.ExpectType(
				inter,
				publicKeyValue,
				sema.PublicKeyType,
				locationRange,
			)

			publicKey, err := NewPublicKeyFromValue(inter, locationRange, publicKeyValue)
			if err != nil {
				panic(err)
			}

			signature, err := interpreter.ByteArrayValueToByteSlice(inter, signatureValue, locationRange)
			if err != nil {
				panic(err)
			}

			var valid bool
			errors.WrapPanic(func() {
				valid, err = verifier.BLSVerifyPOP(publicKey, signature)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}
			return interpreter.BoolValue(valid)
		},
	)
}

type PublicKeyFunctionsHandler interface {
	PublicKeySignatureVerifier
	BLSPoPVerifier
}

func PublicKeyFunctions(
	inter *interpreter.Interpreter,
	publicKeyValue *interpreter.CompositeValue,
	handler PublicKeyFunctionsHandler,
) *interpreter.FunctionOrderedMap {
	functions := orderedmap.New[interpreter.FunctionOrderedMap](2)

	functions.Set(
		sema.PublicKeyTypeVerifyFunctionName,
		newPublicKeyVerifySignatureFunction(inter, publicKeyValue, handler),
	)

	functions.Set(
		sema.PublicKeyTypeVerifyPoPFunctionName,
		newPublicKeyVerifyPoPFunction(inter, publicKeyValue, handler),
	)

	return functions
}
