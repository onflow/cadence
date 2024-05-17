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
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func NewAuthAccountValue(
	address common.Address,
) *SimpleCompositeValue {
	return &SimpleCompositeValue{
		QualifiedIdentifier: sema.AuthAccountType.QualifiedIdentifier(),
		typeID:              sema.AuthAccountType.ID(),
		staticType:          interpreter.PrimitiveStaticTypeAuthAccount,
		Kind:                common.CompositeKindStructure,
		fields: map[string]Value{
			sema.AuthAccountAddressField: AddressValue(address),
			// TODO: add the remaining fields
		},
	}
}

// members

func init() {
	typeName := interpreter.PrimitiveStaticTypeAuthAccount.String()

	// AuthAccount.link
	RegisterTypeBoundFunction(typeName, sema.AuthAccountLinkField, NativeFunctionValue{
		ParameterCount: len(sema.AuthAccountTypeLinkFunctionType.Parameters),
		Function: func(config *Config, typeArgs []StaticType, args ...Value) Value {
			borrowType, ok := typeArgs[0].(interpreter.ReferenceStaticType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			authAccount, ok := args[0].(*SimpleCompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			address := authAccount.GetMember(config, sema.AuthAccountAddressField)
			addressValue, ok := address.(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			newCapabilityPath, ok := args[1].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			targetPath, ok := args[2].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			newCapabilityDomain := newCapabilityPath.Domain.Identifier()
			newCapabilityIdentifier := newCapabilityPath.Identifier

			//if interpreter.storedValueExists(
			//	address,
			//	newCapabilityDomain,
			//	newCapabilityIdentifier,
			//) {
			//	return Nil
			//}

			// Write new value

			// Note that this will be metered twice if Atree validation is enabled.
			linkValue := NewLinkValue(targetPath, borrowType)

			WriteStored(
				config.MemoryGauge,
				config.Storage,
				common.Address(addressValue),
				newCapabilityDomain,
				newCapabilityIdentifier,
				linkValue,
			)

			return NewSomeValueNonCopying(
				NewCapabilityValue(
					addressValue,
					newCapabilityPath,
					borrowType,
				),
			)
		},
	})

	// AuthAccount.save
	RegisterTypeBoundFunction(typeName, sema.AuthAccountSaveField, NativeFunctionValue{
		ParameterCount: len(sema.AuthAccountTypeSaveFunctionType.Parameters),
		Function: func(config *Config, typeArs []StaticType, args ...Value) Value {
			authAccount, ok := args[0].(*SimpleCompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			address := authAccount.GetMember(config, sema.AuthAccountAddressField)
			addressValue, ok := address.(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			value := args[1]

			path, ok := args[2].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := path.Domain.Identifier()
			identifier := path.Identifier

			// Prevent an overwrite

			//if interpreter.storedValueExists(
			//	address,
			//	domain,
			//	identifier,
			//) {
			//	panic("overwrite error")
			//}

			value = value.Transfer(
				config,
				atree.Address(addressValue),
				true,
				nil,
			)

			// Write new value

			WriteStored(
				config.MemoryGauge,
				config.Storage,
				common.Address(addressValue),
				domain,
				identifier,
				value,
			)

			return VoidValue{}
		},
	})

	// AuthAccount.borrow
	RegisterTypeBoundFunction(typeName, sema.AuthAccountBorrowField, NativeFunctionValue{
		ParameterCount: len(sema.AuthAccountTypeBorrowFunctionType.Parameters),
		Function: func(config *Config, typeArgs []StaticType, args ...Value) Value {
			authAccount, ok := args[0].(*SimpleCompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			path, ok := args[1].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			referenceType, ok := typeArgs[0].(interpreter.ReferenceStaticType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			address := authAccount.GetMember(config, sema.AuthAccountAddressField)
			addressValue, ok := address.(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			reference := NewStorageReferenceValue(
				config.Storage,
				referenceType.Authorized,
				common.Address(addressValue),
				path,
				referenceType.BorrowedType,
			)

			// Attempt to dereference,
			// which reads the stored value
			// and performs a dynamic type check

			referenced, err := reference.dereference(config.MemoryGauge)
			if err != nil {
				panic(err)
			}
			if referenced == nil {
				return NilValue{}
			}

			return NewSomeValueNonCopying(reference)
		},
	})
}
