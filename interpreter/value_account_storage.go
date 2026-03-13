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

package interpreter

import (
	"fmt"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

// Account.Storage

var account_StorageTypeID = sema.Account_StorageType.ID()
var account_StorageStaticType StaticType = PrimitiveStaticTypeAccount_Storage
var account_StorageFieldNames []string = nil

// NewAccountStorageValue constructs an Account.Storage value.
// When allowedPaths is nil, all paths are accessible (unlimited storage).
// When allowedPaths is non-nil, only the specified paths are accessible.
func NewAccountStorageValue(
	gauge common.MemoryGauge,
	address AddressValue,
	storageUsedGet func(context MemberAccessibleContext) UInt64Value,
	storageCapacityGet func(context MemberAccessibleContext) UInt64Value,
	allowedPaths map[PathValue]struct{},
) Value {

	var storageValue *SimpleCompositeValue

	methods := map[string]FunctionValue{}

	computeLazyStoredMethod := func(name string, context MemberAccessibleContext) FunctionValue {
		switch name {
		case sema.Account_StorageTypeForEachPublicFunctionName:
			return newStorageIterationFunction(
				context,
				storageValue,
				sema.Account_StorageTypeForEachPublicFunctionType,
				address,
				common.PathDomainPublic,
				sema.PublicPathType,
				allowedPaths,
			)

		case sema.Account_StorageTypeForEachStoredFunctionName:
			return newStorageIterationFunction(
				context,
				storageValue,
				sema.Account_StorageTypeForEachStoredFunctionType,
				address,
				common.PathDomainStorage,
				sema.StoragePathType,
				allowedPaths,
			)

		case sema.Account_StorageTypeTypeFunctionName:
			return authAccountStorageTypeFunction(context, storageValue, address, allowedPaths)

		case sema.Account_StorageTypeLoadFunctionName:
			return authAccountStorageLoadFunction(context, storageValue, address, allowedPaths)

		case sema.Account_StorageTypeCopyFunctionName:
			return authAccountStorageCopyFunction(context, storageValue, address, allowedPaths)

		case sema.Account_StorageTypeSaveFunctionName:
			return authAccountStorageSaveFunction(context, storageValue, address, allowedPaths)

		case sema.Account_StorageTypeBorrowFunctionName:
			return authAccountStorageBorrowFunction(context, storageValue, address, allowedPaths)

		case sema.Account_StorageTypeCheckFunctionName:
			return authAccountStorageCheckFunction(context, storageValue, address, allowedPaths)

		case sema.Account_StorageTypeLimitedToPathsFunctionName:
			return accountStorageLimitedToPathsFunction(
				context,
				storageValue,
				address,
				storageUsedGet,
				storageCapacityGet,
				allowedPaths,
			)
		}

		return nil
	}

	computeField := func(name string, context MemberAccessibleContext) Value {
		switch name {
		case sema.Account_StorageTypePublicPathsFieldName:
			return publicAccountPaths(context, address, allowedPaths)

		case sema.Account_StorageTypeStoragePathsFieldName:
			return storageAccountPaths(context, address, allowedPaths)

		case sema.Account_StorageTypeUsedFieldName:
			return storageUsedGet(context)

		case sema.Account_StorageTypeCapacityFieldName:
			return storageCapacityGet(context)
		}

		return nil
	}

	methodsGetter := func(name string, context MemberAccessibleContext) FunctionValue {
		method, ok := methods[name]
		if !ok {
			method = computeLazyStoredMethod(name, context)
			if method != nil {
				methods[name] = method
			}
		}

		return method
	}

	var str string
	stringer := func(context ValueStringContext, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(context, common.AccountStorageStringMemoryUsage)
			addressStr := address.MeteredString(context, seenReferences)
			str = fmt.Sprintf("Account.Storage(%s)", addressStr)
		}
		return str
	}

	storageValue = NewSimpleCompositeValue(
		gauge,
		account_StorageTypeID,
		account_StorageStaticType,
		account_StorageFieldNames,
		// No fields, only computed fields, and methods.
		nil,
		computeField,
		methodsGetter,
		nil,
		stringer,
	).WithPrivateField(AccountTypePrivateAddressFieldName, address)

	if allowedPaths != nil {
		storageValue = storageValue.WithPrivateField(
			accountStoragePrivateAllowedPathsFieldName,
			allowedPaths,
		)
	}

	return storageValue
}

// isPathAllowed returns true if allowedPaths is nil (unlimited) or the path is in the set.
func isPathAllowed(path PathValue, allowedPaths map[PathValue]struct{}) bool {
	if allowedPaths == nil {
		return true
	}
	_, ok := allowedPaths[path]
	return ok
}

const accountStoragePrivateAllowedPathsFieldName = "allowedPaths"

// getAllowedPathsFromReceiver extracts the allowed paths set
// from a receiver's private field.
// Returns nil if no restriction is set.
func getAllowedPathsFromReceiver(receiver Value) map[PathValue]struct{} {
	composite, ok := receiver.(*SimpleCompositeValue)
	if !ok {
		return nil
	}

	field := composite.PrivateField(accountStoragePrivateAllowedPathsFieldName)
	if field == nil {
		return nil
	}

	allowedPaths, ok := field.(map[PathValue]struct{})
	if !ok {
		return nil
	}

	return allowedPaths
}

// getEffectiveAllowedPaths returns the allowed paths for a storage operation.
// If closureAllowedPaths is non-nil (interpreter path), it is used directly.
// Otherwise, the allowed paths are extracted from the receiver's private field (BBQ VM path).
func getEffectiveAllowedPaths(receiver Value, closureAllowedPaths map[PathValue]struct{}) map[PathValue]struct{} {
	if closureAllowedPaths != nil {
		return closureAllowedPaths
	}
	return getAllowedPathsFromReceiver(receiver)
}

func accountStorageLimitedToPathsFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	address AddressValue,
	storageUsedGet func(context MemberAccessibleContext) UInt64Value,
	storageCapacityGet func(context MemberAccessibleContext) UInt64Value,
	existingAllowedPaths map[PathValue]struct{},
) BoundFunctionValue {

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		sema.Account_StorageTypeLimitedToPathsFunctionType,
		NativeAccountStorageLimitedToPathsFunction(&address, existingAllowedPaths),
	)
}

func NativeAccountStorageLimitedToPathsFunction(
	addressPointer *AddressValue,
	allowedPaths map[PathValue]struct{},
) NativeFunction {
	return func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		address := GetAddressValue(receiver, addressPointer)
		existingAllowedPaths := getEffectiveAllowedPaths(receiver, allowedPaths)

		pathsArray, ok := args[0].(*ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		newAllowedPaths := make(map[PathValue]struct{})
		pathsArray.Iterate(
			context,
			func(element Value) (resume bool) {
				pathValue, ok := element.(PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				if !isPathAllowed(pathValue, existingAllowedPaths) {
					return true
				}
				newAllowedPaths[pathValue] = struct{}{}
				return true
			},
			false,
		)

		limitedStorageValue := NewAccountStorageValue(
			context,
			address,
			nil,
			nil,
			newAllowedPaths,
		)

		authorization := NewEntitlementSetAuthorization(
			context,
			func() []common.TypeID {
				return []common.TypeID{
					sema.StorageType.ID(),
				}
			},
			1,
			sema.Conjunction,
		)

		return NewEphemeralReferenceValue(
			context,
			authorization,
			limitedStorageValue,
			sema.Account_StorageType,
		)
	}
}
