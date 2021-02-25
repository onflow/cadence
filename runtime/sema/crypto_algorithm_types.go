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

//go:generate go run golang.org/x/tools/cmd/stringer -type=SignatureAlgorithm
//go:generate go run golang.org/x/tools/cmd/stringer -type=HashAlgorithm

var SignatureAlgorithms = []BuiltinEnumCase{
	ECDSA_P256,
	ECDSA_Secp256k1,
}

var HashAlgorithms = []BuiltinEnumCase{
	SHA2_256,
	SHA3_256,
}

const SignatureAlgorithmTypeName = "SignatureAlgorithm2"

const SignatureAlgorithmDocStringECDSA_P256 = `
ECDSA_P256 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the NIST P-256 curve
`
const SignatureAlgorithmDocStringECDSA_Secp256k1 = `
ECDSA_Secp256k1 is Elliptic Curve Digital Signature Algorithm (ECDSA) on the secp256k1 curve
`

var SignatureAlgorithmType = newEnumType(SignatureAlgorithmTypeName, &UInt8Type{})

type SignatureAlgorithm int

const (
	// Supported signing algorithms
	ECDSA_P256 SignatureAlgorithm = iota
	ECDSA_Secp256k1
	//BLSBLS12381
)

// Name returns the string representation of this signing algorithm.
func (algo SignatureAlgorithm) Name() string {
	return algo.String()
}

func (algo SignatureAlgorithm) RawValue() int {
	return int(algo)
}

func (algo SignatureAlgorithm) DocString() string {
	// NOTE: Define this in the same order as the `SignatureAlgorithm` iota
	return [...]string{
		SignatureAlgorithmDocStringECDSA_P256,
		SignatureAlgorithmDocStringECDSA_Secp256k1,
	}[algo]
}

const HashAlgorithmTypeName = "HashAlgorithm2"

const HashAlgorithmDocStringSHA2_256 = `
SHA2_256 is Secure Hashing Algorithm 2 (SHA-2) with a 256-bit digest
`
const HashAlgorithmDocStringSHA3_256 = `
SHA3_256 is Secure Hashing Algorithm 3 (SHA-3) with a 256-bit digest
`

// PublicKeyType represents the public key associated with an account key.
var HashAlgorithmType = newEnumType(HashAlgorithmTypeName, &UInt8Type{})

type HashAlgorithm int

const (
	// Supported hashing algorithms
	SHA2_256 HashAlgorithm = iota
	SHA3_256
	//KMAC128
	//SHA3_384
	//SHA2_384
)

func (algo HashAlgorithm) Name() string {
	return algo.String()
}

func (algo HashAlgorithm) RawValue() int {
	return int(algo)
}

func (algo HashAlgorithm) DocString() string {
	// NOTE: Define this in the same order as the `SignatureAlgorithm` iota
	return [...]string{
		HashAlgorithmDocStringSHA2_256,
		HashAlgorithmDocStringSHA3_256,
	}[algo]
}

func newEnumType(identifier string, rawType Type) *BuiltinStructType {
	accountKeyType := &BuiltinStructType{
		Identifier:           identifier,
		EnumRawType:          rawType,
		IsInvalid:            false,
		IsResource:           false,
		Storable:             true,
		Equatable:            true,
		ExternallyReturnable: true,
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
	return accountKeyType
}
