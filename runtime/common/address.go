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

package common

import (
	"encoding/hex"
	goErrors "errors"
	"fmt"
	"strings"
)

var addressOverflowError = goErrors.New("address too large")

const AddressLength = 8

type Address [AddressLength]byte

// MustBytesToAddress returns Address with value b.
//
// If the address is too large, then the function panics.
//
func MustBytesToAddress(b []byte) Address {
	address, err := BytesToAddress(b)
	if err != nil {
		panic(err)
	}
	return address
}

// BytesToAddress returns Address with value b.
//
// If the address is too large, then the function returns an error.
//
func BytesToAddress(b []byte) (Address, error) {
	if len(b) > AddressLength {
		return Address{}, addressOverflowError
	}
	var a Address
	a.SetBytes(b)
	return a, nil
}

// Hex returns the hex string representation of the address.
func (a Address) Hex() string {
	return fmt.Sprintf("%x", a[:])
}

func (a Address) String() string {
	return a.Hex()
}

// SetBytes sets the address to the value of b.
//
// If b is larger than len(a) it will panic.
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}

	copy(a[AddressLength-len(b):], b)
}

// Bytes returns address without leading zeros.
// The fast path is inlined and handles the most common
// scenario of address having no leading zeros.
// Otherwise, bytes() is called to trim leading zeros.
func (a Address) Bytes() []byte {
	if a[0] != 0 {
		return a[:]
	}
	return a.bytes()
}

// bytes returns address bytes after trimming leading zeros.
func (a *Address) bytes() []byte {
	// Trim leading zeros
	leadingZeros := 0
	for _, b := range a {
		if b != 0 {
			break
		}
		leadingZeros += 1
	}

	return a[leadingZeros:]
}

func (a Address) ShortHexWithPrefix() string {
	hexString := fmt.Sprintf("%x", [AddressLength]byte(a))
	return fmt.Sprintf("0x%s", strings.TrimLeft(hexString, "0"))
}

func (a Address) HexWithPrefix() string {
	return fmt.Sprintf("0x%x", [AddressLength]byte(a))
}

// HexToAddress converts a hex string to an Address.
func HexToAddress(h string) (Address, error) {
	trimmed := strings.TrimPrefix(h, "0x")
	trimmed = strings.TrimPrefix(trimmed, "Fx")
	if len(trimmed)%2 == 1 {
		trimmed = "0" + trimmed
	}
	b, err := hex.DecodeString(trimmed)
	if err != nil {
		return Address{}, err
	}
	return BytesToAddress(b)
}
