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

func (e *Encoder) EncodeInterfaceType(t cadence.InterfaceType) (err error) {
	err = common_codec.EncodeLocation(&e.w, t.InterfaceTypeLocation())
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, t.InterfaceTypeQualifiedIdentifier())
	if err != nil {
		return
	}

	err = common_codec.EncodeArray(&e.w, t.InterfaceFields(), func(field cadence.Field) (err error) {
		return e.encodeField(field)
	})

	return common_codec.EncodeArray(&e.w, t.InterfaceInitializers(), func(parameters []cadence.Parameter) (err error) {
		return common_codec.EncodeArray(&e.w, parameters, func(parameter cadence.Parameter) (err error) {
			return e.encodeParameter(parameter)
		})
	})
}

func (d *Decoder) DecodeStructInterfaceType() (t *cadence.StructInterfaceType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeInterfaceType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredStructInterfaceType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeResourceInterfaceType() (t *cadence.ResourceInterfaceType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeInterfaceType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredResourceInterfaceType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeContractInterfaceType() (t *cadence.ContractInterfaceType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeInterfaceType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredContractInterfaceType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) decodeInterfaceType() (
	location common.Location,
	qualifiedIdentifier string,
	fields []cadence.Field,
	initializers [][]cadence.Parameter,
	err error,
) {
	location, err = common_codec.DecodeLocation(&d.r, d.memoryGauge)
	if err != nil {
		return
	}

	qualifiedIdentifier, err = common_codec.DecodeString(&d.r)
	if err != nil {
		return
	}

	fields, err = common_codec.DecodeArray(&d.r, func() (field cadence.Field, err error) {
		return d.decodeField()
	})
	if err != nil {
		return
	}

	initializers, err = common_codec.DecodeArray(&d.r, func() ([]cadence.Parameter, error) {
		return common_codec.DecodeArray(&d.r, func() (cadence.Parameter, error) {
			return d.decodeParameter()
		})
	})

	return
}
