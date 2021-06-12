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
	SignatureAlgorithmECDSA_secp256k1,
	SignatureAlgorithmBLS_BLS12_381,
}

var HashAlgorithms = []CryptoAlgorithm{
	HashAlgorithmSHA2_256,
	HashAlgorithmSHA2_384,
	HashAlgorithmSHA3_256,
	HashAlgorithmSHA3_384,
	HashAlgorithmKMAC128_BLS_BLS12_381,
}

var SignatureAlgorithmType = newNativeEnumType(
	SignatureAlgorithmTypeName,
	UInt8Type,
	nil,
)

type SignatureAlgorithm uint8

const (
	// Supported signing algorithms
	SignatureAlgorithmUnknown SignatureAlgorithm = iota
	SignatureAlgorithmECDSA_P256
	SignatureAlgorithmECDSA_secp256k1
	SignatureAlgorithmBLS_BLS12_381
)

// Name returns the string representation of this signing algorithm.
func (algo SignatureAlgorithm) Name() string {
	switch algo {
	case SignatureAlgorithmUnknown:
		return "unknown"
	case SignatureAlgorithmECDSA_P256:
		return "ECDSA_P256"
	case SignatureAlgorithmECDSA_secp256k1:
		return "ECDSA_secp256k1"
	case SignatureAlgorithmBLS_BLS12_381:
		return "BLS_BLS12_381"
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
	case SignatureAlgorithmECDSA_secp256k1:
		return 2
	case SignatureAlgorithmBLS_BLS12_381:
		return 3
	}

	panic(errors.NewUnreachableError())
}

func (algo SignatureAlgorithm) DocString() string {
	switch algo {
	case SignatureAlgorithmUnknown:
		return ""
	case SignatureAlgorithmECDSA_P256:
		return SignatureAlgorithmDocStringECDSA_P256
	case SignatureAlgorithmECDSA_secp256k1:
		return SignatureAlgorithmDocStringECDSA_secp256k1
	case SignatureAlgorithmBLS_BLS12_381:
		return SignatureAlgorithmDocStringBLS_BLS12_381
	}

	panic(errors.NewUnreachableError())
}

const HashAlgorithmTypeHashFunctionName = "hash"

var HashAlgorithmTypeHashFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "data",
			TypeAnnotation: NewTypeAnnotation(
				&VariableSizedType{
					Type: UInt8Type,
				},
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: UInt8Type,
		},
	),
}

const HashAlgorithmTypeHashFunctionDocString = `
Returns the hash of the given data
`

const HashAlgorithmTypeHashWithTagFunctionName = "hashWithTag"

var HashAlgorithmTypeHashWithTagFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "data",
			TypeAnnotation: NewTypeAnnotation(
				&VariableSizedType{
					Type: UInt8Type,
				},
			),
		},
		{
			Identifier:     "tag",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: UInt8Type,
		},
	),
}

const HashAlgorithmTypeHashWithTagFunctionDocString = `
Returns the hash of the given data and tag
`

var HashAlgorithmType = newNativeEnumType(
	HashAlgorithmTypeName,
	UInt8Type,
	func(enumType *CompositeType) []*Member {
		return []*Member{
			NewPublicFunctionMember(
				enumType,
				HashAlgorithmTypeHashFunctionName,
				HashAlgorithmTypeHashFunctionType,
				HashAlgorithmTypeHashFunctionDocString,
			),
			NewPublicFunctionMember(
				enumType,
				HashAlgorithmTypeHashWithTagFunctionName,
				HashAlgorithmTypeHashWithTagFunctionType,
				HashAlgorithmTypeHashWithTagFunctionDocString,
			),
		}
	},
)

type HashAlgorithm uint8

const (
	// Supported hashing algorithms
	HashAlgorithmUnknown HashAlgorithm = iota
	HashAlgorithmSHA2_256
	HashAlgorithmSHA2_384
	HashAlgorithmSHA3_256
	HashAlgorithmSHA3_384
	HashAlgorithmKMAC128_BLS_BLS12_381
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
	case HashAlgorithmKMAC128_BLS_BLS12_381:
		return "KMAC128_BLS_BLS12_381"
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
	case HashAlgorithmKMAC128_BLS_BLS12_381:
		return 5
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
	case HashAlgorithmKMAC128_BLS_BLS12_381:
		return HashAlgorithmDocStringKMAC128_BLS_BLS12_381
	}

	panic(errors.NewUnreachableError())
}

func newNativeEnumType(
	identifier string,
	rawType Type,
	membersConstructor func(enumType *CompositeType) []*Member,
) *CompositeType {
	ty := &CompositeType{
		Identifier:  identifier,
		EnumRawType: rawType,
		Kind:        common.CompositeKindEnum,
		importable:  true,
	}

	// Members of the enum type are *not* the enum cases!
	// Each individual enum case is an instance of the enum type,
	// so only has a single member, the raw value field,
	// plus potentially other fields and functions

	var members []*Member
	if membersConstructor != nil {
		members = membersConstructor(ty)
	}

	members = append(members,
		NewPublicConstantFieldMember(
			ty,
			EnumRawValueFieldName,
			rawType,
			enumRawValueFieldDocString,
		),
	)

	ty.Members = GetMembersAsMap(members)
	ty.Fields = getFieldNames(members)
	return ty
}

const SignatureAlgorithmTypeName = "SignatureAlgorithm"

const SignatureAlgorithmDocStringECDSA_P256 = `
ECDSA_P256 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the NIST P-256 curve
`

const SignatureAlgorithmDocStringECDSA_secp256k1 = `
ECDSA_secp256k1 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the secp256k1 curve
`

const SignatureAlgorithmDocStringBLS_BLS12_381 = `
BLS_BLS12_381 is BLS signature scheme on the BLS12-381 curve.
The scheme is set-up so that signatures are in G_1 (curve over the prime field)
while public keys are in G_2 (curve over the prime field extension).
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

const HashAlgorithmDocStringKMAC128_BLS_BLS12_381 = `
KMAC128_BLS_BLS12_381 is an instance of KECCAK Message Authentication Code (KMAC128) mac algorithm,
that can be used as the hashing algorithm for BLS signature scheme on the curve BLS12-381.
This is a customized version of KMAC128 that is compatible with the hashing to curve 
used in BLS signatures.
`
