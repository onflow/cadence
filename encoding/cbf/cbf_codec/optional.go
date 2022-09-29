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
	"github.com/onflow/cadence/encoding/cbf/common_codec"
)

func (e *Encoder) EncodeOptional(value cadence.Optional) (err error) {
	isNil := value.Value == nil
	err = common_codec.EncodeBool(&e.w, isNil)
	if isNil || err != nil {
		return
	}

	return e.EncodeValue(value.Value)
}

func (d *Decoder) DecodeOptional() (value cadence.Optional, err error) {
	isNil, err := d.DecodeBool()
	if isNil || err != nil {
		return
	}

	innerValue, err := d.DecodeValue()
	value = cadence.NewMeteredOptional(d.memoryGauge, innerValue)
	return
}
