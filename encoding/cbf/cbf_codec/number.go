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

package cbf_codec

import (
	"math/big"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/cbf/common_codec"
	"github.com/onflow/cadence/runtime/common"
)

func (d *Decoder) DecodeInt() (value cadence.Int, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	value = cadence.NewMeteredIntFromBig(
		d.memoryGauge,
		common.NewBigIntMemoryUsage(common.BigIntByteLength(i)),
		func() *big.Int {
			return i
		},
	)
	return
}

func (d *Decoder) DecodeInt8() (value cadence.Int8, err error) {
	i, err := common_codec.DecodeNumber[int8](&d.r)
	value = cadence.Int8(i)
	return
}

func (d *Decoder) DecodeInt16() (value cadence.Int16, err error) {
	i, err := common_codec.DecodeNumber[int16](&d.r)
	value = cadence.Int16(i)
	return
}

func (d *Decoder) DecodeInt32() (value cadence.Int32, err error) {
	i, err := common_codec.DecodeNumber[int32](&d.r)
	value = cadence.Int32(i)
	return
}

func (d *Decoder) DecodeInt64() (value cadence.Int64, err error) {
	i, err := common_codec.DecodeNumber[int64](&d.r)
	value = cadence.Int64(i)
	return
}

func (d *Decoder) DecodeInt128() (value cadence.Int128, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredInt128FromBig(
		d.memoryGauge,
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeInt256() (value cadence.Int256, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredInt256FromBig(
		d.memoryGauge,
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeUInt() (value cadence.UInt, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredUIntFromBig(
		d.memoryGauge,
		common.NewBigIntMemoryUsage(common.BigIntByteLength(i)),
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeUInt8() (value cadence.UInt8, err error) {
	i, err := common_codec.DecodeNumber[uint8](&d.r)
	value = cadence.UInt8(i)
	return
}

func (d *Decoder) DecodeUInt16() (value cadence.UInt16, err error) {
	i, err := common_codec.DecodeNumber[uint16](&d.r)
	value = cadence.UInt16(i)
	return
}

func (d *Decoder) DecodeUInt32() (value cadence.UInt32, err error) {
	i, err := common_codec.DecodeNumber[uint32](&d.r)
	value = cadence.UInt32(i)
	return
}

func (d *Decoder) DecodeUInt64() (value cadence.UInt64, err error) {
	i, err := common_codec.DecodeNumber[uint64](&d.r)
	value = cadence.UInt64(i)
	return
}

func (d *Decoder) DecodeUInt128() (value cadence.UInt128, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredUInt128FromBig(
		d.memoryGauge,
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeUInt256() (value cadence.UInt256, err error) {
	i, err := common_codec.DecodeBigInt(&d.r)
	if err != nil {
		return
	}

	return cadence.NewMeteredUInt256FromBig(
		d.memoryGauge,
		func() *big.Int {
			return i
		},
	)
}

func (d *Decoder) DecodeWord8() (value cadence.Word8, err error) {
	i, err := common_codec.DecodeNumber[uint8](&d.r)
	if err != nil {
		return
	}
	value = cadence.Word8(i)
	return
}

func (d *Decoder) DecodeWord16() (value cadence.Word16, err error) {
	i, err := common_codec.DecodeNumber[uint16](&d.r)
	if err != nil {
		return
	}
	value = cadence.Word16(i)
	return
}

func (d *Decoder) DecodeWord32() (value cadence.Word32, err error) {
	i, err := common_codec.DecodeNumber[uint32](&d.r)
	if err != nil {
		return
	}
	value = cadence.Word32(i)
	return
}

func (d *Decoder) DecodeWord64() (value cadence.Word64, err error) {
	i, err := common_codec.DecodeNumber[uint64](&d.r)
	if err != nil {
		return
	}
	value = cadence.Word64(i)
	return
}

func (d *Decoder) DecodeFix64() (value cadence.Fix64, err error) {
	i, err := common_codec.DecodeNumber[int64](&d.r)
	if err != nil {
		return
	}
	value = cadence.Fix64(i)
	return
}

func (d *Decoder) DecodeUFix64() (value cadence.UFix64, err error) {
	i, err := common_codec.DecodeNumber[uint64](&d.r)
	if err != nil {
		return
	}
	value = cadence.UFix64(i)
	return
}
