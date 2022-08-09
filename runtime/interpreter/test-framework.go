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
 *
 * Based on https://github.com/wk8/go-ordered-map, Copyright Jean Rougé
 *
 */

package interpreter

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// TestFramework is the interface to be implemented by the test providers.
// Cadence standard library talk to the test providers via this interface.
// This is used as a way to inject test provider dependencies dynamically.
//
type TestFramework interface {
	RunScript(code string, args []Value) *ScriptResult

	CreateAccount() (*Account, error)

	AddTransaction(
		code string,
		authorizer *common.Address,
		signers []*Account,
		args []Value,
	) error

	ExecuteNextTransaction() *TransactionResult

	CommitBlock() error

	DeployContract(
		name string,
		code string,
		args []Value,
		authorizer common.Address,
		signers []*Account,
	) error
}

type ScriptResult struct {
	Value Value
	Error error
}

type TransactionResult struct {
	Error error
}

type Account struct {
	Address    common.Address
	AccountKey *AccountKey
	PrivateKey []byte
}

type AccountKey struct {
	KeyIndex  int
	PublicKey *PublicKey
	HashAlgo  sema.HashAlgorithm
	Weight    int
	IsRevoked bool
}

type PublicKey struct {
	PublicKey []byte
	SignAlgo  sema.SignatureAlgorithm
}

// TestFrameworkNotProvidedError is the error thrown if test-stdlib functionality is
// used without providing a test-framework implementation.
//
type TestFrameworkNotProvidedError struct{}

var _ errors.InternalError = TestFrameworkNotProvidedError{}

func (TestFrameworkNotProvidedError) IsInternalError() {}

func (e TestFrameworkNotProvidedError) Error() string {
	return "test framework not provided"
}
