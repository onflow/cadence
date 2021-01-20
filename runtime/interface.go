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

type Accounts interface {

	// creates a new account address and set the exists flag for this account
	CreateAccount(caller Location) (address Address, err error)
	// Exists returns true if the account exists
	Exists(address Address) (exists bool, err error)
	// SuspendAccount suspends an account
	SuspendAccount(address Address, caller Location) error
	// UnsuspendAccount unsuspend an account
	UnsuspendAccount(address Address, caller Location) error

	// TODO ramtin merge this with 	Code(location Location) error
	// AccountContractCode returns the code associated with an account contract.
	ContractCode(address AddressLocation) (code []byte, err error)
	// UpdateAccountContractCode updates the code associated with an account contract.
	UpdateContractCode(address AddressLocation, code []byte, caller Location) (err error)
	// RemoveContractCode removes the code associated with an account contract.
	RemoveContractCode(address AddressLocation, caller Location) (err error)

	// AddAccountKey appends a key to an account.
	AddAccountKey(address Address, publicKey []byte, caller Location) error
	// RemoveAccountKey removes a key from an account by index.
	RevokeAccountKey(address Address, index uint, caller Location) (publicKey []byte, err error)
	// TODO RAMTIN do I need the tag here?
	VerifyAccountSignature(address Address, index uint, signature []byte, tag string, signedData []byte) (bool, error)

	// StoredKeys returns list of keys and their sizes owned by this account
	StoredKeys(caller Location)
	// StorageUsed gets storage used in bytes by the address at the moment of the function call.
	StorageUsed(address Address, caller Location) (value uint64, err error)
	// StorageCapacity gets storage capacity in bytes on the address.
	StorageCapacity(address Address, caller Location) (value uint64, err error) // Do we need this?

	// Value gets a value for the given key in the storage, owned by the given account.
	Value(owner, key []byte, caller Location) (value []byte, err error)
	// SetValue sets a value for the given key in the storage, owned by the given account.
	SetValue(owner, key, value []byte, caller Location) (err error)
	// ValueExists returns true if the given key exists in the storage, owned by the given account.
	ValueExists(owner, key []byte, caller Location) (exists bool, err error)
}

type Results interface {
	AppendLog(string) error
	Logs() ([]string, error)

	AppendEvent(cadence.Event) error
	Events() ([]cadence.Event, error)

	AppendError(error)
	Errors() multierror.Error

	AddComputationUsed(uint64)
	ComputationSpent() uint64
}

type Metrics interface {
	ProgramParsed(location common.Location, duration time.Duration)
	ProgramChecked(location common.Location, duration time.Duration)
	ProgramInterpreted(location common.Location, duration time.Duration)

	ValueEncoded(duration time.Duration)
	ValueDecoded(duration time.Duration)
}

type HighLevelAccounts interface {
	Accounts

	// HighLevelStorageEnabled should return true
	// if the functions of HighLevelStorage should be called,
	// e.g. SetCadenceValue
	HighLevelStorageEnabled() bool

	// SetCadenceValue sets a value for the given key in the storage, owned by the given account.
	SetCadenceValue(owner Address, key string, value cadence.Value) (err error)
}

type EmptyRuntimeInterface struct{}

var _ Interface = &EmptyRuntimeInterface{}

func (i *EmptyRuntimeInterface) ResolveLocation(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
	return []ResolvedLocation{
		{
			Location:    location,
			Identifiers: identifiers,
		},
	}, nil
}

func (i *EmptyRuntimeInterface) GetCode(_ Location) ([]byte, error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) GetCachedProgram(_ Location) (*ast.Program, error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) CacheProgram(_ Location, _ *ast.Program) error {
	return nil
}

func (i *EmptyRuntimeInterface) ValueExists(_, _ []byte) (exists bool, err error) {
	return false, nil
}

func (i *EmptyRuntimeInterface) GetValue(_, _ []byte) (value []byte, err error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) SetValue(_, _, _ []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) CreateAccount(_ Address) (address Address, err error) {
	return Address{}, nil
}

func (i *EmptyRuntimeInterface) AddAccountKey(_ Address, _ []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) RemoveAccountKey(_ Address, _ int) (publicKey []byte, err error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) UpdateAccountCode(_ Address, _ []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) UpdateAccountContractCode(_ Address, _ string, _ []byte) (err error) {
	return nil
}

func (i *EmptyRuntimeInterface) GetAccountContractCode(_ Address, _ string) (code []byte, err error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) RemoveAccountContractCode(_ Address, _ string) (err error) {
	return nil
}

func (i *EmptyRuntimeInterface) GetSigningAccounts() ([]Address, error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) Log(_ string) error {
	return nil
}

func (i *EmptyRuntimeInterface) EmitEvent(_ cadence.Event) error {
	return nil
}

func (i *EmptyRuntimeInterface) GenerateUUID() (uint64, error) {
	return 0, nil
}

func (i *EmptyRuntimeInterface) GetComputationLimit() uint64 {
	return 0
}

func (i *EmptyRuntimeInterface) SetComputationUsed(uint64) error {
	return nil
}

func (i *EmptyRuntimeInterface) DecodeArgument(_ []byte, _ cadence.Type) (cadence.Value, error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	return 0, nil
}

func (i *EmptyRuntimeInterface) GetBlockAtHeight(_ uint64) (block Block, exists bool, err error) {
	return
}

func (i *EmptyRuntimeInterface) UnsafeRandom() (uint64, error) {
	return 0, nil
}

func (i *EmptyRuntimeInterface) VerifySignature(
	_ []byte,
	_ string,
	_ []byte,
	_ []byte,
	_ string,
	_ string,
) (bool, error) {
	return false, nil
}

func (i *EmptyRuntimeInterface) Hash(
	_ []byte,
	_ string,
) ([]byte, error) {
	return nil, nil
}

func (i EmptyRuntimeInterface) GetStorageUsed(_ Address) (uint64, error) {
	return 0, nil
}

func (i EmptyRuntimeInterface) GetStorageCapacity(_ Address) (uint64, error) {
	return 0, nil
}
