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
		panic("Usage: decode-slab <address-hex> <index> <data-hex>")
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
		panic(fmt.Errorf("failed to parse data: %w", err))
	}

	var slabIndex atree.SlabIndex
	binary.BigEndian.PutUint64(slabIndex[:], uint64(index))

	slabID := atree.NewSlabID(
		atree.Address(address),
		slabIndex,
	)

	slab, err := decodeSlab(slabID, data)
	if err != nil {
		panic(fmt.Errorf("failed to decode slab %s: %w", slabID, err))
	}

	fmt.Print(slab)
}
