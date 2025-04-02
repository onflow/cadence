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
	"github.com/onflow/cadence/stdlib"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {
	accountCapabilitiesTypeName := sema.Account_CapabilitiesType.QualifiedIdentifier()

	// Account.Capabilities.get
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		sema.Account_CapabilitiesTypeGetFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_CapabilitiesTypeGetFunctionType.Parameters),
			Function: func(config *Config, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				address := getAddressMetaInfoFromValue(args[0])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[1:]

				borrowType := typeArguments[0]
				semaBorrowType := interpreter.MustConvertStaticToSemaType(borrowType, config)

				return stdlib.GetCapability(
					arguments,
					semaBorrowType,
					false,
					config,
					address,
					EmptyLocationRange,
					config,
				)
			},
		})

	// Account.Capabilities.publish
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		sema.Account_CapabilitiesTypePublishFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_CapabilitiesTypePublishFunctionType.Parameters),
			Function: func(config *Config, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				accountAddress := getAddressMetaInfoFromValue(args[0])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[1:]

				return stdlib.PublishCapability(
					config,
					config,
					arguments,
					accountAddress,
					EmptyLocationRange,
				)
			},
		})
}
