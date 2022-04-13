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

package rlp

import (
	"encoding/binary"
	"errors"
	"math"
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
	MaxLongLengthAllowed  = math.MaxInt64
)

var (
	ErrEmptyInput        = errors.New("input data is empty")
	ErrInvalidStartIndex = errors.New("invalid start index")
	ErrIncompleteInput   = errors.New("incomplete input! not enough bytes to read")
	ErrNonCanonicalInput = errors.New("non-canonical encoded input")
	ErrDataSizeTooLarge  = errors.New("data size is larger than what is supported")
	ErrListSizeMismatch  = errors.New("list size doesn't match the size of items")
	ErrTypeMismatch      = errors.New("type extracted from input doesn't match the function")
)

// ReadSize looks at the first byte at startIndex to decode the type and reads as many bytes as needed
// to determine the data byte size, it returns a flag if the type is string, start index of data part in the input,
// number of bytes that has to be read for data (from start index of data) and error if any.
//
// it only supports RLP canonical form.  RLP canonical form requires:
//   - if string is 0-55 bytes long (first byte is [0x80, 0xb7]), string length can't be 1 while string <= 0x7f
//   - if string is more than 55 bytes long (first byte is [0xb8, 0xbf]), string length can't be <= 55
//   - if string is more than 55 bytes long (first byte is [0xb8, 0xbf]), string length can't be encoded with leading 0s
//   - if list payload is more than 55 bytes long (first byte is [0xf8, 0xff]), list payload length can't be <= 55
//   - if list payload is more than 55 bytes long (first byte is [0xf8, 0xff]), list payload length can't be encoded with leading 0s
func ReadSize(inp []byte, startIndex int) (isString bool, dataStartIndex, dataSize int, err error) {
	if len(inp) == 0 {
		return false, 0, 0, ErrEmptyInput
	}

	// check startIndex is in the range
	if startIndex >= len(inp) {
		return false, 0, 0, ErrInvalidStartIndex
	}

	firstByte := inp[startIndex]
	startIndex++

	// single character space - first byte holds the data itslef
	if firstByte <= ByteRangeEnd {
		return true, startIndex - 1, 1, nil
	}

	// short string space (0-55 bytes long string)
	// firstByte minus the start range for the short string returns the data size
	// valid range of firstByte is [0x80, 0xB7].
	if firstByte <= ShortStringRangeEnd {
		strLen := uint(firstByte - ShortStringRangeStart)
		return true, startIndex, int(strLen), nil
	}

	// short list space
	// firstByte minus the start range for the short list would return the data size
	if firstByte >= ShortListRangeStart && firstByte <= ShortListRangeEnd {
		strLen := uint(firstByte - ShortListRangeStart)
		return false, startIndex, int(strLen), nil
	}

	// string and list long space

	var bytesToReadForLen uint
	// long string mode (55+ long strings)
	// firstByte minus the end range of short string, returns the number of bytes to read
	// for calculating the the len of data. bytesToReadForlen is at least 1 and at most 8.
	if firstByte >= LongStringRangeStart && firstByte <= LongStringRangeEnd {
		bytesToReadForLen = uint(firstByte - ShortStringRangeEnd)
		isString = true
	}

	// long list mode
	// firstByte minus the end range of short list, returns the number of bytes to read
	// for calculating the the len of data. bytesToReadForlen is at least 1 and at most 8.
	if firstByte >= LongListRangeStart {
		bytesToReadForLen = uint(firstByte - ShortListRangeEnd)
		isString = false
	}

	// check atleast there is one more byte to read
	if startIndex >= len(inp) {
		return false, 0, 0, ErrIncompleteInput
	}

	// bytesToReadForLen with value of zero never happens
	// optimization for a single extra byte for size
	if bytesToReadForLen == 1 {
		strLen := uint(inp[startIndex])
		startIndex++
		if strLen <= MaxShortLengthAllowed {
			// encoding is not canonical, unnecessary bytes used for encoding
			// should have encoded as a short string
			return false, 0, 0, ErrNonCanonicalInput
		}
		return isString, startIndex, int(strLen), nil
	}

	// several bytes case
	// note that its not possible for bytesToReadForLen to go beyond 8
	start := int(8 - bytesToReadForLen)

	// if any trailing zero bytes, unnecessary bytes were used for encoding
	// checking only the first byte is sufficient
	if inp[startIndex] == 0 {
		return false, 0, 0, ErrNonCanonicalInput
	}

	endIndex := startIndex + int(bytesToReadForLen)
	// check endIndex is in the range
	if endIndex > len(inp) {
		return false, 0, 0, ErrIncompleteInput
	}

	// allocate 8 bytes
	lenData := make([]byte, 8)
	// but copy to lower part only
	copy(lenData[start:], inp[startIndex:endIndex])

	startIndex += int(bytesToReadForLen)
	strLen := uint(binary.BigEndian.Uint64(lenData))

	// no need to check strLen <= MaxShortLengthAllowed since bytesToReadForLen is at least 2 here
	// and can not contain leading zero byte.
	if strLen > MaxLongLengthAllowed {
		return false, 0, 0, ErrDataSizeTooLarge
	}
	return isString, startIndex, int(strLen), nil
}

// DecodeString decodes a RLP-encoded string given the startIndex
// it returns decoded string, number of bytes that were read and err if any
func DecodeString(inp []byte, startIndex int) (str []byte, bytesRead int, err error) {
	// read data size info
	isString, dataStartIndex, dataSize, err := ReadSize(inp, startIndex)
	if err != nil {
		return nil, 0, err
	}
	// check type
	if !isString {
		return nil, 0, ErrTypeMismatch
	}

	// single character special case
	if dataSize == 1 && startIndex == dataStartIndex {
		return []byte{inp[dataStartIndex]}, 1, nil
	}

	// if for data we have to read only a single extra byte and that byte
	// is in the range of characters, we are using two bytes instead of 1 byte
	if dataSize == 1 && inp[dataStartIndex] <= ByteRangeEnd {
		return nil, 0, ErrNonCanonicalInput
	}

	// collect and return string
	dataEndIndex := dataStartIndex + dataSize
	if dataEndIndex > len(inp) {
		return nil, 0, ErrIncompleteInput
	}

	return inp[dataStartIndex:dataEndIndex], dataEndIndex - startIndex, nil
}

// DecodeList decodes a RLP-encoded list given the startIndex
// it returns a list of encodedItems, number of bytes that were read and err if any
func DecodeList(inp []byte, startIndex int) (encodedItems [][]byte, bytesRead int, err error) {
	// read data size info
	isString, dataStartIndex, listDataSize, err := ReadSize(inp, startIndex)
	if err != nil {
		return nil, 0, err
	}

	// check type
	if isString {
		return nil, 0, ErrTypeMismatch
	}

	retList := make([][]byte, 0)

	// special case - empty list
	if listDataSize == 0 {
		return retList, 1, nil
	}

	if listDataSize+dataStartIndex > len(inp) {
		return nil, 0, ErrIncompleteInput
	}

	var itemStartIndex, itemEndIndex, dataBytesRead int
	itemStartIndex = dataStartIndex

	for dataBytesRead < listDataSize {
		_, itemDataStartIndex, itemSize, err := ReadSize(inp, itemStartIndex)
		if err != nil {
			return nil, 0, err
		}
		// collect encoded item
		itemEndIndex = itemDataStartIndex + itemSize
		if itemEndIndex > len(inp) {
			return nil, 0, ErrIncompleteInput
		}
		retList = append(retList, inp[itemStartIndex:itemEndIndex])
		dataBytesRead += itemEndIndex - itemStartIndex
		itemStartIndex = itemEndIndex
	}
	if dataBytesRead != listDataSize {
		return nil, 0, ErrListSizeMismatch
	}

	return retList, itemEndIndex - startIndex, nil
}
