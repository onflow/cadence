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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

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

				path, ok := args[2].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				domain := path.Domain.StorageDomain()
				identifier := path.Identifier

				locationRange := EmptyLocationRange

				// Prevent an overwrite

				storageMapKey := interpreter.StringStorageMapKey(identifier)

				if interpreter.StoredValueExists(config, address, domain, storageMapKey) {
					panic(
						interpreter.OverwriteError{
							Address:       interpreter.AddressValue(address),
							Path:          path,
							LocationRange: locationRange,
						},
					)
				}

				value = value.Transfer(
					config,
					locationRange,
					atree.Address(address),
					true,
					nil,
					nil,
					true, // value is standalone because it is from invocation.Arguments[0].
				)

				// Write new value

				config.WriteStored(
					address,
					domain,
					storageMapKey,
					value,
				)

				return interpreter.Void
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

				path, ok := args[1].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				referenceType, ok := typeArgs[0].(*interpreter.ReferenceStaticType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				semaType := interpreter.MustConvertStaticToSemaType(referenceType.ReferencedType, config)

				reference := interpreter.NewStorageReferenceValue(
					config,
					referenceType.Authorization,
					address,
					path,
					semaType,
				)

				// Attempt to dereference,
				// which reads the stored value
				// and performs a dynamic type check

				value := reference.ReferencedValue(config, EmptyLocationRange, true)
				if value == nil {
					return interpreter.Nil
				}

				return interpreter.NewSomeValueNonCopying(config, reference)
			},
		})
}
