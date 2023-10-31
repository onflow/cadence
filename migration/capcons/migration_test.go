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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
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

type testMigrationReporter struct {
	linkMigrations             []testCapConsLinkMigration
	pathCapabilityMigrations   []testCapConsPathCapabilityMigration
	missingCapabilityIDs       []testCapConsMissingCapabilityID
	cyclicLinkCyclicLinkErrors []CyclicLinkError
}

func (t *testMigrationReporter) CyclicLink(cyclicLinkError CyclicLinkError) {
	t.cyclicLinkCyclicLinkErrors = append(
		t.cyclicLinkCyclicLinkErrors,
		cyclicLinkError,
	)
}

var _ MigrationReporter = &testMigrationReporter{}

func (t *testMigrationReporter) MigratedLink(
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

func (t *testMigrationReporter) MigratedPathCapability(
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

func (t *testMigrationReporter) MissingCapabilityID(
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

func TestMigration(t *testing.T) {

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
              fun saveExisting(
                  publicCap: Capability,
                  privateCap: Capability,
                  wrapper: fun(Capability): AnyStruct
              ) {
                  self.account.storage.save(wrapper(publicCap), to: /storage/publicCapValue)
                  self.account.storage.save(wrapper(privateCap), to: /storage/privateCapValue)
              }

              access(all)
              fun checkMigratedValues(getter: fun(AnyStruct): Capability) {
                  self.account.storage.save(<-create R(), to: /storage/r)
                  self.checkMigratedValue(
                  capValuePath: /storage/publicCapValue, getter: getter)
                  self.checkMigratedValue(capValuePath: /storage/privateCapValue, getter: getter)
              }

              access(self)
              fun checkMigratedValue(capValuePath: StoragePath, getter: fun(AnyStruct): Capability) {
                  let capValue = self.account.storage.copy<AnyStruct>(from: capValuePath)!
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
		nextScriptLocation := NewScriptLocationGenerator()

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

		setupTransactionLocation := nextTransactionLocation()
		contractLocation := common.NewAddressLocation(nil, address, "Test")

		environment := runtime.NewBaseInterpreterEnvironment(runtime.Config{})

		// Inject old PathCapabilityValues.
		// We don't have a way to create them in Cadence anymore.
		//
		// Equivalent to:
		//   let publicCap = account.getCapability<&Test.R>(/public/r)
		//   let privateCap = account.getCapability<&Test.R>(/private/r)

		// TODO: what about the migration of reference type authorized flag?

		rCompositeStaticType := interpreter.NewCompositeStaticTypeComputeTypeID(nil, contractLocation, "Test.R")
		rReferenceStaticType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.UnauthorizedAccess,
			rCompositeStaticType,
		)

		for name, domain := range map[string]common.PathDomain{
			"publicCap":  common.PathDomainPublic,
			"privateCap": common.PathDomainPrivate,
		} {
			environment.DeclareValue(
				stdlib.StandardLibraryValue{
					Name: name,
					Type: &sema.CapabilityType{},
					Kind: common.DeclarationKindConstant,
					Value: &interpreter.PathCapabilityValue{ //nolint:staticcheck
						BorrowType: rReferenceStaticType,
						Path: interpreter.PathValue{
							Domain:     domain,
							Identifier: "r",
						},
						Address: interpreter.AddressValue(address),
					},
				},
				setupTransactionLocation,
			)
		}

		// Create and store links.
		//
		// Equivalent to:
		//   // Working chain
		//   account.link<&Test.R>(/public/r, target: /private/r)
		//   account.link<&Test.R>(/private/r, target: /storage/r)
		//   // Cyclic chain
		//   account.link<&Test.R>(/public/r2, target: /private/r2)
		//   account.link<&Test.R>(/private/r2, target: /public/r2)

		storage, inter, err := rt.Storage(runtime.Context{
			Interface: runtimeInterface,
		})
		require.NoError(t, err)

		for sourcePath, targetPath := range map[interpreter.PathValue]interpreter.PathValue{
			// Working chain
			{
				Domain:     common.PathDomainPublic,
				Identifier: "r",
			}: {
				Domain:     common.PathDomainPrivate,
				Identifier: "r",
			},
			{
				Domain:     common.PathDomainPrivate,
				Identifier: "r",
			}: {
				Domain:     common.PathDomainStorage,
				Identifier: "r",
			},
			// Cyclic chain
			{
				Domain:     common.PathDomainPublic,
				Identifier: "r2",
			}: {
				Domain:     common.PathDomainPrivate,
				Identifier: "r2",
			},
			{
				Domain:     common.PathDomainPrivate,
				Identifier: "r2",
			}: {
				Domain:     common.PathDomainPublic,
				Identifier: "r2",
			},
		} {

			storage.GetStorageMap(address, sourcePath.Domain.Identifier(), true).
				SetValue(
					inter,
					interpreter.StringStorageMapKey(sourcePath.Identifier),
					interpreter.PathLinkValue{ //nolint:staticcheck
						Type: rReferenceStaticType,
						TargetPath: interpreter.PathValue{
							Domain:     targetPath.Domain,
							Identifier: targetPath.Identifier,
						},
					},
				)
		}

		err = storage.Commit(inter, false)
		require.NoError(t, err)

		setupTx := fmt.Sprintf(
			// language=cadence
			`
              import Test from 0x1

              transaction {
                  prepare(signer: &Account) {
                     Test.saveExisting(
                         publicCap: publicCap,
                         privateCap: privateCap,
                         wrapper: %s
                     )
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
				Interface:   runtimeInterface,
				Environment: environment,
				Location:    setupTransactionLocation,
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

		reporter := &testMigrationReporter{}

		err = migrator.Migrate(reporter)
		require.NoError(t, err)

		// Check migrated links

		assert.Equal(t,
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
		assert.Equal(
			t,
			[]CyclicLinkError{
				{
					Paths: []interpreter.PathValue{
						{Domain: common.PathDomainPublic, Identifier: "r2"},
						{Domain: common.PathDomainPrivate, Identifier: "r2"},
						{Domain: common.PathDomainPublic, Identifier: "r2"},
					},
					Address: address,
				},
				{
					Paths: []interpreter.PathValue{
						{Domain: common.PathDomainPrivate, Identifier: "r2"},
						{Domain: common.PathDomainPublic, Identifier: "r2"},
						{Domain: common.PathDomainPrivate, Identifier: "r2"},
					},
					Address: address,
				},
			},
			reporter.cyclicLinkCyclicLinkErrors,
		)

		// Check migrated capabilities

		assert.Equal(t,
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
		require.Empty(t, reporter.missingCapabilityIDs)

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
				Location:  nextScriptLocation(),
			},
		)
		require.NoError(t, err)
	}

	t.Run("directly", func(t *testing.T) {
		t.Parallel()

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
	// TODO: account link

}
