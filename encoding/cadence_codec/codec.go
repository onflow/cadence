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

package cadence_codec

import (
	"fmt"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding"
	customCodec "github.com/onflow/cadence/encoding/cbf/cbf_codec"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
)

type CadenceCodec struct {
	Encoder encoding.Codec
}

func (c CadenceCodec) Encode(value cadence.Value) ([]byte, error) {
	return c.Encoder.Encode(value)
}

func (c CadenceCodec) MustEncode(value cadence.Value) []byte {
	return c.Encoder.MustEncode(value)
}

func (c CadenceCodec) Decode(gauge common.MemoryGauge, bytes []byte) (cadence.Value, error) {
	codec, err := c.chooseCodec(bytes)
	if err != nil {
		return nil, err
	}
	return codec.Decode(gauge, bytes)
}

func (c CadenceCodec) MustDecode(gauge common.MemoryGauge, bytes []byte) cadence.Value {
	codec, err := c.chooseCodec(bytes)
	if err != nil {
		panic(err)
	}
	return codec.MustDecode(gauge, bytes)
}

func (c CadenceCodec) chooseCodec(bytes []byte) (codec encoding.Codec, err error) {
	if len(bytes) == 0 {
		err = fmt.Errorf("cannot decode empty bytes")
		return
	}

	if bytes[0] == '{' {
		codec = jsoncdc.JsonCodec{}
	} else {
		codec = customCodec.CadenceBinaryFormatCodec{}
	}
	return
}

var _ encoding.Codec = CadenceCodec{}
