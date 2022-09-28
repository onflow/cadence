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
	"github.com/onflow/cadence/runtime/common"
)

func (d *Decoder) DecodeString() (s cadence.String, err error) {
	length, err := common_codec.DecodeStringHeader(&d.r, d.maxSize())
	if err != nil {
		return
	}

	s, err2 := cadence.NewMeteredString(
		d.memoryGauge,
		common.NewCadenceStringMemoryUsage(length),
		func() (str string) {
			// this inner error is set to the DecodeString method's `err` return value, so it's still captured
			str, err = common_codec.DecodeStringElements(&d.r, length)
			return
		},
	)

	// outer error is captured too
	if err2 != nil {
		return
	}
	return
}
