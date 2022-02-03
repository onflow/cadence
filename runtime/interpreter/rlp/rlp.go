/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package rlp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	ByteRangeStart        = 0x00 // not in use, here only for inclusivity
	ByteRangeEnd          = 0x7f
	ShortStringRangeStart = 0x80
	ShortStringRangeEnd   = 0xb7
	LongStringRangeStart  = 0xb8
	LongStringRangeEnd    = 0xbf
	ShortListRangeStart   = 0xc0
	ShortListRangeEnd     = 0xf7
	LongListRangeStart    = 0xf8
	LongListRangeEnd      = 0xff // not in use, here only for inclusivity
	MaxShortLengthAllowed = 55
)

// TODO make error messages better

func rlpReadSize(inp []byte, startIndex int) (isString bool, dataStartIndex, dataSize int, err error) {
	// check startIndex is in the range
	if startIndex >= len(inp) {
		return false, 0, 0, fmt.Errorf("startIndex error")
	}

	firstByte := inp[startIndex]
	startIndex++

	// single character mode - first byte is the only data
	if firstByte < ShortStringRangeStart {
		return true, startIndex - 1, 1, nil
	}

	// short strings mode (0-55 bytes long string)
	// firstByte minus the start range for the short strings would return the data size
	// valid range of firstByte is [0x80, 0xB7].
	if firstByte < LongStringRangeStart {
		strLen := uint(firstByte - ShortStringRangeStart)
		return true, startIndex, int(strLen), nil
	}

	// short list mode
	// firstByte minus the start range for the short list would return the data size
	if firstByte >= ShortListRangeStart && firstByte <= ShortListRangeEnd {
		strLen := uint(firstByte - ShortListRangeStart)
		return false, startIndex, int(strLen), nil
	}

	// string and list long mode

	var bytesToReadForLen uint
	// long string mode (55+ long strings)
	// firstByte minus the end range of short string, returns the number of bytes
	if firstByte >= LongStringRangeStart && firstByte <= LongStringRangeEnd {
		bytesToReadForLen = uint(firstByte - ShortStringRangeEnd)
		isString = true
	}

	// long list mode
	if firstByte >= LongListRangeStart {
		bytesToReadForLen = uint(firstByte - ShortStringRangeEnd)
		isString = false
	}

	// check atleast 1 byte is there to read
	if int(startIndex) >= len(inp) {
		return false, 0, 0, fmt.Errorf("not enough bytes to read string size")
	}

	// bytesToReadForLen with value of zero never happens
	// optimization for a single extra byte for size
	if bytesToReadForLen == 1 {
		strLen := uint(inp[startIndex])
		startIndex++
		if strLen <= MaxShortLengthAllowed {
			// encoding is not canonical, unnecessary bytes used for encoding
			// should have encoded as a short string
			return false, 0, 0, fmt.Errorf("non canonical encoding")
		}
		return isString, startIndex, int(strLen), nil
	}

	// several bytes case

	// allocate 8 bytes
	lenData := make([]byte, 8)
	// but copy to lower part only
	start := int(8 - bytesToReadForLen)

	// encodign is not canonical, unnecessary bytes used for encoding
	// checking only the first byte ensures that we don't have included
	// trailing empty bytes in the encoding
	if inp[startIndex] == 0 {
		return false, 0, 0, fmt.Errorf("non canonical encoding")
	}

	endIndex := startIndex + int(bytesToReadForLen)
	// check endIndex is in the range
	if int(endIndex) > len(inp) {
		return false, 0, 0, fmt.Errorf("not enough bytes to read string size 2")
	}

	copy(lenData[start:], inp[startIndex:endIndex])
	startIndex += int(bytesToReadForLen)
	strLen := uint(binary.BigEndian.Uint64(lenData))

	if strLen <= MaxShortLengthAllowed {
		// encoding is not canonical, unnecessary bytes used for encoding
		// should have encoded as a short string
		return false, 0, 0, fmt.Errorf("non canonical encoding")
	}
	return isString, startIndex, int(strLen), nil
}

func RLPDecodeString(inp []byte, startIndex int) (str []byte, nextStartIndex int, err error) {
	// read data size info
	isString, dataStartIndex, dataSize, err := rlpReadSize(inp, startIndex)
	if err != nil {
		return nil, 0, fmt.Errorf("decode string failed: %w", err)
	}
	// check type
	if !isString {
		return nil, 0, errors.New("type mismatch, expected string but got list")
	}
	// single character special case
	if dataSize == 1 && startIndex == dataStartIndex {
		return []byte{inp[dataStartIndex]}, startIndex + 1, nil
	}

	// collect and return string
	dataEndIndex := dataStartIndex + dataSize
	if dataEndIndex > len(inp) {
		return nil, 0, fmt.Errorf("not enough bytes to read string data")
	}
	return inp[dataStartIndex:dataEndIndex], dataEndIndex, nil
}

func RLPDecodeList(inp []byte, startIndex int) (encodedItems [][]byte, newStartIndex int, err error) {
	// read data size info
	isString, dataStartIndex, listDataSize, err := rlpReadSize(inp, startIndex)
	if err != nil {
		return nil, 0, fmt.Errorf("decode string failed: %w", err)
	}

	// check type
	if isString {
		return nil, 0, errors.New("type mismatch, expected list but got string")
	}

	itemStartIndex := dataStartIndex
	bytesRead := 0
	retList := make([][]byte, 0)

	for bytesRead < int(listDataSize) {

		_, itemDataStartIndex, itemSize, err := rlpReadSize(inp, itemStartIndex)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot read list item: %w", err)
		}
		// collect encoded item
		itemEndIndex := itemDataStartIndex + itemSize
		if itemEndIndex > len(inp) {
			return nil, 0, fmt.Errorf("not enough bytes to read string data")
		}
		retList = append(retList, inp[itemDataStartIndex:itemEndIndex])
		bytesRead += itemEndIndex - itemStartIndex
		itemStartIndex = itemEndIndex
	}

	return retList, itemStartIndex, nil
}
