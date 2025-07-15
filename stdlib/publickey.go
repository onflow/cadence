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
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/vm"
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
			Identifier:     sema.PublicKeyTypeSignatureAlgorithmFieldName,
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

		return validator.ValidatePublicKey(publicKey)
	}
}

func NewInterpreterPublicKeyConstructor(
	publicKeyValidator PublicKeyValidator,
) StandardLibraryValue {
	return NewInterpreterStandardLibraryStaticFunction(
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

			context := invocation.InvocationContext
			locationRange := invocation.LocationRange

			return NewPublicKeyFromFields(
				context,
				locationRange,
				publicKey,
				signAlgo,
				publicKeyValidator,
			)
		},
	)
}

func NewVMPublicKeyConstructor(
	publicKeyValidator PublicKeyValidator,
) StandardLibraryValue {
	return NewVMStandardLibraryStaticFunction(
		sema.PublicKeyTypeName,
		publicKeyConstructorFunctionType,
		publicKeyConstructorFunctionDocString,
		func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {

			publicKey, ok := args[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			signAlgo, ok := args[1].(*interpreter.SimpleCompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return NewPublicKeyFromFields(
				context,
				interpreter.EmptyLocationRange,
				publicKey,
				signAlgo,
				publicKeyValidator,
			)
		},
	)
}

func NewPublicKeyFromFields(
	context interpreter.PublicKeyCreationContext,
	locationRange interpreter.LocationRange,
	publicKey *interpreter.ArrayValue,
	signAlgo *interpreter.SimpleCompositeValue,
	publicKeyValidator PublicKeyValidator,
) *interpreter.CompositeValue {
	return interpreter.NewPublicKeyValue(
		context,
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
	signAlgoField := publicKey.GetMember(context, locationRange, sema.PublicKeyTypeSignatureAlgorithmFieldName)
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

func newInterpreterPublicKeyVerifySignatureFunction(
	inter *interpreter.Interpreter,
	publicKeyValue *interpreter.CompositeValue,
	verifier PublicKeySignatureVerifier,
) interpreter.BoundFunctionValue {
	return interpreter.NewBoundHostFunctionValue(
		inter,
		publicKeyValue,
		sema.PublicKeyTypeVerifyFunctionType,
		func(publicKeyValue *interpreter.CompositeValue, invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.InvocationContext
			locationRange := invocation.LocationRange

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

			return PublicKeyVerifySignature(
				inter,
				locationRange,
				publicKeyValue,
				signatureValue,
				signedDataValue,
				domainSeparationTagValue,
				hashAlgorithmValue,
				verifier,
			)
		},
	)
}

func NewVMPublicKeyVerifySignatureFunction(verifier PublicKeySignatureVerifier) VMFunction {
	return VMFunction{
		BaseType: sema.PublicKeyType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.PublicKeyTypeVerifyFunctionName,
			sema.PublicKeyTypeVerifyFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = vm.GetReceiverAndArgs(context, args)

				publicKeyValue, ok := receiver.(*interpreter.CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				signatureValue, ok := args[0].(*interpreter.ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				signedDataValue, ok := args[1].(*interpreter.ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				domainSeparationTagValue, ok := args[2].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				hashAlgorithmValue, ok := args[3].(*interpreter.SimpleCompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return PublicKeyVerifySignature(
					context,
					interpreter.EmptyLocationRange,
					publicKeyValue,
					signatureValue,
					signedDataValue,
					domainSeparationTagValue,
					hashAlgorithmValue,
					verifier,
				)
			},
		),
	}
}

func PublicKeyVerifySignature(
	context interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
	publicKeyValue *interpreter.CompositeValue,
	signatureValue *interpreter.ArrayValue,
	signedDataValue *interpreter.ArrayValue,
	domainSeparationTagValue *interpreter.StringValue,
	hashAlgorithmValue *interpreter.SimpleCompositeValue,
	verifier PublicKeySignatureVerifier,
) interpreter.Value {
	interpreter.ExpectType(
		context,
		publicKeyValue,
		sema.PublicKeyType,
		locationRange,
	)

	signature, err := interpreter.ByteArrayValueToByteSlice(context, signatureValue, locationRange)
	if err != nil {
		panic(errors.NewUnexpectedError("failed to get signature. %w", err))
	}

	signedData, err := interpreter.ByteArrayValueToByteSlice(context, signedDataValue, locationRange)
	if err != nil {
		panic(errors.NewUnexpectedError("failed to get signed data. %w", err))
	}

	domainSeparationTag := domainSeparationTagValue.Str

	hashAlgorithm := NewHashAlgorithmFromValue(context, locationRange, hashAlgorithmValue)

	publicKey, err := NewPublicKeyFromValue(context, locationRange, publicKeyValue)
	if err != nil {
		return interpreter.FalseValue
	}

	valid, err := verifier.VerifySignature(
		signature,
		domainSeparationTag,
		signedData,
		publicKey.PublicKey,
		publicKey.SignAlgo,
		hashAlgorithm,
	)
	if err != nil {
		panic(err)
	}

	return interpreter.BoolValue(valid)
}

type BLSPoPVerifier interface {
	// BLSVerifyPOP verifies a proof of possession (PoP) for the receiver public key.
	BLSVerifyPOP(publicKey *PublicKey, signature []byte) (bool, error)
}

func newInterpreterPublicKeyVerifyPoPFunction(
	inter *interpreter.Interpreter,
	publicKeyValue *interpreter.CompositeValue,
	verifier BLSPoPVerifier,
) interpreter.BoundFunctionValue {
	return interpreter.NewBoundHostFunctionValue(
		inter,
		publicKeyValue,
		sema.PublicKeyTypeVerifyPoPFunctionType,
		func(publicKeyValue *interpreter.CompositeValue, invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.InvocationContext
			locationRange := invocation.LocationRange

			signatureValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return PublicKeyVerifyPoP(
				inter,
				locationRange,
				publicKeyValue,
				signatureValue,
				verifier,
			)
		},
	)
}

func NewVMPublicKeyVerifyPoPFunction(verifier BLSPoPVerifier) VMFunction {
	return VMFunction{
		BaseType: sema.PublicKeyType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.PublicKeyTypeVerifyPoPFunctionName,
			sema.PublicKeyTypeVerifyPoPFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = vm.GetReceiverAndArgs(context, args)

				publicKeyValue, ok := receiver.(*interpreter.CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				signatureValue, ok := args[0].(*interpreter.ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return PublicKeyVerifyPoP(
					context,
					interpreter.EmptyLocationRange,
					publicKeyValue,
					signatureValue,
					verifier,
				)
			},
		),
	}
}

func PublicKeyVerifyPoP(
	context interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
	publicKeyValue *interpreter.CompositeValue,
	signatureValue *interpreter.ArrayValue,
	verifier BLSPoPVerifier,
) interpreter.Value {

	interpreter.ExpectType(
		context,
		publicKeyValue,
		sema.PublicKeyType,
		locationRange,
	)

	publicKey, err := NewPublicKeyFromValue(context, locationRange, publicKeyValue)
	if err != nil {
		panic(err)
	}

	signature, err := interpreter.ByteArrayValueToByteSlice(context, signatureValue, locationRange)
	if err != nil {
		panic(err)
	}

	valid, err := verifier.BLSVerifyPOP(publicKey, signature)
	if err != nil {
		panic(err)
	}

	return interpreter.BoolValue(valid)
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
		newInterpreterPublicKeyVerifySignatureFunction(inter, publicKeyValue, handler),
	)

	functions.Set(
		sema.PublicKeyTypeVerifyPoPFunctionName,
		newInterpreterPublicKeyVerifyPoPFunction(inter, publicKeyValue, handler),
	)

	return functions
}
