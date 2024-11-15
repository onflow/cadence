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
	"github.com/onflow/cadence/stdlib"
)

// EmptyRuntimeInterface is an empty implementation of runtime.Interface.
// It can be embedded in other types implementing runtime.Interface to avoid having to implement all methods.
type EmptyRuntimeInterface struct{}

var _ Interface = EmptyRuntimeInterface{}

func (EmptyRuntimeInterface) MeterMemory(_ common.MemoryUsage) error {
	// NO-OP
	return nil
}

func (EmptyRuntimeInterface) ResolveLocation(_ []Identifier, _ Location) ([]ResolvedLocation, error) {
	panic("unexpected call to ResolveLocation")
}

func (EmptyRuntimeInterface) GetOrLoadProgram(_ Location, _ func() (*interpreter.Program, error)) (*interpreter.Program, error) {
	panic("unexpected call to GetOrLoadProgram")
}

func (EmptyRuntimeInterface) GetAccountContractCode(_ common.AddressLocation) (code []byte, err error) {
	panic("unexpected call to GetAccountContractCode")
}

func (EmptyRuntimeInterface) MeterComputation(_ common.ComputationKind, _ uint) error {
	// NO-OP
	return nil
}

func (EmptyRuntimeInterface) ComputationUsed() (uint64, error) {
	panic("unexpected call to ComputationUsed")
}

func (i EmptyRuntimeInterface) ComputationRemaining(_ common.ComputationKind) uint {
	panic("unexpected call to ComputationRemaining")
}

func (EmptyRuntimeInterface) MemoryUsed() (uint64, error) {
	panic("unexpected call to MemoryUsed")
}

func (EmptyRuntimeInterface) InteractionUsed() (uint64, error) {
	panic("unexpected call to InteractionUsed")
}

func (EmptyRuntimeInterface) GetCode(_ Location) ([]byte, error) {
	panic("unexpected call to GetCode")
}

func (EmptyRuntimeInterface) SetInterpreterSharedState(_ *interpreter.SharedState) {
	panic("unexpected call to SetInterpreterSharedState")
}

func (EmptyRuntimeInterface) GetInterpreterSharedState() *interpreter.SharedState {
	panic("unexpected call to GetInterpreterSharedState")
}

func (EmptyRuntimeInterface) GetValue(_, _ []byte) (value []byte, err error) {
	panic("unexpected call to GetValue")
}

func (EmptyRuntimeInterface) SetValue(_, _, _ []byte) (err error) {
	panic("unexpected call to SetValue")
}

func (EmptyRuntimeInterface) ValueExists(_, _ []byte) (exists bool, err error) {
	panic("unexpected call to ValueExists")
}

func (EmptyRuntimeInterface) AllocateSlabIndex(_ []byte) (atree.SlabIndex, error) {
	panic("unexpected call to AllocateSlabIndex")
}

func (EmptyRuntimeInterface) CreateAccount(_ Address) (address Address, err error) {
	panic("unexpected call to CreateAccount")
}

func (EmptyRuntimeInterface) AddAccountKey(
	_ Address,
	_ *PublicKey,
	_ HashAlgorithm,
	_ int,
) (*AccountKey, error) {
	panic("unexpected call to AddAccountKey")
}

func (EmptyRuntimeInterface) GetAccountKey(_ Address, _ uint32) (*AccountKey, error) {
	panic("unexpected call to GetAccountKey")
}

func (EmptyRuntimeInterface) AccountKeysCount(_ Address) (uint32, error) {
	panic("unexpected call to AccountKeysCount")
}

func (EmptyRuntimeInterface) RevokeAccountKey(_ Address, _ uint32) (*AccountKey, error) {
	panic("unexpected call to RevokeAccountKey")
}

func (EmptyRuntimeInterface) UpdateAccountContractCode(_ common.AddressLocation, _ []byte) (err error) {
	panic("unexpected call to UpdateAccountContractCode")
}

func (EmptyRuntimeInterface) RemoveAccountContractCode(_ common.AddressLocation) (err error) {
	panic("unexpected call to RemoveAccountContractCode")
}

func (EmptyRuntimeInterface) GetSigningAccounts() ([]Address, error) {
	panic("unexpected call to GetSigningAccounts")
}

func (EmptyRuntimeInterface) ProgramLog(_ string) error {
	panic("unexpected call to ProgramLog")
}

func (EmptyRuntimeInterface) EmitEvent(_ cadence.Event) error {
	panic("unexpected call to EmitEvent")
}

func (EmptyRuntimeInterface) GenerateUUID() (uint64, error) {
	panic("unexpected call to GenerateUUID")
}

func (EmptyRuntimeInterface) DecodeArgument(_ []byte, _ cadence.Type) (cadence.Value, error) {
	panic("unexpected call to DecodeArgument")
}

func (EmptyRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	panic("unexpected call to GetCurrentBlockHeight")
}

func (EmptyRuntimeInterface) GetBlockAtHeight(_ uint64) (block Block, exists bool, err error) {
	panic("unexpected call to GetBlockAtHeight")
}

func (EmptyRuntimeInterface) ReadRandom(_ []byte) error {
	panic("unexpected call to ReadRandom")
}

func (EmptyRuntimeInterface) VerifySignature(
	_ []byte,
	_ string,
	_ []byte,
	_ []byte,
	_ SignatureAlgorithm,
	_ HashAlgorithm,
) (bool, error) {
	panic("unexpected call to VerifySignature")
}

func (EmptyRuntimeInterface) Hash(_ []byte, _ string, _ HashAlgorithm) ([]byte, error) {
	panic("unexpected call to Hash")
}

func (EmptyRuntimeInterface) GetAccountBalance(_ common.Address) (value uint64, err error) {
	panic("unexpected call to GetAccountBalance")
}

func (EmptyRuntimeInterface) GetAccountAvailableBalance(_ common.Address) (value uint64, err error) {
	panic("unexpected call to GetAccountAvailableBalance")
}

func (EmptyRuntimeInterface) GetStorageUsed(_ Address) (value uint64, err error) {
	panic("unexpected call to GetStorageUsed")
}

func (EmptyRuntimeInterface) GetStorageCapacity(_ Address) (value uint64, err error) {
	panic("unexpected call to GetStorageCapacity")
}

func (EmptyRuntimeInterface) ImplementationDebugLog(_ string) error {
	panic("unexpected call to ImplementationDebugLog")
}

func (EmptyRuntimeInterface) ValidatePublicKey(_ *PublicKey) error {
	panic("unexpected call to ValidatePublicKey")
}

func (EmptyRuntimeInterface) GetAccountContractNames(_ Address) ([]string, error) {
	panic("unexpected call to GetAccountContractNames")
}

func (EmptyRuntimeInterface) RecordTrace(_ string, _ Location, _ time.Duration, _ []attribute.KeyValue) {
	panic("unexpected call to RecordTrace")
}

func (EmptyRuntimeInterface) BLSVerifyPOP(_ *PublicKey, _ []byte) (bool, error) {
	panic("unexpected call to BLSVerifyPOP")
}

func (EmptyRuntimeInterface) BLSAggregateSignatures(_ [][]byte) ([]byte, error) {
	panic("unexpected call to BLSAggregateSignatures")
}

func (EmptyRuntimeInterface) BLSAggregatePublicKeys(_ []*PublicKey) (*PublicKey, error) {
	panic("unexpected call to BLSAggregatePublicKeys")
}

func (EmptyRuntimeInterface) ResourceOwnerChanged(
	_ *interpreter.Interpreter,
	_ *interpreter.CompositeValue,
	_ common.Address,
	_ common.Address,
) {
	panic("unexpected call to ResourceOwnerChanged")
}

func (EmptyRuntimeInterface) GenerateAccountID(_ common.Address) (uint64, error) {
	panic("unexpected call to GenerateAccountID")
}

func (EmptyRuntimeInterface) RecoverProgram(_ *ast.Program, _ common.Location) ([]byte, error) {
	panic("unexpected call to RecoverProgram")
}

func (EmptyRuntimeInterface) ValidateAccountCapabilitiesGet(
	_ *interpreter.Interpreter,
	_ interpreter.LocationRange,
	_ interpreter.AddressValue,
	_ interpreter.PathValue,
	_ *sema.ReferenceType,
	_ *sema.ReferenceType,
) (bool, error) {
	panic("unexpected call to ValidateAccountCapabilitiesGet")
}

func (EmptyRuntimeInterface) ValidateAccountCapabilitiesPublish(
	_ *interpreter.Interpreter,
	_ interpreter.LocationRange,
	_ interpreter.AddressValue,
	_ interpreter.PathValue,
	_ *interpreter.ReferenceStaticType,
) (bool, error) {
	panic("unexpected call to ValidateAccountCapabilitiesPublish")
}

func (EmptyRuntimeInterface) MinimumRequiredVersion() (string, error) {
	return "0.0.0", nil
}

func (EmptyRuntimeInterface) CompileWebAssembly(_ []byte) (stdlib.WebAssemblyModule, error) {
	panic("unexpected call to CompileWebAssembly")
}
