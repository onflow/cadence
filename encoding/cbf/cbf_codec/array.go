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
	"fmt"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/cbf/common_codec"
)

func (e *Encoder) EncodeArray(value cadence.Array) (err error) {
	switch v := value.ArrayType.(type) {
	case cadence.VariableSizedArrayType, nil: // unknown type still needs length
		err = common_codec.EncodeLength(&e.w, len(value.Values))
		if err != nil {
			return
		}
	case cadence.ConstantSizedArrayType:
		if len(value.Values) != int(v.Size) {
			return common_codec.CodecError(fmt.Sprintf("constant size array size=%d but has %d elements", v.Size, len(value.Values)))
		}
	}

	for _, element := range value.Values {
		err = e.EncodeValue(element)
		if err != nil {
			return err
		}
	}

	return
}

func (d *Decoder) DecodeUntypedArray() (array cadence.Array, err error) {
	size, err := common_codec.DecodeLength(&d.r)
	if err != nil {
		return
	}
	return d.decodeArray(nil, size)
}

func (d *Decoder) DecodeVariableArray(arrayType cadence.VariableSizedArrayType) (array cadence.Array, err error) {
	size, err := common_codec.DecodeLength(&d.r)
	if err != nil {
		return
	}
	return d.decodeArray(arrayType, size)
}

func (d *Decoder) DecodeConstantArray(arrayType cadence.ConstantSizedArrayType) (array cadence.Array, err error) {
	size := int(arrayType.Size)
	return d.decodeArray(arrayType, size)
}

func (d *Decoder) decodeArray(arrayType cadence.ArrayType, size int) (array cadence.Array, err error) {
	array, err = cadence.NewMeteredArray(d.memoryGauge, size, func() (elements []cadence.Value, err error) {
		elements = make([]cadence.Value, 0, size)
		for i := 0; i < size; i++ {
			var elementValue cadence.Value
			elementValue, err = d.DecodeValue()
			if err != nil {
				return
			}
			elements = append(elements, elementValue)
		}

		return elements, nil
	})

	array = array.WithType(arrayType)

	return
}
