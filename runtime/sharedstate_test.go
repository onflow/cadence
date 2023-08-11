/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeSharedState(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAddress := common.MustBytesToAddress([]byte{0x1})

	deploy1 := DeploymentTransaction("C1", []byte(`
        access(all) contract C1 {
            access(all) fun hello() {
                log("Hello from C1!")
            }
        }
    `))

	deploy2 := DeploymentTransaction("C2", []byte(`
        access(all) contract C2 {
            access(all) fun hello() {
                log("Hello from C2!")
            }
        }
    `))

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event
	var loggedMessages []string

	var interpreterState *interpreter.SharedState

	var ledgerReads []ownerKeyPair

	ledger := newTestLedger(
		func(owner, key, value []byte) {
			ledgerReads = append(
				ledgerReads,
				ownerKeyPair{
					owner: owner,
					key:   key,
				},
			)
		},
		nil,
	)

	runtimeInterface := &testRuntimeInterface{
		storage: ledger,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		removeAccountContractCode: func(location common.AddressLocation) error {
			delete(accountCodes, location)
			return nil
		},
		resolveLocation: multipleIdentifierLocationResolver,
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		setInterpreterSharedState: func(state *interpreter.SharedState) {
			interpreterState = state
		},
		getInterpreterSharedState: func() *interpreter.SharedState {
			return interpreterState
		},
	}

	environment := NewBaseInterpreterEnvironment(Config{})

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contracts

	for _, source := range [][]byte{
		deploy1,
		deploy2,
	} {
		err := runtime.ExecuteTransaction(
			Script{
				Source: source,
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
		require.NoError(t, err)
	}

	assert.NotEmpty(t, accountCodes)

	// Call C1.hello using transaction

	loggedMessages = nil

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
                import C1 from 0x1

                transaction {
                    prepare(signer: AuthAccount) {
                        C1.hello()
                    }
                }
            `),
			Arguments: nil,
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{`"Hello from C1!"`}, loggedMessages)

	// Call C1.hello manually

	loggedMessages = nil

	_, err = runtime.InvokeContractFunction(
		common.AddressLocation{
			Address: signerAddress,
			Name:    "C1",
		},
		"hello",
		nil,
		nil,
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{`"Hello from C1!"`}, loggedMessages)

	// Call C2.hello manually

	loggedMessages = nil

	_, err = runtime.InvokeContractFunction(
		common.AddressLocation{
			Address: signerAddress,
			Name:    "C2",
		},
		"hello",
		nil,
		nil,
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{`"Hello from C2!"`}, loggedMessages)

	// Assert shared state was used,
	// i.e. data was not re-read

	require.Equal(t,
		[]ownerKeyPair{
			{
				owner: signerAddress[:],
				key:   []byte(StorageDomainContract),
			},
			{
				owner: signerAddress[:],
				key:   []byte(StorageDomainContract),
			},
			{
				owner: signerAddress[:],
				key:   []byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			},
		},
		ledgerReads,
	)
}
