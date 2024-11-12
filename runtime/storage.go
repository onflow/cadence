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

type Storage struct {
	*atree.PersistentSlabStorage

	// NewAccountStorageMapSlabIndices contains root slab index of new accounts' storage map.
	// The indices are saved using Ledger.SetValue() during Commit().
	// Key is StorageKey{address, accountStorageKey} and value is 8-byte slab index.
	NewAccountStorageMapSlabIndices *orderedmap.OrderedMap[interpreter.StorageKey, atree.SlabIndex]

	// unmigratedAccounts are accounts that were accessed but not migrated.
	unmigratedAccounts *orderedmap.OrderedMap[common.Address, struct{}]

	// cachedAccountStorageMaps is a cache of account storage maps.
	// Key is StorageKey{address, accountStorageKey} and value is account storage map.
	cachedAccountStorageMaps map[interpreter.StorageKey]*interpreter.AccountStorageMap

	// cachedDomainStorageMaps is a cache of domain storage maps.
	// Key is StorageKey{address, domain} and value is domain storage map.
	cachedDomainStorageMaps map[interpreter.StorageDomainKey]*interpreter.DomainStorageMap

	// contractUpdates is a cache of contract updates.
	// Key is StorageKey{contract_address, contract_name} and value is contract composite value.
	contractUpdates *orderedmap.OrderedMap[interpreter.StorageKey, *interpreter.CompositeValue]

	Ledger atree.Ledger

	memoryGauge common.MemoryGauge
}

var _ atree.SlabStorage = &Storage{}
var _ interpreter.Storage = &Storage{}

func NewStorage(ledger atree.Ledger, memoryGauge common.MemoryGauge) *Storage {
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
	return &Storage{
		Ledger:                   ledger,
		PersistentSlabStorage:    persistentSlabStorage,
		cachedAccountStorageMaps: map[interpreter.StorageKey]*interpreter.AccountStorageMap{},
		cachedDomainStorageMaps:  map[interpreter.StorageDomainKey]*interpreter.DomainStorageMap{},
		memoryGauge:              memoryGauge,
	}
}

const storageIndexLength = 8

// GetStorageMap returns existing or new domain storage map for the given account and domain.
func (s *Storage) GetStorageMap(
	inter *interpreter.Interpreter,
	address common.Address,
	domain common.StorageDomain,
	createIfNotExists bool,
) (
	storageMap *interpreter.DomainStorageMap,
) {

	// Account can be migrated account, new account, or unmigrated account.
	//
	//
	// ### Migrated Account
	//
	// Migrated account is account with AccountStorageKey register.
	// Migrated account has account storage map, which contains domain storage maps
	// with domain as key.
	//
	// If domain exists in the account storage map, domain storage map is returned.
	//
	// If domain doesn't exist and createIfNotExists is true,
	// new domain storage map is created, inserted into account storage map, and returned.
	//
	// If domain doesn't exist and createIfNotExists is false, nil is returned.
	//
	//
	// ### New Account
	//
	// New account is account without AccountStorageKey register and without any domain registers.
	// NOTE: new account's AccountStorageKey register is persisted in Commit().
	//
	// If createIfNotExists is true,
	// - new account storage map is created
	// - new domain storage map is created
	// - domain storage map is inserted into account storage map
	// - domain storage map is returned
	//
	// If createIfNotExists is false, nil is returned.
	//
	//
	// ### Unmigrated Account
	//
	// Unmigrated account is account with at least one domain register.
	// Unmigrated account has domain registers and corresponding domain storage maps.
	//
	// If domain exists (domain register exists), domain storage map is loaded and returned.
	//
	// If domain doesn't exist and createIfNotExists is true,
	// new domain storage map is created and returned.
	// NOTE: given account would be migrated in Commit() since this is write op.
	//
	// If domain doesn't exist and createIfNotExists is false, nil is returned.
	//
	//
	// ### Migration of unmigrated accounts
	//
	// Migration happens in Commit() if unmigrated account has write ops.
	// NOTE: Commit() is not called by this function.
	//
	// Specifically,
	// - unmigrated account is migrated in Commit() if there are write ops.
	//   For example, inserting values in unmigrated account triggers migration in Commit().
	// - unmigrated account is unchanged if there are only read ops.
	//   For example, iterating values in unmigrated account doesn't trigger migration,
	//   and checking if domain exists doesn't trigger migration.

	// Get cached domain storage map if it exists.

	domainStorageKey := interpreter.NewStorageDomainKey(s.memoryGauge, address, domain)

	if domainStorageMap := s.cachedDomainStorageMaps[domainStorageKey]; domainStorageMap != nil {
		return domainStorageMap
	}

	// Get (or create) domain storage map from existing account storage map
	// if account is migrated account.

	accountStorageKey := interpreter.NewStorageKey(s.memoryGauge, address, AccountStorageKey)

	accountStorageMap, err := s.getAccountStorageMap(accountStorageKey)
	if err != nil {
		panic(err)
	}

	if accountStorageMap != nil {
		// This is migrated account.

		// Get (or create) domain storage map from account storage map.
		domainStorageMap := accountStorageMap.GetDomain(s.memoryGauge, inter, domain, createIfNotExists)

		// Cache domain storage map
		if domainStorageMap != nil {
			s.cachedDomainStorageMaps[domainStorageKey] = domainStorageMap
		}

		return domainStorageMap
	}

	// At this point, account is either new or unmigrated account.

	domainStorageMap, err := getDomainStorageMapFromLegacyDomainRegister(s.Ledger, s.PersistentSlabStorage, address, domain)
	if err != nil {
		panic(err)
	}

	if domainStorageMap != nil {
		// This is a unmigrated account with given domain register.

		// Cache domain storage map
		s.cachedDomainStorageMaps[domainStorageKey] = domainStorageMap

		// Add account to unmigrated account list
		s.addUnmigratedAccount(address)

		return domainStorageMap
	}

	// At this point, account is either new account or unmigrated account without given domain.

	// Domain doesn't exist.  Return early if createIfNotExists is false.

	if !createIfNotExists {
		return nil
	}

	// Handle unmigrated account
	unmigrated, err := s.isUnmigratedAccount(address)
	if err != nil {
		panic(err)
	}
	if unmigrated {
		// Add account to unmigrated account list
		s.addUnmigratedAccount(address)

		// Create new domain storage map
		domainStorageMap := interpreter.NewDomainStorageMap(s.memoryGauge, s, atree.Address(address))

		// Cache new domain storage map
		s.cachedDomainStorageMaps[domainStorageKey] = domainStorageMap

		return domainStorageMap
	}

	// Handle new account

	// Create account storage map
	accountStorageMap = interpreter.NewAccountStorageMap(s.memoryGauge, s, atree.Address(address))

	// Cache account storage map
	s.cachedAccountStorageMaps[accountStorageKey] = accountStorageMap

	// Create new domain storage map as an element in account storage map
	domainStorageMap = accountStorageMap.NewDomain(s.memoryGauge, inter, domain)

	// Cache domain storage map
	s.cachedDomainStorageMaps[domainStorageKey] = domainStorageMap

	// Save new account and its account storage map root SlabID to new accout list
	s.addNewAccount(accountStorageKey, accountStorageMap.SlabID().Index())

	return domainStorageMap
}

// getAccountStorageMap returns AccountStorageMap if exists, or nil otherwise.
func (s *Storage) getAccountStorageMap(accountStorageKey interpreter.StorageKey) (*interpreter.AccountStorageMap, error) {

	// Return cached account storage map if available.

	accountStorageMap := s.cachedAccountStorageMaps[accountStorageKey]
	if accountStorageMap != nil {
		return accountStorageMap, nil
	}

	// Load account storage map if account storage register exists.

	accountStorageSlabIndex, accountStorageRegisterExists, err := getSlabIndexFromRegisterValue(
		s.Ledger,
		accountStorageKey.Address,
		[]byte(accountStorageKey.Key),
	)
	if err != nil {
		return nil, err
	}
	if !accountStorageRegisterExists {
		return nil, nil
	}

	slabID := atree.NewSlabID(
		atree.Address(accountStorageKey.Address),
		accountStorageSlabIndex,
	)

	accountStorageMap = interpreter.NewAccountStorageMapWithRootID(s, slabID)

	// Cache account storage map

	s.cachedAccountStorageMaps[accountStorageKey] = accountStorageMap

	return accountStorageMap, nil
}

// getDomainStorageMapFromLegacyDomainRegister returns domain storage map from legacy domain register.
func getDomainStorageMapFromLegacyDomainRegister(
	ledger atree.Ledger,
	storage atree.SlabStorage,
	address common.Address,
	domain common.StorageDomain,
) (*interpreter.DomainStorageMap, error) {
	domainStorageSlabIndex, domainRegisterExists, err := getSlabIndexFromRegisterValue(
		ledger,
		address,
		[]byte(domain.Identifier()))
	if err != nil {
		return nil, err
	}
	if !domainRegisterExists {
		return nil, nil
	}

	slabID := atree.NewSlabID(atree.Address(address), domainStorageSlabIndex)
	return interpreter.NewDomainStorageMapWithRootID(storage, slabID), nil
}

func (s *Storage) addUnmigratedAccount(address common.Address) {
	if s.unmigratedAccounts == nil {
		s.unmigratedAccounts = &orderedmap.OrderedMap[common.Address, struct{}]{}
	}
	if !s.unmigratedAccounts.Contains(address) {
		s.unmigratedAccounts.Set(address, struct{}{})
	}
}

func (s *Storage) addNewAccount(accountStorageKey interpreter.StorageKey, slabIndex atree.SlabIndex) {
	if s.NewAccountStorageMapSlabIndices == nil {
		s.NewAccountStorageMapSlabIndices = &orderedmap.OrderedMap[interpreter.StorageKey, atree.SlabIndex]{}
	}
	s.NewAccountStorageMapSlabIndices.Set(accountStorageKey, slabIndex)
}

// isUnmigratedAccount returns true if given account has any domain registers.
func (s *Storage) isUnmigratedAccount(address common.Address) (bool, error) {
	if s.unmigratedAccounts != nil &&
		s.unmigratedAccounts.Contains(address) {
		return true, nil
	}

	// Check most frequently used domains first, such as storage, public, private.
	for _, domain := range common.AllStorageDomains {
		_, domainExists, err := getSlabIndexFromRegisterValue(
			s.Ledger,
			address,
			[]byte(domain.Identifier()))
		if err != nil {
			return false, err
		}
		if domainExists {
			return true, nil
		}
	}

	return false, nil
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
	storageMap := s.GetStorageMap(inter, key.Address, common.StorageDomainContract, true)
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

// NondeterministicCommit serializes and commits all values in the deltas storage
// in nondeterministic order.  This function is used when commit ordering isn't
// required (e.g. migration programs).
func (s *Storage) NondeterministicCommit(inter *interpreter.Interpreter, commitContractUpdates bool) error {
	return s.commit(inter, commitContractUpdates, false)
}

func (s *Storage) commit(inter *interpreter.Interpreter, commitContractUpdates bool, deterministic bool) error {

	if commitContractUpdates {
		s.commitContractUpdates(inter)
	}

	err := s.commitNewStorageMaps()
	if err != nil {
		return err
	}

	// Migrate accounts that have write ops before calling PersistentSlabStorage.FastCommit().
	err = s.migrateAccountsIfNeeded(inter)
	if err != nil {
		return err
	}

	// Commit the underlying slab storage's writes

	size := s.PersistentSlabStorage.DeltasSizeWithoutTempAddresses()
	if size > 0 {
		inter.ReportComputation(common.ComputationKindEncodeValue, uint(size))
		usage := common.NewBytesMemoryUsage(int(size))
		common.UseMemory(s.memoryGauge, usage)
	}

	deltas := s.PersistentSlabStorage.DeltasWithoutTempAddresses()
	common.UseMemory(s.memoryGauge, common.NewAtreeEncodedSlabMemoryUsage(deltas))

	// TODO: report encoding metric for all encoded slabs
	if deterministic {
		return s.PersistentSlabStorage.FastCommit(runtime.NumCPU())
	} else {
		return s.PersistentSlabStorage.NondeterministicFastCommit(runtime.NumCPU())
	}
}

func (s *Storage) commitNewStorageMaps() error {
	if s.NewAccountStorageMapSlabIndices == nil {
		return nil
	}

	for pair := s.NewAccountStorageMapSlabIndices.Oldest(); pair != nil; pair = pair.Next() {
		var err error
		errors.WrapPanic(func() {
			err = s.Ledger.SetValue(
				pair.Key.Address[:],
				[]byte(pair.Key.Key),
				pair.Value[:],
			)
		})
		if err != nil {
			return interpreter.WrappedExternalError(err)
		}
	}

	return nil
}

func (s *Storage) migrateAccountsIfNeeded(inter *interpreter.Interpreter) error {
	if s.unmigratedAccounts == nil || s.unmigratedAccounts.Len() == 0 {
		return nil
	}
	return s.migrateAccounts(inter)
}

func (s *Storage) migrateAccounts(inter *interpreter.Interpreter) error {
	// Predicate function allows migration for accounts with write ops.
	migrateAccountPred := func(address common.Address) bool {
		return s.PersistentSlabStorage.HasUnsavedChanges(atree.Address(address))
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

		return getDomainStorageMapFromLegacyDomainRegister(ledger, storage, address, domain)
	}

	migrator := NewDomainRegisterMigration(s.Ledger, s.PersistentSlabStorage, inter, s.memoryGauge)
	migrator.SetGetDomainStorageMapFunc(getDomainStorageMap)

	migratedAccounts, err := migrator.MigrateAccounts(s.unmigratedAccounts, migrateAccountPred)
	if err != nil {
		return err
	}

	if migratedAccounts == nil {
		return nil
	}

	// Update internal state with migrated accounts
	for pair := migratedAccounts.Oldest(); pair != nil; pair = pair.Next() {
		address := pair.Key
		accountStorageMap := pair.Value

		// Cache migrated account storage map
		accountStorageKey := interpreter.NewStorageKey(s.memoryGauge, address, AccountStorageKey)
		s.cachedAccountStorageMaps[accountStorageKey] = accountStorageMap

		// Remove migrated accounts from unmigratedAccounts
		s.unmigratedAccounts.Delete(address)
	}

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

	// Get cached account storage map slab IDs.
	for _, storageMap := range s.cachedAccountStorageMaps { //nolint:maprange
		storageMapStorageIDs = append(
			storageMapStorageIDs,
			storageMap.SlabID(),
		)
	}

	// Get cached unmigrated domain storage map slab IDs
	for storageKey, storageMap := range s.cachedDomainStorageMaps { //nolint:maprange
		address := storageKey.Address

		if s.unmigratedAccounts != nil &&
			s.unmigratedAccounts.Contains(address) {

			domainValueID := storageMap.ValueID()

			slabID := atree.NewSlabID(
				atree.Address(address),
				atree.SlabIndex(domainValueID[8:]),
			)

			storageMapStorageIDs = append(
				storageMapStorageIDs,
				slabID,
			)
		}
	}

	sort.Slice(storageMapStorageIDs, func(i, j int) bool {
		a := storageMapStorageIDs[i]
		b := storageMapStorageIDs[j]
		return a.Compare(b) < 0
	})

	found := map[atree.SlabID]struct{}{}

	for _, storageMapStorageID := range storageMapStorageIDs {
		if _, ok := accountRootSlabIDs[storageMapStorageID]; !ok {
			return errors.NewUnexpectedError("account storage map (and unmigrated domain storage map) points to non-root slab %s", storageMapStorageID)
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
