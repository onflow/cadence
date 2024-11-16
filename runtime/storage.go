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

package runtime

import (
	"fmt"
	"runtime"
	"sort"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

const (
	AccountStorageKey = "stored"
)

type StorageConfig struct {
	StorageFormatV2Enabled bool
}

type Storage struct {
	*atree.PersistentSlabStorage

	// cachedDomainStorageMaps is a cache of domain storage maps.
	// Key is StorageKey{address, domain} and value is domain storage map.
	cachedDomainStorageMaps map[interpreter.StorageDomainKey]*interpreter.DomainStorageMap

	// cachedV1Accounts contains the cached result of determining
	// if the account is in storage format v1 or not.
	cachedV1Accounts map[common.Address]bool

	// contractUpdates is a cache of contract updates.
	// Key is StorageKey{contract_address, contract_name} and value is contract composite value.
	contractUpdates *orderedmap.OrderedMap[interpreter.StorageKey, *interpreter.CompositeValue]

	Ledger atree.Ledger

	memoryGauge common.MemoryGauge

	Config StorageConfig

	AccountStorageV1      *AccountStorageV1
	AccountStorageV2      *AccountStorageV2
	scheduledV2Migrations []common.Address
}

var _ atree.SlabStorage = &Storage{}
var _ interpreter.Storage = &Storage{}

func NewStorage(
	ledger atree.Ledger,
	memoryGauge common.MemoryGauge,
	config StorageConfig,
) *Storage {
	decodeStorable := func(
		decoder *cbor.StreamDecoder,
		slabID atree.SlabID,
		inlinedExtraData []atree.ExtraData,
	) (
		atree.Storable,
		error,
	) {
		return interpreter.DecodeStorable(
			decoder,
			slabID,
			inlinedExtraData,
			memoryGauge,
		)
	}

	decodeTypeInfo := func(decoder *cbor.StreamDecoder) (atree.TypeInfo, error) {
		return interpreter.DecodeTypeInfo(decoder, memoryGauge)
	}

	ledgerStorage := atree.NewLedgerBaseStorage(ledger)
	persistentSlabStorage := atree.NewPersistentSlabStorage(
		ledgerStorage,
		interpreter.CBOREncMode,
		interpreter.CBORDecMode,
		decodeStorable,
		decodeTypeInfo,
	)

	accountStorageV1 := NewAccountStorageV1(
		ledger,
		persistentSlabStorage,
		memoryGauge,
	)

	var accountStorageV2 *AccountStorageV2
	if config.StorageFormatV2Enabled {
		accountStorageV2 = NewAccountStorageV2(
			ledger,
			persistentSlabStorage,
			memoryGauge,
		)
	}

	return &Storage{
		Ledger:                ledger,
		PersistentSlabStorage: persistentSlabStorage,
		memoryGauge:           memoryGauge,
		Config:                config,
		AccountStorageV1:      accountStorageV1,
		AccountStorageV2:      accountStorageV2,
	}
}

const storageIndexLength = 8

// GetDomainStorageMap returns existing or new domain storage map for the given account and domain.
func (s *Storage) GetDomainStorageMap(
	inter *interpreter.Interpreter,
	address common.Address,
	domain common.StorageDomain,
	createIfNotExists bool,
) (
	domainStorageMap *interpreter.DomainStorageMap,
) {
	// Get cached domain storage map if it exists.

	domainStorageKey := interpreter.NewStorageDomainKey(s.memoryGauge, address, domain)

	if s.cachedDomainStorageMaps != nil {
		domainStorageMap = s.cachedDomainStorageMaps[domainStorageKey]
		if domainStorageMap != nil {
			return domainStorageMap
		}
	}

	defer func() {
		// Cache domain storage map
		if domainStorageMap != nil {
			s.cacheDomainStorageMap(
				domainStorageKey,
				domainStorageMap,
			)
		}
	}()

	if !s.Config.StorageFormatV2Enabled || s.IsV1Account(address) {
		domainStorageMap = s.AccountStorageV1.GetDomainStorageMap(
			address,
			domain,
			createIfNotExists,
		)

		if domainStorageMap != nil {
			s.cacheIsV1Account(address, true)
		}

	} else {
		domainStorageMap = s.AccountStorageV2.GetDomainStorageMap(
			inter,
			address,
			domain,
			createIfNotExists,
		)
	}

	return domainStorageMap
}

// IsV1Account returns true if given account is in account storage format v1.
func (s *Storage) IsV1Account(address common.Address) (isV1 bool) {

	// Check cache

	if isV1, present := s.cachedV1Accounts[address]; present {
		return isV1
	}

	// Cache result

	defer func() {
		s.cacheIsV1Account(address, isV1)
	}()

	// First check if account storage map exists.
	// In that case the account was already migrated to account storage format v2,
	// and we do not need to check the domain storage map registers.

	accountStorageMapExists, err := hasAccountStorageMap(s.Ledger, address)
	if err != nil {
		panic(err)
	}
	if accountStorageMapExists {
		return false
	}

	// Check if a storage map register exists for any of the domains.
	// Check the most frequently used domains first, such as storage, public, private.
	for _, domain := range common.AllStorageDomains {
		_, domainExists, err := getSlabIndexFromRegisterValue(
			s.Ledger,
			address,
			[]byte(domain.Identifier()),
		)
		if err != nil {
			panic(err)
		}
		if domainExists {
			return true
		}
	}

	return false
}

func (s *Storage) cacheIsV1Account(address common.Address, isV1 bool) {
	if s.cachedV1Accounts == nil {
		s.cachedV1Accounts = map[common.Address]bool{}
	}
	s.cachedV1Accounts[address] = isV1
}

func (s *Storage) cacheDomainStorageMap(
	storageDomainKey interpreter.StorageDomainKey,
	domainStorageMap *interpreter.DomainStorageMap,
) {
	if s.cachedDomainStorageMaps == nil {
		s.cachedDomainStorageMaps = map[interpreter.StorageDomainKey]*interpreter.DomainStorageMap{}
	}

	s.cachedDomainStorageMaps[storageDomainKey] = domainStorageMap
}

// getSlabIndexFromRegisterValue returns register value as atree.SlabIndex.
// This function returns error if
// - underlying ledger panics, or
// - underlying ledger returns error when retrieving ledger value, or
// - retrieved ledger value is invalid (for atree.SlabIndex).
func getSlabIndexFromRegisterValue(
	ledger atree.Ledger,
	address common.Address,
	key []byte,
) (atree.SlabIndex, bool, error) {
	var data []byte
	var err error
	errors.WrapPanic(func() {
		data, err = ledger.GetValue(address[:], key)
	})
	if err != nil {
		return atree.SlabIndex{}, false, interpreter.WrappedExternalError(err)
	}

	dataLength := len(data)

	if dataLength == 0 {
		return atree.SlabIndex{}, false, nil
	}

	isStorageIndex := dataLength == storageIndexLength
	if !isStorageIndex {
		// Invalid data in register

		// TODO: add dedicated error type?
		return atree.SlabIndex{}, false, errors.NewUnexpectedError(
			"invalid storage index for storage map of account '%x': expected length %d, got %d",
			address[:], storageIndexLength, dataLength,
		)
	}

	return atree.SlabIndex(data), true, nil
}

func (s *Storage) recordContractUpdate(
	location common.AddressLocation,
	contractValue *interpreter.CompositeValue,
) {
	key := interpreter.NewStorageKey(s.memoryGauge, location.Address, location.Name)

	// NOTE: do NOT delete the map entry,
	// otherwise the removal write is lost

	if s.contractUpdates == nil {
		s.contractUpdates = &orderedmap.OrderedMap[interpreter.StorageKey, *interpreter.CompositeValue]{}
	}
	s.contractUpdates.Set(key, contractValue)
}

func (s *Storage) contractUpdateRecorded(
	location common.AddressLocation,
) bool {
	if s.contractUpdates == nil {
		return false
	}

	key := interpreter.NewStorageKey(s.memoryGauge, location.Address, location.Name)
	return s.contractUpdates.Contains(key)
}

type ContractUpdate struct {
	ContractValue *interpreter.CompositeValue
	Key           interpreter.StorageKey
}

func SortContractUpdates(updates []ContractUpdate) {
	sort.Slice(updates, func(i, j int) bool {
		a := updates[i].Key
		b := updates[j].Key
		return a.IsLess(b)
	})
}

// commitContractUpdates writes the contract updates to storage.
// The contract updates were delayed so they are not observable during execution.
func (s *Storage) commitContractUpdates(inter *interpreter.Interpreter) {
	if s.contractUpdates == nil {
		return
	}

	for pair := s.contractUpdates.Oldest(); pair != nil; pair = pair.Next() {
		s.writeContractUpdate(inter, pair.Key, pair.Value)
	}
}

func (s *Storage) writeContractUpdate(
	inter *interpreter.Interpreter,
	key interpreter.StorageKey,
	contractValue *interpreter.CompositeValue,
) {
	storageMap := s.GetDomainStorageMap(inter, key.Address, common.StorageDomainContract, true)
	// NOTE: pass nil instead of allocating a Value-typed  interface that points to nil
	storageMapKey := interpreter.StringStorageMapKey(key.Key)
	if contractValue == nil {
		storageMap.WriteValue(inter, storageMapKey, nil)
	} else {
		storageMap.WriteValue(inter, storageMapKey, contractValue)
	}
}

// Commit serializes/saves all values in the readCache in storage (through the runtime interface).
func (s *Storage) Commit(inter *interpreter.Interpreter, commitContractUpdates bool) error {
	return s.commit(inter, commitContractUpdates, true)
}

// Deprecated: NondeterministicCommit serializes and commits all values in the deltas storage
// in nondeterministic order.  This function is used when commit ordering isn't
// required (e.g. migration programs).
func (s *Storage) NondeterministicCommit(inter *interpreter.Interpreter, commitContractUpdates bool) error {
	return s.commit(inter, commitContractUpdates, false)
}

func (s *Storage) commit(inter *interpreter.Interpreter, commitContractUpdates bool, deterministic bool) error {

	if commitContractUpdates {
		s.commitContractUpdates(inter)
	}

	err := s.AccountStorageV1.commit()
	if err != nil {
		return err
	}

	if s.Config.StorageFormatV2Enabled {
		err = s.AccountStorageV2.commit()
		if err != nil {
			return err
		}

		err = s.migrateV1AccountsToV2(inter)
		if err != nil {
			return err
		}
	}

	// Commit the underlying slab storage's writes

	slabStorage := s.PersistentSlabStorage

	size := slabStorage.DeltasSizeWithoutTempAddresses()
	if size > 0 {
		inter.ReportComputation(common.ComputationKindEncodeValue, uint(size))
		usage := common.NewBytesMemoryUsage(int(size))
		common.UseMemory(inter, usage)
	}

	deltas := slabStorage.DeltasWithoutTempAddresses()
	common.UseMemory(inter, common.NewAtreeEncodedSlabMemoryUsage(deltas))

	// TODO: report encoding metric for all encoded slabs
	if deterministic {
		return slabStorage.FastCommit(runtime.NumCPU())
	} else {
		return slabStorage.NondeterministicFastCommit(runtime.NumCPU())
	}
}

func (s *Storage) ScheduleV2Migration(address common.Address) {
	s.scheduledV2Migrations = append(s.scheduledV2Migrations, address)
}

func (s *Storage) ScheduleV2MigrationForModifiedAccounts() {
	for address, isV1 := range s.cachedV1Accounts { //nolint:maprange
		if isV1 && s.PersistentSlabStorage.HasUnsavedChanges(atree.Address(address)) {
		        s.ScheduleV2Migration(address)
		}
		}

		s.ScheduleV2Migration(address)
	}
}

func (s *Storage) migrateV1AccountsToV2(inter *interpreter.Interpreter) error {

	if !s.Config.StorageFormatV2Enabled {
		return errors.NewUnexpectedError("cannot migrate to storage format v2, as it is not enabled")
	}

	if len(s.scheduledV2Migrations) == 0 {
		return nil
	}

	// getDomainStorageMap function returns cached domain storage map if it is available
	// before loading domain storage map from storage.
	// This is necessary to migrate uncommitted (new) but cached domain storage map.
	getDomainStorageMap := func(
		ledger atree.Ledger,
		storage atree.SlabStorage,
		address common.Address,
		domain common.StorageDomain,
	) (*interpreter.DomainStorageMap, error) {
		domainStorageKey := interpreter.NewStorageDomainKey(s.memoryGauge, address, domain)

		// Get cached domain storage map if available.
		domainStorageMap := s.cachedDomainStorageMaps[domainStorageKey]

		if domainStorageMap != nil {
			return domainStorageMap, nil
		}

		return getDomainStorageMapFromV1DomainRegister(ledger, storage, address, domain)
	}

	migrator := NewDomainRegisterMigration(
		s.Ledger,
		s.PersistentSlabStorage,
		inter,
		s.memoryGauge,
		getDomainStorageMap,
	)

	// Ensure the scheduled accounts are migrated in a deterministic order

	sort.Slice(
		s.scheduledV2Migrations,
		func(i, j int) bool {
			address1 := s.scheduledV2Migrations[i]
			address2 := s.scheduledV2Migrations[j]
			return address1.Compare(address2) < 0
		},
	)

	for _, address := range s.scheduledV2Migrations {

		accountStorageMap, err := migrator.MigrateAccount(address)
		if err != nil {
			return err
		}

		// TODO: is this all that is needed?

		s.AccountStorageV2.cacheAccountStorageMap(
			address,
			accountStorageMap,
		)

		s.cacheIsV1Account(address, false)
	}

	s.scheduledV2Migrations = nil

	return nil
}

func (s *Storage) CheckHealth() error {

	// Check slab storage health
	rootSlabIDs, err := atree.CheckStorageHealth(s, -1)
	if err != nil {
		return err
	}

	// Find account / non-temporary root slab IDs

	accountRootSlabIDs := make(map[atree.SlabID]struct{}, len(rootSlabIDs))

	// NOTE: map range is safe, as it creates a subset
	for rootSlabID := range rootSlabIDs { //nolint:maprange
		if rootSlabID.HasTempAddress() {
			continue
		}

		accountRootSlabIDs[rootSlabID] = struct{}{}
	}

	// Check that account storage maps and unmigrated domain storage maps
	// match returned root slabs from atree.CheckStorageHealth.

	var storageMapStorageIDs []atree.SlabID

	if s.Config.StorageFormatV2Enabled {
		// Get cached account storage map slab IDs.
		storageMapStorageIDs = append(
			storageMapStorageIDs,
			s.AccountStorageV2.cachedRootSlabIDs()...,
		)
	}

	// Get slab IDs of cached domain storage maps that are in account storage format v1.
	for storageKey, storageMap := range s.cachedDomainStorageMaps { //nolint:maprange
		address := storageKey.Address

		// Only accounts in storage format v1 store domain storage maps
		// directly at the root of the account
		if !s.IsV1Account(address) {
			continue
		}

		storageMapStorageIDs = append(
			storageMapStorageIDs,
			storageMap.SlabID(),
		)
	}

	sort.Slice(
		storageMapStorageIDs,
		func(i, j int) bool {
			a := storageMapStorageIDs[i]
			b := storageMapStorageIDs[j]
			return a.Compare(b) < 0
		},
	)

	found := map[atree.SlabID]struct{}{}

	for _, storageMapStorageID := range storageMapStorageIDs {
		if _, ok := accountRootSlabIDs[storageMapStorageID]; !ok {
			return errors.NewUnexpectedError(
				"account storage map (and unmigrated domain storage map) points to non-root slab %s",
				storageMapStorageID,
			)
		}

		found[storageMapStorageID] = struct{}{}
	}

	// Check that all slabs in slab storage
	// are referenced by storables in account storage.
	// If a slab is not referenced, it is garbage.

	if len(accountRootSlabIDs) > len(found) {
		var unreferencedRootSlabIDs []atree.SlabID

		for accountRootSlabID := range accountRootSlabIDs { //nolint:maprange
			if _, ok := found[accountRootSlabID]; ok {
				continue
			}

			unreferencedRootSlabIDs = append(
				unreferencedRootSlabIDs,
				accountRootSlabID,
			)
		}

		sort.Slice(unreferencedRootSlabIDs, func(i, j int) bool {
			a := unreferencedRootSlabIDs[i]
			b := unreferencedRootSlabIDs[j]
			return a.Compare(b) < 0
		})

		return UnreferencedRootSlabsError{
			UnreferencedRootSlabIDs: unreferencedRootSlabIDs,
		}
	}

	return nil
}

type UnreferencedRootSlabsError struct {
	UnreferencedRootSlabIDs []atree.SlabID
}

var _ errors.InternalError = UnreferencedRootSlabsError{}

func (UnreferencedRootSlabsError) IsInternalError() {}

func (e UnreferencedRootSlabsError) Error() string {
	return fmt.Sprintf("slabs not referenced: %s", e.UnreferencedRootSlabIDs)
}
