/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"errors"
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	errors2 "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/internal"
	"github.com/onflow/cadence/runtime/trampoline"
)

type CryptoSignatureVerifier interface {
	VerifySignature(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm string,
		hashAlgorithm string,
	) (bool, error)
}

type CryptoHasher interface {
	Hash(
		data []byte,
		hashAlgorithm string,
	) ([]byte, error)
}

var CryptoChecker = func() *sema.Checker {

	code := internal.MustAssetString("contracts/crypto.cdc")

	program, err := parser2.ParseProgram(code)
	if err != nil {
		panic(err)
	}

	location := common.IdentifierLocation("Crypto")

	var checker *sema.Checker
	checker, err = sema.NewChecker(
		program,
		location,
		sema.WithPredeclaredValues(BuiltinFunctions.ToSemaValueDeclarations()),
		sema.WithPredeclaredTypes(BuiltinTypes.ToTypeDeclarations()),
	)
	if err != nil {
		panic(err)
	}

	err = checker.Check()
	if err != nil {
		panic(err)
	}

	return checker
}()

var cryptoContractType = func() *sema.CompositeType {
	variable, ok := CryptoChecker.Elaboration.GlobalTypes.Get("Crypto")
	if !ok {
		panic(errors2.NewUnreachableError())
	}
	return variable.Type.(*sema.CompositeType)
}()

var cryptoContractInitializerTypes = func() (result []sema.Type) {
	result = make([]sema.Type, len(cryptoContractType.ConstructorParameters))
	for i, parameter := range cryptoContractType.ConstructorParameters {
		result[i] = parameter.TypeAnnotation.Type
	}
	return result
}()

func newCryptoContractVerifySignatureFunction(signatureVerifier CryptoSignatureVerifier) interpreter.FunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {
			signature, err := interpreter.ByteArrayValueToByteSlice(invocation.Arguments[0])
			if err != nil {
				panic(fmt.Errorf("verifySignature: invalid signature argument: %w", err))
			}

			tagStringValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.New("verifySignature: invalid tag argument: not a string"))
			}
			tag := tagStringValue.Str

			signedData, err := interpreter.ByteArrayValueToByteSlice(invocation.Arguments[2])
			if err != nil {
				panic(fmt.Errorf("verifySignature: invalid signed data argument: %w", err))
			}

			publicKey, err := interpreter.ByteArrayValueToByteSlice(invocation.Arguments[3])
			if err != nil {
				panic(fmt.Errorf("verifySignature: invalid public key argument: %w", err))
			}

			signatureAlgorithmStringValue, ok := invocation.Arguments[4].(*interpreter.StringValue)
			if !ok {
				panic(errors.New("verifySignature: invalid signature algorithm argument: not a string"))
			}
			signatureAlgorithm := signatureAlgorithmStringValue.Str

			hashAlgorithmStringValue, ok := invocation.Arguments[5].(*interpreter.StringValue)
			if !ok {
				panic(errors.New("verifySignature: invalid hash algorithm argument: not a string"))
			}
			hashAlgorithm := hashAlgorithmStringValue.Str

			isValid, err := signatureVerifier.VerifySignature(signature,
				tag,
				signedData,
				publicKey,
				signatureAlgorithm,
				hashAlgorithm,
			)
			if err != nil {
				panic(err)
			}

			return trampoline.Done{Result: interpreter.BoolValue(isValid)}
		},
	)
}

func newCryptoContractSignatureVerifier(signatureVerifier CryptoSignatureVerifier) *interpreter.CompositeValue {
	implIdentifier := CryptoChecker.Location.
		QualifiedIdentifier(cryptoContractInitializerTypes[0].ID()) +
		"Impl"

	result := interpreter.NewCompositeValue(
		CryptoChecker.Location,
		implIdentifier,
		common.CompositeKindStructure,
		nil,
		nil,
	)

	result.Functions = map[string]interpreter.FunctionValue{
		"verify": newCryptoContractVerifySignatureFunction(signatureVerifier),
	}

	return result
}

func newCryptoContractHashFunction(hasher CryptoHasher) interpreter.FunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {
			data, err := interpreter.ByteArrayValueToByteSlice(invocation.Arguments[0])
			if err != nil {
				panic(fmt.Errorf("hash: invalid data argument: %w", err))
			}

			hashAlgorithmStringValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.New("hash: invalid hash algorithm argument: not a string"))
			}
			hashAlgorithm := hashAlgorithmStringValue.Str

			digest, err := hasher.Hash(data, hashAlgorithm)
			if err != nil {
				panic(err)

			}

			result := interpreter.ByteSliceToByteArrayValue(digest)

			return trampoline.Done{Result: result}
		},
	)
}

func newCryptoContractHasher(hasher CryptoHasher) *interpreter.CompositeValue {
	implIdentifier := CryptoChecker.Location.
		QualifiedIdentifier(cryptoContractInitializerTypes[1].ID()) +
		"Impl"

	result := interpreter.NewCompositeValue(
		CryptoChecker.Location,
		implIdentifier,
		common.CompositeKindStructure,
		nil,
		nil,
	)

	result.Functions = map[string]interpreter.FunctionValue{
		"hash": newCryptoContractHashFunction(hasher),
	}

	return result
}

func NewCryptoContract(
	inter *interpreter.Interpreter,
	constructor interpreter.FunctionValue,
	signatureVerifier CryptoSignatureVerifier,
	hasher CryptoHasher,
	invocationRange ast.Range,
) (
	*interpreter.CompositeValue,
	error,
) {

	var cryptoContractInitializerArguments = []interpreter.Value{
		newCryptoContractSignatureVerifier(signatureVerifier),
		newCryptoContractHasher(hasher),
	}

	value, err := inter.InvokeFunctionValue(
		constructor,
		cryptoContractInitializerArguments,
		cryptoContractInitializerTypes,
		cryptoContractInitializerTypes,
		invocationRange,
	)
	if err != nil {
		return nil, err
	}

	compositeValue := value.(*interpreter.CompositeValue)

	return compositeValue, nil
}
