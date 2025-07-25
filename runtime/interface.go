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

package runtime

import (
	"time"

	"github.com/onflow/atree"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type Interface interface {
	MeterInterface

	// ResolveLocation resolves an import location.
	ResolveLocation(identifiers []Identifier, location Location) ([]ResolvedLocation, error)
	// GetCode returns the code at a given location
	GetCode(location Location) ([]byte, error)
	// GetOrLoadProgram returns the program for the given location, if available,
	// or sets the program by calling the given load function.
	//
	// For implementations:
	// - Perform a lookup for the location and return the program if it exists.
	// - If the program does not exist, call load, and store the result,
	//   *EVEN IF loading failed* (program is nil / error is non-nil)!
	// - During execution of a high-level program (e.g. script, transaction, etc.),
	//   this function MUST always return the *same* program,
	//   i.e. it may NOT return a different program,
	//   an elaboration in the program that is not annotating the AST in the program;
	//   or a program/elaboration and then nothing in a subsequent call.
	// - This function MUST also return exactly what was previously returned from load,
	//   *EVEN IF loading failed* (program is nil / error is non-nil),
	//   and it may NOT return something different
	// - Do NOT implement this as a cache!
	GetOrLoadProgram(
		location Location,
		load func() (*Program, error),
	) (*Program, error)
	// GetValue gets a value for the given key in the storage, owned by the given account.
	GetValue(owner, key []byte) (value []byte, err error)
	// SetValue sets a value for the given key in the storage, owned by the given account.
	SetValue(owner, key, value []byte) (err error)
	// ValueExists returns true if the given key exists in the storage, owned by the given account.
	ValueExists(owner, key []byte) (exists bool, err error)
	// AllocateSlabIndex allocates a new slab index under the given account.
	AllocateSlabIndex(owner []byte) (atree.SlabIndex, error)
	// CreateAccount creates a new account.
	CreateAccount(payer Address) (address Address, err error)
	// AddAccountKey appends a key to an account.
	AddAccountKey(address Address, publicKey *PublicKey, hashAlgo HashAlgorithm, weight int) (*AccountKey, error)
	// GetAccountKey retrieves a key from an account by index.
	GetAccountKey(address Address, index uint32) (*AccountKey, error)
	AccountKeysCount(address Address) (uint32, error)
	// RevokeAccountKey removes a key from an account by index.
	RevokeAccountKey(address Address, index uint32) (*AccountKey, error)
	// UpdateAccountContractCode updates the code associated with an account contract.
	UpdateAccountContractCode(location common.AddressLocation, code []byte) (err error)
	// GetAccountContractCode returns the code associated with an account contract.
	GetAccountContractCode(location common.AddressLocation) (code []byte, err error)
	// RemoveAccountContractCode removes the code associated with an account contract.
	RemoveAccountContractCode(location common.AddressLocation) (err error)
	// GetSigningAccounts returns the signing accounts.
	GetSigningAccounts() ([]Address, error)
	// ProgramLog logs program logs.
	ProgramLog(string) error
	// EmitEvent is called when an event is emitted by the runtime.
	EmitEvent(cadence.Event) error
	// GenerateUUID is called to generate a UUID.
	GenerateUUID() (uint64, error)
	// DecodeArgument decodes a transaction/script argument against the given type.
	DecodeArgument(argument []byte, argumentType cadence.Type) (cadence.Value, error)
	// GetCurrentBlockHeight returns the current block height.
	GetCurrentBlockHeight() (uint64, error)
	// GetBlockAtHeight returns the block at the given height.
	GetBlockAtHeight(height uint64) (block Block, exists bool, err error)
	// ReadRandom reads pseudo-random bytes into the input slice, using distributed randomness.
	ReadRandom([]byte) error
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
	Hash(data []byte, tag string, hashAlgorithm HashAlgorithm) ([]byte, error)
	// GetAccountBalance gets accounts default flow token balance.
	GetAccountBalance(address common.Address) (value uint64, err error)
	// GetAccountAvailableBalance gets accounts default flow token balance - balance that is reserved for storage.
	GetAccountAvailableBalance(address common.Address) (value uint64, err error)
	// GetStorageUsed gets storage used in bytes by the address at the moment of the function call.
	GetStorageUsed(address Address) (value uint64, err error)
	// GetStorageCapacity gets storage capacity in bytes on the address.
	GetStorageCapacity(address Address) (value uint64, err error)
	// ImplementationDebugLog logs implementation log statements on a debug-level
	ImplementationDebugLog(message string) error
	// ValidatePublicKey verifies the validity of a public key.
	ValidatePublicKey(key *PublicKey) error
	// GetAccountContractNames returns the names of all contracts deployed in an account.
	GetAccountContractNames(address Address) ([]string, error)
	// RecordTrace records an opentelemetry trace.
	RecordTrace(operation string, location Location, duration time.Duration, attrs []attribute.KeyValue)
	// BLSVerifyPOP verifies a proof of possession (PoP) for the receiver public key.
	BLSVerifyPOP(publicKey *PublicKey, signature []byte) (bool, error)
	// BLSAggregateSignatures aggregate multiple BLS signatures into one.
	BLSAggregateSignatures(signatures [][]byte) ([]byte, error)
	// BLSAggregatePublicKeys aggregate multiple BLS public keys into one.
	BLSAggregatePublicKeys(publicKeys []*PublicKey) (*PublicKey, error)
	// ResourceOwnerChanged gets called when a resource's owner changed (if enabled)
	ResourceOwnerChanged(
		interpreter *interpreter.Interpreter,
		resource *interpreter.CompositeValue,
		oldOwner common.Address,
		newOwner common.Address,
	)
	// GenerateAccountID generates a new, *non-zero*, unique ID for the given account.
	GenerateAccountID(address common.Address) (uint64, error)
	RecoverProgram(program *ast.Program, location common.Location) ([]byte, error)
	ValidateAccountCapabilitiesGet(
		context interpreter.AccountCapabilityGetValidationContext,
		locationRange interpreter.LocationRange,
		address interpreter.AddressValue,
		path interpreter.PathValue,
		wantedBorrowType *sema.ReferenceType,
		capabilityBorrowType *sema.ReferenceType,
	) (bool, error)
	ValidateAccountCapabilitiesPublish(
		context interpreter.AccountCapabilityPublishValidationContext,
		locationRange interpreter.LocationRange,
		address interpreter.AddressValue,
		path interpreter.PathValue,
		capabilityBorrowType *interpreter.ReferenceStaticType,
	) (bool, error)

	MinimumRequiredVersion() (string, error)
}

type MeterInterface interface {
	// MeterMemory gets called when new memory is allocated or used by the interpreter
	MeterMemory(usage common.MemoryUsage) error
	// MeterComputation is a callback method for metering computation, it returns error
	// when computation passes the limit (set by the environment)
	MeterComputation(usage common.ComputationUsage) error
	// ComputationUsed returns the total computation used in the current runtime.
	ComputationUsed() (uint64, error)
	// MemoryUsed returns the total memory (estimate) used in the current runtime.
	MemoryUsed() (uint64, error)
	// InteractionUsed returns the total storage interaction used in the current runtime.
	InteractionUsed() (uint64, error)
}

type Metrics interface {
	ProgramParsed(location Location, duration time.Duration)
	ProgramChecked(location Location, duration time.Duration)
	ProgramInterpreted(location Location, duration time.Duration)
}
