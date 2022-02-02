package rlp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ItemType represents the type of an item
type ItemType uint8

const (
	Byte   ItemType = 0
	String ItemType = 1
	List   ItemType = 2
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
	case Byte:
		return "Byte"
	case String:
		return "String"
	case List:
		return "List"
	default:
		return fmt.Sprintf("Unknown ItemType (%d)", it)
	}
}

type Item interface {
	Type() ItemType
}

var _ Item = ByteItem(0)
var _ Item = StringItem("")
var _ Item = ListItem{}

type ByteItem int

func (ByteItem) Type() ItemType {
	return Byte
}

type StringItem string

func (StringItem) Type() ItemType {
	return String
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
	ShortListRangeStart   = 0xc0
	ShortListRangeEnd     = 0xf7
	LongListRangeStart    = 0xf8
	LongListRangeEnd      = 0xff // not in use, here only for inclusivity
)

func peakNextType(inp []byte, startIndex int) ItemType {
	// TODO check len
	firstByte := inp[startIndex]
	switch {
	case firstByte < ShortStringRangeStart:
		return Byte
	case firstByte < ShortListRangeStart:
		return String
	default:
		return List
	}
}

// TODO add max size limits

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
	switch peakNextType(inp, 0) {
	case Byte:
		item, nextIndex, err = ReadByteItem(inp, 0)
	case String:
		item, nextIndex, err = ReadStringItem(inp, 0)
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

func ReadByteItem(inp []byte, startIndex int) (data ByteItem, nextStartIndex int, err error) {
	if startIndex >= len(inp) {
		return 0, startIndex, fmt.Errorf("startIndex error") // TODO make this more formal
	}
	if len(inp) >= MaxInputByteSize {
		return 0, 0, errors.New("max input size has reached")
	}
	firstByte := inp[startIndex]
	startIndex++
	if firstByte > ByteRangeEnd {
		return 0, startIndex, fmt.Errorf("type mismatch")
	}
	return ByteItem(firstByte), startIndex, nil
}

func ReadStringItem(inp []byte, startIndex int) (str StringItem, nextStartIndex int, err error) {
	if startIndex >= len(inp) {
		return "", 0, fmt.Errorf("startIndex error") // TODO make this more formal
	}
	if len(inp) >= MaxInputByteSize {
		return "", 0, errors.New("max input size has reached")
	}

	var strLen uint
	firstByte := inp[startIndex]
	startIndex++

	if firstByte > LongStringRangeEnd {
		return "", 0, fmt.Errorf("type mismatch")
	}

	// one byte
	if firstByte < ShortStringRangeStart {
		return StringItem(firstByte), startIndex, nil
	}

	// short strings
	// if a string is 0-55 bytes long, the RLP encoding consists
	// of a single byte with value 0x80 plus the length of the string
	// followed by the string. The range of the first byte is thus [0x80, 0xB7].
	if firstByte < LongStringRangeStart {
		strLen = uint(firstByte - ShortStringRangeStart)
		// TODO check for non zero len
		endIndex := startIndex + int(strLen)
		if len(inp) < int(endIndex) {
			// TODO validate the range
			return "", 0, fmt.Errorf("not enough bytes to read")
		}
		return StringItem(inp[startIndex:endIndex]), endIndex, nil
	}

	// long string otherwise
	// If a string is more than 55 bytes long, the RLP encoding consists of a
	// single byte with value 0xB7 plus the length of the length of the
	// string in binary form (big endian), followed by the length of the string, followed
	// by the string. For example, a length-1024 string would be encoded as
	// 0xB90400 followed by the string. The range of the first byte is thus
	// [0xB8, 0xBF].

	bytesToReadForLen := uint(firstByte - ShortStringRangeEnd)
	switch bytesToReadForLen {
	case 0:
		// this condition never happens - TODO remove it
		return "", 0, fmt.Errorf("invalid string size")

	case 1:
		strLen = uint(inp[startIndex])
		startIndex++

	default:
		// allocate 8 bytes
		lenData := make([]byte, 8)
		// but copy to lower part only
		start := int(8 - bytesToReadForLen)

		// TODO check on size we want to read
		copy(lenData[start:], inp[startIndex:startIndex+int(bytesToReadForLen)])
		startIndex += int(bytesToReadForLen)
		strLen = uint(binary.BigEndian.Uint64(lenData))
	}

	if strLen >= MaxStringSize {
		return "", 0, fmt.Errorf("max string size has been hit")
	}

	endIndex := startIndex + int(strLen)
	if len(inp) < int(endIndex) {
		// TODO validate the range
		return "", 0, fmt.Errorf("not enough bytes to read")
	}
	return StringItem(inp[startIndex:endIndex]), endIndex, nil
}

func ReadListItem(inp []byte, startIndex int, depth int) (str ListItem, newStartIndex int, err error) {
	if len(inp) == 0 {
		return nil, 0, fmt.Errorf("input is empty")
	}
	if len(inp) >= MaxInputByteSize {
		return nil, 0, errors.New("max input size has reached")
	}
	if depth >= MaxDepthAllowed {
		return nil, 0, errors.New("max depth has been reached")
	}
	var listDataSize uint
	retList := make([]Item, 0)

	firstByte := inp[startIndex]
	startIndex++

	if firstByte < ShortListRangeStart {
		return nil, 0, fmt.Errorf("type mismatch")
	}

	if firstByte < LongListRangeStart { // short list
		// TODO check max depth, and max byte readable
		// TODO check for non zero len

		listDataSize = uint(firstByte - ShortListRangeStart)
		listDataStartIndex := startIndex
		listDataPrevIndex := startIndex
		bytesRead := 0
		for i := 0; bytesRead < int(listDataSize); i++ {
			itemType := peakNextType(inp, startIndex)
			var item Item
			listDataPrevIndex = listDataStartIndex
			switch itemType {
			case Byte:
				item, listDataStartIndex, err = ReadByteItem(inp, listDataStartIndex)
			case String:
				item, listDataStartIndex, err = ReadStringItem(inp, listDataStartIndex)
			case List:
				item, listDataStartIndex, err = ReadListItem(inp, listDataStartIndex, depth+1)
			}
			if err != nil {
				return nil, 0, fmt.Errorf("cannot read list item: %w", err)
			}
			retList = append(retList, item)
			bytesRead += listDataStartIndex - listDataPrevIndex
		}

		return retList, listDataStartIndex, nil
	}

	bytesToReadForLen := uint(firstByte - ShortListRangeEnd)
	// TODO
	// if bytesToReadForLen < 56 {
	// 	// error canonical size ????
	// }
	switch bytesToReadForLen {
	case 0:
		return nil, startIndex, fmt.Errorf("invalid list size")

	case 1:
		listDataSize = uint(inp[startIndex])
		startIndex++

	default:
		// allocate 8 bytes
		lenData := make([]byte, 8)
		// but copy to lower part only
		start := int(8 - bytesToReadForLen)

		// TODO check on size we want to read
		copy(lenData[start:], inp[startIndex:startIndex+int(bytesToReadForLen)])
		startIndex += int(bytesToReadForLen)
		listDataSize = uint(binary.BigEndian.Uint64(lenData))
	}

	// TODO check max depth, and max byte readable
	// TODO check for non zero len
	listDataStartIndex := startIndex
	listDataPrevIndex := startIndex
	bytesRead := 0
	for i := 0; bytesRead < int(listDataSize); i++ {
		itemType := peakNextType(inp, startIndex)
		var item Item
		listDataPrevIndex = listDataStartIndex
		switch itemType {
		case Byte:
			item, listDataStartIndex, err = ReadByteItem(inp, listDataStartIndex)
		case String:
			item, listDataStartIndex, err = ReadStringItem(inp, listDataStartIndex)
		case List:
			item, listDataStartIndex, err = ReadListItem(inp, listDataStartIndex, depth+1)
		}
		if err != nil {
			return nil, 0, fmt.Errorf("cannot read list item: %w", err)
		}
		retList = append(retList, item)
		bytesRead += listDataStartIndex - listDataPrevIndex
	}
	return retList, listDataStartIndex, nil
}
