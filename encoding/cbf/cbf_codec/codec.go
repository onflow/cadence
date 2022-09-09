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
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding"
	"github.com/onflow/cadence/runtime/common"
)

type CadenceBinaryFormatCodec struct{}

func (v CadenceBinaryFormatCodec) Encode(value cadence.Value) ([]byte, error) {
	return EncodeValue(value)
}

func (v CadenceBinaryFormatCodec) MustEncode(value cadence.Value) []byte {
	return MustEncode(value)
}

func (v CadenceBinaryFormatCodec) Decode(gauge common.MemoryGauge, bytes []byte) (cadence.Value, error) {
	return DecodeValue(gauge, bytes)
}

func (v CadenceBinaryFormatCodec) MustDecode(gauge common.MemoryGauge, bytes []byte) cadence.Value {
	return MustDecode(gauge, bytes)
}

var _ encoding.Codec = CadenceBinaryFormatCodec{}
