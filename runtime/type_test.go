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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
)

func TestRuntimeTypeStorage(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	tx1 := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
          signer.save(Type<Int>(), to: /storage/intType)
        }
      }
    `)

	tx2 := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
          let intType = signer.load<Type>(from: /storage/intType)
          log(intType?.identifier)
        }
      }
    `)

	var loggedMessage string

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{
				common.BytesToAddress([]byte{42}),
			}, nil
		},
		log: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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

func TestBlockTimestamp(t *testing.T) {

	t.Parallel()

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		script := []byte(`
			transaction {
				prepare() {
					let block = getCurrentBlock()
					var ts: UFix64 = block.timestamp
					log(ts.isInstance(Type<UFix64>()))

					var div: UFix64 = 4.0

					// Shouldn't panic
					var res = ts/div
				}
			}
        `)

		var loggedMessage string

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
			log: func(message string) {
				loggedMessage = message
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()
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
		assert.Equal(t, "true", loggedMessage, "Block.timestamp is not UFix64")

	})

	t.Run("script", func(t *testing.T) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		script := []byte(`
			pub fun main(): [UFix64] {
				let block = getCurrentBlock()
				var ts: UFix64 = block.timestamp

				var div: UFix64 = 4.0

				// Shouldn't panic
				var res = ts/div

				return [ts, res]
			}
        `)

		value, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: &testRuntimeInterface{},
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)
		require.NotNil(t, value)

		require.IsType(t, cadence.Array{}, value)
		values := value.(cadence.Array).Values

		require.Equal(t, 2, len(values))
		assert.IsType(t, cadence.UFix64Type{}, values[0].Type())
		assert.IsType(t, cadence.UFix64Type{}, values[1].Type())
	})
}
