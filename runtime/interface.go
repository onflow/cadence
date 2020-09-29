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

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
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

type Interface interface {
	// ResolveLocation resolves an import location.
	ResolveLocation(identifiers []Identifier, location Location) []ResolvedLocation
	// GetCode returns the code at a given location
	GetCode(location Location) ([]byte, error)
	// GetCachedProgram attempts to get a parsed program from a cache.
	GetCachedProgram(Location) (*ast.Program, error)
	// CacheProgram adds a parsed program to a cache.
	CacheProgram(Location, *ast.Program) error
	// GetValue gets a value for the given key in the storage, owned by the given account.
	GetValue(owner, key []byte) (value []byte, err error)
	// SetValue sets a value for the given key in the storage, owned by the given account.
	SetValue(owner, key, value []byte) (err error)
	// CreateAccount creates a new account.
	CreateAccount(payer Address) (address Address, err error)
	// AddAccountKey appends a key to an account.
	AddAccountKey(address Address, publicKey []byte) error
	// RemoveAccountKey removes a key from an account by index.
	RemoveAccountKey(address Address, index int) (publicKey []byte, err error)
	// UpdateAccountCode updates the code associated with an account.
	UpdateAccountCode(address Address, code []byte) (err error)
	// UpdateAccountContractCode updates the code associated with an account contract.
	UpdateAccountContractCode(address Address, name string, code []byte) (err error)
	// GetAccountContractCode returns the code associated with an account contract.
	GetAccountContractCode(address Address, name string) (code []byte, err error)
	// GetSigningAccounts returns the signing accounts.
	GetSigningAccounts() []Address
	// Log logs a string.
	Log(string)
	// EmitEvent is called when an event is emitted by the runtime.
	EmitEvent(cadence.Event)
	// ValueExists returns true if the given key exists in the storage, owned by the given account.
	ValueExists(owner, key []byte) (exists bool, err error)
	// GenerateUUID is called to generate a UUID.
	GenerateUUID() uint64
	// GetComputationLimit returns the computation limit. A value <= 0 means there is no limit
	GetComputationLimit() uint64
	// DecodeArgument decodes a transaction argument against the given type.
	DecodeArgument(argument []byte, argumentType cadence.Type) (cadence.Value, error)
	// GetCurrentBlockHeight returns the current block height.
	GetCurrentBlockHeight() uint64
	// GetBlockAtHeight returns the block at the given height.
	GetBlockAtHeight(height uint64) (block Block, exists bool, err error)
	// UnsafeRandom returns a random uint64, where the process of random number derivation is not cryptographically
	// secure.
	UnsafeRandom() uint64
	// VerifySignature returns true if the given signature was produced by signing the given tag + data
	// using the given public key, signature algorithm, and hash algorithm.
	VerifySignature(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm string,
		hashAlgorithm string,
	) bool

	// Hash returns the digest of hashing the given data with using the given hash algorithm
	Hash(data []byte, hashAlgorithm string) []byte
}

type HighLevelStorage interface {
	Interface

	// HighLevelStorageEnabled should return true
	// if the functions of HighLevelStorage should be called,
	// e.g. SetCadenceValue
	HighLevelStorageEnabled() bool

	// SetCadenceValue sets a value for the given key in the storage, owned by the given account.
	SetCadenceValue(owner Address, key string, value cadence.Value) (err error)
}

type Metrics interface {
	ProgramParsed(location ast.Location, duration time.Duration)
	ProgramChecked(location ast.Location, duration time.Duration)
	ProgramInterpreted(location ast.Location, duration time.Duration)
	ValueEncoded(duration time.Duration)
	ValueDecoded(duration time.Duration)
}

type EmptyRuntimeInterface struct{}

var _ Interface = &EmptyRuntimeInterface{}

func (i *EmptyRuntimeInterface) ResolveLocation(identifiers []Identifier, location Location) []ResolvedLocation {
	return []ResolvedLocation{
		{
			Location:    location,
			Identifiers: identifiers,
		},
	}
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

func (i *EmptyRuntimeInterface) GetSigningAccounts() []Address {
	return nil
}

func (i *EmptyRuntimeInterface) Log(_ string) {}

func (i *EmptyRuntimeInterface) EmitEvent(_ cadence.Event) {}

func (i *EmptyRuntimeInterface) GenerateUUID() uint64 {
	return 0
}

func (i *EmptyRuntimeInterface) GetComputationLimit() uint64 {
	return 0
}

func (i *EmptyRuntimeInterface) DecodeArgument(_ []byte, _ cadence.Type) (cadence.Value, error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) GetCurrentBlockHeight() uint64 {
	return 0
}

func (i *EmptyRuntimeInterface) GetBlockAtHeight(_ uint64) (block Block, exists bool, err error) {
	return
}

func (i *EmptyRuntimeInterface) UnsafeRandom() uint64 {
	return 0
}

func (i *EmptyRuntimeInterface) VerifySignature(
	_ []byte,
	_ string,
	_ []byte,
	_ []byte,
	_ string,
	_ string,
) bool {
	return false
}

func (i *EmptyRuntimeInterface) Hash(
	_ []byte,
	_ string,
) []byte {
	return nil
}
