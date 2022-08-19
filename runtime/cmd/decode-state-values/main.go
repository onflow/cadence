/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

// A utility program that parses a state dump in JSON Lines format and decodes all values

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"
	"github.com/schollz/progressbar/v3"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

type stringSlice []string

func (s stringSlice) String() string {
	return strings.Join(s, ", ")
}

func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

var addressesFlag stringSlice

func init() {
	flag.Var(&addressesFlag, "addresses", "only keep ledger keys for given addresses")
}

var gzipFlag = flag.Bool("gzip", false, "set true if input file is gzipped")
var printFlag = flag.Bool("print", false, "print parsed data (filtered, if addresses are given)")
var loadFlag = flag.Bool("load", false, "load the parsed data")
var checkSlabsFlag = flag.Bool("check-slabs", false, "check slabs")
var checkValuesFlag = flag.Bool("check-values", false, "check values")

const keyPartCount = 3

type storageKey [keyPartCount]string

var storage = map[storageKey][]byte{}

var storagePathSeparator = "\x1f"

// '$' + 8 byte index
const slabKeyLength = 9

func isSlabStorageKey(key string) bool {
	return len(key) == slabKeyLength && key[0] == '$'
}

func storageKeySlabStorageID(address atree.Address, key string) atree.StorageID {
	if !isSlabStorageKey(key) {
		return atree.StorageIDUndefined
	}
	var result atree.StorageID
	result.Address = address
	copy(result.Index[:], key[1:])
	return result
}

func decodeStorable(decoder *cbor.StreamDecoder, storableSlabStorageID atree.StorageID) (atree.Storable, error) {
	return interpreter.DecodeStorable(decoder, storableSlabStorageID, nil)
}

func decodeTypeInfo(decoder *cbor.StreamDecoder) (atree.TypeInfo, error) {
	return interpreter.DecodeTypeInfo(decoder, nil)
}

func decodeSlab(id atree.StorageID, data []byte) (atree.Slab, error) {
	return atree.DecodeSlab(
		id,
		data,
		interpreter.CBORDecMode,
		decodeStorable,
		decodeTypeInfo,
	)
}

func storageIDStorageKey(id atree.StorageID) storageKey {
	return storageKey{
		string(id.Address[:]),
		"",
		"$" + string(id.Index[:]),
	}
}

// slabStorage

type slabStorage struct{}

var _ atree.SlabStorage = &slabStorage{}

func (s *slabStorage) Retrieve(id atree.StorageID) (atree.Slab, bool, error) {
	data, ok := storage[storageIDStorageKey(id)]
	if !ok {
		return nil, false, nil
	}

	slab, err := decodeSlab(id, data)
	if err != nil {
		return nil, true, err
	}

	return slab, true, nil
}

func (s *slabStorage) Store(_ atree.StorageID, _ atree.Slab) error {
	panic("unexpected Store call")
}

func (s *slabStorage) Remove(_ atree.StorageID) error {
	panic("unexpected Remove call")
}

func (s *slabStorage) GenerateStorageID(_ atree.Address) (atree.StorageID, error) {
	panic("unexpected GenerateStorageID call")
}

func (s *slabStorage) SlabIterator() (atree.SlabIterator, error) {
	var slabs []struct {
		atree.StorageID
		storageKey
	}

	// NOTE: iteration over map is safe,
	// as result is sorted below

	for key := range storage { //nolint:maprangecheck

		var address atree.Address
		copy(address[:], key[0])
		storageID := storageKeySlabStorageID(address, key[2])
		if storageID == atree.StorageIDUndefined {
			continue
		}

		slabs = append(slabs, struct {
			atree.StorageID
			storageKey
		}{
			StorageID:  storageID,
			storageKey: key,
		})
	}

	sort.Slice(slabs, func(i, j int) bool {
		a := slabs[i]
		b := slabs[j]
		return a.StorageID.Compare(b.StorageID) < 0
	})

	var i int

	bar := progressbar.Default(int64(len(slabs)))

	return func() (atree.StorageID, atree.Slab) {
		if i >= len(slabs) {
			_ = bar.Close()
			return atree.StorageIDUndefined, nil
		}

		slabEntry := slabs[i]
		i++

		_ = bar.Add(1)

		storageID := slabEntry.StorageID
		data := storage[slabEntry.storageKey]

		slab, err := decodeSlab(storageID, data)
		if err != nil {
			log.Fatalf("failed to decode slab @ %s", storageID)
		}

		return storageID, slab
	}, nil
}

func (s *slabStorage) Count() int {
	return len(storage)
}

// interpreterStorage

type interpreterStorage struct {
	*slabStorage
}

var _ interpreter.Storage = &interpreterStorage{}

func (i interpreterStorage) GetStorageMap(_ common.Address, _ string, _ bool) *interpreter.StorageMap {
	panic("unexpected GetStorageMap call")
}

func (i interpreterStorage) CheckHealth() error {
	panic("unexpected CheckHealth call")
}

// load

func load() {

	log.Println("Validating slabs ...")

	slabStorage := &slabStorage{}

	if *checkSlabsFlag {
		_, err := atree.CheckStorageHealth(slabStorage, -1)
		if err != nil {
			log.Fatalf("Slab storage problem: %s", err)
		}
	}

	log.Println("Loading decoded values ...")

	interpreterStorage := &interpreterStorage{
		slabStorage: slabStorage,
	}

	inter, err := interpreter.NewInterpreter(
		nil,
		nil,
		&interpreter.Config{
			Storage: interpreterStorage,
		},
	)
	if err != nil {
		log.Fatalf("Failed to create interpreter: %s", err)
	}

	bar := progressbar.Default(int64(len(storage)))

	var slabNotFoundErrCount int

	for storageKey, data := range storage { //nolint:maprangecheck
		_ = bar.Add(1)

		// Check the key is a non-root slab or a storage path
		key := storageKey[2]

		var address atree.Address
		copy(address[:], storageKey[0])

		err := loadStorageKey(key, address, data, inter, slabStorage)
		var slabNotFoundErr *atree.SlabNotFoundError
		if errors.As(err, &slabNotFoundErr) {
			slabNotFoundErrCount++
		}
	}

	log.Printf("Loaded all values. %d failed due to missing slabs", slabNotFoundErrCount)
}

func loadStorageKey(
	key string,
	address atree.Address,
	data []byte,
	inter *interpreter.Interpreter,
	slabStorage *slabStorage,
) (err error) {

	defer func() {
		if r := recover(); r != nil {
			log.Printf("failed to load storage key @ 0x%x %s: %v", address, key, r)
			err, _ = r.(error)
		}
	}()

	// If the key is for a slab (format '$' + storage index),
	// then attempt to decode the slab

	if isSlabStorageKey(key) {

		// Only decode each slab if it was not already decoded
		// for the slab health check

		if !*checkSlabsFlag {

			var storageIndex atree.StorageIndex
			// Skip '$' prefix
			copy(storageIndex[:], key[1:])

			storageID := atree.StorageID{
				Address: address,
				Index:   storageIndex,
			}

			_, err := decodeSlab(storageID, data)
			if err != nil {
				log.Printf(
					"Failed to decode slab @ %s: %s (size: %d)",
					storageID, err, len(data),
				)
				return err
			}
		}
	} else {
		// If the key is an account path,
		// decode the storable, and load the value

		keyParts := strings.SplitN(key, storagePathSeparator, 2)

		isStoragePath := len(keyParts) == 2 &&
			common.PathDomainFromIdentifier(keyParts[0]) != common.PathDomainUnknown

		if isStoragePath {

			reader := bytes.NewReader(data)
			decoder := interpreter.CBORDecMode.NewStreamDecoder(reader)
			storable, err := interpreter.DecodeStorable(decoder, atree.StorageIDUndefined, nil)
			if err != nil {
				log.Printf(
					"Failed to decode storable @ 0x%x %s: %s (data: %x)\n",
					address, key, err, data,
				)
				return err
			}

			atreeValue, err := storable.StoredValue(slabStorage)
			if err != nil {
				log.Printf(
					"Failed to load stored value @ 0x%x %s: %s",
					address, key, err,
				)
				return err
			}

			value, err := interpreter.ConvertStoredValue(inter, atreeValue)
			if err != nil {
				log.Printf(
					"Failed to convert stored value @ 0x%x %s: %s",
					address, key, err,
				)
				return err
			}

			interpreter.InspectValue(
				inter,
				value,
				func(v interpreter.Value) bool {

					if composite, ok := v.(*interpreter.CompositeValue); ok &&
						composite.Kind == common.CompositeKindResource &&
						composite.ResourceUUID(inter, interpreter.ReturnEmptyLocationRange) == nil {

						log.Printf(
							"Failed to get UUID for resource @ 0x%x %s",
							address, key,
						)
					}

					return true
				},
			)

			if *checkValuesFlag {
				inter.ValidateAtreeValue(value)
			}
		}
	}

	return nil
}

type encodedKeyPart struct {
	Value string
}

type encodedKey struct {
	KeyParts []encodedKeyPart
}

type encodedEntry struct {
	Value string
	Key   encodedKey
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		panic("missing path argument")
	}

	var addresses []common.Address

	for _, hexAddress := range addressesFlag {
		address, err := common.HexToAddress(hexAddress)
		if err != nil {
			log.Fatalf("Invalid address: %s", hexAddress)
		}
		addresses = append(addresses, address)
	}

	file, err := os.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	read(file, addresses)

	if *loadFlag {
		load()
	}

	if *printFlag {
		for key, value := range storage { //nolint:maprangecheck
			var keyParts []encodedKeyPart

			for _, keyPart := range key {
				keyParts = append(keyParts, encodedKeyPart{
					Value: hex.EncodeToString([]byte(keyPart)),
				})
			}

			entry := encodedEntry{
				Value: hex.EncodeToString(value),
				Key: encodedKey{
					KeyParts: keyParts,
				},
			}

			encoded, err := json.Marshal(entry)
			if err != nil {
				log.Fatal(err)
			}
			log.Println(encoded)
		}
	}
}

func read(file *os.File, addresses []common.Address) {

	log.Println("Reading file ...")

	filter := len(addresses) > 0

	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	fileSize := stat.Size()

	bar := progressbar.DefaultBytes(fileSize, "(processed JSON bytes)")

	progressReader := progressbar.NewReader(file, bar)
	defer progressReader.Close()

	var inputReader io.Reader = &progressReader
	if *gzipFlag {
		gzipReader, err := gzip.NewReader(inputReader)
		if err != nil {
			log.Fatal(err)
		}
		defer gzipReader.Close()
		inputReader = gzipReader
	}

	reader := bufio.NewReader(inputReader)

	decoder := json.NewDecoder(reader)

	var emptyLines int
	var line int

payloadLoop:
	for ; true; line++ {
		var e encodedEntry

		err = decoder.Decode(&e)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		currentKeyPartCount := len(e.Key.KeyParts)
		if currentKeyPartCount < keyPartCount {
			if currentKeyPartCount > 0 {
				log.Fatalf("Invalid storage key parts on line %d: %#+v", line, e.Key)
			}
			emptyLines++
			continue
		}

		var storageKey [keyPartCount]string
		for i := 0; i < keyPartCount; i++ {
			keyPart := e.Key.KeyParts[i].Value
			k, err := hex.DecodeString(keyPart)
			if err != nil {
				log.Fatalf(
					"Failed to hex-decode key part %d of %s (%s): %s",
					i, e.Key, keyPart, err,
				)
			}
			// Treat bytes as string,
			// so resulting array of strings can be used as a map key
			storageKey[i] = string(k)
		}

		if filter {
			owner := common.MustBytesToAddress([]byte(storageKey[0]))
			var found bool
			for _, address := range addresses {
				if owner == address {
					found = true
					break
				}
			}
			if !found {
				continue payloadLoop
			}
		}

		data, err := hex.DecodeString(e.Value)
		if err != nil {
			log.Fatalf("Invalid value: %s", err)
		}

		// Ignore empty slabs
		if len(data) > 0 {
			storage[storageKey] = data
		}
	}

	log.Printf(
		"read %d lines (%d empty, %f%%)",
		line, emptyLines, float32(emptyLines*100)/float32(line),
	)
}
