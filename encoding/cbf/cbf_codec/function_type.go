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

func (e *Encoder) EncodeFunctionType(t *cadence.FunctionType) (err error) {
	err = common_codec.EncodeString(&e.w, t.ID())
	if err != nil {
		return
	}

	err = common_codec.EncodeArray(&e.w, t.Parameters, e.encodeParameter)
	if err != nil {
		return
	}

	return e.EncodeType(t.ReturnType)
}

func (d *Decoder) DecodeFunctionType() (t *cadence.FunctionType, err error) {
	id, err := common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	parameters, err := common_codec.DecodeArray(&d.r, d.decodeParameter)
	if err != nil {
		return
	}

	returnType, err := d.DecodeType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredFunctionType(d.memoryGauge, id, parameters, returnType)
	return
}
