package interpreter

import (
	"bytes"
)

// Magic is the prefix that is added to all encoded values
//
var Magic = []byte{0x0, 0xCA, 0xDE, 0x0, 0x1}
var MagicLength = len(Magic)

// HasMagic tests whether the given data  begins with the magic prefix.
//
func HasMagic(data []byte) bool {
	return bytes.HasPrefix(data, Magic)
}

// StripMagic returns the given data without the magic prefix.
// If the data doesn't start with Magic, the data is returned unchanged.
//
func StripMagic(data []byte) []byte {
	return bytes.TrimPrefix(data, Magic)
}

// PrependMagic returns the given data with the magic prefix.
// The function does *not* check if the data already has the prefix.
//
func PrependMagic(unprefixedData []byte) (result []byte) {
	result = make([]byte, MagicLength+len(unprefixedData))
	copy(result[:MagicLength], Magic)
	copy(result[MagicLength:], unprefixedData)
	return result
}
