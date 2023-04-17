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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
)

const authAccountFunctionDocString = `
Creates a new account, paid by the given existing account
`

var authAccountFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
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
		locationRange interpreter.LocationRange,
	)
}

type AuthAccountHandler interface {
	BalanceProvider
	AvailableBalanceProvider
	StorageUsedProvider
	StorageCapacityProvider
	AccountEncodedKeyAdditionHandler
	AccountEncodedKeyRevocationHandler
	AuthAccountKeysHandler
	AuthAccountContractsHandler
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
			locationRange := invocation.LocationRange

			inter.ExpectType(
				payer,
				sema.AuthAccountType,
				locationRange,
			)

			payerValue := payer.GetMember(
				inter,
				locationRange,
				sema.AuthAccountTypeAddressFieldName,
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
					errors.WrapPanic(func() {
						address, err = creator.CreateAccount(payerAddress)
					})
					if err != nil {
						panic(err)
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
	Parameters: []sema.Parameter{{
		Label:          sema.ArgumentLabelNotRequired,
		Identifier:     "address",
		TypeAnnotation: sema.NewTypeAnnotation(sema.TheAddressType),
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
		newAddPublicKeyFunction(gauge, handler, addressValue),
		newRemovePublicKeyFunction(gauge, handler, addressValue),
		func() interpreter.Value {
			return newAuthAccountContractsValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAuthAccountKeysValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAuthAccountInboxValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			capabilities := newAuthAccountCapabilitiesValue(
				gauge,
				addressValue,
			)
			return interpreter.NewEphemeralReferenceValue(
				gauge,
				false,
				capabilities,
				sema.AuthAccountTypeCapabilitiesFieldType.Type,
			)
		},
	)
}

type AuthAccountContractsHandler interface {
	AccountContractProvider
	AccountContractAdditionHandler
	AccountContractRemovalHandler
	AccountContractNamesProvider
}

func newAuthAccountContractsValue(
	gauge common.MemoryGauge,
	handler AuthAccountContractsHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAuthAccountContractsValue(
		gauge,
		addressValue,
		newAuthAccountContractsChangeFunction(
			sema.AuthAccountContractsTypeAddFunctionType,
			gauge,
			handler,
			addressValue,
			false,
		),
		newAuthAccountContractsChangeFunction(
			sema.AuthAccountContractsTypeUpdate__experimentalFunctionType,
			gauge,
			handler,
			addressValue,
			true,
		),
		newAccountContractsGetFunction(
			sema.AuthAccountContractsTypeGetFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountContractsBorrowFunction(
			sema.AuthAccountContractsTypeBorrowFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAuthAccountContractsRemoveFunction(
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

type AuthAccountKeysHandler interface {
	AccountKeyProvider
	AccountKeyAdditionHandler
	AccountKeyRevocationHandler
}

func newAuthAccountKeysValue(
	gauge common.MemoryGauge,
	handler AuthAccountKeysHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAuthAccountKeysValue(
		gauge,
		addressValue,
		newAccountKeysAddFunction(
			gauge,
			handler,
			addressValue,
		),
		newAccountKeysGetFunction(
			sema.AuthAccountKeysTypeGetFunctionType,
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
			sema.AuthAccountKeysTypeForEachFunctionType,
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
			errors.WrapPanic(func() {
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
	CommitStorageTemporarily(inter *interpreter.Interpreter) error
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
					panic(err)
				}
				return capacity
			},
		)
	}
}

type StorageCapacityProvider interface {
	CommitStorageTemporarily(inter *interpreter.Interpreter) error
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
					panic(err)
				}
				return capacity
			},
		)

	}
}

type AccountEncodedKeyAdditionHandler interface {
	EventEmitter
	// AddEncodedAccountKey appends an encoded key to an account.
	AddEncodedAccountKey(address common.Address, key []byte) error
}

func newAddPublicKeyFunction(
	gauge common.MemoryGauge,
	handler AccountEncodedKeyAdditionHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.AuthAccountTypeAddPublicKeyFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			publicKeyValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			locationRange := invocation.LocationRange

			publicKey, err := interpreter.ByteArrayValueToByteSlice(gauge, publicKeyValue, locationRange)
			if err != nil {
				panic("addPublicKey requires the first argument to be a byte array")
			}

			errors.WrapPanic(func() {
				err = handler.AddEncodedAccountKey(address, publicKey)
			})
			if err != nil {
				panic(err)
			}

			inter := invocation.Interpreter
			handler.EmitEvent(
				inter,
				AccountKeyAddedEventType,
				[]interpreter.Value{
					addressValue,
					publicKeyValue,
				},
				locationRange,
			)

			return interpreter.Void
		},
	)
}

type AccountEncodedKeyRevocationHandler interface {
	EventEmitter
	// RevokeEncodedAccountKey removes a key from an account by index, add returns the encoded key.
	RevokeEncodedAccountKey(address common.Address, index int) ([]byte, error)
}

func newRemovePublicKeyFunction(
	gauge common.MemoryGauge,
	handler AccountEncodedKeyRevocationHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.AuthAccountTypeRemovePublicKeyFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			index, ok := invocation.Arguments[0].(interpreter.IntValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			var publicKey []byte
			var err error
			errors.WrapPanic(func() {
				publicKey, err = handler.RevokeEncodedAccountKey(address, index.ToInt(invocation.LocationRange))
			})
			if err != nil {
				panic(err)
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			publicKeyValue := interpreter.ByteSliceToByteArrayValue(
				inter,
				publicKey,
			)

			handler.EmitEvent(
				inter,
				AccountKeyRemovedEventType,
				[]interpreter.Value{
					addressValue,
					publicKeyValue,
				},
				locationRange,
			)

			return interpreter.Void
		},
	)
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
		sema.AuthAccountKeysTypeAddFunctionType,
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
				panic(err)
			}

			handler.EmitEvent(
				inter,
				AccountKeyAddedEventType,
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
				handler,
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
				panic(err)
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
					provider,
					provider,
				),
			)
		},
	)
}

// the AccountKey in `forEachKey(_ f: ((AccountKey): Bool)): Void`
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
					provider,
					provider,
				)
			}

			var count uint64
			var err error

			errors.WrapPanic(func() {
				count, err = provider.AccountKeysCount(address)
			})

			if err != nil {
				panic(err)
			}

			var accountKey *AccountKey

			for index := uint64(0); index < count; index++ {
				errors.WrapPanic(func() {
					accountKey, err = provider.GetAccountKey(address, int(index))
				})
				if err != nil {
					panic(err)
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
				panic(err)
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
		sema.AuthAccountKeysTypeRevokeFunctionType,
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
				panic(err)
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
				AccountKeyRemovedEventType,
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
					handler,
					handler,
				),
			)
		},
	)
}

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
			sema.PublicAccountKeysTypeGetFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountKeysForEachFunction(
			sema.PublicAccountKeysTypeForEachFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountKeysCountGetter(
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
			sema.PublicAccountContractsTypeGetFunctionType,
			gauge,
			handler,
			addressValue,
		),
		newAccountContractsBorrowFunction(
			sema.PublicAccountContractsTypeBorrowFunctionType,
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

const inboxStorageDomain = "inbox"

func accountInboxPublishFunction(
	gauge common.MemoryGauge,
	handler EventEmitter,
	address common.Address,
	providerValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.AuthAccountInboxTypePublishFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			value, ok := invocation.Arguments[0].(*interpreter.StorageCapabilityValue)
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
				atree.Address(address),
				true,
				nil,
			)

			inter.WriteStored(address, inboxStorageDomain, nameValue.Str, publishedValue)

			return interpreter.Void
		},
	)
}

func accountInboxUnpublishFunction(
	gauge common.MemoryGauge,
	handler EventEmitter,
	address common.Address,
	providerValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.AuthAccountInboxTypePublishFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			readValue := inter.ReadStored(address, inboxStorageDomain, nameValue.Str)
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
			)

			inter.WriteStored(address, inboxStorageDomain, nameValue.Str, nil)

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

func accountInboxClaimFunction(
	gauge common.MemoryGauge,
	handler EventEmitter,
	recipientValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		gauge,
		sema.AuthAccountInboxTypePublishFunctionType,
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

			readValue := inter.ReadStored(providerAddress, inboxStorageDomain, nameValue.Str)
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
			)

			inter.WriteStored(providerAddress, inboxStorageDomain, nameValue.Str, nil)

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

func newAuthAccountInboxValue(
	gauge common.MemoryGauge,
	handler EventEmitter,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	address := addressValue.ToAddress()
	return interpreter.NewAuthAccountInboxValue(
		gauge,
		addressValue,
		accountInboxPublishFunction(gauge, handler, address, addressValue),
		accountInboxUnpublishFunction(gauge, handler, address, addressValue),
		accountInboxClaimFunction(gauge, handler, addressValue),
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
				return interpreter.Nil
			}
		},
	)
}

func newAccountContractsBorrowFunction(
	functionType *sema.FunctionType,
	gauge common.MemoryGauge,
	handler PublicAccountContractsHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		functionType,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.Interpreter

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
				panic(err)
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

			reference := interpreter.NewEphemeralReferenceValue(
				inter,
				false,
				contractValue,
				referenceType.Type,
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
	RecordContractUpdate(location common.AddressLocation, value *interpreter.CompositeValue)
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
}

// newAuthAccountContractsChangeFunction called when e.g.
// - adding: `AuthAccount.contracts.add(name: "Foo", code: [...])` (isUpdate = false)
// - updating: `AuthAccount.contracts.update__experimental(name: "Foo", code: [...])` (isUpdate = true)
func newAuthAccountContractsChangeFunction(
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

			code, err := interpreter.ByteArrayValueToByteSlice(gauge, newCodeValue, locationRange)
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
				// Ensure that no contract/contract interface with the given name exists already

				if len(existingCode) > 0 {
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
					gauge,
					oldCode,
					parser.Config{},
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
	// If the code declares a contract, instantiate it and store it.
	//
	// This function might be called when
	// 1. A contract is deployed (contractType is non-nil).
	// 2. A contract interface is deployed (contractType is nil).
	//
	// If a contract is deployed, it is only instantiated
	// when options.createContract is true,
	// i.e. the Cadence `add` function is used.
	// If the Cadence `update__experimental` function is used,
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
		return err
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

func newAuthAccountContractsRemoveFunction(
	gauge common.MemoryGauge,
	handler AccountContractRemovalHandler,
	addressValue interpreter.AddressValue,
) *interpreter.HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return interpreter.NewHostFunctionValue(
		gauge,
		sema.AuthAccountContractsTypeRemoveFunctionType,
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
				panic(err)
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
					panic(err)
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
Returns the public account for the given address
`

var getAccountFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "address",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.TheAddressType,
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
			return newPublicAccountKeysValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newPublicAccountContractsValue(
				gauge,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			capabilities := newPublicAccountCapabilitiesValue(
				gauge,
				addressValue,
			)
			return interpreter.NewEphemeralReferenceValue(
				gauge,
				false,
				capabilities,
				sema.PublicAccountTypeCapabilitiesFieldType.Type,
			)
		},
	)
}

func NewAccountKeyValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	accountKey *AccountKey,
	publicKeySignatureVerifier PublicKeySignatureVerifier,
	blsPoPVerifier BLSPoPVerifier,
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
			publicKeySignatureVerifier,
			blsPoPVerifier,
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
	return interpreter.ByteSliceToByteArrayValue(inter, codeHash[:])
}

func newAuthAccountStorageCapabilitiesValue(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	// TODO:
	return interpreter.NewAuthAccountStorageCapabilitiesValue(
		gauge,
		addressValue,
		nil,
		nil,
		nil,
		nil,
	)
}

func newAuthAccountAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	// TODO:
	return interpreter.NewAuthAccountAccountCapabilitiesValue(
		gauge,
		addressValue,
		nil,
		nil,
		nil,
		nil,
	)
}

func newAuthAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	// TODO:
	return interpreter.NewAuthAccountCapabilitiesValue(
		gauge,
		addressValue,
		nil,
		nil,
		nil,
		nil,
		func() interpreter.Value {
			storageCapabilities := newAuthAccountStorageCapabilitiesValue(
				gauge,
				addressValue,
			)
			return interpreter.NewEphemeralReferenceValue(
				gauge,
				false,
				storageCapabilities,
				sema.AuthAccountCapabilitiesTypeStorageFieldType.Type,
			)
		},
		func() interpreter.Value {
			accountCapabilities := newAuthAccountAccountCapabilitiesValue(
				gauge,
				addressValue,
			)
			return interpreter.NewEphemeralReferenceValue(
				gauge,
				false,
				accountCapabilities,
				sema.AuthAccountCapabilitiesTypeAccountFieldType.Type,
			)
		},
	)
}

func newPublicAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	// TODO:
	return interpreter.NewPublicAccountCapabilitiesValue(
		gauge,
		addressValue,
		nil,
		nil,
	)
}
