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

package main

import (
	"time"

	"github.com/onflow/atree"
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/flow-go/cmd/util/ledger/util"
	"github.com/onflow/flow-go/model/flow"
	"go.opentelemetry.io/otel/attribute"
)

// ReadOnlyRuntimeInterface is a runtime interface that can be used in migrations.
type ReadOnlyRuntimeInterface struct {
	*util.PayloadSnapshot
}

func (ReadOnlyRuntimeInterface) ResolveLocation(
	_ []runtime.Identifier,
	_ runtime.Location,
) ([]runtime.ResolvedLocation, error) {
	panic("unexpected ResolveLocation call")
}

func (ReadOnlyRuntimeInterface) GetCode(_ runtime.Location) ([]byte, error) {
	panic("unexpected GetCode call")
}

func (ReadOnlyRuntimeInterface) GetAccountContractCode(_ common.AddressLocation) ([]byte, error) {
	panic("unexpected GetAccountContractCode call")
}

func (ReadOnlyRuntimeInterface) GetOrLoadProgram(
	_ runtime.Location,
	_ func() (*interpreter.Program, error),
) (*interpreter.Program, error) {
	panic("unexpected GetOrLoadProgram call")
}

func (ReadOnlyRuntimeInterface) MeterMemory(_ common.MemoryUsage) error {
	return nil
}

func (ReadOnlyRuntimeInterface) MeterComputation(_ common.ComputationKind, _ uint) error {
	return nil
}

func (i ReadOnlyRuntimeInterface) GetValue(owner, key []byte) (value []byte, err error) {
	registerID := flow.NewRegisterID(flow.Address(owner), string(key))
	return i.PayloadSnapshot.Get(registerID)
}

func (ReadOnlyRuntimeInterface) SetValue(_, _, _ []byte) (err error) {
	panic("unexpected SetValue call")
}

func (ReadOnlyRuntimeInterface) CreateAccount(_ runtime.Address) (address runtime.Address, err error) {
	panic("unexpected CreateAccount call")
}

func (ReadOnlyRuntimeInterface) AddEncodedAccountKey(_ runtime.Address, _ []byte) error {
	panic("unexpected AddEncodedAccountKey call")
}

func (ReadOnlyRuntimeInterface) RevokeEncodedAccountKey(_ runtime.Address, _ int) (publicKey []byte, err error) {
	panic("unexpected RevokeEncodedAccountKey call")
}

func (ReadOnlyRuntimeInterface) AddAccountKey(
	_ runtime.Address,
	_ *runtime.PublicKey,
	_ runtime.HashAlgorithm,
	_ int,
) (*runtime.AccountKey, error) {
	panic("unexpected AddAccountKey call")
}

func (ReadOnlyRuntimeInterface) GetAccountKey(_ runtime.Address, _ int) (*runtime.AccountKey, error) {
	panic("unexpected GetAccountKey call")
}

func (ReadOnlyRuntimeInterface) RevokeAccountKey(_ runtime.Address, _ int) (*runtime.AccountKey, error) {
	panic("unexpected RevokeAccountKey call")
}

func (ReadOnlyRuntimeInterface) UpdateAccountContractCode(_ common.AddressLocation, _ []byte) (err error) {
	panic("unexpected UpdateAccountContractCode call")
}

func (ReadOnlyRuntimeInterface) RemoveAccountContractCode(common.AddressLocation) (err error) {
	panic("unexpected RemoveAccountContractCode call")
}

func (ReadOnlyRuntimeInterface) GetSigningAccounts() ([]runtime.Address, error) {
	panic("unexpected GetSigningAccounts call")
}

func (ReadOnlyRuntimeInterface) ProgramLog(_ string) error {
	panic("unexpected ProgramLog call")
}

func (ReadOnlyRuntimeInterface) EmitEvent(_ cadence.Event) error {
	panic("unexpected EmitEvent call")
}

func (i ReadOnlyRuntimeInterface) ValueExists(owner, key []byte) (exists bool, err error) {
	registerID := flow.NewRegisterID(flow.Address(owner), string(key))
	_, exists = i.Payloads[registerID]
	return
}

func (ReadOnlyRuntimeInterface) GenerateUUID() (uint64, error) {
	panic("unexpected GenerateUUID call")
}

func (ReadOnlyRuntimeInterface) GetComputationLimit() uint64 {
	panic("unexpected GetComputationLimit call")
}

func (ReadOnlyRuntimeInterface) SetComputationUsed(_ uint64) error {
	panic("unexpected SetComputationUsed call")
}

func (ReadOnlyRuntimeInterface) DecodeArgument(_ []byte, _ cadence.Type) (cadence.Value, error) {
	panic("unexpected DecodeArgument call")
}

func (ReadOnlyRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	panic("unexpected GetCurrentBlockHeight call")
}

func (ReadOnlyRuntimeInterface) GetBlockAtHeight(_ uint64) (block runtime.Block, exists bool, err error) {
	panic("unexpected GetBlockAtHeight call")
}

func (ReadOnlyRuntimeInterface) ReadRandom([]byte) error {
	panic("unexpected ReadRandom call")
}

func (ReadOnlyRuntimeInterface) VerifySignature(
	_ []byte,
	_ string,
	_ []byte,
	_ []byte,
	_ runtime.SignatureAlgorithm,
	_ runtime.HashAlgorithm,
) (bool, error) {
	panic("unexpected VerifySignature call")
}

func (ReadOnlyRuntimeInterface) Hash(_ []byte, _ string, _ runtime.HashAlgorithm) ([]byte, error) {
	panic("unexpected Hash call")
}

func (ReadOnlyRuntimeInterface) GetAccountBalance(_ common.Address) (value uint64, err error) {
	panic("unexpected GetAccountBalance call")
}

func (ReadOnlyRuntimeInterface) GetAccountAvailableBalance(_ common.Address) (value uint64, err error) {
	panic("unexpected GetAccountAvailableBalance call")
}

func (ReadOnlyRuntimeInterface) GetStorageUsed(_ runtime.Address) (value uint64, err error) {
	panic("unexpected GetStorageUsed call")
}

func (ReadOnlyRuntimeInterface) GetStorageCapacity(_ runtime.Address) (value uint64, err error) {
	panic("unexpected GetStorageCapacity call")
}

func (ReadOnlyRuntimeInterface) ImplementationDebugLog(_ string) error {
	panic("unexpected ImplementationDebugLog call")
}

func (ReadOnlyRuntimeInterface) ValidatePublicKey(_ *runtime.PublicKey) error {
	panic("unexpected ValidatePublicKey call")
}

func (ReadOnlyRuntimeInterface) GetAccountContractNames(_ runtime.Address) ([]string, error) {
	panic("unexpected GetAccountContractNames call")
}

func (ReadOnlyRuntimeInterface) AllocateStorageIndex(_ []byte) (atree.StorageIndex, error) {
	panic("unexpected AllocateStorageIndex call")
}

func (ReadOnlyRuntimeInterface) ComputationUsed() (uint64, error) {
	panic("unexpected ComputationUsed call")
}

func (ReadOnlyRuntimeInterface) MemoryUsed() (uint64, error) {
	panic("unexpected MemoryUsed call")
}

func (ReadOnlyRuntimeInterface) InteractionUsed() (uint64, error) {
	panic("unexpected InteractionUsed call")
}

func (ReadOnlyRuntimeInterface) SetInterpreterSharedState(_ *interpreter.SharedState) {
	// NO-OP
}

func (ReadOnlyRuntimeInterface) GetInterpreterSharedState() *interpreter.SharedState {
	return nil
}

func (ReadOnlyRuntimeInterface) AccountKeysCount(_ runtime.Address) (uint64, error) {
	panic("unexpected AccountKeysCount call")
}

func (ReadOnlyRuntimeInterface) BLSVerifyPOP(_ *runtime.PublicKey, _ []byte) (bool, error) {
	panic("unexpected BLSVerifyPOP call")
}

func (ReadOnlyRuntimeInterface) BLSAggregateSignatures(_ [][]byte) ([]byte, error) {
	panic("unexpected BLSAggregateSignatures call")
}

func (ReadOnlyRuntimeInterface) BLSAggregatePublicKeys(_ []*runtime.PublicKey) (*runtime.PublicKey, error) {
	panic("unexpected BLSAggregatePublicKeys call")
}

func (ReadOnlyRuntimeInterface) ResourceOwnerChanged(
	_ *interpreter.Interpreter,
	_ *interpreter.CompositeValue,
	_ common.Address,
	_ common.Address,
) {
	panic("unexpected ResourceOwnerChanged call")
}

func (ReadOnlyRuntimeInterface) GenerateAccountID(_ common.Address) (uint64, error) {
	panic("unexpected GenerateAccountID call")
}

func (ReadOnlyRuntimeInterface) RecordTrace(_ string, _ runtime.Location, _ time.Duration, _ []attribute.KeyValue) {
	panic("unexpected RecordTrace call")
}
