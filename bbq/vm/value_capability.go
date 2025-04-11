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
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// Members

func init() {
	typeName := interpreter.PrimitiveStaticTypeCapability.String()

	// Capability.borrow
	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.CapabilityTypeBorrowFunctionName,
			// TODO: Should the borrow type need to be changed for each usage?
			sema.CapabilityTypeBorrowFunctionType(nil),
			func(config *Config, typeArguments []bbq.StaticType, args ...Value) Value {
				capabilityValue := getReceiver[*interpreter.IDCapabilityValue](config, args[receiverIndex])
				capabilityID := capabilityValue.ID

				if capabilityID == interpreter.InvalidCapabilityID {
					return interpreter.Nil
				}

				capabilityBorrowType := interpreter.MustConvertStaticToSemaType(capabilityValue.BorrowType, config).(*sema.ReferenceType)

				var typeParameter sema.Type
				if len(typeArguments) > 0 {
					typeParameter = interpreter.MustConvertStaticToSemaType(typeArguments[0], config)
				}

				address := capabilityValue.Address()

				return interpreter.CapabilityBorrow(
					config,
					typeParameter,
					address,
					capabilityID,
					capabilityBorrowType,
					EmptyLocationRange,
				)
			},
		),
	)
}
