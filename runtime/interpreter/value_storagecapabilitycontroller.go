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
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

var storageCapabilityControllerFieldNames = []string{
	sema.StorageCapabilityControllerTypeBorrowTypeFieldName,
	sema.StorageCapabilityControllerTypeCapabilityIDFieldName,
}

func NewStorageCapabilityControllerValue(
	gauge common.MemoryGauge,
	capabilityID uint64,
	borrowType StaticType,
	delete func() error,
	getTarget func() (PathValue, error),
	retarget func(newPath PathValue) error,
) Value {

	borrowTypeValue := NewTypeValue(gauge, borrowType)
	fields := map[string]Value{
		sema.StorageCapabilityControllerTypeBorrowTypeFieldName: borrowTypeValue,
		sema.StorageCapabilityControllerTypeCapabilityIDFieldName: NewUInt64Value(gauge, func() uint64 {
			return capabilityID
		}),
	}

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.StorageCapabilityControllerTypeTargetFunctionName:
			return NewHostFunctionValue(
				gauge,
				sema.StorageCapabilityControllerTypeTargetFunctionType,
				func(invocation Invocation) Value {
					target, err := getTarget()
					if err != nil {
						panic(err)
					}

					return target
				},
			)

		case sema.StorageCapabilityControllerTypeDeleteFunctionName:
			return NewHostFunctionValue(
				gauge,
				sema.StorageCapabilityControllerTypeDeleteFunctionType,
				func(invocation Invocation) Value {
					err := delete()
					if err != nil {
						panic(err)
					}

					return Void
				},
			)

		case sema.StorageCapabilityControllerTypeRetargetFunctionName:
			return NewHostFunctionValue(
				gauge,
				sema.StorageCapabilityControllerTypeRetargetFunctionType,
				func(invocation Invocation) Value {
					newTarget, ok := invocation.Arguments[0].(PathValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					err := retarget(newTarget)
					if !ok {
						panic(err)
					}

					return Void
				},
			)
		}

		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.StorageCapabilityControllerStringMemoryUsage)

			borrowTypeStr := borrowTypeValue.MeteredString(gauge, seenReferences)

			memoryUsage := common.NewStringMemoryUsage(OverEstimateUintStringLength(uint(capabilityID)))
			common.UseMemory(memoryGauge, memoryUsage)

			idStr := fmt.Sprint(capabilityID)

			str = format.StorageCapabilityController(borrowTypeStr, idStr)
		}

		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		sema.StorageCapabilityControllerType.ID(),
		PrimitiveStaticTypeStorageCapabilityController,
		storageCapabilityControllerFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}
