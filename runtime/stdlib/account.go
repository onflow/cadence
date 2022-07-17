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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
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

type EventEmitter interface {
	EmitEvent(
		inter *interpreter.Interpreter,
		eventType *sema.CompositeType,
		values []interpreter.Value,
		getLocationRange func() interpreter.LocationRange,
	) error
}

type AuthAccountHandler interface {
}

type AccountCreator interface {
	EventEmitter
	AuthAccountHandler
	// CreateAccount creates a new account.
	CreateAccount(payer common.Address) (address common.Address, err error)
}

func NewAuthAccountConstructor(creator AccountCreator) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"AuthAccount",
		authAccountFunctionType,
		authAccountFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {

			payer, ok := invocation.Arguments[0].(interpreter.MemberAccessibleValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			getLocationRange := invocation.GetLocationRange

			inter.ExpectType(
				payer,
				sema.AuthAccountType,
				getLocationRange,
			)

			payerValue := payer.GetMember(
				inter,
				getLocationRange,
				sema.AuthAccountAddressField,
			)
			if payerValue == nil {
				panic(errors.NewUnexpectedError("payer address is not set"))
			}

			payerAddressValue, ok := payerValue.(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnexpectedError("payer address is not address"))
			}

			payerAddress := payerAddressValue.ToAddress()

			addressValue := interpreter.NewAddressValueFromConstructor(
				inter,
				func() (address common.Address) {
					var err error
					wrapPanic(func() {
						address, err = creator.CreateAccount(payerAddress)
					})
					if err != nil {
						panic(err)
					}

					return
				},
			)

			err := creator.EmitEvent(
				inter,
				AccountCreatedEventType,
				[]interpreter.Value{addressValue},
				getLocationRange,
			)
			if err != nil {
				panic(err)
			}

			return NewAuthAccountValue(
				inter,
				creator,
				addressValue,
			)
		})
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

func NewGetAuthAccountFunction(handler AuthAccountHandler) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"getAuthAccount",
		getAuthAccountFunctionType,
		getAuthAccountDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			accountAddress, ok := invocation.Arguments[0].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return NewAuthAccountValue(
				invocation.Interpreter,
				handler,
				accountAddress,
			)
		})
}

func NewAuthAccountValue(
	gauge common.MemoryGauge,
	handler AuthAccountHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	// TODO:
	return nil
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
	//			return newAuthAccountContractsValue(
	//				gauge,
	//				addressValue,
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
}

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
//
//
//func (r *interpreterRuntime) newAccountContractsGetFunction(
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
//			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//			name := nameValue.Str
//
//			var code []byte
//			var err error
//			wrapPanic(func() {
//				code, err = runtimeInterface.GetAccountContractCode(address, name)
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			if len(code) > 0 {
//				return interpreter.NewSomeValueNonCopying(
//					invocation.Interpreter,
//					interpreter.NewDeployedContractValue(
//						invocation.Interpreter,
//						addressValue,
//						nameValue,
//						interpreter.ByteSliceToByteArrayValue(
//							invocation.Interpreter,
//							code,
//						),
//					),
//				)
//			} else {
//				return interpreter.NewNilValue(invocation.Interpreter)
//			}
//		},
//		sema.AuthAccountContractsTypeGetFunctionType,
//	)
//}
//
//// newAuthAccountContractsChangeFunction called when e.g.
//// - adding: `AuthAccount.contracts.add(name: "Foo", code: [...])` (isUpdate = false)
//// - updating: `AuthAccount.contracts.update__experimental(name: "Foo", code: [...])` (isUpdate = true)
////
//func (r *interpreterRuntime) newAuthAccountContractsChangeFunction(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	startContext Context,
//	storage *Storage,
//	interpreterOptions []interpreter.Option,
//	checkerOptions []sema.Option,
//	isUpdate bool,
//) *interpreter.HostFunctionValue {
//	return interpreter.NewHostFunctionValue(
//		gauge,
//		func(invocation interpreter.Invocation) interpreter.Value {
//
//			const requiredArgumentCount = 2
//
//			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//
//			newCodeValue, ok := invocation.Arguments[1].(*interpreter.ArrayValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//
//			constructorArguments := invocation.Arguments[requiredArgumentCount:]
//			constructorArgumentTypes := invocation.ArgumentTypes[requiredArgumentCount:]
//
//			code, err := interpreter.ByteArrayValueToByteSlice(gauge, newCodeValue)
//			if err != nil {
//				panic(runtimeErrors.NewDefaultUserError("add requires the second argument to be an array"))
//			}
//
//			// Get the existing code
//
//			nameArgument := nameValue.Str
//
//			if nameArgument == "" {
//				panic(runtimeErrors.NewDefaultUserError(
//					"contract name argument cannot be empty." +
//						"it must match the name of the deployed contract declaration or contract interface declaration",
//				))
//			}
//
//			address := addressValue.ToAddress()
//			existingCode, err := startContext.Interface.GetAccountContractCode(address, nameArgument)
//			if err != nil {
//				panic(err)
//			}
//
//			if isUpdate {
//				// We are updating an existing contract.
//				// Ensure that there's a contract/contract-interface with the given name exists already
//
//				if len(existingCode) == 0 {
//					panic(runtimeErrors.NewDefaultUserError(
//						"cannot update non-existing contract with name %q in account %s",
//						nameArgument,
//						address.ShortHexWithPrefix(),
//					))
//				}
//
//			} else {
//				// We are adding a new contract.
//				// Ensure that no contract/contract interface with the given name exists already
//
//				if len(existingCode) > 0 {
//					panic(runtimeErrors.NewDefaultUserError(
//						"cannot overwrite existing contract with name %q in account %s",
//						nameArgument,
//						address.ShortHexWithPrefix(),
//					))
//				}
//			}
//
//			// Check the code
//
//			location := common.NewAddressLocation(invocation.Interpreter, address, nameArgument)
//
//			context := startContext.WithLocation(location)
//
//			handleContractUpdateError := func(err error) {
//				if err == nil {
//					return
//				}
//
//				// Update the code for the error pretty printing
//				// NOTE: only do this when an error occurs
//
//				context.SetCode(context.Location, code)
//
//				panic(&InvalidContractDeploymentError{
//					Err:           err,
//					LocationRange: invocation.GetLocationRange(),
//				})
//			}
//
//			// NOTE: do NOT use the program obtained from the host environment, as the current program.
//			// Always re-parse and re-check the new program.
//
//			// NOTE: *DO NOT* store the program – the new or updated program
//			// should not be effective during the execution
//
//			const storeProgram = false
//
//			program, err := r.parseAndCheckProgram(
//				code,
//				context,
//				checkerOptions,
//				storeProgram,
//				importResolutionResults{},
//			)
//			if err != nil {
//				// Update the code for the error pretty printing
//				// NOTE: only do this when an error occurs
//
//				context.SetCode(context.Location, code)
//
//				panic(&InvalidContractDeploymentError{
//					Err:           err,
//					LocationRange: invocation.GetLocationRange(),
//				})
//			}
//
//			// The code may declare exactly one contract or one contract interface.
//
//			var contractTypes []*sema.CompositeType
//			var contractInterfaceTypes []*sema.InterfaceType
//
//			program.Elaboration.GlobalTypes.Foreach(func(_ string, variable *sema.Variable) {
//				switch ty := variable.Type.(type) {
//				case *sema.CompositeType:
//					if ty.Kind == common.CompositeKindContract {
//						contractTypes = append(contractTypes, ty)
//					}
//
//				case *sema.InterfaceType:
//					if ty.CompositeKind == common.CompositeKindContract {
//						contractInterfaceTypes = append(contractInterfaceTypes, ty)
//					}
//				}
//			})
//
//			var deployedType sema.Type
//			var contractType *sema.CompositeType
//			var contractInterfaceType *sema.InterfaceType
//			var declaredName string
//			var declarationKind common.DeclarationKind
//
//			switch {
//			case len(contractTypes) == 1 && len(contractInterfaceTypes) == 0:
//				contractType = contractTypes[0]
//				declaredName = contractType.Identifier
//				deployedType = contractType
//				declarationKind = common.DeclarationKindContract
//			case len(contractInterfaceTypes) == 1 && len(contractTypes) == 0:
//				contractInterfaceType = contractInterfaceTypes[0]
//				declaredName = contractInterfaceType.Identifier
//				deployedType = contractInterfaceType
//				declarationKind = common.DeclarationKindContractInterface
//			}
//
//			if deployedType == nil {
//				// Update the code for the error pretty printing
//				// NOTE: only do this when an error occurs
//
//				context.SetCode(context.Location, code)
//
//				panic(runtimeErrors.NewDefaultUserError(
//					"invalid %s: the code must declare exactly one contract or contract interface",
//					declarationKind.Name(),
//				))
//			}
//
//			// The declared contract or contract interface must have the name
//			// passed to the constructor as the first argument
//
//			if declaredName != nameArgument {
//				// Update the code for the error pretty printing
//				// NOTE: only do this when an error occurs
//
//				context.SetCode(context.Location, code)
//
//				panic(runtimeErrors.NewDefaultUserError(
//					"invalid %s: the name argument must match the name of the declaration: got %q, expected %q",
//					declarationKind.Name(),
//					nameArgument,
//					declaredName,
//				))
//			}
//
//			// Validate the contract update (if enabled)
//
//			if r.contractUpdateValidationEnabled && isUpdate {
//
//				oldCode, err := r.getCode(context)
//				handleContractUpdateError(err)
//
//				oldProgram, err := parser.ParseProgram(string(oldCode), gauge)
//
//				if !ignoreUpdatedProgramParserError(err) {
//					handleContractUpdateError(err)
//				}
//
//				validator := NewContractUpdateValidator(
//					context.Location,
//					nameArgument,
//					oldProgram,
//					program.Program,
//				)
//				err = validator.Validate()
//				handleContractUpdateError(err)
//			}
//
//			inter := invocation.Interpreter
//
//			err = r.updateAccountContractCode(
//				program,
//				context,
//				storage,
//				declaredName,
//				code,
//				addressValue,
//				contractType,
//				constructorArguments,
//				constructorArgumentTypes,
//				interpreterOptions,
//				checkerOptions,
//				updateAccountContractCodeOptions{
//					createContract: !isUpdate,
//				},
//			)
//			if err != nil {
//				// Update the code for the error pretty printing
//				// NOTE: only do this when an error occurs
//
//				context.SetCode(context.Location, code)
//
//				panic(err)
//			}
//
//			codeHashValue := CodeToHashValue(inter, code)
//
//			eventArguments := []exportableValue{
//				newExportableValue(addressValue, inter),
//				newExportableValue(codeHashValue, inter),
//				newExportableValue(nameValue, inter),
//			}
//
//			var eventType *sema.CompositeType
//
//			if isUpdate {
//				eventType = stdlib.AccountContractUpdatedEventType
//			} else {
//				eventType = stdlib.AccountContractAddedEventType
//			}
//
//			emitEventFields(
//				inter,
//				invocation.GetLocationRange,
//				eventType,
//				eventArguments,
//				startContext.Interface.EmitEvent,
//			)
//
//			return interpreter.NewDeployedContractValue(
//				inter,
//				addressValue,
//				nameValue,
//				newCodeValue,
//			)
//		},
//		sema.AuthAccountContractsTypeAddFunctionType,
//	)
//}

//func (r *interpreterRuntime) newAuthAccountContractsRemoveFunction(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//	runtimeInterface Interface,
//	storage *Storage,
//) *interpreter.HostFunctionValue {
//
//	// Converted addresses can be cached and don't have to be recomputed on each function invocation
//	address := addressValue.ToAddress()
//
//	return interpreter.NewHostFunctionValue(
//		gauge,
//		func(invocation interpreter.Invocation) interpreter.Value {
//
//			inter := invocation.Interpreter
//			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
//			if !ok {
//				panic(runtimeErrors.NewUnreachableError())
//			}
//			name := nameValue.Str
//
//			// Get the current code
//
//			var code []byte
//			var err error
//			wrapPanic(func() {
//				code, err = runtimeInterface.GetAccountContractCode(address, name)
//			})
//			if err != nil {
//				panic(err)
//			}
//
//			// Only remove the contract code, remove the contract value, and emit an event,
//			// if there is currently code deployed for the given contract name
//
//			if len(code) > 0 {
//
//				// NOTE: *DO NOT* call SetProgram – the program removal
//				// should not be effective during the execution, only after
//
//				// Deny removing a contract, if the contract validation is enabled, and
//				// the existing code contains enums.
//				if r.contractUpdateValidationEnabled {
//					existingProgram, err := parser.ParseProgram(string(code), gauge)
//
//					// If the existing code is not parsable (i.e: `err != nil`), that shouldn't be a reason to
//					// fail the contract removal. Therefore, validate only if the code is a valid one.
//					if err == nil && containsEnumsInProgram(existingProgram) {
//						panic(&ContractRemovalError{
//							Name:          name,
//							LocationRange: invocation.GetLocationRange(),
//						})
//					}
//				}
//
//				wrapPanic(func() {
//					err = runtimeInterface.RemoveAccountContractCode(address, name)
//				})
//				if err != nil {
//					panic(err)
//				}
//
//				// NOTE: the contract recording function delays the write
//				// until the end of the execution of the program
//
//				storage.recordContractUpdate(
//					addressValue.ToAddress(),
//					name,
//					nil,
//				)
//
//				codeHashValue := CodeToHashValue(inter, code)
//
//				emitEventFields(
//					inter,
//					invocation.GetLocationRange,
//					stdlib.AccountContractRemovedEventType,
//					[]exportableValue{
//						newExportableValue(addressValue, inter),
//						newExportableValue(codeHashValue, inter),
//						newExportableValue(nameValue, inter),
//					},
//					runtimeInterface.EmitEvent,
//				)
//
//				return interpreter.NewSomeValueNonCopying(
//					inter,
//					interpreter.NewDeployedContractValue(
//						inter,
//						addressValue,
//						nameValue,
//						interpreter.ByteSliceToByteArrayValue(
//							inter,
//							code,
//						),
//					),
//				)
//			} else {
//				return interpreter.NewNilValue(invocation.Interpreter)
//			}
//		},
//		sema.AuthAccountContractsTypeRemoveFunctionType,
//	)
//}

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

type PublicAccountHandler interface {
}

func NewGetAccountFunction(handler PublicAccountHandler) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"getAccount",
		getAccountFunctionType,
		getAccountFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			accountAddress, ok := invocation.Arguments[0].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return NewPublicAccountValue(
				invocation.Interpreter,
				handler,
				accountAddress,
			)
		},
	)
}

func NewPublicAccountValue(
	inter *interpreter.Interpreter,
	handler PublicAccountHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	// TODO:
	return nil
	//return interpreter.NewPublicAccountValue(
	////gauge,
	////accountAddress,
	////accountBalanceGetFunction(accountAddress, runtimeInterface),
	////accountAvailableBalanceGetFunction(accountAddress, runtimeInterface),
	////storageUsedGetFunction(accountAddress, runtimeInterface, storage),
	////storageCapacityGetFunction(accountAddress, runtimeInterface, storage),
	////func() interpreter.Value {
	////	return r.newPublicAccountKeys(gauge, accountAddress, runtimeInterface)
	////},
	////func() interpreter.Value {
	////	return r.newPublicAccountContracts(gauge, accountAddress, runtimeInterface)
	////},
	//)
}
