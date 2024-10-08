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

package capcons

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type testCapConHandler struct {
	ids        map[common.Address]uint64
	events     []cadence.Event
	dropEvents bool
}

var _ stdlib.CapabilityControllerIssueHandler = &testCapConHandler{}

func (g *testCapConHandler) GenerateAccountID(address common.Address) (uint64, error) {
	if g.ids == nil {
		g.ids = make(map[common.Address]uint64)
	}
	g.ids[address]++
	return g.ids[address], nil
}

func (g *testCapConHandler) EmitEvent(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	values []interpreter.Value,
) {
	if g.dropEvents {
		return
	}

	runtime.EmitEventFields(
		inter,
		locationRange,
		eventType,
		values,
		func(event cadence.Event) error {
			g.events = append(g.events, event)
			return nil
		},
	)
}

type testCapConsLinkMigration struct {
	accountAddressPath interpreter.AddressPath
	capabilityID       interpreter.UInt64Value
}

type testCapConsPathCapabilityMigration struct {
	accountAddress common.Address
	addressPath    interpreter.AddressPath
	borrowType     *interpreter.ReferenceStaticType
	capabilityID   interpreter.UInt64Value
}

type testCapConsMissingCapabilityID struct {
	accountAddress common.Address
	addressPath    interpreter.AddressPath
}

type testStorageCapConIssued struct {
	accountAddress common.Address
	addressPath    interpreter.AddressPath
	borrowType     *interpreter.ReferenceStaticType
	capabilityID   interpreter.UInt64Value
}

type testStorageCapConsMissingBorrowType struct {
	targetPath interpreter.AddressPath
	storedPath interpreter.AddressPath
}

type testStorageCapConsInferredBorrowType struct {
	targetPath interpreter.AddressPath
	borrowType *interpreter.ReferenceStaticType
	storedPath interpreter.AddressPath
}

type testMigration struct {
	storageKey    interpreter.StorageKey
	storageMapKey interpreter.StorageMapKey
	migration     string
}

type testMigrationReporter struct {
	migrations                       []testMigration
	errors                           []error
	linkMigrations                   []testCapConsLinkMigration
	pathCapabilityMigrations         []testCapConsPathCapabilityMigration
	missingCapabilityIDs             []testCapConsMissingCapabilityID
	issuedStorageCapCons             []testStorageCapConIssued
	missingStorageCapConBorrowTypes  []testStorageCapConsMissingBorrowType
	inferredStorageCapConBorrowTypes []testStorageCapConsInferredBorrowType
	cyclicLinkErrors                 []CyclicLinkError
	missingTargets                   []interpreter.AddressPath
}

var _ migrations.Reporter = &testMigrationReporter{}
var _ LinkMigrationReporter = &testMigrationReporter{}
var _ CapabilityMigrationReporter = &testMigrationReporter{}
var _ StorageCapabilityMigrationReporter = &testMigrationReporter{}

func (t *testMigrationReporter) Migrated(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	migration string,
) {
	t.migrations = append(
		t.migrations,
		testMigration{
			storageKey:    storageKey,
			storageMapKey: storageMapKey,
			migration:     migration,
		},
	)
}

func (t *testMigrationReporter) Error(err error) {
	t.errors = append(t.errors, err)
}
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
	capabilityID interpreter.UInt64Value,
) {
	t.pathCapabilityMigrations = append(
		t.pathCapabilityMigrations,
		testCapConsPathCapabilityMigration{
			accountAddress: accountAddress,
			addressPath:    addressPath,
			borrowType:     borrowType,
			capabilityID:   capabilityID,
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

func (t *testMigrationReporter) MissingBorrowType(
	targetPath interpreter.AddressPath,
	storedPath interpreter.AddressPath,
) {
	t.missingStorageCapConBorrowTypes = append(
		t.missingStorageCapConBorrowTypes,
		testStorageCapConsMissingBorrowType{
			targetPath: targetPath,
			storedPath: storedPath,
		},
	)
}

func (t *testMigrationReporter) InferredMissingBorrowType(
	targetPath interpreter.AddressPath,
	borrowType *interpreter.ReferenceStaticType,
	storedPath interpreter.AddressPath,
) {
	t.inferredStorageCapConBorrowTypes = append(
		t.inferredStorageCapConBorrowTypes,
		testStorageCapConsInferredBorrowType{
			targetPath: targetPath,
			borrowType: borrowType,
			storedPath: storedPath,
		},
	)
}

func (t *testMigrationReporter) IssuedStorageCapabilityController(
	accountAddress common.Address,
	addressPath interpreter.AddressPath,
	borrowType *interpreter.ReferenceStaticType,
	capabilityID interpreter.UInt64Value,
) {
	t.issuedStorageCapCons = append(
		t.issuedStorageCapCons,
		testStorageCapConIssued{
			accountAddress: accountAddress,
			addressPath:    addressPath,
			borrowType:     borrowType,
			capabilityID:   capabilityID,
		},
	)
}

func (t *testMigrationReporter) CyclicLink(cyclicLinkError CyclicLinkError) {
	t.cyclicLinkErrors = append(
		t.cyclicLinkErrors,
		cyclicLinkError,
	)
}

func (t *testMigrationReporter) MissingTarget(
	accountAddressPath interpreter.AddressPath,
) {
	t.missingTargets = append(
		t.missingTargets,
		accountAddressPath,
	)
}

func (t *testMigrationReporter) DictionaryKeyConflict(addressPath interpreter.AddressPath) {
	// For testing purposes, record the conflict as an error
	t.errors = append(t.errors, fmt.Errorf("dictionary key conflict: %s", addressPath))
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

func storeTestAccountLinks(accountLinks []interpreter.PathValue, storage *runtime.Storage, inter *interpreter.Interpreter) {
	for _, sourcePath := range accountLinks {
		storage.GetStorageMap(testAddress, sourcePath.Domain.Identifier(), true).
			SetValue(
				inter,
				interpreter.StringStorageMapKey(sourcePath.Identifier),
				interpreter.AccountLinkValue{}, //nolint:staticcheck
			)
	}
}

func storeTestPathLinks(t *testing.T, pathLinks []testLink, storage *runtime.Storage, inter *interpreter.Interpreter) {
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
}

func testPathCapabilityValueMigration(
	t *testing.T,
	capabilityValue *interpreter.PathCapabilityValue, //nolint:staticcheck
	pathLinks []testLink,
	accountLinks []interpreter.PathValue,
	expectedMigrations []testMigration,
	expectedErrors []error,
	expectedPathMigrations []testCapConsPathCapabilityMigration,
	expectedMissingCapabilityIDs []testCapConsMissingCapabilityID,
	expectedEvents []string,
	setupFunction string,
	checkFunction string,
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
	{
		// Create new runtime.Storage used for storing path and account links
		storage, inter, err := rt.Storage(runtime.Context{
			Interface: runtimeInterface,
		})
		require.NoError(t, err)

		storeTestPathLinks(t, pathLinks, storage, inter)

		storeTestAccountLinks(accountLinks, storage, inter)

		err = storage.Commit(inter, false)
		require.NoError(t, err)
	}

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

	// Create new runtime.Storage for migration.
	// WARNING: don't reuse old storage (created for storing path and account links)
	// because it has outdated cache after ExecuteTransaction() commits new data
	// to underlying ledger.
	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := &testMigrationReporter{}

	privatePublicCapabilityMapping := &PathCapabilityMapping{}
	typedStorageCapabilityMapping := &PathTypeCapabilityMapping{}
	untypedStorageCapabilityMapping := &PathCapabilityMapping{}

	handler := &testCapConHandler{}

	storageDomainCapabilities := &AccountsCapabilities{}

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&StorageCapMigration{
				StorageDomainCapabilities: storageDomainCapabilities,
			},
		),
	)

	storageCapabilities := storageDomainCapabilities.Get(testAddress)
	if storageCapabilities != nil {
		IssueAccountCapabilities(
			inter,
			storage,
			reporter,
			testAddress,
			storageCapabilities,
			handler,
			typedStorageCapabilityMapping,
			untypedStorageCapabilityMapping,
			func(_ interpreter.StaticType) interpreter.Authorization {
				return interpreter.UnauthorizedAccess
			},
		)
	}

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&LinkValueMigration{
				CapabilityMapping: privatePublicCapabilityMapping,
				IssueHandler:      handler,
				Handler:           handler,
				Reporter:          reporter,
			},
		),
	)

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&CapabilityValueMigration{
				PrivatePublicCapabilityMapping:  privatePublicCapabilityMapping,
				TypedStorageCapabilityMapping:   typedStorageCapabilityMapping,
				UntypedStorageCapabilityMapping: untypedStorageCapabilityMapping,
				Reporter:                        reporter,
			},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Equal(t,
		expectedMigrations,
		reporter.migrations,
	)
	assert.Equal(t,
		expectedPathMigrations,
		reporter.pathCapabilityMigrations,
	)
	require.Equal(t,
		expectedMissingCapabilityIDs,
		reporter.missingCapabilityIDs,
	)
	require.Equal(t,
		expectedErrors,
		reporter.errors,
	)

	err = storage.CheckHealth()
	require.NoError(t, err)

	require.Equal(t,
		expectedEvents,
		nonDeploymentEventStrings(handler.events),
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

func nonDeploymentEventStrings(events []cadence.Event) []string {
	accountContractAddedEventTypeID := stdlib.AccountContractAddedEventType.ID()

	strings := make([]string, 0, len(events))
	for _, event := range events {
		// Skip deployment events, i.e. contract added to account
		if common.TypeID(event.Type().ID()) == accountContractAddedEventTypeID {
			continue
		}
		strings = append(strings, event.String())
	}
	return strings
}

func TestPathCapabilityValueMigration(t *testing.T) {

	t.Parallel()

	type linkTestCase struct {
		name                         string
		capabilityValue              *interpreter.PathCapabilityValue //nolint:staticcheck
		pathLinks                    []testLink
		accountLinks                 []interpreter.PathValue
		expectedMigrations           []testMigration
		expectedErrors               []error
		expectedPathMigrations       []testCapConsPathCapabilityMigration
		expectedMissingCapabilityIDs []testCapConsMissingCapabilityID
		borrowShouldFail             bool
		expectedEvents               []string
	}

	expectedWrappedCapabilityValueMigration := testMigration{
		storageKey: interpreter.StorageKey{
			Address: testAddress,
			Key:     common.PathDomainStorage.Identifier(),
		},
		storageMapKey: interpreter.StringStorageMapKey("wrappedCapability"),
		migration:     "CapabilityValueMigration",
	}

	linkTestCases := []linkTestCase{
		{
			name: "Path links, working chain (public -> private -> storage)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/public/test, target: /private/test)
				{
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
				// Equivalent to:
				//   link<&Test.R>(/private/test, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType:   testRReferenceStaticType,
					capabilityID: 2,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
				`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
			},
		},
		{
			name: "Path links, working chain (public -> storage)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/public/test, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType:   testRReferenceStaticType,
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
			},
		},
		{
			name: "Path links, working chain (private -> storage)",
			// Equivalent to: getCapability<&Test.R>(/private/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/private/test, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPrivate,
							testPathIdentifier,
						),
					},
					borrowType:   testRReferenceStaticType,
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
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
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/private/test, target: /private/test2)
				{
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
				// Equivalent to:
				//   link<&Test.R>(/private/test2, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey("test2"),
					migration:     "LinkValueMigration",
				},
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPrivate,
							testPathIdentifier,
						),
					},
					borrowType:   testRReferenceStaticType,
					capabilityID: 2,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
				`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
			},
		},
		// NOTE: this migrates a broken capability to a broken capability
		{
			name: "Path links, valid chain (public -> storage), different borrow type",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.S>(/public/test, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType:   testRReferenceStaticType,
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.S>(), path: /storage/test)`,
			},
			borrowShouldFail: true,
		},
		{
			name: "Path links, cyclic chain (public -> private -> public)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/public/test, target: /private/test)
				{
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
				// Equivalent to:
				//   link<&Test.R>(/private/test, target: /public/test)
				{
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
			expectedPathMigrations: nil,
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
			expectedEvents: []string{},
		},
		{
			name: "Path links, missing source (public -> private)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks:              nil,
			expectedPathMigrations: nil,
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
			expectedEvents: []string{},
		},
		{
			name: "Path links, missing target (public -> private)",
			// Equivalent to: getCapability<&Test.R>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/public/test, target: /private/test)
				{
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
			expectedPathMigrations: nil,
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
			expectedEvents: []string{},
		},
		{
			name: "Path link, storage path",
			// Equivalent to: getCapability<&Test.R>(/storage/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				testRReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: testPathIdentifier,
				},
			),
			pathLinks: nil,
			expectedMigrations: []testMigration{
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainStorage,
							testPathIdentifier,
						),
					},
					borrowType:   testRReferenceStaticType,
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
			},
		},
		{
			name: "Account link, working chain (public), unauthorized",
			// Equivalent to: getCapability<&Account>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				unauthorizedAccountReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			),
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/public/test)
				{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType:   unauthorizedAccountReferenceStaticType,
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				"flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())",
			},
		},
		{
			name: "Account link, working chain (public), authorized",
			// Equivalent to: getCapability<auth(Capabilities, Contracts, Inbox, Keys, Storage) &Account>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.FullyEntitledAccountAccess,
					interpreter.PrimitiveStaticTypeAccount,
				),
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			),
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/public/test)
				{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPublic,
							testPathIdentifier,
						),
					},
					borrowType:   fullyEntitledAccountReferenceStaticType,
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				"flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())",
			},
		},
		{
			name: "Account link, working chain (private), unauthorized",
			// Equivalent to: getCapability<&Account>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				unauthorizedAccountReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			),
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/private/test)
				{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPrivate,
							testPathIdentifier,
						),
					},
					borrowType:   unauthorizedAccountReferenceStaticType,
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				"flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())",
			},
		},
		{
			name: "Account link, working chain (private), authorized",
			// Equivalent to: getCapability<auth(Capabilities, Contracts, Inbox, Keys, Storage) &Account>(/public/test)
			capabilityValue: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				fullyEntitledAccountReferenceStaticType,
				interpreter.AddressValue(testAddress),
				interpreter.PathValue{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			),
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/private/test)
				{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				expectedWrappedCapabilityValueMigration,
			},
			expectedPathMigrations: []testCapConsPathCapabilityMigration{
				{
					accountAddress: testAddress,
					addressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.NewUnmeteredPathValue(
							common.PathDomainPrivate,
							testPathIdentifier,
						),
					},
					borrowType:   fullyEntitledAccountReferenceStaticType,
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				"flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())",
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
				linkTestCase.expectedErrors,
				linkTestCase.expectedPathMigrations,
				linkTestCase.expectedMissingCapabilityIDs,
				linkTestCase.expectedEvents,
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

func testLinkMigration(
	t *testing.T,
	pathLinks []testLink,
	accountLinks []interpreter.PathValue,
	expectedMigrations []testMigration,
	expectedErrors []error,
	expectedLinkMigrations []testCapConsLinkMigration,
	expectedCyclicLinkErrors []CyclicLinkError,
	expectedMissingTargets []interpreter.AddressPath,
	expectedEvents []string,
) {
	require.True(t,
		len(expectedLinkMigrations) == 0 ||
			(len(expectedCyclicLinkErrors) == 0 && len(expectedMissingTargets) == 0),
	)

	// language=cadence
	contract := `
      access(all)
      contract Test {

          access(all)
          resource R {}

          access(all)
          struct S {}
      }
    `

	rt := NewTestInterpreterRuntime()

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

	// Create and store path and account links

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	storeTestPathLinks(t, pathLinks, storage, inter)

	storeTestAccountLinks(accountLinks, storage, inter)

	err = storage.Commit(inter, false)
	require.NoError(t, err)

	// Migrate

	migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := &testMigrationReporter{}

	capabilityMapping := &PathCapabilityMapping{}

	handler := &testCapConHandler{}

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&LinkValueMigration{
				CapabilityMapping: capabilityMapping,
				IssueHandler:      handler,
				Handler:           handler,
				Reporter:          reporter,
			},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Equal(t,
		expectedMigrations,
		reporter.migrations,
	)
	assert.Equal(t,
		expectedLinkMigrations,
		reporter.linkMigrations,
	)
	assert.Equal(t,
		expectedCyclicLinkErrors,
		reporter.cyclicLinkErrors,
	)
	assert.Equal(t,
		expectedMissingTargets,
		reporter.missingTargets,
	)
	require.Equal(t,
		expectedErrors,
		reporter.errors,
	)

	err = storage.CheckHealth()
	require.NoError(t, err)

	require.Equal(t,
		expectedEvents,
		nonDeploymentEventStrings(handler.events),
	)
}

func TestLinkMigration(t *testing.T) {

	t.Parallel()

	type linkTestCase struct {
		name                     string
		pathLinks                []testLink
		accountLinks             []interpreter.PathValue
		expectedMigrations       []testMigration
		expectedErrors           []error
		expectedLinkMigrations   []testCapConsLinkMigration
		expectedCyclicLinkErrors []CyclicLinkError
		expectedMissingTargets   []interpreter.AddressPath
		expectedEvents           []string
	}

	linkTestCases := []linkTestCase{
		{
			name: "Path links, working chain (public -> private -> storage)",
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/public/test, target: /private/test)
				{
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
				// Equivalent to:
				//   link<&Test.R>(/private/test, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
			},
			expectedLinkMigrations: []testCapConsLinkMigration{
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 1,
				},
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPublic,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 2,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
				`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
			},
		},
		{
			name: "Path links, working chain (public -> storage)",
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/public/test, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
			},
			expectedLinkMigrations: []testCapConsLinkMigration{
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPublic,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
			},
		},
		{
			name: "Path links, working chain (private -> storage)",
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/private/test, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
			},
			expectedLinkMigrations: []testCapConsLinkMigration{
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
			},
		},
		// Test that the migration also follows capability controller,
		// which were already previously migrated from links.
		// Following the (capability value) should not borrow it,
		// i.e. require the storage target to exist,
		// but rather just get the storage target
		{
			name: "Path links, working chain (private -> private -> storage)",
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/private/test, target: /private/test2)
				{
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
				// Equivalent to:
				//   link<&Test.R>(/private/test2, target: /storage/test)
				{
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
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey("test2"),
					migration:     "LinkValueMigration",
				},
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
			},
			expectedLinkMigrations: []testCapConsLinkMigration{
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPrivate,
							Identifier: "test2",
						},
					},
					capabilityID: 1,
				},
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 2,
				},
			},
			expectedEvents: []string{
				`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
				`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/test)`,
			},
		},
		{
			name: "Path links, cyclic chain (public -> private -> public)",
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/public/test, target: /private/test)
				{
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
				// Equivalent to:
				//   link<&Test.R>(/private/test, target: /public/test)
				{
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
			expectedCyclicLinkErrors: []CyclicLinkError{
				{
					Address: testAddress,
					Paths: []interpreter.PathValue{
						{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
						{
							Domain:     common.PathDomainPublic,
							Identifier: testPathIdentifier,
						},
						{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
					},
				},
				{
					Address: testAddress,
					Paths: []interpreter.PathValue{
						{
							Domain:     common.PathDomainPublic,
							Identifier: testPathIdentifier,
						},
						{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
						{
							Domain:     common.PathDomainPublic,
							Identifier: testPathIdentifier,
						},
					},
				},
			},
			expectedEvents: []string{},
		},
		{
			name: "Path links, missing target (public -> private)",
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Test.R>(/public/test, target: /private/test)
				{
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
			expectedMissingTargets: []interpreter.AddressPath{
				{
					Address: testAddress,
					Path: interpreter.PathValue{
						Identifier: testPathIdentifier,
						Domain:     common.PathDomainPublic,
					},
				},
			},
			expectedEvents: []string{},
		},
		{
			name: "Account link, working chain (public)",
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/public/test)
				{
					Domain:     common.PathDomainPublic,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
			},
			expectedLinkMigrations: []testCapConsLinkMigration{
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPublic,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())`,
			},
		},
		{
			name: "Account link, working chain (private)",
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/private/test)
				{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
			},
			expectedLinkMigrations: []testCapConsLinkMigration{
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 1,
				},
			},
			expectedEvents: []string{
				`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())`,
			},
		},
		{
			name: "Account link, working chain (public -> private), unauthorized",
			pathLinks: []testLink{
				// Equivalent to:
				//   link<&Account>(/public/test, target: /private/test)
				{
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPublic,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					borrowType: unauthorizedAccountReferenceStaticType,
				},
			},
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/private/test)
				{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
			},
			expectedLinkMigrations: []testCapConsLinkMigration{
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 1,
				},
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPublic,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 2,
				},
			},
			expectedEvents: []string{
				`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())`,
				`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
			},
		},
		{
			name: "Account link, working chain (public -> private), authorized",
			pathLinks: []testLink{
				// Equivalent to:
				//   link<auth(Capabilities, Contracts, Inbox, Keys, Storage) &Account>(
				//       /public/test,
				//       target: /private/test
				//   )
				{
					sourcePath: interpreter.PathValue{
						Domain:     common.PathDomainPublic,
						Identifier: testPathIdentifier,
					},
					targetPath: interpreter.PathValue{
						Domain:     common.PathDomainPrivate,
						Identifier: testPathIdentifier,
					},
					borrowType: interpreter.NewReferenceStaticType(
						nil,
						interpreter.FullyEntitledAccountAccess,
						interpreter.PrimitiveStaticTypeAccount,
					),
				},
			},
			accountLinks: []interpreter.PathValue{
				// Equivalent to:
				//   linkAccount(/private/test)
				{
					Domain:     common.PathDomainPrivate,
					Identifier: testPathIdentifier,
				},
			},
			expectedMigrations: []testMigration{
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPrivate.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
				{
					storageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainPublic.Identifier(),
					},
					storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
					migration:     "LinkValueMigration",
				},
			},
			expectedLinkMigrations: []testCapConsLinkMigration{
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPrivate,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 1,
				},
				{
					accountAddressPath: interpreter.AddressPath{
						Address: testAddress,
						Path: interpreter.PathValue{
							Domain:     common.PathDomainPublic,
							Identifier: testPathIdentifier,
						},
					},
					capabilityID: 2,
				},
			},
			expectedEvents: []string{
				`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())`,
				`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<auth(Capabilities,Contracts,Inbox,Keys,Storage)&Account>())`,
			},
		},
	}

	test := func(linkTestCase linkTestCase) {

		t.Run(linkTestCase.name, func(t *testing.T) {
			t.Parallel()

			testLinkMigration(
				t,
				linkTestCase.pathLinks,
				linkTestCase.accountLinks,
				linkTestCase.expectedMigrations,
				linkTestCase.expectedErrors,
				linkTestCase.expectedLinkMigrations,
				linkTestCase.expectedCyclicLinkErrors,
				linkTestCase.expectedMissingTargets,
				linkTestCase.expectedEvents,
			)
		})
	}

	for _, linkTestCase := range linkTestCases {
		test(linkTestCase)
	}
}

func TestPublishedPathCapabilityValueMigration(t *testing.T) {

	t.Parallel()

	// Equivalent to: &Int
	borrowType := interpreter.NewReferenceStaticType(
		nil,
		interpreter.UnauthorizedAccess,
		interpreter.PrimitiveStaticTypeInt,
	)

	// Equivalent to: getCapability<&Int>(/public/test)
	capabilityValue := interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
		borrowType,
		interpreter.AddressValue(testAddress),
		interpreter.PathValue{
			Domain:     common.PathDomainPublic,
			Identifier: testPathIdentifier,
		},
	)

	pathLinks := []testLink{
		// Equivalent to:
		//   link<&Int>(/public/test, target: /private/test)
		{
			sourcePath: interpreter.PathValue{
				Domain:     common.PathDomainPublic,
				Identifier: testPathIdentifier,
			},
			targetPath: interpreter.PathValue{
				Domain:     common.PathDomainPrivate,
				Identifier: testPathIdentifier,
			},
			borrowType: borrowType,
		},
		// Equivalent to:
		//   link<&Int>(/private/test, target: /storage/test)
		{
			sourcePath: interpreter.PathValue{
				Domain:     common.PathDomainPrivate,
				Identifier: testPathIdentifier,
			},
			targetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: testPathIdentifier,
			},
			borrowType: borrowType,
		},
	}

	expectedMigrations := []testMigration{
		{
			storageKey: interpreter.StorageKey{
				Address: testAddress,
				Key:     common.PathDomainPrivate.Identifier(),
			},
			storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
			migration:     "LinkValueMigration",
		},
		{
			storageKey: interpreter.StorageKey{
				Address: testAddress,
				Key:     common.PathDomainPublic.Identifier(),
			},
			storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
			migration:     "LinkValueMigration",
		},
		{
			storageKey: interpreter.StorageKey{
				Address: testAddress,
				Key:     stdlib.InboxStorageDomain,
			},
			storageMapKey: interpreter.StringStorageMapKey("foo"),
			migration:     "CapabilityValueMigration",
		},
	}

	expectedPathMigrations := []testCapConsPathCapabilityMigration{
		{
			accountAddress: testAddress,
			addressPath: interpreter.AddressPath{
				Address: testAddress,
				Path: interpreter.NewUnmeteredPathValue(
					common.PathDomainPublic,
					testPathIdentifier,
				),
			},
			borrowType:   borrowType,
			capabilityID: 2,
		},
	}

	rt := NewTestInterpreterRuntime()

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{testAddress}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	// Setup

	setupTransactionLocation := nextTransactionLocation()

	environment := runtime.NewScriptInterpreterEnvironment(runtime.Config{})

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

	// Create and store path links
	{
		storage, inter, err := rt.Storage(runtime.Context{
			Interface: runtimeInterface,
		})
		require.NoError(t, err)

		storeTestPathLinks(t, pathLinks, storage, inter)

		err = storage.Commit(inter, false)
		require.NoError(t, err)
	}

	// Save capability values into account

	// language=cadence
	setupTx := `
      transaction {
          prepare(signer: auth(PublishInboxCapability) &Account) {
             signer.inbox.publish(cap, name: "foo", recipient: 0x2)
          }
      }
    `

	err := rt.ExecuteTransaction(
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

	// Create new runtime.Storage for migration.
	// WARNING: don't reuse old storage (created for storing path links)
	// because it has outdated cache after ExecuteTransaction() commits new data
	// to underlying ledger.
	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := &testMigrationReporter{}

	privatePublicCapabilityMapping := &PathCapabilityMapping{}
	storageCapabilityMapping := &PathTypeCapabilityMapping{}

	handler := &testCapConHandler{}

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&LinkValueMigration{
				CapabilityMapping: privatePublicCapabilityMapping,
				IssueHandler:      handler,
				Handler:           handler,
				Reporter:          reporter,
			},
		),
	)

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&CapabilityValueMigration{
				PrivatePublicCapabilityMapping: privatePublicCapabilityMapping,
				TypedStorageCapabilityMapping:  storageCapabilityMapping,
				Reporter:                       reporter,
			},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Equal(t,
		expectedMigrations,
		reporter.migrations,
	)
	assert.Equal(t,
		expectedPathMigrations,
		reporter.pathCapabilityMigrations,
	)
	require.Nil(t, reporter.missingCapabilityIDs)

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	require.Equal(t,
		[]string{
			`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Int>(), path: /storage/test)`,
			`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Int>(), path: /storage/test)`,
		},
		nonDeploymentEventStrings(handler.events),
	)

	// language=cadence
	checkScript := `
	  access(all)
	  fun main() {
	      getAuthAccount<auth(ClaimInboxCapability) &Account>(0x2)
	          .inbox.claim<&Int>("foo", provider: 0x1)!
	  }
	`

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

func TestUntypedPathCapabilityValueMigration(t *testing.T) {

	t.Parallel()

	// Equivalent to: &Int
	linkBorrowType := interpreter.NewReferenceStaticType(
		nil,
		interpreter.UnauthorizedAccess,
		interpreter.PrimitiveStaticTypeInt,
	)

	// Equivalent to: getCapability(/public/test)
	capabilityValue := interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
		// NOTE: no borrow type
		nil,
		interpreter.AddressValue(testAddress),
		interpreter.PathValue{
			Domain:     common.PathDomainPublic,
			Identifier: testPathIdentifier,
		},
	)

	pathLinks := []testLink{
		// Equivalent to:
		//   link<&Int>(/public/test, target: /private/test)
		{
			sourcePath: interpreter.PathValue{
				Domain:     common.PathDomainPublic,
				Identifier: testPathIdentifier,
			},
			targetPath: interpreter.PathValue{
				Domain:     common.PathDomainPrivate,
				Identifier: testPathIdentifier,
			},
			borrowType: linkBorrowType,
		},
		// Equivalent to:
		//   link<&Int>(/private/test, target: /storage/test)
		{
			sourcePath: interpreter.PathValue{
				Domain:     common.PathDomainPrivate,
				Identifier: testPathIdentifier,
			},
			targetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: testPathIdentifier,
			},
			borrowType: linkBorrowType,
		},
	}

	expectedMigrations := []testMigration{
		{
			storageKey: interpreter.StorageKey{
				Address: testAddress,
				Key:     common.PathDomainPrivate.Identifier(),
			},
			storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
			migration:     "LinkValueMigration",
		},
		{
			storageKey: interpreter.StorageKey{
				Address: testAddress,
				Key:     common.PathDomainPublic.Identifier(),
			},
			storageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
			migration:     "LinkValueMigration",
		},
		{
			storageKey: interpreter.StorageKey{
				Address: testAddress,
				Key:     common.PathDomainStorage.Identifier(),
			},
			storageMapKey: interpreter.StringStorageMapKey("cap"),
			migration:     "CapabilityValueMigration",
		},
	}

	expectedPathMigrations := []testCapConsPathCapabilityMigration{
		{
			accountAddress: testAddress,
			addressPath: interpreter.AddressPath{
				Address: testAddress,
				Path: interpreter.NewUnmeteredPathValue(
					common.PathDomainPublic,
					testPathIdentifier,
				),
			},
			// NOTE: link / cap con's borrow type is used
			borrowType:   linkBorrowType,
			capabilityID: 2,
		},
	}

	rt := NewTestInterpreterRuntime()

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{testAddress}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	// Setup

	setupTransactionLocation := nextTransactionLocation()

	environment := runtime.NewScriptInterpreterEnvironment(runtime.Config{})

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

	// Create and store path links
	{
		storage, inter, err := rt.Storage(runtime.Context{
			Interface: runtimeInterface,
		})
		require.NoError(t, err)

		storeTestPathLinks(t, pathLinks, storage, inter)

		err = storage.Commit(inter, false)
		require.NoError(t, err)
	}

	// Save capability values into account

	// language=cadence
	setupTx := `
      transaction {
          prepare(signer: auth(SaveValue) &Account) {
             signer.storage.save(42, to: /storage/test)
             signer.storage.save(cap, to: /storage/cap)
          }
      }
    `

	err := rt.ExecuteTransaction(
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

	// Create new runtime.Storage for migration.
	// WARNING: don't reuse old storage (created for storing path links)
	// because it has outdated cache after ExecuteTransaction() commits new data
	// to underlying ledger.
	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := &testMigrationReporter{}

	privatePublicCapabilityMapping := &PathCapabilityMapping{}
	storageCapabilityMapping := &PathTypeCapabilityMapping{}

	handler := &testCapConHandler{}

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&LinkValueMigration{
				CapabilityMapping: privatePublicCapabilityMapping,
				IssueHandler:      handler,
				Handler:           handler,
				Reporter:          reporter,
			},
		),
	)

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&CapabilityValueMigration{
				PrivatePublicCapabilityMapping: privatePublicCapabilityMapping,
				TypedStorageCapabilityMapping:  storageCapabilityMapping,
				Reporter:                       reporter,
			},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Equal(t,
		expectedMigrations,
		reporter.migrations,
	)
	assert.Equal(t,
		expectedPathMigrations,
		reporter.pathCapabilityMigrations,
	)
	require.Nil(t, reporter.missingCapabilityIDs)

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	require.Equal(t,
		[]string{
			`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Int>(), path: /storage/test)`,
			`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Int>(), path: /storage/test)`,
		},
		nonDeploymentEventStrings(handler.events),
	)

	// Check

	// language=cadence
	checkScript := `
	  access(all)
	  fun main() {
	      let cap = getAuthAccount<auth(CopyValue) &Account>(0x1)
	          .storage.copy<Capability>(from: /storage/cap)!
          assert(*cap.borrow<&Int>()! == 42)
	  }
	`

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

func TestCanSkipCapabilityValueMigration(t *testing.T) {

	t.Parallel()

	testCases := map[interpreter.StaticType]bool{

		// Primitive types, like Bool and Address

		interpreter.PrimitiveStaticTypeBool:    true,
		interpreter.PrimitiveStaticTypeAddress: true,

		// Number and Path types, like UInt8 and StoragePath

		interpreter.PrimitiveStaticTypeUInt8:       true,
		interpreter.PrimitiveStaticTypeStoragePath: true,

		// Capability types

		interpreter.PrimitiveStaticTypeCapability: false,
		&interpreter.CapabilityStaticType{
			BorrowType: interpreter.PrimitiveStaticTypeString,
		}: false,
		&interpreter.CapabilityStaticType{
			BorrowType: interpreter.PrimitiveStaticTypeCharacter,
		}: false,

		// Existential types, like AnyStruct and AnyResource

		interpreter.PrimitiveStaticTypeAnyStruct:   false,
		interpreter.PrimitiveStaticTypeAnyResource: false,
	}

	test := func(ty interpreter.StaticType, expected bool) {

		t.Run(ty.String(), func(t *testing.T) {

			t.Parallel()

			t.Run("base", func(t *testing.T) {

				t.Parallel()

				actual := CanSkipCapabilityValueMigration(ty)
				assert.Equal(t, expected, actual)

			})

			t.Run("optional", func(t *testing.T) {

				t.Parallel()

				optionalType := interpreter.NewOptionalStaticType(nil, ty)

				actual := CanSkipCapabilityValueMigration(optionalType)
				assert.Equal(t, expected, actual)
			})

			t.Run("variable-sized", func(t *testing.T) {

				t.Parallel()

				arrayType := interpreter.NewVariableSizedStaticType(nil, ty)

				actual := CanSkipCapabilityValueMigration(arrayType)
				assert.Equal(t, expected, actual)
			})

			t.Run("constant-sized", func(t *testing.T) {

				t.Parallel()

				arrayType := interpreter.NewConstantSizedStaticType(nil, ty, 2)

				actual := CanSkipCapabilityValueMigration(arrayType)
				assert.Equal(t, expected, actual)
			})

			t.Run("dictionary key", func(t *testing.T) {

				t.Parallel()

				dictionaryType := interpreter.NewDictionaryStaticType(
					nil,
					ty,
					interpreter.PrimitiveStaticTypeInt,
				)

				actual := CanSkipCapabilityValueMigration(dictionaryType)
				assert.Equal(t, expected, actual)

			})

			t.Run("dictionary value", func(t *testing.T) {

				t.Parallel()

				dictionaryType := interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeInt,
					ty,
				)

				actual := CanSkipCapabilityValueMigration(dictionaryType)
				assert.Equal(t, expected, actual)
			})
		})
	}

	for ty, expected := range testCases {
		test(ty, expected)
	}
}

func TestStorageCapMigration(t *testing.T) {
	t.Parallel()

	// &String
	testBorrowType1 := interpreter.NewReferenceStaticType(
		nil,
		interpreter.UnauthorizedAccess,
		interpreter.PrimitiveStaticTypeString,
	)

	// &Int
	testBorrowType2 := interpreter.NewReferenceStaticType(
		nil,
		interpreter.UnauthorizedAccess,
		interpreter.PrimitiveStaticTypeInt,
	)

	testPath := interpreter.PathValue{
		Domain:     common.PathDomainStorage,
		Identifier: testPathIdentifier,
	}

	testAddressPath := interpreter.AddressPath{
		Address: testAddress,
		Path:    testPath,
	}

	// Store 3 capabilities:
	// - all target the same storage path
	// - the first has a different borrow type than the last two
	// - the last two have the same borrow type

	// Equivalent to: getCapability<&String>(/storage/test)
	capabilityValue1 := interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
		testBorrowType1,
		interpreter.AddressValue(testAddress),
		testPath,
	)

	// Equivalent to: getCapability<&Int>(/storage/test)
	capabilityValue2 := interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
		testBorrowType2,
		interpreter.AddressValue(testAddress),
		testPath,
	)

	// Equivalent to: getCapability<&Int>(/storage/test)
	capabilityValue3 := interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
		testBorrowType2,
		interpreter.AddressValue(testAddress),
		testPath,
	)

	rt := NewTestInterpreterRuntime()

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{testAddress}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Setup

	setupTransactionLocation := nextTransactionLocation()

	environment := runtime.NewScriptInterpreterEnvironment(runtime.Config{})

	// Inject the path capability values.
	//
	// We don't have a way to create a path capability value in a Cadence program anymore,
	// so we have to inject it manually.

	for i, capabilityValue := range []*interpreter.PathCapabilityValue{ //nolint:staticcheck
		capabilityValue1,
		capabilityValue2,
		capabilityValue3,
	} {
		environment.DeclareValue(
			stdlib.StandardLibraryValue{
				Name:  fmt.Sprintf("cap%d", i+1),
				Type:  &sema.CapabilityType{},
				Kind:  common.DeclarationKindConstant,
				Value: capabilityValue,
			},
			setupTransactionLocation,
		)
	}

	// Save capability value into account

	// language=cadence
	setupTx := `
      transaction {
          prepare(signer: auth(SaveValue) &Account) {
             signer.storage.save(cap1, to: /storage/cap1)
             signer.storage.save(cap2, to: /storage/cap2)
             signer.storage.save(cap3, to: /storage/cap3)
          }
      }
    `

	err := rt.ExecuteTransaction(
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

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := &testMigrationReporter{}
	handler := &testCapConHandler{}
	storageDomainCapabilities := &AccountsCapabilities{}
	typedCapabilityMapping := &PathTypeCapabilityMapping{}
	untypedCapabilityMapping := &PathCapabilityMapping{}

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&StorageCapMigration{
				StorageDomainCapabilities: storageDomainCapabilities,
			},
		),
	)

	storageCapabilities := storageDomainCapabilities.Get(testAddress)
	require.NotNil(t, storageCapabilities)

	IssueAccountCapabilities(
		inter,
		storage,
		reporter,
		testAddress,
		storageCapabilities,
		handler,
		typedCapabilityMapping,
		untypedCapabilityMapping,
		func(_ interpreter.StaticType) interpreter.Authorization {
			return interpreter.UnauthorizedAccess
		},
	)

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&CapabilityValueMigration{
				TypedStorageCapabilityMapping:   typedCapabilityMapping,
				UntypedStorageCapabilityMapping: untypedCapabilityMapping,
				Reporter:                        reporter,
			},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Empty(t, reporter.errors)

	testStorageKey := interpreter.StorageKey{
		Address: testAddress,
		Key:     common.PathDomainStorage.Identifier(),
	}

	assert.Equal(t,
		[]testMigration{
			{
				storageKey:    testStorageKey,
				storageMapKey: interpreter.StringStorageMapKey("cap1"),
				migration:     "CapabilityValueMigration",
			},
			{
				storageKey:    testStorageKey,
				storageMapKey: interpreter.StringStorageMapKey("cap3"),
				migration:     "CapabilityValueMigration",
			},
			{
				storageKey:    testStorageKey,
				storageMapKey: interpreter.StringStorageMapKey("cap2"),
				migration:     "CapabilityValueMigration",
			},
		},
		reporter.migrations,
	)

	assert.Equal(t,
		[]testStorageCapConIssued{
			{
				accountAddress: testAddress,
				addressPath:    testAddressPath,
				capabilityID:   1,
				borrowType:     testBorrowType1,
			},
			{
				accountAddress: testAddress,
				addressPath:    testAddressPath,
				capabilityID:   2,
				borrowType:     testBorrowType2,
			},
		},
		reporter.issuedStorageCapCons,
	)

	assert.Equal(t,
		[]testCapConsPathCapabilityMigration{
			{
				accountAddress: testAddress,
				addressPath:    testAddressPath,
				borrowType:     testBorrowType1,
				capabilityID:   1,
			},
			{
				accountAddress: testAddress,
				addressPath:    testAddressPath,
				borrowType:     testBorrowType2,
				capabilityID:   2,
			},
			{
				accountAddress: testAddress,
				addressPath:    testAddressPath,
				borrowType:     testBorrowType2,
				capabilityID:   2,
			},
		},
		reporter.pathCapabilityMigrations,
	)

	err = storage.CheckHealth()
	require.NoError(t, err)

	type actual struct {
		address    common.Address
		capability AccountCapability
	}

	var actuals []actual

	storageDomainCapabilities.ForEach(
		testAddress,
		func(accountCapability AccountCapability) bool {
			actuals = append(
				actuals,
				actual{
					address:    testAddress,
					capability: accountCapability,
				},
			)
			return true
		},
	)

	assert.Equal(t,
		[]actual{
			{
				address: testAddress,
				capability: AccountCapability{
					TargetPath: testPath,
					BorrowType: testBorrowType1,
					StoredPath: Path{
						Domain: "storage",
						Path:   "cap1",
					},
				},
			},
			{
				address: testAddress,
				capability: AccountCapability{
					TargetPath: testPath,
					BorrowType: testBorrowType2,
					StoredPath: Path{
						Domain: "storage",
						Path:   "cap3",
					},
				},
			},
			{
				address: testAddress,
				capability: AccountCapability{
					TargetPath: testPath,
					BorrowType: testBorrowType2,
					StoredPath: Path{
						Domain: "storage",
						Path:   "cap2",
					},
				},
			},
		},
		actuals,
	)

}

func TestUntypedStorageCapMigration(t *testing.T) {
	t.Parallel()

	testPath := interpreter.PathValue{
		Domain:     common.PathDomainStorage,
		Identifier: testPathIdentifier,
	}

	testAddressPath := interpreter.AddressPath{
		Address: testAddress,
		Path:    testPath,
	}

	capabilityValue := interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
		// Borrow type must be nil.
		nil,
		interpreter.AddressValue(testAddress),
		testPath,
	)

	rt := NewTestInterpreterRuntime()

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{testAddress}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Setup

	setupTransactionLocation := nextTransactionLocation()

	environment := runtime.NewScriptInterpreterEnvironment(runtime.Config{})

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

	// Save capability value into account

	// language=cadence
	setupTx := `
      transaction {
          prepare(signer: auth(SaveValue) &Account) {
             signer.storage.save("Target value", to: /storage/test)
             signer.storage.save(cap, to: /storage/cap)
          }
      }
    `

	err := rt.ExecuteTransaction(
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

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := &testMigrationReporter{}
	typedStorageCapabilityMapping := &PathTypeCapabilityMapping{}
	untypedStorageCapabilityMapping := &PathCapabilityMapping{}
	handler := &testCapConHandler{
		// Avoid loading contract code for made-up entitlement
		dropEvents: true,
	}
	storageDomainCapabilities := &AccountsCapabilities{}

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&StorageCapMigration{
				StorageDomainCapabilities: storageDomainCapabilities,
			},
		),
	)

	storageCapabilities := storageDomainCapabilities.Get(testAddress)
	require.NotNil(t, storageCapabilities)

	inferredAuth := interpreter.NewEntitlementSetAuthorization(
		nil,
		func() []common.TypeID {
			return []common.TypeID{
				common.AddressLocation{
					Address: testAddress,
					Name:    "Foo",
				}.TypeID(nil, "Bar"),
			}
		},
		1,
		sema.Conjunction,
	)

	IssueAccountCapabilities(
		inter,
		storage,
		reporter,
		testAddress,
		storageCapabilities,
		handler,
		typedStorageCapabilityMapping,
		untypedStorageCapabilityMapping,
		func(_ interpreter.StaticType) interpreter.Authorization {
			return inferredAuth
		},
	)

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&CapabilityValueMigration{
				PrivatePublicCapabilityMapping:  &PathCapabilityMapping{},
				TypedStorageCapabilityMapping:   typedStorageCapabilityMapping,
				UntypedStorageCapabilityMapping: untypedStorageCapabilityMapping,
				Reporter:                        reporter,
			},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Equal(
		t,
		[]testMigration{
			{
				storageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     common.PathDomainStorage.Identifier(),
				},
				storageMapKey: interpreter.StringStorageMapKey("cap"),
				migration:     "CapabilityValueMigration",
			},
		},
		reporter.migrations,
	)

	require.Empty(t, reporter.errors)
	require.Empty(t, reporter.missingCapabilityIDs)

	require.Equal(
		t,
		[]testStorageCapConsMissingBorrowType{
			{
				targetPath: testAddressPath,
				storedPath: interpreter.AddressPath{
					Address: testAddress,
					Path: interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "cap",
					},
				},
			},
		},
		reporter.missingStorageCapConBorrowTypes,
	)

	require.Equal(
		t,
		[]testStorageCapConIssued{
			{
				accountAddress: testAddress,
				addressPath:    testAddressPath,
				borrowType: interpreter.NewReferenceStaticType(
					nil,
					inferredAuth,
					interpreter.PrimitiveStaticTypeString,
				),
				capabilityID: 1,
			},
		},
		reporter.issuedStorageCapCons,
	)

	require.Equal(
		t,
		[]testStorageCapConsInferredBorrowType{
			{
				targetPath: testAddressPath,
				borrowType: interpreter.NewReferenceStaticType(
					nil,
					inferredAuth,
					interpreter.PrimitiveStaticTypeString,
				),
				storedPath: interpreter.AddressPath{
					Address: testAddress,
					Path: interpreter.PathValue{
						Identifier: "cap",
						Domain:     common.PathDomainStorage,
					},
				},
			},
		},
		reporter.inferredStorageCapConBorrowTypes,
	)

	err = storage.CheckHealth()
	require.NoError(t, err)

	type actual struct {
		address    common.Address
		capability AccountCapability
	}

	var actuals []actual

	storageDomainCapabilities.ForEach(
		testAddress,
		func(accountCapability AccountCapability) bool {
			actuals = append(
				actuals,
				actual{
					address:    testAddress,
					capability: accountCapability,
				},
			)
			return true
		},
	)

	assert.Equal(t,
		[]actual{
			{
				address: testAddress,
				capability: AccountCapability{
					TargetPath: testPath,
					StoredPath: Path{
						Domain: "storage",
						Path:   "cap",
					},
				},
			},
		},
		actuals,
	)
}

func TestUntypedStorageCapWithMissingTargetMigration(t *testing.T) {
	t.Parallel()

	addressA := common.MustBytesToAddress([]byte{0x1})
	addressB := common.MustBytesToAddress([]byte{0x2})

	targetPath := interpreter.PathValue{
		Domain:     common.PathDomainStorage,
		Identifier: testPathIdentifier,
	}

	// Capability targets `addressB`.
	// Capability itself is stored in `addressA`

	capabilityValue := interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
		// Borrow type must be nil.
		nil,
		interpreter.AddressValue(addressB),
		targetPath,
	)

	rt := NewTestInterpreterRuntime()

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{addressA}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Setup

	setupTransactionLocation := nextTransactionLocation()

	environment := runtime.NewScriptInterpreterEnvironment(runtime.Config{})

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

	// Save capability value into account

	// language=cadence
	setupTx := `
      transaction {
          prepare(signer: auth(SaveValue) &Account) {
             // There is no value stored at the target path
             signer.storage.save(cap, to: /storage/cap)
          }
      }
    `

	err := rt.ExecuteTransaction(
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

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	migration, err := migrations.NewStorageMigration(inter, storage, "test", addressA)
	require.NoError(t, err)

	reporter := &testMigrationReporter{}
	typedStorageCapabilityMapping := &PathTypeCapabilityMapping{}
	untypedStorageCapabilityMapping := &PathCapabilityMapping{}
	handler := &testCapConHandler{}
	storageDomainCapabilities := &AccountsCapabilities{}

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&StorageCapMigration{
				StorageDomainCapabilities: storageDomainCapabilities,
			},
		),
	)

	storageCapabilities := storageDomainCapabilities.Get(addressB)
	require.NotNil(t, storageCapabilities)

	IssueAccountCapabilities(
		inter,
		storage,
		reporter,
		addressA,
		storageCapabilities,
		handler,
		typedStorageCapabilityMapping,
		untypedStorageCapabilityMapping,
		func(_ interpreter.StaticType) interpreter.Authorization {
			return interpreter.UnauthorizedAccess
		},
	)

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			&CapabilityValueMigration{
				PrivatePublicCapabilityMapping:  &PathCapabilityMapping{},
				TypedStorageCapabilityMapping:   typedStorageCapabilityMapping,
				UntypedStorageCapabilityMapping: untypedStorageCapabilityMapping,
				Reporter:                        reporter,
			},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Empty(t, reporter.migrations)
	require.Empty(t, reporter.errors)
	require.Empty(t, reporter.issuedStorageCapCons)
	require.Empty(t, reporter.inferredStorageCapConBorrowTypes)

	require.Equal(
		t,
		[]testCapConsMissingCapabilityID{
			{
				accountAddress: addressA,
				addressPath: interpreter.AddressPath{
					Address: addressB,
					Path:    targetPath,
				},
			},
		},
		reporter.missingCapabilityIDs,
	)

	require.Equal(
		t,
		[]testStorageCapConsMissingBorrowType{
			{
				targetPath: interpreter.AddressPath{
					Address: addressA,
					Path:    targetPath,
				},
				storedPath: interpreter.AddressPath{
					Address: testAddress,
					Path: interpreter.PathValue{
						Identifier: "cap",
						Domain:     common.PathDomainStorage,
					},
				},
			},
		},
		reporter.missingStorageCapConBorrowTypes,
	)

	err = storage.CheckHealth()
	require.NoError(t, err)

	type actual struct {
		address    common.Address
		capability AccountCapability
	}

	var actuals []actual

	storageDomainCapabilities.ForEach(
		addressB,
		func(accountCapability AccountCapability) bool {
			actuals = append(
				actuals,
				actual{
					address:    addressA,
					capability: accountCapability,
				},
			)
			return true
		},
	)

	assert.Equal(t,
		[]actual{
			{
				address: addressA,
				capability: AccountCapability{
					TargetPath: targetPath,
					StoredPath: Path{
						Domain: "storage",
						Path:   "cap",
					},
				},
			},
		},
		actuals,
	)
}
