package interpreter

import (
	"encoding/binary"
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
	RLPByteRangeStart        = 0x00 // not in use, here only for inclusivity
	RLPByteRangeEnd          = 0x7f
	RLPShortStringRangeStart = 0x80
	RLPShortStringRangeEnd   = 0xb7
	RLPLongStringRangeStart  = 0xb8
	RLPLongStringRangeEnd    = 0xbf
	RLPShortListRangeStart   = 0xc0
	RLPShortListRangeEnd     = 0xf7
	RLPLongListRangeStart    = 0xf8
	RLPLongListRangeEnd      = 0xff // not in use, here only for inclusivity
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

func peakNextType(inp []byte, startIndex int) ItemType {
	// TODO check len
	firstByte := inp[startIndex]
	switch {
	case firstByte < RLPShortStringRangeStart:
		return Byte
	case firstByte < RLPShortListRangeStart:
		return String
	default:
		return List
	}
}

type Item interface {
	Type() ItemType
}

var _ Item = ByteItem(0)
var _ Item = StringItem("")
var _ Item = ListItem{}

type ByteItem uint8

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

// TODO add max size limits

// func RLPDecode(input []byte) (offset int, dataLen int, dataType itemType, err error) {
// 	length = len(input)
// 	if length == 0 {
// 		return 0, 0, 0, errors.New("input is null")
// 	}
// }

// func RLPReadUInt8 ([UInt8Value], rest [UInt8Value], error ) // reading a byte
// RLPReadString (data string, rest [UInt8], error)
// RLPReadList   (array of [UInt8], rest [UInt8], error)

// TODO change inp []byte to the our ideal inputs
// make it a reader

func RLPReadByteItem(inp []byte, startIndex int) (data ByteItem, nextStartIndex int, err error) {
	if startIndex >= len(inp) {
		return 0, startIndex, fmt.Errorf("startIndex error") // TODO make this more formal
	}
	firstByte := inp[startIndex]
	startIndex++
	if firstByte > RLPByteRangeEnd {
		return 0, startIndex, fmt.Errorf("type mismatch")
	}
	return ByteItem(firstByte), startIndex, nil
}

func RLPReadStringItem(inp []byte, startIndex int) (str StringItem, nextStartIndex int, err error) {

	var strLen uint

	if startIndex >= len(inp) {
		return "", startIndex, fmt.Errorf("startIndex error") // TODO make this more formal
	}

	firstByte := inp[startIndex]
	startIndex++

	if firstByte > RLPLongStringRangeEnd {
		return "", startIndex, fmt.Errorf("type mismatch")
	}

	// one byte
	if firstByte < RLPShortStringRangeStart {
		return StringItem(firstByte), startIndex, nil
	}

	// short strings
	// if a string is 0-55 bytes long, the RLP encoding consists
	// of a single byte with value 0x80 plus the length of the string
	// followed by the string. The range of the first byte is thus [0x80, 0xB7].
	if firstByte < RLPLongStringRangeStart {
		strLen = uint(firstByte - RLPShortStringRangeStart)
		// TODO check for non zero len
		endIndex := startIndex + int(strLen)
		if len(inp) < int(endIndex) {
			// TODO validate the range
			return "", startIndex, fmt.Errorf("not enough bytes to read")
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

	bytesToReadForLen := uint(firstByte - RLPShortStringRangeEnd)
	switch bytesToReadForLen {
	case 0:
		// this condition never happens - TODO remove it
		return "", startIndex, fmt.Errorf("invalid string size")

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

	endIndex := startIndex + int(strLen)
	if len(inp) < int(endIndex) {
		// TODO validate the range
		return "", startIndex, fmt.Errorf("not enough bytes to read")
	}
	return StringItem(inp[startIndex:endIndex]), endIndex, nil
}

func RLPReadListItem(inp []byte, startIndex int) (str ListItem, newStartIndex int, err error) {

	var listDataSize uint
	retList := make([]Item, 0)

	if len(inp) == 0 {
		return nil, 0, fmt.Errorf("input is empty")
	}

	firstByte := inp[startIndex]
	startIndex++

	if firstByte < RLPShortListRangeStart {
		return nil, 0, fmt.Errorf("type mismatch")
	}

	if firstByte < RLPLongListRangeStart { // short list
		// TODO check max depth, and max byte readable
		// TODO check for non zero len

		listDataSize = uint(firstByte - RLPShortListRangeStart)
		listDataStartIndex := startIndex
		listDataPrevIndex := startIndex
		bytesRead := 0
		for i := 0; bytesRead < int(listDataSize); i++ {
			itemType := peakNextType(inp, startIndex)
			var item Item
			listDataPrevIndex = listDataStartIndex
			switch itemType {
			case Byte:
				item, listDataStartIndex, err = RLPReadByteItem(inp, listDataStartIndex)
			case String:
				item, listDataStartIndex, err = RLPReadStringItem(inp, listDataStartIndex)
			case List:
				item, listDataStartIndex, err = RLPReadListItem(inp, listDataStartIndex)
			}
			if err != nil {
				return nil, 0, fmt.Errorf("cannot read list item: %w", err)
			}
			retList = append(retList, item)
			bytesRead += listDataStartIndex - listDataPrevIndex
		}

		return retList, listDataStartIndex, nil
	}

	bytesToReadForLen := uint(firstByte - RLPShortListRangeEnd)
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
			item, listDataStartIndex, err = RLPReadByteItem(inp, listDataStartIndex)
		case String:
			item, listDataStartIndex, err = RLPReadStringItem(inp, listDataStartIndex)
		case List:
			item, listDataStartIndex, err = RLPReadListItem(inp, listDataStartIndex)
		}
		if err != nil {
			return nil, 0, fmt.Errorf("cannot read list item: %w", err)
		}
		retList = append(retList, item)
		bytesRead += listDataStartIndex - listDataPrevIndex
	}
	return retList, listDataStartIndex, nil
}
