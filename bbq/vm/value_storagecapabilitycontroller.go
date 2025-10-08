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
	"github.com/onflow/cadence/bbq/commons"
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
			interpreter.NewNativeDeletionCheckedStorageCapabilityControllerFunction(
				interpreter.NativeStorageCapabilityControllerSetTagFunction,
			),
		),
	)

	registerBuiltinTypeBoundFunction(
		storageCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.StorageCapabilityControllerTypeDeleteFunctionName,
			sema.StorageCapabilityControllerTypeDeleteFunctionType,
			interpreter.NewNativeDeletionCheckedStorageCapabilityControllerFunction(
				interpreter.NativeStorageCapabilityControllerDeleteFunction,
			),
		),
	)

	registerBuiltinTypeBoundFunction(
		storageCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.StorageCapabilityControllerTypeTargetFunctionName,
			sema.StorageCapabilityControllerTypeTargetFunctionType,
			interpreter.NewNativeDeletionCheckedStorageCapabilityControllerFunction(
				interpreter.NativeStorageCapabilityControllerTargetFunction,
			),
		),
	)

	registerBuiltinTypeBoundFunction(
		storageCapabilityControllerTypeName,
		NewNativeFunctionValue(
			sema.StorageCapabilityControllerTypeRetargetFunctionName,
			sema.StorageCapabilityControllerTypeRetargetFunctionType,
			interpreter.NewNativeDeletionCheckedStorageCapabilityControllerFunction(
				interpreter.NativeStorageCapabilityControllerRetargetFunction,
			),
		),
	)
}
