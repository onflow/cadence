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

package runtime_utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"time"

	"github.com/onflow/atree"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type TestRuntimeInterface struct {
	Storage TestLedger

	OnResolveLocation func(
		identifiers []runtime.Identifier,
		location runtime.Location,
	) (
		[]runtime.ResolvedLocation,
		error,
	)
	OnGetCode          func(_ runtime.Location) ([]byte, error)
	OnGetAndSetProgram func(
		location runtime.Location,
		load func() (*interpreter.Program, error),
	) (*interpreter.Program, error)
	OnSetInterpreterSharedState func(state *interpreter.SharedState)
	OnGetInterpreterSharedState func() *interpreter.SharedState
	OnCreateAccount             func(payer runtime.Address) (address runtime.Address, err error)
	OnAddEncodedAccountKey      func(address runtime.Address, publicKey []byte) error
	OnRemoveEncodedAccountKey   func(address runtime.Address, index int) (publicKey []byte, err error)
	OnAddAccountKey             func(
		address runtime.Address,
		publicKey *stdlib.PublicKey,
		hashAlgo runtime.HashAlgorithm,
		weight int,
	) (*stdlib.AccountKey, error)
	OnGetAccountKey             func(address runtime.Address, index uint32) (*stdlib.AccountKey, error)
	OnRemoveAccountKey          func(address runtime.Address, index uint32) (*stdlib.AccountKey, error)
	OnAccountKeysCount          func(address runtime.Address) (uint32, error)
	OnUpdateAccountContractCode func(location common.AddressLocation, code []byte) error
	OnGetAccountContractCode    func(location common.AddressLocation) (code []byte, err error)
	OnRemoveAccountContractCode func(location common.AddressLocation) (err error)
	OnGetSigningAccounts        func() ([]runtime.Address, error)
	OnProgramLog                func(string)
	OnEmitEvent                 func(cadence.Event) error
	OnResourceOwnerChanged      func(
		interpreter *interpreter.Interpreter,
		resource *interpreter.CompositeValue,
		oldAddress common.Address,
		newAddress common.Address,
	)
	OnGenerateUUID       func() (uint64, error)
	OnMeterComputation   func(compKind common.ComputationKind, intensity uint) error
	OnDecodeArgument     func(b []byte, t cadence.Type) (cadence.Value, error)
	OnProgramParsed      func(location runtime.Location, duration time.Duration)
	OnProgramChecked     func(location runtime.Location, duration time.Duration)
	OnProgramInterpreted func(location runtime.Location, duration time.Duration)
	OnReadRandom         func([]byte) error
	OnVerifySignature    func(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm runtime.SignatureAlgorithm,
		hashAlgorithm runtime.HashAlgorithm,
	) (bool, error)
	OnHash func(
		data []byte,
		tag string,
		hashAlgorithm runtime.HashAlgorithm,
	) ([]byte, error)
	OnSetCadenceValue            func(owner runtime.Address, key string, value cadence.Value) (err error)
	OnGetAccountBalance          func(_ runtime.Address) (uint64, error)
	OnGetAccountAvailableBalance func(_ runtime.Address) (uint64, error)
	OnGetStorageUsed             func(_ runtime.Address) (uint64, error)
	OnGetStorageCapacity         func(_ runtime.Address) (uint64, error)
	Programs                     map[runtime.Location]*interpreter.Program
	OnImplementationDebugLog     func(message string) error
	OnValidatePublicKey          func(publicKey *stdlib.PublicKey) error
	OnBLSVerifyPOP               func(pk *stdlib.PublicKey, s []byte) (bool, error)
	OnBLSAggregateSignatures     func(sigs [][]byte) ([]byte, error)
	OnBLSAggregatePublicKeys     func(keys []*stdlib.PublicKey) (*stdlib.PublicKey, error)
	OnGetAccountContractNames    func(address runtime.Address) ([]string, error)
	OnRecordTrace                func(
		operation string,
		location runtime.Location,
		duration time.Duration,
		attrs []attribute.KeyValue,
	)
	OnMeterMemory                    func(usage common.MemoryUsage) error
	OnComputationUsed                func() (uint64, error)
	OnMemoryUsed                     func() (uint64, error)
	OnInteractionUsed                func() (uint64, error)
	OnGenerateAccountID              func(address common.Address) (uint64, error)
	OnRecoverProgram                 func(program *ast.Program, location common.Location) ([]byte, error)
	OnValidateAccountCapabilitiesGet func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		address interpreter.AddressValue,
		path interpreter.PathValue,
		wantedBorrowType *sema.ReferenceType,
		capabilityBorrowType *sema.ReferenceType,
	) (bool, error)
	OnValidateAccountCapabilitiesPublish func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		address interpreter.AddressValue,
		path interpreter.PathValue,
		capabilityBorrowType *interpreter.ReferenceStaticType,
	) (bool, error)
	OnCompileWebAssembly func(bytes []byte) (stdlib.WebAssemblyModule, error)

	lastUUID            uint64
	accountIDs          map[common.Address]uint64
	updatedContractCode bool
}

// TestRuntimeInterface should implement Interface
var _ runtime.Interface = &TestRuntimeInterface{}

func (i *TestRuntimeInterface) ResolveLocation(
	identifiers []runtime.Identifier,
	location runtime.Location,
) ([]runtime.ResolvedLocation, error) {
	if i.OnResolveLocation == nil {
		return []runtime.ResolvedLocation{
			{
				Location:    location,
				Identifiers: identifiers,
			},
		}, nil
	}
	return i.OnResolveLocation(identifiers, location)
}

func (i *TestRuntimeInterface) GetCode(location runtime.Location) ([]byte, error) {
	if i.OnGetCode == nil {
		return nil, nil
	}
	return i.OnGetCode(location)
}

func (i *TestRuntimeInterface) GetOrLoadProgram(
	location runtime.Location,
	load func() (*interpreter.Program, error),
) (
	program *interpreter.Program,
	err error,
) {
	if i.OnGetAndSetProgram == nil {
		if i.Programs == nil {
			i.Programs = map[runtime.Location]*interpreter.Program{}
		}

		var ok bool
		program, ok = i.Programs[location]
		if ok {
			return
		}

		program, err = load()

		// NOTE: important: still set empty program,
		// even if error occurred

		i.Programs[location] = program

		return
	}

	return i.OnGetAndSetProgram(location, load)
}

func (i *TestRuntimeInterface) SetInterpreterSharedState(state *interpreter.SharedState) {
	if i.OnSetInterpreterSharedState == nil {
		return
	}

	i.OnSetInterpreterSharedState(state)
}

func (i *TestRuntimeInterface) GetInterpreterSharedState() *interpreter.SharedState {
	if i.OnGetInterpreterSharedState == nil {
		return nil
	}

	return i.OnGetInterpreterSharedState()
}

func (i *TestRuntimeInterface) ValueExists(owner, key []byte) (exists bool, err error) {
	if i.Storage.OnValueExists == nil {
		panic("must specify TestRuntimeInterface.Storage.OnValueExists")
	}
	return i.Storage.ValueExists(owner, key)
}

func (i *TestRuntimeInterface) GetValue(owner, key []byte) (value []byte, err error) {
	if i.Storage.OnGetValue == nil {
		panic("must specify TestRuntimeInterface.Storage.OnGetValue")
	}
	return i.Storage.GetValue(owner, key)
}

func (i *TestRuntimeInterface) SetValue(owner, key, value []byte) (err error) {
	if i.Storage.OnSetValue == nil {
		panic("must specify TestRuntimeInterface.Storage.SetValue")
	}
	return i.Storage.SetValue(owner, key, value)
}

func (i *TestRuntimeInterface) AllocateSlabIndex(owner []byte) (atree.SlabIndex, error) {
	if i.Storage.OnAllocateSlabIndex == nil {
		panic("must specify TestRuntimeInterface.storage.OnAllocateSlabIndex")
	}
	return i.Storage.AllocateSlabIndex(owner)
}

func (i *TestRuntimeInterface) CreateAccount(payer runtime.Address) (address runtime.Address, err error) {
	if i.OnCreateAccount == nil {
		panic("must specify TestRuntimeInterface.OnCreateAccount")
	}
	return i.OnCreateAccount(payer)
}

func (i *TestRuntimeInterface) AddEncodedAccountKey(address runtime.Address, publicKey []byte) error {
	if i.OnAddEncodedAccountKey == nil {
		panic("must specify TestRuntimeInterface.OnAddEncodedAccountKey")
	}
	return i.OnAddEncodedAccountKey(address, publicKey)
}

func (i *TestRuntimeInterface) RevokeEncodedAccountKey(address runtime.Address, index int) ([]byte, error) {
	if i.OnRemoveEncodedAccountKey == nil {
		panic("must specify TestRuntimeInterface.OnRemoveEncodedAccountKey")
	}
	return i.OnRemoveEncodedAccountKey(address, index)
}

func (i *TestRuntimeInterface) AddAccountKey(
	address runtime.Address,
	publicKey *stdlib.PublicKey,
	hashAlgo runtime.HashAlgorithm,
	weight int,
) (*stdlib.AccountKey, error) {
	if i.OnAddAccountKey == nil {
		panic("must specify TestRuntimeInterface.OnAddAccountKey")
	}
	return i.OnAddAccountKey(address, publicKey, hashAlgo, weight)
}

func (i *TestRuntimeInterface) GetAccountKey(address runtime.Address, index uint32) (*stdlib.AccountKey, error) {
	if i.OnGetAccountKey == nil {
		panic("must specify TestRuntimeInterface.OnGetAccountKey")
	}
	return i.OnGetAccountKey(address, index)
}

func (i *TestRuntimeInterface) AccountKeysCount(address runtime.Address) (uint32, error) {
	if i.OnAccountKeysCount == nil {
		panic("must specify TestRuntimeInterface.OnAccountKeysCount")
	}
	return i.OnAccountKeysCount(address)
}

func (i *TestRuntimeInterface) RevokeAccountKey(address runtime.Address, index uint32) (*stdlib.AccountKey, error) {
	if i.OnRemoveAccountKey == nil {
		panic("must specify TestRuntimeInterface.OnRemoveAccountKey")
	}
	return i.OnRemoveAccountKey(address, index)
}

func (i *TestRuntimeInterface) UpdateAccountContractCode(location common.AddressLocation, code []byte) (err error) {
	if i.OnUpdateAccountContractCode == nil {
		panic("must specify TestRuntimeInterface.OnUpdateAccountContractCode")
	}

	err = i.OnUpdateAccountContractCode(location, code)
	if err != nil {
		return err
	}

	i.updatedContractCode = true

	return nil
}

func (i *TestRuntimeInterface) GetAccountContractCode(location common.AddressLocation) (code []byte, err error) {
	if i.OnGetAccountContractCode == nil {
		panic("must specify TestRuntimeInterface.OnGetAccountContractCode")
	}
	return i.OnGetAccountContractCode(location)
}

func (i *TestRuntimeInterface) RemoveAccountContractCode(location common.AddressLocation) (err error) {
	if i.OnRemoveAccountContractCode == nil {
		panic("must specify TestRuntimeInterface.OnRemoveAccountContractCode")
	}
	return i.OnRemoveAccountContractCode(location)
}

func (i *TestRuntimeInterface) GetSigningAccounts() ([]runtime.Address, error) {
	if i.OnGetSigningAccounts == nil {
		return nil, nil
	}
	return i.OnGetSigningAccounts()
}

func (i *TestRuntimeInterface) ProgramLog(message string) error {
	if i.OnProgramLog == nil {
		panic("must specify TestRuntimeInterface.OnProgramLog")
	}
	i.OnProgramLog(message)
	return nil
}

func (i *TestRuntimeInterface) EmitEvent(event cadence.Event) error {
	if i.OnEmitEvent == nil {
		panic("must specify TestRuntimeInterface.OnEmitEvent")
	}
	return i.OnEmitEvent(event)
}

func (i *TestRuntimeInterface) ResourceOwnerChanged(
	interpreter *interpreter.Interpreter,
	resource *interpreter.CompositeValue,
	oldOwner common.Address,
	newOwner common.Address,
) {
	if i.OnResourceOwnerChanged != nil {
		i.OnResourceOwnerChanged(
			interpreter,
			resource,
			oldOwner,
			newOwner,
		)
	}
}

func (i *TestRuntimeInterface) GenerateUUID() (uint64, error) {
	if i.OnGenerateUUID == nil {
		i.lastUUID++
		return i.lastUUID, nil
	}
	return i.OnGenerateUUID()
}

func (i *TestRuntimeInterface) MeterComputation(compKind common.ComputationKind, intensity uint) error {
	if i.OnMeterComputation == nil {
		return nil
	}
	return i.OnMeterComputation(compKind, intensity)
}

func (i *TestRuntimeInterface) DecodeArgument(b []byte, t cadence.Type) (cadence.Value, error) {
	if i.OnDecodeArgument == nil {
		panic("must specify TestRuntimeInterface.OnDecodeArgument")
	}
	return i.OnDecodeArgument(b, t)
}

func (i *TestRuntimeInterface) ProgramParsed(location runtime.Location, duration time.Duration) {
	if i.OnProgramParsed == nil {
		return
	}
	i.OnProgramParsed(location, duration)
}

func (i *TestRuntimeInterface) ProgramChecked(location runtime.Location, duration time.Duration) {
	if i.OnProgramChecked == nil {
		return
	}
	i.OnProgramChecked(location, duration)
}

func (i *TestRuntimeInterface) ProgramInterpreted(location runtime.Location, duration time.Duration) {
	if i.OnProgramInterpreted == nil {
		return
	}
	i.OnProgramInterpreted(location, duration)
}

func (i *TestRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	return 1, nil
}

func (i *TestRuntimeInterface) GetBlockAtHeight(height uint64) (block stdlib.Block, exists bool, err error) {

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, height)
	if err != nil {
		panic(err)
	}

	encoded := buf.Bytes()
	var hash stdlib.BlockHash
	copy(hash[sema.BlockTypeIdFieldType.Size-int64(len(encoded)):], encoded)

	block = stdlib.Block{
		Height:    height,
		View:      height,
		Hash:      hash,
		Timestamp: time.Unix(int64(height), 0).UnixNano(),
	}
	return block, true, nil
}

func (i *TestRuntimeInterface) ReadRandom(buffer []byte) error {
	if i.OnReadRandom == nil {
		return nil
	}
	return i.OnReadRandom(buffer)
}

func (i *TestRuntimeInterface) VerifySignature(
	signature []byte,
	tag string,
	signedData []byte,
	publicKey []byte,
	signatureAlgorithm runtime.SignatureAlgorithm,
	hashAlgorithm runtime.HashAlgorithm,
) (bool, error) {
	if i.OnVerifySignature == nil {
		return false, nil
	}
	return i.OnVerifySignature(
		signature,
		tag,
		signedData,
		publicKey,
		signatureAlgorithm,
		hashAlgorithm,
	)
}

func (i *TestRuntimeInterface) Hash(data []byte, tag string, hashAlgorithm runtime.HashAlgorithm) ([]byte, error) {
	if i.OnHash == nil {
		return nil, nil
	}
	return i.OnHash(data, tag, hashAlgorithm)
}

func (i *TestRuntimeInterface) SetCadenceValue(owner common.Address, key string, value cadence.Value) (err error) {
	if i.OnSetCadenceValue == nil {
		panic("must specify TestRuntimeInterface.OnSetCadenceValue")
	}
	return i.OnSetCadenceValue(owner, key, value)
}

func (i *TestRuntimeInterface) GetAccountBalance(address runtime.Address) (uint64, error) {
	if i.OnGetAccountBalance == nil {
		panic("must specify TestRuntimeInterface.OnGetAccountBalance")
	}
	return i.OnGetAccountBalance(address)
}

func (i *TestRuntimeInterface) GetAccountAvailableBalance(address runtime.Address) (uint64, error) {
	if i.OnGetAccountAvailableBalance == nil {
		panic("must specify TestRuntimeInterface.OnGetAccountAvailableBalance")
	}
	return i.OnGetAccountAvailableBalance(address)
}

func (i *TestRuntimeInterface) GetStorageUsed(address runtime.Address) (uint64, error) {
	if i.OnGetStorageUsed == nil {
		panic("must specify TestRuntimeInterface.OnGetStorageUsed")
	}
	return i.OnGetStorageUsed(address)
}

func (i *TestRuntimeInterface) GetStorageCapacity(address runtime.Address) (uint64, error) {
	if i.OnGetStorageCapacity == nil {
		panic("must specify TestRuntimeInterface.OnGetStorageCapacity")
	}
	return i.OnGetStorageCapacity(address)
}

func (i *TestRuntimeInterface) ImplementationDebugLog(message string) error {
	if i.OnImplementationDebugLog == nil {
		return nil
	}
	return i.OnImplementationDebugLog(message)
}

func (i *TestRuntimeInterface) ValidatePublicKey(key *stdlib.PublicKey) error {
	if i.OnValidatePublicKey == nil {
		return errors.New("mock defaults to public key validation failure")
	}

	return i.OnValidatePublicKey(key)
}

func (i *TestRuntimeInterface) BLSVerifyPOP(key *stdlib.PublicKey, s []byte) (bool, error) {
	if i.OnBLSVerifyPOP == nil {
		return false, nil
	}

	return i.OnBLSVerifyPOP(key, s)
}

func (i *TestRuntimeInterface) BLSAggregateSignatures(sigs [][]byte) ([]byte, error) {
	if i.OnBLSAggregateSignatures == nil {
		return []byte{}, nil
	}

	return i.OnBLSAggregateSignatures(sigs)
}

func (i *TestRuntimeInterface) BLSAggregatePublicKeys(keys []*stdlib.PublicKey) (*stdlib.PublicKey, error) {
	if i.OnBLSAggregatePublicKeys == nil {
		return nil, nil
	}

	return i.OnBLSAggregatePublicKeys(keys)
}

func (i *TestRuntimeInterface) GetAccountContractNames(address runtime.Address) ([]string, error) {
	if i.OnGetAccountContractNames == nil {
		return []string{}, nil
	}

	return i.OnGetAccountContractNames(address)
}

func (i *TestRuntimeInterface) GenerateAccountID(address common.Address) (uint64, error) {
	if i.OnGenerateAccountID == nil {
		if i.accountIDs == nil {
			i.accountIDs = map[common.Address]uint64{}
		}
		i.accountIDs[address]++
		return i.accountIDs[address], nil
	}

	return i.OnGenerateAccountID(address)
}

func (i *TestRuntimeInterface) CompileWebAssembly(bytes []byte) (stdlib.WebAssemblyModule, error) {
	if i.OnCompileWebAssembly == nil {
		return nil, nil
	}

	return i.OnCompileWebAssembly(bytes)
}

func (i *TestRuntimeInterface) RecordTrace(
	operation string,
	location runtime.Location,
	duration time.Duration,
	attrs []attribute.KeyValue,
) {
	if i.OnRecordTrace == nil {
		return
	}
	i.OnRecordTrace(operation, location, duration, attrs)
}

func (i *TestRuntimeInterface) MeterMemory(usage common.MemoryUsage) error {
	if i.OnMeterMemory == nil {
		return nil
	}

	return i.OnMeterMemory(usage)
}

func (i *TestRuntimeInterface) ComputationUsed() (uint64, error) {
	if i.OnComputationUsed == nil {
		return 0, nil
	}

	return i.OnComputationUsed()
}

func (i *TestRuntimeInterface) MemoryUsed() (uint64, error) {
	if i.OnMemoryUsed == nil {
		return 0, nil
	}

	return i.OnMemoryUsed()
}

func (i *TestRuntimeInterface) InteractionUsed() (uint64, error) {
	if i.OnInteractionUsed == nil {
		return 0, nil
	}

	return i.OnInteractionUsed()
}

func (i *TestRuntimeInterface) ComputationRemaining(kind common.ComputationKind) uint {
	if i.OnComputationRemaining == nil {
		return math.MaxUint
	}
	return i.OnComputationRemaining(kind)
}

func (i *TestRuntimeInterface) onTransactionExecutionStart() {
	i.InvalidateUpdatedPrograms()
}

func (i *TestRuntimeInterface) onScriptExecutionStart() {
	i.InvalidateUpdatedPrograms()
}

func (i *TestRuntimeInterface) InvalidateUpdatedPrograms() {
	if i.updatedContractCode {
		// iteration order does not matter
		for location := range i.Programs { //nolint:maprange
			delete(i.Programs, location)
		}
		i.updatedContractCode = false
	}
}

func (i *TestRuntimeInterface) RecoverProgram(program *ast.Program, location common.Location) ([]byte, error) {
	if i.OnRecoverProgram == nil {
		return nil, nil
	}
	return i.OnRecoverProgram(program, location)
}

func (i *TestRuntimeInterface) ValidateAccountCapabilitiesGet(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	address interpreter.AddressValue,
	path interpreter.PathValue,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) (bool, error) {
	if i.OnValidateAccountCapabilitiesGet == nil {
		return true, nil
	}
	return i.OnValidateAccountCapabilitiesGet(
		inter,
		locationRange,
		address,
		path,
		wantedBorrowType,
		capabilityBorrowType,
	)
}

func (i *TestRuntimeInterface) ValidateAccountCapabilitiesPublish(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	address interpreter.AddressValue,
	path interpreter.PathValue,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) (bool, error) {
	if i.OnValidateAccountCapabilitiesPublish == nil {
		return true, nil
	}
	return i.OnValidateAccountCapabilitiesPublish(
		inter,
		locationRange,
		address,
		path,
		capabilityBorrowType,
	)
}
