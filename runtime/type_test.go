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

package runtime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
)

func TestRuntimeTypeStorage(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	tx1 := []byte(`
      transaction {
        prepare(signer: auth(Storage) &Account) {
          signer.storage.save(Type<Int>(), to: /storage/intType)
        }
      }
    `)

	tx2 := []byte(`
      transaction {
        prepare(signer: auth(Storage) &Account) {
          let intType = signer.storage.load<Type>(from: /storage/intType)
          log(intType?.identifier)
        }
      }
    `)

	var loggedMessage string

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{
				common.MustBytesToAddress([]byte{42}),
			}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: tx1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, `"Int"`, loggedMessage)
}

func TestRuntimeBlockFieldTypes(t *testing.T) {

	t.Parallel()

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
            transaction {
                prepare() {
                    let block = getCurrentBlock()
                    let id = block.id
                    log(id.isInstance(Type<[UInt8; 32]>()))

                    var ts: UFix64 = block.timestamp
                    log(ts.isInstance(Type<UFix64>()))

                    var div: UFix64 = 4.0

                    // Shouldn't panic
                    var res = ts/div
                }
            }
        `)

		var loggedMessages []string

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnGetSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()
		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
		assert.Equal(
			t,
			[]string{
				"true",
				"true",
			},
			loggedMessages,
		)

	})

	t.Run("script", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
            access(all) fun main(): [UFix64] {
                let block = getCurrentBlock()

                let id = block.id
                log(id.isInstance(Type<[UInt8; 32]>()))

                var ts: UFix64 = block.timestamp
                log(ts.isInstance(Type<UFix64>()))

                var div: UFix64 = 4.0

                // Shouldn't panic
                var res = ts/div

                return [ts, res]
            }
        `)

		storage := NewTestLedger(nil, nil)

		var loggedMessages []string

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}

		value, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)
		require.NotNil(t, value)

		require.IsType(t, cadence.Array{}, value)
		values := value.(cadence.Array).Values

		require.Equal(t, 2, len(values))
		assert.IsType(t, cadence.UFix64Type, values[0].Type())
		assert.IsType(t, cadence.UFix64Type, values[1].Type())

		assert.Equal(
			t,
			[]string{
				"true",
				"true",
			},
			loggedMessages,
		)
	})
}
