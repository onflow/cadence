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
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// Members

func init() {
	typeName := interpreter.PrimitiveStaticTypeCapability.String()

	// Capability.borrow
	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValueWithDerivedType(
			sema.CapabilityTypeBorrowFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				capability := receiver.(*interpreter.IDCapabilityValue)
				borrowType := context.SemaTypeFromStaticType(capability.BorrowType).(*sema.ReferenceType)
				return sema.CapabilityTypeBorrowFunctionType(borrowType)
			},
			interpreter.NativeCapabilityBorrowFunction(nil, nil, nil),
		),
	)

	// Capability.check
	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValueWithDerivedType(
			sema.CapabilityTypeCheckFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				capability := receiver.(*interpreter.IDCapabilityValue)
				borrowType := context.SemaTypeFromStaticType(capability.BorrowType).(*sema.ReferenceType)
				return sema.CapabilityTypeCheckFunctionType(borrowType)
			},
			interpreter.NativeCapabilityCheckFunction(nil, nil, nil),
		),
	)
}
