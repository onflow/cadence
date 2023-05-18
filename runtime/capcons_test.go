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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

type testAccountIDGenerator struct {
	ids map[common.Address]uint64
}

func (g *testAccountIDGenerator) GenerateAccountID(address common.Address) (uint64, error) {
	if g.ids == nil {
		g.ids = make(map[common.Address]uint64)
	}
	g.ids[address]++
	return g.ids[address], nil
}

type testCapConsLinkMigration struct {
	addressPath  interpreter.AddressPath
	capabilityID interpreter.UInt64Value
}

type testCapConsMigrationReporter struct {
	linkMigrations []testCapConsLinkMigration
}

func (t *testCapConsMigrationReporter) MigratedLink(
	addressPath interpreter.AddressPath,
	capabilityID interpreter.UInt64Value,
) {
	t.linkMigrations = append(
		t.linkMigrations,
		testCapConsLinkMigration{
			addressPath:  addressPath,
			capabilityID: capabilityID,
		},
	)
}

var _ CapConsMigrationReporter = &testCapConsMigrationReporter{}

func TestCapConsMigration(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	// language=cadence
	contract := `
      pub contract Test {
         pub resource R {}
      }
    `

	address := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contract

	deployTransaction := DeploymentTransaction("Test", []byte(contract))
	err := rt.ExecuteTransaction(
		Script{
			Source: deployTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Setup

	// language=cadence
	linkTransaction := `
      import Test from 0x1

      transaction {
          prepare(signer: AuthAccount) {
             signer.link<&Test.R>(/public/r, target: /private/r)
             signer.link<&Test.R>(/private/r, target: /storage/r)
          }
      }
    `
	err = rt.ExecuteTransaction(
		Script{
			Source: []byte(linkTransaction),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Migrate

	migrator, err := NewCapConsMigration(
		rt,
		Context{
			Interface: runtimeInterface,
		},
	)
	require.NoError(t, err)

	reporter := &testCapConsMigrationReporter{}

	migrator.Migrate(
		NewAddressSliceIterator([]common.Address{address}),
		&testAccountIDGenerator{},
		reporter,
	)

	require.Equal(t,
		[]testCapConsLinkMigration{
			{
				addressPath: interpreter.AddressPath{
					Address: address,
					Path: interpreter.NewUnmeteredPathValue(
						common.PathDomainPublic,
						"r",
					),
				},
				capabilityID: 1,
			},
			{
				addressPath: interpreter.AddressPath{
					Address: address,
					Path: interpreter.NewUnmeteredPathValue(
						common.PathDomainPrivate,
						"r",
					),
				},
				capabilityID: 2,
			},
		},
		reporter.linkMigrations,
	)
}
