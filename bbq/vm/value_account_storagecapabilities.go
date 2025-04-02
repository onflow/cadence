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
	"github.com/onflow/cadence/stdlib"
)

// members

func init() {
	accountStorageCapabilitiesTypeName := sema.Account_StorageCapabilitiesType.QualifiedIdentifier()

	// Account.StorageCapabilities.issue
	RegisterTypeBoundFunction(
		accountStorageCapabilitiesTypeName,
		sema.Account_StorageCapabilitiesTypeIssueFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageCapabilitiesTypeIssueFunctionType.Parameters),
			Function: func(config *Config, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.StorageCapabilities)
				accountAddress := getAddressMetaInfoFromValue(args[receiverIndex]).ToAddress()

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				// Get borrow type type-argument
				typeParameter := typeArguments[0]
				semaType := interpreter.MustConvertStaticToSemaType(typeParameter, config)

				return stdlib.IssueCapability(
					arguments,
					config,
					EmptyLocationRange,
					config.GetAccountHandler(),
					accountAddress,
					semaType,
				)
			},
		},
	)
}
