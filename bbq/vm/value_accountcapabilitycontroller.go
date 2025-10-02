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
	accountCapabilityControllerTypeName := commons.TypeQualifier(sema.AccountCapabilityControllerType)

	registerBuiltinTypeBoundFunction(
		accountCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.AccountCapabilityControllerTypeSetTagFunctionName,
			sema.AccountCapabilityControllerTypeSetTagFunctionType,
			func(context *Context, _ []bbq.StaticType, receiver Value, args ...Value) Value {

				newTagValue, ok := args[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				v := getCheckedAccountCapabilityControllerReceiver(receiver)

				v.SetTag(context, newTagValue)

				return interpreter.Void
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		accountCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.AccountCapabilityControllerTypeDeleteFunctionName,
			sema.AccountCapabilityControllerTypeDeleteFunctionType,
			func(context *Context, _ []bbq.StaticType, receiver Value, _ ...Value) Value {

				v := getCheckedAccountCapabilityControllerReceiver(receiver)

				v.Delete(context)

				v.SetDeleted()

				return interpreter.Void
			},
		),
	)
}

func getCheckedAccountCapabilityControllerReceiver(receiver Value) *interpreter.AccountCapabilityControllerValue {
	v, ok := receiver.(*interpreter.AccountCapabilityControllerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// NOTE: check if controller is already deleted
	v.CheckDeleted()

	return v
}
