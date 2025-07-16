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
	// Any member methods goes here
	deployedContractTypeName := commons.TypeQualifier(sema.DeployedContractType)

	// Methods on `DeployedContract` value.

	registerBuiltinTypeBoundFunction(
		deployedContractTypeName,
		NewNativeFunctionValue(
			sema.DeployedContractTypePublicTypesFunctionName,
			sema.DeployedContractTypePublicTypesFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				// arg[0] is the receiver. Actual arguments starts from 1.
				deployedContract, args := SplitTypedReceiverAndArgs[*interpreter.SimpleCompositeValue](context, args) // nolint:ineffassign

				addressFieldValue := deployedContract.GetMember(
					context,
					EmptyLocationRange,
					sema.DeployedContractTypeAddressFieldName,
				)
				addressValue, ok := addressFieldValue.(interpreter.AddressValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				nameFieldValue := deployedContract.GetMember(
					context,
					EmptyLocationRange,
					sema.DeployedContractTypeNameFieldName,
				)

				nameValue, ok := nameFieldValue.(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return interpreter.DeployedContractPublicTypes(
					context,
					common.Address(addressValue),
					nameValue,
				)
			},
		),
	)
}
