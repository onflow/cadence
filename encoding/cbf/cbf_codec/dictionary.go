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

func (e *Encoder) EncodeDictionary(value cadence.Dictionary) (err error) {
	err = e.EncodeDictionaryType(value.DictionaryType)
	if err != nil {
		return
	}
	err = common_codec.EncodeLength(&e.w, len(value.Pairs))
	if err != nil {
		return
	}
	for _, kv := range value.Pairs {
		err = e.EncodeValue(kv.Key)
		if err != nil {
			return
		}
		err = e.EncodeValue(kv.Value)
		if err != nil {
			return
		}
	}
	return
}

func (d *Decoder) DecodeDictionary() (dict cadence.Dictionary, err error) {
	dictType, err := d.DecodeDictionaryType()
	if err != nil {
		return
	}

	size, err := common_codec.DecodeLength(&d.r, d.maxSize())
	if err != nil {
		return
	}

	dict, err = cadence.NewMeteredDictionary(d.memoryGauge, size, func() (pairs []cadence.KeyValuePair, err error) {
		pairs = make([]cadence.KeyValuePair, 0, size)
		var key, value cadence.Value
		for i := 0; i < size; i++ {
			key, err = d.DecodeValue()
			if err != nil {
				return
			}
			value, err = d.DecodeValue()
			if err != nil {
				return
			}
			pairs = append(pairs, cadence.NewMeteredKeyValuePair(d.memoryGauge, key, value))
		}
		return
	})

	dict = dict.WithType(dictType)

	return
}
