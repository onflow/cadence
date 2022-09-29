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

func (e *Encoder) EncodeCompositeType(t cadence.CompositeType) (err error) {
	err = common_codec.EncodeLocation(&e.w, t.CompositeTypeLocation())
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, t.CompositeTypeQualifiedIdentifier())
	if err != nil {
		return
	}

	err = common_codec.EncodeArray(&e.w, t.CompositeFields(), func(field cadence.Field) (err error) {
		return e.encodeField(field)
	})

	return common_codec.EncodeArray(&e.w, t.CompositeInitializers(), func(parameters []cadence.Parameter) (err error) {
		return common_codec.EncodeArray(&e.w, parameters, e.encodeParameter)
	})
}

func (d *Decoder) DecodeContractType() (t *cadence.ContractType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeCompositeType()
	if err != nil {
		return
	}
	t = cadence.NewMeteredContractType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeStructType() (t *cadence.StructType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeCompositeType()
	if err != nil {
		return
	}
	t = cadence.NewMeteredStructType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (d *Decoder) DecodeResourceType() (t *cadence.ResourceType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeCompositeType()
	if err != nil {
		return
	}
	t = cadence.NewMeteredResourceType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers)
	return
}

func (e *Encoder) EncodeEnumType(t *cadence.EnumType) (err error) {
	err = e.EncodeCompositeType(t)
	if err != nil {
		return
	}

	return e.EncodeType(t.RawType)
}

func (d *Decoder) DecodeEnumType() (t *cadence.EnumType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeCompositeType()
	if err != nil {
		return
	}

	rawType, err := d.DecodeType()
	if err != nil {
		return
	}

	t = cadence.NewMeteredEnumType(d.memoryGauge, location, qualifiedIdentifier, rawType, fields, initializers)
	return
}

func (d *Decoder) DecodeEventType() (t *cadence.EventType, err error) {
	location, qualifiedIdentifier, fields, initializers, err := d.decodeCompositeType()
	if err != nil {
		return
	}
	// TODO verify that initializers is always at least length 1 for EventType
	t = cadence.NewMeteredEventType(d.memoryGauge, location, qualifiedIdentifier, fields, initializers[0])
	return
}

func (d *Decoder) decodeCompositeType() (
	location common.Location,
	qualifiedIdentifier string,
	fields []cadence.Field,
	initializers [][]cadence.Parameter,
	err error) {
	location, err = common_codec.DecodeLocation(&d.r, d.maxSize(), d.memoryGauge)
	if err != nil {
		return
	}

	qualifiedIdentifier, err = common_codec.DecodeString(&d.r, d.maxSize())
	if err != nil {
		return
	}

	fields, err = common_codec.DecodeArray(&d.r, d.maxSize(), d.decodeField)
	if err != nil {
		return
	}

	initializers, err = common_codec.DecodeArray(&d.r, d.maxSize(), func() ([]cadence.Parameter, error) {
		return common_codec.DecodeArray(&d.r, d.maxSize(), d.decodeParameter)
	})

	return
}

func (e *Encoder) encodeField(field cadence.Field) (err error) {
	err = common_codec.EncodeString(&e.w, field.Identifier)
	if err != nil {
		return
	}

	return e.EncodeType(field.Type)
}

func (d *Decoder) decodeField() (field cadence.Field, err error) {
	// TODO meter
	field.Identifier, err = common_codec.DecodeString(&d.r, d.maxSize())
	if err != nil {
		return
	}

	field.Type, err = d.DecodeType()
	return
}

func (e *Encoder) encodeParameter(parameter cadence.Parameter) (err error) {
	err = common_codec.EncodeString(&e.w, parameter.Label)
	if err != nil {
		return
	}

	err = common_codec.EncodeString(&e.w, parameter.Identifier)
	if err != nil {
		return
	}

	return e.EncodeType(parameter.Type)
}

func (d *Decoder) decodeParameter() (parameter cadence.Parameter, err error) {
	// TODO meter?
	parameter.Label, err = common_codec.DecodeString(&d.r, d.maxSize())
	if err != nil {
		return
	}

	parameter.Identifier, err = common_codec.DecodeString(&d.r, d.maxSize())
	if err != nil {
		return
	}

	parameter.Type, err = d.DecodeType()
	return
}
