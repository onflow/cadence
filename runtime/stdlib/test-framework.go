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

package stdlib

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// TestFramework is the interface to be implemented by the test providers.
// Cadence standard library talk to the test providers via this interface.
// This is used as a way to inject test provider dependencies dynamically.
//
type TestFramework interface {
	RunScript(code string, arguments []interpreter.Value) *ScriptResult

	CreateAccount() (*Account, error)

	AddTransaction(
		code string,
		authorizers []common.Address,
		signers []*Account,
		arguments []interpreter.Value,
	) error

	ExecuteNextTransaction() *TransactionResult

	CommitBlock() error

	DeployContract(
		name string,
		code string,
		account *Account,
		arguments []interpreter.Value,
	) error

	ReadFile(string) (string, error)

	UseConfiguration(configuration *Configuration)
}

type ScriptResult struct {
	Value interpreter.Value
	Error error
}

type TransactionResult struct {
	Error error
}

type Account struct {
	Address   common.Address
	PublicKey *PublicKey
}

type Configuration struct {
	Addresses map[string]common.Address
}
