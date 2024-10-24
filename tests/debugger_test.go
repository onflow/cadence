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

package tests

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/tests/runtime_utils"
)

func TestRuntimeDebugger(t *testing.T) {

	t.Parallel()

	// Prepare the debugger

	debugger := interpreter.NewDebugger()

	// Request a pause. Does not wait
	debugger.RequestPause()

	// Run the transaction.
	// It will pause/block immediately,
	// so run it in a goroutine

	var wg sync.WaitGroup
	wg.Add(1)

	var logged bool

	go func() {
		defer wg.Done()

		config := DefaultTestInterpreterConfig
		config.Debugger = debugger
		runtime := NewTestInterpreterRuntimeWithConfig(config)

		address := common.MustBytesToAddress([]byte{0x1})

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnProgramLog: func(message string) {
				logged = true
				require.Equal(t, `"Hello, World!"`, message)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                  transaction {
                      prepare(signer: &Account) {
			    	      let answer = 42
                          log("Hello, World!")
                      }
                  }
                `),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	}()

	// Wait for the transaction to run into the pause
	stop := debugger.Pause()

	require.IsType(t, &ast.VariableDeclaration{}, stop.Statement)

	// Step to next statement
	stop = debugger.Next()

	require.IsType(t, &ast.ExpressionStatement{}, stop.Statement)

	activation := debugger.CurrentActivation(stop.Interpreter)
	variable := activation.Find("answer")
	require.NotNil(t, variable)

	value := variable.GetValue(stop.Interpreter)
	require.Equal(
		t,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		value,
	)

	debugger.Continue()

	// Wait for the transaction to finish execution
	wg.Wait()

	require.True(t, logged)
}

func TestRuntimeDebuggerBreakpoints(t *testing.T) {

	t.Parallel()

	nextTransactionLocation := NewTransactionLocationGenerator()
	location := nextTransactionLocation()

	// Prepare the debugger

	debugger := interpreter.NewDebugger()

	// Add a breakpoint
	debugger.AddBreakpoint(location, 5)

	// Run the transaction.
	// It will pause/block at the breakpoint,
	// so run it in a goroutine

	var wg sync.WaitGroup
	wg.Add(1)

	var logged bool

	go func() {
		defer wg.Done()

		config := DefaultTestInterpreterConfig
		config.Debugger = debugger
		runtime := NewTestInterpreterRuntimeWithConfig(config)

		address := common.MustBytesToAddress([]byte{0x1})

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnProgramLog: func(message string) {
				logged = true
				require.Equal(t, `"Hello, World!"`, message)
			},
		}

		err := runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                  transaction {
                      prepare(signer: &Account) {
                          let answer = 42
                          log("Hello, World!")
                      }
                  }
                `),
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.NoError(t, err)
	}()

	// Wait for the transaction to run into the breakpoint
	stop := <-debugger.Stops()

	require.IsType(t, &ast.ExpressionStatement{}, stop.Statement)

	activation := debugger.CurrentActivation(stop.Interpreter)
	variable := activation.Find("answer")
	require.NotNil(t, variable)

	value := variable.GetValue(stop.Interpreter)
	require.Equal(
		t,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		value,
	)

	debugger.Continue()

	// Wait for the transaction to finish execution
	wg.Wait()

	require.True(t, logged)
}
