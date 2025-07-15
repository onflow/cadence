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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {
	storageCapabilityControllerTypeName := commons.TypeQualifier(sema.StorageCapabilityControllerType)

	registerBuiltinTypeBoundFunction(
		storageCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.StorageCapabilityControllerTypeSetTagFunctionName,
			sema.StorageCapabilityControllerTypeSetTagFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = GetReceiverAndArgs(context, args)

				newTagValue, ok := args[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				v := getCheckedStorageCapabilityControllerReceiver(receiver)

				v.SetTag(context, newTagValue)

				return interpreter.Void
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		storageCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.StorageCapabilityControllerTypeDeleteFunctionName,
			sema.StorageCapabilityControllerTypeDeleteFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, _ = GetReceiverAndArgs(context, args)

				v := getCheckedStorageCapabilityControllerReceiver(receiver)

				v.Delete(context, EmptyLocationRange)

				v.SetDeleted()

				return interpreter.Void
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		storageCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.StorageCapabilityControllerTypeTargetFunctionName,
			sema.StorageCapabilityControllerTypeTargetFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = GetReceiverAndArgs(context, args) // nolint:staticcheck

				v := getCheckedStorageCapabilityControllerReceiver(receiver)

				return v.TargetPath
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		storageCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.StorageCapabilityControllerTypeRetargetFunctionName,
			sema.StorageCapabilityControllerTypeRetargetFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = GetReceiverAndArgs(context, args)

				// Get path argument

				newTargetPathValue, ok := args[0].(interpreter.PathValue)
				if !ok || newTargetPathValue.Domain != common.PathDomainStorage {
					panic(errors.NewUnreachableError())
				}

				v := getCheckedStorageCapabilityControllerReceiver(receiver)

				v.SetTarget(context, EmptyLocationRange, newTargetPathValue)
				v.TargetPath = newTargetPathValue

				return interpreter.Void
			},
		),
	)
}

func getCheckedStorageCapabilityControllerReceiver(receiver Value) *interpreter.StorageCapabilityControllerValue {
	v, ok := receiver.(*interpreter.StorageCapabilityControllerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// NOTE: check if controller is already deleted
	v.CheckDeleted()

	return v
}
