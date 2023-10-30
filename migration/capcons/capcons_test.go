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

package capcons

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
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

var _ MigrationReporter = &testCapConsMigrationReporter{}

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

	test := func(t *testing.T, setupFunction, checkFunction string) {

		rt := NewTestInterpreterRuntime()

		// language=cadence
		contract := `
          access(all)
          contract Test {

              access(all)
              resource R {}

              access(all)
              struct CapabilityWrapper {

                  access(all)
                  let capability: Capability

                  init(_ capability: Capability) {
                      self.capability = capability
                  }
              }

              access(all)
              struct CapabilityOptionalWrapper {

                  access(all)
                  let capability: Capability?

                  init(_ capability: Capability?) {
                      self.capability = capability
                  }
              }

              access(all)
              struct CapabilityArrayWrapper {

                  access(all)
                  let capabilities: [Capability]

                  init(_ capabilities: [Capability]) {
                      self.capabilities = capabilities
                  }
              }

              access(all)
              struct CapabilityDictionaryWrapper {

                  access(all)
                  let capabilities: {Int: Capability}

                  init(_ capabilities: {Int: Capability}) {
                      self.capabilities = capabilities
                  }
              }

              access(all)
              fun saveExisting(_ wrapper: fun(Capability): AnyStruct) {
                  self.account.link<&Test.R>(/public/r, target: /private/r)
                  self.account.link<&Test.R>(/private/r, target: /storage/r)

                  let publicCap = self.account.getCapability<&Test.R>(/public/r)
                  let privateCap = self.account.getCapability<&Test.R>(/private/r)

                  self.account.save(wrapper(publicCap), to: /storage/publicCapValue)
                  self.account.save(wrapper(privateCap), to: /storage/privateCapValue)
              }

              access(all)
              fun checkMigratedValues(getter: fun(AnyStruct): Capability) {
                  self.account.save(<-create R(), to: /storage/r)
                  self.checkMigratedValue(capValuePath: /storage/publicCapValue, getter: getter)
                  self.checkMigratedValue(capValuePath: /storage/privateCapValue, getter: getter)
              }

              access(self)
              fun checkMigratedValue(capValuePath: StoragePath, getter: fun(AnyStruct): Capability) {
                  let capValue = self.account.copy<AnyStruct>(from: capValuePath)!
                  let cap = getter(capValue)
                  assert(cap.id != 0)
                  let ref = cap.borrow<&R>()!
              }
          }
        `

		accountCodes := map[runtime.Location][]byte{}
		var events []cadence.Event
		var loggedMessages []string

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location runtime.Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]runtime.Address, error) {
				return []runtime.Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		deployTransaction := utils.DeploymentTransaction("Test", []byte(contract))
		err := rt.ExecuteTransaction(
			runtime.Script{
				Source: deployTransaction,
			},
			runtime.Context{
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
			runtime.Script{
				Source: []byte(setupTx),
			},
			runtime.Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Migrate

		migrator, err := NewMigration(
			rt,
			runtime.Context{
				Interface: runtimeInterface,
			},
			&AddressSliceIterator{
				Addresses: []common.Address{
					address,
				},
			},
			&testAccountIDGenerator{},
		)
		require.NoError(t, err)

		reporter := &testCapConsMigrationReporter{}

		err = migrator.Migrate(reporter)
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

              access(all)
              fun main() {
                 Test.checkMigratedValues(getter: %s)
              }
            `,
			checkFunction,
		)
		_, err = rt.ExecuteScript(
			runtime.Script{
				Source: []byte(checkScript),
			},
			runtime.Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)
	}

	t.Run("directly", func(t *testing.T) {
		t.Parallel()
		t.Skip()

		test(t,
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
		t.Skip()

		test(t,
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
		t.Skip()

		test(t,
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
		t.Skip()

		test(t,
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
		t.Skip()

		test(t,
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
