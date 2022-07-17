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
	BalanceProvider
	AvailableBalanceProvider
	StorageUsedProvider
	StorageCapacityProvider
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
		},
	)
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

			gauge := invocation.Interpreter

			return NewAuthAccountValue(
				gauge,
				handler,
				accountAddress,
			)
		},
	)
}

func NewAuthAccountValue(
	gauge common.MemoryGauge,
	handler AuthAccountHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAuthAccountValue(
		gauge,
		addressValue,
		newAccountBalanceGetFunction(gauge, handler, addressValue),
		newAccountAvailableBalanceGetFunction(gauge, handler, addressValue),
		newStorageUsedGetFunction(handler, addressValue),
		newStorageCapacityGetFunction(handler, addressValue),
		// TODO:
		nil,
		nil,
		nil,
		nil,
		//newAddPublicKeyFunction(gauge, handler, addressValue),
		//newRemovePublicKeyFunction(gauge, handler, addressValue),
		//func() interpreter.Value {
		//	return newAuthAccountContractsValue(
		//		gauge,
		//		handler,
		//		addressValue,
		//	)
		//},
		//func() interpreter.Value {
		//	return newAuthAccountKeysValjue(
		//		gauge,
		//		handler,
		//		addressValue,
		//	)
		//},
	)
}

//func newAuthAccountContracts(
//	gauge common.MemoryGauge,
//	addressValue interpreter.AddressValue,
//) interpreter.Value {
//	return interpreter.NewAuthAccountContractsValue(
//		gauge,
//		addressValue,
//		newAuthAccountContractsChangeFunction(
//			gauge,
//          handler,
//			addressValue,
//			false,
//		),
//		newAuthAccountContractsChangeFunction(
//			gauge,
//          handler,
//			addressValue,
//			true,
//		),
//		newAccountContractsGetFunction(
//			gauge,
//          handler,
//			addressValue,
//		),
//		newAuthAccountContractsRemoveFunction(
//			gauge,
//          handler,
//			addressValue,
//		),
//		r.newAccountContractsGetNamesFunction(
//          handler,
//			addressValue,
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

type BalanceProvider interface {
	// GetAccountBalance gets accounts default flow token balance.
	GetAccountBalance(address common.Address) (uint64, error)
}

func newAccountBalanceGetFunction(
	gauge common.MemoryGauge,
	provider BalanceProvider,
	addressValue interpreter.AddressValue,
) func() interpreter.UFix64Value {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func() interpreter.UFix64Value {
		return interpreter.NewUFix64Value(gauge, func() (balance uint64) {
			var err error
			wrapPanic(func() {
				balance, err = provider.GetAccountBalance(address)
			})
			if err != nil {
				panic(err)
			}

			return
		})
	}
}

type AvailableBalanceProvider interface {
	// GetAccountAvailableBalance gets accounts default flow token balance - balance that is reserved for storage.
	GetAccountAvailableBalance(address common.Address) (uint64, error)
}

func newAccountAvailableBalanceGetFunction(
	gauge common.MemoryGauge,
	provider AvailableBalanceProvider,
	addressValue interpreter.AddressValue,
) func() interpreter.UFix64Value {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func() interpreter.UFix64Value {
		return interpreter.NewUFix64Value(gauge, func() (balance uint64) {
			var err error
			wrapPanic(func() {
				balance, err = provider.GetAccountAvailableBalance(address)
			})
			if err != nil {
				panic(err)
			}

			return
		})
	}
}

type StorageUsedProvider interface {
	CommitStorage(inter *interpreter.Interpreter, commitContractUpdates bool) error
	// GetStorageUsed gets storage used in bytes by the address at the moment of the function call.
	GetStorageUsed(address common.Address) (uint64, error)
}

func newStorageUsedGetFunction(
	provider StorageUsedProvider,
	addressValue interpreter.AddressValue,
) func(inter *interpreter.Interpreter) interpreter.UInt64Value {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func(inter *interpreter.Interpreter) interpreter.UInt64Value {

		// NOTE: flush the cached values, so the host environment
		// can properly calculate the amount of storage used by the account
		const commitContractUpdates = false
		err := provider.CommitStorage(inter, commitContractUpdates)
		if err != nil {
			panic(err)
		}

		return interpreter.NewUInt64Value(
			inter,
			func() uint64 {
				var capacity uint64
				wrapPanic(func() {
					capacity, err = provider.GetStorageUsed(address)
				})
				if err != nil {
					panic(err)
				}
				return capacity
			},
		)
	}
}

type StorageCapacityProvider interface {
	CommitStorage(inter *interpreter.Interpreter, commitContractUpdates bool) error
	// GetStorageCapacity gets storage capacity in bytes on the address.
	GetStorageCapacity(address common.Address) (uint64, error)
}

func newStorageCapacityGetFunction(
	provider StorageCapacityProvider,
	addressValue interpreter.AddressValue,
) func(inter *interpreter.Interpreter) interpreter.UInt64Value {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func(inter *interpreter.Interpreter) interpreter.UInt64Value {

		// NOTE: flush the cached values, so the host environment
		// can properly calculate the amount of storage available for the account
		const commitContractUpdates = false
		err := provider.CommitStorage(inter, commitContractUpdates)
		if err != nil {
			panic(err)
		}

		return interpreter.NewUInt64Value(
			inter,
			func() uint64 {
				var capacity uint64
				wrapPanic(func() {
					capacity, err = provider.GetStorageCapacity(address)
				})
				if err != nil {
					panic(err)
				}
				return capacity
			},
		)

	}
}

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
//				AccountKeyAddedEventType,
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
//				AccountKeyRemovedEventType,
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
//				AccountKeyAddedEventType,
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

type AccountKey struct {
	KeyIndex  int
	PublicKey *PublicKey
	HashAlgo  sema.HashAlgorithm
	Weight    int
	IsRevoked bool
}

type PublicKey struct {
	PublicKey []byte
	SignAlgo  sema.SignatureAlgorithm
}

type AccountKeyProvider interface {
	// GetAccountKey retrieves a key from an account by index.
	GetAccountKey(address common.Address, index int) (*AccountKey, error)
}

func newAccountKeysGetFunction(
	gauge common.MemoryGauge,
	provider AccountKeyProvider,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		func(invocation interpreter.Invocation) interpreter.Value {
			indexValue, ok := invocation.Arguments[0].(interpreter.IntValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			index := indexValue.ToInt()

			var err error
			var accountKey *AccountKey
			wrapPanic(func() {
				accountKey, err = provider.GetAccountKey(address, index)
			})

			if err != nil {
				panic(err)
			}

			// Here it is expected the host function to return a nil key, if a key is not found at the given index.
			// This is done because, if the host function returns an error when a key is not found, then
			// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
			if accountKey == nil {
				return interpreter.NewNilValue(invocation.Interpreter)
			}

			inter := invocation.Interpreter

			return interpreter.NewSomeValueNonCopying(
				inter,
				NewAccountKeyValue(
					inter,
					invocation.GetLocationRange,
					accountKey,
					// public keys are assumed to be already validated.
					func(
						_ *interpreter.Interpreter,
						_ func() interpreter.LocationRange,
						_ *interpreter.CompositeValue,
					) error {
						return nil
					},
				),
			)
		},
		sema.AccountKeysTypeGetFunctionType,
	)
}

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
//				AccountKeyRemovedEventType,
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

type PublicAccountKeysHandler interface {
	AccountKeyProvider
}

func newPublicAccountKeysValue(
	gauge common.MemoryGauge,
	handler PublicAccountKeysHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewPublicAccountKeysValue(
		gauge,
		addressValue,
		newAccountKeysGetFunction(
			gauge,
			handler,
			addressValue,
		),
	)
}

type PublicAccountContractsHandler interface {
	AccountContractNamesProvider
	AccountContractProvider
}

func newPublicAccountContractsValue(
	gauge common.MemoryGauge,
	handler PublicAccountContractsHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewPublicAccountContractsValue(
		gauge,
		addressValue,
		newAccountContractsGetFunction(
			gauge,
			handler,
			addressValue,
		),
		newAccountContractsGetNamesFunction(
			addressValue,
			handler,
		),
	)
}

type AccountContractNamesProvider interface {
	// GetAccountContractNames returns the names of all contracts deployed in an account.
	GetAccountContractNames(address common.Address) ([]string, error)
}

func newAccountContractsGetNamesFunction(
	addressValue interpreter.AddressValue,
	provider AccountContractNamesProvider,
) func(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
) *interpreter.ArrayValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func(
		inter *interpreter.Interpreter,
		getLocationRange func() interpreter.LocationRange,
	) *interpreter.ArrayValue {
		var names []string
		var err error
		wrapPanic(func() {
			names, err = provider.GetAccountContractNames(address)
		})
		if err != nil {
			panic(err)
		}

		values := make([]interpreter.Value, len(names))
		for i, name := range names {
			memoryUsage := common.NewStringMemoryUsage(len(name))
			values[i] = interpreter.NewStringValue(
				inter,
				memoryUsage,
				func() string {
					return name
				},
			)
		}

		arrayType := interpreter.NewVariableSizedStaticType(
			inter,
			interpreter.NewPrimitiveStaticType(
				inter,
				interpreter.PrimitiveStaticTypeString,
			),
		)

		return interpreter.NewArrayValue(
			inter,
			getLocationRange,
			arrayType,
			common.Address{},
			values...,
		)
	}
}

type AccountContractProvider interface {
	// GetAccountContractCode returns the code associated with an account contract.
	GetAccountContractCode(address common.Address, name string) ([]byte, error)
}

func newAccountContractsGetFunction(
	gauge common.MemoryGauge,
	provider AccountContractProvider,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		func(invocation interpreter.Invocation) interpreter.Value {
			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			name := nameValue.Str

			var code []byte
			var err error
			wrapPanic(func() {
				code, err = provider.GetAccountContractCode(address, name)
			})
			if err != nil {
				panic(err)
			}

			if len(code) > 0 {
				return interpreter.NewSomeValueNonCopying(
					invocation.Interpreter,
					interpreter.NewDeployedContractValue(
						invocation.Interpreter,
						addressValue,
						nameValue,
						interpreter.ByteSliceToByteArrayValue(
							invocation.Interpreter,
							code,
						),
					),
				)
			} else {
				return interpreter.NewNilValue(invocation.Interpreter)
			}
		},
		sema.AuthAccountContractsTypeGetFunctionType,
	)
}

//// newAuthAccountContractsChangeFunction called when e.g.
//// - adding: `AuthAccount.contracts.add(name: "Foo", code: [...])` (isUpdate = false)
//// - updating: `AuthAccount.contracts.update__experimental(name: "Foo", code: [...])` (isUpdate = true)
////
//func (r *interpreterRuntime) newAuthAccountContractsChangeFunction(
//	gauge common.MemoryGauge,
//  handler,
//	addressValue interpreter.AddressValue,
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
//			existingCode, err := handler.GetAccountContractCode(address, nameArgument)
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
//				eventType = AccountContractUpdatedEventType
//			} else {
//				eventType = AccountContractAddedEventType
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
//					AccountContractRemovedEventType,
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
	BalanceProvider
	AvailableBalanceProvider
	StorageUsedProvider
	StorageCapacityProvider
	PublicAccountKeysHandler
	PublicAccountContractsHandler
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
	gauge common.MemoryGauge,
	handler PublicAccountHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewPublicAccountValue(
		gauge,
		addressValue,
		newAccountBalanceGetFunction(gauge, handler, addressValue),
		newAccountAvailableBalanceGetFunction(gauge, handler, addressValue),
		newStorageUsedGetFunction(handler, addressValue),
		newStorageCapacityGetFunction(handler, addressValue),
		func() interpreter.Value {
			return newPublicAccountKeysValue(gauge, handler, addressValue)
		},
		func() interpreter.Value {
			return newPublicAccountContractsValue(gauge, handler, addressValue)
		},
	)
}

func NewAccountKeyValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	accountKey *AccountKey,
	validatePublicKey interpreter.PublicKeyValidationHandlerFunc,
) interpreter.Value {
	return interpreter.NewAccountKeyValue(
		inter,
		interpreter.NewIntValueFromInt64(inter, int64(accountKey.KeyIndex)),
		NewPublicKeyValue(
			inter,
			getLocationRange,
			accountKey.PublicKey,
			validatePublicKey,
		),
		NewHashAlgorithmCase(
			interpreter.UInt8Value(accountKey.HashAlgo.RawValue()),
		),
		interpreter.NewUFix64ValueWithInteger(
			inter, func() uint64 {
				return uint64(accountKey.Weight)
			},
		),
		interpreter.BoolValue(accountKey.IsRevoked),
	)
}

func NewPublicKeyFromValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	publicKey interpreter.MemberAccessibleValue,
) (
	*PublicKey,
	error,
) {
	// publicKey field
	key := publicKey.GetMember(inter, getLocationRange, sema.PublicKeyPublicKeyField)

	byteArray, err := interpreter.ByteArrayValueToByteSlice(inter, key)
	if err != nil {
		return nil, errors.NewUnexpectedError("public key needs to be a byte array. %w", err)
	}

	// sign algo field
	signAlgoField := publicKey.GetMember(inter, getLocationRange, sema.PublicKeySignAlgoField)
	if signAlgoField == nil {
		return nil, errors.NewUnexpectedError("sign algorithm is not set")
	}

	signAlgoValue, ok := signAlgoField.(*interpreter.SimpleCompositeValue)
	if !ok {
		return nil, errors.NewUnexpectedError(
			"sign algorithm does not belong to type: %s",
			sema.SignatureAlgorithmType.QualifiedString(),
		)
	}

	rawValue := signAlgoValue.GetMember(inter, getLocationRange, sema.EnumRawValueFieldName)
	if rawValue == nil {
		return nil, errors.NewDefaultUserError("sign algorithm raw value is not set")
	}

	signAlgoRawValue, ok := rawValue.(interpreter.UInt8Value)
	if !ok {
		return nil, errors.NewUnexpectedError(
			"sign algorithm raw-value does not belong to type: %s",
			sema.UInt8Type.QualifiedString(),
		)
	}

	return &PublicKey{
		PublicKey: byteArray,
		SignAlgo:  sema.SignatureAlgorithm(signAlgoRawValue.ToInt()),
	}, nil
}

func NewPublicKeyValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	publicKey *PublicKey,
	validatePublicKey interpreter.PublicKeyValidationHandlerFunc,
) *interpreter.CompositeValue {
	return interpreter.NewPublicKeyValue(
		inter,
		getLocationRange,
		interpreter.ByteSliceToByteArrayValue(
			inter,
			publicKey.PublicKey,
		),
		NewSignatureAlgorithmCase(
			interpreter.UInt8Value(publicKey.SignAlgo.RawValue()),
		),
		func(
			inter *interpreter.Interpreter,
			getLocationRange func() interpreter.LocationRange,
			publicKeyValue *interpreter.CompositeValue,
		) error {
			return validatePublicKey(inter, getLocationRange, publicKeyValue)
		},
	)
}

func NewHashAlgorithmFromValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	value interpreter.Value,
) sema.HashAlgorithm {
	hashAlgoValue := value.(*interpreter.SimpleCompositeValue)

	rawValue := hashAlgoValue.GetMember(inter, getLocationRange, sema.EnumRawValueFieldName)
	if rawValue == nil {
		panic("cannot find hash algorithm raw value")
	}

	hashAlgoRawValue := rawValue.(interpreter.UInt8Value)

	return sema.HashAlgorithm(hashAlgoRawValue.ToInt())
}
