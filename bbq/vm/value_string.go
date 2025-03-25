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

package vm

import (
	"strings"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

//
//type StringValue struct {
//	Str []byte
//}
//
//var _ Value = StringValue{}
//
//func NewStringValue(str string) StringValue {
//	return StringValue{
//		Str: []byte(str),
//	}
//}
//
//func NewStringValueFromBytes(bytes []byte) StringValue {
//	return StringValue{
//		Str: bytes,
//	}
//}
//
//func (StringValue) isValue() {}
//
//func (StringValue) StaticType(StaticTypeContext) bbq.StaticType {
//	return interpreter.PrimitiveStaticTypeString
//}
//
//func (v StringValue) Transfer(TransferContext, atree.Address, bool, atree.Storable) Value {
//	return v
//}
//
//func (v StringValue) String() string {
//	return string(v.Str)
//}

// members

func init() {
	typeName := interpreter.PrimitiveStaticTypeString.String()

	RegisterTypeBoundFunction(typeName, sema.StringTypeConcatFunctionName, NativeFunctionValue{
		ParameterCount: len(sema.StringTypeConcatFunctionType.Parameters),
		Function: func(config *Config, typeArguments []bbq.StaticType, value ...Value) Value {
			first := value[0].(*interpreter.StringValue)
			second := value[1].(*interpreter.StringValue)

			// TODO: Use `StringValue.Concat()`
			var sb strings.Builder
			sb.WriteString(first.Str)
			sb.WriteString(second.Str)
			return interpreter.NewUnmeteredStringValue(sb.String())
		},
	})
}
