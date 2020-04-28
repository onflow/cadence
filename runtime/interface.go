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
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
)

type Interface interface {
	// ResolveImport resolves an import of a program.
	ResolveImport(Location) ([]byte, error)
	// GetCachedProgram attempts to get a parsed program from a cache.
	GetCachedProgram(Location) (*ast.Program, error)
	// CacheProgram adds a parsed program to a cache.
	CacheProgram(Location, *ast.Program) error
	// GetValue gets a value for the given key in the storage, controlled and owned by the given accounts.
	GetValue(owner, controller, key []byte) (value []byte, err error)
	// SetValue sets a value for the given key in the storage, controlled and owned by the given accounts.
	SetValue(owner, controller, key, value []byte) (err error)
	// CreateAccount creates a new account with the given public keys and code.
	CreateAccount(publicKeys [][]byte) (address Address, err error)
	// AddAccountKey appends a key to an account.
	AddAccountKey(address Address, publicKey []byte) error
	// RemoveAccountKey removes a key from an account by index.
	RemoveAccountKey(address Address, index int) (publicKey []byte, err error)
	// CheckCode checks the validity of the code.
	CheckCode(address Address, code []byte) (err error)
	// UpdateAccountCode updates the code associated with an account.
	UpdateAccountCode(address Address, code []byte, checkPermission bool) (err error)
	// GetSigningAccounts returns the signing accounts.
	GetSigningAccounts() []Address
	// Log logs a string.
	Log(string)
	// EmitEvent is called when an event is emitted by the runtime.
	EmitEvent(cadence.Event)
	// ValueExists returns true if the given key exists in the storage, controlled and owned by the given accounts.
	ValueExists(owner, controller, key []byte) (exists bool, err error)
	// GenerateUUID is called to generate a UUID.
	GenerateUUID() uint64
	// GetComputationLimit returns the computation limit. A value <= 0 means there is no limit
	GetComputationLimit() uint64
	// DecodeArgument decodes a transaction argument against the given type.
	DecodeArgument(b []byte, t cadence.Type) (cadence.Value, error)
}

type EmptyRuntimeInterface struct{}

func (i *EmptyRuntimeInterface) ResolveImport(location Location) ([]byte, error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) GetCachedProgram(location Location) (*ast.Program, error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) CacheProgram(location Location, program *ast.Program) error {
	return nil
}

func (i *EmptyRuntimeInterface) ValueExists(controller, owner, key []byte) (exists bool, err error) {
	return false, nil
}

func (i *EmptyRuntimeInterface) GetValue(controller, owner, key []byte) (value []byte, err error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) SetValue(controller, owner, key, value []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) CreateAccount(publicKeys [][]byte) (address Address, err error) {
	return Address{}, nil
}

func (i *EmptyRuntimeInterface) AddAccountKey(address Address, publicKey []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) RemoveAccountKey(address Address, index int) (publicKey []byte, err error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) CheckCode(address Address, code []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) UpdateAccountCode(address Address, code []byte, checkPermission bool) error {
	return nil
}

func (i *EmptyRuntimeInterface) GetSigningAccounts() []Address {
	return nil
}

func (i *EmptyRuntimeInterface) Log(message string) {}

func (i *EmptyRuntimeInterface) EmitEvent(event cadence.Event) {}

func (i *EmptyRuntimeInterface) GenerateUUID() uint64 {
	return 0
}

func (i *EmptyRuntimeInterface) GetComputationLimit() uint64 {
	return 0
}

func (i *EmptyRuntimeInterface) DecodeArgument(b []byte, t cadence.Type) (cadence.Value, error) {
	return nil, nil
}
