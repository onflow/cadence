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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func TestRuntimeSharedState(t *testing.T) {

	t.Parallel()

	config := DefaultTestInterpreterConfig
	config.StorageFormatV2Enabled = true
	runtime := NewTestInterpreterRuntimeWithConfig(config)

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

	ledger := NewTestLedger(
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

	runtimeInterface := &TestRuntimeInterface{
		Storage: ledger,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnRemoveAccountContractCode: func(location common.AddressLocation) error {
			delete(accountCodes, location)
			return nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnSetInterpreterSharedState: func(state *interpreter.SharedState) {
			interpreterState = state
		},
		OnGetInterpreterSharedState: func() *interpreter.SharedState {
			return interpreterState
		},
	}

	environment := NewBaseInterpreterEnvironment(config)

	nextTransactionLocation := NewTransactionLocationGenerator()

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
                    prepare(signer: &Account) {
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
			// Read account domain register to check if it is a migrated account
			// Read returns no value.
			{
				owner: signerAddress[:],
				key:   []byte(AccountStorageKey),
			},
			// Read contract domain register to check if it is a unmigrated account
			// Read returns no value.
			{
				owner: signerAddress[:],
				key:   []byte(StorageDomainContract),
			},
			// Read all available domain registers to check if it is a new account
			// Read returns no value.
			{
				owner: signerAddress[:],
				key:   []byte(common.PathDomainStorage.Identifier()),
			},
			{
				owner: signerAddress[:],
				key:   []byte(common.PathDomainPrivate.Identifier()),
			},
			{
				owner: signerAddress[:],
				key:   []byte(common.PathDomainPublic.Identifier()),
			},
			{
				owner: signerAddress[:],
				key:   []byte(StorageDomainContract),
			},
			{
				owner: signerAddress[:],
				key:   []byte(stdlib.InboxStorageDomain),
			},
			{
				owner: signerAddress[:],
				key:   []byte(stdlib.CapabilityControllerStorageDomain),
			},
			{
				owner: signerAddress[:],
				key:   []byte(stdlib.CapabilityControllerTagStorageDomain),
			},
			{
				owner: signerAddress[:],
				key:   []byte(stdlib.PathCapabilityStorageDomain),
			},
			{
				owner: signerAddress[:],
				key:   []byte(stdlib.AccountCapabilityStorageDomain),
			},
			// Read account domain register
			{
				owner: signerAddress[:],
				key:   []byte(AccountStorageKey),
			},
			// Read account storage map
			{
				owner: signerAddress[:],
				key:   []byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			},
		},
		ledgerReads,
	)
}
