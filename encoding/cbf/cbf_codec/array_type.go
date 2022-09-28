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

func (e *Encoder) EncodeVariableArrayType(t cadence.VariableSizedArrayType) (err error) {
	return e.EncodeType(t.Element())
}

func (d *Decoder) DecodeVariableArrayType() (t cadence.VariableSizedArrayType, err error) {
	elementType, err := d.DecodeType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredVariableSizedArrayType(d.memoryGauge, elementType)
	return
}

func (e *Encoder) EncodeConstantArrayType(t cadence.ConstantSizedArrayType) (err error) {
	err = e.EncodeType(t.Element())
	if err != nil {
		return
	}

	return common_codec.EncodeLength(&e.w, int(t.Size))
}

func (d *Decoder) DecodeConstantArrayType() (t cadence.ConstantSizedArrayType, err error) {
	elementType, err := d.DecodeType()
	if err != nil {
		return
	}

	size, err := common_codec.DecodeLength(&d.r, d.maxSize())
	if err != nil {
		return
	}

	t = cadence.NewMeteredConstantSizedArrayType(d.memoryGauge, uint(size), elementType)
	return
}
