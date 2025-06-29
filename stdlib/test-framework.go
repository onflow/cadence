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
 *
 */

package stdlib

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
)

// TestFramework & Blockchain are the interfaces to be implemented by
// the test providers.
// Cadence standard library talks to test providers via these interfaces.
// This is used as a way to inject test provider dependencies dynamically.

type TestFramework interface {
	EmulatorBackend() Blockchain

	ReadFile(string) (string, error)
}

type Blockchain interface {
	RunScript(
		context TestFrameworkScriptExecutionContext,
		code string, arguments []interpreter.Value,
	) *ScriptResult

	CreateAccount() (*Account, error)

	GetAccount(interpreter.AddressValue) (*Account, error)

	AddTransaction(
		context TestFrameworkAddTransactionContext,
		code string,
		authorizers []common.Address,
		signers []*Account,
		arguments []interpreter.Value,
	) error

	ExecuteNextTransaction() *TransactionResult

	CommitBlock() error

	DeployContract(
		context TestFrameworkContractDeploymentContext,
		name string,
		path string,
		arguments []interpreter.Value,
	) error

	Logs() []string

	ServiceAccount() (*Account, error)

	Events(
		context TestFrameworkEventsContext,
		eventType interpreter.StaticType,
	) interpreter.Value

	Reset(uint64)

	MoveTime(int64)

	CreateSnapshot(string) error

	LoadSnapshot(string) error
}

type ScriptResult struct {
	Value interpreter.Value
	Error error
}

type TransactionResult struct {
	Error error
}

type Account struct {
	PublicKey *PublicKey
	Address   common.Address
}

type TestFrameworkScriptExecutionContext interface {
	interpreter.ValueExportContext
}

var _ TestFrameworkScriptExecutionContext = &interpreter.Interpreter{}

type TestFrameworkAddTransactionContext interface {
	interpreter.ValueExportContext
}

var _ TestFrameworkAddTransactionContext = &interpreter.Interpreter{}

type TestFrameworkContractDeploymentContext interface {
	interpreter.ValueExportContext
}

var _ TestFrameworkContractDeploymentContext = &interpreter.Interpreter{}

type TestFrameworkEventsContext interface {
	common.MemoryGauge
	interpreter.ArrayCreationContext
	interpreter.ArrayCreationContext
	interpreter.MemberAccessibleContext
}

var _ TestFrameworkEventsContext = &interpreter.Interpreter{}
