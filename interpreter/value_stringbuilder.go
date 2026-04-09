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

package interpreter

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

// StringBuilder

const stringBuilderBuilderFieldName = "builder"

var stringBuilderStaticType StaticType = PrimitiveStaticTypeStringBuilder

// stringBuilderFunction is the `StringBuilder` constructor function.
// It is stateless, hence it can be re-used across interpreters.
var stringBuilderFunction = NewUnmeteredStaticHostFunctionValueFromNativeFunction(
	sema.StringBuilderFunctionType,
	NativeStringBuilderConstructor,
)

var NativeStringBuilderConstructor = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		_ []Value,
	) Value {
		return NewStringBuilderValue(context)
	},
)

func getStringBuilderBuilder(receiver Value) *strings.Builder {
	compositeValue := AssertValueOfType[*SimpleCompositeValue](receiver)
	return compositeValue.PrivateField(stringBuilderBuilderFieldName).(*strings.Builder)
}

func NewStringBuilderValue(gauge common.MemoryGauge) *SimpleCompositeValue {
	var builder strings.Builder

	common.UseMemory(gauge, common.StringBuilderMemoryUsage)

	computeField := func(name string, _ MemberAccessibleContext) Value {
		switch name {
		case sema.StringBuilderTypeLengthFieldName:
			return NewIntValueFromInt64(gauge, int64(builder.Len()))
		}
		return nil
	}

	// Declare the value variable so the closure below can capture it.
	// It will be assigned after NewSimpleCompositeValue returns.
	var value *SimpleCompositeValue

	var methods map[string]FunctionValue

	functionMemberGetter := func(name string, context MemberAccessibleContext) FunctionValue {
		if methods == nil {
			methods = make(map[string]FunctionValue)
		}

		method, ok := methods[name]
		if ok {
			return method
		}

		var nativeFunc NativeFunction
		var funcType *sema.FunctionType

		switch name {
		case sema.StringBuilderTypeAppendFunctionName:
			nativeFunc = NativeStringBuilderAppendFunction
			funcType = sema.StringBuilderTypeAppendFunctionType
		case sema.StringBuilderTypeAppendCharacterFunctionName:
			nativeFunc = NativeStringBuilderAppendCharacterFunction
			funcType = sema.StringBuilderTypeAppendCharacterFunctionType
		case sema.StringBuilderTypeClearFunctionName:
			nativeFunc = NativeStringBuilderClearFunction
			funcType = sema.StringBuilderTypeClearFunctionType
		case sema.StringBuilderTypeToStringFunctionName:
			nativeFunc = NativeStringBuilderToStringFunction
			funcType = sema.StringBuilderTypeToStringFunctionType
		}

		if nativeFunc != nil {
			self := Value(value)
			method = NewBoundHostFunctionValue(
				context,
				self,
				funcType,
				nativeFunc,
			)
			methods[name] = method
		}

		return method
	}

	stringer := func(context ValueStringContext, _ SeenReferences) string {
		typeId := string(sema.StringBuilderType.TypeID)
		length := builder.Len()
		str := fmt.Sprintf("%s(length: %d)", typeId, length)
		common.UseMemory(context, common.NewRawStringMemoryUsage(len(str)))
		return str
	}

	value = NewSimpleCompositeValue(
		gauge,
		sema.StringBuilderType.TypeID,
		stringBuilderStaticType,
		nil,
		nil,
		computeField,
		functionMemberGetter,
		nil,
		stringer,
	)
	value.WithPrivateField(stringBuilderBuilderFieldName, &builder)

	return value
}

var NativeStringBuilderAppendFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		builder := getStringBuilderBuilder(receiver)
		str := AssertValueOfType[*StringValue](args[0])
		common.UseMemory(context, common.NewRawStringMemoryUsage(len(str.Str)))
		builder.WriteString(str.Str)
		return Void
	},
)

var NativeStringBuilderAppendCharacterFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		builder := getStringBuilderBuilder(receiver)
		char := AssertValueOfType[CharacterValue](args[0])
		common.UseMemory(context, common.NewRawStringMemoryUsage(len(char.Str)))
		builder.WriteString(char.Str)
		return Void
	},
)

var NativeStringBuilderClearFunction = NativeFunction(
	func(
		_ NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		_ []Value,
	) Value {
		builder := getStringBuilderBuilder(receiver)
		builder.Reset()
		return Void
	},
)

var NativeStringBuilderToStringFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		_ []Value,
	) Value {
		builder := getStringBuilderBuilder(receiver)
		str := builder.String()
		common.UseMemory(context, common.NewRawStringMemoryUsage(len(str)))
		return NewUnmeteredStringValue(str)
	},
)
