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
}

// LogFunction

const logFunctionDocString = `
Logs a string representation of the given value
`

var LogFunction = NewStandardLibraryFunction(
	"log",
	logFunctionType,
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

		validationFunc := invocation.Interpreter.PublicKeyValidationHandler

		return interpreter.NewPublicKeyValue(
			publicKey,
			signAlgo,
			validationFunc,
		)
	},
)

// BuiltinValues

var BuiltinValues = StandardLibraryValues{
	SignatureAlgorithmValue,
	HashAlgorithmValue,
}

var SignatureAlgorithmValue = StandardLibraryValue{
	Name: sema.SignatureAlgorithmTypeName,
	Type: cryptoAlgorithmEnumType(
		sema.SignatureAlgorithmType,
		sema.SignatureAlgorithms,
	),
	Value: cryptoAlgorithmEnumValue(
		sema.SignatureAlgorithms,
		NewSignatureAlgorithmCase,
	),
	Kind: common.DeclarationKindEnum,
}

func NewSignatureAlgorithmCase(rawValue uint8) *interpreter.CompositeValue {
	return interpreter.NewEnumCaseValue(
		sema.SignatureAlgorithmType,
		interpreter.UInt8Value(rawValue),
		nil,
	)
}

var hashAlgorithmFunctions = map[string]interpreter.FunctionValue{
	sema.HashAlgorithmTypeHashFunctionName:        hashAlgorithmHashFunction,
	sema.HashAlgorithmTypeHashWithTagFunctionName: hashAlgorithmHashWithTagFunction,
}

func NewHashAlgorithmCase(rawValue uint8) *interpreter.CompositeValue {
	return interpreter.NewEnumCaseValue(
		sema.HashAlgorithmType,
		interpreter.UInt8Value(rawValue),
		hashAlgorithmFunctions,
	)
}

var HashAlgorithmValue = StandardLibraryValue{
	Name: sema.HashAlgorithmTypeName,
	Type: cryptoAlgorithmEnumType(
		sema.HashAlgorithmType,
		sema.HashAlgorithms,
	),
	Value: cryptoAlgorithmEnumValue(
		sema.HashAlgorithms,
		NewHashAlgorithmCase,
	),
	Kind: common.DeclarationKindEnum,
}

var hashAlgorithmHashFunction = interpreter.NewHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		dataValue := invocation.Arguments[0].(*interpreter.ArrayValue)
		hashAlgoValue := invocation.Self

		invocation.Interpreter.ExpectType(
			hashAlgoValue,
			sema.HashAlgorithmType,
			invocation.GetLocationRange,
		)

		return invocation.Interpreter.HashHandler(dataValue, nil, hashAlgoValue)
	},
)

var hashAlgorithmHashWithTagFunction = interpreter.NewHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		dataValue := invocation.Arguments[0].(*interpreter.ArrayValue)
		tagValue := invocation.Arguments[1].(*interpreter.StringValue)
		hashAlgoValue := invocation.Self

		invocation.Interpreter.ExpectType(
			hashAlgoValue,
			sema.HashAlgorithmType,
			invocation.GetLocationRange,
		)

		return invocation.Interpreter.HashHandler(
			dataValue,
			tagValue,
			hashAlgoValue,
		)
	},
)

func cryptoAlgorithmEnumType(enumType *sema.CompositeType, enumCases []sema.CryptoAlgorithm) *sema.FunctionType {
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
	enumCases []sema.CryptoAlgorithm,
	caseConstructor func(rawValue uint8) *interpreter.CompositeValue,
) interpreter.Value {

	caseCount := len(enumCases)
	caseValues := make([]*interpreter.CompositeValue, caseCount)
	constructorNestedVariables := interpreter.NewStringVariableOrderedMap()

	for i, enumCase := range enumCases {
		rawValue := enumCase.RawValue()
		caseValue := caseConstructor(rawValue)
		caseValues[i] = caseValue
		constructorNestedVariables.Set(
			enumCase.Name(),
			interpreter.NewVariableWithValue(caseValue),
		)
	}

	return interpreter.EnumConstructorFunction(caseValues, constructorNestedVariables)
}
