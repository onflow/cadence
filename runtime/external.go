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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// ExternalInterface is an implementation of runtime.Interface which forwards all calls to the embedded Interface.
// All calls' panics and errors are wrapped.
type ExternalInterface struct {
	Interface Interface
}

var _ Interface = ExternalInterface{}
var _ Metrics = ExternalInterface{}

func (e ExternalInterface) MeterMemory(usage common.MemoryUsage) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.MeterMemory(usage)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) MeterComputation(usage common.ComputationUsage) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.MeterComputation(usage)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ComputationUsed() (usage uint64, err error) {
	errors.WrapPanic(func() {
		usage, err = e.Interface.ComputationUsed()
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) MemoryUsed() (usage uint64, err error) {
	errors.WrapPanic(func() {
		usage, err = e.Interface.MemoryUsed()
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) InteractionUsed() (usage uint64, err error) {
	errors.WrapPanic(func() {
		usage, err = e.Interface.InteractionUsed()
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ResolveLocation(
	identifiers []Identifier,
	location Location,
) (
	locations []ResolvedLocation,
	err error,
) {
	errors.WrapPanic(func() {
		locations, err = e.Interface.ResolveLocation(identifiers, location)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetCode(location Location) (code []byte, err error) {
	errors.WrapPanic(func() {
		code, err = e.Interface.GetCode(location)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetOrLoadProgram(
	location Location,
	load func() (*interpreter.Program, error),
) (
	program *interpreter.Program,
	err error,
) {
	errors.WrapPanic(func() {
		program, err = e.Interface.GetOrLoadProgram(location, load)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) SetInterpreterSharedState(state *interpreter.SharedState) {
	errors.WrapPanic(func() {
		e.Interface.SetInterpreterSharedState(state)
	})
	// No error to wrap
}

func (e ExternalInterface) GetInterpreterSharedState() (state *interpreter.SharedState) {
	errors.WrapPanic(func() {
		state = e.Interface.GetInterpreterSharedState()
	})
	// No error to wrap
	return
}

func (e ExternalInterface) GetValue(owner, key []byte) (value []byte, err error) {
	errors.WrapPanic(func() {
		value, err = e.Interface.GetValue(owner, key)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) SetValue(owner, key, value []byte) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.SetValue(owner, key, value)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ValueExists(owner, key []byte) (exists bool, err error) {
	errors.WrapPanic(func() {
		exists, err = e.Interface.ValueExists(owner, key)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) AllocateSlabIndex(owner []byte) (index atree.SlabIndex, err error) {
	errors.WrapPanic(func() {
		index, err = e.Interface.AllocateSlabIndex(owner)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) CreateAccount(payer Address) (address Address, err error) {
	errors.WrapPanic(func() {
		address, err = e.Interface.CreateAccount(payer)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) AddAccountKey(
	address Address,
	publicKey *PublicKey,
	hashAlgo HashAlgorithm,
	weight int,
) (
	key *AccountKey,
	err error,
) {
	errors.WrapPanic(func() {
		key, err = e.Interface.AddAccountKey(
			address,
			publicKey,
			hashAlgo,
			weight,
		)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetAccountKey(address Address, index uint32) (key *AccountKey, err error) {
	errors.WrapPanic(func() {
		key, err = e.Interface.GetAccountKey(address, index)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) AccountKeysCount(address Address) (count uint32, err error) {
	errors.WrapPanic(func() {
		count, err = e.Interface.AccountKeysCount(address)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) RevokeAccountKey(address Address, index uint32) (key *AccountKey, err error) {
	errors.WrapPanic(func() {
		key, err = e.Interface.RevokeAccountKey(address, index)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) UpdateAccountContractCode(location common.AddressLocation, code []byte) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.UpdateAccountContractCode(location, code)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetAccountContractCode(location common.AddressLocation) (code []byte, err error) {
	errors.WrapPanic(func() {
		code, err = e.Interface.GetAccountContractCode(location)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) RemoveAccountContractCode(location common.AddressLocation) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.RemoveAccountContractCode(location)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetSigningAccounts() (addresses []Address, err error) {
	errors.WrapPanic(func() {
		addresses, err = e.Interface.GetSigningAccounts()
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ProgramLog(message string) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.ProgramLog(message)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) EmitEvent(event cadence.Event) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.EmitEvent(event)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GenerateUUID() (uuid uint64, err error) {
	errors.WrapPanic(func() {
		uuid, err = e.Interface.GenerateUUID()
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) DecodeArgument(argument []byte, argumentType cadence.Type) (value cadence.Value, err error) {
	errors.WrapPanic(func() {
		value, err = e.Interface.DecodeArgument(argument, argumentType)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetCurrentBlockHeight() (height uint64, err error) {
	errors.WrapPanic(func() {
		height, err = e.Interface.GetCurrentBlockHeight()
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetBlockAtHeight(height uint64) (block Block, exists bool, err error) {
	errors.WrapPanic(func() {
		block, exists, err = e.Interface.GetBlockAtHeight(height)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ReadRandom(bytes []byte) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.ReadRandom(bytes)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) VerifySignature(
	signature []byte,
	tag string,
	signedData []byte,
	publicKey []byte,
	signatureAlgorithm SignatureAlgorithm,
	hashAlgorithm HashAlgorithm,
) (
	valid bool,
	err error,
) {
	errors.WrapPanic(func() {
		valid, err = e.Interface.VerifySignature(
			signature,
			tag,
			signedData,
			publicKey,
			signatureAlgorithm,
			hashAlgorithm,
		)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) Hash(data []byte, tag string, hashAlgorithm HashAlgorithm) (hash []byte, err error) {
	errors.WrapPanic(func() {
		hash, err = e.Interface.Hash(data, tag, hashAlgorithm)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetAccountBalance(address common.Address) (value uint64, err error) {
	errors.WrapPanic(func() {
		value, err = e.Interface.GetAccountBalance(address)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetAccountAvailableBalance(address common.Address) (value uint64, err error) {
	errors.WrapPanic(func() {
		value, err = e.Interface.GetAccountAvailableBalance(address)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetStorageUsed(address Address) (usage uint64, err error) {
	errors.WrapPanic(func() {
		usage, err = e.Interface.GetStorageUsed(address)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetStorageCapacity(address Address) (capacity uint64, err error) {
	errors.WrapPanic(func() {
		capacity, err = e.Interface.GetStorageCapacity(address)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ImplementationDebugLog(message string) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.ImplementationDebugLog(message)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ValidatePublicKey(key *PublicKey) (err error) {
	errors.WrapPanic(func() {
		err = e.Interface.ValidatePublicKey(key)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) GetAccountContractNames(address Address) (names []string, err error) {
	errors.WrapPanic(func() {
		names, err = e.Interface.GetAccountContractNames(address)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) RecordTrace(
	operation string,
	location Location,
	duration time.Duration,
	attrs []attribute.KeyValue,
) {
	errors.WrapPanic(func() {
		e.Interface.RecordTrace(
			operation,
			location,
			duration,
			attrs,
		)
	})
	// No error to wrap
}

func (e ExternalInterface) BLSVerifyPOP(publicKey *PublicKey, signature []byte) (valid bool, err error) {
	errors.WrapPanic(func() {
		valid, err = e.Interface.BLSVerifyPOP(publicKey, signature)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) BLSAggregateSignatures(signatures [][]byte) (signature []byte, err error) {
	errors.WrapPanic(func() {
		signature, err = e.Interface.BLSAggregateSignatures(signatures)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) BLSAggregatePublicKeys(publicKeys []*PublicKey) (publicKey *PublicKey, err error) {
	errors.WrapPanic(func() {
		publicKey, err = e.Interface.BLSAggregatePublicKeys(publicKeys)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ResourceOwnerChanged(
	interpreter *interpreter.Interpreter,
	resource *interpreter.CompositeValue,
	oldOwner common.Address,
	newOwner common.Address,
) {
	errors.WrapPanic(func() {
		e.Interface.ResourceOwnerChanged(interpreter, resource, oldOwner, newOwner)
	})
	// No error to wrap
}

func (e ExternalInterface) GenerateAccountID(address common.Address) (id uint64, err error) {
	errors.WrapPanic(func() {
		id, err = e.Interface.GenerateAccountID(address)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) RecoverProgram(program *ast.Program, location common.Location) (code []byte, err error) {
	errors.WrapPanic(func() {
		code, err = e.Interface.RecoverProgram(program, location)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ValidateAccountCapabilitiesGet(
	context interpreter.AccountCapabilityGetValidationContext,
	locationRange interpreter.LocationRange,
	address interpreter.AddressValue,
	path interpreter.PathValue,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) (
	valid bool,
	err error,
) {
	errors.WrapPanic(func() {
		valid, err = e.Interface.ValidateAccountCapabilitiesGet(
			context,
			locationRange,
			address,
			path,
			wantedBorrowType,
			capabilityBorrowType,
		)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ValidateAccountCapabilitiesPublish(
	context interpreter.AccountCapabilityPublishValidationContext,
	locationRange interpreter.LocationRange,
	address interpreter.AddressValue,
	path interpreter.PathValue,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) (
	ok bool,
	err error,
) {
	errors.WrapPanic(func() {
		ok, err = e.Interface.ValidateAccountCapabilitiesPublish(
			context,
			locationRange,
			address,
			path,
			capabilityBorrowType,
		)
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) MinimumRequiredVersion() (version string, err error) {
	errors.WrapPanic(func() {
		version, err = e.Interface.MinimumRequiredVersion()
	})
	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}
	return
}

func (e ExternalInterface) ProgramParsed(location Location, duration time.Duration) {
	metrics, ok := e.Interface.(Metrics)
	if !ok {
		return
	}
	errors.WrapPanic(func() {
		metrics.ProgramParsed(location, duration)
	})
	// No error to wrap
}

func (e ExternalInterface) ProgramChecked(location Location, duration time.Duration) {
	metrics, ok := e.Interface.(Metrics)
	if !ok {
		return
	}
	errors.WrapPanic(func() {
		metrics.ProgramChecked(location, duration)
	})
	// No error to wrap
}

func (e ExternalInterface) ProgramInterpreted(location Location, duration time.Duration) {
	metrics, ok := e.Interface.(Metrics)
	if !ok {
		return
	}
	errors.WrapPanic(func() {
		metrics.ProgramInterpreted(location, duration)
	})
	// No error to wrap
}
