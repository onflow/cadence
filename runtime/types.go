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

package runtime

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

const BlockHashLength = 32

type BlockHash [BlockHashLength]byte

type Block struct {
	Height    uint64
	View      uint64
	Hash      BlockHash
	Timestamp int64
}

type ResolvedLocation = sema.ResolvedLocation
type Identifier = ast.Identifier
type Location = common.Location

type SignatureAlgorithm = sema.SignatureAlgorithm

// NOTE: do *NOT* replace with iota or assign literal values,
// the values should be exactly the same as the ones declared in sema!

const (
	// Supported signing algorithms
	SignatureAlgorithmUnknown         = sema.SignatureAlgorithmUnknown
	SignatureAlgorithmECDSA_P256      = sema.SignatureAlgorithmECDSA_P256
	SignatureAlgorithmECDSA_secp256k1 = sema.SignatureAlgorithmECDSA_secp256k1
	SignatureAlgorithmBLS_BLS12_381   = sema.SignatureAlgorithmBLS_BLS12_381
)

type HashAlgorithm = sema.HashAlgorithm

// NOTE: do *NOT* replace with iota or assign literal values,
// the values should be exactly the same as the ones declared in sema!

const (
	// Supported hashing algorithms
	HashAlgorithmUnknown               = sema.HashAlgorithmUnknown
	HashAlgorithmSHA2_256              = sema.HashAlgorithmSHA2_256
	HashAlgorithmSHA2_384              = sema.HashAlgorithmSHA2_384
	HashAlgorithmSHA3_256              = sema.HashAlgorithmSHA3_256
	HashAlgorithmSHA3_384              = sema.HashAlgorithmSHA3_384
	HashAlgorithmKMAC128_BLS_BLS12_381 = sema.HashAlgorithmKMAC128_BLS_BLS12_381
	HashAlgorithmKECCAK_256            = sema.HashAlgorithmKECCAK_256
)

type AccountKey struct {
	KeyIndex  int
	PublicKey *PublicKey
	HashAlgo  HashAlgorithm
	Weight    int
	IsRevoked bool
}

type PublicKey struct {
	PublicKey []byte
	SignAlgo  SignatureAlgorithm
}
