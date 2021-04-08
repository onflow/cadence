/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package sema

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=SignatureAlgorithm
//go:generate go run golang.org/x/tools/cmd/stringer -type=HashAlgorithm

var SignatureAlgorithms = []CryptoAlgorithm{
	SignatureAlgorithmECDSA_P256,
	SignatureAlgorithmECDSA_Secp256k1,
}

var HashAlgorithms = []CryptoAlgorithm{
	HashAlgorithmSHA2_256,
	HashAlgorithmSHA2_384,
	HashAlgorithmSHA3_256,
	HashAlgorithmSHA3_384,
}

var SignatureAlgorithmType = newNativeEnumType(SignatureAlgorithmTypeName, UInt8Type)

type SignatureAlgorithm uint8

const (
	// Supported signing algorithms
	SignatureAlgorithmUnknown SignatureAlgorithm = iota
	SignatureAlgorithmECDSA_P256
	SignatureAlgorithmECDSA_Secp256k1
)

// Name returns the string representation of this signing algorithm.
func (algo SignatureAlgorithm) Name() string {
	switch algo {
	case SignatureAlgorithmUnknown:
		return "unknown"
	case SignatureAlgorithmECDSA_P256:
		return "ECDSA_P256"
	case SignatureAlgorithmECDSA_Secp256k1:
		return "ECDSA_Secp256k1"
	}

	panic(errors.NewUnreachableError())
}

func (algo SignatureAlgorithm) RawValue() uint8 {
	// NOTE: only add new algorithms, do *NOT* change existing items,
	// reuse raw values for other items, swap the order, etc.
	//
	// Existing stored values use these raw values and should not change

	switch algo {
	case SignatureAlgorithmUnknown:
		return 0
	case SignatureAlgorithmECDSA_P256:
		return 1
	case SignatureAlgorithmECDSA_Secp256k1:
		return 2
	}

	panic(errors.NewUnreachableError())
}

func (algo SignatureAlgorithm) DocString() string {
	switch algo {
	case SignatureAlgorithmUnknown:
		return ""
	case SignatureAlgorithmECDSA_P256:
		return SignatureAlgorithmDocStringECDSA_P256
	case SignatureAlgorithmECDSA_Secp256k1:
		return SignatureAlgorithmDocStringECDSA_Secp256k1
	}

	panic(errors.NewUnreachableError())
}

var HashAlgorithmType = newNativeEnumType(HashAlgorithmTypeName, UInt8Type)

type HashAlgorithm uint8

const (
	// Supported hashing algorithms
	HashAlgorithmUnknown HashAlgorithm = iota
	HashAlgorithmSHA2_256
	HashAlgorithmSHA2_384
	HashAlgorithmSHA3_256
	HashAlgorithmSHA3_384
)

func (algo HashAlgorithm) Name() string {
	switch algo {
	case HashAlgorithmUnknown:
		return "unknown"
	case HashAlgorithmSHA2_256:
		return "SHA2_256"
	case HashAlgorithmSHA2_384:
		return "SHA2_384"
	case HashAlgorithmSHA3_256:
		return "SHA3_256"
	case HashAlgorithmSHA3_384:
		return "SHA3_384"
	}

	panic(errors.NewUnreachableError())
}

func (algo HashAlgorithm) RawValue() uint8 {
	// NOTE: only add new algorithms, do *NOT* change existing items,
	// reuse raw values for other items, swap the order, etc.
	//
	// Existing stored values use these raw values and should not change

	switch algo {
	case HashAlgorithmUnknown:
		return 0
	case HashAlgorithmSHA2_256:
		return 1
	case HashAlgorithmSHA2_384:
		return 2
	case HashAlgorithmSHA3_256:
		return 3
	case HashAlgorithmSHA3_384:
		return 4
	}

	panic(errors.NewUnreachableError())
}

func (algo HashAlgorithm) DocString() string {
	switch algo {
	case HashAlgorithmUnknown:
		return ""
	case HashAlgorithmSHA2_256:
		return HashAlgorithmDocStringSHA2_256
	case HashAlgorithmSHA2_384:
		return HashAlgorithmDocStringSHA2_384
	case HashAlgorithmSHA3_256:
		return HashAlgorithmDocStringSHA3_256
	case HashAlgorithmSHA3_384:
		return HashAlgorithmDocStringSHA3_384
	}

	panic(errors.NewUnreachableError())
}

func newNativeEnumType(identifier string, rawType Type) *CompositeType {
	accountKeyType := &CompositeType{
		Identifier:  identifier,
		EnumRawType: rawType,
		Kind:        common.CompositeKindEnum,
	}

	// Members of the enum type are *not* the enum cases!
	// Each individual enum case is an instance of the enum type,
	// so only has a single member, the raw value field
	var members = []*Member{
		NewPublicEnumCaseMember(
			rawType,
			EnumRawValueFieldName,
			enumRawValueFieldDocString,
		),
	}

	accountKeyType.Members = GetMembersAsMap(members)
	accountKeyType.Fields = getFieldNames(members)
	return accountKeyType
}

const SignatureAlgorithmTypeName = "SignatureAlgorithm"

const SignatureAlgorithmDocStringECDSA_P256 = `
ECDSA_P256 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the NIST P-256 curve
`

const SignatureAlgorithmDocStringECDSA_Secp256k1 = `
ECDSA_Secp256k1 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the secp256k1 curve
`

const HashAlgorithmTypeName = "HashAlgorithm"

const HashAlgorithmDocStringSHA2_256 = `
SHA2_256 is Secure Hashing Algorithm 2 (SHA-2) with a 256-bit digest
`

const HashAlgorithmDocStringSHA2_384 = `
SHA2_384 is Secure Hashing Algorithm 2 (SHA-2) with a 384-bit digest
`

const HashAlgorithmDocStringSHA3_256 = `
SHA3_256 is Secure Hashing Algorithm 3 (SHA-3) with a 256-bit digest
`

const HashAlgorithmDocStringSHA3_384 = `
SHA3_384 is Secure Hashing Algorithm 3 (SHA-3) with a 384-bit digest
`
