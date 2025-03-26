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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

//
//import (
//	"github.com/onflow/cadence/bbq"
//	"github.com/onflow/cadence/common"
//	"github.com/onflow/cadence/errors"
//	"github.com/onflow/cadence/interpreter"
//	"github.com/onflow/cadence/sema"
//)
//
//func NewAccountStorageCapabilitiesValue(accountAddress common.Address) *SimpleCompositeValue {
//	return &SimpleCompositeValue{
//		typeID:     sema.Account_StorageCapabilitiesType.ID(),
//		staticType: interpreter.PrimitiveStaticTypeAccount_StorageCapabilities,
//		Kind:       common.CompositeKindStructure,
//		fields:     map[string]Value{
//			// TODO: add the remaining fields
//		},
//		metadata: map[string]any{
//			sema.AccountTypeAddressFieldName: accountAddress,
//		},
//	}
//}
//
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
				accountAddress := getAddressMetaInfoFromValue(args[0])

				// Path argument
				targetPathValue, ok := args[1].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if !ok || targetPathValue.Domain != common.PathDomainStorage {
					panic(errors.NewUnreachableError())
				}

				// Get borrow type type-argument
				ty := typeArguments[0]

				// Issue capability controller and return capability

				return checkAndIssueStorageCapabilityControllerWithType(
					config,
					config.GetAccountHandler(),
					accountAddress,
					targetPathValue,
					ty,
				)
			},
		})
}
