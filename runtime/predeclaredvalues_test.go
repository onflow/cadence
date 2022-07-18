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
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimePredeclaredValues(t *testing.T) {

	t.Parallel()

	valueDeclaration := stdlib.StandardLibraryValue{
		Name:  "foo",
		Type:  sema.IntType,
		Kind:  common.DeclarationKindFunction,
		Value: interpreter.NewUnmeteredIntValueFromInt64(2),
	}

	contract := []byte(`
	  pub contract C {
	      pub fun foo(): Int {
	          return foo
	      }
	  }
	`)

	script := []byte(`
	  import C from 0x1

	  pub fun main(): Int {
		  return foo + C.foo()
	  }
	`)

	runtime := newTestInterpreterRuntime()

	deploy := utils.DeploymentTransaction("C", contract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	// Run transaction

	transactionEnvironment := NewBaseEnvironment(valueDeclaration)

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface:   runtimeInterface,
			Location:    common.TransactionLocation{},
			Environment: transactionEnvironment,
		},
	)
	require.NoError(t, err)

	// Run script

	scriptEnvironment := NewScriptEnvironment(valueDeclaration)

	result, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface:   runtimeInterface,
			Location:    common.ScriptLocation{},
			Environment: scriptEnvironment,
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		cadence.Int{Value: big.NewInt(4)},
		result,
	)
}
