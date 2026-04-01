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
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

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
	if len(os.Args) >= 2 && os.Args[1] == "diff" {
		var reader io.Reader
		if len(os.Args) >= 3 {
			f, err := os.Open(os.Args[2])
			if err != nil {
				panic(fmt.Errorf("failed to open file: %w", err))
			}
			defer f.Close()
			reader = f
		} else {
			reader = os.Stdin
		}
		processDiff(reader)
		return
	}

	if len(os.Args) < 3 {
		panic("Usage: decode-slab <address-hex> <index> <data-hex> [<data-hex>]\n       decode-slab diff [file]")
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

		diffs := compareSlabs(slabID, slab1, slab2, data, data2)
		if diffs == 0 {
			fmt.Println("No differences found")
		} else {
			fmt.Printf("%d difference(s) found\n", diffs)
			os.Exit(1)
		}
	}
}

func compareSlabs(slabID atree.SlabID, slab1, slab2 atree.Slab, data1, data2 []byte) int {
	rootPath := fmt.Sprintf("slab(%s)", slabID)

	if mapDataSlab1, ok := slab1.(*atree.MapDataSlab); ok {
		if mapDataSlab2, ok := slab2.(*atree.MapDataSlab); ok {
			return compareMapDataSlabs(rootPath, mapDataSlab1, mapDataSlab2)
		}
	}

	if arrayDataSlab1, ok := slab1.(*atree.ArrayDataSlab); ok {
		if arrayDataSlab2, ok := slab2.(*atree.ArrayDataSlab); ok {
			return compareArrayDataSlabs(rootPath, arrayDataSlab1, arrayDataSlab2)
		}
	}

	if bytes.Equal(data1, data2) {
		return 0
	}

	fmt.Printf("%s: slabs are different\n", rootPath)
	return 1
}

// parseSlabLine parses a line of the form "<address_hex>/$<index> = <data_hex>"
// or "<address_hex>/$<index> =" (empty data).
func parseSlabLine(line string) (slabIDStr string, address []byte, index int, dataHex string, ok bool) {
	idPart, dataHex, found := strings.Cut(line, " =")
	if !found {
		return "", nil, 0, "", false
	}
	dataHex = strings.TrimSpace(dataHex)

	dollarIdx := strings.Index(idPart, "/$")
	if dollarIdx < 0 {
		return "", nil, 0, "", false
	}

	addressHex := idPart[:dollarIdx]
	indexStr := idPart[dollarIdx+2:]

	addr, err := hex.DecodeString(addressHex)
	if err != nil {
		return "", nil, 0, "", false
	}

	idx, err := strconv.Atoi(indexStr)
	if err != nil {
		return "", nil, 0, "", false
	}

	return idPart, addr, idx, dataHex, true
}

func makeSlabID(address []byte, index int) atree.SlabID {
	var slabIndex atree.SlabIndex
	binary.BigEndian.PutUint64(slabIndex[:], uint64(index))
	return atree.NewSlabID(
		atree.Address(address),
		slabIndex,
	)
}

func processDiff(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	type slabChange struct {
		address []byte
		index   int
		oldHex  string
		newHex  string
		hasOld  bool
		hasNew  bool
	}

	var slabIDs []string
	changes := make(map[string]*slabChange)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip diff headers
		if strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "@@") {
			continue
		}

		if len(line) == 0 {
			continue
		}

		prefix := line[0]
		if prefix != ' ' && prefix != '-' && prefix != '+' {
			continue
		}

		// Only process changed lines
		if prefix == ' ' {
			continue
		}

		content := line[1:]

		slabIDStr, address, index, dataHex, ok := parseSlabLine(content)
		if !ok {
			continue
		}

		change, exists := changes[slabIDStr]
		if !exists {
			change = &slabChange{
				address: address,
				index:   index,
			}
			changes[slabIDStr] = change
			slabIDs = append(slabIDs, slabIDStr)
		}

		if prefix == '-' {
			change.oldHex = dataHex
			change.hasOld = true
		} else {
			change.newHex = dataHex
			change.hasNew = true
		}
	}

	if err := scanner.Err(); err != nil {
		panic(fmt.Errorf("failed to read diff: %w", err))
	}

	if len(slabIDs) == 0 {
		fmt.Println("No changed slabs found in diff")
		return
	}

	totalDiffs := 0

	for _, slabIDStr := range slabIDs {
		change := changes[slabIDStr]
		slabID := makeSlabID(change.address, change.index)

		fmt.Printf("=== %s ===\n", slabIDStr)

		switch {
		case change.hasOld && !change.hasNew:
			// Removed slab
			if change.oldHex == "" {
				fmt.Println("Removed (was empty)")
			} else {
				data, err := hex.DecodeString(change.oldHex)
				if err != nil {
					fmt.Printf("Failed to decode old hex: %s\n", err)
				} else {
					slab, err := decodeSlab(slabID, data)
					if err != nil {
						fmt.Printf("Failed to decode old slab: %s\n", err)
					} else {
						fmt.Printf("Removed: %s\n", slab)
					}
				}
			}
			totalDiffs++

		case !change.hasOld && change.hasNew:
			// Added slab
			if change.newHex == "" {
				fmt.Println("Added (empty)")
			} else {
				data, err := hex.DecodeString(change.newHex)
				if err != nil {
					fmt.Printf("Failed to decode new hex: %s\n", err)
				} else {
					slab, err := decodeSlab(slabID, data)
					if err != nil {
						fmt.Printf("Failed to decode new slab: %s\n", err)
					} else {
						fmt.Printf("Added: %s\n", slab)
					}
				}
			}
			totalDiffs++

		case change.hasOld && change.hasNew:
			// Modified slab
			if change.oldHex == change.newHex {
				fmt.Println("No change (identical data)")
				fmt.Println()
				continue
			}

			if change.oldHex == "" {
				data, err := hex.DecodeString(change.newHex)
				if err != nil {
					fmt.Printf("Failed to decode new hex: %s\n", err)
				} else {
					slab, err := decodeSlab(slabID, data)
					if err != nil {
						fmt.Printf("Failed to decode new slab: %s\n", err)
					} else {
						fmt.Printf("Was empty, now: %s\n", slab)
					}
				}
				totalDiffs++
			} else if change.newHex == "" {
				data, err := hex.DecodeString(change.oldHex)
				if err != nil {
					fmt.Printf("Failed to decode old hex: %s\n", err)
				} else {
					slab, err := decodeSlab(slabID, data)
					if err != nil {
						fmt.Printf("Failed to decode old slab: %s\n", err)
					} else {
						fmt.Printf("Was: %s, now empty\n", slab)
					}
				}
				totalDiffs++
			} else {
				oldData, err := hex.DecodeString(change.oldHex)
				if err != nil {
					fmt.Printf("Failed to decode old hex: %s\n", err)
					break
				}

				newData, err := hex.DecodeString(change.newHex)
				if err != nil {
					fmt.Printf("Failed to decode new hex: %s\n", err)
					break
				}

				slab1, err := decodeSlab(slabID, oldData)
				if err != nil {
					fmt.Printf("Failed to decode old slab: %s\n", err)
					break
				}

				slab2, err := decodeSlab(slabID, newData)
				if err != nil {
					fmt.Printf("Failed to decode new slab: %s\n", err)
					break
				}

				fmt.Printf("Old: %s\n", slab1)
				fmt.Printf("New: %s\n", slab2)

				diffs := compareSlabs(slabID, slab1, slab2, oldData, newData)
				if diffs == 0 {
					fmt.Println("No structural differences found")
				} else {
					fmt.Printf("%d difference(s)\n", diffs)
				}
				totalDiffs += diffs
			}
		}

		fmt.Println()
	}

	if totalDiffs > 0 {
		fmt.Printf("Total: %d difference(s) across %d changed slab(s)\n", totalDiffs, len(slabIDs))
		os.Exit(1)
	} else {
		fmt.Println("No differences found")
	}
}

func compareArrayDataSlabs(path string, slab1 *atree.ArrayDataSlab, slab2 *atree.ArrayDataSlab) int {
	diffs := 0

	childStorables1 := slab1.ChildStorables()
	childStorables2 := slab2.ChildStorables()

	if len(childStorables1) != len(childStorables2) {
		fmt.Printf("%s: different count: %d vs %d\n", path, len(childStorables1), len(childStorables2))
		diffs++
	}

	for i, childStorable1 := range childStorables1 {
		childPath := fmt.Sprintf("%s[%d]", path, i)

		if i >= len(childStorables2) {
			fmt.Printf("%s: missing in slab 2\n", childPath)
			diffs++
			continue
		}

		childStorable2 := childStorables2[i]

		diffs += compareChildStorables(
			childPath,
			childStorable1,
			childStorable2,
		)
	}

	return diffs
}

type mapEntry struct {
	key   atree.MapKey
	value atree.MapValue
}

func newMapDataSlabIndexer(entries map[string]mapEntry) func(key atree.MapKey, value atree.MapValue) error {
	return func(key atree.MapKey, value atree.MapValue) error {
		entries[string(encodeStorable(key))] = mapEntry{
			key:   key,
			value: value,
		}
		return nil
	}
}

func compareMapDataSlabs(path string, slab1 *atree.MapDataSlab, slab2 *atree.MapDataSlab) int {
	diffs := 0

	if slab1.Count() != slab2.Count() {
		fmt.Printf("%s: different count: %d vs %d\n", path, slab1.Count(), slab2.Count())
		diffs++
	}

	entries1 := make(map[string]mapEntry)
	err := slab1.Iterate(nil, newMapDataSlabIndexer(entries1))
	if err != nil {
		panic(fmt.Errorf("Error iterating slab 1: %w\n", err))
	}

	entries2 := make(map[string]mapEntry)
	err = slab2.Iterate(nil, newMapDataSlabIndexer(entries2))
	if err != nil {
		panic(fmt.Errorf("Error iterating slab 2: %w\n", err))
	}

	for encodedKey, entry1 := range entries1 { //nolint:maprange
		entry2, ok := entries2[encodedKey]
		if !ok {
			fmt.Printf("%s[%q]: missing in slab 2\n", path, entry1.key)
			diffs++
			continue
		}

		value1 := entry1.value
		value2 := entry2.value

		diffs += compareChildStorables(
			fmt.Sprintf("%s[%q]", path, entry1.key),
			value1,
			value2,
		)
	}

	for encodedKey, entry2 := range entries2 { //nolint:maprange
		_, ok := entries1[encodedKey]
		if !ok {
			fmt.Printf("%s[%q]: missing in slab 1\n", path, entry2.key)
			diffs++
		}
	}

	return diffs
}

func compareChildStorables(path string, storable1, storable2 atree.Storable) int {
	if mapDataSlabValue1, ok := storable1.(*atree.MapDataSlab); ok {
		if mapDataSlabValue2, ok := storable2.(*atree.MapDataSlab); ok {
			return compareMapDataSlabs(path, mapDataSlabValue1, mapDataSlabValue2)
		}
	}

	if arrayDataSlabValue1, ok := storable1.(*atree.ArrayDataSlab); ok {
		if arrayDataSlabValue2, ok := storable2.(*atree.ArrayDataSlab); ok {
			return compareArrayDataSlabs(path, arrayDataSlabValue1, arrayDataSlabValue2)
		}
	}

	if someStorable1, ok := storable1.(interpreter.SomeStorable); ok {
		if someStorable2, ok := storable2.(interpreter.SomeStorable); ok {
			return compareChildStorables(
				path+".some",
				someStorable1.Storable,
				someStorable2.Storable,
			)
		}
	}

	if !bytes.Equal(encodeStorable(storable1), encodeStorable(storable2)) {
		fmt.Printf(
			"%s: %q (%T) vs %q (%T)\n",
			path,
			storable1,
			storable1,
			storable2,
			storable2,
		)
		return 1
	}

	return 0
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
