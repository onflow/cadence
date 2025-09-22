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

// A utility program that decodes a slab from its hex-encoded representation.

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/interpreter"
)

func decodeStorable(
	decoder *cbor.StreamDecoder,
	storableSlabStorageID atree.SlabID,
	inlinedExtraData []atree.ExtraData,
) (atree.Storable, error) {
	return interpreter.DecodeStorable(
		decoder,
		storableSlabStorageID,
		inlinedExtraData,
		nil,
	)
}

func decodeTypeInfo(decoder *cbor.StreamDecoder) (atree.TypeInfo, error) {
	return interpreter.DecodeTypeInfo(decoder, nil)
}

func decodeSlab(id atree.SlabID, data []byte) (atree.Slab, error) {
	return atree.DecodeSlab(
		id,
		data,
		interpreter.CBORDecMode,
		decodeStorable,
		decodeTypeInfo,
	)
}

func main() {
	if len(os.Args) < 3 {
		panic("Usage: decode-slab <address-hex> <index> <data-hex> [<data-hex>]")
	}

	address, err := hex.DecodeString(os.Args[1])
	if err != nil {
		panic(fmt.Errorf("failed to parse address: %w", err))
	}

	index, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic(fmt.Errorf("failed to parse index: %w", err))
	}

	data, err := hex.DecodeString(os.Args[3])
	if err != nil {
		panic(fmt.Errorf("failed to parse data of slab: %w", err))
	}

	var slabIndex atree.SlabIndex
	binary.BigEndian.PutUint64(slabIndex[:], uint64(index))

	slabID := atree.NewSlabID(
		atree.Address(address),
		slabIndex,
	)

	slab1, err := decodeSlab(slabID, data)
	if err != nil {
		panic(fmt.Errorf("failed to decode slab: %w", err))
	}

	fmt.Println(slab1)
	fmt.Println()

	if len(os.Args) > 4 {

		data2, err := hex.DecodeString(os.Args[4])
		if err != nil {
			panic(fmt.Errorf("failed to parse data of slab 2: %w", err))
		}

		slab2, err := decodeSlab(slabID, data2)
		if err != nil {
			panic(fmt.Errorf("failed to decode slab 2: %w", err))
		}

		fmt.Println(slab2)
		fmt.Println()

		if mapDataSlab1, ok := slab1.(*atree.MapDataSlab); ok {
			if mapDataSlab2, ok := slab2.(*atree.MapDataSlab); ok {
				compareMapDataSlabs(mapDataSlab1, mapDataSlab2)
				return
			}
		}

		if !bytes.Equal(data, data2) {
			fmt.Printf("Slabs are different!")
			os.Exit(1)
		}
	}
}

type mapEntry struct {
	key   atree.MapKey
	value atree.MapValue
}

func compareMapDataSlabs(slab1 *atree.MapDataSlab, slab2 *atree.MapDataSlab) {

	if slab1.Count() != slab2.Count() {
		fmt.Printf("Different count: %d vs %d\n", slab1.Count(), slab2.Count())
	}

	newIndexer := func(entries map[string]mapEntry) func(key atree.MapKey, value atree.MapValue) error {
		return func(key atree.MapKey, value atree.MapValue) error {
			entries[string(encodeStorable(key))] = mapEntry{
				key:   key,
				value: value,
			}
			return nil
		}
	}

	entries1 := make(map[string]mapEntry)
	err := slab1.Iterate(nil, newIndexer(entries1))
	if err != nil {
		panic(fmt.Errorf("Error iterating slab 1: %w\n", err))
	}

	entries2 := make(map[string]mapEntry)
	err = slab2.Iterate(nil, newIndexer(entries2))
	if err != nil {
		panic(fmt.Errorf("Error iterating slab 2: %w\n", err))
	}

	for encodedKey, entry1 := range entries1 {
		entry2, ok := entries2[encodedKey]
		if !ok {
			fmt.Printf("Key %q missing in slab 2\n", entry1.key)
			continue
		}

		value1 := entry1.value
		value2 := entry2.value

		if mapDataSlabValue1, ok := value1.(*atree.MapDataSlab); ok {
			if mapDataSlabValue2, ok := value2.(*atree.MapDataSlab); ok {
				compareMapDataSlabs(mapDataSlabValue1, mapDataSlabValue2)
				continue
			}
		}

		if !bytes.Equal(encodeStorable(value1), encodeStorable(value2)) {
			fmt.Printf(
				"Different value for key %q: %q vs %q\n",
				entry1.key,
				value1,
				value2,
			)
		}
	}

	for encodedKey, entry2 := range entries2 {
		_, ok := entries1[encodedKey]
		if !ok {
			fmt.Printf("Key %q missing in slab 1\n", entry2.key)
		}
	}
}

func encodeStorable(storable atree.Storable) []byte {
	var buf bytes.Buffer
	encoder := atree.NewEncoder(&buf, interpreter.CBOREncMode)
	err := storable.Encode(encoder)
	if err != nil {
		panic(fmt.Errorf("failed to encode storable: %w", err))
	}

	err = encoder.CBOR.Flush()
	if err != nil {
		panic(fmt.Errorf("failed to flush encoder: %w", err))
	}

	return buf.Bytes()
}
