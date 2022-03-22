/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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
	"math/big"
	"unsafe"
)

type MemoryUsage struct {
	Kind   MemoryKind
	Amount uint64
}

type MemoryGauge interface {
	MeterMemory(usage MemoryUsage) error
}

func NewConstantMemoryUsage(kind MemoryKind) MemoryUsage {
	return MemoryUsage{
		Kind:   kind,
		Amount: 1,
	}
}

func NewStringMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindString,
		Amount: uint64(length) + 1, // +1 to account for empty strings
	}
}

func NewRawStringMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindRawString,
		Amount: uint64(length) + 1, // +1 to account for empty strings
	}
}

func NewBigIntMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindBigInt,
		Amount: uint64(bytes),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const bigIntWordSize = int(unsafe.Sizeof(big.Word(0)))

func BigIntByteLength(v *big.Int) int {
	// NOTE: big.Int.Bits() actually returns bytes:
	// []big.Word, where big.Word = uint
	return len(v.Bits()) * bigIntWordSize
}

func NewPlusBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		) + bigIntWordSize,
	)
}

func NewMinusBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewMulBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		BigIntByteLength(a) +
			BigIntByteLength(b),
	)
}

func NewTypeMemoryUsage(staticTypeAsString string) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindTypeValue,
		Amount: uint64(len(staticTypeAsString)),
	}
}

func NewCharacterMemoryUsage(length int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindCharacter,
		Amount: uint64(length),
	}
}

// UseConstantMemory uses a pre-determined amount of memory
//
func UseConstantMemory(memoryGauge MemoryGauge, kind MemoryKind) {
	UseMemory(memoryGauge, MemoryUsage{
		Kind:   kind,
		Amount: 1,
	})
}

func NewModBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewDivBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewBitwiseOrBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewBitwiseXorBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewBitwiseAndBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewBitwiseLeftShiftBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		BigIntByteLength(a) +
			BigIntByteLength(b),
	)
}

func NewBitwiseRightShiftBigIntMemoryUsage(a, b *big.Int) MemoryUsage {
	return NewBigIntMemoryUsage(
		// TODO: https://github.com/dapperlabs/cadence-private-issues/issues/32
		max(
			BigIntByteLength(a),
			BigIntByteLength(b),
		),
	)
}

func NewNumberMemoryUsage(bytes int) MemoryUsage {
	return MemoryUsage{
		Kind:   MemoryKindNumber,
		Amount: uint64(bytes),
	}
}

func UseMemory(gauge MemoryGauge, usage MemoryUsage) {
	if gauge == nil {
		return
	}

	err := gauge.MeterMemory(usage)
	if err != nil {
		panic(err)
	}
}
