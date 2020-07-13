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

func TestRuntimeHighLevelStorage(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	tx := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
          signer.save(42, to: /storage/number)
        }
      }
    `)

	var storedValue cadence.Value

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() []Address {
			return []Address{
				common.BytesToAddress([]byte{42}),
			}
		},
		setCadenceValue: func(owner common.Address, key string, value cadence.Value) (err error) {
			if storedValue != nil {
				t.Fail()
			}
			storedValue = value

			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(tx, nil, runtimeInterface, nextTransactionLocation())
	require.NoError(t, err)

	assert.Equal(t, cadence.NewInt(42), storedValue)
}
