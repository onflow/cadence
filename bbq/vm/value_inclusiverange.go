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
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierInclusiveRange,
		NewNativeFunctionValueWithDerivedType(
			sema.InclusiveRangeTypeContainsFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				rangeType, ok := receiver.StaticType(context).(interpreter.InclusiveRangeStaticType)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				elementType := interpreter.MustConvertStaticToSemaType(rangeType.ElementType, context)
				return sema.InclusiveRangeContainsFunctionType(elementType)
			},
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = SplitReceiverAndArgs(context, args)

				rangeValue, ok := receiver.(*interpreter.CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				needleInteger, ok := args[0].(interpreter.IntegerValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				rangeType, ok := rangeValue.StaticType(context).(interpreter.InclusiveRangeStaticType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return interpreter.InclusiveRangeContains(
					rangeValue,
					rangeType,
					context,
					EmptyLocationRange,
					needleInteger,
				)
			},
		),
	)
}
