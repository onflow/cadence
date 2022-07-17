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

package runtime

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type Environment struct {
	baseActivation      *interpreter.VariableActivation
	baseValueActivation *sema.VariableActivation
	Interface           Interface
	Storage             *Storage
}

var _ stdlib.Logger = &Environment{}
var _ stdlib.UnsafeRandomGenerator = &Environment{}
var _ stdlib.BlockAtHeightProvider = &Environment{}
var _ stdlib.CurrentBlockProvider = &Environment{}
var _ stdlib.PublicAccountHandler = &Environment{}
var _ stdlib.AccountCreator = &Environment{}
var _ stdlib.EventEmitter = &Environment{}
var _ stdlib.AuthAccountHandler = &Environment{}

func newEnvironment() *Environment {
	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseActivation := interpreter.NewVariableActivation(nil, interpreter.BaseActivation)
	return &Environment{
		baseActivation:      baseActivation,
		baseValueActivation: baseValueActivation,
	}
}

func (e *Environment) Declare(valueDeclaration stdlib.StandardLibraryValue) {
	e.baseValueActivation.DeclareValue(valueDeclaration)
	e.baseActivation.Declare(valueDeclaration)
}

func NewBaseEnvironment(declarations ...stdlib.StandardLibraryValue) *Environment {
	env := newEnvironment()
	for _, valueDeclaration := range stdlib.BuiltinValues {
		env.Declare(valueDeclaration)
	}
	env.Declare(stdlib.NewLogFunction(env))
	env.Declare(stdlib.NewUnsafeRandomFunction(env))
	env.Declare(stdlib.NewGetBlockFunction(env))
	env.Declare(stdlib.NewGetCurrentBlockFunction(env))
	env.Declare(stdlib.NewGetAccountFunction(env))
	env.Declare(stdlib.NewAuthAccountConstructor(env))
	for _, declaration := range declarations {
		env.Declare(declaration)
	}
	return env
}

func NewScriptEnvironment(declarations ...stdlib.StandardLibraryValue) *Environment {
	env := NewBaseEnvironment(declarations...)
	env.Declare(stdlib.NewGetAuthAccountFunction(env))
	return env
}

func (e *Environment) ProgramLog(message string) error {
	return e.Interface.ProgramLog(message)
}

func (e *Environment) UnsafeRandom() (uint64, error) {
	return e.Interface.UnsafeRandom()
}

func (e *Environment) GetBlockAtHeight(height uint64) (block stdlib.Block, exists bool, err error) {
	return e.Interface.GetBlockAtHeight(height)
}

func (e *Environment) GetCurrentBlockHeight() (uint64, error) {
	return e.Interface.GetCurrentBlockHeight()
}

func (e *Environment) GetAccountBalance(address common.Address) (uint64, error) {
	return e.Interface.GetAccountBalance(address)
}

func (e *Environment) GetAccountAvailableBalance(address common.Address) (uint64, error) {
	return e.Interface.GetAccountAvailableBalance(address)
}

func (e *Environment) CommitStorage(inter *interpreter.Interpreter, commitContractUpdates bool) error {
	return e.Storage.Commit(inter, commitContractUpdates)
}

func (e *Environment) GetStorageUsed(address common.Address) (uint64, error) {
	return e.Interface.GetStorageUsed(address)
}

func (e *Environment) GetStorageCapacity(address common.Address) (uint64, error) {
	return e.Interface.GetStorageCapacity(address)
}

func (e *Environment) GetAccountKey(address common.Address, index int) (*stdlib.AccountKey, error) {
	return e.Interface.GetAccountKey(address, index)
}

func (e *Environment) GetAccountContractNames(address common.Address) ([]string, error) {
	return e.Interface.GetAccountContractNames(address)
}

func (e *Environment) GetAccountContractCode(address common.Address, name string) ([]byte, error) {
	return e.Interface.GetAccountContractCode(address, name)
}

func (e *Environment) CreateAccount(payer common.Address) (address common.Address, err error) {
	return e.Interface.CreateAccount(payer)
}

func (e *Environment) EmitEvent(
	inter *interpreter.Interpreter,
	eventType *sema.CompositeType,
	values []interpreter.Value,
	getLocationRange func() interpreter.LocationRange,
) {
	eventFields := make([]exportableValue, 0, len(values))

	for _, value := range values {
		eventFields = append(eventFields, newExportableValue(value, inter))
	}

	emitEventFields(
		inter,
		getLocationRange,
		eventType,
		eventFields,
		e.Interface.EmitEvent,
	)
}

func (e *Environment) AddEncodedAccountKey(address common.Address, key []byte) error {
	return e.Interface.AddEncodedAccountKey(address, key)
}

func (e *Environment) RevokeEncodedAccountKey(address common.Address, index int) ([]byte, error) {
	return e.Interface.RevokeEncodedAccountKey(address, index)
}

func (e *Environment) AddAccountKey(
	address common.Address,
	key *stdlib.PublicKey,
	algo sema.HashAlgorithm,
	weight int,
) (*stdlib.AccountKey, error) {
	return e.Interface.AddAccountKey(address, key, algo, weight)
}

func (e *Environment) RevokeAccountKey(address common.Address, index int) (*stdlib.AccountKey, error) {
	return e.Interface.RevokeAccountKey(address, index)
}

func (e *Environment) RemoveAccountContractCode(address common.Address, name string) error {
	return e.Interface.RemoveAccountContractCode(address, name)
}
