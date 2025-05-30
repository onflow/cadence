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
	goRuntime "runtime"
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
}

type StorageFormat uint8

const (
	StorageFormatUnknown StorageFormat = iota
	StorageFormatV1
	StorageFormatV2
)

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

	AccountStorageV1 *AccountStorageV1
	AccountStorageV2 *AccountStorageV2
}

var _ atree.SlabStorage = &Storage{}
var _ interpreter.Storage = &Storage{}

func NewPersistentSlabStorage(
	ledger atree.Ledger,
	memoryGauge common.MemoryGauge,
) *atree.PersistentSlabStorage {
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

	return atree.NewPersistentSlabStorage(
		ledgerStorage,
		interpreter.CBOREncMode,
		interpreter.CBORDecMode,
		decodeStorable,
		decodeTypeInfo,
	)
}

func NewStorage(
	ledger atree.Ledger,
	memoryGauge common.MemoryGauge,
	config StorageConfig,
) *Storage {
	persistentSlabStorage := NewPersistentSlabStorage(ledger, memoryGauge)

	accountStorageV1 := NewAccountStorageV1(
		ledger,
		persistentSlabStorage,
		memoryGauge,
	)

	accountStorageV2 := NewAccountStorageV2(
		ledger,
		persistentSlabStorage,
		memoryGauge,
	)

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
	storageMutationTracker interpreter.StorageMutationTracker,
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

	// Check if cached account format is available.

	cachedFormat, known := s.getCachedAccountFormat(address)
	if known {
		return s.getDomainStorageMap(
			cachedFormat,
			storageMutationTracker,
			address,
			domain,
			createIfNotExists,
		)
	}

	// Check if account is v2 (by reading "stored" register).

	if s.isV2Account(address) {
		return s.getDomainStorageMapForV2Account(
			storageMutationTracker,
			address,
			domain,
			createIfNotExists,
		)
	}

	// Check if account is v1 (by reading requested domain register).

	if s.hasDomainRegister(address, domain) {
		return s.getDomainStorageMapForV1Account(
			address,
			domain,
			createIfNotExists,
		)
	}

	// Domain register doesn't exist.

	// Return early if !createIfNotExists to avoid more register reading.

	if !createIfNotExists {
		return nil
	}

	// At this point, account is either new account or v1 account without requested domain register.

	// Check if account is v1 (by reading more domain registers)

	if s.isV1Account(address) {
		return s.getDomainStorageMapForV1Account(
			address,
			domain,
			createIfNotExists,
		)
	}

	// New account is treated as v2 account when feature flag is enabled.

	return s.getDomainStorageMapForV2Account(
		storageMutationTracker,
		address,
		domain,
		createIfNotExists,
	)
}

func (s *Storage) getDomainStorageMapForV1Account(
	address common.Address,
	domain common.StorageDomain,
	createIfNotExists bool,
) *interpreter.DomainStorageMap {
	domainStorageMap := s.AccountStorageV1.GetDomainStorageMap(
		address,
		domain,
		createIfNotExists,
	)

	s.cacheIsV1Account(address, true)

	return domainStorageMap
}

func (s *Storage) getDomainStorageMapForV2Account(
	storageMutationTracker interpreter.StorageMutationTracker,
	address common.Address,
	domain common.StorageDomain,
	createIfNotExists bool,
) *interpreter.DomainStorageMap {
	domainStorageMap := s.AccountStorageV2.GetDomainStorageMap(
		storageMutationTracker,
		address,
		domain,
		createIfNotExists,
	)

	s.cacheIsV1Account(address, false)

	return domainStorageMap
}

func (s *Storage) getDomainStorageMap(
	format StorageFormat,
	storageMutationTracker interpreter.StorageMutationTracker,
	address common.Address,
	domain common.StorageDomain,
	createIfNotExists bool,
) *interpreter.DomainStorageMap {
	switch format {

	case StorageFormatV1:
		return s.getDomainStorageMapForV1Account(
			address,
			domain,
			createIfNotExists,
		)

	case StorageFormatV2:
		return s.getDomainStorageMapForV2Account(
			storageMutationTracker,
			address,
			domain,
			createIfNotExists,
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

func (s *Storage) getCachedAccountFormat(address common.Address) (format StorageFormat, known bool) {
	isV1, cached := s.cachedV1Accounts[address]
	if !cached {
		return StorageFormatUnknown, false
	}
	if isV1 {
		return StorageFormatV1, true
	} else {
		return StorageFormatV2, true
	}
}

// isV2Account returns true if given account is in account storage format v2.
func (s *Storage) isV2Account(address common.Address) bool {
	accountStorageMapExists, err := hasAccountStorageMap(s.Ledger, address)
	if err != nil {
		panic(err)
	}

	return accountStorageMapExists
}

// hasDomainRegister returns true if given account has given domain register.
// NOTE: account storage format v1 has domain registers.
func (s *Storage) hasDomainRegister(address common.Address, domain common.StorageDomain) (domainExists bool) {
	_, domainExists, err := readSlabIndexFromRegister(
		s.Ledger,
		address,
		[]byte(domain.Identifier()),
	)
	if err != nil {
		panic(err)
	}

	return domainExists
}

// isV1Account returns true if given account is in account storage format v1
// by checking if any of the domain registers exist.
func (s *Storage) isV1Account(address common.Address) (isV1 bool) {

	// Check if a storage map register exists for any of the domains.
	// Check the most frequently used domains first, such as storage, public, private.
	for _, domain := range common.AllStorageDomains {
		domainExists := s.hasDomainRegister(address, domain)
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
func (s *Storage) commitContractUpdates(context interpreter.ValueTransferContext) {
	if s.contractUpdates == nil {
		return
	}

	for pair := s.contractUpdates.Oldest(); pair != nil; pair = pair.Next() {
		s.writeContractUpdate(context, pair.Key, pair.Value)
	}
}

func (s *Storage) writeContractUpdate(
	context interpreter.ValueTransferContext,
	key interpreter.StorageKey,
	contractValue *interpreter.CompositeValue,
) {
	storageMap := s.GetDomainStorageMap(context, key.Address, common.StorageDomainContract, true)
	// NOTE: pass nil instead of allocating a Value-typed  interface that points to nil
	storageMapKey := interpreter.StringStorageMapKey(key.Key)
	if contractValue == nil {
		storageMap.WriteValue(context, storageMapKey, nil)
	} else {
		storageMap.WriteValue(context, storageMapKey, contractValue)
	}
}

// Commit serializes/saves all values in the readCache in storage (through the runtime interface).
func (s *Storage) Commit(context interpreter.ValueTransferContext, commitContractUpdates bool) error {
	return s.commit(context, commitContractUpdates, true)
}

// Deprecated: NondeterministicCommit serializes and commits all values in the deltas storage
// in nondeterministic order.  This function is used when commit ordering isn't
// required (e.g. migration programs).
func (s *Storage) NondeterministicCommit(inter *interpreter.Interpreter, commitContractUpdates bool) error {
	return s.commit(inter, commitContractUpdates, false)
}

func (s *Storage) commit(context interpreter.ValueTransferContext, commitContractUpdates bool, deterministic bool) error {

	if commitContractUpdates {
		s.commitContractUpdates(context)
	}

	err := s.AccountStorageV1.commit()
	if err != nil {
		return err
	}

	err = s.AccountStorageV2.commit()
	if err != nil {
		return err
	}

	// Commit the underlying slab storage's writes

	slabStorage := s.PersistentSlabStorage

	size := slabStorage.DeltasSizeWithoutTempAddresses()
	if size > 0 {
		common.UseComputation(
			context,
			common.ComputationUsage{
				Kind:      common.ComputationKindEncodeValue,
				Intensity: size,
			},
		)

		common.UseMemory(
			context,
			common.NewBytesMemoryUsage(int(size)),
		)
	}

	deltas := slabStorage.DeltasWithoutTempAddresses()
	common.UseMemory(context, common.NewAtreeEncodedSlabMemoryUsage(deltas))

	// TODO: report encoding metric for all encoded slabs
	workerCount := goRuntime.NumCPU()
	if deterministic {
		return slabStorage.FastCommit(workerCount)
	} else {
		return slabStorage.NondeterministicFastCommit(workerCount)
	}
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
	storageMapStorageIDs = append(
		storageMapStorageIDs,
		s.AccountStorageV2.cachedRootSlabIDs()...,
	)

	// Get slab IDs of cached domain storage maps that are in account storage format v1.
	for storageKey, storageMap := range s.cachedDomainStorageMaps { //nolint:maprange
		address := storageKey.Address

		// Only accounts in storage format v1 store domain storage maps
		// directly at the root of the account
		if !s.isV1Account(address) {
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

// AccountStorageFormat returns either StorageFormatV1 or StorageFormatV2 for existing accounts,
// and StorageFormatUnknown for non-existing accounts.
func (s *Storage) AccountStorageFormat(address common.Address) (format StorageFormat) {
	cachedFormat, known := s.getCachedAccountFormat(address)
	if known {
		return cachedFormat
	}

	defer func() {
		// Cache account fomat
		switch format {
		case StorageFormatV1:
			s.cacheIsV1Account(address, true)
		case StorageFormatV2:
			s.cacheIsV1Account(address, false)
		}
	}()

	if s.isV2Account(address) {
		return StorageFormatV2
	}

	if s.isV1Account(address) {
		return StorageFormatV1
	}

	return StorageFormatUnknown
}

type UnreferencedRootSlabsError struct {
	UnreferencedRootSlabIDs []atree.SlabID
}

var _ errors.InternalError = UnreferencedRootSlabsError{}

func (UnreferencedRootSlabsError) IsInternalError() {}

func (e UnreferencedRootSlabsError) Error() string {
	return fmt.Sprintf(
		"%s slabs not referenced: %s",
		errors.InternalErrorMessagePrefix,
		e.UnreferencedRootSlabIDs,
	)
}

func CommitStorage(context interpreter.ValueTransferContext, storage *Storage, checkStorageHealth bool) error {
	const commitContractUpdates = true
	err := storage.Commit(context, commitContractUpdates)
	if err != nil {
		return err
	}

	if checkStorageHealth {
		err = storage.CheckHealth()
		if err != nil {
			return err
		}
	}

	return nil
}
