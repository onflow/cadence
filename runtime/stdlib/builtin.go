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
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// This file defines functions built-in to Cadence.

// AssertFunction

const assertFunctionDocString = `
Terminates the program if the given condition is false, and reports a message which explains how the condition is false. Use this function for internal sanity checks.

The message argument is optional.
`

var AssertFunction = NewStandardLibraryFunction(
	"assert",
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "condition",
				TypeAnnotation: sema.NewTypeAnnotation(sema.BoolType),
			},
			{
				Identifier:     "message",
				TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			sema.VoidType,
		),
		RequiredArgumentCount: sema.RequiredArgumentCount(1),
	},
	assertFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		result := invocation.Arguments[0].(interpreter.BoolValue)
		if !result {
			var message string
			if len(invocation.Arguments) > 1 {
				message = invocation.Arguments[1].(*interpreter.StringValue).Str
			}
			panic(AssertionError{
				Message:       message,
				LocationRange: invocation.GetLocationRange(),
			})
		}
		return interpreter.VoidValue{}
	},
)

// PanicError

type PanicError struct {
	Message string
	interpreter.LocationRange
}

func (e PanicError) Error() string {
	return fmt.Sprintf("panic: %s", e.Message)
}

// PanicFunction

const panicFunctionDocString = `
Terminates the program unconditionally and reports a message which explains why the unrecoverable error occurred.
`

var PanicFunction = NewStandardLibraryFunction(
	"panic",
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "message",
				TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			sema.NeverType,
		),
	},
	panicFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		message := invocation.Arguments[0].(*interpreter.StringValue)
		panic(PanicError{
			Message:       message.Str,
			LocationRange: invocation.GetLocationRange(),
		})
	},
)

// BuiltinFunctions

var BuiltinFunctions = StandardLibraryFunctions{
	AssertFunction,
	PanicFunction,
	CreatePublicKeyFunction,
	AggregateBLSSignaturesFunction,
	AggregateBLSPublicKeysFunction,
}

// LogFunction

const logFunctionDocString = `
Logs a string representation of the given value
`

var LogFunction = NewStandardLibraryFunction(
	"log",
	LogFunctionType,
	logFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		println(invocation.Arguments[0].String())
		return interpreter.VoidValue{}
	},
)

// HelperFunctions

var HelperFunctions = StandardLibraryFunctions{
	LogFunction,
}

const createPublicKeyFunctionDocString = `
Constructs a new public key
`

var CreatePublicKeyFunction = NewStandardLibraryFunction(
	sema.PublicKeyTypeName,
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Identifier:     sema.PublicKeyPublicKeyField,
				TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{Type: sema.UInt8Type}),
			},
			{
				Identifier:     sema.PublicKeySignAlgoField,
				TypeAnnotation: sema.NewTypeAnnotation(sema.SignatureAlgorithmType),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.PublicKeyType),
	},
	createPublicKeyFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		publicKey := invocation.Arguments[0].(*interpreter.ArrayValue)
		signAlgo := invocation.Arguments[1].(*interpreter.CompositeValue)

		inter := invocation.Interpreter

		return interpreter.NewPublicKeyValue(
			inter,
			invocation.GetLocationRange,
			publicKey,
			signAlgo,
			inter.PublicKeyValidationHandler,
		)
	},
)

const aggregateBLSSignaturesFunctionDocString = `
This is a specific function for the BLS signature scheme. 
It aggregates multiple BLS signatures into one, 
considering the proof of possession as a defense against rogue attacks.

Signatures could be generated from the same or distinct messages, 
they could also be the aggregation of other signatures.
The order of the signatures in the slice does not matter since the aggregation is commutative. 
No subgroup membership check is performed on the input signatures.
The function errors if the array is empty or if decoding one of the signature fails. 
`

var AggregateBLSSignaturesFunction = NewStandardLibraryFunction(
	"AggregateBLSSignatures",
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "signatures",
				TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{Type: sema.ByteArrayType}),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.ByteArrayType),
	},
	aggregateBLSSignaturesFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		signatures := invocation.Arguments[0].(*interpreter.ArrayValue)
		return AggregateBLSSignatures(invocation.Interpreter, signatures)
	},
)

const aggregateBLSPublicKeysFunctionDocString = `
This is a specific function for the BLS signature scheme. 
It aggregates multiple BLS public keys into one.

The order of the public keys in the slice does not matter since the aggregation is commutative. 
No subgroup membership check is performed on the input keys.
The function errors if the array is empty or any of the input keys is not a BLS key.
`

var AggregateBLSPublicKeysFunction = NewStandardLibraryFunction(
	"AggregateBLSPublicKeys",
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "keys",
				TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{Type: sema.PublicKeyType}),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.PublicKeyType),
	},
	aggregateBLSPublicKeysFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		publicKeys := invocation.Arguments[0].(*interpreter.ArrayValue)
		return AggregateBLSPublicKeys(
			invocation.Interpreter,
			invocation.GetLocationRange,
			publicKeys,
		)
	},
)

func AggregateBLSPublicKeys(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	publicKeys *interpreter.ArrayValue,
) interpreter.Value {
	publicKeyArray := make([]interpreter.MemberAccessibleValue, 0, publicKeys.Count())
	publicKeys.Iterate(func(element interpreter.Value) (resume bool) {
		publicKey := element.(interpreter.MemberAccessibleValue)
		publicKeyArray = append(publicKeyArray, publicKey)
		return true
	})

	aggregatedKey, err := inter.AggregateBLSPublicKeysHandler(
		inter,
		getLocationRange,
		publicKeyArray,
	)
	if err != nil {
		panic(err)
	}

	return aggregatedKey
}

func AggregateBLSSignatures(
	inter *interpreter.Interpreter,
	signatures *interpreter.ArrayValue,
) interpreter.Value {
	bytesArray := make([][]byte, 0, signatures.Count())
	signatures.Iterate(func(element interpreter.Value) (resume bool) {
		sig := element.(*interpreter.ArrayValue)
		bytes := make([]byte, 0, sig.Count())
		sig.Iterate(func(element interpreter.Value) (resume bool) {
			i := element.(interpreter.UInt8Value)
			bytes = append(bytes, byte(i))
			return true
		})
		bytesArray = append(bytesArray, bytes)
		return true
	})

	aggregatedBytes, err := inter.AggregateBLSSignaturesHandler(
		bytesArray,
	)
	if err != nil {
		panic(err)
	}

	aggregatedSignature := make([]interpreter.Value, 0, len(aggregatedBytes))
	for _, b := range aggregatedBytes {
		aggregatedSignature = append(aggregatedSignature, interpreter.UInt8Value(b))
	}

	return interpreter.NewArrayValue(
		inter,
		interpreter.ByteArrayStaticType,
		signatures.GetOwner(),
		aggregatedSignature...,
	)
}

// BuiltinValues

func BuiltinValues() StandardLibraryValues {
	signatureAlgorithmValue := StandardLibraryValue{
		Name: sema.SignatureAlgorithmTypeName,
		Type: cryptoAlgorithmEnumConstructorType(
			sema.SignatureAlgorithmType,
			sema.SignatureAlgorithms,
		),
		ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
			return cryptoAlgorithmEnumValue(
				inter,
				interpreter.ReturnEmptyLocationRange,
				sema.SignatureAlgorithmType,
				sema.SignatureAlgorithms,
				NewSignatureAlgorithmCase,
			)
		},
		Kind: common.DeclarationKindEnum,
	}

	hashAlgorithmValue := StandardLibraryValue{
		Name: sema.HashAlgorithmTypeName,
		Type: cryptoAlgorithmEnumConstructorType(
			sema.HashAlgorithmType,
			sema.HashAlgorithms,
		),
		ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
			return cryptoAlgorithmEnumValue(
				inter,
				interpreter.ReturnEmptyLocationRange,
				sema.HashAlgorithmType,
				sema.HashAlgorithms,
				NewHashAlgorithmCase,
			)
		},
		Kind: common.DeclarationKindEnum,
	}

	return StandardLibraryValues{
		signatureAlgorithmValue,
		hashAlgorithmValue,
	}
}

func NewSignatureAlgorithmCase(inter *interpreter.Interpreter, rawValue uint8) *interpreter.CompositeValue {
	return interpreter.NewEnumCaseValue(
		inter,
		sema.SignatureAlgorithmType,
		interpreter.UInt8Value(rawValue),
		nil,
	)
}

var hashAlgorithmFunctions = map[string]interpreter.FunctionValue{
	sema.HashAlgorithmTypeHashFunctionName:        hashAlgorithmHashFunction,
	sema.HashAlgorithmTypeHashWithTagFunctionName: hashAlgorithmHashWithTagFunction,
}

func NewHashAlgorithmCase(inter *interpreter.Interpreter, rawValue uint8) *interpreter.CompositeValue {
	return interpreter.NewEnumCaseValue(
		inter,
		sema.HashAlgorithmType,
		interpreter.UInt8Value(rawValue),
		hashAlgorithmFunctions,
	)
}

var hashAlgorithmHashFunction = interpreter.NewHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		dataValue := invocation.Arguments[0].(*interpreter.ArrayValue)
		hashAlgoValue := invocation.Self

		inter := invocation.Interpreter

		getLocationRange := invocation.GetLocationRange

		inter.ExpectType(
			hashAlgoValue,
			sema.HashAlgorithmType,
			getLocationRange,
		)

		return inter.HashHandler(
			inter,
			getLocationRange,
			dataValue,
			nil,
			hashAlgoValue,
		)
	},
	sema.HashAlgorithmTypeHashFunctionType,
)

var hashAlgorithmHashWithTagFunction = interpreter.NewHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		dataValue := invocation.Arguments[0].(*interpreter.ArrayValue)
		tagValue := invocation.Arguments[1].(*interpreter.StringValue)
		hashAlgoValue := invocation.Self

		inter := invocation.Interpreter

		getLocationRange := invocation.GetLocationRange

		inter.ExpectType(
			hashAlgoValue,
			sema.HashAlgorithmType,
			getLocationRange,
		)

		return inter.HashHandler(
			inter,
			getLocationRange,
			dataValue,
			tagValue,
			hashAlgoValue,
		)
	},
	sema.HashAlgorithmTypeHashWithTagFunctionType,
)

func cryptoAlgorithmEnumConstructorType(
	enumType *sema.CompositeType,
	enumCases []sema.CryptoAlgorithm,
) *sema.FunctionType {

	members := make([]*sema.Member, len(enumCases))
	for i, algo := range enumCases {
		members[i] = sema.NewPublicConstantFieldMember(
			enumType,
			algo.Name(),
			enumType,
			algo.DocString(),
		)
	}

	constructorType := &sema.FunctionType{
		IsConstructor: true,
		Parameters: []*sema.Parameter{
			{
				Identifier:     sema.EnumRawValueFieldName,
				TypeAnnotation: sema.NewTypeAnnotation(enumType.EnumRawType),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			&sema.OptionalType{
				Type: enumType,
			},
		),
		Members: sema.GetMembersAsMap(members),
	}

	return constructorType
}

func cryptoAlgorithmEnumValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	enumType *sema.CompositeType,
	enumCases []sema.CryptoAlgorithm,
	caseConstructor func(inter *interpreter.Interpreter, rawValue uint8) *interpreter.CompositeValue,
) interpreter.Value {

	caseCount := len(enumCases)
	caseValues := make([]*interpreter.CompositeValue, caseCount)
	constructorNestedVariables := map[string]*interpreter.Variable{}

	for i, enumCase := range enumCases {
		rawValue := enumCase.RawValue()
		caseValue := caseConstructor(inter, rawValue)
		caseValues[i] = caseValue
		constructorNestedVariables[enumCase.Name()] =
			interpreter.NewVariableWithValue(caseValue)
	}

	return interpreter.EnumConstructorFunction(
		inter,
		getLocationRange,
		enumType,
		caseValues,
		constructorNestedVariables,
	)
}
