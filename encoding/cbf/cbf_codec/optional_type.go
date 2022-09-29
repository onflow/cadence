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

func (e *Encoder) EncodeOptionalType(t cadence.OptionalType) (err error) {
	err = common_codec.EncodeBool(&e.w, t.Type == nil)
	if err != nil {
		return
	}

	return e.EncodeType(t.Type)
}

func (d *Decoder) DecodeOptionalType() (t cadence.OptionalType, err error) {
	isNil, err := common_codec.DecodeBool(&d.r)
	if isNil || err != nil {
		return
	}

	elementType, err := d.DecodeType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredOptionalType(d.memoryGauge, elementType)
	return
}
