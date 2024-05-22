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

package vm

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func NewPublicAccountValue(
	address common.Address,
) *SimpleCompositeValue {
	return &SimpleCompositeValue{
		typeID:     sema.PublicAccountType.ID(),
		staticType: interpreter.PrimitiveStaticTypePublicAccount,
		Kind:       common.CompositeKindStructure,
		fields: map[string]Value{
			sema.PublicAccountAddressField: AddressValue(address),
			// TODO: add the remaining fields
		},
	}
}

// members

func init() {
	typeName := interpreter.PrimitiveStaticTypePublicAccount.String()

	// PublicAccount.getCapability
	RegisterTypeBoundFunction(typeName, sema.PublicAccountGetCapabilityField, NativeFunctionValue{
		ParameterCount: len(sema.PublicAccountTypeGetCapabilityFunctionType.Parameters),
		Function: func(config *Config, typeArguments []StaticType, args ...Value) Value {
			// Get address field from the receiver (PublicAccount)
			authAccount, ok := args[0].(*SimpleCompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			address := authAccount.GetMember(config, sema.PublicAccountAddressField)
			addressValue, ok := address.(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Path argument
			path, ok := args[1].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			//pathStaticType := path.StaticType(config.MemoryGauge)
			//
			//if !IsSubType(pathStaticType, pathType) {
			//	panic(fmt.Errorf("type mismatch"))
			//}

			// NOTE: the type parameter is optional, for backwards compatibility

			var borrowType *interpreter.ReferenceStaticType
			if len(typeArguments) > 0 {
				ty := typeArguments[1]
				// we handle the nil case for this below
				borrowType, _ = ty.(*interpreter.ReferenceStaticType)
			}

			return NewCapabilityValue(addressValue, path, borrowType)
		},
	})
}
