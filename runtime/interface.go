/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package runtime

import (
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

const BlockHashLength = 32

type BlockHash [BlockHashLength]byte

type Block struct {
	Height    uint64
	View      uint64
	Hash      BlockHash
	Timestamp int64
}

type ResolvedLocation = sema.ResolvedLocation
type Identifier = ast.Identifier
type Location = common.Location
type AddressLocation = common.AddressLocation

// Accounts manages changes to the accounts
//
// errors returns by methods are non-fatal ones (e.g. accountNotExist),
// fatal errors (e.g. Storage failure) are called by panic inside the methods
// methods with power of edits have caller as an argument which can be used
// to limit access to these method calls from smart contracts only, transaction only, or
// specific smart contracts (e.g. service account).
type Accounts interface {
	// NewAccount creates a new account address and set the exists flag for this account
	NewAccount(caller Location) (address Address, err error)
	// AccountExists returns true if the account exists
	AccountExists(address Address) (exists bool, err error)
	// NumberOfAccounts returns the number of accounts
	NumberOfAccounts(caller Location) (count uint64, err error)

	// SuspendAccount suspends an account (set suspend flag to true)
	SuspendAccount(address Address, caller Location) error
	// UnsuspendAccount unsuspend an account (set suspend flag to false)
	UnsuspendAccount(address Address, caller Location) error
	// returns true if account is suspended
	IsAccountSuspended(address Address) (isSuspended bool, err error)
}

// AccountContracts manages contracts stored under accounts
type AccountContracts interface {
	// AccountContractCode returns the code associated with an account contract.
	ContractCode(address AddressLocation) (code []byte, err error)
	// UpdateAccountContractCode updates the code associated with an account contract.
	UpdateContractCode(address AddressLocation, code []byte, caller Location) (err error)
	// RemoveContractCode removes the code associated with an account contract.
	RemoveContractCode(address AddressLocation, caller Location) (err error)
	// Contracts returns a list of contract names under this account
	Contracts(address AddressLocation, caller Location) (name []string, err error)
}

// AccountStorage stores and retrives account key/values
type AccountStorage interface {
	// Value gets a value for the given key in the storage, owned by the given account.
	Value(key StorageKey, caller Location) (value StorageValue, err error)
	// SetValue sets a value for the given key in the storage, owned by the given account.
	SetValue(key StorageKey, value StorageValue, caller Location) (err error)
	// ValueExists returns true if the given key exists in the storage, owned by the given account.
	ValueExists(key StorageKey, caller Location) (exists bool, err error)
	// StoredKeys returns list of keys and their sizes owned by this account
	StoredKeys(address Address, caller Location) (iter StorageKeyIterator, err error)
	// StorageUsed gets storage used in bytes by the address at the moment of the function call.
	StorageUsed(address Address, caller Location) (value uint64, err error)
	// Note: StorageCapacity has been moved to injected methods (similar to get balance)
}

// AccountKeys manages account keys
type AccountKeys interface {
	// AddAccountKey appends a key to an account.
	AddAccountKey(address Address, publicKey []byte, caller Location) error
	// RemoveAccountKey removes a key from an account by index.
	RevokeAccountKey(address Address, index int, caller Location) (publicKey []byte, err error)
	// AccountPublicKey returns the account key for the given index
	AccountPublicKey(address Address, index int, caller Location) (publicKey []byte, err error)
}

type LocationResolver interface {
	// GetCode returns the code at a given location
	GetCode(location Location) ([]byte, error)
	// ResolveLocation resolves an import location.
	ResolveLocation(identifiers []Identifier, location Location) ([]ResolvedLocation, error)
}

// Results are responsible to capture artifacts generated
// when running the runnable
//
// Results won't be directly callable by users
// and will be used by the runtime to capture outputs
type Results interface {
	// AppendLog appends a log to the log collection
	AppendLog(string) error
	// Logs returns all the logs
	Logs() ([]string, error)
	// returns log i of the log collection
	LogAt(i uint) (string, error)
	// returns number of logs
	LogCount() uint

	// AppendEvent appends an event to the event collection
	AppendEvent(cadence.Event) error
	// Events returns all the events
	Events() ([]cadence.Event, error)
	// returns event i of the event collection
	EventAt(i uint) (cadence.Event, error)
	// returns number of events
	EventCount() uint

	// AppendError appends a non-fatal error
	AppendError(error) error
	// return all the errors
	Errors() multierror.Error
	// returns event i of the event collection, the first error is the actual runtime error, the second error
	// returns if there is any error while fetching the error i
	ErrorAt(i uint) (Error, error)
	// returns number of errors
	ErrorCount() uint

	// AddComputationUsed adds a new uint64 value to the computationUsed (computation accumulator)
	AddComputationUsed(uint64) error
	// ComputationSpent returns the total amount of computation spent during the execution
	ComputationSpent() uint64
	// ComputationLimit returns the max computation limit allowed while running
	// Ramtin: (we might not need this to be passed and just be enforced in the Results)
	ComputationLimit() uint64
}

// CacheProvider provides caching functionality to the cadence runtime
//
// cache should be tx failure aware (rollback changes)
// cache should not break verification (register touches when returns)
type CacheProvider interface {
	// GetCachedProgram attempts to get a parsed program from a cache.
	GetCachedProgram(Location) (*ast.Program, error)
	// CacheProgram adds a parsed program to a cache.
	CacheProgram(Location, *ast.Program) error
}

type CryptoProvider interface {
	// VerifySignature returns true if the given signature was produced by signing the given tag + data
	// using the given public key, signature algorithm, and hash algorithm.
	VerifySignature(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm string,
		hashAlgorithm string,
	) (bool, error)
	// Hash returns the digest of hashing the given data with using the given hash algorithm
	Hash(data []byte, hashAlgorithm string) ([]byte, error)
}

type Metrics interface {
	// ProgramParsed captures the time spent on parsing the program
	ProgramParsed(location common.Location, duration time.Duration)
	// ProgramChecked captures the time spent on checking the parsed program
	ProgramChecked(location common.Location, duration time.Duration)
	// ProgramInterpreted captures the time spent on interpreting the parsed and checked program
	ProgramInterpreted(location common.Location, duration time.Duration)

	// ValueEncoded captures the time spent on encoding a value
	// TODO: maybe add type
	ValueEncoded(duration time.Duration)
	// ValueDecoded capture the time spent on decoding an encoded value
	// TODO: maybe add type
	ValueDecoded(duration time.Duration)
}

type HighLevelAccountStorage interface {
	// HighLevelStorageEnabled should return true
	// if the functions of HighLevelStorage should be called,
	// e.g. SetCadenceValue
	HighLevelStorageEnabled() bool

	// SetCadenceValue sets a value for the given key in the storage, owned by the given account.
	SetCadenceValue(owner Address, key string, value cadence.Value) (err error)
}

type EmptyAccounts struct{}

var _ Accounts = &EmptyAccounts{}

func (i *EmptyAccounts) NewAccount(_ Location) (Address, error) {
	return Address{}, nil
}

func (i *EmptyAccounts) AccountExists(_ Address) (bool, error) {
	return false, nil
}

func (i *EmptyAccounts) NumberOfAccounts(_ Location) (uint64, error) {
	return 0, nil
}

func (i *EmptyAccounts) SuspendAccount(_ Address, _ Location) error {
	return nil
}

func (i *EmptyAccounts) UnsuspendAccount(_ Address, _ Location) error {
	return nil
}

func (i *EmptyAccounts) IsAccountSuspended(_ Address) (bool, error) {
	return false, nil
}

type EmptyAccountContracts struct{}

var _ AccountContracts = &EmptyAccountContracts{}

func (i *EmptyAccountContracts) ContractCode(_ AddressLocation) ([]byte, error) {
	return nil, nil
}

func (i *EmptyAccountContracts) UpdateContractCode(_ AddressLocation, _ []byte, _ Location) (err error) {
	return nil
}

func (i *EmptyAccountContracts) RemoveContractCode(_ AddressLocation, _ Location) (err error) {
	return nil
}

func (i *EmptyAccountContracts) Contracts(_ AddressLocation, _ Location) (name []string, err error) {
	return nil, nil
}

type EmptyAccountStorage struct{}

var _ AccountStorage = &EmptyAccountStorage{}

func (i *EmptyAccountStorage) ValueExists(_ StorageKey, _ Location) (exists bool, err error) {
	return false, nil
}

func (i *EmptyAccountStorage) Value(_ StorageKey, _ Location) (value StorageValue, err error) {
	return nil, nil
}

func (i *EmptyAccountStorage) SetValue(_ StorageKey, _ StorageValue, _ Location) error {
	return nil
}

func (i EmptyAccountStorage) StorageUsed(_ Address, _ Location) (uint64, error) {
	return 0, nil
}

func (i EmptyAccountStorage) StorageCapacity(_ Address, _ Location) (uint64, error) {
	return 0, nil
}

func (i EmptyAccountStorage) StoredKeys(_ Address, _ Location) (StorageKeyIterator, error) {
	return nil, nil
}

type EmptyAccountKeys struct{}

var _ AccountKeys = &EmptyAccountKeys{}

func (i *EmptyAccountKeys) AddAccountKey(_ Address, _ []byte, _ Location) error {
	return nil
}

func (i *EmptyAccountKeys) RevokeAccountKey(_ Address, _ int, _ Location) ([]byte, error) {
	return nil, nil
}

func (i *EmptyAccountKeys) AccountPublicKey(_ Address, _ int, _ Location) ([]byte, error) {
	return nil, nil
}

type EmptyCryptoProvider struct{}

var _ CryptoProvider = &EmptyCryptoProvider{}

func (i *EmptyCryptoProvider) VerifySignature(
	_ []byte,
	_ string,
	_ []byte,
	_ []byte,
	_ string,
	_ string,
) (bool, error) {
	return false, nil
}

func (i *EmptyCryptoProvider) Hash(
	_ []byte,
	_ string,
) ([]byte, error) {
	return nil, nil
}

type EmptyCacheProvider struct{}

var _ CacheProvider = &EmptyCacheProvider{}

func (i *EmptyCacheProvider) GetCachedProgram(_ Location) (*ast.Program, error) {
	return nil, nil
}

func (i *EmptyCacheProvider) CacheProgram(_ Location, _ *ast.Program) error {
	return nil
}

type EmptyResults struct{}

var _ Results = &EmptyResults{}

func (i *EmptyResults) AppendLog(_ string) error {
	return nil
}

func (i *EmptyResults) Logs() ([]string, error) {
	return nil, nil
}

func (i *EmptyResults) LogAt(_ uint) (string, error) {
	return "", nil
}

func (i *EmptyResults) LogCount() uint {
	return 0
}

func (i *EmptyResults) AppendEvent(_ cadence.Event) error {
	return nil
}

func (i *EmptyResults) Events() ([]cadence.Event, error) {
	return nil, nil
}

func (i *EmptyResults) EventAt(_ uint) (cadence.Event, error) {
	return cadence.Event{}, nil
}

func (i *EmptyResults) EventCount() uint {
	return 0
}

func (i *EmptyResults) AppendError(_ error) error {
	return nil
}

func (i *EmptyResults) Errors() multierror.Error {
	return multierror.Error{}
}

func (i *EmptyResults) ErrorAt(_ uint) (Error, error) {
	return Error{}, nil
}

func (i *EmptyResults) ErrorCount() uint {
	return 0
}

func (i *EmptyResults) AddComputationUsed(_ uint64) error {
	return nil
}

func (i *EmptyResults) ComputationSpent() uint64 {
	return 0
}

func (i *EmptyResults) ComputationLimit() uint64 {
	return 0
}

// func (i *EmptyRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
// 	return 0, nil
// }

// func (i *EmptyRuntimeInterface) GetBlockAtHeight(_ uint64) (block Block, exists bool, err error) {
// 	return
// }

// func (i *EmptyRuntimeInterface) GenerateUUID() (uint64, error) {
// 	return 0, nil
// }

// func (i *EmptyRuntimeInterface) UnsafeRandom() (uint64, error) {
// 	return 0, nil
// }

// func (i *EmptyAccounts) ResolveLocation(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
// 	return []ResolvedLocation{
// 		{
// 			Location:    location,
// 			Identifiers: identifiers,
// 		},
// 	}, nil
// }
