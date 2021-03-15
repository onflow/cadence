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

const (
	// Supported signing algorithms
	SignatureAlgorithmECDSA_P256 SignatureAlgorithm = iota
	SignatureAlgorithmECDSA_Secp256k1
	SignatureAlgorithmBLS_BLS12381
)

type HashAlgorithm = sema.HashAlgorithm

const (
	// Supported hashing algorithms
	HashAlgorithmSHA2_256 HashAlgorithm = iota
	HashAlgorithmSHA2_384
	HashAlgorithmSHA3_256
	HashAlgorithmSHA3_384
	HashAlgorithmKMAC_128
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
