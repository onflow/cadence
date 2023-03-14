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

package vm

import (
	"strings"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type StringValue struct {
	Str []byte
}

var _ Value = StringValue{}

func (StringValue) isValue() {}

func (StringValue) StaticType(common.MemoryGauge) StaticType {
	return interpreter.PrimitiveStaticTypeString
}

func (v StringValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v StringValue) String() string {
	return string(v.Str)
}

// members

const (
	StringConcatFunctionName = "concat"
)

func init() {
	typeName := interpreter.PrimitiveStaticTypeString.String()

	RegisterTypeBoundFunction(typeName, StringConcatFunctionName, NativeFunctionValue{
		ParameterCount: len(sema.StringTypeConcatFunctionType.Parameters),
		Function: func(value ...Value) Value {
			first := value[0].(StringValue)
			second := value[1].(StringValue)
			var sb strings.Builder
			sb.Write(first.Str)
			sb.Write(second.Str)
			return StringValue{
				[]byte(sb.String()),
			}
		},
	})
}
