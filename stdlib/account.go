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

package stdlib

import (
	goerrors "errors"
	"fmt"

	"golang.org/x/crypto/sha3"
	"golang.org/x/xerrors"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
)

const accountFunctionName = "Account"

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
		context interpreter.ValueExportContext,
		locationRange interpreter.LocationRange,
		eventType *sema.CompositeType,
		values []interpreter.Value,
	)
}

type AccountIDGenerator interface {
	// GenerateAccountID generates a new, *non-zero*, unique ID for the given account.
	GenerateAccountID(address common.Address) (uint64, error)
}

type StorageCommitter interface {
	CommitStorageTemporarily(context interpreter.ValueTransferContext) error
}

type CapabilityControllerIssueHandler interface {
	EventEmitter
	AccountIDGenerator
}

type CapabilityControllerHandler interface {
	EventEmitter
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

func NewInterpreterAccountConstructor(creator AccountCreator) StandardLibraryValue {
	return NewInterpreterStandardLibraryStaticFunction(
		accountFunctionName,
		accountFunctionType,
		accountFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {

			payer, ok := invocation.Arguments[0].(interpreter.MemberAccessibleValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.InvocationContext
			locationRange := invocation.LocationRange

			return NewAccount(
				inter,
				payer,
				locationRange,
				creator,
			)
		},
	)
}

func NewVMAccountConstructor(creator AccountCreator) StandardLibraryValue {
	return NewVMStandardLibraryStaticFunction(
		accountFunctionName,
		accountFunctionType,
		accountFunctionDocString,
		func(context *vm.Context, _ []bbq.StaticType, arguments ...interpreter.Value) interpreter.Value {

			payer, ok := arguments[0].(interpreter.MemberAccessibleValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return NewAccount(
				context,
				payer,
				interpreter.EmptyLocationRange,
				creator,
			)
		},
	)
}

func NewAccount(
	context interpreter.MemberAccessibleContext,
	payer interpreter.MemberAccessibleValue,
	locationRange interpreter.LocationRange,
	creator AccountCreator,
) interpreter.Value {

	interpreter.ExpectType(
		context,
		payer,
		sema.AccountReferenceType,
		locationRange,
	)

	payerValue := payer.GetMember(
		context,
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

	err := creator.CommitStorageTemporarily(context)
	if err != nil {
		panic(err)
	}

	addressValue := interpreter.NewAddressValueFromConstructor(
		context,
		func() (address common.Address) {
			var err error
			address, err = creator.CreateAccount(payerAddress)
			if err != nil {
				panic(err)
			}

			return
		},
	)

	creator.EmitEvent(
		context,
		locationRange,
		AccountCreatedEventType,
		[]interpreter.Value{addressValue},
	)

	return NewAccountReferenceValue(
		context,
		creator,
		addressValue,
		interpreter.FullyEntitledAccountAccess,
		locationRange,
	)
}

const GetAuthAccountFunctionName = "getAuthAccount"
const getAuthAccountFunctionDocString = `
Returns the account for the given address. Only available in scripts
`

// GetAuthAccountFunctionType represents the type
//
//	fun getAuthAccount<T: &Account>(_ address: Address): T
var GetAuthAccountFunctionType = func() *sema.FunctionType {

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

func NewInterpreterGetAuthAccountFunction(handler AccountHandler) StandardLibraryValue {
	return NewInterpreterStandardLibraryStaticFunction(
		GetAuthAccountFunctionName,
		GetAuthAccountFunctionType,
		getAuthAccountFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			accountAddress, ok := invocation.Arguments[0].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.InvocationContext
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

func NewVMGetAuthAccountFunction(handler AccountHandler) StandardLibraryValue {
	return NewVMStandardLibraryStaticFunction(
		GetAuthAccountFunctionName,
		GetAuthAccountFunctionType,
		getAuthAccountFunctionDocString,
		func(context *vm.Context, typeArguments []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
			accountAddress, ok := args[0].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			referenceType, ok := typeArguments[0].(*interpreter.ReferenceStaticType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return NewAccountReferenceValue(
				context,
				handler,
				accountAddress,
				referenceType.Authorization,
				interpreter.EmptyLocationRange,
			)
		},
	)
}

func NewAccountReferenceValue(
	context interpreter.AccountCreationContext,
	handler AccountHandler,
	addressValue interpreter.AddressValue,
	authorization interpreter.Authorization,
	locationRange interpreter.LocationRange,
) interpreter.Value {
	account := NewAccountValue(context, handler, addressValue)
	return interpreter.NewEphemeralReferenceValue(
		context,
		authorization,
		account,
		sema.AccountType,
		locationRange,
	)
}

func NewAccountValue(
	context interpreter.AccountCreationContext,
	handler AccountHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {

	return interpreter.NewAccountValue(
		context,
		addressValue,
		newAccountBalanceGetFunction(context, handler, addressValue),
		newAccountAvailableBalanceGetFunction(context, handler, addressValue),
		func() interpreter.Value {
			return newAccountStorageValue(
				context,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountContractsValue(
				context,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountKeysValue(
				context,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountInboxValue(
				context,
				handler,
				addressValue,
			)
		},
		func() interpreter.Value {
			return newAccountCapabilitiesValue(
				context,
				addressValue,
				handler,
				handler,
			)
		},
	)
}

type AccountContractAdditionAndNamesHandler interface {
	AccountContractAdditionHandler
	AccountContractNamesProvider
}

type AccountContractsHandler interface {
	AccountContractProvider
	AccountContractAdditionAndNamesHandler
	AccountContractRemovalHandler
}

func newAccountContractsValue(
	context interpreter.AccountContractCreationContext,
	handler AccountContractsHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountContractsValue(
		context,
		addressValue,
		newAccountContractsChangeFunction(
			context,
			sema.Account_ContractsTypeAddFunctionType,
			handler,
			addressValue,
			false,
		),
		newAccountContractsChangeFunction(
			context,
			sema.Account_ContractsTypeUpdateFunctionType,
			handler,
			addressValue,
			true,
		),
		newAccountContractsTryUpdateFunction(
			context,
			sema.Account_ContractsTypeUpdateFunctionType,
			handler,
			addressValue,
		),
		newInterpreterAccountContractsGetFunction(
			context,
			handler,
			addressValue,
		),
		newInterpreterAccountContractsBorrowFunction(
			context,
			handler,
			addressValue,
		),
		newAccountContractsRemoveFunction(
			context,
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
	context interpreter.AccountKeyCreationContext,
	handler AccountKeysHandler,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountKeysValue(
		context,
		addressValue,
		newInterpreterAccountKeysAddFunction(
			context,
			handler,
			addressValue,
		),
		newInterpreterAccountKeysGetFunction(
			context,
			sema.Account_KeysTypeGetFunctionType,
			handler,
			addressValue,
		),
		newInterpreterAccountKeysRevokeFunction(
			context,
			handler,
			addressValue,
		),
		newInterpreterAccountKeysForEachFunction(
			context,
			handler,
			addressValue,
		),
		newAccountKeysCountGetter(context, handler, addressValue),
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
		return interpreter.NewUFix64Value(gauge, func() uint64 {
			balance, err := provider.GetAccountBalance(address)
			if err != nil {
				panic(err)
			}

			return balance
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
		return interpreter.NewUFix64Value(gauge, func() uint64 {
			balance, err := provider.GetAccountAvailableBalance(address)
			if err != nil {
				panic(err)
			}

			return balance
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
) func(context interpreter.MemberAccessibleContext) interpreter.UInt64Value {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func(context interpreter.MemberAccessibleContext) interpreter.UInt64Value {

		// NOTE: flush the cached values, so the host environment
		// can properly calculate the amount of storage used by the account
		err := provider.CommitStorageTemporarily(context)
		if err != nil {
			panic(err)
		}

		return interpreter.NewUInt64Value(
			context,
			func() uint64 {
				capacity, err := provider.GetStorageUsed(address)
				if err != nil {
					panic(err)
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
) func(context interpreter.MemberAccessibleContext) interpreter.UInt64Value {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func(context interpreter.MemberAccessibleContext) interpreter.UInt64Value {

		// NOTE: flush the cached values, so the host environment
		// can properly calculate the amount of storage available for the account
		err := provider.CommitStorageTemporarily(context)
		if err != nil {
			panic(err)
		}

		return interpreter.NewUInt64Value(
			context,
			func() uint64 {
				capacity, err := provider.GetStorageCapacity(address)
				if err != nil {
					panic(err)
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

func newInterpreterAccountKeysAddFunction(
	context interpreter.AccountKeyCreationContext,
	handler AccountKeyAdditionHandler,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountKeys interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountKeys,
			sema.Account_KeysTypeAddFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				inter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				publicKeyValue, ok := invocation.Arguments[0].(*interpreter.CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				hashAlgoValue := invocation.Arguments[1]

				weightValue, ok := invocation.Arguments[2].(interpreter.UFix64Value)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountKeysAdd(
					inter,
					addressValue,
					publicKeyValue,
					hashAlgoValue,
					weightValue,
					locationRange,
					handler,
				)
			},
		)
	}
}

func NewVMAccountKeysAddFunction(
	handler AccountKeyAdditionHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_KeysType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_KeysTypeAddFunctionName,
			sema.Account_KeysTypeAddFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver)

				publicKeyValue, ok := args[0].(*interpreter.CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				hashAlgoValue := args[1]

				weightValue, ok := args[2].(interpreter.UFix64Value)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountKeysAdd(
					context,
					address,
					publicKeyValue,
					hashAlgoValue,
					weightValue,
					interpreter.EmptyLocationRange,
					handler,
				)
			},
		),
	}
}

func AccountKeysAdd(
	context interpreter.AccountKeyCreationContext,
	addressValue interpreter.AddressValue,
	publicKeyValue interpreter.MemberAccessibleValue,
	hashAlgoValue interpreter.Value,
	weightValue interpreter.UFix64Value,
	locationRange interpreter.LocationRange,
	handler AccountKeyAdditionHandler,
) interpreter.Value {
	publicKey, err := NewPublicKeyFromValue(context, locationRange, publicKeyValue)
	if err != nil {
		panic(err)
	}

	hashAlgo := NewHashAlgorithmFromValue(context, locationRange, hashAlgoValue)

	weight := weightValue.ToInt(locationRange)

	address := addressValue.ToAddress()

	accountKey, err := handler.AddAccountKey(
		address,
		publicKey,
		hashAlgo,
		weight,
	)
	if err != nil {
		panic(err)
	}

	accountKeyIndex := interpreter.NewIntValueFromInt64(context, int64(accountKey.KeyIndex))

	handler.EmitEvent(
		context,
		locationRange,
		AccountKeyAddedFromPublicKeyEventType,
		[]interpreter.Value{
			addressValue,
			publicKeyValue,
			weightValue,
			hashAlgoValue,
			accountKeyIndex,
		},
	)

	return NewAccountKeyValue(
		context,
		locationRange,
		accountKey,
		handler,
	)
}

type AccountKey struct {
	PublicKey *PublicKey
	KeyIndex  uint32
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
	GetAccountKey(address common.Address, index uint32) (*AccountKey, error)
	AccountKeysCount(address common.Address) (uint32, error)
}

func newInterpreterAccountKeysGetFunction(
	context interpreter.FunctionCreationContext,
	functionType *sema.FunctionType,
	provider AccountKeyProvider,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountKeys interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {

		// Converted addresses can be cached and don't have to be recomputed on each function invocation
		address := addressValue.ToAddress()

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountKeys,
			functionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				inter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				indexValue, ok := invocation.Arguments[0].(interpreter.IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountKeysGet(
					inter,
					address,
					indexValue,
					locationRange,
					provider,
				)
			},
		)
	}
}

func NewVMAccountKeysGetFunction(
	provider AccountKeyProvider,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_KeysType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_KeysTypeGetFunctionName,
			sema.Account_KeysTypeGetFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				indexValue, ok := args[0].(interpreter.IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountKeysGet(
					context,
					address,
					indexValue,
					interpreter.EmptyLocationRange,
					provider,
				)
			},
		),
	}
}

func AccountKeysGet(
	context interpreter.AccountKeyCreationContext,
	address common.Address,
	indexValue interpreter.IntValue,
	locationRange interpreter.LocationRange,
	provider AccountKeyProvider,
) interpreter.Value {
	index := indexValue.ToUint32(locationRange)

	accountKey, err := provider.GetAccountKey(address, index)
	if err != nil {
		panic(err)
	}

	// Here it is expected the host function to return a nil key, if a key is not found at the given index.
	// This is done because, if the host function returns an error when a key is not found, then
	// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
	if accountKey == nil {
		return interpreter.Nil
	}

	return interpreter.NewSomeValueNonCopying(
		context,
		NewAccountKeyValue(
			context,
			locationRange,
			accountKey,
			provider,
		),
	)
}

// accountKeysForEachCallbackTypeParams are the parameter types of the callback function of
// `Account.Keys.forEachKey(_ f: fun(AccountKey): Bool)`
var accountKeysForEachCallbackTypeParams = []sema.Type{sema.AccountKeyType}

func newInterpreterAccountKeysForEachFunction(
	context interpreter.FunctionCreationContext,
	provider AccountKeyProvider,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountKeys interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountKeys,
			sema.Account_KeysTypeForEachFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				fnValue, ok := invocation.Arguments[0].(interpreter.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountKeysForEach(
					invocationContext,
					address,
					fnValue,
					locationRange,
					provider,
				)
			},
		)
	}
}

func NewVMAccountKeysForEachFunction(
	provider AccountKeyProvider,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_KeysType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_KeysTypeForEachFunctionName,
			sema.Account_KeysTypeForEachFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				fnValue, ok := args[0].(interpreter.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountKeysForEach(
					context,
					address,
					fnValue,
					interpreter.EmptyLocationRange,
					provider,
				)
			},
		),
	}
}

func AccountKeysForEach(
	invocationContext interpreter.InvocationContext,
	address common.Address,
	fnValue interpreter.FunctionValue,
	locationRange interpreter.LocationRange,
	provider AccountKeyProvider,
) interpreter.Value {
	fnValueType := fnValue.FunctionType(invocationContext)
	parameterTypes := fnValueType.ParameterTypes()
	returnType := fnValueType.ReturnTypeAnnotation.Type

	liftKeyToValue := func(key *AccountKey) interpreter.Value {
		return NewAccountKeyValue(
			invocationContext,
			locationRange,
			key,
			provider,
		)
	}

	count, err := provider.AccountKeysCount(address)
	if err != nil {
		panic(err)
	}

	for index := uint32(0); index < count; index++ {

		accountKey, err := provider.GetAccountKey(address, index)
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

		res, err := interpreter.InvokeFunctionValue(
			invocationContext,
			fnValue,
			[]interpreter.Value{liftedKey},
			accountKeysForEachCallbackTypeParams,
			parameterTypes,
			returnType,
			locationRange,
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
}

func newAccountKeysCountGetter(
	gauge common.MemoryGauge,
	provider AccountKeyProvider,
	addressValue interpreter.AddressValue,
) interpreter.AccountKeysCountGetter {
	address := addressValue.ToAddress()

	return func() interpreter.UInt64Value {
		return interpreter.NewUInt64Value(gauge, func() uint64 {
			var count uint32
			var err error

			count, err = provider.AccountKeysCount(address)
			if err != nil {
				// The provider might not be able to fetch the number of account keys
				// e.g. when the account does not exist
				panic(err)
			}

			return uint64(count)
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
	RevokeAccountKey(address common.Address, index uint32) (*AccountKey, error)
}

func newInterpreterAccountKeysRevokeFunction(
	context interpreter.FunctionCreationContext,
	handler AccountKeyRevocationHandler,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountKeys interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountKeys,
			sema.Account_KeysTypeRevokeFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				inter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				indexValue, ok := invocation.Arguments[0].(interpreter.IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountKeysRevoke(
					inter,
					addressValue,
					indexValue,
					locationRange,
					handler,
				)
			},
		)
	}
}

func NewVMAccountKeysRevokeFunction(
	handler AccountKeyRevocationHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_KeysType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_KeysTypeRevokeFunctionName,
			sema.Account_KeysTypeRevokeFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				addressValue := vm.GetAccountTypePrivateAddressValue(receiver)

				indexValue, ok := args[0].(interpreter.IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountKeysRevoke(
					context,
					addressValue,
					indexValue,
					interpreter.EmptyLocationRange,
					handler,
				)
			},
		),
	}
}

func AccountKeysRevoke(
	context interpreter.InvocationContext,
	addressValue interpreter.AddressValue,
	indexValue interpreter.IntValue,
	locationRange interpreter.LocationRange,
	handler AccountKeyRevocationHandler,
) interpreter.Value {
	index := indexValue.ToUint32(locationRange)

	address := addressValue.ToAddress()

	accountKey, err := handler.RevokeAccountKey(address, index)
	if err != nil {
		panic(err)
	}

	// Here it is expected the host function to return a nil key, if a key is not found at the given index.
	// This is done because, if the host function returns an error when a key is not found, then
	// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
	if accountKey == nil {
		return interpreter.Nil
	}

	handler.EmitEvent(
		context,
		locationRange,
		AccountKeyRemovedFromPublicKeyIndexEventType,
		[]interpreter.Value{
			addressValue,
			indexValue,
		},
	)

	return interpreter.NewSomeValueNonCopying(
		context,
		NewAccountKeyValue(
			context,
			locationRange,
			accountKey,
			handler,
		),
	)
}

func newInterpreterAccountInboxPublishFunction(
	context interpreter.FunctionCreationContext,
	handler EventEmitter,
	providerValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountInbox interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountInbox,
			sema.Account_InboxTypePublishFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				inter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				value, ok := invocation.Arguments[0].(interpreter.CapabilityValue)
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

				return AccountInboxPublish(
					inter,
					locationRange,
					providerValue,
					recipientValue,
					nameValue,
					value,
					handler,
				)
			},
		)
	}
}

func NewVMAccountInboxPublishFunction(
	handler EventEmitter,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_InboxType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_InboxTypePublishFunctionName,
			sema.Account_InboxTypePublishFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				providerValue := vm.GetAccountTypePrivateAddressValue(receiver)

				value, ok := args[0].(interpreter.CapabilityValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				nameValue, ok := args[1].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				recipientValue := args[2].(interpreter.AddressValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountInboxPublish(
					context,
					interpreter.EmptyLocationRange,
					providerValue,
					recipientValue,
					nameValue,
					value,
					handler,
				)
			},
		),
	}
}

func AccountInboxPublish(
	context interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
	providerValue interpreter.AddressValue,
	recipientValue interpreter.AddressValue,
	nameValue *interpreter.StringValue,
	value interpreter.CapabilityValue,
	handler EventEmitter,
) interpreter.Value {
	handler.EmitEvent(
		context,
		locationRange,
		AccountInboxPublishedEventType,
		[]interpreter.Value{
			providerValue,
			recipientValue,
			nameValue,
			interpreter.NewTypeValue(context, value.StaticType(context)),
		},
	)

	publishedValue := interpreter.NewPublishedValue(context, recipientValue, value).Transfer(
		context,
		locationRange,
		atree.Address(providerValue),
		true,
		nil,
		nil,
		true, // New PublishedValue is standalone.
	)

	storageMapKey := interpreter.StringStorageMapKey(nameValue.Str)

	context.WriteStored(
		common.Address(providerValue),
		common.StorageDomainInbox,
		storageMapKey,
		publishedValue,
	)

	return interpreter.Void
}

func newInterpreterAccountInboxUnpublishFunction(
	context interpreter.FunctionCreationContext,
	handler EventEmitter,
	providerValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountInbox interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountInbox,
			sema.Account_InboxTypeUnpublishFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				inter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair == nil {
					panic(errors.NewUnreachableError())
				}
				borrowType := typeParameterPair.Value

				return AccountInboxUnpublish(
					inter,
					locationRange,
					providerValue,
					borrowType,
					nameValue,
					handler,
				)
			},
		)
	}
}

func NewVMAccountInboxUnpublishFunction(
	handler EventEmitter,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_InboxType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_InboxTypeUnpublishFunctionName,
			sema.Account_InboxTypeUnpublishFunctionType,
			func(context *vm.Context, typeArguments []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				providerValue := vm.GetAccountTypePrivateAddressValue(receiver)

				nameValue, ok := args[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				borrowType := interpreter.MustConvertStaticToSemaType(typeArguments[0], context)

				return AccountInboxUnpublish(
					context,
					interpreter.EmptyLocationRange,
					providerValue,
					borrowType,
					nameValue,
					handler,
				)
			},
		),
	}
}

func AccountInboxUnpublish(
	context interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
	providerValue interpreter.AddressValue,
	borrowType sema.Type,
	nameValue *interpreter.StringValue,
	handler EventEmitter,
) interpreter.Value {
	storageMapKey := interpreter.StringStorageMapKey(nameValue.Str)

	provider := common.Address(providerValue)

	readValue := context.ReadStored(
		provider,
		common.StorageDomainInbox,
		storageMapKey,
	)
	if readValue == nil {
		return interpreter.Nil
	}
	publishedValue, ok := readValue.(*interpreter.PublishedValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	capabilityType := sema.NewCapabilityType(context, borrowType)
	publishedType := publishedValue.Value.StaticType(context)
	if !interpreter.IsSubTypeOfSemaType(context, publishedType, capabilityType) {
		panic(&interpreter.ForceCastTypeMismatchError{
			ExpectedType:  capabilityType,
			ActualType:    interpreter.MustConvertStaticToSemaType(publishedType, context),
			LocationRange: locationRange,
		})
	}

	value := publishedValue.Value.Transfer(
		context,
		locationRange,
		atree.Address{},
		true,
		nil,
		nil,
		false, // publishedValue is an element in storage map because it is returned by ReadStored.
	)

	context.WriteStored(
		provider,
		common.StorageDomainInbox,
		storageMapKey,
		nil,
	)

	handler.EmitEvent(
		context,
		locationRange,
		AccountInboxUnpublishedEventType,
		[]interpreter.Value{
			providerValue,
			nameValue,
		},
	)

	return interpreter.NewSomeValueNonCopying(context, value)
}

func newInterpreterAccountInboxClaimFunction(
	context interpreter.FunctionCreationContext,
	handler EventEmitter,
	recipientValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountInbox interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountInbox,
			sema.Account_InboxTypeClaimFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				inter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				providerValue, ok := invocation.Arguments[1].(interpreter.AddressValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair == nil {
					panic(errors.NewUnreachableError())
				}

				borrowType := typeParameterPair.Value

				return AccountInboxClaim(
					inter,
					locationRange,
					providerValue,
					recipientValue,
					nameValue,
					borrowType,
					handler,
				)
			},
		)
	}
}

func NewVMAccountInboxClaimFunction(
	handler EventEmitter,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_InboxType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_InboxTypeClaimFunctionName,
			sema.Account_InboxTypeClaimFunctionType,
			func(context *vm.Context, typeArguments []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				recipientValue := vm.GetAccountTypePrivateAddressValue(receiver)

				nameValue, ok := args[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				providerValue, ok := args[1].(interpreter.AddressValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				borrowType := interpreter.MustConvertStaticToSemaType(typeArguments[0], context)

				return AccountInboxClaim(
					context,
					interpreter.EmptyLocationRange,
					providerValue,
					recipientValue,
					nameValue,
					borrowType,
					handler,
				)
			},
		),
	}
}

func AccountInboxClaim(
	context interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
	providerValue interpreter.AddressValue,
	recipientValue interpreter.AddressValue,
	nameValue *interpreter.StringValue,
	borrowType sema.Type,
	handler EventEmitter,
) interpreter.Value {
	providerAddress := providerValue.ToAddress()

	storageMapKey := interpreter.StringStorageMapKey(nameValue.Str)

	readValue := context.ReadStored(
		providerAddress,
		common.StorageDomainInbox,
		storageMapKey,
	)
	if readValue == nil {
		return interpreter.Nil
	}
	publishedValue, ok := readValue.(*interpreter.PublishedValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// compare the intended recipient with the caller
	if !publishedValue.Recipient.Equal(context, locationRange, recipientValue) {
		return interpreter.Nil
	}

	ty := sema.NewCapabilityType(context, borrowType)
	publishedType := publishedValue.Value.StaticType(context)
	if !interpreter.IsSubTypeOfSemaType(context, publishedType, ty) {
		panic(&interpreter.ForceCastTypeMismatchError{
			ExpectedType:  ty,
			ActualType:    interpreter.MustConvertStaticToSemaType(publishedType, context),
			LocationRange: locationRange,
		})
	}

	value := publishedValue.Value.Transfer(
		context,
		locationRange,
		atree.Address{},
		true,
		nil,
		nil,
		false, // publishedValue is an element in storage map because it is returned by ReadStored.
	)

	context.WriteStored(
		providerAddress,
		common.StorageDomainInbox,
		storageMapKey,
		nil,
	)

	handler.EmitEvent(
		context,
		locationRange,
		AccountInboxClaimedEventType,
		[]interpreter.Value{
			providerValue,
			recipientValue,
			nameValue,
		},
	)

	return interpreter.NewSomeValueNonCopying(context, value)
}

func newAccountInboxValue(
	context interpreter.FunctionCreationContext,
	handler EventEmitter,
	addressValue interpreter.AddressValue,
) interpreter.Value {
	return interpreter.NewAccountInboxValue(
		context,
		addressValue,
		newInterpreterAccountInboxPublishFunction(context, handler, addressValue),
		newInterpreterAccountInboxUnpublishFunction(context, handler, addressValue),
		newInterpreterAccountInboxClaimFunction(context, handler, addressValue),
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
	context interpreter.MemberAccessibleContext,
	locationRange interpreter.LocationRange,
) *interpreter.ArrayValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return func(
		context interpreter.MemberAccessibleContext,
		locationRange interpreter.LocationRange,
	) *interpreter.ArrayValue {
		names, err := provider.GetAccountContractNames(address)
		if err != nil {
			panic(err)
		}

		values := make([]interpreter.Value, len(names))
		for i, name := range names {
			memoryUsage := common.NewStringMemoryUsage(len(name))
			values[i] = interpreter.NewStringValue(
				context,
				memoryUsage,
				func() string {
					return name
				},
			)
		}

		arrayType := interpreter.NewVariableSizedStaticType(
			context,
			interpreter.NewPrimitiveStaticType(
				context,
				interpreter.PrimitiveStaticTypeString,
			),
		)

		return interpreter.NewArrayValue(
			context,
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

func newInterpreterAccountContractsGetFunction(
	context interpreter.FunctionCreationContext,
	provider AccountContractProvider,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountContracts interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountContracts,
			sema.Account_ContractsTypeGetFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				context := invocation.InvocationContext

				nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountContractsGet(
					context,
					addressValue,
					nameValue,
					provider,
				)
			},
		)
	}
}

func NewVMAccountContractsGetFunction(provider AccountContractProvider) VMFunction {
	return VMFunction{
		BaseType: sema.Account_ContractsType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_ContractsTypeGetFunctionName,
			sema.Account_ContractsTypeGetFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				addressValue := vm.GetAccountTypePrivateAddressValue(receiver)

				nameValue, ok := args[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountContractsGet(
					context,
					addressValue,
					nameValue,
					provider,
				)
			},
		),
	}
}

func AccountContractsGet(
	context interpreter.InvocationContext,
	addressValue interpreter.AddressValue,
	nameValue *interpreter.StringValue,
	provider AccountContractProvider,
) interpreter.Value {
	name := nameValue.Str
	location := common.NewAddressLocation(
		context,
		common.Address(addressValue),
		name,
	)

	code, err := provider.GetAccountContractCode(location)
	if err != nil {
		panic(err)
	}

	if len(code) > 0 {
		return interpreter.NewSomeValueNonCopying(
			context,
			interpreter.NewDeployedContractValue(
				context,
				addressValue,
				nameValue,
				interpreter.ByteSliceToByteArrayValue(
					context,
					code,
				),
			),
		)
	} else {
		return interpreter.Nil
	}
}

func newInterpreterAccountContractsBorrowFunction(
	context interpreter.AccountContractBorrowContext,
	handler AccountContractsHandler,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountContracts interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {

		// Converted addresses can be cached and don't have to be recomputed on each function invocation
		address := addressValue.ToAddress()

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountContracts,
			sema.Account_ContractsTypeBorrowFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair == nil {
					panic(errors.NewUnreachableError())
				}
				borrowType := typeParameterPair.Value

				return AccountContractsBorrow(
					invocationContext,
					locationRange,
					address,
					nameValue,
					borrowType,
					handler,
				)
			},
		)
	}
}

func NewVMAccountContractsBorrowFunction(handler AccountContractsHandler) VMFunction {
	return VMFunction{
		BaseType: sema.Account_ContractsType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_ContractsTypeBorrowFunctionName,
			sema.Account_ContractsTypeBorrowFunctionType,
			func(context *vm.Context, typeArguments []bbq.StaticType, args ...vm.Value) vm.Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				nameValue, ok := args[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				borrowType := interpreter.MustConvertStaticToSemaType(typeArguments[0], context)

				return AccountContractsBorrow(
					context,
					interpreter.EmptyLocationRange,
					address,
					nameValue,
					borrowType,
					handler,
				)
			},
		),
	}
}

func AccountContractsBorrow(
	invocationContext interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
	address common.Address,
	nameValue *interpreter.StringValue,
	borrowType sema.Type,
	handler AccountContractsHandler,
) interpreter.Value {
	name := nameValue.Str
	location := common.NewAddressLocation(invocationContext, address, name)

	referenceType, ok := borrowType.(*sema.ReferenceType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	if referenceType.Authorization != sema.UnauthorizedAccess {
		panic(errors.NewDefaultUserError("cannot borrow a reference with an authorization"))
	}

	// Check if the contract exists

	code, err := handler.GetAccountContractCode(location)
	if err != nil {
		panic(err)
	}
	if len(code) == 0 {
		return interpreter.Nil
	}

	// Load the contract and get the contract composite value.
	// The requested contract may be a contract interface,
	// in which case there will be no contract composite value.

	contractLocation := common.NewAddressLocation(invocationContext, address, name)

	contractValue, err := invocationContext.GetContractValue(contractLocation)
	if err != nil {
		var notDeclaredErr interpreter.NotDeclaredError
		if goerrors.As(err, &notDeclaredErr) {
			return interpreter.Nil
		}

		panic(err)
	}

	// Check the type

	staticType := contractValue.StaticType(invocationContext)
	if !interpreter.IsSubTypeOfSemaType(invocationContext, staticType, referenceType.Type) {
		return interpreter.Nil
	}

	// No need to track the referenced value, since the reference is taken to a contract value.
	// A contract value would never be moved or destroyed, within the execution of a program.
	reference := interpreter.NewEphemeralReferenceValue(
		invocationContext,
		interpreter.UnauthorizedAccess,
		contractValue,
		referenceType.Type,
		locationRange,
	)

	return interpreter.NewSomeValueNonCopying(
		invocationContext,
		reference,
	)
}

type ContractAdditionTracker interface {
	// StartContractAddition starts adding a contract.
	StartContractAddition(location common.AddressLocation)

	// EndContractAddition ends adding the contract
	EndContractAddition(location common.AddressLocation)

	// IsContractBeingAdded checks whether a contract is being added in the current execution.
	IsContractBeingAdded(location common.AddressLocation) bool
}

type SimpleContractAdditionTracker struct {
	deployedContracts map[common.AddressLocation]struct{}
}

func NewSimpleContractAdditionTracker() *SimpleContractAdditionTracker {
	return &SimpleContractAdditionTracker{}
}

var _ ContractAdditionTracker = &SimpleContractAdditionTracker{}

func (t *SimpleContractAdditionTracker) StartContractAddition(location common.AddressLocation) {
	if t.deployedContracts == nil {
		t.deployedContracts = map[common.AddressLocation]struct{}{}
	}

	t.deployedContracts[location] = struct{}{}
}

func (t *SimpleContractAdditionTracker) EndContractAddition(location common.AddressLocation) {
	delete(t.deployedContracts, location)
}

func (t *SimpleContractAdditionTracker) IsContractBeingAdded(location common.AddressLocation) bool {
	_, contains := t.deployedContracts[location]
	return contains
}

type AccountContractAdditionHandler interface {
	EventEmitter
	AccountContractProvider
	ContractAdditionTracker

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
	LoadContractValue(
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

// newAccountContractsChangeFunction called when e.g.
// - adding: `Account.contracts.add(name: "Foo", code: [...])` (isUpdate = false)
// - updating: `Account.contracts.update(name: "Foo", code: [...])` (isUpdate = true)
func newAccountContractsChangeFunction(
	context interpreter.FunctionCreationContext,
	functionType *sema.FunctionType,
	handler AccountContractAdditionAndNamesHandler,
	addressValue interpreter.AddressValue,
	isUpdate bool,
) interpreter.BoundFunctionGenerator {
	return func(accountContracts interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountContracts,
			functionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				return changeAccountContracts(invocation, handler, addressValue, isUpdate)
			},
		)
	}
}

type OldProgramError struct {
	Err      error
	Location common.Location
}

var _ errors.UserError = &OldProgramError{}
var _ errors.ParentError = &OldProgramError{}

func (e *OldProgramError) IsUserError() {}

func (e *OldProgramError) Error() string {
	return "problem with old program"
}

func (e *OldProgramError) Unwrap() error {
	return e.Err
}

func (e *OldProgramError) ChildErrors() []error {
	return []error{e.Err}
}

func (e *OldProgramError) ImportLocation() common.Location {
	return e.Location
}

func changeAccountContracts(
	invocation interpreter.Invocation,
	handler AccountContractAdditionAndNamesHandler,
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

	newCode, err := interpreter.ByteArrayValueToByteSlice(invocation.InvocationContext, newCodeValue, locationRange)
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
	location := common.NewAddressLocation(invocation.InvocationContext, address, contractName)

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
	handleContractUpdateError := func(err error, code []byte) {
		if err == nil {
			return
		}

		// Update the code for the error pretty printing.
		// The code may be the new code, or the old code if this is an update.
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
		newCode,
		location,
		getAndSetProgram,
	)
	handleContractUpdateError(err, newCode)

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

		handler.TemporarilyRecordCode(location, newCode)

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

		handler.TemporarilyRecordCode(location, newCode)

		panic(errors.NewDefaultUserError(
			"invalid %s: the name argument must match the name of the declaration: got %q, expected %q",
			declarationKind.Name(),
			contractName,
			declaredName,
		))
	}

	// Validate the contract update

	inter := invocation.InvocationContext

	if isUpdate {
		oldCode, err := handler.GetAccountContractCode(location)
		handleContractUpdateError(err, newCode)

		memoryGauge := invocation.InvocationContext
		oldProgram, err := parser.ParseProgram(
			memoryGauge,
			oldCode,
			parser.Config{
				IgnoreLeadingIdentifierEnabled: true,
			},
		)

		if err != nil && !ignoreUpdatedProgramParserError(err) {
			// NOTE: Errors are usually in the new program / new code,
			// but here we failed for the old program / old code.
			err = &OldProgramError{
				Err:      err,
				Location: location,
			}
			handleContractUpdateError(err, oldCode)
		}

		validator := NewContractUpdateValidator(
			location,
			contractName,
			handler,
			oldProgram,
			program.Program,
		)

		err = validator.Validate()
		handleContractUpdateError(err, newCode)
	}

	err = updateAccountContractCode(
		handler,
		location,
		program,
		newCode,
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

		handler.TemporarilyRecordCode(location, newCode)

		panic(err)
	}

	var eventType *sema.CompositeType

	if isUpdate {
		eventType = AccountContractUpdatedEventType
	} else {
		eventType = AccountContractAddedEventType
	}

	codeHashValue := CodeToHashValue(inter, newCode)

	handler.EmitEvent(
		inter,
		locationRange,
		eventType,
		[]interpreter.Value{
			addressValue,
			codeHashValue,
			nameValue,
		},
	)

	return interpreter.NewDeployedContractValue(
		inter,
		addressValue,
		nameValue,
		newCodeValue,
	)
}

func newAccountContractsTryUpdateFunction(
	context interpreter.FunctionCreationContext,
	functionType *sema.FunctionType,
	handler AccountContractAdditionAndNamesHandler,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountContracts interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountContracts,
			functionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) (deploymentResult interpreter.Value) {
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
						optionalDeployedContract = interpreter.NewSomeValueNonCopying(invocation.InvocationContext, deployedContract)
					}

					deploymentResult = interpreter.NewDeploymentResultValue(context, optionalDeployedContract)
				}()

				deployedContract = changeAccountContracts(invocation, handler, addressValue, true)
				return
			},
		)
	}
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
	err = handler.UpdateAccountContractCode(location, code)
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

	return handler.LoadContractValue(
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
	context interpreter.FunctionCreationContext,
	handler AccountContractRemovalHandler,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountContracts interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {

		// Converted addresses can be cached and don't have to be recomputed on each function invocation
		address := addressValue.ToAddress()

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountContracts,
			sema.Account_ContractsTypeRemoveFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				inter := invocation.InvocationContext
				nameValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				name := nameValue.Str
				location := common.NewAddressLocation(invocation.InvocationContext, address, name)

				// Get the current code

				code, err := handler.GetAccountContractCode(location)
				if err != nil {
					panic(err)
				}

				// Only remove the contract code, remove the contract value, and emit an event,
				// if there is currently code deployed for the given contract name

				if len(code) > 0 {
					locationRange := invocation.LocationRange

					// NOTE: *DO NOT* call setProgram – the program removal
					// should not be effective during the execution, only after

					existingProgram, err := parser.ParseProgram(inter, code, parser.Config{})

					// If the existing code is not parsable (i.e: `err != nil`),
					// that shouldn't be a reason to fail the contract removal.
					// Therefore, validate only if the code is a valid one.
					if err == nil && containsEnumsInProgram(existingProgram) {
						panic(&ContractRemovalError{
							Name:          name,
							LocationRange: locationRange,
						})
					}

					err = handler.RemoveAccountContractCode(location)
					if err != nil {
						panic(err)
					}

					// NOTE: the contract recording function delays the write
					// until the end of the execution of the program

					handler.RecordContractRemoval(location)

					codeHashValue := CodeToHashValue(inter, code)

					handler.EmitEvent(
						inter,
						locationRange,
						AccountContractRemovedEventType,
						[]interpreter.Value{
							addressValue,
							codeHashValue,
							nameValue,
						},
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

const GetAccountFunctionName = "getAccount"

var GetAccountFunctionType = sema.NewSimpleFunctionType(
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

func NewInterpreterGetAccountFunction(handler AccountHandler) StandardLibraryValue {
	return NewInterpreterStandardLibraryStaticFunction(
		GetAccountFunctionName,
		GetAccountFunctionType,
		getAccountFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {

			inter := invocation.InvocationContext
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

func NewVMGetAccountFunction(handler AccountHandler) StandardLibraryValue {
	return NewVMStandardLibraryStaticFunction(
		GetAccountFunctionName,
		GetAccountFunctionType,
		getAccountFunctionDocString,
		func(context *vm.Context, _ []bbq.StaticType, arguments ...interpreter.Value) interpreter.Value {
			address, ok := arguments[0].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return NewAccountReferenceValue(
				context,
				handler,
				address,
				interpreter.UnauthorizedAccess,
				interpreter.EmptyLocationRange,
			)
		},
	)
}

func NewAccountKeyValue(
	context interpreter.AccountKeyCreationContext,
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
		context,
		interpreter.NewIntValueFromInt64(context, int64(accountKey.KeyIndex)),
		NewPublicKeyValue(
			context,
			locationRange,
			accountKey.PublicKey,
		),
		hashAlgorithm,
		interpreter.NewUFix64ValueWithInteger(
			context, func() uint64 {
				return uint64(accountKey.Weight)
			},
			locationRange,
		),
		interpreter.BoolValue(accountKey.IsRevoked),
	)
}

func NewHashAlgorithmFromValue(
	context interpreter.MemberAccessibleContext,
	locationRange interpreter.LocationRange,
	value interpreter.Value,
) sema.HashAlgorithm {
	hashAlgoValue := value.(*interpreter.SimpleCompositeValue)

	rawValue := hashAlgoValue.GetMember(context, locationRange, sema.EnumRawValueFieldName)
	if rawValue == nil {
		panic("cannot find hash algorithm raw value")
	}

	hashAlgoRawValue := rawValue.(interpreter.UInt8Value)

	return sema.HashAlgorithm(hashAlgoRawValue.ToInt(locationRange))
}

func CodeToHashValue(context interpreter.ArrayCreationContext, code []byte) *interpreter.ArrayValue {
	codeHash := sha3.Sum256(code)
	return interpreter.ByteSliceToConstantSizedByteArrayValue(context, codeHash[:])
}

func newAccountStorageCapabilitiesValue(
	context interpreter.StorageCapabilityCreationContext,
	addressValue interpreter.AddressValue,
	issueHandler CapabilityControllerIssueHandler,
	handler CapabilityControllerHandler,
) interpreter.Value {
	return interpreter.NewAccountStorageCapabilitiesValue(
		context,
		addressValue,
		newInterpreterAccountStorageCapabilitiesGetControllerFunction(context, addressValue, handler),
		newInterpreterAccountStorageCapabilitiesGetControllersFunction(context, addressValue, handler),
		newInterpreterAccountStorageCapabilitiesForEachControllerFunction(context, addressValue, handler),
		newInterpreterAccountStorageCapabilitiesIssueFunction(context, issueHandler, addressValue),
		newInterpreterAccountStorageCapabilitiesIssueWithTypeFunction(context, issueHandler, addressValue),
	)
}

func newAccountAccountCapabilitiesValue(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	issueHandler CapabilityControllerIssueHandler,
	handler CapabilityControllerHandler,
) interpreter.Value {
	accountCapabilities := interpreter.NewAccountAccountCapabilitiesValue(
		context,
		addressValue,
		newInterpreterAccountAccountCapabilitiesGetControllerFunction(context, addressValue, handler),
		newInterpreterAccountAccountCapabilitiesGetControllersFunction(context, addressValue, handler),
		newInterpreterAccountAccountCapabilitiesForEachControllerFunction(context, addressValue, handler),
		newInterpreterAccountAccountCapabilitiesIssueFunction(context, addressValue, issueHandler),
		newInterpreterAccountAccountCapabilitiesIssueWithTypeFunction(context, addressValue, issueHandler),
	)

	return accountCapabilities
}

func newAccountCapabilitiesValue(
	context interpreter.AccountCapabilityCreationContext,
	addressValue interpreter.AddressValue,
	issueHandler CapabilityControllerIssueHandler,
	handler CapabilityControllerHandler,
) interpreter.Value {
	return interpreter.NewAccountCapabilitiesValue(
		context,
		addressValue,
		newInterpreterAccountCapabilitiesGetFunction(context, addressValue, handler, false),
		newInterpreterAccountCapabilitiesGetFunction(context, addressValue, handler, true),
		newInterpreterAccountCapabilitiesExistsFunction(context, addressValue),
		newInterpreterAccountCapabilitiesPublishFunction(context, addressValue, handler),
		newInterpreterAccountCapabilitiesUnpublishFunction(context, addressValue, handler),
		func() interpreter.Value {
			return newAccountStorageCapabilitiesValue(
				context,
				addressValue,
				issueHandler,
				handler,
			)
		},
		func() interpreter.Value {
			return newAccountAccountCapabilitiesValue(
				context,
				addressValue,
				issueHandler,
				handler,
			)
		},
	)
}

func newInterpreterAccountStorageCapabilitiesGetControllerFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerHandler,
) interpreter.BoundFunctionGenerator {
	return func(storageCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()
		return interpreter.NewBoundHostFunctionValue(
			context,
			storageCapabilities,
			sema.Account_StorageCapabilitiesTypeGetControllerFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get capability ID argument

				capabilityIDValue, ok := invocation.Arguments[0].(interpreter.UInt64Value)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountStorageCapabilitiesGetController(
					invocationContext,
					handler,
					capabilityIDValue,
					address,
					locationRange,
				)
			},
		)
	}
}

func NewVMAccountStorageCapabilitiesGetControllerFunction(
	handler CapabilityControllerHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_StorageCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeGetControllerFunctionName,
			sema.Account_StorageCapabilitiesTypeGetControllerFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				accountAddress := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Get capability ID argument
				capabilityIDValue, ok := args[0].(interpreter.UInt64Value)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountStorageCapabilitiesGetController(
					context,
					handler,
					capabilityIDValue,
					accountAddress,
					interpreter.EmptyLocationRange,
				)
			},
		),
	}
}

func AccountStorageCapabilitiesGetController(
	invocationContext interpreter.InvocationContext,
	handler CapabilityControllerHandler,
	capabilityIDValue interpreter.UInt64Value,
	address common.Address,
	locationRange interpreter.LocationRange,
) interpreter.Value {
	capabilityID := uint64(capabilityIDValue)

	referenceValue := getStorageCapabilityControllerReference(
		invocationContext,
		locationRange,
		address,
		capabilityID,
		handler,
	)
	if referenceValue == nil {
		return interpreter.Nil
	}

	return interpreter.NewSomeValueNonCopying(invocationContext, referenceValue)
}

var storageCapabilityControllerReferencesArrayStaticType = &interpreter.VariableSizedStaticType{
	Type: &interpreter.ReferenceStaticType{
		ReferencedType: interpreter.PrimitiveStaticTypeStorageCapabilityController,
		Authorization:  interpreter.UnauthorizedAccess,
	},
}

func newInterpreterAccountStorageCapabilitiesGetControllersFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerHandler,
) interpreter.BoundFunctionGenerator {
	return func(storageCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()
		return interpreter.NewBoundHostFunctionValue(
			context,
			storageCapabilities,
			sema.Account_StorageCapabilitiesTypeGetControllersFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get path argument

				targetPathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountStorageCapabilitiesGetControllers(
					invocationContext,
					handler,
					targetPathValue,
					address,
					locationRange,
				)
			},
		)
	}
}

func NewVMAccountStorageCapabilitiesGetControllersFunction(
	handler CapabilityControllerHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_StorageCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeGetControllersFunctionName,
			sema.Account_StorageCapabilitiesTypeGetControllersFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				accountAddress := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Get path argument
				targetPathValue, ok := args[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountStorageCapabilitiesGetControllers(
					context,
					handler,
					targetPathValue,
					accountAddress,
					interpreter.EmptyLocationRange,
				)
			},
		),
	}
}

func AccountStorageCapabilitiesGetControllers(
	invocationContext interpreter.InvocationContext,
	handler CapabilityControllerHandler,
	targetPathValue interpreter.PathValue,
	address common.Address,
	locationRange interpreter.LocationRange,
) interpreter.Value {

	if targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewUnreachableError())
	}

	// Get capability controllers iterator

	nextCapabilityID, count :=
		getStorageCapabilityControllerIDsIterator(invocationContext, address, targetPathValue)

	var capabilityControllerIndex uint64 = 0

	return interpreter.NewArrayValueWithIterator(
		invocationContext,
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

			referenceValue := getStorageCapabilityControllerReference(
				invocationContext,
				locationRange,
				address,
				capabilityID,
				handler,
			)
			if referenceValue == nil {
				panic(errors.NewUnreachableError())
			}

			return referenceValue
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

func newInterpreterAccountStorageCapabilitiesForEachControllerFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerHandler,
) interpreter.BoundFunctionGenerator {
	return func(storageCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()

		return interpreter.NewBoundHostFunctionValue(
			context,
			storageCapabilities,
			sema.Account_StorageCapabilitiesTypeForEachControllerFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get path argument
				targetPathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get function argument
				functionValue, ok := invocation.Arguments[1].(interpreter.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountStorageCapabilitiesForeachController(
					invocationContext,
					handler,
					functionValue,
					address,
					targetPathValue,
					locationRange,
				)
			},
		)
	}
}
func NewVMAccountStorageCapabilitiesForEachControllerFunction(
	handler CapabilityControllerHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_StorageCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeForEachControllerFunctionName,
			sema.Account_StorageCapabilitiesTypeForEachControllerFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				accountAddress := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Get path argument
				targetPathValue, ok := args[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get function argument
				functionValue, ok := args[1].(vm.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountStorageCapabilitiesForeachController(
					context,
					handler,
					functionValue,
					accountAddress,
					targetPathValue,
					interpreter.EmptyLocationRange,
				)
			},
		),
	}
}

func AccountStorageCapabilitiesForeachController(
	invocationContext interpreter.InvocationContext,
	handler CapabilityControllerHandler,
	functionValue interpreter.FunctionValue,
	address common.Address,
	targetPathValue interpreter.PathValue,
	locationRange interpreter.LocationRange,
) interpreter.Value {
	if targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewUnreachableError())
	}

	functionValueType := functionValue.FunctionType(invocationContext)
	parameterTypes := functionValueType.ParameterTypes()
	returnType := functionValueType.ReturnTypeAnnotation.Type

	// Prevent mutations (record/unrecord) to storage capability controllers
	// for this address/path during iteration

	addressPath := interpreter.AddressPath{
		Address: address,
		Path:    targetPathValue,
	}

	iterations := invocationContext.GetCapabilityControllerIterations()
	iterations[addressPath]++
	defer func() {
		iterations[addressPath]--
		if iterations[addressPath] <= 0 {
			delete(iterations, addressPath)
		}
	}()

	// Get capability controllers iterator

	nextCapabilityID, _ :=
		getStorageCapabilityControllerIDsIterator(invocationContext, address, targetPathValue)

	for {
		capabilityID, ok := nextCapabilityID()
		if !ok {
			break
		}

		referenceValue := getStorageCapabilityControllerReference(
			invocationContext,
			locationRange,
			address,
			capabilityID,
			handler,
		)
		if referenceValue == nil {
			panic(errors.NewUnreachableError())
		}

		res, err := interpreter.InvokeFunctionValue(
			invocationContext,
			functionValue,
			[]interpreter.Value{referenceValue},
			accountStorageCapabilitiesForEachControllerCallbackTypeParams,
			parameterTypes,
			returnType,
			locationRange,
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

		// It is not safe to check this at the beginning of the loop
		// (i.e. on the next invocation of the callback),
		// because if the mutation performed in the callback reorganized storage
		// such that the iteration pointer is now at the end,
		// we will not invoke the callback again but will still silently skip elements of storage.
		//
		// In order to be safe, we perform this check here to effectively enforce
		// that users return `false` from their callback in all cases where storage is mutated.
		if invocationContext.MutationDuringCapabilityControllerIteration() {
			panic(CapabilityControllersMutatedDuringIterationError{
				LocationRange: locationRange,
			})
		}
	}

	return interpreter.Void
}

func newInterpreterAccountStorageCapabilitiesIssueFunction(
	context interpreter.FunctionCreationContext,
	handler CapabilityControllerIssueHandler,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(storageCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()
		return interpreter.NewBoundHostFunctionValue(
			context,
			storageCapabilities,
			sema.Account_StorageCapabilitiesTypeIssueWithTypeFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange
				arguments := invocation.Arguments

				// Get borrow-type type-argument
				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				typeParameter := typeParameterPair.Value

				return AccountStorageCapabilitiesIssue(
					arguments,
					invocationContext,
					locationRange,
					handler,
					address,
					typeParameter,
				)
			},
		)
	}
}

func NewVMAccountStorageCapabilitiesIssueFunction(
	handler CapabilityControllerIssueHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_StorageCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeIssueFunctionName,
			sema.Account_StorageCapabilitiesTypeIssueFunctionType,
			func(context *vm.Context, typeArguments []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				accountAddress := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Get borrow type type-argument
				typeParameter := typeArguments[0]
				semaType := interpreter.MustConvertStaticToSemaType(typeParameter, context)

				return AccountStorageCapabilitiesIssue(
					args,
					context,
					interpreter.EmptyLocationRange,
					handler,
					accountAddress,
					semaType,
				)
			},
		),
	}
}

func AccountStorageCapabilitiesIssue(
	arguments []interpreter.Value,
	invocationContext interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
	handler CapabilityControllerIssueHandler,
	address common.Address,
	typeParameter sema.Type,
) interpreter.Value {

	// Get path argument

	targetPathValue, ok := arguments[0].(interpreter.PathValue)
	if !ok || targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewUnreachableError())
	}

	// Issue capability controller and return capability

	return checkAndIssueStorageCapabilityControllerWithType(
		invocationContext,
		locationRange,
		handler,
		address,
		targetPathValue,
		typeParameter,
	)
}

func newInterpreterAccountStorageCapabilitiesIssueWithTypeFunction(
	context interpreter.FunctionCreationContext,
	handler CapabilityControllerIssueHandler,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(storageCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()
		return interpreter.NewBoundHostFunctionValue(
			context,
			storageCapabilities,
			sema.Account_StorageCapabilitiesTypeIssueWithTypeFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get path argument

				targetPathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get type argument

				typeValue, ok := invocation.Arguments[1].(interpreter.TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountStorageCapabilitiesIssueWithType(
					invocationContext,
					handler,
					typeValue,
					address,
					targetPathValue,
					locationRange,
				)
			},
		)
	}
}

func NewVMAccountStorageCapabilitiesIssueWithTypeFunction(
	handler CapabilityControllerIssueHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_StorageCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeIssueWithTypeFunctionName,
			sema.Account_StorageCapabilitiesTypeIssueWithTypeFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				accountAddress := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Get path argument
				targetPathValue, ok := args[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get type argument
				typeValue, ok := args[1].(interpreter.TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountStorageCapabilitiesIssueWithType(
					context,
					handler,
					typeValue,
					accountAddress,
					targetPathValue,
					interpreter.EmptyLocationRange,
				)
			},
		),
	}
}

func AccountStorageCapabilitiesIssueWithType(
	invocationContext interpreter.InvocationContext,
	handler CapabilityControllerIssueHandler,
	typeValue interpreter.TypeValue,
	address common.Address,
	targetPathValue interpreter.PathValue,
	locationRange interpreter.LocationRange,
) interpreter.Value {
	if targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewUnreachableError())
	}

	ty, err := interpreter.ConvertStaticToSemaType(invocationContext, typeValue.Type)
	if err != nil {
		panic(errors.NewUnexpectedErrorFromCause(err))
	}

	// Issue capability controller and return capability

	return checkAndIssueStorageCapabilityControllerWithType(
		invocationContext,
		locationRange,
		handler,
		address,
		targetPathValue,
		ty,
	)
}

func checkAndIssueStorageCapabilityControllerWithType(
	context interpreter.CapabilityControllerContext,
	locationRange interpreter.LocationRange,
	handler CapabilityControllerIssueHandler,
	address common.Address,
	targetPathValue interpreter.PathValue,
	ty sema.Type,
) *interpreter.IDCapabilityValue {

	borrowType, ok := ty.(*sema.ReferenceType)
	if !ok {
		panic(&interpreter.InvalidCapabilityIssueTypeError{
			ExpectedTypeDescription: "reference type",
			ActualType:              ty,
			LocationRange:           locationRange,
		})
	}

	// Issue capability controller

	borrowStaticType := interpreter.ConvertSemaReferenceTypeToStaticReferenceType(context, borrowType)

	capabilityIDValue := IssueStorageCapabilityController(
		context,
		locationRange,
		handler,
		address,
		borrowStaticType,
		targetPathValue,
	)

	if capabilityIDValue == interpreter.InvalidCapabilityID {
		panic(interpreter.InvalidCapabilityIDError{})
	}

	// Return controller's capability

	return interpreter.NewCapabilityValue(
		context,
		capabilityIDValue,
		interpreter.NewAddressValue(context, address),
		borrowStaticType,
	)
}

func IssueStorageCapabilityController(
	context interpreter.CapabilityControllerContext,
	locationRange interpreter.LocationRange,
	handler CapabilityControllerIssueHandler,
	address common.Address,
	borrowStaticType *interpreter.ReferenceStaticType,
	targetPathValue interpreter.PathValue,
) interpreter.UInt64Value {
	if targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewDefaultUserError(
			"invalid storage capability target path domain: %s",
			targetPathValue.Domain.Identifier(),
		))
	}

	// Create and write StorageCapabilityController

	capabilityID, err := handler.GenerateAccountID(address)
	if err != nil {
		panic(err)
	}
	if capabilityID == 0 {
		panic(errors.NewUnexpectedError("invalid zero account ID"))
	}

	capabilityIDValue := interpreter.UInt64Value(capabilityID)

	controller := interpreter.NewStorageCapabilityControllerValue(
		context,
		borrowStaticType,
		capabilityIDValue,
		targetPathValue,
	)

	storeCapabilityController(context, address, capabilityIDValue, controller)
	recordStorageCapabilityController(context, locationRange, address, targetPathValue, capabilityIDValue)

	addressValue := interpreter.AddressValue(address)
	typeValue := interpreter.NewTypeValue(context, borrowStaticType)

	handler.EmitEvent(
		context,
		locationRange,
		StorageCapabilityControllerIssuedEventType,
		[]interpreter.Value{
			capabilityIDValue,
			addressValue,
			typeValue,
			targetPathValue,
		},
	)

	return capabilityIDValue
}

func newInterpreterAccountAccountCapabilitiesIssueFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerIssueHandler,
) interpreter.BoundFunctionGenerator {
	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			sema.Account_AccountCapabilitiesTypeIssueFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get borrow type type argument

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				ty := typeParameterPair.Value

				// Issue capability controller and return capability

				return checkAndIssueAccountCapabilityControllerWithType(
					invocationContext,
					locationRange,
					handler,
					address,
					ty,
				)
			},
		)
	}
}

func NewVMAccountAccountCapabilitiesIssueFunction(
	handler CapabilityControllerIssueHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_AccountCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_AccountCapabilitiesTypeIssueFunctionName,
			sema.Account_AccountCapabilitiesTypeIssueFunctionType,
			func(context *vm.Context, typeArguments []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:] // nolint:staticcheck

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Issue capability controller and return capability

				ty := interpreter.MustConvertStaticToSemaType(typeArguments[0], context)

				return checkAndIssueAccountCapabilityControllerWithType(
					context,
					interpreter.EmptyLocationRange,
					handler,
					address,
					ty,
				)
			},
		),
	}
}

func newInterpreterAccountAccountCapabilitiesIssueWithTypeFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerIssueHandler,
) interpreter.BoundFunctionGenerator {
	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			sema.Account_AccountCapabilitiesTypeIssueFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get type argument

				typeValue, ok := invocation.Arguments[0].(interpreter.TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				ty, err := interpreter.ConvertStaticToSemaType(invocationContext, typeValue.Type)
				if err != nil {
					panic(errors.NewUnexpectedErrorFromCause(err))
				}

				// Issue capability controller and return capability

				return checkAndIssueAccountCapabilityControllerWithType(
					invocationContext,
					locationRange,
					handler,
					address,
					ty,
				)
			},
		)
	}
}

func NewVMAccountAccountCapabilitiesIssueWithTypeFunction(
	handler CapabilityControllerIssueHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_AccountCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_AccountCapabilitiesTypeIssueWithTypeFunctionName,
			sema.Account_AccountCapabilitiesTypeIssueWithTypeFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...vm.Value) vm.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Get type argument

				typeValue, ok := args[0].(interpreter.TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				ty, err := interpreter.ConvertStaticToSemaType(context, typeValue.Type)
				if err != nil {
					panic(errors.NewUnexpectedErrorFromCause(err))
				}

				// Issue capability controller and return capability

				return checkAndIssueAccountCapabilityControllerWithType(
					context,
					interpreter.EmptyLocationRange,
					handler,
					address,
					ty,
				)
			},
		),
	}
}

func checkAndIssueAccountCapabilityControllerWithType(
	context interpreter.CapabilityControllerContext,
	locationRange interpreter.LocationRange,
	handler CapabilityControllerIssueHandler,
	address common.Address,
	ty sema.Type,
) *interpreter.IDCapabilityValue {

	// Get and check borrow type

	typeBound := sema.AccountReferenceType
	if !sema.IsSubType(ty, typeBound) {
		panic(&interpreter.InvalidCapabilityIssueTypeError{
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

	borrowStaticType := interpreter.ConvertSemaReferenceTypeToStaticReferenceType(context, borrowType)

	capabilityIDValue :=
		IssueAccountCapabilityController(
			context,
			locationRange,
			handler,
			address,
			borrowStaticType,
		)

	if capabilityIDValue == interpreter.InvalidCapabilityID {
		panic(interpreter.InvalidCapabilityIDError{})
	}

	// Return controller's capability

	return interpreter.NewCapabilityValue(
		context,
		capabilityIDValue,
		interpreter.NewAddressValue(context, address),
		borrowStaticType,
	)
}

func IssueAccountCapabilityController(
	context interpreter.CapabilityControllerContext,
	locationRange interpreter.LocationRange,
	handler CapabilityControllerIssueHandler,
	address common.Address,
	borrowStaticType *interpreter.ReferenceStaticType,
) interpreter.UInt64Value {
	// Create and write AccountCapabilityController

	capabilityID, err := handler.GenerateAccountID(address)
	if err != nil {
		panic(err)
	}
	if capabilityID == 0 {
		panic(errors.NewUnexpectedError("invalid zero account ID"))
	}

	capabilityIDValue := interpreter.UInt64Value(capabilityID)

	controller := interpreter.NewAccountCapabilityControllerValue(
		context,
		borrowStaticType,
		capabilityIDValue,
	)

	storeCapabilityController(context, address, capabilityIDValue, controller)
	recordAccountCapabilityController(
		context,
		locationRange,
		address,
		capabilityIDValue,
	)

	addressValue := interpreter.AddressValue(address)
	typeValue := interpreter.NewTypeValue(context, borrowStaticType)

	handler.EmitEvent(
		context,
		locationRange,
		AccountCapabilityControllerIssuedEventType,
		[]interpreter.Value{
			capabilityIDValue,
			addressValue,
			typeValue,
		},
	)

	return capabilityIDValue
}

// storeCapabilityController stores a capability controller in the account's capability ID to controller storage map
func storeCapabilityController(
	storageWriter interpreter.StorageWriter,
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
	controller interpreter.CapabilityControllerValue,
) {
	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue)

	existed := storageWriter.WriteStored(
		address,
		common.StorageDomainCapabilityController,
		storageMapKey,
		controller,
	)
	if existed {
		panic(errors.NewUnreachableError())
	}
}

// removeCapabilityController removes a capability controller from the account's capability ID to controller storage map
func removeCapabilityController(
	context interpreter.StorageContext,
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) {
	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue)

	existed := context.WriteStored(
		address,
		common.StorageDomainCapabilityController,
		storageMapKey,
		nil,
	)
	if !existed {
		panic(errors.NewUnreachableError())
	}

	SetCapabilityControllerTag(
		context,
		address,
		uint64(capabilityIDValue),
		nil,
	)
}

// getCapabilityController gets the capability controller for the given capability ID
func getCapabilityController(
	storageReader interpreter.StorageReader,
	address common.Address,
	capabilityID uint64,
	handler CapabilityControllerHandler,
) interpreter.CapabilityControllerValue {

	storageMapKey := interpreter.Uint64StorageMapKey(capabilityID)

	readValue := storageReader.ReadStored(
		address,
		common.StorageDomainCapabilityController,
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
			newStorageCapabilityControllerDeleteFunction(
				address,
				controller,
				handler,
			)

		controller.SetTarget =
			newStorageCapabilityControllerSetTargetFunction(
				address,
				controller,
				handler,
			)

	case *interpreter.AccountCapabilityControllerValue:
		capabilityID := controller.CapabilityID

		controller.GetCapability =
			newCapabilityControllerGetCapabilityFunction(address, controller)

		controller.GetTag =
			newCapabilityControllerGetTagFunction(address, capabilityID)
		controller.SetTag =
			newCapabilityControllerSetTagFunction(address, capabilityID)

		controller.Delete =
			newAccountCapabilityControllerDeleteFunction(
				address,
				controller,
				handler,
			)
	}

	return controller
}

func getStorageCapabilityControllerReference(
	context interpreter.CapabilityControllerReferenceContext,
	locationRange interpreter.LocationRange,
	address common.Address,
	capabilityID uint64,
	handler CapabilityControllerHandler,
) *interpreter.EphemeralReferenceValue {

	capabilityController := getCapabilityController(
		context,
		address,
		capabilityID,
		handler,
	)
	if capabilityController == nil {
		return nil
	}

	storageCapabilityController, ok := capabilityController.(*interpreter.StorageCapabilityControllerValue)
	if !ok {
		return nil
	}

	return interpreter.NewEphemeralReferenceValue(
		context,
		interpreter.UnauthorizedAccess,
		storageCapabilityController,
		sema.StorageCapabilityControllerType,
		locationRange,
	)
}

func newStorageCapabilityControllerSetTargetFunction(
	address common.Address,
	controller *interpreter.StorageCapabilityControllerValue,
	handler CapabilityControllerHandler,
) func(interpreter.CapabilityControllerContext, interpreter.LocationRange, interpreter.PathValue) {
	return func(
		context interpreter.CapabilityControllerContext,
		locationRange interpreter.LocationRange,
		newTargetPathValue interpreter.PathValue,
	) {
		oldTargetPathValue := controller.TargetPath
		capabilityID := controller.CapabilityID

		unrecordStorageCapabilityController(
			context,
			locationRange,
			address,
			oldTargetPathValue,
			capabilityID,
		)
		recordStorageCapabilityController(
			context,
			locationRange,
			address,
			newTargetPathValue,
			capabilityID,
		)

		addressValue := interpreter.AddressValue(address)

		handler.EmitEvent(
			context,
			locationRange,
			StorageCapabilityControllerTargetChangedEventType,
			[]interpreter.Value{
				capabilityID,
				addressValue,
				newTargetPathValue,
			},
		)
	}
}

func newStorageCapabilityControllerDeleteFunction(
	address common.Address,
	controller *interpreter.StorageCapabilityControllerValue,
	handler CapabilityControllerHandler,
) func(interpreter.CapabilityControllerContext, interpreter.LocationRange) {
	return func(
		context interpreter.CapabilityControllerContext,
		locationRange interpreter.LocationRange,
	) {
		targetPathValue := controller.TargetPath
		capabilityID := controller.CapabilityID

		unrecordStorageCapabilityController(
			context,
			locationRange,
			address,
			targetPathValue,
			capabilityID,
		)
		removeCapabilityController(
			context,
			address,
			capabilityID,
		)

		addressValue := interpreter.AddressValue(address)

		handler.EmitEvent(
			context,
			locationRange,
			StorageCapabilityControllerDeletedEventType,
			[]interpreter.Value{
				capabilityID,
				addressValue,
			},
		)
	}
}

var capabilityIDSetStaticType = &interpreter.DictionaryStaticType{
	KeyType:   interpreter.PrimitiveStaticTypeUInt64,
	ValueType: interpreter.NilStaticType,
}

func recordStorageCapabilityController(
	context interpreter.CapabilityControllerContext,
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

	iterations := context.GetCapabilityControllerIterations()
	if iterations[addressPath] > 0 {
		context.SetMutationDuringCapabilityControllerIteration()
	}

	identifier := targetPathValue.Identifier

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	storageMap := context.Storage().GetDomainStorageMap(
		context,
		address,
		common.StorageDomainPathCapability,
		true,
	)

	setKey := capabilityIDValue
	setValue := interpreter.Nil

	readValue := storageMap.ReadValue(context, storageMapKey)
	if readValue == nil {
		capabilityIDSet := interpreter.NewDictionaryValueWithAddress(
			context,
			locationRange,
			capabilityIDSetStaticType,
			address,
			setKey,
			setValue,
		)
		storageMap.SetValue(context, storageMapKey, capabilityIDSet)
	} else {
		capabilityIDSet := readValue.(*interpreter.DictionaryValue)
		existing := capabilityIDSet.Insert(context, locationRange, setKey, setValue)
		if existing != interpreter.Nil {
			panic(errors.NewUnreachableError())
		}
	}
}

func getPathCapabilityIDSet(
	context interpreter.StorageContext,
	targetPathValue interpreter.PathValue,
	address common.Address,
) *interpreter.DictionaryValue {
	if targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewUnreachableError())
	}

	identifier := targetPathValue.Identifier

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	storageMap := context.Storage().GetDomainStorageMap(
		context,
		address,
		common.StorageDomainPathCapability,
		false,
	)
	if storageMap == nil {
		return nil
	}

	readValue := storageMap.ReadValue(context, storageMapKey)
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
	context interpreter.CapabilityControllerContext,
	locationRange interpreter.LocationRange,
	address common.Address,
	targetPathValue interpreter.PathValue,
	capabilityIDValue interpreter.UInt64Value,
) {
	addressPath := interpreter.AddressPath{
		Address: address,
		Path:    targetPathValue,
	}

	iterations := context.GetCapabilityControllerIterations()
	if iterations[addressPath] > 0 {
		context.SetMutationDuringCapabilityControllerIteration()
	}

	capabilityIDSet := getPathCapabilityIDSet(context, targetPathValue, address)
	if capabilityIDSet == nil {
		panic(errors.NewUnreachableError())
	}

	existing := capabilityIDSet.Remove(context, locationRange, capabilityIDValue)
	if existing == interpreter.Nil {
		panic(errors.NewUnreachableError())
	}

	// Remove capability set if empty

	if capabilityIDSet.Count() == 0 {
		storageMap := context.Storage().GetDomainStorageMap(
			context,
			address,
			common.StorageDomainPathCapability,
			true,
		)
		if storageMap == nil {
			panic(errors.NewUnreachableError())
		}

		identifier := targetPathValue.Identifier

		storageMapKey := interpreter.StringStorageMapKey(identifier)

		if !storageMap.RemoveValue(context, storageMapKey) {
			panic(errors.NewUnreachableError())
		}
	}
}

func getStorageCapabilityControllerIDsIterator(
	context interpreter.StorageContext,
	address common.Address,
	targetPathValue interpreter.PathValue,
) (
	nextCapabilityID func() (uint64, bool),
	count uint64,
) {
	capabilityIDSet := getPathCapabilityIDSet(context, targetPathValue, address)
	if capabilityIDSet == nil {
		return func() (uint64, bool) {
			return 0, false
		}, 0
	}

	iterator := capabilityIDSet.Iterator()

	count = uint64(capabilityIDSet.Count())
	nextCapabilityID = func() (uint64, bool) {
		keyValue := iterator.NextKey(context)
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

func recordAccountCapabilityController(
	context interpreter.CapabilityControllerContext,
	_ interpreter.LocationRange,
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) {
	addressPath := interpreter.AddressPath{
		Address: address,
	}

	iterations := context.GetCapabilityControllerIterations()
	if iterations[addressPath] > 0 {
		context.SetMutationDuringCapabilityControllerIteration()
	}

	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue)

	storageMap := context.Storage().GetDomainStorageMap(
		context,
		address,
		common.StorageDomainAccountCapability,
		true,
	)

	existed := storageMap.SetValue(context, storageMapKey, interpreter.NilValue{})
	if existed {
		panic(errors.NewUnreachableError())
	}
}

func unrecordAccountCapabilityController(
	context interpreter.CapabilityControllerContext,
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) {
	addressPath := interpreter.AddressPath{
		Address: address,
	}

	interpreter.MaybeSetMutationDuringCapConIteration(context, addressPath)

	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue)

	storageMap := context.Storage().GetDomainStorageMap(
		context,
		address,
		common.StorageDomainAccountCapability,
		true,
	)

	existed := storageMap.RemoveValue(context, storageMapKey)
	if !existed {
		panic(errors.NewUnreachableError())
	}
}

func getAccountCapabilityControllerIDsIterator(
	context interpreter.StorageContext,
	address common.Address,
) (
	nextCapabilityID func() (uint64, bool),
	count uint64,
) {
	storageMap := context.Storage().GetDomainStorageMap(
		context,
		address,
		common.StorageDomainAccountCapability,
		false,
	)
	if storageMap == nil {
		return func() (uint64, bool) {
			return 0, false
		}, 0
	}

	iterator := storageMap.Iterator(context)

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

func newInterpreterAccountCapabilitiesPublishFunction(
	context interpreter.FunctionCreationContext,
	accountAddressValue interpreter.AddressValue,
	handler CapabilityControllerHandler,
) interpreter.BoundFunctionGenerator {

	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			sema.Account_CapabilitiesTypePublishFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange
				arguments := invocation.Arguments

				// Get capability argument
				capabilityValue, ok := arguments[0].(interpreter.CapabilityValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get path argument
				pathValue, ok := invocation.Arguments[1].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountCapabilitiesPublish(
					invocationContext,
					handler,
					capabilityValue,
					pathValue,
					accountAddressValue,
					locationRange,
				)
			},
		)
	}
}

func NewVMAccountCapabilitiesPublishFunction(
	handler CapabilityControllerHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_CapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_CapabilitiesTypePublishFunctionName,
			sema.Account_CapabilitiesTypePublishFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				accountAddress := vm.GetAccountTypePrivateAddressValue(receiver)

				// Get capability argument
				capabilityValue, ok := args[0].(interpreter.CapabilityValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get path argument
				pathValue, ok := args[1].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountCapabilitiesPublish(
					context,
					handler,
					capabilityValue,
					pathValue,
					accountAddress,
					interpreter.EmptyLocationRange,
				)
			},
		),
	}
}

func AccountCapabilitiesPublish(
	invocationContext interpreter.InvocationContext,
	handler CapabilityControllerHandler,
	capabilityValue interpreter.CapabilityValue,
	pathValue interpreter.PathValue,
	accountAddressValue interpreter.AddressValue,
	locationRange interpreter.LocationRange,
) interpreter.Value {

	if pathValue.Domain != common.PathDomainPublic {
		panic(errors.NewUnreachableError())
	}

	accountAddress := accountAddressValue.ToAddress()

	capabilityAddressValue := capabilityValue.Address()
	if capabilityAddressValue != accountAddressValue {
		panic(&interpreter.CapabilityAddressPublishingError{
			LocationRange:     locationRange,
			CapabilityAddress: capabilityAddressValue,
			AccountAddress:    accountAddressValue,
		})
	}

	domain := pathValue.Domain.StorageDomain()
	identifier := pathValue.Identifier

	capabilityType, ok := capabilityValue.StaticType(invocationContext).(*interpreter.CapabilityStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	borrowType := capabilityType.BorrowType

	// It is possible to have legacy capabilities without borrow type.
	// So perform the validation only if the borrow type is present.
	if borrowType != nil {
		capabilityBorrowType, ok := borrowType.(*interpreter.ReferenceStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		publishHandler := invocationContext.GetValidateAccountCapabilitiesPublishHandler()
		if publishHandler != nil {
			valid, err := publishHandler(
				invocationContext,
				locationRange,
				capabilityAddressValue,
				pathValue,
				capabilityBorrowType,
			)
			if err != nil {
				panic(err)
			}
			if !valid {
				panic(&interpreter.EntitledCapabilityPublishingError{
					LocationRange: locationRange,
					BorrowType:    capabilityBorrowType,
					Path:          pathValue,
				})
			}
		}
	}

	// Prevent an overwrite

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	if interpreter.StoredValueExists(
		invocationContext,
		accountAddress,
		domain,
		storageMapKey,
	) {
		panic(&interpreter.OverwriteError{
			Address:       accountAddressValue,
			Path:          pathValue,
			LocationRange: locationRange,
		})
	}

	capabilityValue, ok = capabilityValue.Transfer(
		invocationContext,
		locationRange,
		atree.Address(accountAddress),
		true,
		nil,
		nil,
		true, // capabilityValue is standalone because it is from invocation.Arguments[0].
	).(interpreter.CapabilityValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Write new value

	invocationContext.WriteStored(
		accountAddress,
		domain,
		storageMapKey,
		capabilityValue,
	)

	handler.EmitEvent(
		invocationContext,
		locationRange,
		CapabilityPublishedEventType,
		[]interpreter.Value{
			accountAddressValue,
			pathValue,
			capabilityValue,
		},
	)

	return interpreter.Void
}

func newInterpreterAccountCapabilitiesUnpublishFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerHandler,
) interpreter.BoundFunctionGenerator {

	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			sema.Account_CapabilitiesTypeUnpublishFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get path argument
				pathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountCapabilitiesUnpublish(
					invocationContext,
					handler,
					pathValue,
					addressValue,
					locationRange,
				)
			},
		)
	}
}

func NewVMAccountCapabilitiesUnpublishFunction(
	handler CapabilityControllerHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_CapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_CapabilitiesTypeUnpublishFunctionName,
			sema.Account_CapabilitiesTypeUnpublishFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				accountAddress := vm.GetAccountTypePrivateAddressValue(receiver)

				// Get path argument.
				pathValue, ok := args[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountCapabilitiesUnpublish(
					context,
					handler,
					pathValue,
					accountAddress,
					interpreter.EmptyLocationRange,
				)
			},
		),
	}
}

func AccountCapabilitiesUnpublish(
	invocationContext interpreter.InvocationContext,
	handler CapabilityControllerHandler,
	pathValue interpreter.PathValue,
	addressValue interpreter.AddressValue,
	locationRange interpreter.LocationRange,
) interpreter.Value {

	if pathValue.Domain != common.PathDomainPublic {
		panic(errors.NewUnreachableError())
	}

	domain := pathValue.Domain.StorageDomain()
	identifier := pathValue.Identifier

	// Read/remove capability

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	address := addressValue.ToAddress()

	readValue := invocationContext.ReadStored(address, domain, storageMapKey)
	if readValue == nil {
		return interpreter.Nil
	}

	var capabilityValue interpreter.CapabilityValue
	switch readValue := readValue.(type) {
	case interpreter.CapabilityValue:
		capabilityValue = readValue

	case interpreter.PathLinkValue: //nolint:staticcheck
		// If the stored value is a path link,
		// it failed to be migrated during the Cadence 1.0 migration.
		// Use an invalid capability value instead

		capabilityValue = interpreter.NewInvalidCapabilityValue(
			invocationContext,
			addressValue,
			readValue.Type,
		)

	default:
		panic(errors.NewUnreachableError())
	}

	capabilityValue, ok := capabilityValue.Transfer(
		invocationContext,
		locationRange,
		atree.Address{},
		true,
		nil,
		nil,
		false, // capabilityValue is an element of storage map.
	).(interpreter.CapabilityValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	invocationContext.WriteStored(
		address,
		domain,
		storageMapKey,
		nil,
	)

	handler.EmitEvent(
		invocationContext,
		locationRange,
		CapabilityUnpublishedEventType,
		[]interpreter.Value{
			addressValue,
			pathValue,
		},
	)

	return interpreter.NewSomeValueNonCopying(invocationContext, capabilityValue)
}

func canBorrow(
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) bool {

	// Ensure the wanted borrow type is not more permissive than the capability borrow type

	if !wantedBorrowType.Authorization.
		PermitsAccess(capabilityBorrowType.Authorization) {

		return false
	}

	// Ensure the wanted borrow type is a subtype or supertype of the capability borrow type

	return sema.IsSubType(wantedBorrowType.Type, capabilityBorrowType.Type) ||
		sema.IsSubType(capabilityBorrowType.Type, wantedBorrowType.Type)
}

func getCheckedCapabilityController(
	context interpreter.GetCapabilityControllerContext,
	capabilityAddressValue interpreter.AddressValue,
	capabilityIDValue interpreter.UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
	handler CapabilityControllerHandler,
) (
	interpreter.CapabilityControllerValue,
	*sema.ReferenceType,
) {
	if wantedBorrowType == nil {
		wantedBorrowType = capabilityBorrowType
	} else if !canBorrow(wantedBorrowType, capabilityBorrowType) {
		return nil, nil
	}

	capabilityAddress := capabilityAddressValue.ToAddress()
	capabilityID := uint64(capabilityIDValue)

	controller := getCapabilityController(
		context,
		capabilityAddress,
		capabilityID,
		handler,
	)
	if controller == nil {
		return nil, nil
	}

	controllerBorrowStaticType := controller.CapabilityControllerBorrowType()

	controllerBorrowType, ok :=
		interpreter.MustConvertStaticToSemaType(controllerBorrowStaticType, context).(*sema.ReferenceType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	if !canBorrow(wantedBorrowType, controllerBorrowType) {
		return nil, nil
	}

	return controller, wantedBorrowType
}

func GetCheckedCapabilityControllerReference(
	context interpreter.GetCapabilityControllerReferenceContext,
	locationRange interpreter.LocationRange,
	capabilityAddressValue interpreter.AddressValue,
	capabilityIDValue interpreter.UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
	handler CapabilityControllerHandler,
) interpreter.ReferenceValue {
	controller, resultBorrowType := getCheckedCapabilityController(
		context,
		capabilityAddressValue,
		capabilityIDValue,
		wantedBorrowType,
		capabilityBorrowType,
		handler,
	)
	if controller == nil {
		return nil
	}

	capabilityAddress := capabilityAddressValue.ToAddress()

	return controller.ReferenceValue(
		context,
		capabilityAddress,
		resultBorrowType,
		locationRange,
	)
}

func BorrowCapabilityController(
	context interpreter.BorrowCapabilityControllerContext,
	locationRange interpreter.LocationRange,
	capabilityAddress interpreter.AddressValue,
	capabilityID interpreter.UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
	handler CapabilityControllerHandler,
) interpreter.ReferenceValue {
	referenceValue := GetCheckedCapabilityControllerReference(
		context,
		locationRange,
		capabilityAddress,
		capabilityID,
		wantedBorrowType,
		capabilityBorrowType,
		handler,
	)
	if referenceValue == nil {
		return nil
	}

	// Attempt to dereference,
	// which reads the stored value
	// and performs a dynamic type check

	referencedValue := referenceValue.ReferencedValue(
		context,
		locationRange,
		false,
	)
	if referencedValue == nil {
		return nil
	}

	return referenceValue
}

func CheckCapabilityController(
	context interpreter.CheckCapabilityControllerContext,
	locationRange interpreter.LocationRange,
	capabilityAddress interpreter.AddressValue,
	capabilityID interpreter.UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
	handler CapabilityControllerHandler,
) interpreter.BoolValue {

	referenceValue := GetCheckedCapabilityControllerReference(
		context,
		locationRange,
		capabilityAddress,
		capabilityID,
		wantedBorrowType,
		capabilityBorrowType,
		handler,
	)
	if referenceValue == nil {
		return interpreter.FalseValue
	}

	// Attempt to dereference,
	// which reads the stored value
	// and performs a dynamic type check

	referencedValue := referenceValue.ReferencedValue(
		context,
		locationRange,
		false,
	)

	return referencedValue != nil
}

func newInterpreterAccountCapabilitiesGetFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	controllerHandler CapabilityControllerHandler,
	borrow bool,
) interpreter.BoundFunctionGenerator {
	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		var funcType *sema.FunctionType
		if borrow {
			funcType = sema.Account_CapabilitiesTypeBorrowFunctionType
		} else {
			funcType = sema.Account_CapabilitiesTypeGetFunctionType
		}

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			funcType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange
				typeParameter := invocation.TypeParameterTypes.Oldest().Value

				// Get path argument
				pathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountCapabilitiesGet(
					invocationContext,
					controllerHandler,
					pathValue,
					typeParameter,
					borrow,
					addressValue,
					locationRange,
				)
			},
		)
	}
}

func NewVMAccountCapabilitiesGetFunction(
	controllerHandler CapabilityControllerHandler,
	borrow bool,
) VMFunction {

	var (
		funcName string
		funcType *sema.FunctionType
	)
	if borrow {
		funcName = sema.Account_CapabilitiesTypeBorrowFunctionName
		funcType = sema.Account_CapabilitiesTypeBorrowFunctionType
	} else {
		funcName = sema.Account_CapabilitiesTypeGetFunctionName
		funcType = sema.Account_CapabilitiesTypeGetFunctionType
	}

	return VMFunction{
		BaseType: sema.Account_CapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			funcName,
			funcType,
			func(context *vm.Context, typeArguments []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver)

				pathValue, ok := args[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				borrowType := typeArguments[0]
				semaBorrowType := interpreter.MustConvertStaticToSemaType(borrowType, context)

				return AccountCapabilitiesGet(
					context,
					controllerHandler,
					pathValue,
					semaBorrowType,
					borrow,
					address,
					interpreter.EmptyLocationRange,
				)
			},
		),
	}
}

func AccountCapabilitiesGet(
	invocationContext interpreter.InvocationContext,
	controllerHandler CapabilityControllerHandler,
	pathValue interpreter.PathValue,
	typeParameter sema.Type,
	borrow bool,
	addressValue interpreter.AddressValue,
	locationRange interpreter.LocationRange,
) interpreter.Value {
	if pathValue.Domain != common.PathDomainPublic {
		panic(errors.NewUnreachableError())
	}

	domain := pathValue.Domain.StorageDomain()
	identifier := pathValue.Identifier

	// Get borrow type type argument

	// `Never` is never a supertype of any stored value
	if typeParameter.Equal(sema.NeverType) {
		if borrow {
			return interpreter.Nil
		} else {
			return interpreter.NewInvalidCapabilityValue(
				invocationContext,
				addressValue,
				interpreter.PrimitiveStaticTypeNever,
			)
		}
	}

	wantedBorrowType, ok := typeParameter.(*sema.ReferenceType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var failValue interpreter.Value
	if borrow {
		failValue = interpreter.Nil
	} else {
		failValue =
			interpreter.NewInvalidCapabilityValue(
				invocationContext,
				addressValue,
				interpreter.ConvertSemaToStaticType(invocationContext, wantedBorrowType),
			)
	}

	// Read stored capability, if any

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	address := addressValue.ToAddress()

	readValue := invocationContext.ReadStored(address, domain, storageMapKey)
	if readValue == nil {
		return failValue
	}

	var (
		capabilityID               interpreter.UInt64Value
		capabilityAddress          interpreter.AddressValue
		capabilityStaticBorrowType interpreter.StaticType
	)
	switch readValue := readValue.(type) {
	case *interpreter.IDCapabilityValue:
		capabilityID = readValue.ID
		capabilityAddress = readValue.Address()
		capabilityStaticBorrowType = readValue.BorrowType

	case *interpreter.PathCapabilityValue: //nolint:staticcheck
		capabilityID = interpreter.InvalidCapabilityID
		capabilityAddress = readValue.Address()
		capabilityStaticBorrowType = readValue.BorrowType
		if capabilityStaticBorrowType == nil {
			capabilityStaticBorrowType = &interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeNever,
			}
		}

	case interpreter.PathLinkValue: //nolint:staticcheck
		// If the stored value is a path link,
		// it failed to be migrated during the Cadence 1.0 migration.
		capabilityID = interpreter.InvalidCapabilityID
		capabilityAddress = addressValue
		capabilityStaticBorrowType = readValue.Type

	default:
		panic(errors.NewUnreachableError())
	}

	capabilityBorrowType, ok :=
		interpreter.MustConvertStaticToSemaType(capabilityStaticBorrowType, invocationContext).(*sema.ReferenceType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	getHandler := invocationContext.GetValidateAccountCapabilitiesGetHandler()
	if getHandler != nil {
		valid, err := getHandler(
			invocationContext,
			locationRange,
			addressValue,
			pathValue,
			wantedBorrowType,
			capabilityBorrowType,
		)
		if err != nil {
			panic(err)
		}
		if !valid {
			return failValue
		}
	}

	var resultValue interpreter.Value
	if borrow {
		// When borrowing,
		// check the controller and types,
		// and return a checked reference

		resultValue = BorrowCapabilityController(
			invocationContext,
			locationRange,
			capabilityAddress,
			capabilityID,
			wantedBorrowType,
			capabilityBorrowType,
			controllerHandler,
		)
	} else {
		// When not borrowing,
		// check the controller and types,
		// and return a capability

		controller, resultBorrowType := getCheckedCapabilityController(
			invocationContext,
			capabilityAddress,
			capabilityID,
			wantedBorrowType,
			capabilityBorrowType,
			controllerHandler,
		)
		if controller != nil {
			resultBorrowStaticType :=
				interpreter.ConvertSemaReferenceTypeToStaticReferenceType(invocationContext, resultBorrowType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			resultValue = interpreter.NewCapabilityValue(
				invocationContext,
				capabilityID,
				capabilityAddress,
				resultBorrowStaticType,
			)
		}
	}

	if resultValue == nil {
		return failValue
	}

	if borrow {
		resultValue = interpreter.NewSomeValueNonCopying(
			invocationContext,
			resultValue,
		)
	}

	return resultValue
}

func newInterpreterAccountCapabilitiesExistsFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
) interpreter.BoundFunctionGenerator {
	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			sema.Account_CapabilitiesTypeExistsFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {
				invocationContext := invocation.InvocationContext
				pathValue, ok := invocation.Arguments[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountCapabilitiesExists(
					invocationContext,
					pathValue,
					address,
				)
			},
		)
	}
}

var VMAccountCapabilitiesExistsFunction = VMFunction{
	BaseType: sema.Account_CapabilitiesType,
	FunctionValue: vm.NewNativeFunctionValue(
		sema.Account_CapabilitiesTypeExistsFunctionName,
		sema.Account_CapabilitiesTypeExistsFunctionType,
		func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
			var receiver interpreter.Value

			// arg[0] is the receiver. Actual arguments starts from 1.
			receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

			// Get address field from the receiver
			accountAddress := vm.GetAccountTypePrivateAddressValue(receiver)

			pathValue, ok := args[0].(interpreter.PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return AccountCapabilitiesExists(
				context,
				pathValue,
				accountAddress.ToAddress(),
			)
		},
	),
}

func AccountCapabilitiesExists(
	invocationContext interpreter.InvocationContext,
	pathValue interpreter.PathValue,
	address common.Address,
) interpreter.Value {
	// Get path argument
	if pathValue.Domain != common.PathDomainPublic {
		panic(errors.NewUnreachableError())
	}

	domain := pathValue.Domain.StorageDomain()
	identifier := pathValue.Identifier

	// Read stored capability, if any

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	return interpreter.BoolValue(
		interpreter.StoredValueExists(invocationContext, address, domain, storageMapKey),
	)
}

func getAccountCapabilityControllerReference(
	context interpreter.CapabilityControllerReferenceContext,
	locationRange interpreter.LocationRange,
	address common.Address,
	capabilityID uint64,
	handler CapabilityControllerHandler,
) *interpreter.EphemeralReferenceValue {

	capabilityController := getCapabilityController(
		context,
		address,
		capabilityID,
		handler,
	)
	if capabilityController == nil {
		return nil
	}

	accountCapabilityController, ok := capabilityController.(*interpreter.AccountCapabilityControllerValue)
	if !ok {
		return nil
	}

	return interpreter.NewEphemeralReferenceValue(
		context,
		interpreter.UnauthorizedAccess,
		accountCapabilityController,
		sema.AccountCapabilityControllerType,
		locationRange,
	)
}

func newInterpreterAccountAccountCapabilitiesGetControllerFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerHandler,
) interpreter.BoundFunctionGenerator {
	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			sema.Account_AccountCapabilitiesTypeGetControllerFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				inter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get capability ID argument

				capabilityIDValue, ok := invocation.Arguments[0].(interpreter.UInt64Value)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				capabilityID := uint64(capabilityIDValue)

				referenceValue := getAccountCapabilityControllerReference(
					inter,
					locationRange,
					address,
					capabilityID,
					handler,
				)
				if referenceValue == nil {
					return interpreter.Nil
				}

				return interpreter.NewSomeValueNonCopying(inter, referenceValue)
			},
		)
	}
}

func NewVMAccountAccountCapabilitiesGetControllerFunction(
	handler CapabilityControllerHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_AccountCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_AccountCapabilitiesTypeGetControllerFunctionName,
			sema.Account_AccountCapabilitiesTypeGetControllerFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Get capability ID argument

				capabilityIDValue, ok := args[0].(interpreter.UInt64Value)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				capabilityID := uint64(capabilityIDValue)

				referenceValue := getAccountCapabilityControllerReference(
					context,
					interpreter.EmptyLocationRange,
					address,
					capabilityID,
					handler,
				)
				if referenceValue == nil {
					return interpreter.Nil
				}

				return interpreter.NewSomeValueNonCopying(context, referenceValue)
			},
		),
	}
}

var accountCapabilityControllerReferencesArrayStaticType = &interpreter.VariableSizedStaticType{
	Type: &interpreter.ReferenceStaticType{
		ReferencedType: interpreter.PrimitiveStaticTypeAccountCapabilityController,
		Authorization:  interpreter.UnauthorizedAccess,
	},
}

func newInterpreterAccountAccountCapabilitiesGetControllersFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerHandler,
) interpreter.BoundFunctionGenerator {
	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()
		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			sema.Account_AccountCapabilitiesTypeGetControllersFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				inter := invocation.InvocationContext
				locationRange := invocation.LocationRange

				return accountAccountCapabilitiesGetControllers(
					inter,
					address,
					locationRange,
					handler,
				)
			},
		)
	}
}

func NewVMAccountAccountCapabilitiesGetControllersFunction(
	handler CapabilityControllerHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_AccountCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_AccountCapabilitiesTypeGetControllersFunctionName,
			sema.Account_AccountCapabilitiesTypeGetControllersFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {

				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:] // nolint:staticcheck

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				return accountAccountCapabilitiesGetControllers(
					context,
					address,
					interpreter.EmptyLocationRange,
					handler,
				)
			},
		),
	}
}

func accountAccountCapabilitiesGetControllers(
	context interpreter.InvocationContext,
	address common.Address,
	locationRange interpreter.LocationRange,
	handler CapabilityControllerHandler,
) interpreter.Value {
	// Get capability controllers iterator

	nextCapabilityID, count := getAccountCapabilityControllerIDsIterator(context, address)

	var capabilityControllerIndex uint64 = 0

	return interpreter.NewArrayValueWithIterator(
		context,
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
				context,
				locationRange,
				address,
				capabilityID,
				handler,
			)
			if referenceValue == nil {
				panic(errors.NewUnreachableError())
			}

			return referenceValue
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

func newInterpreterAccountAccountCapabilitiesForEachControllerFunction(
	context interpreter.FunctionCreationContext,
	addressValue interpreter.AddressValue,
	handler CapabilityControllerHandler,
) interpreter.BoundFunctionGenerator {
	return func(accountCapabilities interpreter.MemberAccessibleValue) interpreter.BoundFunctionValue {
		address := addressValue.ToAddress()

		return interpreter.NewBoundHostFunctionValue(
			context,
			accountCapabilities,
			sema.Account_AccountCapabilitiesTypeForEachControllerFunctionType,
			func(_ interpreter.MemberAccessibleValue, invocation interpreter.Invocation) interpreter.Value {

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				// Get function argument

				functionValue, ok := invocation.Arguments[0].(interpreter.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountCapabilitiesForEachController(
					invocationContext,
					address,
					functionValue,
					locationRange,
					handler,
				)
			},
		)
	}
}

func NewVMAccountAccountCapabilitiesForEachControllerFunction(
	handler CapabilityControllerHandler,
) VMFunction {
	return VMFunction{
		BaseType: sema.Account_AccountCapabilitiesType,
		FunctionValue: vm.NewNativeFunctionValue(
			sema.Account_AccountCapabilitiesTypeForEachControllerFunctionName,
			sema.Account_AccountCapabilitiesTypeForEachControllerFunctionType,
			func(context *vm.Context, _ []bbq.StaticType, args ...interpreter.Value) interpreter.Value {
				var receiver interpreter.Value

				// arg[0] is the receiver. Actual arguments starts from 1.
				receiver, args = args[vm.ReceiverIndex], args[vm.TypeBoundFunctionArgumentOffset:]

				// Get address field from the receiver
				address := vm.GetAccountTypePrivateAddressValue(receiver).ToAddress()

				// Get function argument

				functionValue, ok := args[0].(interpreter.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return AccountCapabilitiesForEachController(
					context,
					address,
					functionValue,
					interpreter.EmptyLocationRange,
					handler,
				)
			},
		),
	}
}

func AccountCapabilitiesForEachController(
	invocationContext interpreter.InvocationContext,
	address common.Address,
	functionValue interpreter.FunctionValue,
	locationRange interpreter.LocationRange,
	handler CapabilityControllerHandler,
) interpreter.Value {
	functionValueType := functionValue.FunctionType(invocationContext)
	parameterTypes := functionValueType.ParameterTypes()
	returnType := functionValueType.ReturnTypeAnnotation.Type

	// Prevent mutations (record/unrecord) to account capability controllers
	// for this address during iteration

	addressPath := interpreter.AddressPath{
		Address: address,
	}
	iterations := invocationContext.GetCapabilityControllerIterations()
	iterations[addressPath]++
	defer func() {
		iterations[addressPath]--
		if iterations[addressPath] <= 0 {
			delete(iterations, addressPath)
		}
	}()

	// Get capability controllers iterator

	nextCapabilityID, _ :=
		getAccountCapabilityControllerIDsIterator(invocationContext, address)

	for {
		capabilityID, ok := nextCapabilityID()
		if !ok {
			break
		}

		referenceValue := getAccountCapabilityControllerReference(
			invocationContext,
			locationRange,
			address,
			capabilityID,
			handler,
		)
		if referenceValue == nil {
			panic(errors.NewUnreachableError())
		}

		res, err := interpreter.InvokeFunctionValue(
			invocationContext,
			functionValue,
			[]interpreter.Value{referenceValue},
			accountAccountCapabilitiesForEachControllerCallbackTypeParams,
			parameterTypes,
			returnType,
			locationRange,
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

		// It is not safe to check this at the beginning of the loop
		// (i.e. on the next invocation of the callback),
		// because if the mutation performed in the callback reorganized storage
		// such that the iteration pointer is now at the end,
		// we will not invoke the callback again but will still silently skip elements of storage.
		//
		// In order to be safe, we perform this check here to effectively enforce
		// that users return `false` from their callback in all cases where storage is mutated.
		if invocationContext.MutationDuringCapabilityControllerIteration() {
			panic(CapabilityControllersMutatedDuringIterationError{
				LocationRange: locationRange,
			})
		}
	}

	return interpreter.Void
}

func newAccountCapabilityControllerDeleteFunction(
	address common.Address,
	controller *interpreter.AccountCapabilityControllerValue,
	handler CapabilityControllerHandler,
) func(interpreter.CapabilityControllerContext, interpreter.LocationRange) {
	return func(context interpreter.CapabilityControllerContext, locationRange interpreter.LocationRange) {
		capabilityID := controller.CapabilityID

		unrecordAccountCapabilityController(
			context,
			address,
			capabilityID,
		)
		removeCapabilityController(
			context,
			address,
			capabilityID,
		)

		addressValue := interpreter.AddressValue(address)

		handler.EmitEvent(
			context,
			locationRange,
			AccountCapabilityControllerDeletedEventType,
			[]interpreter.Value{
				capabilityID,
				addressValue,
			},
		)
	}
}

func getCapabilityControllerTag(
	storageReader interpreter.StorageReader,
	address common.Address,
	capabilityID uint64,
) *interpreter.StringValue {

	value := storageReader.ReadStored(
		address,
		common.StorageDomainCapabilityControllerTag,
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
) func(common.MemoryGauge) *interpreter.IDCapabilityValue {

	addressValue := interpreter.AddressValue(address)
	capabilityID := controller.ControllerCapabilityID()
	borrowType := controller.CapabilityControllerBorrowType()

	return func(gauge common.MemoryGauge) *interpreter.IDCapabilityValue {
		return interpreter.NewCapabilityValue(
			gauge,
			capabilityID,
			addressValue,
			borrowType,
		)
	}
}

func newCapabilityControllerGetTagFunction(
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) func(interpreter.StorageReader) *interpreter.StringValue {

	return func(storageReader interpreter.StorageReader) *interpreter.StringValue {
		return getCapabilityControllerTag(
			storageReader,
			address,
			uint64(capabilityIDValue),
		)
	}
}

func SetCapabilityControllerTag(
	storageWriter interpreter.StorageWriter,
	address common.Address,
	capabilityID uint64,
	tagValue *interpreter.StringValue,
) {
	// avoid typed nil
	var value interpreter.Value
	if tagValue != nil {
		value = tagValue
	}

	storageWriter.WriteStored(
		address,
		common.StorageDomainCapabilityControllerTag,
		interpreter.Uint64StorageMapKey(capabilityID),
		value,
	)
}

func newCapabilityControllerSetTagFunction(
	address common.Address,
	capabilityIDValue interpreter.UInt64Value,
) func(interpreter.StorageWriter, *interpreter.StringValue) {
	return func(storageWriter interpreter.StorageWriter, tagValue *interpreter.StringValue) {
		SetCapabilityControllerTag(
			storageWriter,
			address,
			uint64(capabilityIDValue),
			tagValue,
		)
	}
}
