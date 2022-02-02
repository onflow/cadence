package rlp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ItemType represents the type of an item
type ItemType uint8

// TODO idea: maybe just bytes and list
// and do conversion on bytes type

// TODO idea:
// just expose helper methods, so the user can do it step by step
// this would help with having access to the encoded version of each property

// TODO check all the input ranges

const (
	Bytes ItemType = 0 // what is called string in some other implementations
	List  ItemType = 1
)

const (
	// TODO adjust these numbers based on requirements
	MaxInputByteSize  = 1 << 32
	MaxStringSize     = 1 << 16
	MaxListItemCounts = 1 << 16
	MaxDepthAllowed   = 1 << 16
)

func (it ItemType) String() string {
	switch it {
	case Bytes:
		return "Bytes"
	case List:
		return "List"
	default:
		return fmt.Sprintf("Unknown ItemType (%d)", it)
	}
}

type Item interface {
	Type() ItemType
}

var _ Item = BytesItem("")
var _ Item = ListItem{}

type BytesItem []byte

func (BytesItem) Type() ItemType {
	return Bytes
}

type ListItem []Item

func (ListItem) Type() ItemType {
	return List
}

func (l ListItem) Get(index int) Item {
	return l[index]
}

const (
	ByteRangeStart        = 0x00 // not in use, here only for inclusivity
	ByteRangeEnd          = 0x7f
	ShortStringRangeStart = 0x80
	ShortStringRangeEnd   = 0xb7
	LongStringRangeStart  = 0xb8
	LongStringRangeEnd    = 0xbf
	MaxShortStringLength  = ShortStringRangeEnd - ShortStringRangeStart
	ShortListRangeStart   = 0xc0
	ShortListRangeEnd     = 0xf7
	MaxShortListLength    = ShortListRangeEnd - ShortListRangeStart
	LongListRangeStart    = 0xf8
	LongListRangeEnd      = 0xff // not in use, here only for inclusivity
)

func peekNextType(inp []byte, startIndex int) (ItemType, error) {
	if startIndex >= len(inp) {
		return 0, fmt.Errorf("startIndex error")
	}
	firstByte := inp[startIndex]
	if firstByte < ShortListRangeStart {
		return Bytes, nil
	}
	return List, nil
}

func Decode(inp []byte) (Item, error) {
	if len(inp) == 0 {
		return nil, errors.New("data is empty")
	}
	if len(inp) >= MaxInputByteSize {
		return nil, errors.New("max input size has reached")
	}

	var item Item
	var nextIndex int
	var err error

	nextType, err := peekNextType(inp, 0)
	if err != nil {
		return nil, err
	}

	switch nextType {
	case Bytes:
		item, nextIndex, err = ReadBytesItem(inp, 0)
	case List:
		item, nextIndex, err = ReadListItem(inp, 0, 0)
	}

	if err != nil {
		return nil, err
	}

	if len(inp) != nextIndex {
		return nil, errors.New("unused data in the stream")
	}

	return item, nil
}

func ReadBytesItem(inp []byte, startIndex int) (str BytesItem, nextStartIndex int, err error) {
	// check input size
	if len(inp) >= MaxInputByteSize {
		return nil, 0, errors.New("max input size has reached")
	}

	// check startIndex boundries
	if startIndex >= len(inp) {
		return nil, 0, fmt.Errorf("startIndex error")
	}

	var strLen uint
	firstByte := inp[startIndex]
	startIndex++

	// check the type is right
	if firstByte > LongStringRangeEnd {
		return nil, 0, fmt.Errorf("type mismatch")
	}

	// single character mode
	if firstByte < ShortStringRangeStart {
		return []byte{firstByte}, startIndex, nil
	}

	// short strings mode (0-55 bytes long string)
	// firstByte minus the start range for the short strings would return the data size
	// valid range of firstByte is [0x80, 0xB7].
	if firstByte < LongStringRangeStart {
		strLen = uint(firstByte - ShortStringRangeStart)
		endIndex := startIndex + int(strLen)
		// check endIndex is in the range
		if len(inp) < int(endIndex) {
			return nil, 0, fmt.Errorf("not enough bytes to read")
		}
		return inp[startIndex:endIndex], endIndex, nil
	}

	// long string mode (55+ long strings)
	// firstByte minus the end range of short string, returns the number of bytes
	// we need to read to compute the length of the length of the
	// string in binary form (big endian)
	// e.g. a length-1024 string would be encoded as 0xB9 0x04 0x00 followed by the string.
	// valid range of firstByte is [0xB8, 0xBF].
	bytesToReadForLen := uint(firstByte - ShortStringRangeEnd)
	switch bytesToReadForLen {
	case 0:
		// this condition never happens - TODO remove it
		return nil, 0, fmt.Errorf("invalid string size")

	case 1:
		strLen = uint(inp[startIndex])
		startIndex++
		if strLen <= MaxShortStringLength {
			// encoding is not canonical, unnecessary bytes used for encoding
			// should have encoded as a short string
			return nil, 0, fmt.Errorf("non canonical encoding")
		}

	default:
		// allocate 8 bytes
		lenData := make([]byte, 8)
		// but copy to lower part only
		start := int(8 - bytesToReadForLen)

		// encodign is not canonical, unnecessary bytes used for encoding
		// checking only the first byte ensures that we don't have included
		// trailing empty bytes in the encoding
		if inp[startIndex] == 0 {
			return nil, 0, fmt.Errorf("non canonical encoding")
		}

		copy(lenData[start:], inp[startIndex:startIndex+int(bytesToReadForLen)])
		startIndex += int(bytesToReadForLen)
		strLen = uint(binary.BigEndian.Uint64(lenData))

		if strLen <= MaxShortStringLength {
			// encoding is not canonical, unnecessary bytes used for encoding
			// should have encoded as a short string
			return nil, 0, fmt.Errorf("non canonical encoding")
		}
	}

	// this is not the limit by RLP but a protection for memory
	if strLen >= MaxStringSize {
		return nil, 0, fmt.Errorf("max string size has been hit (this is not a limit of RLP)")
	}

	endIndex := startIndex + int(strLen)
	if len(inp) < int(endIndex) {
		return nil, 0, fmt.Errorf("not enough bytes to read")
	}
	return inp[startIndex:endIndex], endIndex, nil
}

func ReadListItem(inp []byte, startIndex int, depth int) (str ListItem, newStartIndex int, err error) {
	// check input size
	if len(inp) >= MaxInputByteSize {
		return nil, 0, errors.New("max input size has reached")
	}

	// prevents infinite recursive calls
	if depth >= MaxDepthAllowed {
		return nil, 0, errors.New("max depth has been reached")
	}

	// check startIndex boundries
	if startIndex >= len(inp) {
		return nil, 0, fmt.Errorf("startIndex error")
	}

	firstByte := inp[startIndex]
	startIndex++

	if firstByte < ShortListRangeStart {
		return nil, 0, fmt.Errorf("type mismatch")
	}

	var listDataSize uint
	retList := make([]Item, 0)

	// short list mode
	if firstByte < LongListRangeStart {
		listDataSize = uint(firstByte - ShortListRangeStart)
		listDataStartIndex := startIndex
		bytesRead := 0
		for i := 0; bytesRead < int(listDataSize); i++ {
			itemType, err := peekNextType(inp, startIndex)
			if err != nil {
				return nil, 0, err
			}
			var item Item
			var newStartIndex int
			switch itemType {
			case Bytes:
				item, newStartIndex, err = ReadBytesItem(inp, listDataStartIndex)
			case List:
				item, newStartIndex, err = ReadListItem(inp, listDataStartIndex, depth+1)
			}

			if err != nil {
				return nil, 0, fmt.Errorf("cannot read list item: %w", err)
			}
			retList = append(retList, item)
			bytesRead += newStartIndex - listDataStartIndex
			listDataStartIndex = newStartIndex
		}

		// check bytescounts
		if bytesRead != int(listDataSize) {
			return nil, 0, errors.New("more bytes where used by items of the list than what has been reported")
		}

		return retList, listDataStartIndex, nil
	}

	bytesToReadForLen := uint(firstByte - ShortListRangeEnd)
	switch bytesToReadForLen {
	// this case never happens
	// case 0:
	// 	return nil, startIndex, fmt.Errorf("invalid list size")

	case 1:
		listDataSize = uint(inp[startIndex])
		startIndex++
		if listDataSize <= MaxShortListLength {
			// encoding is not canonical, unnecessary bytes used for encoding
			// should have encoded as a short string
			return nil, 0, fmt.Errorf("non canonical encoding")
		}

	default:
		// allocate 8 bytes
		lenData := make([]byte, 8)
		// but copy to lower part only
		start := int(8 - bytesToReadForLen)

		// encodign is not canonical, unnecessary bytes used for encoding
		// checking only the first byte ensures that we don't have included
		// trailing empty bytes in the encoding
		if inp[startIndex] == 0 {
			return nil, 0, fmt.Errorf("non canonical encoding")
		}

		endIndex := startIndex + int(bytesToReadForLen)
		if endIndex > len(inp) {
			return nil, 0, fmt.Errorf("not enough data to read")
		}
		// TODO check on size we want to read
		copy(lenData[start:], inp[startIndex:endIndex])
		startIndex += int(bytesToReadForLen)
		listDataSize = uint(binary.BigEndian.Uint64(lenData))

		if listDataSize <= MaxShortListLength {
			// encoding is not canonical, unnecessary bytes used for encoding
			// should have encoded as a short string
			return nil, 0, fmt.Errorf("non canonical encoding")
		}

	}

	listDataStartIndex := startIndex
	listDataPrevIndex := startIndex
	bytesRead := 0
	for i := 0; bytesRead < int(listDataSize); i++ {
		itemType, err := peekNextType(inp, startIndex)
		if err != nil {
			return nil, 0, err
		}
		var item Item
		listDataPrevIndex = listDataStartIndex
		switch itemType {
		case Bytes:
			item, listDataStartIndex, err = ReadBytesItem(inp, listDataStartIndex)
		case List:
			item, listDataStartIndex, err = ReadListItem(inp, listDataStartIndex, depth+1)
		}
		if err != nil {
			return nil, 0, fmt.Errorf("cannot read list item: %w", err)
		}
		retList = append(retList, item)
		bytesRead += listDataStartIndex - listDataPrevIndex

		// check bytescounts
		if bytesRead != int(listDataSize) {
			return nil, 0, errors.New("more bytes where used by items of the list than what has been reported")
		}
	}
	return retList, listDataStartIndex, nil
}
