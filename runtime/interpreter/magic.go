package interpreter

import (
	"bytes"
	"encoding/binary"
)

// Magic is the prefix that is added to all encoded values
//
var Magic = []byte{0x0, 0xCA, 0xDE}
var MagicLength = len(Magic)

const CurrentEncodingVersion uint16 = 2
const VersionEncodingLength = 2

var fullPrefixLength = MagicLength + VersionEncodingLength

// HasMagic tests whether the given data  begins with the magic prefix.
//
func HasMagic(data []byte) bool {
	return bytes.HasPrefix(data, Magic)
}

// StripMagic returns the given data with the magic prefix and version removed.
//
// If the data doesn't start with Magic, the data is returned unchanged
// and the version is 0.
//
func StripMagic(data []byte) (trimmed []byte, version uint16) {
	if !HasMagic(data) || len(data) < fullPrefixLength {
		return data, 0
	}

	version = binary.BigEndian.Uint16(data[MagicLength:fullPrefixLength])

	return data[fullPrefixLength:], version

}

// PrependMagic returns the given data with the magic prefix.
// The function does *not* check if the data already has the prefix.
//
func PrependMagic(unprefixedData []byte, version uint16) (result []byte) {
	result = make([]byte, fullPrefixLength+len(unprefixedData))
	copy(result[:MagicLength], Magic)
	binary.BigEndian.PutUint16(result[MagicLength:fullPrefixLength], version)
	copy(result[fullPrefixLength:], unprefixedData)
	return result
}
