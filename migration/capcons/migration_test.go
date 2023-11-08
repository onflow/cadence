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
	accountAddressPath interpreter.AddressPath
	capabilityID       interpreter.UInt64Value
}

type testCapConsPathCapabilityMigration struct {
	accountAddress common.Address
	addressPath    interpreter.AddressPath
	borrowType     *interpreter.ReferenceStaticType
}

type testCapConsMissingCapabilityID struct {
	accountAddress common.Address
	addressPath    interpreter.AddressPath
}

type testMissingTarget struct {
	accountAddressValue interpreter.AddressValue
	path                interpreter.PathValue
}

type testMigrationReporter struct {
	linkMigrations             []testCapConsLinkMigration
	pathCapabilityMigrations   []testCapConsPathCapabilityMigration
	missingCapabilityIDs       []testCapConsMissingCapabilityID
	cyclicLinkCyclicLinkErrors []CyclicLinkError
	missingTargets             []testMissingTarget
}

var _ MigrationReporter = &testMigrationReporter{}

func (t *testMigrationReporter) MigratedLink(
	accountAddressPath interpreter.AddressPath,
	capabilityID interpreter.UInt64Value,
) {
	t.linkMigrations = append(
		t.linkMigrations,
		testCapConsLinkMigration{
			accountAddressPath: accountAddressPath,
			capabilityID:       capabilityID,
		},
	)
}

func (t *testMigrationReporter) MigratedPathCapability(
	accountAddress common.Address,
	addressPath interpreter.AddressPath,
	borrowType *interpreter.ReferenceStaticType,
) {
	t.pathCapabilityMigrations = append(
		t.pathCapabilityMigrations,
		testCapConsPathCapabilityMigration{
			accountAddress: accountAddress,
			addressPath:    addressPath,
			borrowType:     borrowType,
		},
	)
}

func (t *testMigrationReporter) MissingCapabilityID(
	accountAddress common.Address,
	addressPath interpreter.AddressPath,
) {
	t.missingCapabilityIDs = append(
		t.missingCapabilityIDs,
		testCapConsMissingCapabilityID{
			accountAddress: accountAddress,
			addressPath:    addressPath,
		},
	)
}

func (t *testMigrationReporter) CyclicLink(cyclicLinkError CyclicLinkError) {
	t.cyclicLinkCyclicLinkErrors = append(
		t.cyclicLinkCyclicLinkErrors,
		cyclicLinkError,
	)
}

func (t *testMigrationReporter) MissingTarget(
	accountAddressValue interpreter.AddressValue,
	path interpreter.PathValue,
) {
	t.missingTargets = append(
		t.missingTargets,
		testMissingTarget{
			accountAddressValue: accountAddressValue,
			path:                path,
		},
	)
}

const testPathIdentifier = "test"

var testAddress = common.MustBytesToAddress([]byte{0x1})

var testRCompositeStaticType = interpreter.NewCompositeStaticTypeComputeTypeID(
	nil,
	common.NewAddressLocation(nil, testAddress, "Test"),
	"Test.R",
)

var testRReferenceStaticType = interpreter.NewReferenceStaticType(
	nil,
	interpreter.UnauthorizedAccess,
	testRCompositeStaticType,
)

var testSCompositeStaticType = interpreter.NewCompositeStaticTypeComputeTypeID(
	nil,
	common.NewAddressLocation(nil, testAddress, "Test"),
	"Test.S",
)

var testSReferenceStaticType = interpreter.NewReferenceStaticType(
	nil,
	interpreter.UnauthorizedAccess,
	testSCompositeStaticType,
)

type testLink struct {
	sourcePath interpreter.PathValue
	targetPath interpreter.PathValue
	borrowType *interpreter.ReferenceStaticType
}

func testPathCapabilityValueMigration(
	t *testing.T,
	capabilityValue *interpreter.PathCapabilityValue,
	pathLinks []testLink,
	accountLinks []interpreter.PathValue,
	expectedMigrations []testCapConsPathCapabilityMigration,
	expectedMissingCapabilityIDs []testCapConsMissingCapabilityID,
	setupFunction, checkFunction string,
	borrowShouldFail bool,
) {
	require.True(t,
		len(expectedMigrations) == 0 ||
			len(expectedMissingCapabilityIDs) == 0,
	)

	rt := NewTestInterpreterRuntime()

	// language=cadence
	contract := `
      access(all)
      contract Test {

          access(all)
          resource R {}

          access(all)
          struct S {}

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
              capability: Capability,
              wrapper: fun(Capability): AnyStruct
          ) {
              self.account.storage.save(
                  wrapper(capability),
                  to: /storage/wrappedCapability
              )
          }

          access(all)
          fun checkMigratedCapabilityValueWithPathLink(getter: fun(AnyStruct): Capability, borrowShouldFail: Bool) {
              self.account.storage.save(<-create R(), to: /storage/test)
              let capValue = self.account.storage.copy<AnyStruct>(from: /storage/wrappedCapability)!
              let cap = getter(capValue)
              assert(cap.id != 0)
              let ref = cap.borrow<&R>()
              if borrowShouldFail {
                  assert(ref == nil)
              } else {
                  assert(ref != nil)
              }
          }

          access(all)
          fun checkMigratedCapabilityValueWithAccountLink(getter: fun(AnyStruct): Capability, borrowShouldFail: Bool) {
              let capValue = self.account.storage.copy<AnyStruct>(from: /storage/wrappedCapability)!
              let cap = getter(capValue)
              assert(cap.id != 0)
              let ref = cap.check<&Account>()
              if borrowShouldFail {
                  assert(ref == nil)
              } else {
                  assert(ref != nil)
              }
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
			return []runtime.Address{testAddress}, nil
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

	environment := runtime.NewBaseInterpreterEnvironment(runtime.Config{})

	// Inject the path capability value.
	//
	// We don't have a way to create a path capability value in a Cadence program anymore,
	// so we have to inject it manually.

	environment.DeclareValue(
		stdlib.StandardLibraryValue{
			Name:  "cap",
			Type:  &sema.CapabilityType{},
			Kind:  common.DeclarationKindConstant,
			Value: capabilityValue,
		},
		setupTransactionLocation,
	)

	// Create and store path and account links

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	for _, testLink := range pathLinks {
		sourcePath := testLink.sourcePath
		targetPath := testLink.targetPath

		require.NotNil(t, testLink.borrowType)

		storage.GetStorageMap(testAddress, sourcePath.Domain.Identifier(), true).
			SetValue(
				inter,
				interpreter.StringStorageMapKey(sourcePath.Identifier),
				interpreter.PathLinkValue{ //nolint:staticcheck
					Type: testLink.borrowType,
					TargetPath: interpreter.PathValue{
						Domain:     targetPath.Domain,
						Identifier: targetPath.Identifier,
					},
				},
			)
	}

	for _, sourcePath := range accountLinks {
		storage.GetStorageMap(testAddress, sourcePath.Domain.Identifier(), true).
			SetValue(
				inter,
				interpreter.StringStorageMapKey(sourcePath.Identifier),
				interpreter.AccountLinkValue{}, //nolint:staticcheck
			)
	}

	err = storage.Commit(inter, false)
	require.NoError(t, err)

	// Save capability values into account

	setupTx := fmt.Sprintf(
		// language=cadence
		`
          import Test from 0x1

          transaction {
              prepare(signer: &Account) {
                 Test.saveExisting(
                     capability: cap,
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
		storage,
		inter,
		&AddressSliceIterator{
			Addresses: []common.Address{
				testAddress,
			},
		},
		&testAccountIDGenerator{},
	)
	require.NoError(t, err)

	reporter := &testMigrationReporter{}

	err = migrator.Migrate(reporter)
	require.NoError(t, err)

	// Check migrated capabilities

	assert.Equal(t,
		expectedMigrations,
		reporter.pathCapabilityMigrations,
	)
	require.Equal(t,
		expectedMissingCapabilityIDs,
		reporter.missingCapabilityIDs,
	)

	if len(expectedMissingCapabilityIDs) == 0 {

		checkFunctionName := "checkMigratedCapabilityValueWithPathLink"
		if len(accountLinks) > 0 {
			checkFunctionName = "checkMigratedCapabilityValueWithAccountLink"
		}

		// Check

		checkScript := fmt.Sprintf(
			// language=cadence
			`
	          import Test from 0x1

	          access(all)
	          fun main() {
	             Test.%s(getter: %s, borrowShouldFail: %v)
	          }
	        `,
			checkFunctionName,
			checkFunction,
			borrowShouldFail,
		)
		_, err = rt.ExecuteScript(
			runtime.Script{
				Source: []byte(checkScript),
			},
			runtime.Context{
				Interface:   runtimeInterface,
				Environment: environment,
				Location:    nextScriptLocation(),
			},
		)
		require.NoError(t, err)
	}
}

func TestPathCapabilityValueMigration(t *testing.T) {

	t.Parallel()

	type linkTestCase struct {
		name                         string
		capabilityValue              *interpreter.PathCapabilityValue
		pathLinks                    []testLink
		accountLinks                 []interpreter.PathValue
		expectedMigrations           []testCapConsPathCapabilityMigration
		expectedMissingCapabilityIDs []testCapConsMissingCapabilityID
		borrowShouldFail             bool
	}

	linkTestCases := []linkTestCase{
		{
			name: "Path links, working chain (public -> private -> storage)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: testRReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			pathLinks: []testLink{
				{
					// Equivalent to:
					//   link<&Test.R>(/public/test, target: /private/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPublic,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					borrowType: testRReferenceStaticType,
				},
				{
					// Equivalent to:
					//   link<&Test.R>(/private/test, target: /storage/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: testPathIdentifier,
					},
					borrowType: testRReferenceStaticType,
				},
			},
			expectedMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType: testRReferenceStaticType,
				},
			},
		},
		{
			name: "Path links, working chain (public -> storage)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: testRReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			pathLinks: []testLink{
				{
					// Equivalent to:
					//   link<&Test.R>(/public/test, target: /storage/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPublic,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: testPathIdentifier,
					},
					borrowType: testRReferenceStaticType,
				},
			},
			expectedMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType: testRReferenceStaticType,
				},
			},
		},
		{
			name: "Path links, working chain (private -> storage)",
			// Equivalent to: getCapability<&Test.R>(/private/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: testRReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			pathLinks: []testLink{
				{
					// Equivalent to:
					//   link<&Test.R>(/private/test, target: /storage/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: testPathIdentifier,
					},
					borrowType: testRReferenceStaticType,
				},
			},
			expectedMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPrivate,
							testPathIdentifier,
						),
					},
					borrowType: testRReferenceStaticType,
				},
			},
		},
		// Test that the migration also follows capability controller,
		// which were already previously migrated from links.
		// Following the (capability value) should not borrow it,
		// i.e. require the storage target to exist,
		// but rather just get the storage target
		{
			name: "Path links, working chain (private -> private -> storage)",
			// Equivalent to: getCapability<&Test.R>(/private/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: testRReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			pathLinks: []testLink{
				{
					// Equivalent to:
					//   link<&Test.R>(/private/test, target: /private/test2)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: "test2",
					},
					borrowType: testRReferenceStaticType,
				},
				{
					// Equivalent to:
					//   link<&Test.R>(/private/test2, target: /storage/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: "test2",
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: testPathIdentifier,
					},
					borrowType: testRReferenceStaticType,
				},
			},
			expectedMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPrivate,
							testPathIdentifier,
						),
					},
					borrowType: testRReferenceStaticType,
				},
			},
		},
		// TODO: verify
		{
			name: "Path links, valid chain (public -> storage), different borrow type",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: testRReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			pathLinks: []testLink{
				{
					// Equivalent to:
					//   link<&Test.S>(/public/test, target: /storage/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPublic,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: testPathIdentifier,
					},
					//
					borrowType: testSReferenceStaticType,
				},
			},
			expectedMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType: testRReferenceStaticType,
				},
			},
			borrowShouldFail: true,
		},
		{
			name: "Path links, cyclic chain (public -> private -> public)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: testRReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			pathLinks: []testLink{
				{
					// Equivalent to:
					//   link<&Test.R>(/public/test, target: /private/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPublic,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					borrowType: testRReferenceStaticType,
				},
				{
					// Equivalent to:
					//   link<&Test.R>(/private/test, target: /public/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainPublic,
						Identifier: testPathIdentifier,
					},
					borrowType: testRReferenceStaticType,
				},
			},
			expectedMigrations: nil,
			expectedMissingCapabilityIDs: []testCapConsMissingCapabilityID{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
				},
			},
		},
		{
			name: "Path links, missing source (public -> private)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: testRReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			pathLinks:          nil,
			expectedMigrations: nil,
			expectedMissingCapabilityIDs: []testCapConsMissingCapabilityID{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
				},
			},
		},
		{
			name: "Path links, missing target (public -> private)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: testRReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			pathLinks: []testLink{
				{
					// Equivalent to:
					//   link<&Test.R>(/public/test, target: /private/test)
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPublic,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					borrowType: testRReferenceStaticType,
				},
			},
			expectedMigrations: nil,
			expectedMissingCapabilityIDs: []testCapConsMissingCapabilityID{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
				},
			},
		},
		{
			name: "Account link, working chain (public)",
			// Equivalent to: getCapability<&AuthAccount>(/public/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: authAccountReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/public/test)
				{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType: fullyEntitledAccountReferenceStaticType,
				},
			},
		},
		{
			name: "Account link, working chain (private)",
			// Equivalent to: getCapability<&AuthAccount>(/public/test)
			capabilityValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: authAccountReferenceStaticType,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
				Address: interpreter.AddressValue(testAddress),
			},
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/private/test)
				{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPrivate,
							testPathIdentifier,
						),
					},
					borrowType: fullyEntitledAccountReferenceStaticType,
				},
			},
		},
	}

	type valueTestCase struct {
		name          string
		setupFunction string
		checkFunction string
	}

	valueTestCases := []valueTestCase{
		{
			name: "directly",
			// language=cadence
			setupFunction: `
              fun (cap: Capability): AnyStruct {
                  return cap
              }
            `,
			// language=cadence
			checkFunction: `
              fun (value: AnyStruct): Capability {
                  return value as! Capability
              }
            `,
		},
		{
			name: "composite",
			// language=cadence
			setupFunction: `
              fun (cap: Capability): AnyStruct {
                  return Test.CapabilityWrapper(cap)
              }
            `,
			// language=cadence
			checkFunction: `
              fun (value: AnyStruct): Capability {
                  let wrapper = value as! Test.CapabilityWrapper
                  return wrapper.capability
              }
            `,
		},
		{
			name: "optional",
			// language=cadence
			setupFunction: `
              fun (cap: Capability): AnyStruct {
                  return Test.CapabilityOptionalWrapper(cap)
              }
            `,
			// language=cadence
			checkFunction: `
              fun (value: AnyStruct): Capability {
                  let wrapper = value as! Test.CapabilityOptionalWrapper
                  return wrapper.capability!
              }
            `,
		},
		{
			name: "array",
			// language=cadence
			setupFunction: `
              fun (cap: Capability): AnyStruct {
                  return Test.CapabilityArrayWrapper([cap])
              }
            `,
			// language=cadence
			checkFunction: `
              fun (value: AnyStruct): Capability {
                  let wrapper = value as! Test.CapabilityArrayWrapper
                  return wrapper.capabilities[0]
              }
            `,
		},
		{
			name: "dictionary",

			// language=cadence
			setupFunction: `
              fun (cap: Capability): AnyStruct {
                  return Test.CapabilityDictionaryWrapper({2: cap})
              }
            `,
			// language=cadence
			checkFunction: `
              fun (value: AnyStruct): Capability {
                  let wrapper = value as! Test.CapabilityDictionaryWrapper
                  return wrapper.capabilities[2]!
              }
           `,
		},
	}

	test := func(linkTestCase linkTestCase, valueTestCase valueTestCase) {
		testName := fmt.Sprintf(
			"%s, %s",
			linkTestCase.name,
			valueTestCase.name,
		)

		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			testPathCapabilityValueMigration(
				t,
				linkTestCase.capabilityValue,
				linkTestCase.pathLinks,
				linkTestCase.accountLinks,
				linkTestCase.expectedMigrations,
				linkTestCase.expectedMissingCapabilityIDs,
				valueTestCase.setupFunction,
				valueTestCase.checkFunction,
				linkTestCase.borrowShouldFail,
			)
		})
	}

	for _, linkTestCase := range linkTestCases {
		for _, valueTestCase := range valueTestCases {
			test(linkTestCase, valueTestCase)
		}
	}
}

// TODO: add more cases
// TODO: test migrated links
