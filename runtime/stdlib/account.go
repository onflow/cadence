/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package stdlib

import (
	"github.com/onflow/cadence/runtime/sema"
)

const authAccountFunctionDocString = `
Creates a new account, paid by the given existing account
`

var authAccountFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Identifier: "payer",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.AuthAccountType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.AuthAccountType,
	),
}

const getAuthAccountDocString = `
Returns the AuthAccount for the given address. Only available in scripts
`

var getAuthAccountFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{{
		Label:          sema.ArgumentLabelNotRequired,
		Identifier:     "address",
		TypeAnnotation: sema.NewTypeAnnotation(&sema.AddressType{}),
	}},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.AuthAccountType),
}

const getAccountFunctionDocString = `
Returns the public account for the given address
`

var getAccountFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "address",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.AddressType{},
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.PublicAccountType,
	),
}

//		NewStandardLibraryFunction(
//			"AuthAccount",
//			authAccountFunctionType,
//			authAccountFunctionDocString,
//			impls.CreateAccount,
//		),
//		NewStandardLibraryFunction(
//			"getAccount",
//			getAccountFunctionType,
//			getAccountFunctionDocString,
//			impls.GetAccount,
//		),

// NewStandardLibraryFunction(
//				"getAuthAccount",
//				getAuthAccountFunctionType,
//				getAuthAccountDocString,
//				r.newGetAuthAccountFunction(context, storage, interpreterOptions, checkerOptions),
//			)
//
//func (r *interpreterRuntime) newAuthAccountValue(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	context Context,
//	storage *Storage,
//	interpreterOptions []interpreter.Option,
//	checkerOptions []sema.Option,
//) interpreter.Value {
//	return interpreter.NewAuthAccountValue(
//		gauge,
//		addressValue,
//		accountBalanceGetFunction(addressValue, context.Interface),
//		accountAvailableBalanceGetFunction(addressValue, context.Interface),
//		storageUsedGetFunction(addressValue, context.Interface, storage),
//		storageCapacityGetFunction(addressValue, context.Interface, storage),
//		r.newAddPublicKeyFunction(gauge, addressValue, context.Interface),
//		r.newRemovePublicKeyFunction(gauge, addressValue, context.Interface),
//		func() interpreter.Value {
//			return r.newAuthAccountContracts(
//				gauge,
//				addressValue,
//				context,
//				storage,
//				interpreterOptions,
//				checkerOptions,
//			)
//		},
//		func() interpreter.Value {
//			return r.newAuthAccountKeys(
//				gauge,
//				addressValue,
//				context.Interface,
//			)
//		},
//	)
//}
//
//
//
//func (r *interpreterRuntime) newGetAuthAccountFunction(
//	context Context,
//	storage *Storage,
//	interpreterOptions []interpreter.Option,
//	checkerOptions []sema.Option,
//) interpreter.HostFunction {
//	return func(invocation interpreter.Invocation) interpreter.Value {
//		accountAddress, ok := invocation.Arguments[0].(interpreter.AddressValue)
//		if !ok {
//			panic(runtimeErrors.NewUnreachableError())
//		}
//
//		return r.newAuthAccountValue(
//			invocation.Interpreter,
//			accountAddress,
//			context,
//			storage,
//			interpreterOptions,
//			checkerOptions,
//		)
//	}
//}
//
//func (r *interpreterRuntime) newGetAccountFunction(runtimeInterface Interface, storage *Storage) interpreter.HostFunction {
//	return func(invocation interpreter.Invocation) interpreter.Value {
//		accountAddress, ok := invocation.Arguments[0].(interpreter.AddressValue)
//		if !ok {
//			panic(runtimeErrors.NewUnreachableError())
//		}
//
//		return r.getPublicAccount(
//			invocation.Interpreter,
//			accountAddress,
//			runtimeInterface,
//			storage,
//		)
//	}
//}
//
//func (r *interpreterRuntime) getPublicAccount(
//	gauge common.MemoryGauge,
//	accountAddress interpreter.AddressValue,
//	runtimeInterface Interface,
//	storage *Storage,
//) interpreter.Value {
//
//	return interpreter.NewPublicAccountValue(
//		gauge,
//		accountAddress,
//		accountBalanceGetFunction(accountAddress, runtimeInterface),
//		accountAvailableBalanceGetFunction(accountAddress, runtimeInterface),
//		storageUsedGetFunction(accountAddress, runtimeInterface, storage),
//		storageCapacityGetFunction(accountAddress, runtimeInterface, storage),
//		func() interpreter.Value {
//			return r.newPublicAccountKeys(gauge, accountAddress, runtimeInterface)
//		},
//		func() interpreter.Value {
//			return r.newPublicAccountContracts(gauge, accountAddress, runtimeInterface)
//		},
//	)
//}
//
//func (r *interpreterRuntime) newAuthAccountContracts(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	context Context,
//	storage *Storage,
//	interpreterOptions []interpreter.Option,
//	checkerOptions []sema.Option,
//) interpreter.Value {
//	return interpreter.NewAuthAccountContractsValue(
//		gauge,
//		addressValue,
//		r.newAuthAccountContractsChangeFunction(
//			gauge,
//			addressValue,
//			context,
//			storage,
//			interpreterOptions,
//			checkerOptions,
//			false,
//		),
//		r.newAuthAccountContractsChangeFunction(
//			gauge,
//			addressValue,
//			context,
//			storage,
//			interpreterOptions,
//			checkerOptions,
//			true,
//		),
//		r.newAccountContractsGetFunction(
//			gauge,
//			addressValue,
//			context.Interface,
//		),
//		r.newAuthAccountContractsRemoveFunction(
//			gauge,
//			addressValue,
//			context.Interface,
//			storage,
//		),
//		r.newAccountContractsGetNamesFunction(
//			addressValue,
//			context.Interface,
//		),
//	)
//}
//
//func (r *interpreterRuntime) newAuthAccountKeys(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) interpreter.Value {
//	return interpreter.NewAuthAccountKeysValue(
//		gauge,
//		addressValue,
//		r.newAccountKeysAddFunction(
//			gauge,
//			addressValue,
//			runtimeInterface,
//		),
//		r.newAccountKeysGetFunction(
//			gauge,
//			addressValue,
//			runtimeInterface,
//		),
//		r.newAccountKeysRevokeFunction(
//			gauge,
//			addressValue,
//			runtimeInterface,
//		),
//	)
//}
//
//func (r *interpreterRuntime) newCreateAccountFunction(
//	context Context,
//	storage *Storage,
//	interpreterOptions []interpreter.Option,
//	checkerOptions []sema.Option,
//) interpreter.HostFunction {
//	return func(invocation interpreter.Invocation) interpreter.Value {
//
//		payer, ok := invocation.Arguments[0].(interpreter.MemberAccessibleValue)
//		if !ok {
//			panic(runtimeErrors.NewUnreachableError())
//		}
//
//		inter := invocation.Interpreter
//		getLocationRange := invocation.GetLocationRange
//
//		invocation.Interpreter.ExpectType(
//			payer,
//			sema.AuthAccountType,
//			getLocationRange,
//		)
//
//		payerAddressValue := payer.GetMember(
//			inter,
//			getLocationRange,
//			sema.AuthAccountAddressField,
//		)
//		if payerAddressValue == nil {
//			panic("address is not set")
//		}
//
//		payerAddress := payerAddressValue.(interpreter.AddressValue).ToAddress()
//
//		addressGetter := func() (address common.Address) {
//			var err error
//			wrapPanic(func() {
//				address, err = context.Interface.CreateAccount(payerAddress)
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			return
//		}
//
//		addressValue := interpreter.NewAddressValueFromConstructor(
//			invocation.Interpreter,
//			addressGetter,
//		)
//
//		r.emitAccountEvent(
//			inter,
//			stdlib.AccountCreatedEventType,
//			context.Interface,
//			[]exportableValue{
//				newExportableValue(addressValue, inter),
//			},
//			getLocationRange,
//		)
//
//		return r.newAuthAccountValue(
//			inter,
//			addressValue,
//			context,
//			storage,
//			interpreterOptions,
//			checkerOptions,
//		)
//	}
//}
//
//func accountBalanceGetFunction(
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) func() interpreter.UFix64Value {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return func() interpreter.UFix64Value {
//		balanceGetter := func() (balance uint64) {
//			var err error
//			wrapPanic(func() {
//				balance, err = runtimeInterface.GetAccountBalance(address)
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			return
//		}
//
//		return interpreter.NewUFix64Value(runtimeInterface, balanceGetter)
//	}
//}
//
//func accountAvailableBalanceGetFunction(
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) func() interpreter.UFix64Value {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return func() interpreter.UFix64Value {
//		balanceGetter := func() (balance uint64) {
//			var err error
//			wrapPanic(func() {
//				balance, err = runtimeInterface.GetAccountAvailableBalance(address)
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			return
//		}
//
//		return interpreter.NewUFix64Value(runtimeInterface, balanceGetter)
//	}
//}
//
//func storageUsedGetFunction(
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//	storage *Storage,
//) func(inter *interpreter.Interpreter) interpreter.UInt64Value {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return func(inter *interpreter.Interpreter) interpreter.UInt64Value {
//
//		// NOTE: flush the cached values, so the host environment
//		// can properly calculate the amount of storage used by the account
//		const commitContractUpdates = false
//		err := storage.Commit(inter, commitContractUpdates)
//		if err != nil {
//			panic(err)
//		}
//
//		return interpreter.NewUInt64Value(
//			inter,
//			func() uint64 {
//				var capacity uint64
//				wrapPanic(func() {
//					capacity, err = runtimeInterface.GetStorageUsed(address)
//				})
//				if err != nil {
//					panic(err)
//				}
//				return capacity
//			},
//		)
//	}
//}
//
//func storageCapacityGetFunction(
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//	storage *Storage,
//) func(inter *interpreter.Interpreter) interpreter.UInt64Value {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return func(inter *interpreter.Interpreter) interpreter.UInt64Value {
//
//		var err error
//
//		// NOTE: flush the cached values, so the host environment
//		// can properly calculate the amount of storage available for the account
//		const commitContractUpdates = false
//		err = storage.Commit(inter, commitContractUpdates)
//		if err != nil {
//			panic(err)
//		}
//
//		return interpreter.NewUInt64Value(
//			inter,
//			func() uint64 {
//				var capacity uint64
//				wrapPanic(func() {
//					capacity, err = runtimeInterface.GetStorageCapacity(address)
//				})
//				if err != nil {
//					panic(err)
//				}
//				return capacity
//			},
//		)
//
//	}
//}
//
//func (r *interpreterRuntime) newAddPublicKeyFunction(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) *interpreter.HostFunctionValue {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return interpreter.NewHostFunctionValue(
//		gauge,
//		func(invocation interpreter.Invocation) interpreter.Value {
//			publicKeyValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//
//			publicKey, err := interpreter.ByteArrayValueToByteSlice(gauge, publicKeyValue)
//			if err != nil {
//				panic("addPublicKey requires the first argument to be a byte array")
//			}
//
//			wrapPanic(func() {
//				err = runtimeInterface.AddEncodedAccountKey(address, publicKey)
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			inter := invocation.Interpreter
//
//			r.emitAccountEvent(
//				gauge,
//				stdlib.AccountKeyAddedEventType,
//				runtimeInterface,
//				[]exportableValue{
//					newExportableValue(addressValue, inter),
//					newExportableValue(publicKeyValue, inter),
//				},
//				invocation.GetLocationRange,
//			)
//
//			return interpreter.VoidValue{}
//		},
//		sema.AuthAccountTypeAddPublicKeyFunctionType,
//	)
//}
//
//func (r *interpreterRuntime) newRemovePublicKeyFunction(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) *interpreter.HostFunctionValue {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return interpreter.NewHostFunctionValue(
//		gauge,
//		func(invocation interpreter.Invocation) interpreter.Value {
//			index, ok := invocation.Arguments[0].(interpreter.IntValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//
//			var publicKey []byte
//			var err error
//			wrapPanic(func() {
//				publicKey, err = runtimeInterface.RevokeEncodedAccountKey(address, index.ToInt())
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			inter := invocation.Interpreter
//
//			publicKeyValue := interpreter.ByteSliceToByteArrayValue(
//				inter,
//				publicKey,
//			)
//
//			r.emitAccountEvent(
//				gauge,
//				stdlib.AccountKeyRemovedEventType,
//				runtimeInterface,
//				[]exportableValue{
//					newExportableValue(addressValue, inter),
//					newExportableValue(publicKeyValue, inter),
//				},
//				invocation.GetLocationRange,
//			)
//
//			return interpreter.VoidValue{}
//		},
//		sema.AuthAccountTypeRemovePublicKeyFunctionType,
//	)
//}
//
//func (r *interpreterRuntime) newAccountKeysAddFunction(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) *interpreter.HostFunctionValue {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return interpreter.NewHostFunctionValue(
//		gauge,
//		func(invocation interpreter.Invocation) interpreter.Value {
//			publicKeyValue, ok := invocation.Arguments[0].(*interpreter.CompositeValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//
//			inter := invocation.Interpreter
//			getLocationRange := invocation.GetLocationRange
//
//			publicKey, err := NewPublicKeyFromValue(inter, getLocationRange, publicKeyValue)
//			if err != nil {
//				panic(err)
//			}
//
//			hashAlgo := NewHashAlgorithmFromValue(inter, getLocationRange, invocation.Arguments[1])
//			weightValue, ok := invocation.Arguments[2].(interpreter.UFix64Value)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//			weight := weightValue.ToInt()
//
//			var accountKey *AccountKey
//			wrapPanic(func() {
//				accountKey, err = runtimeInterface.AddAccountKey(address, publicKey, hashAlgo, weight)
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			r.emitAccountEvent(
//				inter,
//				stdlib.AccountKeyAddedEventType,
//				runtimeInterface,
//				[]exportableValue{
//					newExportableValue(addressValue, inter),
//					newExportableValue(publicKeyValue, inter),
//				},
//				invocation.GetLocationRange,
//			)
//
//			return NewAccountKeyValue(
//				inter,
//				getLocationRange,
//				accountKey,
//				inter.PublicKeyValidationHandler,
//			)
//		},
//		sema.AuthAccountKeysTypeAddFunctionType,
//	)
//}
//
//func (r *interpreterRuntime) newAccountKeysGetFunction(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) *interpreter.HostFunctionValue {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return interpreter.NewHostFunctionValue(
//		gauge,
//		func(invocation interpreter.Invocation) interpreter.Value {
//			indexValue, ok := invocation.Arguments[0].(interpreter.IntValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//			index := indexValue.ToInt()
//
//			var err error
//			var accountKey *AccountKey
//			wrapPanic(func() {
//				accountKey, err = runtimeInterface.GetAccountKey(address, index)
//			})
//
//			if err != nil {
//				panic(err)
//			}
//
//			// Here it is expected the host function to return a nil key, if a key is not found at the given index.
//			// This is done because, if the host function returns an error when a key is not found, then
//			// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
//			if accountKey == nil {
//				return interpreter.NewNilValue(invocation.Interpreter)
//			}
//
//			inter := invocation.Interpreter
//
//			return interpreter.NewSomeValueNonCopying(
//				inter,
//				NewAccountKeyValue(
//					inter,
//					invocation.GetLocationRange,
//					accountKey,
//					DoNotValidatePublicKey, // key from FVM has already been validated
//				),
//			)
//		},
//		sema.AccountKeysTypeGetFunctionType,
//	)
//}
//
//func (r *interpreterRuntime) newAccountKeysRevokeFunction(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) *interpreter.HostFunctionValue {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return interpreter.NewHostFunctionValue(
//		gauge,
//		func(invocation interpreter.Invocation) interpreter.Value {
//			indexValue, ok := invocation.Arguments[0].(interpreter.IntValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//			index := indexValue.ToInt()
//
//			var err error
//			var accountKey *AccountKey
//			wrapPanic(func() {
//				accountKey, err = runtimeInterface.RevokeAccountKey(address, index)
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			// Here it is expected the host function to return a nil key, if a key is not found at the given index.
//			// This is done because, if the host function returns an error when a key is not found, then
//			// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
//			if accountKey == nil {
//				return interpreter.NewNilValue(invocation.Interpreter)
//			}
//
//			inter := invocation.Interpreter
//
//			r.emitAccountEvent(
//				inter,
//				stdlib.AccountKeyRemovedEventType,
//				runtimeInterface,
//				[]exportableValue{
//					newExportableValue(addressValue, inter),
//					newExportableValue(indexValue, inter),
//				},
//				invocation.GetLocationRange,
//			)
//
//			return interpreter.NewSomeValueNonCopying(
//				inter,
//				NewAccountKeyValue(
//					inter,
//					invocation.GetLocationRange,
//					accountKey,
//					DoNotValidatePublicKey, // key from FVM has already been validated
//				),
//			)
//		},
//		sema.AuthAccountKeysTypeRevokeFunctionType,
//	)
//}
//
//func (r *interpreterRuntime) newPublicAccountKeys(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) interpreter.Value {
//	return interpreter.NewPublicAccountKeysValue(
//		gauge,
//		addressValue,
//		r.newAccountKeysGetFunction(
//			gauge,
//			addressValue,
//			runtimeInterface,
//		),
//	)
//}
//
//func (r *interpreterRuntime) newPublicAccountContracts(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) interpreter.Value {
//	return interpreter.NewPublicAccountContractsValue(
//		gauge,
//		addressValue,
//		r.newAccountContractsGetFunction(
//			gauge,
//			addressValue,
//			runtimeInterface,
//		),
//		r.newAccountContractsGetNamesFunction(
//			addressValue,
//			runtimeInterface,
//		),
//	)
//}
//
//func (r *interpreterRuntime) newAccountContractsGetNamesFunction(
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//) func(
//	inter *interpreter.Interpreter,
//	getLocationRange func() interpreter.LocationRange,
//) *interpreter.ArrayValue {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return func(
//		inter *interpreter.Interpreter,
//		getLocationRange func() interpreter.LocationRange,
//	) *interpreter.ArrayValue {
//		var names []string
//		var err error
//		wrapPanic(func() {
//			names, err = runtimeInterface.GetAccountContractNames(address)
//		})
//		if err != nil {
//			panic(err)
//		}
//
//		values := make([]interpreter.Value, len(names))
//		for i, name := range names {
//			memoryUsage := common.NewStringMemoryUsage(len(name))
//			values[i] = interpreter.NewStringValue(
//				inter,
//				memoryUsage,
//				func() string {
//					return name
//				},
//			)
//		}
//
//		arrayType := interpreter.NewVariableSizedStaticType(
//			inter,
//			interpreter.NewPrimitiveStaticType(
//				inter,
//				interpreter.PrimitiveStaticTypeString,
//			),
//		)
//
//		return interpreter.NewArrayValue(
//			inter,
//			getLocationRange,
//			arrayType,
//			common.Address{},
//			values...,
//		)
//	}
//}
