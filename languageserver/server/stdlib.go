/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package server

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type standardLibrary struct {
	baseValueActivation *sema.VariableActivation
}

var _ stdlib.Logger = standardLibrary{}
var _ stdlib.UnsafeRandomGenerator = standardLibrary{}
var _ stdlib.BlockAtHeightProvider = standardLibrary{}
var _ stdlib.CurrentBlockProvider = standardLibrary{}
var _ stdlib.PublicAccountHandler = standardLibrary{}

func (standardLibrary) ProgramLog(_ string) error {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) UnsafeRandom() (uint64, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetBlockAtHeight(_ uint64) (stdlib.Block, bool, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetCurrentBlockHeight() (uint64, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetAccountBalance(_ common.Address) (uint64, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetAccountAvailableBalance(_ common.Address) (uint64, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) CommitStorageTemporarily(_ *interpreter.Interpreter) error {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetStorageUsed(_ common.Address) (uint64, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetStorageCapacity(_ common.Address) (uint64, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetAccountKey(_ common.Address, _ int) (*stdlib.AccountKey, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetAccountContractNames(_ common.Address) ([]string, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) GetAccountContractCode(_ common.Address, _ string) ([]byte, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) EmitEvent(
	_ *interpreter.Interpreter,
	_ *sema.CompositeType,
	_ []interpreter.Value,
	_ func() interpreter.LocationRange,
) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) AddEncodedAccountKey(_ common.Address, _ []byte) error {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) RevokeEncodedAccountKey(_ common.Address, _ int) ([]byte, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) AddAccountKey(
	_ common.Address,
	_ *stdlib.PublicKey,
	_ sema.HashAlgorithm,
	_ int,
) (
	*stdlib.AccountKey,
	error,
) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) RevokeAccountKey(_ common.Address, _ int) (*stdlib.AccountKey, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) ParseAndCheckProgram(_ []byte, _ common.Location, _ bool) (*interpreter.Program, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) UpdateAccountContractCode(_ common.Address, _ string, _ []byte) error {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) RecordContractUpdate(_ common.Address, _ string, _ *interpreter.CompositeValue) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) InterpretContract(
	_ common.AddressLocation,
	_ *interpreter.Program,
	_ string,
	_ stdlib.DeployedContractConstructorInvocation,
) (*interpreter.CompositeValue, error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) TemporarilyRecordCode(_ common.AddressLocation, _ []byte) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) RemoveAccountContractCode(_ common.Address, _ string) error {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) RecordContractRemoval(_ common.Address, _ string) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func (standardLibrary) CreateAccount(_ common.Address) (address common.Address, err error) {
	// Implementation should never be called,
	// only its definition is used for type-checking
	panic(errors.NewUnreachableError())
}

func newStandardLibrary() (result standardLibrary) {
	// TODO: either:
	//   - use runtime's script environment. requires it to become more configurable
	//   - separate out stdlib definitions from script environment, and use them here instead

	result.baseValueActivation = sema.NewVariableActivation(sema.BaseValueActivation)
	for _, valueDeclaration := range append(
		stdlib.BuiltinValues[:],
		stdlib.NewLogFunction(result),
		stdlib.NewUnsafeRandomFunction(result),
		stdlib.NewGetBlockFunction(result),
		stdlib.NewGetCurrentBlockFunction(result),
		stdlib.NewGetAccountFunction(result),
		stdlib.NewAuthAccountConstructor(result),
		stdlib.NewGetAuthAccountFunction(result),
	) {
		result.baseValueActivation.DeclareValue(valueDeclaration)
	}

	return
}
