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

      pub fun save(kind: String, capability: Capability, account: AuthAccount) {
          account.save(
              capability,
              to: StoragePath(identifier: kind.concat("_capability"))!
          )
          // TODO:
          // account.save(
          //     Test.CapabilityWrapper(capability),
          //     to: StoragePath(identifier: kind.concat("_capabilityWrapper"))!
          // )
          // account.save(
          //     Test.CapabilityOptionalWrapper(capability),
          //     to: StoragePath(identifier: kind.concat("_capabilityOptionalWrapper"))!
          // )
          // account.save(
          //     Test.CapabilityArrayWrapper([capability]),
          //     to: StoragePath(identifier: kind.concat("_capabilityArrayWrapper"))!
          // )
          // account.save(
          //     Test.CapabilityDictionaryWrapper({0: capability}),
          //     to: StoragePath(identifier: kind.concat("_capabilityDictionaryWrapper"))!
          // )
          // TODO: add more cases
          // TODO: test non existing
      }

      transaction {
          prepare(signer: AuthAccount) {
             signer.link<&Test.R>(/public/r, target: /private/r)
             signer.link<&Test.R>(/private/r, target: /storage/r)

             let publicCap = signer.getCapability<&Test.R>(/public/r)
             let privateCap = signer.getCapability<&Test.R>(/private/r)

             save(
                 kind: "public",
                 capability: publicCap,
                 account: signer
			 )
             save(
                 kind: "private",
                 capability: privateCap,
                 account: signer
             )
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

	err = migrator.Migrate(
		&AddressSliceIterator{Addresses: []common.Address{address}},
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

	storage, _, err := rt.Storage(Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	storageMap := storage.GetStorageMap(address, pathDomainStorage, false)

	expectedReferenceStaticType := interpreter.ReferenceStaticType{
		BorrowedType: interpreter.CompositeStaticType{
			Location: common.AddressLocation{
				Name:    "Test",
				Address: address,
			},
			QualifiedIdentifier: "Test.R",
			TypeID:              "A.0000000000000001.Test.R",
		},
	}

	expectedCapabilityID := interpreter.UInt64Value(1)
	for _, kind := range []string{"public", "private"} {

		publicCapability := storageMap.ReadValue(nil, interpreter.StringStorageMapKey(kind+"_capability"))
		require.IsType(t, &interpreter.IDCapabilityValue{}, publicCapability)
		if publicCapability, ok := publicCapability.(*interpreter.IDCapabilityValue); ok {
			assert.Equal(t, expectedCapabilityID, publicCapability.ID)
			expectedCapabilityID++

			assert.Equal(t, address, common.Address(publicCapability.Address))
			assert.True(t, publicCapability.BorrowType.Equal(expectedReferenceStaticType))
		}
	}
}
