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
	"github.com/onflow/cadence/runtime/trampoline"
)

// This file defines functions built-in to Cadence.

// AssertFunction

var AssertFunction = NewStandardLibraryFunction(
	"assert",
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "condition",
				TypeAnnotation: sema.NewTypeAnnotation(&sema.BoolType{}),
			},
			{
				Identifier:     "message",
				TypeAnnotation: sema.NewTypeAnnotation(&sema.StringType{}),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			sema.VoidType,
		),
		RequiredArgumentCount: sema.RequiredArgumentCount(1),
	},
	func(invocation interpreter.Invocation) trampoline.Trampoline {
		result := invocation.Arguments[0].(interpreter.BoolValue)
		if !result {
			var message string
			if len(invocation.Arguments) > 1 {
				message = invocation.Arguments[1].(*interpreter.StringValue).Str
			}
			panic(AssertionError{
				Message:       message,
				LocationRange: invocation.LocationRange,
			})
		}
		return trampoline.Done{}
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

var PanicFunction = NewStandardLibraryFunction(
	"panic",
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "message",
				TypeAnnotation: sema.NewTypeAnnotation(&sema.StringType{}),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			sema.NeverType,
		),
	},
	func(invocation interpreter.Invocation) trampoline.Trampoline {
		message := invocation.Arguments[0].(*interpreter.StringValue)
		panic(PanicError{
			Message:       message.Str,
			LocationRange: invocation.LocationRange,
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

var LogFunction = NewStandardLibraryFunction(
	"log",
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: sema.NewTypeAnnotation(&sema.AnyStructType{}),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			sema.VoidType,
		),
	},
	func(invocation interpreter.Invocation) trampoline.Trampoline {
		fmt.Printf("%v\n", invocation.Arguments[0])
		result := interpreter.VoidValue{}
		return trampoline.Done{Result: result}
	},
)

// HelperFunctions

var HelperFunctions = StandardLibraryFunctions{
	LogFunction,
}

var CreatePublicKeyFunction = NewStandardLibraryFunction(
	sema.PublicKeyTypeName,
	&sema.FunctionType{
		Parameters: []*sema.Parameter{
			{
				Label:          sema.PublicKeyPublicKeyField,
				Identifier:     sema.PublicKeyPublicKeyField,
				TypeAnnotation: sema.NewTypeAnnotation(&sema.VariableSizedType{Type: &sema.UInt8Type{}}),
			},
			{
				Label:          sema.PublicKeySignAlgoField,
				Identifier:     sema.PublicKeySignAlgoField,
				TypeAnnotation: sema.NewTypeAnnotation(sema.SignatureAlgorithmType),
			},
		},
		ReturnTypeAnnotation:  sema.NewTypeAnnotation(sema.PublicKeyType),
		RequiredArgumentCount: sema.RequiredArgumentCount(2),
	},

	func(invocation interpreter.Invocation) trampoline.Trampoline {
		publicKey := invocation.Arguments[0].(*interpreter.ArrayValue)
		signAlgo := invocation.Arguments[1].(*interpreter.BuiltinStructValue)

		value := interpreter.NewPublicKeyValue(publicKey, signAlgo)
		return trampoline.Done{Result: value}
	},
)

// BuiltinValues

var BuiltinValues = StandardLibraryValues{
	SignatureAlgorithmValue,
	HashAlgorithmValue,
}

var SignatureAlgorithmValue = StandardLibraryValue{
	Name:  sema.SignatureAlgorithmTypeName,
	Type:  getEnumType(sema.SignatureAlgorithmType, sema.SignatureAlgorithms),
	Value: getEnumValue(sema.SignatureAlgorithmType, sema.SignatureAlgorithms),
	Kind:  common.DeclarationKindEnum,
}

var HashAlgorithmValue = StandardLibraryValue{
	Name:  sema.HashAlgorithmTypeName,
	Type:  getEnumType(sema.HashAlgorithmType, sema.HashAlgorithms),
	Value: getEnumValue(sema.HashAlgorithmType, sema.HashAlgorithms),
	Kind:  common.DeclarationKindEnum,
}

func getEnumType(enumType *sema.BuiltinStructType, enumCases []sema.BuiltinEnumCase) *sema.SpecialFunctionType {
	members := make([]*sema.Member, len(enumCases))
	for _, algo := range enumCases {
		members[algo.RawValue()] = sema.NewPublicEnumCaseMember(
			enumType,
			algo.Name(),
			algo.DocString(),
		)
	}

	constructorType := &sema.SpecialFunctionType{
		FunctionType: &sema.FunctionType{
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
		},
		Members: sema.GetMembersAsMap(members),
	}

	return constructorType
}

func getEnumValue(enumType *sema.BuiltinStructType, enumCases []sema.BuiltinEnumCase) (value interpreter.Value) {
	caseCount := len(enumCases)
	caseValues := make([]*interpreter.BuiltinStructValue, caseCount)
	constructorMembers := make(map[string]interpreter.Value, caseCount)

	for _, enumCase := range enumCases {
		rawValue := enumCase.RawValue()
		caseValue := interpreter.NewEnumCaseValue(enumType, rawValue)
		caseValues[rawValue] = caseValue
		constructorMembers[enumCase.Name()] = caseValue
	}

	constructor := interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {
			rawValueArgument := invocation.Arguments[0].(interpreter.IntegerValue).ToInt()
			var result interpreter.Value = interpreter.NilValue{}

			if rawValueArgument >= 0 && rawValueArgument < caseCount {
				caseValue := caseValues[rawValueArgument]
				result = interpreter.NewSomeValueOwningNonCopying(caseValue)
			}

			return trampoline.Done{Result: result}
		},
	)

	constructor.Members = constructorMembers
	value = constructor
	return value
}
