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
	"fmt"
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

type testCapConsPathCapabilityMigration struct {
	address     common.Address
	addressPath interpreter.AddressPath
}

type testCapConsMissingCapabilityID struct {
	address     common.Address
	addressPath interpreter.AddressPath
}

type testCapConsMigrationReporter struct {
	linkMigrations           []testCapConsLinkMigration
	pathCapabilityMigrations []testCapConsPathCapabilityMigration
	missingCapabilityIDs     []testCapConsMissingCapabilityID
}

var _ CapConsMigrationReporter = &testCapConsMigrationReporter{}

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

func (t *testCapConsMigrationReporter) MigratedPathCapability(
	address common.Address,
	addressPath interpreter.AddressPath,
) {
	t.pathCapabilityMigrations = append(
		t.pathCapabilityMigrations,
		testCapConsPathCapabilityMigration{
			address:     address,
			addressPath: addressPath,
		},
	)
}

func (t *testCapConsMigrationReporter) MissingCapabilityID(
	address common.Address,
	addressPath interpreter.AddressPath,
) {
	t.missingCapabilityIDs = append(
		t.missingCapabilityIDs,
		testCapConsMissingCapabilityID{
			address:     address,
			addressPath: addressPath,
		},
	)
}

func TestCapConsMigration(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	test := func(setupFunction, checkFunction string) {

		rt := newTestInterpreterRuntime()

		// language=cadence
		contract := `
          pub contract Test {

              pub resource R {}

              pub struct CapabilityWrapper {

                  pub let capability: Capability

                  init(_ capability: Capability) {
                      self.capability = capability
                  }
              }

              pub struct CapabilityOptionalWrapper {

                  pub let capability: Capability?

                  init(_ capability: Capability?) {
                      self.capability = capability
                  }
              }

              pub struct CapabilityArrayWrapper {

                  pub let capabilities: [Capability]

                  init(_ capabilities: [Capability]) {
                      self.capabilities = capabilities
                  }
              }

              pub struct CapabilityDictionaryWrapper {

                  pub let capabilities: {Int: Capability}

                  init(_ capabilities: {Int: Capability}) {
                      self.capabilities = capabilities
                  }
              }

              pub fun saveExisting(_ wrapper: ((Capability): AnyStruct)) {
                  self.account.link<&Test.R>(/public/r, target: /private/r)
                  self.account.link<&Test.R>(/private/r, target: /storage/r)

                  let publicCap = self.account.getCapability<&Test.R>(/public/r)
                  let privateCap = self.account.getCapability<&Test.R>(/private/r)

                  self.account.save(wrapper(publicCap), to: /storage/publicCapValue)
                  self.account.save(wrapper(privateCap), to: /storage/privateCapValue)
              }

              pub fun checkMigratedValues(getter: ((AnyStruct): Capability)) {
                  self.account.save(<-create R(), to: /storage/r)
                  self.checkMigratedValue(capValuePath: /storage/publicCapValue, getter: getter)
                  self.checkMigratedValue(capValuePath: /storage/privateCapValue, getter: getter)
              }

              priv fun checkMigratedValue(capValuePath: StoragePath, getter: ((AnyStruct): Capability)) {
                  let capValue = self.account.copy<AnyStruct>(from: capValuePath)!
                  let cap = getter(capValue)
                  assert(cap.id != 0)
                  let ref = cap.borrow<&R>()!
              }
          }
        `

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

		setupTx := fmt.Sprintf(
			// language=cadence
			`
              import Test from 0x1

              transaction {
                  prepare(signer: AuthAccount) {
                     Test.saveExisting(%s)
                  }
              }
            `,
			setupFunction,
		)
		err = rt.ExecuteTransaction(
			Script{
				Source: []byte(setupTx),
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

		err = migrator.Migrate(
			&AddressSliceIterator{
				Addresses: []common.Address{
					address,
				},
			},
			&testAccountIDGenerator{},
			reporter,
		)
		require.NoError(t, err)

		// Check migrated links

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

		// Check migrated capabilities

		require.Equal(t,
			[]testCapConsPathCapabilityMigration{
				{
					address: address,
					addressPath: interpreter.AddressPath{
						Address: address,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPrivate,
							"r",
						),
					},
				},
				{
					address: address,
					addressPath: interpreter.AddressPath{
						Address: address,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							"r",
						),
					},
				},
			},
			reporter.pathCapabilityMigrations,
		)

		// Check

		checkScript := fmt.Sprintf(
			// language=cadence
			`
              import Test from 0x1

              pub fun main() {
                 Test.checkMigratedValues(getter: %s)
              }
            `,
			checkFunction,
		)
		_, err = rt.ExecuteScript(
			Script{
				Source: []byte(checkScript),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)
	}

	t.Run("directly", func(t *testing.T) {
		t.Parallel()

		test(
			// language=cadence
			`
              fun (cap: Capability): AnyStruct {
                  return cap
              }
            `,
			// language=cadence
			`
              fun (value: AnyStruct): Capability {
                  return value as! Capability
              }
            `,
		)
	})

	t.Run("composite", func(t *testing.T) {
		t.Parallel()

		test(
			// language=cadence
			`
              fun (cap: Capability): AnyStruct {
                  return Test.CapabilityWrapper(cap)
              }
            `,
			// language=cadence
			`
              fun (value: AnyStruct): Capability {
                  let wrapper = value as! Test.CapabilityWrapper
                  return wrapper.capability
              }
            `,
		)
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()

		test(
			// language=cadence
			`
              fun (cap: Capability): AnyStruct {
                  return Test.CapabilityOptionalWrapper(cap)
              }
            `,
			// language=cadence
			`
              fun (value: AnyStruct): Capability {
                  let wrapper = value as! Test.CapabilityOptionalWrapper
                  return wrapper.capability!
              }
            `,
		)
	})

	t.Run("array", func(t *testing.T) {
		t.Parallel()

		test(
			// language=cadence
			`
              fun (cap: Capability): AnyStruct {
                  return Test.CapabilityArrayWrapper([cap])
              }
            `,
			// language=cadence
			`
              fun (value: AnyStruct): Capability {
                  let wrapper = value as! Test.CapabilityArrayWrapper
                  return wrapper.capabilities[0]
              }
            `,
		)
	})

	t.Run("dictionary, value", func(t *testing.T) {
		t.Parallel()

		test(
			// language=cadence
			`
              fun (cap: Capability): AnyStruct {
                  return Test.CapabilityDictionaryWrapper({2: cap})
              }
            `,
			// language=cadence
			`
              fun (value: AnyStruct): Capability {
                  let wrapper = value as! Test.CapabilityDictionaryWrapper
                  return wrapper.capabilities[2]!
              }
            `,
		)
	})

	// TODO: add more cases
	// TODO: test non existing

}
