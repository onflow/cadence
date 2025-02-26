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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

func NewAccountStorageValue(accountAddress common.Address) *SimpleCompositeValue {
	return &SimpleCompositeValue{
		typeID:     sema.Account_StorageType.ID(),
		staticType: interpreter.PrimitiveStaticTypeAccount_Storage,
		Kind:       common.CompositeKindStructure,
		fields:     map[string]Value{
			// TODO: add the remaining fields
		},
		metadata: map[string]any{
			sema.AccountTypeAddressFieldName: accountAddress,
		},
	}
}

// members

func init() {

	accountStorageTypeName := sema.Account_StorageType.QualifiedIdentifier()

	// Account.Storage.save
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeSaveFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeSaveFunctionType.Parameters),
			Function: func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {
				address := getAddressMetaInfoFromValue(args[0])

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
					atree.Address(address),
					true,
					nil,
				)

				// Write new value

				storageDomain, _ := common.StorageDomainFromIdentifier(domain)
				WriteStored(
					config,
					address,
					storageDomain,
					interpreter.StringStorageMapKey(identifier),
					value,
				)

				return VoidValue{}
			},
		})

	// Account.Storage.borrow
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeBorrowFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeBorrowFunctionType.Parameters),
			Function: func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAddressMetaInfoFromValue(args[0])

				path, ok := args[1].(PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				referenceType, ok := typeArgs[0].(*interpreter.ReferenceStaticType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				reference := NewStorageReferenceValue(
					referenceType.Authorization,
					address,
					path,
					referenceType.ReferencedType,
				)

				// Attempt to dereference,
				// which reads the stored value
				// and performs a dynamic type check

				referenced, err := reference.dereference(config)
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
