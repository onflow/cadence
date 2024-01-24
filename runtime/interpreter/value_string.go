/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package interpreter

import (
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

func stringFunctionEncodeHex(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter
	memoryUsage := common.NewStringMemoryUsage(
		safeMul(argument.Count(), 2, invocation.LocationRange),
	)
	return NewStringValue(
		inter,
		memoryUsage,
		func() string {
			bytes, _ := ByteArrayValueToByteSlice(inter, argument, invocation.LocationRange)
			return hex.EncodeToString(bytes)
		},
	)
}

func stringFunctionFromUtf8(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter
	// naively read the entire byte array before validating
	buf, err := ByteArrayValueToByteSlice(inter, argument, invocation.LocationRange)

	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if !utf8.Valid(buf) {
		return Nil
	}

	memoryUsage := common.NewStringMemoryUsage(len(buf))

	return NewSomeValueNonCopying(
		inter,
		NewStringValue(inter, memoryUsage, func() string {
			return string(buf)
		}),
	)
}

func stringFunctionFromCharacters(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter

	// NewStringMemoryUsage already accounts for empty string.
	common.UseMemory(inter, common.NewStringMemoryUsage(0))
	var builder strings.Builder

	argument.Iterate(inter, func(element Value) (resume bool) {
		character := element.(CharacterValue)
		// Construct directly instead of using NewStringMemoryUsage to avoid
		// having to decrement by 1 due to double counting of empty string.
		common.UseMemory(inter,
			common.MemoryUsage{
				Kind:   common.MemoryKindStringValue,
				Amount: uint64(len(character)),
			},
		)
		builder.WriteString(string(character))

		return true
	})

	return NewUnmeteredStringValue(builder.String())
}

func stringFunctionJoin(invocation Invocation) Value {
	stringArray, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter

	switch stringArray.Count() {
	case 0:
		return EmptyString
	case 1:
		return stringArray.Get(inter, invocation.LocationRange, 0)
	}

	separator, ok := invocation.Arguments[1].(*StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// NewStringMemoryUsage already accounts for empty string.
	common.UseMemory(inter, common.NewStringMemoryUsage(0))
	var builder strings.Builder
	first := true

	stringArray.Iterate(inter, func(element Value) (resume bool) {

		// Meter computation for iterating the array.
		inter.ReportComputation(common.ComputationKindLoop, 1)

		// Add separator
		if !first {
			// Construct directly instead of using NewStringMemoryUsage to avoid
			// having to decrement by 1 due to double counting of empty string.
			common.UseMemory(inter,
				common.MemoryUsage{
					Kind:   common.MemoryKindStringValue,
					Amount: uint64(len(separator.Str)),
				},
			)
			builder.WriteString(separator.Str)
		}
		first = false

		str, ok := element.(*StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// Construct directly instead of using NewStringMemoryUsage to avoid
		// having to decrement by 1 due to double counting of empty string.
		common.UseMemory(inter,
			common.MemoryUsage{
				Kind:   common.MemoryKindStringValue,
				Amount: uint64(len(str.Str)),
			},
		)
		builder.WriteString(str.Str)

		return true
	})

	return NewUnmeteredStringValue(builder.String())
}

// stringFunction is the `String` function. It is stateless, hence it can be re-used across interpreters.
var stringFunction = func() Value {
	functionValue := NewUnmeteredHostFunctionValue(
		sema.StringFunctionType,
		func(invocation Invocation) Value {
			return EmptyString
		},
	)

	addMember := func(name string, value Value) {
		if functionValue.NestedVariables == nil {
			functionValue.NestedVariables = map[string]*Variable{}
		}
		// these variables are not needed to be metered as they are only ever declared once,
		// and can be considered base interpreter overhead
		functionValue.NestedVariables[name] = NewVariableWithValue(nil, value)
	}

	addMember(
		sema.StringTypeEncodeHexFunctionName,
		NewUnmeteredHostFunctionValue(
			sema.StringTypeEncodeHexFunctionType,
			stringFunctionEncodeHex,
		),
	)

	addMember(
		sema.StringTypeFromUtf8FunctionName,
		NewUnmeteredHostFunctionValue(
			sema.StringTypeFromUtf8FunctionType,
			stringFunctionFromUtf8,
		),
	)

	addMember(
		sema.StringTypeFromCharactersFunctionName,
		NewUnmeteredHostFunctionValue(
			sema.StringTypeFromCharactersFunctionType,
			stringFunctionFromCharacters,
		),
	)

	addMember(
		sema.StringTypeJoinFunctionName,
		NewUnmeteredHostFunctionValue(
			sema.StringTypeJoinFunctionType,
			stringFunctionJoin,
		),
	)

	return functionValue
}()
