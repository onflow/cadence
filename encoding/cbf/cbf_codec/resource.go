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

func (e *Encoder) EncodeResource(value cadence.Resource) (err error) {
	err = e.EncodeCompositeType(value.ResourceType)
	if err != nil {
		return
	}

	return common_codec.EncodeArray(&e.w, value.Fields, e.EncodeValue)
}

func (d *Decoder) DecodeResource() (s cadence.Resource, err error) {
	resourceType, err := d.DecodeResourceType()
	if err != nil {
		return
	}

	isNil, length, err := common_codec.DecodeArrayHeader(&d.r, d.maxSize())
	if isNil || err != nil {
		return
	}

	s, err = cadence.NewMeteredResource(
		d.memoryGauge,
		length,
		func() ([]cadence.Value, error) {
			return common_codec.DecodeArrayElements(length, d.DecodeValue)
		},
	)
	if err != nil {
		return
	}

	s = s.WithType(resourceType)
	return
}
