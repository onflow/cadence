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

package stdlib

import (
	"fmt"

	"golang.org/x/crypto/sha3"
	"golang.org/x/xerrors"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
)

const accountFunctionDocString = `
Creates a new account, paid by the given existing account
`

// accountFunctionType is the type
//
//	fun Account(payer: auth(BorrowValue | Storage) &Account):
//	  auth(Storage, Contracts, Keys, Inbox, Capabilities) &Account
var accountFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityImpure,
	[]sema.Parameter{
		{
			Identifier: "payer",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.ReferenceType{
					Authorization: sema.NewEntitlementSetAccess(
						[]*sema.EntitlementType{
							sema.BorrowValueType,
							sema.StorageType,
						},
						sema.Disjunction,
					),
					Type: sema.AccountType,
				},
			),
		},
	},
	sema.FullyEntitledAccountReferenceTypeAnnotation,
)

type EventEmitter interface {
	EmitEvent(
		inter *interpreter.Interpreter,
		eventType *sema.CompositeType,
		values []interpreter.Value,
		locationRange interpreter.LocationRange,
	)
}

type AccountIDGenerator interface {
	// GenerateAccountID generates a new, *non-zero*, unique ID for the given account.
	GenerateAccountID(address common.Address) (uint64, error)
}

type StorageCommitter interface {
	CommitStorageTemporarily(inter *interpreter.Interpreter) error
}

type AccountHandler interface {
	AccountIDGenerator
	BalanceProvider
	AvailableBalanceProvider
	AccountStorageHandler
	AccountKeysHandler
	AccountContractsHandler
}

type AccountCreator interface {
	StorageCommitter
	EventEmitter
	AccountHandler
	// CreateAccount creates a new account.
	CreateAccount(payer common.Address) (address common.Address, err error)
}

func NewAccountConstructor(creator AccountCreator) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"Account",
		accountFunctionType,
		accountFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {

			payer, ok := invocation.Arguments[0].(interpreter.MemberAccessibleValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			inter.ExpectType(
				payer,
				sema.AccountReferenceType,
				locationRange,
			)

			payerValue := payer.GetMember(
				inter,
				locationRange,
				sema.AccountTypeAddressFieldName,
			)
			if payerValue == nil {
				panic(errors.NewUnexpectedError("payer address is not set"))
			}

			payerAddressValue, ok := payerValue.(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnexpectedError("payer address is not address"))
			}

			payerAddress := payerAddressValue.ToAddress()

			err := creator.CommitStorageTemporarily(inter)
			if err != nil {
				panic(err)
			}

			addressValue := interpreter.NewAddressValueFromConstructor(
				inter,
				func() (address common.Address) {
					var err error
					errors.WrapPanic(func() {
						address, err = creator.CreateAccount(payerAddress)
					})
					if err != nil {
						panic(interpreter.WrappedExternalError(err))
					}

					return
				},
			)

			creator.EmitEvent(
				inter,
				AccountCreatedEventType,
				[]interpreter.Value{addressValue},
				locationRange,
			)

			return NewAccountReferenceValue(
				inter,
				creator,
				addressValue,
				interpreter.FullyEntitledAccountAccess,
				locationRange,
			)
		},
	)
}

const getAuthAccountFunctionName = "getAuthAccount"
const getAuthAccountFunctionDocString = `
Returns the account for the given address. Only available in scripts
`

// getAuthAccountFunctionType represents the type
//
//	fun getAuthAccount<T: &Account>(_ address: Address): T
var getAuthAccountFunctionType = func() *sema.FunctionType {

	typeParam := &sema.TypeParameter{
		Name:      "T",
		TypeBound: sema.AccountReferenceType,
	}

	return &sema.FunctionType{
		Purity:         sema.FunctionPurityView,
		TypeParameters: []*sema.TypeParameter{typeParam},
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "address",
				TypeAnnotation: sema.AddressTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			&sema.GenericType{
				TypeParameter: typeParam,
			},
		),
	}
}()

func NewGetAuthAccountFunction(handler AccountHandler) StandardLibraryValue {
	return NewStandardLibraryFunction(
		getAuthAccountFunctionName,
		getAuthAccountFunctionType,
		getAuthAccountFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			accountAddress, ok := invocation.Arguments[0].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := typeParameterPair.Value

			referenceType, ok := ty.(*sema.ReferenceType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			authorization := interpreter.ConvertSemaAccessToStaticAuthorization(
				inter,
				referenceType.Authorization,
			)

			return NewAccountReferenceValue(
				inter,
				handler,
				accountAddress,
				authorization,
				locationRange,
			)
		},
	)
}

func NewAccountReferenceValue(
	inter *interpreter.Interpreter,
	handler AccountHandler,
	addressValue interpreter.AddressValue,
	authorization interpreter.Authorization,
	locationRange interpreter.LocationRange,
) interpreter.Value {
	account := NewAccountValue(inter, handler, addressValue)
	return interpreter.NewEphemeralReferenceValue(
		inter,
		authorization,
		account,
		sema.AccountType,
		locationRange,
	)
}

func NewAccountValue(
	gauge common.MemoryGauge,
	handler AccountHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {

	return interpreter.NewAccountValue(
		gauge,
		addressValue,
		newAccountBalanceGetFunction(gauge, handler, addressValue),
		newAccountAvailableBalanceGetFunction(gauge, handler, addressValue),
		func() interpreter.Value {
			return newAccountStorageValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountContractsValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountKeysValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountInboxValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountCapabilitiesValue(
				gauge,
				handler,
				addressValue,
			)
		},
	)
}

type AccountContractsHandler interface {
	AccountContractProvider
	AccountContractAdditionHandler
	AccountContractRemovalHandler
	AccountContractNamesProvider
}

func newAccountContractsValue(
	gauge common.MemoryGauge,
	handler AccountContractsHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountContractsValue(
		gauge,
		addressValue,
		newAccountContractsChangeFunction(
			sema.Account_ContractsTypeAddFunctionType,
			gauge,
			handler,
			addressValue,
			false,
		),
		newAccountContractsChangeFunction(
			sema.Account_ContractsTypeUpdateFunctionType,
			gauge,
			handler,
			addressValue,
			true,
		),
		newAccountContractsTryUpdateFunction(
			sema.Account_ContractsTypeUpdateFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountContractsGetFunction(
			sema.Account_ContractsTypeGetFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountContractsBorrowFunction(
			sema.Account_ContractsTypeBorrowFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountContractsRemoveFunction(
			gauge,
			handler,
			addressValue,
		),
		newAccountContractsGetNamesFunction(
			handler,
			addressValue,
		),
	)
}

type AccountStorageHandler interface {
	StorageUsedProvider
	StorageCapacityProvider
}

func newAccountStorageValue(
	gauge common.MemoryGauge,
	handler AccountStorageHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountStorageValue(
		gauge,
		addressValue,
		newStorageUsedGetFunction(handler, addressValue),
		newStorageCapacityGetFunction(handler, addressValue),
	)
}

type AccountKeysHandler interface {
	AccountKeyProvider
	AccountKeyAdditionHandler
	AccountKeyRevocationHandler
}

func newAccountKeysValue(
	gauge common.MemoryGauge,
	handler AccountKeysHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountKeysValue(
		gauge,
		addressValue,
		newAccountKeysAddFunction(
			gauge,
			handler,
			addressValue,
		),
		newAccountKeysGetFunction(
			sema.Account_KeysTypeGetFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountKeysRevokeFunction(
			gauge,
			handler,
			addressValue,
		),
		newAccountKeysForEachFunction(
			sema.Account_KeysTypeForEachFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountKeysCountGetter(gauge, handler, addressValue),
	)
}

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
			errors.WrapPanic(func() {
				balance, err = provider.GetAccountBalance(address)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
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
			errors.WrapPanic(func() {
				balance, err = provider.GetAccountAvailableBalance(address)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}

			return
		})
	}
}

type StorageUsedProvider interface {
	StorageCommitter
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
		err := provider.CommitStorageTemporarily(inter)
		if err != nil {
			panic(err)
		}

		return interpreter.NewUInt64Value(
			inter,
			func() uint64 {
				var capacity uint64
				errors.WrapPanic(func() {
					capacity, err = provider.GetStorageUsed(address)
				})
				if err != nil {
					panic(interpreter.WrappedExternalError(err))
				}
				return capacity
			},
		)
	}
}

type StorageCapacityProvider interface {
	StorageCommitter
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
		err := provider.CommitStorageTemporarily(inter)
		if err != nil {
			panic(err)
		}

		return interpreter.NewUInt64Value(
			inter,
			func() uint64 {
				var capacity uint64
				errors.WrapPanic(func() {
					capacity, err = provider.GetStorageCapacity(address)
				})
				if err != nil {
					panic(interpreter.WrappedExternalError(err))
				}
				return capacity
			},
		)

	}
}

type AccountKeyAdditionHandler interface {
	EventEmitter
	PublicKeyValidator
	PublicKeySignatureVerifier
	BLSPoPVerifier
	Hasher
	// AddAccountKey appends a key to an account.
	AddAccountKey(address common.Address, key *PublicKey, algo sema.HashAlgorithm, weight int) (*AccountKey, error)
}

func newAccountKeysAddFunction(
	gauge common.MemoryGauge,
	handler AccountKeyAdditionHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_KeysTypeAddFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			publicKeyValue, ok := invocation.Arguments[0].(*interpreter.CompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			publicKey, err := NewPublicKeyFromValue(inter, locationRange, publicKeyValue)
			if err != nil {
				panic(err)
			}

			hashAlgo := NewHashAlgorithmFromValue(inter, locationRange, invocation.Arguments[1])
			weightValue, ok := invocation.Arguments[2].(interpreter.UFix64Value)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			weight := weightValue.ToInt(locationRange)

			var accountKey *AccountKey
			errors.WrapPanic(func() {
				accountKey, err = handler.AddAccountKey(address, publicKey, hashAlgo, weight)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}

			handler.EmitEvent(
				inter,
				AccountKeyAddedFromPublicKeyEventType,
				[]interpreter.Value{
					addressValue,
					publicKeyValue,
				},
				locationRange,
			)

			return NewAccountKeyValue(
				inter,
				locationRange,
				accountKey,
				handler,
			)
		},
	)
}

type AccountKey struct {
	PublicKey *PublicKey
	KeyIndex  int
	Weight    int
	HashAlgo  sema.HashAlgorithm
	IsRevoked bool
}

type AccountKeyProvider interface {
	PublicKeyValidator
	PublicKeySignatureVerifier
	BLSPoPVerifier
	Hasher
	// GetAccountKey retrieves a key from an account by index.
	GetAccountKey(address common.Address, index int) (*AccountKey, error)
	AccountKeysCount(address common.Address) (uint64, error)
}

func newAccountKeysGetFunction(
	functionType *sema.FunctionType,
	gauge common.MemoryGauge,
	provider AccountKeyProvider,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		functionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			indexValue, ok := invocation.Arguments[0].(interpreter.IntValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			locationRange := invocation.LocationRange
			index := indexValue.ToInt(locationRange)

			var err error
			var accountKey *AccountKey
			errors.WrapPanic(func() {
				accountKey, err = provider.GetAccountKey(address, index)
			})

			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}

			// Here it is expected the host function to return a nil key, if a key is not found at the given index.
			// This is done because, if the host function returns an error when a key is not found, then
			// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
			if accountKey == nil {
				return interpreter.Nil
			}

			inter := invocation.Interpreter

			return interpreter.NewSomeValueNonCopying(
				inter,
				NewAccountKeyValue(
					inter,
					locationRange,
					accountKey,
					provider,
				),
			)
		},
	)
}

// accountKeysForEachCallbackTypeParams are the parameter types of the callback function of
// `Account.Keys.forEachKey(_ f: fun(AccountKey): Bool)`
var accountKeysForEachCallbackTypeParams = []sema.Type{sema.AccountKeyType}

func newAccountKeysForEachFunction(
	functionType *sema.FunctionType,
	gauge common.MemoryGauge,
	provider AccountKeyProvider,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		functionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			fnValue, ok := invocation.Arguments[0].(interpreter.FunctionValue)

			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			newSubInvocation := func(key interpreter.Value) interpreter.Invocation {
				return interpreter.NewInvocation(
					inter,
					nil,
					nil,
					nil,
					[]interpreter.Value{key},
					accountKeysForEachCallbackTypeParams,
					nil,
					locationRange,
				)
			}

			liftKeyToValue := func(key *AccountKey) interpreter.Value {
				return NewAccountKeyValue(
					inter,
					locationRange,
					key,
					provider,
				)
			}

			var count uint64
			var err error

			errors.WrapPanic(func() {
				count, err = provider.AccountKeysCount(address)
			})

			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}

			var accountKey *AccountKey

			for index := uint64(0); index < count; index++ {
				errors.WrapPanic(func() {
					accountKey, err = provider.GetAccountKey(address, int(index))
				})
				if err != nil {
					panic(interpreter.WrappedExternalError(err))
				}

				// Here it is expected the host function to return a nil key, if a key is not found at the given index.
				// This is done because, if the host function returns an error when a key is not found, then
				// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
				if accountKey == nil {
					continue
				}

				liftedKey := liftKeyToValue(accountKey)

				res, err := inter.InvokeFunction(
					fnValue,
					newSubInvocation(liftedKey),
				)
				if err != nil {
					// interpreter panicked while invoking the inner function value
					panic(err)
				}

				shouldContinue, ok := res.(interpreter.BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if !shouldContinue {
					break
				}
			}

			return interpreter.Void
		},
	)
}

func newAccountKeysCountGetter(
	gauge common.MemoryGauge,
	provider AccountKeyProvider,
	addressValue interpreter.AddressValue,
) interpreter.AccountKeysCountGetter {
	address := addressValue.ToAddress()

	return func() interpreter.UInt64Value {
		return interpreter.NewUInt64Value(gauge, func() uint64 {
			var count uint64
			var err error

			errors.WrapPanic(func() {
				count, err = provider.AccountKeysCount(address)
			})
			if err != nil {
				// The provider might not be able to fetch the number of account keys
				// e.g. when the account does not exist
				panic(interpreter.WrappedExternalError(err))
			}

			return count
		})
	}
}

type AccountKeyRevocationHandler interface {
	EventEmitter
	Hasher
	PublicKeyValidator
	PublicKeySignatureVerifier
	BLSPoPVerifier
	// RevokeAccountKey removes a key from an account by index.
	RevokeAccountKey(address common.Address, index int) (*AccountKey, error)
}

func newAccountKeysRevokeFunction(
	gauge common.MemoryGauge,
	handler AccountKeyRevocationHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_KeysTypeRevokeFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			indexValue, ok := invocation.Arguments[0].(interpreter.IntValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			locationRange := invocation.LocationRange
			index := indexValue.ToInt(locationRange)

			var err error
			var accountKey *AccountKey
			errors.WrapPanic(func() {
				accountKey, err = handler.RevokeAccountKey(address, index)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}

			// Here it is expected the host function to return a nil key, if a key is not found at the given index.
			// This is done because, if the host function returns an error when a key is not found, then
			// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
			if accountKey == nil {
				return interpreter.Nil
			}

			inter := invocation.Interpreter

			handler.EmitEvent(
				inter,
				AccountKeyRemovedFromPublicKeyIndexEventType,
				[]interpreter.Value{
					addressValue,
					indexValue,
				},
				locationRange,
			)

			return interpreter.NewSomeValueNonCopying(
				inter,
				NewAccountKeyValue(
					inter,
					locationRange,
					accountKey,
					handler,
				),
			)
		},
	)
}

const InboxStorageDomain = "inbox"

func newAccountInboxPublishFunction(
	gauge common.MemoryGauge,
	handler EventEmitter,
	providerValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	provider := providerValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_InboxTypePublishFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			value, ok := invocation.Arguments[0].(*interpreter.CapabilityValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			nameValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			recipientValue := invocation.Arguments[2].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			handler.EmitEvent(
				inter,
				AccountInboxPublishedEventType,
				[]interpreter.Value{
					providerValue,
					recipientValue,
					nameValue,
					interpreter.NewTypeValue(gauge, value.StaticType(inter)),
				},
				locationRange,
			)

			publishedValue := interpreter.NewPublishedValue(inter, recipientValue, value).Transfer(
				inter,
				locationRange,
				atree.Address(provider),
				true,
				nil,
				nil,
			)

			storageMapKey := interpreter.StringStorageMapKey(nameValue.Str)

			inter.WriteStored(
				provider,
				InboxStorageDomain,
				storageMapKey,
				publishedValue,
			)

			return interpreter.Void
		},
	)
}

func newAccountInboxUnpublishFunction(
	gauge common.MemoryGauge,
	handler EventEmitter,
	providerValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	provider := providerValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_InboxTypeUnpublishFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			storageMapKey := interpreter.StringStorageMapKey(nameValue.Str)

			readValue := inter.ReadStored(provider, InboxStorageDomain, storageMapKey)
			if readValue == nil {
				return interpreter.Nil
			}
			publishedValue := readValue.(*interpreter.PublishedValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := sema.NewCapabilityType(gauge, typeParameterPair.Value)
			publishedType := publishedValue.Value.StaticType(invocation.Interpreter)
			if !inter.IsSubTypeOfSemaType(publishedType, ty) {
				panic(interpreter.ForceCastTypeMismatchError{
					ExpectedType:  ty,
					ActualType:    inter.MustConvertStaticToSemaType(publishedType),
					LocationRange: locationRange,
				})
			}

			value := publishedValue.Value.Transfer(
				inter,
				locationRange,
				atree.Address{},
				true,
				nil,
				nil,
			)

			inter.WriteStored(
				provider,
				InboxStorageDomain,
				storageMapKey,
				nil,
			)

			handler.EmitEvent(
				inter,
				AccountInboxUnpublishedEventType,
				[]interpreter.Value{
					providerValue,
					nameValue,
				},
				locationRange,
			)

			return interpreter.NewSomeValueNonCopying(inter, value)
		},
	)
}

func newAccountInboxClaimFunction(
	gauge common.MemoryGauge,
	handler EventEmitter,
	recipientValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_InboxTypePublishFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			providerValue, ok := invocation.Arguments[1].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			providerAddress := providerValue.ToAddress()

			storageMapKey := interpreter.StringStorageMapKey(nameValue.Str)

			readValue := inter.ReadStored(providerAddress, InboxStorageDomain, storageMapKey)
			if readValue == nil {
				return interpreter.Nil
			}
			publishedValue := readValue.(*interpreter.PublishedValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// compare the intended recipient with the caller
			if !publishedValue.Recipient.Equal(inter, locationRange, recipientValue) {
				return interpreter.Nil
			}

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := sema.NewCapabilityType(gauge, typeParameterPair.Value)
			publishedType := publishedValue.Value.StaticType(invocation.Interpreter)
			if !inter.IsSubTypeOfSemaType(publishedType, ty) {
				panic(interpreter.ForceCastTypeMismatchError{
					ExpectedType:  ty,
					ActualType:    inter.MustConvertStaticToSemaType(publishedType),
					LocationRange: locationRange,
				})
			}

			value := publishedValue.Value.Transfer(
				inter,
				locationRange,
				atree.Address{},
				true,
				nil,
				nil,
			)

			inter.WriteStored(
				providerAddress,
				InboxStorageDomain,
				storageMapKey,
				nil,
			)

			handler.EmitEvent(
				inter,
				AccountInboxClaimedEventType,
				[]interpreter.Value{
					providerValue,
					recipientValue,
					nameValue,
				},
				locationRange,
			)

			return interpreter.NewSomeValueNonCopying(inter, value)
		},
	)
}

func newAccountInboxValue(
	gauge common.MemoryGauge,
	handler EventEmitter,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountInboxValue(
		gauge,
		addressValue,
		newAccountInboxPublishFunction(gauge, handler, addressValue),
		newAccountInboxUnpublishFunction(gauge, handler, addressValue),
		newAccountInboxClaimFunction(gauge, handler, addressValue),
	)
}

type AccountContractNamesProvider interface {
	// GetAccountContractNames returns the names of all contracts deployed in an account.
	GetAccountContractNames(address common.Address) ([]string, error)
}

func newAccountContractsGetNamesFunction(
	provider AccountContractNamesProvider,
	addressValue interpreter.AddressValue,
) func(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
) *interpreter.ArrayValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
	) *interpreter.ArrayValue {
		var names []string
		var err error
		errors.WrapPanic(func() {
			names, err = provider.GetAccountContractNames(address)
		})
		if err != nil {
			panic(interpreter.WrappedExternalError(err))
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
			locationRange,
			arrayType,
			common.ZeroAddress,
			values...,
		)
	}
}

type AccountContractProvider interface {
	// GetAccountContractCode returns the code associated with an account contract.
	GetAccountContractCode(location common.AddressLocation) ([]byte, error)
}

func newAccountContractsGetFunction(
	functionType *sema.FunctionType,
	gauge common.MemoryGauge,
	provider AccountContractProvider,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		functionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			name := nameValue.Str
			location := common.NewAddressLocation(invocation.Interpreter, address, name)

			var code []byte
			var err error
			errors.WrapPanic(func() {
				code, err = provider.GetAccountContractCode(location)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
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
				return interpreter.Nil
			}
		},
	)
}

func newAccountContractsBorrowFunction(
	functionType *sema.FunctionType,
	gauge common.MemoryGauge,
	handler AccountContractsHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		functionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			name := nameValue.Str
			location := common.NewAddressLocation(invocation.Interpreter, address, name)

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}
			ty := typeParameterPair.Value

			referenceType, ok := ty.(*sema.ReferenceType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Check if the contract exists

			var code []byte
			var err error
			errors.WrapPanic(func() {
				code, err = handler.GetAccountContractCode(location)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}
			if len(code) == 0 {
				return interpreter.Nil
			}

			// Load the contract

			contractLocation := common.NewAddressLocation(gauge, address, name)
			inter = inter.EnsureLoaded(contractLocation)
			contractValue, err := inter.GetContractComposite(contractLocation)
			if err != nil {
				panic(err)
			}

			// Check the type

			staticType := contractValue.StaticType(inter)
			if !inter.IsSubTypeOfSemaType(staticType, referenceType.Type) {
				return interpreter.Nil
			}

			// No need to track the referenced value, since the reference is taken to a contract value.
			// A contract value would never be moved or destroyed, within the execution of a program.
			reference := interpreter.NewEphemeralReferenceValue(
				inter,
				interpreter.UnauthorizedAccess,
				contractValue,
				referenceType.Type,
				locationRange,
			)

			return interpreter.NewSomeValueNonCopying(
				inter,
				reference,
			)

		},
	)
}

type AccountContractAdditionHandler interface {
	EventEmitter
	AccountContractProvider
	ParseAndCheckProgram(
		code []byte,
		location common.Location,
		getAndSetProgram bool,
	) (*interpreter.Program, error)
	// UpdateAccountContractCode updates the code associated with an account contract.
	UpdateAccountContractCode(location common.AddressLocation, code []byte) error
	RecordContractUpdate(
		location common.AddressLocation,
		value *interpreter.CompositeValue,
	)
	ContractUpdateRecorded(location common.AddressLocation) bool
	InterpretContract(
		location common.AddressLocation,
		program *interpreter.Program,
		name string,
		invocation DeployedContractConstructorInvocation,
	) (
		*interpreter.CompositeValue,
		error,
	)
	TemporarilyRecordCode(location common.AddressLocation, code []byte)

	// StartContractAddition starts adding a contract.
	StartContractAddition(location common.AddressLocation)

	// EndContractAddition ends adding the contract
	EndContractAddition(location common.AddressLocation)

	// IsContractBeingAdded checks whether a contract is being added in the current execution.
	IsContractBeingAdded(location common.AddressLocation) bool
}

// newAccountContractsChangeFunction called when e.g.
// - adding: `Account.contracts.add(name: "Foo", code: [...])` (isUpdate = false)
// - updating: `Account.contracts.update(name: "Foo", code: [...])` (isUpdate = true)
func newAccountContractsChangeFunction(
	functionType *sema.FunctionType,
	gauge common.MemoryGauge,
	handler AccountContractAdditionHandler,
	addressValue interpreter.AddressValue,
	isUpdate bool,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		functionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			return changeAccountContracts(invocation, handler, addressValue, isUpdate)
		},
	)
}

func changeAccountContracts(
	invocation interpreter.Invocation,
	handler AccountContractAdditionHandler,
	addressValue interpreter.AddressValue,
	isUpdate bool,
) interpreter.Value {

	locationRange := invocation.LocationRange

	const requiredArgumentCount = 2

	nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	newCodeValue, ok := invocation.Arguments[1].(*interpreter.ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	constructorArguments := invocation.Arguments[requiredArgumentCount:]
	constructorArgumentTypes := invocation.ArgumentTypes[requiredArgumentCount:]

	code, err := interpreter.ByteArrayValueToByteSlice(invocation.Interpreter, newCodeValue, locationRange)
	if err != nil {
		panic(errors.NewDefaultUserError("add requires the second argument to be an array"))
	}

	// Get the existing code

	contractName := nameValue.Str

	if contractName == "" {
		panic(errors.NewDefaultUserError(
			"contract name argument cannot be empty." +
				"it must match the name of the deployed contract declaration or contract interface declaration",
		))
	}

	address := addressValue.ToAddress()
	location := common.NewAddressLocation(invocation.Interpreter, address, contractName)

	existingCode, err := handler.GetAccountContractCode(location)
	if err != nil {
		panic(err)
	}

	if isUpdate {
		// We are updating an existing contract.
		// Ensure that there's a contract/contract-interface with the given name exists already

		if len(existingCode) == 0 {
			panic(errors.NewDefaultUserError(
				"cannot update non-existing contract with name %q in account %s",
				contractName,
				address.ShortHexWithPrefix(),
			))
		}

	} else {
		// We are adding a new contract.
		// Ensure that no contract/contract interface with the given name exists already,
		// and no contract deploy or update was recorded before

		if len(existingCode) > 0 ||
			handler.ContractUpdateRecorded(location) ||
			handler.IsContractBeingAdded(location) {

			panic(errors.NewDefaultUserError(
				"cannot overwrite existing contract with name %q in account %s",
				contractName,
				address.ShortHexWithPrefix(),
			))
		}
	}

	// Check the code
	handleContractUpdateError := func(err error) {
		if err == nil {
			return
		}

		// Update the code for the error pretty printing
		// NOTE: only do this when an error occurs

		handler.TemporarilyRecordCode(location, code)

		panic(&InvalidContractDeploymentError{
			Err:           err,
			LocationRange: locationRange,
		})
	}

	// NOTE: do NOT use the program obtained from the host environment, as the current program.
	// Always re-parse and re-check the new program.

	// NOTE: *DO NOT* store the program – the new or updated program
	// should not be effective during the execution

	const getAndSetProgram = false

	program, err := handler.ParseAndCheckProgram(
		code,
		location,
		getAndSetProgram,
	)
	handleContractUpdateError(err)

	// The code may declare exactly one contract or one contract interface.

	var contractTypes []*sema.CompositeType
	var contractInterfaceTypes []*sema.InterfaceType

	program.Elaboration.ForEachGlobalType(func(_ string, variable *sema.Variable) {
		switch ty := variable.Type.(type) {
		case *sema.CompositeType:
			if ty.Kind == common.CompositeKindContract {
				contractTypes = append(contractTypes, ty)
			}

		case *sema.InterfaceType:
			if ty.CompositeKind == common.CompositeKindContract {
				contractInterfaceTypes = append(contractInterfaceTypes, ty)
			}
		}
	})

	var deployedType sema.Type
	var contractType *sema.CompositeType
	var contractInterfaceType *sema.InterfaceType
	var declaredName string
	var declarationKind common.DeclarationKind

	switch {
	case len(contractTypes) == 1 && len(contractInterfaceTypes) == 0:
		contractType = contractTypes[0]
		declaredName = contractType.Identifier
		deployedType = contractType
		declarationKind = common.DeclarationKindContract
	case len(contractInterfaceTypes) == 1 && len(contractTypes) == 0:
		contractInterfaceType = contractInterfaceTypes[0]
		declaredName = contractInterfaceType.Identifier
		deployedType = contractInterfaceType
		declarationKind = common.DeclarationKindContractInterface
	}

	if deployedType == nil {
		// Update the code for the error pretty printing
		// NOTE: only do this when an error occurs

		handler.TemporarilyRecordCode(location, code)

		panic(errors.NewDefaultUserError(
			"invalid %s: the code must declare exactly one contract or contract interface",
			declarationKind.Name(),
		))
	}

	// The declared contract or contract interface must have the name
	// passed to the constructor as the first argument

	if declaredName != contractName {
		// Update the code for the error pretty printing
		// NOTE: only do this when an error occurs

		handler.TemporarilyRecordCode(location, code)

		panic(errors.NewDefaultUserError(
			"invalid %s: the name argument must match the name of the declaration: got %q, expected %q",
			declarationKind.Name(),
			contractName,
			declaredName,
		))
	}

	// Validate the contract update

	if isUpdate {
		oldCode, err := handler.GetAccountContractCode(location)
		handleContractUpdateError(err)

		oldProgram, err := parser.ParseProgram(
			invocation.Interpreter.SharedState.Config.MemoryGauge,
			oldCode,
			parser.Config{
				IgnoreLeadingIdentifierEnabled: true,
			},
		)

		if !ignoreUpdatedProgramParserError(err) {
			handleContractUpdateError(err)
		}

		validator := NewContractUpdateValidator(
			location,
			contractName,
			oldProgram,
			program.Program,
		)
		err = validator.Validate()
		handleContractUpdateError(err)
	}

	inter := invocation.Interpreter

	err = updateAccountContractCode(
		handler,
		location,
		program,
		code,
		contractType,
		constructorArguments,
		constructorArgumentTypes,
		updateAccountContractCodeOptions{
			createContract: !isUpdate,
		},
	)
	if err != nil {
		// Update the code for the error pretty printing
		// NOTE: only do this when an error occurs

		handler.TemporarilyRecordCode(location, code)

		panic(err)
	}

	var eventType *sema.CompositeType

	if isUpdate {
		eventType = AccountContractUpdatedEventType
	} else {
		eventType = AccountContractAddedEventType
	}

	codeHashValue := CodeToHashValue(inter, code)

	handler.EmitEvent(
		inter,
		eventType,
		[]interpreter.Value{
			addressValue,
			codeHashValue,
			nameValue,
		},
		locationRange,
	)

	return interpreter.NewDeployedContractValue(
		inter,
		addressValue,
		nameValue,
		newCodeValue,
	)
}

func newAccountContractsTryUpdateFunction(
	functionType *sema.FunctionType,
	gauge common.MemoryGauge,
	handler AccountContractAdditionHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		functionType,
		func(invocation interpreter.Invocation) (deploymentResult interpreter.Value) {
			var deployedContract interpreter.Value

			defer func() {
				if r := recover(); r != nil {
					rootError := r
					for {
						switch err := r.(type) {
						case errors.UserError, errors.ExternalError:
							// Error is ignored for now.
							// Simply return with a `nil` deployed-contract
						case xerrors.Wrapper:
							r = err.Unwrap()
							continue
						default:
							panic(rootError)
						}

						break
					}
				}

				var optionalDeployedContract interpreter.OptionalValue
				if deployedContract == nil {
					optionalDeployedContract = interpreter.NilOptionalValue
				} else {
					optionalDeployedContract = interpreter.NewSomeValueNonCopying(invocation.Interpreter, deployedContract)
				}

				deploymentResult = interpreter.NewDeploymentResultValue(gauge, optionalDeployedContract)
			}()

			deployedContract = changeAccountContracts(invocation, handler, addressValue, true)
			return
		},
	)
}

// InvalidContractDeploymentError
type InvalidContractDeploymentError struct {
	Err error
	interpreter.LocationRange
}

var _ errors.UserError = &InvalidContractDeploymentError{}
var _ errors.ParentError = &InvalidContractDeploymentError{}

func (*InvalidContractDeploymentError) IsUserError() {}

func (e *InvalidContractDeploymentError) Error() string {
	return fmt.Sprintf("cannot deploy invalid contract: %s", e.Err.Error())
}

func (e *InvalidContractDeploymentError) ChildErrors() []error {
	return []error{
		&InvalidContractDeploymentOriginError{
			LocationRange: e.LocationRange,
		},
		e.Err,
	}
}

func (e *InvalidContractDeploymentError) Unwrap() error {
	return e.Err
}

// InvalidContractDeploymentOriginError
type InvalidContractDeploymentOriginError struct {
	interpreter.LocationRange
}

var _ errors.UserError = &InvalidContractDeploymentOriginError{}

func (*InvalidContractDeploymentOriginError) IsUserError() {}

func (*InvalidContractDeploymentOriginError) Error() string {
	return "cannot deploy invalid contract"
}

// ignoreUpdatedProgramParserError determines if the parsing error
// for a program that is being updated can be ignored.
func ignoreUpdatedProgramParserError(err error) bool {
	parserError, ok := err.(parser.Error)
	if !ok {
		return false
	}

	// Are all parse errors ones that can be ignored?
	for _, parseError := range parserError.Errors {
		// Missing commas in parameter lists were reported starting
		// with https://github.com/onflow/cadence/pull/1073.
		// Allow existing contracts with such an error to be updated
		_, ok := parseError.(*parser.MissingCommaInParameterListError)
		if !ok {
			return false
		}
	}

	return true
}

type updateAccountContractCodeOptions struct {
	createContract bool
}

// updateAccountContractCode updates an account contract's code.
// This function is only used for the new account code/contract API.
func updateAccountContractCode(
	handler AccountContractAdditionHandler,
	location common.AddressLocation,
	program *interpreter.Program,
	code []byte,
	contractType *sema.CompositeType,
	constructorArguments []interpreter.Value,
	constructorArgumentTypes []sema.Type,
	options updateAccountContractCodeOptions,
) error {

	// Start tracking the contract addition.
	// This must be done even before the contract code gets added,
	// to avoid the same contract being updated during the deployment of itself.
	handler.StartContractAddition(location)
	defer handler.EndContractAddition(location)

	// If the code declares a contract, instantiate it and store it.
	//
	// This function might be called when
	// 1. A contract is deployed (contractType is non-nil).
	// 2. A contract interface is deployed (contractType is nil).
	//
	// If a contract is deployed, it is only instantiated
	// when options.createContract is true,
	// i.e. the Cadence `add` function is used.
	// If the Cadence `update` function is used,
	// the new contract will NOT be deployed (options.createContract is false).

	var contractValue *interpreter.CompositeValue

	createContract := contractType != nil && options.createContract

	var err error

	if createContract {
		contractValue, err = instantiateContract(
			handler,
			location,
			program,
			contractType,
			constructorArguments,
			constructorArgumentTypes,
		)

		if err != nil {
			return err
		}
	}

	// NOTE: only update account code if contract instantiation succeeded
	errors.WrapPanic(func() {
		err = handler.UpdateAccountContractCode(location, code)
	})
	if err != nil {
		return interpreter.WrappedExternalError(err)
	}

	if createContract {
		// NOTE: the contract recording delays the write
		// until the end of the execution of the program

		handler.RecordContractUpdate(
			location,
			contractValue,
		)
	}

	return nil
}

type DeployedContractConstructorInvocation struct {
	ContractType         *sema.CompositeType
	ConstructorArguments []interpreter.Value
	ArgumentTypes        []sema.Type
	ParameterTypes       []sema.Type
	Address              common.Address
}

type InvalidContractArgumentError struct {
	ExpectedType sema.Type
	ActualType   sema.Type
	Index        int
}

var _ errors.UserError = &InvalidContractArgumentError{}

func (*InvalidContractArgumentError) IsUserError() {}

func (e *InvalidContractArgumentError) Error() string {
	expected, actual := sema.ErrorMessageExpectedActualTypes(
		e.ExpectedType,
		e.ActualType,
	)

	return fmt.Sprintf(
		"invalid argument at index %d: expected type `%s`, got `%s`",
		e.Index,
		expected,
		actual,
	)
}

func instantiateContract(
	handler AccountContractAdditionHandler,
	location common.AddressLocation,
	program *interpreter.Program,
	contractType *sema.CompositeType,
	constructorArguments []interpreter.Value,
	argumentTypes []sema.Type,
) (*interpreter.CompositeValue, error) {
	parameterTypes := make([]sema.Type, len(contractType.ConstructorParameters))

	for i, constructorParameter := range contractType.ConstructorParameters {
		parameterTypes[i] = constructorParameter.TypeAnnotation.Type
	}

	// Check argument count

	argumentCount := len(argumentTypes)
	parameterCount := len(parameterTypes)

	if argumentCount < parameterCount {
		return nil, errors.NewDefaultUserError(
			"invalid argument count, too few arguments: expected %d, got %d, next missing argument: `%s`",
			parameterCount, argumentCount,
			parameterTypes[argumentCount],
		)
	} else if argumentCount > parameterCount {
		return nil, errors.NewDefaultUserError(
			"invalid argument count, too many arguments: expected %d, got %d",
			parameterCount,
			argumentCount,
		)
	}

	// argumentCount now equals to parameterCount

	// Check arguments match parameter

	for argumentIndex := 0; argumentIndex < argumentCount; argumentIndex++ {
		argumentType := argumentTypes[argumentIndex]
		parameterTye := parameterTypes[argumentIndex]
		if !sema.IsSubType(argumentType, parameterTye) {

			return nil, &InvalidContractArgumentError{
				Index:        argumentIndex,
				ExpectedType: parameterTye,
				ActualType:   argumentType,
			}
		}
	}

	// Use a custom contract value handler that detects if the requested contract value
	// is for the contract declaration that is being deployed.
	//
	// If the contract is the deployed contract, instantiate it using
	// the provided constructor and given arguments.
	//
	// If the contract is not the deployed contract, load it from storage.

	return handler.InterpretContract(
		location,
		program,
		contractType.Identifier,
		DeployedContractConstructorInvocation{
			Address:              location.Address,
			ContractType:         contractType,
			ConstructorArguments: constructorArguments,
			ArgumentTypes:        argumentTypes,
			ParameterTypes:       parameterTypes,
		},
	)
}

type AccountContractRemovalHandler interface {
	EventEmitter
	AccountContractProvider
	RemoveAccountContractCode(location common.AddressLocation) error
	RecordContractRemoval(location common.AddressLocation)
}

func newAccountContractsRemoveFunction(
	gauge common.MemoryGauge,
	handler AccountContractRemovalHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_ContractsTypeRemoveFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			name := nameValue.Str
			location := common.NewAddressLocation(invocation.Interpreter, address, name)

			// Get the current code

			var code []byte
			var err error
			errors.WrapPanic(func() {
				code, err = handler.GetAccountContractCode(location)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}

			// Only remove the contract code, remove the contract value, and emit an event,
			// if there is currently code deployed for the given contract name

			if len(code) > 0 {
				locationRange := invocation.LocationRange

				// NOTE: *DO NOT* call setProgram – the program removal
				// should not be effective during the execution, only after

				existingProgram, err := parser.ParseProgram(gauge, code, parser.Config{})

				// If the existing code is not parsable (i.e: `err != nil`),
				// that shouldn't be a reason to fail the contract removal.
				// Therefore, validate only if the code is a valid one.
				if err == nil && containsEnumsInProgram(existingProgram) {
					panic(&ContractRemovalError{
						Name:          name,
						LocationRange: locationRange,
					})
				}

				errors.WrapPanic(func() {
					err = handler.RemoveAccountContractCode(location)
				})
				if err != nil {
					panic(interpreter.WrappedExternalError(err))
				}

				// NOTE: the contract recording function delays the write
				// until the end of the execution of the program

				handler.RecordContractRemoval(location)

				codeHashValue := CodeToHashValue(inter, code)

				handler.EmitEvent(
					inter,
					AccountContractRemovedEventType,
					[]interpreter.Value{
						addressValue,
						codeHashValue,
						nameValue,
					},
					locationRange,
				)

				return interpreter.NewSomeValueNonCopying(
					inter,
					interpreter.NewDeployedContractValue(
						inter,
						addressValue,
						nameValue,
						interpreter.ByteSliceToByteArrayValue(
							inter,
							code,
						),
					),
				)
			} else {
				return interpreter.Nil
			}
		},
	)
}

// ContractRemovalError
type ContractRemovalError struct {
	interpreter.LocationRange
	Name string
}

var _ errors.UserError = &ContractRemovalError{}

func (*ContractRemovalError) IsUserError() {}

func (e *ContractRemovalError) Error() string {
	return fmt.Sprintf("cannot remove contract `%s`", e.Name)
}

const getAccountFunctionDocString = `
Returns the account for the given address
`

var getAccountFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "address",
			TypeAnnotation: sema.AddressTypeAnnotation,
		},
	},
	sema.AccountReferenceTypeAnnotation,
)

func NewGetAccountFunction(handler AccountHandler) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"getAccount",
		getAccountFunctionType,
		getAccountFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			accountAddress, ok := invocation.Arguments[0].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return NewAccountReferenceValue(
				inter,
				handler,
				accountAddress,
				interpreter.UnauthorizedAccess,
				locationRange,
			)
		},
	)
}

func NewAccountKeyValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	accountKey *AccountKey,
	hasher Hasher,
) interpreter.Value {

	// hash algorithms converted from "native" (non-interpreter) values are assumed to be already valid
	hashAlgorithm, _ := NewHashAlgorithmCase(
		interpreter.UInt8Value(accountKey.HashAlgo.RawValue()),
		hasher,
	)

	return interpreter.NewAccountKeyValue(
		inter,
		interpreter.NewIntValueFromInt64(inter, int64(accountKey.KeyIndex)),
		NewPublicKeyValue(
			inter,
			locationRange,
			accountKey.PublicKey,
		),
		hashAlgorithm,
		interpreter.NewUFix64ValueWithInteger(
			inter, func() uint64 {
				return uint64(accountKey.Weight)
			},
			locationRange,
		),
		interpreter.AsBoolValue(accountKey.IsRevoked),
	)
}

func NewHashAlgorithmFromValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	value interpreter.Value,
) sema.HashAlgorithm {
	hashAlgoValue := value.(*interpreter.SimpleCompositeValue)

	rawValue := hashAlgoValue.GetMember(inter, locationRange, sema.EnumRawValueFieldName)
	if rawValue == nil {
		panic("cannot find hash algorithm raw value")
	}

	hashAlgoRawValue := rawValue.(interpreter.UInt8Value)

	return sema.HashAlgorithm(hashAlgoRawValue.ToInt(locationRange))
}

func CodeToHashValue(inter *interpreter.Interpreter, code []byte) *interpreter.ArrayValue {
	codeHash := sha3.Sum256(code)
	return interpreter.ByteSliceToConstantSizedByteArrayValue(inter, codeHash[:])
}

func newAccountStorageCapabilitiesValue(
	gauge common.MemoryGauge,
	accountIDGenerator AccountIDGenerator,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountStorageCapabilitiesValue(
		gauge,
		addressValue,
		newAccountStorageCapabilitiesGetControllerFunction(gauge, addressValue),
		newAccountStorageCapabilitiesGetControllersFunction(gauge, addressValue),
		newAccountStorageCapabilitiesForEachControllerFunction(gauge, addressValue),
		newAccountStorageCapabilitiesIssueFunction(gauge, accountIDGenerator, addressValue),
		newAccountStorageCapabilitiesIssueWithTypeFunction(gauge, accountIDGenerator, addressValue),
	)
}

func newAccountAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	accountIDGenerator AccountIDGenerator,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountAccountCapabilitiesValue(
		gauge,
		addressValue,
		newAccountAccountCapabilitiesGetControllerFunction(gauge, addressValue),
		newAccountAccountCapabilitiesGetControllersFunction(gauge, addressValue),
		newAccountAccountCapabilitiesForEachControllerFunction(gauge, addressValue),
		newAccountAccountCapabilitiesIssueFunction(gauge, accountIDGenerator, addressValue),
		newAccountAccountCapabilitiesIssueWithTypeFunction(gauge, accountIDGenerator, addressValue),
	)
}

func newAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	idGenerator AccountIDGenerator,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountCapabilitiesValue(
		gauge,
		addressValue,
		newAccountCapabilitiesGetFunction(gauge, addressValue, false),
		newAccountCapabilitiesGetFunction(gauge, addressValue, true),
		newAccountCapabilitiesExistsFunction(gauge, addressValue),
		newAccountCapabilitiesPublishFunction(gauge, addressValue),
		newAccountCapabilitiesUnpublishFunction(gauge, addressValue),
		func() interpreter.Value {
			return newAccountStorageCapabilitiesValue(
				gauge,
				idGenerator,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountAccountCapabilitiesValue(
				gauge,
				idGenerator,
				addressValue,
			)
		},
	)
}

func newAccountStorageCapabilitiesGetControllerFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) interpreter.FunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_StorageCapabilitiesTypeGetControllerFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get capability ID argument

			capabilityIDValue, ok := invocation.Arguments[0].(interpreter.UInt64Value)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			capabilityID := uint64(capabilityIDValue)

			referenceValue := getStorageCapabilityControllerReference(inter, address, capabilityID, locationRange)
			if referenceValue == nil {
				return interpreter.Nil
			}

			return interpreter.NewSomeValueNonCopying(inter, referenceValue)
		},
	)
}

var storageCapabilityControllerReferencesArrayStaticType = &interpreter.VariableSizedStaticType{
	Type: &interpreter.ReferenceStaticType{
		ReferencedType: interpreter.PrimitiveStaticTypeStorageCapabilityController,
		Authorization:  interpreter.UnauthorizedAccess,
	},
}

func newAccountStorageCapabilitiesGetControllersFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) interpreter.FunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_StorageCapabilitiesTypeGetControllersFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get path argument

			targetPathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
			if !ok || targetPathValue.Domain != common.PathDomainStorage {
				panic(errors.NewUnreachableError())
			}

			// Get capability controllers iterator

			nextCapabilityID, count :=
				getStorageCapabilityControllerIDsIterator(inter, address, targetPathValue)

			var capabilityControllerIndex uint64 = 0

			return interpreter.NewArrayValueWithIterator(
				inter,
				storageCapabilityControllerReferencesArrayStaticType,
				common.Address{},
				count,
				func() interpreter.Value {
					if capabilityControllerIndex >= count {
						return nil
					}
					capabilityControllerIndex++

					capabilityID, ok := nextCapabilityID()
					if !ok {
						return nil
					}

					referenceValue := getStorageCapabilityControllerReference(inter, address, capabilityID, locationRange)
					if referenceValue == nil {
						panic(errors.NewUnreachableError())
					}

					return referenceValue
				},
			)
		},
	)
}

// `(&StorageCapabilityController)` in
// `forEachController(forPath: StoragePath, _ function: fun(&StorageCapabilityController): Bool)`
var accountStorageCapabilitiesForEachControllerCallbackTypeParams = []sema.Type{
	&sema.ReferenceType{
		Type:          sema.StorageCapabilityControllerType,
		Authorization: sema.UnauthorizedAccess,
	},
}

func newAccountStorageCapabilitiesForEachControllerFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_StorageCapabilitiesTypeForEachControllerFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get path argument

			targetPathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
			if !ok || targetPathValue.Domain != common.PathDomainStorage {
				panic(errors.NewUnreachableError())
			}

			// Get function argument

			functionValue, ok := invocation.Arguments[1].(interpreter.FunctionValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Prevent mutations (record/unrecord) to storage capability controllers
			// for this address/path during iteration

			addressPath := interpreter.AddressPath{
				Address: address,
				Path:    targetPathValue,
			}
			iterations := inter.SharedState.CapabilityControllerIterations
			iterations[addressPath]++
			defer func() {
				iterations[addressPath]--
				if iterations[addressPath] <= 0 {
					delete(iterations, addressPath)
				}
			}()

			// Get capability controllers iterator

			nextCapabilityID, _ :=
				getStorageCapabilityControllerIDsIterator(inter, address, targetPathValue)

			for {
				capabilityID, ok := nextCapabilityID()
				if !ok {
					break
				}

				referenceValue := getStorageCapabilityControllerReference(inter, address, capabilityID, locationRange)
				if referenceValue == nil {
					panic(errors.NewUnreachableError())
				}

				subInvocation := interpreter.NewInvocation(
					inter,
					nil,
					nil,
					nil,
					[]interpreter.Value{referenceValue},
					accountStorageCapabilitiesForEachControllerCallbackTypeParams,
					nil,
					locationRange,
				)

				res, err := inter.InvokeFunction(functionValue, subInvocation)
				if err != nil {
					// interpreter panicked while invoking the inner function value
					panic(err)
				}

				shouldContinue, ok := res.(interpreter.BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if !shouldContinue {
					break
				}

				// It is not safe to check this at the beginning of the loop
				// (i.e. on the next invocation of the callback),
				// because if the mutation performed in the callback reorganized storage
				// such that the iteration pointer is now at the end,
				// we will not invoke the callback again but will still silently skip elements of storage.
				//
				// In order to be safe, we perform this check here to effectively enforce
				// that users return `false` from their callback in all cases where storage is mutated.
				if inter.SharedState.MutationDuringCapabilityControllerIteration {
					panic(CapabilityControllersMutatedDuringIterationError{
						LocationRange: locationRange,
					})
				}
			}

			return interpreter.Void
		},
	)
}

func newAccountStorageCapabilitiesIssueFunction(
	gauge common.MemoryGauge,
	idGenerator AccountIDGenerator,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_StorageCapabilitiesTypeIssueFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get path argument

			targetPathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
			if !ok || targetPathValue.Domain != common.PathDomainStorage {
				panic(errors.NewUnreachableError())
			}

			// Get borrow type type argument

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			ty := typeParameterPair.Value

			// Issue capability controller and return capability

			return checkAndIssueStorageCapabilityControllerWithType(
				inter,
				locationRange,
				idGenerator,
				address,
				targetPathValue,
				ty,
			)
		},
	)
}

func newAccountStorageCapabilitiesIssueWithTypeFunction(
	gauge common.MemoryGauge,
	idGenerator AccountIDGenerator,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_StorageCapabilitiesTypeIssueFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get path argument

			targetPathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
			if !ok || targetPathValue.Domain != common.PathDomainStorage {
				panic(errors.NewUnreachableError())
			}

			// Get type argument

			typeValue, ok := invocation.Arguments[1].(interpreter.TypeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			ty, err := inter.ConvertStaticToSemaType(typeValue.Type)
			if err != nil {
				panic(errors.NewUnexpectedErrorFromCause(err))
			}

			// Issue capability controller and return capability

			return checkAndIssueStorageCapabilityControllerWithType(
				inter,
				locationRange,
				idGenerator,
				address,
				targetPathValue,
				ty,
			)
		},
	)
}

func checkAndIssueStorageCapabilityControllerWithType(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	idGenerator AccountIDGenerator,
	address common.Address,
	targetPathValue interpreter.PathValue,
	ty sema.Type,
) *interpreter.CapabilityValue {

	borrowType, ok := ty.(*sema.ReferenceType)
	if !ok {
		panic(interpreter.InvalidCapabilityIssueTypeError{
			ExpectedTypeDescription: "reference type",
			ActualType:              ty,
			LocationRange:           locationRange,
		})
	}

	// Issue capability controller

	capabilityIDValue, borrowStaticType := IssueStorageCapabilityController(
		inter,
		locationRange,
		idGenerator,
		address,
		borrowType,
		targetPathValue,
	)

	// Return controller's capability

	return interpreter.NewCapabilityValue(
		inter,
		capabilityIDValue,
		interpreter.NewAddressValue(inter, address),
		borrowStaticType,
	)
}

func IssueStorageCapabilityController(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	idGenerator AccountIDGenerator,
	address common.Address,
	borrowType *sema.ReferenceType,
	targetPathValue interpreter.PathValue,
) (
	interpreter.UInt64Value,
	*interpreter.ReferenceStaticType,
) {
	// Create and write StorageCapabilityController

	borrowStaticType := interpreter.ConvertSemaReferenceTypeToStaticReferenceType(inter, borrowType)

	var capabilityID uint64
	var err error
	errors.WrapPanic(func() {
		capabilityID, err = idGenerator.GenerateAccountID(address)
	})
	if err != nil {
		panic(interpreter.WrappedExternalError(err))
	}
	if capabilityID == 0 {
		panic(errors.NewUnexpectedError("invalid zero account ID"))
	}

	capabilityIDValue := interpreter.UInt64Value(capabilityID)

	controller := interpreter.NewStorageCapabilityControllerValue(
		inter,
		borrowStaticType,
		capabilityIDValue,
		targetPathValue,
	)

	storeCapabilityController(inter, address, capabilityIDValue, controller)
	recordStorageCapabilityController(inter, locationRange, address, targetPathValue, capabilityIDValue)

	return capabilityIDValue, borrowStaticType
}

func newAccountAccountCapabilitiesIssueFunction(
	gauge common.MemoryGauge,
	idGenerator AccountIDGenerator,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_AccountCapabilitiesTypeIssueFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get borrow type type argument

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			ty := typeParameterPair.Value

			// Issue capability controller and return capability

			return checkAndIssueAccountCapabilityControllerWithType(
				inter,
				locationRange,
				idGenerator,
				address,
				ty,
			)
		},
	)
}

func newAccountAccountCapabilitiesIssueWithTypeFunction(
	gauge common.MemoryGauge,
	idGenerator AccountIDGenerator,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_AccountCapabilitiesTypeIssueFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get type argument

			typeValue, ok := invocation.Arguments[0].(interpreter.TypeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			ty, err := inter.ConvertStaticToSemaType(typeValue.Type)
			if err != nil {
				panic(errors.NewUnexpectedErrorFromCause(err))
			}

			// Issue capability controller and return capability

			return checkAndIssueAccountCapabilityControllerWithType(
				inter,
				locationRange,
				idGenerator,
				address,
				ty,
			)
		},
	)
}

func checkAndIssueAccountCapabilityControllerWithType(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	idGenerator AccountIDGenerator,
	address common.Address,
	ty sema.Type,
) *interpreter.CapabilityValue {

	// Get and check borrow type

	typeBound := sema.AccountReferenceType
	if !sema.IsSubType(ty, typeBound) {
		panic(interpreter.InvalidCapabilityIssueTypeError{
			ExpectedTypeDescription: fmt.Sprintf("`%s`", typeBound.QualifiedString()),
			ActualType:              ty,
			LocationRange:           locationRange,
		})
	}

	borrowType, ok := ty.(*sema.ReferenceType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Issue capability controller

	capabilityIDValue, borrowStaticType :=
		IssueAccountCapabilityController(
			inter,
			locationRange,
			idGenerator,
			address,
			borrowType,
		)

	// Return controller's capability

	return interpreter.NewCapabilityValue(
		inter,
		capabilityIDValue,
		interpreter.NewAddressValue(inter, address),
		borrowStaticType,
	)
}

func IssueAccountCapabilityController(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	idGenerator AccountIDGenerator,
	address common.Address,
	borrowType *sema.ReferenceType,
) (
	interpreter.UInt64Value,
	*interpreter.ReferenceStaticType,
) {
	// Create and write AccountCapabilityController

	borrowStaticType := interpreter.ConvertSemaReferenceTypeToStaticReferenceType(inter, borrowType)

	var capabilityID uint64
	var err error
	errors.WrapPanic(func() {
		capabilityID, err = idGenerator.GenerateAccountID(address)
	})
	if err != nil {
		panic(interpreter.WrappedExternalError(err))
	}
	if capabilityID == 0 {
		panic(errors.NewUnexpectedError("invalid zero account ID"))
	}

	capabilityIDValue := interpreter.UInt64Value(capabilityID)

	controller := interpreter.NewAccountCapabilityControllerValue(
		inter,
		borrowStaticType,
		capabilityIDValue,
	)

	storeCapabilityController(inter, address, capabilityIDValue, controller)
	recordAccountCapabilityController(
		inter,
		locationRange,
		address,
		capabilityIDValue,
	)

	return capabilityIDValue, borrowStaticType
}

// CapabilityControllerStorageDomain is the storage domain which stores
// capability controllers by capability ID
const CapabilityControllerStorageDomain = "cap_con"

// storeCapabilityController stores a capability controller in the account's capability ID to controller storage map
func storeCapabilityController(
	inter *interpreter.Interpreter,
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
	controller interpreter.CapabilityControllerValue,
) {
	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue)

	existed := inter.WriteStored(
		address,
		CapabilityControllerStorageDomain,
		storageMapKey,
		controller,
	)
	if existed {
		panic(errors.NewUnreachableError())
	}
}

// removeCapabilityController removes a capability controller from the account's capability ID to controller storage map
func removeCapabilityController(
	inter *interpreter.Interpreter,
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) {
	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue)

	existed := inter.WriteStored(
		address,
		CapabilityControllerStorageDomain,
		storageMapKey,
		nil,
	)
	if !existed {
		panic(errors.NewUnreachableError())
	}

	setCapabilityControllerTag(
		inter,
		address,
		uint64(capabilityIDValue),
		nil,
	)
}

// getCapabilityController gets the capability controller for the given capability ID
func getCapabilityController(
	inter *interpreter.Interpreter,
	address common.Address,
	capabilityID uint64,
) interpreter.CapabilityControllerValue {

	storageMapKey := interpreter.Uint64StorageMapKey(capabilityID)

	readValue := inter.ReadStored(
		address,
		CapabilityControllerStorageDomain,
		storageMapKey,
	)
	if readValue == nil {
		return nil
	}

	controller, ok := readValue.(interpreter.CapabilityControllerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Inject functions
	switch controller := controller.(type) {
	case *interpreter.StorageCapabilityControllerValue:
		capabilityID := controller.CapabilityID

		controller.GetCapability =
			newCapabilityControllerGetCapabilityFunction(address, controller)

		controller.GetTag =
			newCapabilityControllerGetTagFunction(address, capabilityID)
		controller.SetTag =
			newCapabilityControllerSetTagFunction(address, capabilityID)

		controller.Delete =
			newStorageCapabilityControllerDeleteFunction(address, controller)

		controller.SetTarget =
			newStorageCapabilityControllerSetTargetFunction(address, controller)

	case *interpreter.AccountCapabilityControllerValue:
		capabilityID := controller.CapabilityID

		controller.GetCapability =
			newCapabilityControllerGetCapabilityFunction(address, controller)

		controller.GetTag =
			newCapabilityControllerGetTagFunction(address, capabilityID)
		controller.SetTag =
			newCapabilityControllerSetTagFunction(address, capabilityID)

		controller.Delete =
			newAccountCapabilityControllerDeleteFunction(address, controller)
	}

	return controller
}

func getStorageCapabilityControllerReference(
	inter *interpreter.Interpreter,
	address common.Address,
	capabilityID uint64,
	locationRange interpreter.LocationRange,
) *interpreter.EphemeralReferenceValue {

	capabilityController := getCapabilityController(inter, address, capabilityID)
	if capabilityController == nil {
		return nil
	}

	storageCapabilityController, ok := capabilityController.(*interpreter.StorageCapabilityControllerValue)
	if !ok {
		return nil
	}

	return interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.UnauthorizedAccess,
		storageCapabilityController,
		sema.StorageCapabilityControllerType,
		locationRange,
	)
}

func newStorageCapabilityControllerSetTargetFunction(
	address common.Address,
	controller *interpreter.StorageCapabilityControllerValue,
) func(*interpreter.Interpreter, interpreter.LocationRange, interpreter.PathValue) {
	return func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		newTargetPathValue interpreter.PathValue,
	) {
		oldTargetPathValue := controller.TargetPath
		capabilityID := controller.CapabilityID

		unrecordStorageCapabilityController(
			inter,
			locationRange,
			address,
			oldTargetPathValue,
			capabilityID,
		)
		recordStorageCapabilityController(
			inter,
			locationRange,
			address,
			newTargetPathValue,
			capabilityID,
		)
	}
}

func newStorageCapabilityControllerDeleteFunction(
	address common.Address,
	controller *interpreter.StorageCapabilityControllerValue,
) func(*interpreter.Interpreter, interpreter.LocationRange) {
	return func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
	) {
		targetPathValue := controller.TargetPath
		capabilityID := controller.CapabilityID

		unrecordStorageCapabilityController(
			inter,
			locationRange,
			address,
			targetPathValue,
			capabilityID,
		)
		removeCapabilityController(
			inter,
			address,
			capabilityID,
		)
	}
}

var capabilityIDSetStaticType = &interpreter.DictionaryStaticType{
	KeyType:   interpreter.PrimitiveStaticTypeUInt64,
	ValueType: interpreter.NilStaticType,
}

// PathCapabilityStorageDomain is the storage domain which stores
// capability ID dictionaries (sets) by storage path identifier
const PathCapabilityStorageDomain = "path_cap"

func recordStorageCapabilityController(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	address common.Address,
	targetPathValue interpreter.PathValue,
	capabilityIDValue interpreter.UInt64Value,
) {
	if targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewUnreachableError())
	}

	addressPath := interpreter.AddressPath{
		Address: address,
		Path:    targetPathValue,
	}
	if inter.SharedState.CapabilityControllerIterations[addressPath] > 0 {
		inter.SharedState.MutationDuringCapabilityControllerIteration = true
	}

	identifier := targetPathValue.Identifier

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	storageMap := inter.Storage().GetStorageMap(
		address,
		PathCapabilityStorageDomain,
		true,
	)

	setKey := capabilityIDValue
	setValue := interpreter.Nil

	readValue := storageMap.ReadValue(inter, storageMapKey)
	if readValue == nil {
		capabilityIDSet := interpreter.NewDictionaryValueWithAddress(
			inter,
			locationRange,
			capabilityIDSetStaticType,
			address,
			setKey,
			setValue,
		)
		storageMap.SetValue(inter, storageMapKey, capabilityIDSet)
	} else {
		capabilityIDSet := readValue.(*interpreter.DictionaryValue)
		existing := capabilityIDSet.Insert(inter, locationRange, setKey, setValue)
		if existing != interpreter.Nil {
			panic(errors.NewUnreachableError())
		}
	}
}

func getPathCapabilityIDSet(
	inter *interpreter.Interpreter,
	targetPathValue interpreter.PathValue,
	address common.Address,
) *interpreter.DictionaryValue {
	if targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewUnreachableError())
	}

	identifier := targetPathValue.Identifier

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	storageMap := inter.Storage().GetStorageMap(
		address,
		PathCapabilityStorageDomain,
		false,
	)
	if storageMap == nil {
		return nil
	}

	readValue := storageMap.ReadValue(inter, storageMapKey)
	if readValue == nil {
		return nil
	}

	capabilityIDSet, ok := readValue.(*interpreter.DictionaryValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return capabilityIDSet
}

func unrecordStorageCapabilityController(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	address common.Address,
	targetPathValue interpreter.PathValue,
	capabilityIDValue interpreter.UInt64Value,
) {
	addressPath := interpreter.AddressPath{
		Address: address,
		Path:    targetPathValue,
	}
	if inter.SharedState.CapabilityControllerIterations[addressPath] > 0 {
		inter.SharedState.MutationDuringCapabilityControllerIteration = true
	}

	capabilityIDSet := getPathCapabilityIDSet(inter, targetPathValue, address)
	if capabilityIDSet == nil {
		panic(errors.NewUnreachableError())
	}

	existing := capabilityIDSet.Remove(inter, locationRange, capabilityIDValue)
	if existing == interpreter.Nil {
		panic(errors.NewUnreachableError())
	}

	// Remove capability set if empty

	if capabilityIDSet.Count() == 0 {
		storageMap := inter.Storage().GetStorageMap(
			address,
			PathCapabilityStorageDomain,
			true,
		)
		if storageMap == nil {
			panic(errors.NewUnreachableError())
		}

		identifier := targetPathValue.Identifier

		storageMapKey := interpreter.StringStorageMapKey(identifier)

		if !storageMap.RemoveValue(inter, storageMapKey) {
			panic(errors.NewUnreachableError())
		}
	}
}

func getStorageCapabilityControllerIDsIterator(
	inter *interpreter.Interpreter,
	address common.Address,
	targetPathValue interpreter.PathValue,
) (
	nextCapabilityID func() (uint64, bool),
	count uint64,
) {
	capabilityIDSet := getPathCapabilityIDSet(inter, targetPathValue, address)
	if capabilityIDSet == nil {
		return func() (uint64, bool) {
			return 0, false
		}, 0
	}

	iterator := capabilityIDSet.Iterator()

	count = uint64(capabilityIDSet.Count())
	nextCapabilityID = func() (uint64, bool) {
		keyValue := iterator.NextKey(inter)
		if keyValue == nil {
			return 0, false
		}

		capabilityIDValue, ok := keyValue.(interpreter.UInt64Value)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return uint64(capabilityIDValue), true
	}
	return
}

// AccountCapabilityStorageDomain is the storage domain which
// records active account capability controller IDs
const AccountCapabilityStorageDomain = "acc_cap"

func recordAccountCapabilityController(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) {
	addressPath := interpreter.AddressPath{
		Address: address,
	}
	if inter.SharedState.CapabilityControllerIterations[addressPath] > 0 {
		inter.SharedState.MutationDuringCapabilityControllerIteration = true
	}

	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue)

	storageMap := inter.Storage().GetStorageMap(
		address,
		AccountCapabilityStorageDomain,
		true,
	)

	existed := storageMap.SetValue(inter, storageMapKey, interpreter.NilValue{})
	if existed {
		panic(errors.NewUnreachableError())
	}
}

func unrecordAccountCapabilityController(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) {
	addressPath := interpreter.AddressPath{
		Address: address,
	}
	if inter.SharedState.CapabilityControllerIterations[addressPath] > 0 {
		inter.SharedState.MutationDuringCapabilityControllerIteration = true
	}

	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue)

	storageMap := inter.Storage().GetStorageMap(
		address,
		AccountCapabilityStorageDomain,
		true,
	)

	existed := storageMap.RemoveValue(inter, storageMapKey)
	if !existed {
		panic(errors.NewUnreachableError())
	}
}

func getAccountCapabilityControllerIDsIterator(
	inter *interpreter.Interpreter,
	address common.Address,
) (
	nextCapabilityID func() (uint64, bool),
	count uint64,
) {
	storageMap := inter.Storage().GetStorageMap(
		address,
		AccountCapabilityStorageDomain,
		false,
	)
	if storageMap == nil {
		return func() (uint64, bool) {
			return 0, false
		}, 0
	}

	iterator := storageMap.Iterator(inter)

	count = storageMap.Count()
	nextCapabilityID = func() (uint64, bool) {
		keyValue := iterator.NextKey()
		if keyValue == nil {
			return 0, false
		}

		capabilityIDValue, ok := keyValue.(interpreter.Uint64AtreeValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return uint64(capabilityIDValue), true
	}
	return
}

func newAccountCapabilitiesPublishFunction(
	gauge common.MemoryGauge,
	accountAddressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	accountAddress := accountAddressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_CapabilitiesTypePublishFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get capability argument

			var capabilityValue *interpreter.CapabilityValue

			firstValue := invocation.Arguments[0]
			switch firstValue := firstValue.(type) {
			case *interpreter.CapabilityValue:
				capabilityValue = firstValue

			default:
				panic(errors.NewUnreachableError())
			}

			capabilityAddressValue := capabilityValue.Address
			if capabilityAddressValue != accountAddressValue {
				panic(interpreter.CapabilityAddressPublishingError{
					LocationRange:     locationRange,
					CapabilityAddress: capabilityAddressValue,
					AccountAddress:    accountAddressValue,
				})
			}

			// Get path argument

			pathValue, ok := invocation.Arguments[1].(interpreter.PathValue)
			if !ok || pathValue.Domain != common.PathDomainPublic {
				panic(errors.NewUnreachableError())
			}

			domain := pathValue.Domain.Identifier()
			identifier := pathValue.Identifier

			// Prevent an overwrite

			storageMapKey := interpreter.StringStorageMapKey(identifier)

			if inter.StoredValueExists(
				accountAddress,
				domain,
				storageMapKey,
			) {
				panic(interpreter.OverwriteError{
					Address:       accountAddressValue,
					Path:          pathValue,
					LocationRange: locationRange,
				})
			}

			capabilityValue, ok = capabilityValue.Transfer(
				inter,
				locationRange,
				atree.Address(accountAddress),
				true,
				nil,
				nil,
			).(*interpreter.CapabilityValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Write new value

			inter.WriteStored(
				accountAddress,
				domain,
				storageMapKey,
				capabilityValue,
			)

			return interpreter.Void
		},
	)
}

func newAccountCapabilitiesUnpublishFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_CapabilitiesTypeUnpublishFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get path argument

			pathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
			if !ok || pathValue.Domain != common.PathDomainPublic {
				panic(errors.NewUnreachableError())
			}

			domain := pathValue.Domain.Identifier()
			identifier := pathValue.Identifier

			// Read/remove capability

			storageMapKey := interpreter.StringStorageMapKey(identifier)

			readValue := inter.ReadStored(address, domain, storageMapKey)
			if readValue == nil {
				return interpreter.Nil
			}

			var capabilityValue *interpreter.CapabilityValue
			switch readValue := readValue.(type) {
			case *interpreter.CapabilityValue:
				capabilityValue = readValue

			default:
				panic(errors.NewUnreachableError())
			}

			capabilityValue, ok = capabilityValue.Transfer(
				inter,
				locationRange,
				atree.Address{},
				true,
				nil,
				nil,
			).(*interpreter.CapabilityValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter.WriteStored(
				address,
				domain,
				storageMapKey,
				nil,
			)

			return interpreter.NewSomeValueNonCopying(inter, capabilityValue)
		},
	)
}

func getCheckedCapabilityController(
	inter *interpreter.Interpreter,
	capabilityAddressValue interpreter.AddressValue,
	capabilityIDValue interpreter.UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) (
	interpreter.CapabilityControllerValue,
	*sema.ReferenceType,
) {

	if wantedBorrowType == nil {
		wantedBorrowType = capabilityBorrowType
	} else if !sema.IsSubType(capabilityBorrowType, wantedBorrowType) {
		// Ensure wanted borrow type is not more permissive
		// than the capability's borrow type:
		// The wanted type must be a supertype

		return nil, nil
	}

	capabilityAddress := capabilityAddressValue.ToAddress()
	capabilityID := uint64(capabilityIDValue)

	controller := getCapabilityController(inter, capabilityAddress, capabilityID)
	if controller == nil {
		return nil, nil
	}

	// Ensure wanted borrow type is not more permissive
	// than the controller's borrow type:
	// The wanted type must be a supertype

	controllerBorrowStaticType := controller.CapabilityControllerBorrowType()

	controllerBorrowType, ok :=
		inter.MustConvertStaticToSemaType(controllerBorrowStaticType).(*sema.ReferenceType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	if !sema.IsSubType(controllerBorrowType, wantedBorrowType) {
		return nil, nil
	}

	return controller, wantedBorrowType
}

func GetCheckedCapabilityControllerReference(
	inter *interpreter.Interpreter,
	capabilityAddressValue interpreter.AddressValue,
	capabilityIDValue interpreter.UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
	locationRange interpreter.LocationRange,
) interpreter.ReferenceValue {
	controller, resultBorrowType := getCheckedCapabilityController(
		inter,
		capabilityAddressValue,
		capabilityIDValue,
		wantedBorrowType,
		capabilityBorrowType,
	)
	if controller == nil {
		return nil
	}

	capabilityAddress := capabilityAddressValue.ToAddress()

	return controller.ReferenceValue(
		inter,
		capabilityAddress,
		resultBorrowType,
		locationRange,
	)
}

func BorrowCapabilityController(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	capabilityAddress interpreter.AddressValue,
	capabilityID interpreter.UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) interpreter.ReferenceValue {
	referenceValue := GetCheckedCapabilityControllerReference(
		inter,
		capabilityAddress,
		capabilityID,
		wantedBorrowType,
		capabilityBorrowType,
		locationRange,
	)
	if referenceValue == nil {
		return nil
	}

	// Attempt to dereference,
	// which reads the stored value
	// and performs a dynamic type check

	referencedValue := referenceValue.ReferencedValue(
		inter,
		locationRange,
		false,
	)
	if referencedValue == nil {
		return nil
	}

	return referenceValue
}

func CheckCapabilityController(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	capabilityAddress interpreter.AddressValue,
	capabilityID interpreter.UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) interpreter.BoolValue {
	referenceValue := GetCheckedCapabilityControllerReference(
		inter,
		capabilityAddress,
		capabilityID,
		wantedBorrowType,
		capabilityBorrowType,
		locationRange,
	)
	if referenceValue == nil {
		return interpreter.FalseValue
	}

	// Attempt to dereference,
	// which reads the stored value
	// and performs a dynamic type check

	referencedValue := referenceValue.ReferencedValue(
		inter,
		locationRange,
		false,
	)

	return interpreter.AsBoolValue(referencedValue != nil)
}

func newAccountCapabilitiesGetFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
	borrow bool,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()

	var funcType *sema.FunctionType

	if borrow {
		funcType = sema.Account_CapabilitiesTypeBorrowFunctionType
	} else {
		funcType = sema.Account_CapabilitiesTypeGetFunctionType
	}

	return interpreter.NewHostFunctionValue(
		gauge,
		funcType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get path argument

			pathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
			if !ok || pathValue.Domain != common.PathDomainPublic {
				panic(errors.NewUnreachableError())
			}

			domain := pathValue.Domain.Identifier()
			identifier := pathValue.Identifier

			// Get borrow type type argument

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			wantedBorrowType, ok := typeParameterPair.Value.(*sema.ReferenceType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Read stored capability, if any

			storageMapKey := interpreter.StringStorageMapKey(identifier)

			readValue := inter.ReadStored(address, domain, storageMapKey)
			if readValue == nil {
				return interpreter.Nil
			}

			var readCapabilityValue *interpreter.CapabilityValue

			switch readValue := readValue.(type) {
			case *interpreter.CapabilityValue:
				readCapabilityValue = readValue

			default:
				panic(errors.NewUnreachableError())
			}

			capabilityBorrowType, ok :=
				inter.MustConvertStaticToSemaType(readCapabilityValue.BorrowType).(*sema.ReferenceType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			capabilityID := readCapabilityValue.ID
			capabilityAddress := readCapabilityValue.Address

			var resultValue interpreter.Value
			if borrow {
				// When borrowing,
				// check the controller and types,
				// and return a checked reference

				resultValue = BorrowCapabilityController(
					inter,
					locationRange,
					capabilityAddress,
					capabilityID,
					wantedBorrowType,
					capabilityBorrowType,
				)
			} else {
				// When not borrowing,
				// check the controller and types,
				// and return a capability

				controller, resultBorrowType := getCheckedCapabilityController(
					inter,
					capabilityAddress,
					capabilityID,
					wantedBorrowType,
					capabilityBorrowType,
				)
				if controller != nil {
					resultBorrowStaticType :=
						interpreter.ConvertSemaReferenceTypeToStaticReferenceType(inter, resultBorrowType)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					resultValue = interpreter.NewCapabilityValue(
						inter,
						capabilityID,
						capabilityAddress,
						resultBorrowStaticType,
					)
				}
			}

			if resultValue == nil {
				return interpreter.Nil
			}

			return interpreter.NewSomeValueNonCopying(
				inter,
				resultValue,
			)
		},
	)
}

func newAccountCapabilitiesExistsFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_CapabilitiesTypeExistsFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter

			// Get path argument

			pathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
			if !ok || pathValue.Domain != common.PathDomainPublic {
				panic(errors.NewUnreachableError())
			}

			domain := pathValue.Domain.Identifier()
			identifier := pathValue.Identifier

			// Read stored capability, if any

			storageMapKey := interpreter.StringStorageMapKey(identifier)

			return interpreter.AsBoolValue(
				inter.StoredValueExists(address, domain, storageMapKey),
			)
		},
	)
}

func getAccountCapabilityControllerReference(
	inter *interpreter.Interpreter,
	address common.Address,
	capabilityID uint64,
	locationRange interpreter.LocationRange,
) *interpreter.EphemeralReferenceValue {

	capabilityController := getCapabilityController(inter, address, capabilityID)
	if capabilityController == nil {
		return nil
	}

	accountCapabilityController, ok := capabilityController.(*interpreter.AccountCapabilityControllerValue)
	if !ok {
		return nil
	}

	return interpreter.NewEphemeralReferenceValue(
		inter,
		interpreter.UnauthorizedAccess,
		accountCapabilityController,
		sema.AccountCapabilityControllerType,
		locationRange,
	)
}

func newAccountAccountCapabilitiesGetControllerFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) interpreter.FunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_AccountCapabilitiesTypeGetControllerFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get capability ID argument

			capabilityIDValue, ok := invocation.Arguments[0].(interpreter.UInt64Value)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			capabilityID := uint64(capabilityIDValue)

			referenceValue := getAccountCapabilityControllerReference(inter, address, capabilityID, locationRange)
			if referenceValue == nil {
				return interpreter.Nil
			}

			return interpreter.NewSomeValueNonCopying(inter, referenceValue)
		},
	)
}

var accountCapabilityControllerReferencesArrayStaticType = &interpreter.VariableSizedStaticType{
	Type: &interpreter.ReferenceStaticType{
		ReferencedType: interpreter.PrimitiveStaticTypeAccountCapabilityController,
		Authorization:  interpreter.UnauthorizedAccess,
	},
}

func newAccountAccountCapabilitiesGetControllersFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) interpreter.FunctionValue {
	address := addressValue.ToAddress()
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_AccountCapabilitiesTypeGetControllersFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get capability controllers iterator

			nextCapabilityID, count :=
				getAccountCapabilityControllerIDsIterator(inter, address)

			var capabilityControllerIndex uint64 = 0

			return interpreter.NewArrayValueWithIterator(
				inter,
				accountCapabilityControllerReferencesArrayStaticType,
				common.Address{},
				count,
				func() interpreter.Value {
					if capabilityControllerIndex >= count {
						return nil
					}
					capabilityControllerIndex++

					capabilityID, ok := nextCapabilityID()
					if !ok {
						return nil
					}

					referenceValue := getAccountCapabilityControllerReference(
						inter,
						address,
						capabilityID,
						locationRange,
					)
					if referenceValue == nil {
						panic(errors.NewUnreachableError())
					}

					return referenceValue
				},
			)
		},
	)
}

// `(&AccountCapabilityController)` in
// `forEachController(_ function: fun(&AccountCapabilityController): Bool)`
var accountAccountCapabilitiesForEachControllerCallbackTypeParams = []sema.Type{
	&sema.ReferenceType{
		Type:          sema.AccountCapabilityControllerType,
		Authorization: sema.UnauthorizedAccess,
	},
}

// CapabilityControllersMutatedDuringIterationError
type CapabilityControllersMutatedDuringIterationError struct {
	interpreter.LocationRange
}

var _ errors.UserError = CapabilityControllersMutatedDuringIterationError{}

func (CapabilityControllersMutatedDuringIterationError) IsUserError() {}

func (CapabilityControllersMutatedDuringIterationError) Error() string {
	return "capability controller iteration continued after changes to controllers"
}

func newAccountAccountCapabilitiesForEachControllerFunction(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.Account_AccountCapabilitiesTypeForEachControllerFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get function argument

			functionValue, ok := invocation.Arguments[0].(interpreter.FunctionValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Prevent mutations (record/unrecord) to account capability controllers
			// for this address during iteration

			addressPath := interpreter.AddressPath{
				Address: address,
			}
			iterations := inter.SharedState.CapabilityControllerIterations
			iterations[addressPath]++
			defer func() {
				iterations[addressPath]--
				if iterations[addressPath] <= 0 {
					delete(iterations, addressPath)
				}
			}()

			// Get capability controllers iterator

			nextCapabilityID, _ :=
				getAccountCapabilityControllerIDsIterator(inter, address)

			for {
				capabilityID, ok := nextCapabilityID()
				if !ok {
					break
				}

				referenceValue := getAccountCapabilityControllerReference(inter, address, capabilityID, locationRange)
				if referenceValue == nil {
					panic(errors.NewUnreachableError())
				}

				subInvocation := interpreter.NewInvocation(
					inter,
					nil,
					nil,
					nil,
					[]interpreter.Value{referenceValue},
					accountAccountCapabilitiesForEachControllerCallbackTypeParams,
					nil,
					locationRange,
				)

				res, err := inter.InvokeFunction(functionValue, subInvocation)
				if err != nil {
					// interpreter panicked while invoking the inner function value
					panic(err)
				}

				shouldContinue, ok := res.(interpreter.BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if !shouldContinue {
					break
				}

				// It is not safe to check this at the beginning of the loop
				// (i.e. on the next invocation of the callback),
				// because if the mutation performed in the callback reorganized storage
				// such that the iteration pointer is now at the end,
				// we will not invoke the callback again but will still silently skip elements of storage.
				//
				// In order to be safe, we perform this check here to effectively enforce
				// that users return `false` from their callback in all cases where storage is mutated.
				if inter.SharedState.MutationDuringCapabilityControllerIteration {
					panic(CapabilityControllersMutatedDuringIterationError{
						LocationRange: locationRange,
					})
				}
			}

			return interpreter.Void
		},
	)
}

func newAccountCapabilityControllerDeleteFunction(
	address common.Address,
	controller *interpreter.AccountCapabilityControllerValue,
) func(*interpreter.Interpreter, interpreter.LocationRange) {
	return func(inter *interpreter.Interpreter, locationRange interpreter.LocationRange) {
		capabilityID := controller.CapabilityID

		unrecordAccountCapabilityController(
			inter,
			locationRange,
			address,
			capabilityID,
		)
		removeCapabilityController(
			inter,
			address,
			capabilityID,
		)
	}
}

// CapabilityControllerTagStorageDomain is the storage domain which stores
// capability controller tags by capability ID
const CapabilityControllerTagStorageDomain = "cap_tag"

func getCapabilityControllerTag(
	inter *interpreter.Interpreter,
	address common.Address,
	capabilityID uint64,
) *interpreter.StringValue {

	value := inter.ReadStored(
		address,
		CapabilityControllerTagStorageDomain,
		interpreter.Uint64StorageMapKey(capabilityID),
	)
	if value == nil {
		return interpreter.EmptyString
	}

	stringValue, ok := value.(*interpreter.StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return stringValue
}

func newCapabilityControllerGetCapabilityFunction(
	address common.Address,
	controller interpreter.CapabilityControllerValue,
) func(inter *interpreter.Interpreter) *interpreter.CapabilityValue {

	addressValue := interpreter.AddressValue(address)
	capabilityID := controller.ControllerCapabilityID()
	borrowType := controller.CapabilityControllerBorrowType()

	return func(inter *interpreter.Interpreter) *interpreter.CapabilityValue {
		return interpreter.NewCapabilityValue(
			inter,
			capabilityID,
			addressValue,
			borrowType,
		)
	}
}

func newCapabilityControllerGetTagFunction(
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) func(*interpreter.Interpreter) *interpreter.StringValue {

	return func(inter *interpreter.Interpreter) *interpreter.StringValue {
		return getCapabilityControllerTag(
			inter,
			address,
			uint64(capabilityIDValue),
		)
	}
}

func setCapabilityControllerTag(
	inter *interpreter.Interpreter,
	address common.Address,
	capabilityID uint64,
	tagValue *interpreter.StringValue,
) {
	// avoid typed nil
	var value interpreter.Value
	if tagValue != nil {
		value = tagValue
	}

	inter.WriteStored(
		address,
		CapabilityControllerTagStorageDomain,
		interpreter.Uint64StorageMapKey(capabilityID),
		value,
	)
}

func newCapabilityControllerSetTagFunction(
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) func(*interpreter.Interpreter, *interpreter.StringValue) {
	return func(inter *interpreter.Interpreter, tagValue *interpreter.StringValue) {
		setCapabilityControllerTag(
			inter,
			address,
			uint64(capabilityIDValue),
			tagValue,
		)
	}
}
