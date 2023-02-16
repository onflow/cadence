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

var capabilityControllerFieldNames = []string{
	sema.CapabilityControllerTypeIssueHeightFieldName,
	sema.CapabilityControllerTypeBorrowTypeFieldName,
	sema.CapabilityControllerTypeCapabilityIDFieldName,
}

func NewCapabilityControllerValue(
	gauge common.MemoryGauge,
	issueHeight uint64,
	capabilityID uint64,
	targetPath PathValue,
	borrowType StaticType,
	delete func() error,
	retarget func(newPath PathValue) error,
) Value {

	borrowTypeValue := NewTypeValue(gauge, borrowType)
	fields := map[string]Value{
		sema.CapabilityControllerTypeIssueHeightFieldName: NewUInt64Value(gauge, func() uint64 {
			return issueHeight
		}),
		sema.CapabilityControllerTypeBorrowTypeFieldName: borrowTypeValue,
		sema.CapabilityControllerTypeCapabilityIDFieldName: NewUInt64Value(gauge, func() uint64 {
			return capabilityID
		}),
	}

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.CapabilityControllerTypeTargetFunctionName:
			return NewHostFunctionValue(gauge, func(invocation Invocation) Value {
				return targetPath
			}, sema.CapabilityControllerTypeTargetFunctionType)

		case sema.CapabilityControllerTypeDeleteFunctionName:
			return NewHostFunctionValue(gauge, func(invocation Invocation) Value {
				err := delete()
				if err != nil {
					panic(err)
				}

				return Void
			}, sema.CapabilityControllerTypeDeleteFunctionType)

		case sema.CapabilityControllerTypeRetargetFunctionName:
			return NewHostFunctionValue(gauge, func(invocation Invocation) Value {
				newTarget, ok := invocation.Arguments[0].(PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				err := retarget(newTarget)
				if !ok {
					panic(err)
				}

				targetPath = newTarget
				return Void
			}, sema.CapabilityControllerTypeRetargetFunctionType)
		}

		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.CapabilityControllerStringMemoryUsage)

			// "Type<T>()"
			borrowTypeStr := borrowTypeValue.MeteredString(gauge, seenReferences)

			memoryUsage := common.NewStringMemoryUsage(OverEstimateUintStringLength(uint(capabilityID)))
			common.UseMemory(memoryGauge, memoryUsage)

			idStr := fmt.Sprint(capabilityID)

			str = format.CapabilityController(borrowTypeStr, idStr)
		}

		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		sema.CapabilityControllerType.ID(),
		PrimitiveStaticTypeCapabilityController,
		capabilityControllerFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}
