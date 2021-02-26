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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
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

type SignatureAlgorithm = sema.SignatureAlgorithm

const (
	// Supported signing algorithms
	SignatureAlgorithmECDSA_P256 SignatureAlgorithm = iota
	SignatureAlgorithmECDSA_Secp256k1
	//SignatureAlgorithmBLSBLS12381
)

type HashAlgorithm = sema.HashAlgorithm

const (
	// Supported hashing algorithms
	HashAlgorithmSHA2_256 HashAlgorithm = iota
	HashAlgorithmSHA3_256
	//HashAlgorithmKMAC128
	//HashAlgorithmSHA3_384
	//HashAlgorithmSHA2_384
)

type AccountKey struct {
	KeyIndex  int
	PublicKey *PublicKey
	HashAlgo  HashAlgorithm
	Weight    int
	IsRevoked bool
}

type PublicKey struct {
	PublicKey []byte
	SignAlgo  SignatureAlgorithm
}

type Interface interface {
	// ResolveLocation resolves an import location.
	ResolveLocation(identifiers []Identifier, location Location) ([]ResolvedLocation, error)
	// GetCode returns the code at a given location
	GetCode(location Location) ([]byte, error)
	// GetProgram attempts gets the program for the given location, if available.
	//
	// NOTE: During execution, this function must always return the *same* program,
	// i.e. it may NOT return a different program,
	// an elaboration in the program that is not annotating the AST in the program;
	// or a program/elaboration and then nothing in a subsequent call.
	//
	// This function must also return what was set using SetProgram,
	// it may NOT return something different or nothing (!) after SetProgram was called.
	//
	// This is not a caching function!
	//
	GetProgram(Location) (*interpreter.Program, error)
	// SetProgram sets the program for the given location.
	SetProgram(Location, *interpreter.Program) error
	// GetValue gets a value for the given key in the storage, owned by the given account.
	GetValue(owner, key []byte) (value []byte, err error)
	// SetValue sets a value for the given key in the storage, owned by the given account.
	SetValue(owner, key, value []byte) (err error)
	// CreateAccount creates a new account.
	CreateAccount(payer Address) (address Address, err error)
	// AddAccountKey appends a key to an account.
	AddAccountKey(address Address, publicKey *PublicKey, hashAlgo HashAlgorithm, weight int) (*AccountKey, error)
	// GetAccountKey retrieves a key from an account by index.
	GetAccountKey(address Address, index int) (*AccountKey, error)
	// RemoveAccountKey removes a key from an account by index.
	RemoveAccountKey(address Address, index int) (*AccountKey, error)
	// UpdateAccountContractCode updates the code associated with an account contract.
	UpdateAccountContractCode(address Address, name string, code []byte) (err error)
	// GetAccountContractCode returns the code associated with an account contract.
	GetAccountContractCode(address Address, name string) (code []byte, err error)
	// RemoveAccountContractCode removes the code associated with an account contract.
	RemoveAccountContractCode(address Address, name string) (err error)
	// GetSigningAccounts returns the signing accounts.
	GetSigningAccounts() ([]Address, error)
	// ProgramLog logs program logs.
	ProgramLog(string) error
	// EmitEvent is called when an event is emitted by the runtime.
	EmitEvent(cadence.Event) error
	// ValueExists returns true if the given key exists in the storage, owned by the given account.
	ValueExists(owner, key []byte) (exists bool, err error)
	// GenerateUUID is called to generate a UUID.
	GenerateUUID() (uint64, error)
	// GetComputationLimit returns the computation limit. A value <= 0 means there is no limit
	GetComputationLimit() uint64
	// SetComputationUsed reports the amount of computation used.
	SetComputationUsed(used uint64) error
	// DecodeArgument decodes a transaction argument against the given type.
	DecodeArgument(argument []byte, argumentType cadence.Type) (cadence.Value, error)
	// GetCurrentBlockHeight returns the current block height.
	GetCurrentBlockHeight() (uint64, error)
	// GetBlockAtHeight returns the block at the given height.
	GetBlockAtHeight(height uint64) (block Block, exists bool, err error)
	// UnsafeRandom returns a random uint64, where the process of random number derivation is not cryptographically
	// secure.
	UnsafeRandom() (uint64, error)
	// VerifySignature returns true if the given signature was produced by signing the given tag + data
	// using the given public key, signature algorithm, and hash algorithm.
	VerifySignature(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm SignatureAlgorithm,
		hashAlgorithm HashAlgorithm,
	) (bool, error)
	// Hash returns the digest of hashing the given data with using the given hash algorithm
	Hash(data []byte, hashAlgorithm HashAlgorithm) ([]byte, error)
	// GetStorageUsed gets storage used in bytes by the address at the moment of the function call.
	GetStorageUsed(address Address) (value uint64, err error)
	// GetStorageCapacity gets storage capacity in bytes on the address.
	GetStorageCapacity(address Address) (value uint64, err error)
	// ImplementationDebugLog logs implementation log statements on a debug-level
	ImplementationDebugLog(message string) error
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
	ProgramParsed(location common.Location, duration time.Duration)
	ProgramChecked(location common.Location, duration time.Duration)
	ProgramInterpreted(location common.Location, duration time.Duration)
	ValueEncoded(duration time.Duration)
	ValueDecoded(duration time.Duration)
}

type emptyRuntimeInterface struct {
	programs map[common.LocationID]*interpreter.Program
}

// emptyRuntimeInterface should implement Interface
var _ Interface = &emptyRuntimeInterface{}

func NewEmptyRuntimeInterface() Interface {
	return &emptyRuntimeInterface{
		programs: map[common.LocationID]*interpreter.Program{},
	}
}

func (i *emptyRuntimeInterface) ResolveLocation(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
	return []ResolvedLocation{
		{
			Location:    location,
			Identifiers: identifiers,
		},
	}, nil
}

func (i *emptyRuntimeInterface) SetProgram(location Location, program *interpreter.Program) error {
	i.programs[location.ID()] = program

	return nil
}

func (i *emptyRuntimeInterface) GetProgram(location Location) (*interpreter.Program, error) {
	return i.programs[location.ID()], nil
}

func (i *emptyRuntimeInterface) GetCode(_ Location) ([]byte, error) {
	return nil, nil
}

func (i *emptyRuntimeInterface) ValueExists(_, _ []byte) (exists bool, err error) {
	return false, nil
}

func (i *emptyRuntimeInterface) GetValue(_, _ []byte) (value []byte, err error) {
	return nil, nil
}

func (i *emptyRuntimeInterface) SetValue(_, _, _ []byte) error {
	return nil
}

func (i *emptyRuntimeInterface) CreateAccount(_ Address) (address Address, err error) {
	return Address{}, nil
}

func (i *emptyRuntimeInterface) AddAccountKey(_ Address, _ *PublicKey, _ HashAlgorithm, _ int) (*AccountKey, error) {
	return nil, nil
}

func (i *emptyRuntimeInterface) RemoveAccountKey(_ Address, _ int) (*AccountKey, error) {
	return nil, nil
}

func (i *emptyRuntimeInterface) GetAccountKey(_ Address, _ int) (*AccountKey, error) {
	return nil, nil
}

func (i *emptyRuntimeInterface) UpdateAccountCode(_ Address, _ []byte) error {
	return nil
}

func (i *emptyRuntimeInterface) UpdateAccountContractCode(_ Address, _ string, _ []byte) (err error) {
	return nil
}

func (i *emptyRuntimeInterface) GetAccountContractCode(_ Address, _ string) (code []byte, err error) {
	return nil, nil
}

func (i *emptyRuntimeInterface) RemoveAccountContractCode(_ Address, _ string) (err error) {
	return nil
}

func (i *emptyRuntimeInterface) GetSigningAccounts() ([]Address, error) {
	return nil, nil
}

func (i *emptyRuntimeInterface) ProgramLog(_ string) error {
	return nil
}

func (i *emptyRuntimeInterface) EmitEvent(_ cadence.Event) error {
	return nil
}

func (i *emptyRuntimeInterface) GenerateUUID() (uint64, error) {
	return 0, nil
}

func (i *emptyRuntimeInterface) GetComputationLimit() uint64 {
	return 0
}

func (i *emptyRuntimeInterface) SetComputationUsed(uint64) error {
	return nil
}

func (i *emptyRuntimeInterface) DecodeArgument(_ []byte, _ cadence.Type) (cadence.Value, error) {
	return nil, nil
}

func (i *emptyRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	return 0, nil
}

func (i *emptyRuntimeInterface) GetBlockAtHeight(_ uint64) (block Block, exists bool, err error) {
	return
}

func (i *emptyRuntimeInterface) UnsafeRandom() (uint64, error) {
	return 0, nil
}

func (i *emptyRuntimeInterface) ImplementationDebugLog(_ string) error {
	return nil
}

func (i *emptyRuntimeInterface) VerifySignature(
	_ []byte,
	_ string,
	_ []byte,
	_ []byte,
	_ SignatureAlgorithm,
	_ HashAlgorithm,
) (bool, error) {
	return false, nil
}

func (i *emptyRuntimeInterface) Hash(
	_ []byte,
	_ HashAlgorithm,
) ([]byte, error) {
	return nil, nil
}

func (i emptyRuntimeInterface) GetStorageUsed(_ Address) (uint64, error) {
	return 0, nil
}

func (i emptyRuntimeInterface) GetStorageCapacity(_ Address) (uint64, error) {
	return 0, nil
}
