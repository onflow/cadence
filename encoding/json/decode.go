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

package json

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// A Decoder decodes JSON-encoded representations of Cadence values.
type Decoder struct {
	dec   *json.Decoder
	gauge common.MemoryGauge
}

// Decode returns a Cadence value decoded from its JSON-encoded representation.
//
// This function returns an error if the bytes represent JSON that is malformed
// or does not conform to the JSON Cadence specification.
func Decode(gauge common.MemoryGauge, b []byte) (cadence.Value, error) {
	r := bytes.NewReader(b)
	dec := NewDecoder(gauge, r)

	v, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	return v, nil
}

// NewDecoder initializes a Decoder that will decode JSON-encoded bytes from the
// given io.Reader.
func NewDecoder(gauge common.MemoryGauge, r io.Reader) *Decoder {
	return &Decoder{
		dec:   json.NewDecoder(r),
		gauge: gauge,
	}
}

// Decode reads JSON-encoded bytes from the io.Reader and decodes them to a
// Cadence value.
//
// This function returns an error if the bytes represent JSON that is malformed
// or does not conform to the JSON Cadence specification.
func (d *Decoder) Decode() (value cadence.Value, err error) {
	jsonMap := make(map[string]interface{})

	err = d.dec.Decode(&jsonMap)
	if err != nil {
		return nil, fmt.Errorf("json-cdc: failed to decode valid JSON structure: %w", err)
	}

	// capture panics that occur during decoding
	defer func() {
		if r := recover(); r != nil {
			panicErr, isError := r.(error)
			if !isError {
				panic(r)
			}

			err = fmt.Errorf("failed to decode value: %w", panicErr)
		}
	}()

	value = d.decodeJSON(jsonMap)
	return value, nil
}

const (
	typeKey         = "type"
	kindKey         = "kind"
	valueKey        = "value"
	keyKey          = "key"
	nameKey         = "name"
	fieldsKey       = "fields"
	initializersKey = "initializers"
	idKey           = "id"
	targetPathKey   = "targetPath"
	borrowTypeKey   = "borrowType"
	domainKey       = "domain"
	identifierKey   = "identifier"
	staticTypeKey   = "staticType"
	addressKey      = "address"
	pathKey         = "path"
	authorizedKey   = "authorized"
	sizeKey         = "size"
	typeIDKey       = "typeID"
	restrictionsKey = "restrictions"
	labelKey        = "label"
	parametersKey   = "parameters"
	returnKey       = "return"
)

var ErrInvalidJSONCadence = errors.New("invalid JSON Cadence structure")

func (d *Decoder) decodeJSON(v interface{}) cadence.Value {
	obj := toObject(v)

	typeStr := obj.GetString(typeKey)

	// void is a special case, does not have "value" field
	if typeStr == voidTypeStr {
		return d.decodeVoid(obj)
	}

	// object should only contain two keys: "type", "value"
	if len(obj) != 2 {
		panic(ErrInvalidJSONCadence)
	}

	valueJSON := obj.Get(valueKey)

	switch typeStr {
	case optionalTypeStr:
		return d.decodeOptional(valueJSON)
	case boolTypeStr:
		return d.decodeBool(valueJSON)
	case characterTypeStr:
		return d.decodeCharacter(valueJSON)
	case stringTypeStr:
		return d.decodeString(valueJSON)
	case addressTypeStr:
		return d.decodeAddress(valueJSON)
	case intTypeStr:
		return d.decodeInt(valueJSON)
	case int8TypeStr:
		return d.decodeInt8(valueJSON)
	case int16TypeStr:
		return d.decodeInt16(valueJSON)
	case int32TypeStr:
		return d.decodeInt32(valueJSON)
	case int64TypeStr:
		return d.decodeInt64(valueJSON)
	case int128TypeStr:
		return d.decodeInt128(valueJSON)
	case int256TypeStr:
		return d.decodeInt256(valueJSON)
	case uintTypeStr:
		return d.decodeUInt(valueJSON)
	case uint8TypeStr:
		return d.decodeUInt8(valueJSON)
	case uint16TypeStr:
		return d.decodeUInt16(valueJSON)
	case uint32TypeStr:
		return d.decodeUInt32(valueJSON)
	case uint64TypeStr:
		return d.decodeUInt64(valueJSON)
	case uint128TypeStr:
		return d.decodeUInt128(valueJSON)
	case uint256TypeStr:
		return d.decodeUInt256(valueJSON)
	case word8TypeStr:
		return d.decodeWord8(valueJSON)
	case word16TypeStr:
		return d.decodeWord16(valueJSON)
	case word32TypeStr:
		return d.decodeWord32(valueJSON)
	case word64TypeStr:
		return d.decodeWord64(valueJSON)
	case fix64TypeStr:
		return d.decodeFix64(valueJSON)
	case ufix64TypeStr:
		return d.decodeUFix64(valueJSON)
	case arrayTypeStr:
		return d.decodeArray(valueJSON)
	case dictionaryTypeStr:
		return d.decodeDictionary(valueJSON)
	case resourceTypeStr:
		return d.decodeResource(valueJSON)
	case structTypeStr:
		return d.decodeStruct(valueJSON)
	case eventTypeStr:
		return d.decodeEvent(valueJSON)
	case contractTypeStr:
		return d.decodeContract(valueJSON)
	case linkTypeStr:
		return d.decodeLink(valueJSON)
	case pathTypeStr:
		return d.decodePath(valueJSON)
	case typeTypeStr:
		return d.decodeTypeValue(valueJSON)
	case capabilityTypeStr:
		return d.decodeCapability(valueJSON)
	case enumTypeStr:
		return d.decodeEnum(valueJSON)
	}

	panic(ErrInvalidJSONCadence)
}

func (d *Decoder) decodeVoid(m map[string]interface{}) cadence.Void {
	// object should not contain fields other than "type"
	if len(m) != 1 {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewVoid(d.gauge)
}

func (d *Decoder) decodeOptional(valueJSON interface{}) cadence.Optional {
	if valueJSON == nil {
		return cadence.NewOptional(d.gauge, nil)
	}

	return cadence.NewOptional(d.gauge, d.decodeJSON(valueJSON))
}

func (d *Decoder) decodeBool(valueJSON interface{}) cadence.Bool {
	return cadence.NewBool(d.gauge, toBool(valueJSON))
}

func (d *Decoder) decodeCharacter(valueJSON interface{}) cadence.Character {
	asString := toString(valueJSON)
	char, err := cadence.NewCharacter(
		d.gauge,
		common.NewCadenceCharacterMemoryUsage(len(asString)),
		func() string {
			return asString
		})
	if err != nil {
		panic(err)
	}
	return char
}

func (d *Decoder) decodeString(valueJSON interface{}) cadence.String {
	asString := toString(valueJSON)
	str, err := cadence.NewString(
		d.gauge,
		common.NewCadenceStringMemoryUsage(len(asString)),
		func() string {
			return asString
		},
	)
	if err != nil {
		panic(err)
	}
	return str
}

func (d *Decoder) decodeAddress(valueJSON interface{}) cadence.Address {
	v := toString(valueJSON)

	// must include 0x prefix
	if v[:2] != "0x" {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	b, err := hex.DecodeString(v[2:])
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.BytesToUnmeteredAddress(b)
}

func (d *Decoder) decodeBigInt(valueJSON interface{}) *big.Int {
	v := toString(valueJSON)

	i := new(big.Int)
	i, ok := i.SetString(v, 10)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return i
}

func (d *Decoder) decodeInt(valueJSON interface{}) cadence.Int {
	bigInt := d.decodeBigInt(valueJSON)
	return cadence.NewIntFromBig(
		d.gauge,
		common.NewCadenceIntMemoryUsage(
			common.BigIntByteLength(bigInt),
		),
		func() *big.Int {
			return bigInt
		},
	)
}

func (d *Decoder) decodeInt8(valueJSON interface{}) cadence.Int8 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt8(d.gauge, int8(i))
}

func (d *Decoder) decodeInt16(valueJSON interface{}) cadence.Int16 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt16(d.gauge, int16(i))
}

func (d *Decoder) decodeInt32(valueJSON interface{}) cadence.Int32 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt32(d.gauge, int32(i))
}

func (d *Decoder) decodeInt64(valueJSON interface{}) cadence.Int64 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt64(d.gauge, i)
}

func (d *Decoder) decodeInt128(valueJSON interface{}) cadence.Int128 {
	bigInt := d.decodeBigInt(valueJSON)

	value, err := cadence.NewInt128FromBig(
		d.gauge,
		func() *big.Int {
			return bigInt
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeInt256(valueJSON interface{}) cadence.Int256 {
	bigInt := d.decodeBigInt(valueJSON)

	value, err := cadence.NewInt256FromBig(
		d.gauge,
		func() *big.Int {
			return bigInt
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeUInt(valueJSON interface{}) cadence.UInt {
	bigInt := d.decodeBigInt(valueJSON)
	value, err := cadence.NewUIntFromBig(
		d.gauge,
		common.NewCadenceIntMemoryUsage(
			common.BigIntByteLength(bigInt),
		),
		func() *big.Int {
			return bigInt
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeUInt8(valueJSON interface{}) cadence.UInt8 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt8(d.gauge, uint8(i))
}

func (d *Decoder) decodeUInt16(valueJSON interface{}) cadence.UInt16 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt16(d.gauge, uint16(i))
}

func (d *Decoder) decodeUInt32(valueJSON interface{}) cadence.UInt32 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt32(d.gauge, uint32(i))
}

func (d *Decoder) decodeUInt64(valueJSON interface{}) cadence.UInt64 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt64(d.gauge, i)
}

func (d *Decoder) decodeUInt128(valueJSON interface{}) cadence.UInt128 {
	bigInt := d.decodeBigInt(valueJSON)
	value, err := cadence.NewUInt128FromBig(
		d.gauge,
		func() *big.Int {
			return bigInt
		},
	)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeUInt256(valueJSON interface{}) cadence.UInt256 {
	bigInt := d.decodeBigInt(valueJSON)
	value, err := cadence.NewUInt256FromBig(
		d.gauge,
		func() *big.Int {
			return bigInt
		},
	)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeWord8(valueJSON interface{}) cadence.Word8 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord8(d.gauge, uint8(i))
}

func (d *Decoder) decodeWord16(valueJSON interface{}) cadence.Word16 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord16(d.gauge, uint16(i))
}

func (d *Decoder) decodeWord32(valueJSON interface{}) cadence.Word32 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord32(d.gauge, uint32(i))
}

func (d *Decoder) decodeWord64(valueJSON interface{}) cadence.Word64 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord64(d.gauge, i)
}

func (d *Decoder) decodeFix64(valueJSON interface{}) cadence.Fix64 {
	v, err := cadence.NewFix64(d.gauge, func() (int64, error) {
		return cadence.ParseFix64(toString(valueJSON))
	})
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return v
}

func (d *Decoder) decodeUFix64(valueJSON interface{}) cadence.UFix64 {
	v, err := cadence.NewUFix64(d.gauge, func() (uint64, error) {
		return cadence.ParseUFix64(toString(valueJSON))
	})
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return v
}

func (d *Decoder) decodeArray(valueJSON interface{}) cadence.Array {
	v := toSlice(valueJSON)

	value, err := cadence.NewArray(
		d.gauge,
		len(v),
		func() ([]cadence.Value, error) {
			values := make([]cadence.Value, len(v))
			for i, val := range v {
				values[i] = d.decodeJSON(val)
			}
			return values, nil
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeDictionary(valueJSON interface{}) cadence.Dictionary {
	v := toSlice(valueJSON)

	value, err := cadence.NewDictionary(
		d.gauge,
		len(v),
		func() ([]cadence.KeyValuePair, error) {
			pairs := make([]cadence.KeyValuePair, len(v))

			for i, val := range v {
				pairs[i] = d.decodeKeyValuePair(val)
			}

			return pairs, nil
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return value
}

func (d *Decoder) decodeKeyValuePair(valueJSON interface{}) cadence.KeyValuePair {
	obj := toObject(valueJSON)

	key := obj.GetValue(d, keyKey)
	value := obj.GetValue(d, valueKey)

	return cadence.KeyValuePair{
		Key:   key,
		Value: value,
	}
}

type composite struct {
	location            common.Location
	qualifiedIdentifier string
	fieldValues         []cadence.Value
	fieldTypes          []cadence.Field
}

func (d *Decoder) decodeComposite(valueJSON interface{}) composite {
	obj := toObject(valueJSON)

	typeID := obj.GetString(idKey)
	location, qualifiedIdentifier, err := common.DecodeTypeID(d.gauge, typeID)

	if err != nil ||
		location == nil && sema.NativeCompositeTypes[typeID] == nil {

		// If the location is nil, and there is no native composite type with this ID, then its an invalid type.
		// Note: This is moved out from the common.DecodeTypeID() to avoid the circular dependency.
		panic(fmt.Errorf("%s. invalid type ID: `%s`", ErrInvalidJSONCadence, typeID))
	}

	fields := obj.GetSlice(fieldsKey)

	fieldValues := make([]cadence.Value, len(fields))
	fieldTypes := make([]cadence.Field, len(fields))

	for i, field := range fields {
		value, fieldType := d.decodeCompositeField(field)

		fieldValues[i] = value
		fieldTypes[i] = fieldType
	}

	return composite{
		location:            location,
		qualifiedIdentifier: qualifiedIdentifier,
		fieldValues:         fieldValues,
		fieldTypes:          fieldTypes,
	}
}

func (d *Decoder) decodeCompositeField(valueJSON interface{}) (cadence.Value, cadence.Field) {
	obj := toObject(valueJSON)

	name := obj.GetString(nameKey)
	value := obj.GetValue(d, valueKey)

	field := cadence.Field{
		Identifier: name,
		Type:       value.Type(),
	}

	return value, field
}

func (d *Decoder) decodeStruct(valueJSON interface{}) cadence.Struct {
	comp := d.decodeComposite(valueJSON)

	structure, err := cadence.NewStruct(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	return structure.WithType(&cadence.StructType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func (d *Decoder) decodeResource(valueJSON interface{}) cadence.Resource {
	comp := d.decodeComposite(valueJSON)

	resource, err := cadence.NewResource(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}
	return resource.WithType(&cadence.ResourceType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func (d *Decoder) decodeEvent(valueJSON interface{}) cadence.Event {
	comp := d.decodeComposite(valueJSON)

	event, err := cadence.NewEvent(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	return event.WithType(&cadence.EventType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func (d *Decoder) decodeContract(valueJSON interface{}) cadence.Contract {
	comp := d.decodeComposite(valueJSON)

	contract, err := cadence.NewContract(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	return contract.WithType(&cadence.ContractType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func (d *Decoder) decodeEnum(valueJSON interface{}) cadence.Enum {
	comp := d.decodeComposite(valueJSON)

	enum, err := cadence.NewEnum(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	return enum.WithType(&cadence.EnumType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func (d *Decoder) decodeLink(valueJSON interface{}) cadence.Link {
	obj := toObject(valueJSON)

	targetPath, ok := d.decodeJSON(obj.Get(targetPathKey)).(cadence.Path)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return cadence.NewLink(
		d.gauge,
		targetPath,
		obj.GetString(borrowTypeKey),
	)
}

func (d *Decoder) decodePath(valueJSON interface{}) cadence.Path {
	obj := toObject(valueJSON)

	return cadence.NewPath(
		d.gauge,
		obj.GetString(domainKey),
		obj.GetString(identifierKey),
	)
}

func (d *Decoder) decodeParamType(valueJSON interface{}) cadence.Parameter {
	obj := toObject(valueJSON)
	return cadence.Parameter{
		Label:      toString(obj.Get(labelKey)),
		Identifier: toString(obj.Get(idKey)),
		Type:       d.decodeType(obj.Get(typeKey)),
	}
}

func (d *Decoder) decodeParamTypes(params []interface{}) []cadence.Parameter {
	parameters := make([]cadence.Parameter, 0, len(params))

	for _, param := range params {
		parameters = append(parameters, d.decodeParamType(param))
	}

	return parameters
}

func (d *Decoder) decodeFieldTypes(fs []interface{}) []cadence.Field {
	fields := make([]cadence.Field, 0, len(fs))

	for _, field := range fs {
		fields = append(fields, d.decodeFieldType(field))
	}

	return fields
}

func (d *Decoder) decodeFieldType(valueJSON interface{}) cadence.Field {
	obj := toObject(valueJSON)
	return cadence.Field{
		Identifier: toString(obj.Get(idKey)),
		Type:       d.decodeType(obj.Get(typeKey)),
	}
}

func (d *Decoder) decodeFunctionType(returnValue, parametersValue, id interface{}) cadence.Type {
	parameters := d.decodeParamTypes(toSlice(parametersValue))
	returnType := d.decodeType(returnValue)

	return cadence.FunctionType{
		Parameters: parameters,
		ReturnType: returnType,
	}.WithID(toString(id))
}

func (d *Decoder) decodeNominalType(obj jsonObject, kind, typeID string, fs, initializers []interface{}) cadence.Type {
	fields := d.decodeFieldTypes(fs)
	inits := make([][]cadence.Parameter, 0, len(initializers))

	for _, params := range initializers {
		inits = append(inits, d.decodeParamTypes(toSlice(params)))
	}

	location, id, err := common.DecodeTypeID(d.gauge, typeID)
	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	switch kind {
	case "Struct":
		return &cadence.StructType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "Resource":
		return &cadence.ResourceType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "Event":
		return &cadence.EventType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializer:         inits[0],
		}
	case "Contract":
		return &cadence.ContractType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "StructInterface":
		return &cadence.StructInterfaceType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "ResourceInterface":
		return &cadence.ResourceInterfaceType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "ContractInterface":
		return &cadence.ContractInterfaceType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "Enum":
		return &cadence.EnumType{
			Location:            location,
			QualifiedIdentifier: id,
			RawType:             d.decodeType(obj.Get(typeKey)),
			Fields:              fields,
			Initializers:        inits,
		}
	}

	panic(ErrInvalidJSONCadence)
}

func (d *Decoder) decodeRestrictedType(
	typeValue interface{},
	restrictionsValue []interface{},
	typeIDValue string,
) cadence.Type {
	typ := d.decodeType(typeValue)
	restrictions := make([]cadence.Type, 0, len(restrictionsValue))
	for _, restriction := range restrictionsValue {
		restrictions = append(restrictions, d.decodeType(restriction))
	}

	return cadence.RestrictedType{
		Type:         typ,
		Restrictions: restrictions,
	}.WithID(typeIDValue)
}

func (d *Decoder) decodeType(valueJSON interface{}) cadence.Type {
	if valueJSON == "" {
		return nil
	}
	obj := toObject(valueJSON)
	kindValue := toString(obj.Get(kindKey))

	switch kindValue {
	case "Function":
		returnValue := obj.Get(returnKey)
		parametersValue := obj.Get(parametersKey)
		idValue := obj.Get(typeIDKey)
		return d.decodeFunctionType(returnValue, parametersValue, idValue)
	case "Restriction":
		restrictionsValue := obj.Get(restrictionsKey)
		typeIDValue := toString(obj.Get(typeIDKey))
		typeValue := obj.Get(typeKey)
		return d.decodeRestrictedType(typeValue, toSlice(restrictionsValue), typeIDValue)
	case "Optional":
		return cadence.OptionalType{
			Type: d.decodeType(obj.Get(typeKey)),
		}
	case "VariableSizedArray":
		return cadence.VariableSizedArrayType{
			ElementType: d.decodeType(obj.Get(typeKey)),
		}
	case "Capability":
		return cadence.CapabilityType{
			BorrowType: d.decodeType(obj.Get(typeKey)),
		}
	case "Dictionary":
		return cadence.DictionaryType{
			KeyType:     d.decodeType(obj.Get(keyKey)),
			ElementType: d.decodeType(obj.Get(valueKey)),
		}
	case "ConstantSizedArray":
		size := toUInt(obj.Get(sizeKey))
		return cadence.ConstantSizedArrayType{
			ElementType: d.decodeType(obj.Get(typeKey)),
			Size:        size,
		}
	case "Reference":
		auth := toBool(obj.Get(authorizedKey))
		return cadence.ReferenceType{
			Type:       d.decodeType(obj.Get(typeKey)),
			Authorized: auth,
		}
	case "Any":
		return cadence.AnyType{}
	case "AnyStruct":
		return cadence.AnyStructType{}
	case "AnyResource":
		return cadence.AnyResourceType{}
	case "Type":
		return cadence.MetaType{}
	case "Void":
		return cadence.VoidType{}
	case "Never":
		return cadence.NeverType{}
	case "Bool":
		return cadence.BoolType{}
	case "String":
		return cadence.StringType{}
	case "Character":
		return cadence.CharacterType{}
	case "Bytes":
		return cadence.BytesType{}
	case "Address":
		return cadence.AddressType{}
	case "Number":
		return cadence.NumberType{}
	case "SignedNumber":
		return cadence.SignedNumberType{}
	case "Integer":
		return cadence.IntegerType{}
	case "SignedInteger":
		return cadence.SignedIntegerType{}
	case "FixedPoint":
		return cadence.FixedPointType{}
	case "SignedFixedPoint":
		return cadence.SignedFixedPointType{}
	case "Int":
		return cadence.IntType{}
	case "Int8":
		return cadence.Int8Type{}
	case "Int16":
		return cadence.Int16Type{}
	case "Int32":
		return cadence.Int32Type{}
	case "Int64":
		return cadence.Int64Type{}
	case "Int128":
		return cadence.Int128Type{}
	case "Int256":
		return cadence.Int256Type{}
	case "UInt":
		return cadence.UIntType{}
	case "UInt8":
		return cadence.UInt8Type{}
	case "UInt16":
		return cadence.UInt16Type{}
	case "UInt32":
		return cadence.UInt32Type{}
	case "UInt64":
		return cadence.UInt64Type{}
	case "UInt128":
		return cadence.UInt128Type{}
	case "UInt256":
		return cadence.UInt256Type{}
	case "Word8":
		return cadence.Word8Type{}
	case "Word16":
		return cadence.Word16Type{}
	case "Word32":
		return cadence.Word32Type{}
	case "Word64":
		return cadence.Word64Type{}
	case "Fix64":
		return cadence.Fix64Type{}
	case "UFix64":
		return cadence.UFix64Type{}
	case "Path":
		return cadence.PathType{}
	case "CapabilityPath":
		return cadence.CapabilityPathType{}
	case "StoragePath":
		return cadence.StoragePathType{}
	case "PublicPath":
		return cadence.PublicPathType{}
	case "PrivatePath":
		return cadence.PrivatePathType{}
	case "AuthAccount":
		return cadence.AuthAccountType{}
	case "PublicAccount":
		return cadence.PublicAccountType{}
	case "AuthAccount.Keys":
		return cadence.AuthAccountKeysType{}
	case "PublicAccount.Keys":
		return cadence.PublicAccountKeysType{}
	case "AuthAccount.Contracts":
		return cadence.AuthAccountContractsType{}
	case "PublicAccount.Contracts":
		return cadence.PublicAccountContractsType{}
	case "DeployedContract":
		return cadence.DeployedContractType{}
	case "AccountKey":
		return cadence.AccountKeyType{}
	default:
		fieldsValue := obj.Get(fieldsKey)
		typeIDValue := toString(obj.Get(typeIDKey))
		initValue := obj.Get(initializersKey)
		return d.decodeNominalType(obj, kindValue, typeIDValue, toSlice(fieldsValue), toSlice(initValue))
	}
}

func (d *Decoder) decodeTypeValue(valueJSON interface{}) cadence.TypeValue {
	obj := toObject(valueJSON)

	return cadence.NewTypeValue(
		d.gauge,
		d.decodeType(obj.Get(staticTypeKey)),
	)
}

func (d *Decoder) decodeCapability(valueJSON interface{}) cadence.Capability {
	obj := toObject(valueJSON)

	path, ok := d.decodeJSON(obj.Get(pathKey)).(cadence.Path)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewCapability(
		d.gauge,
		path,
		d.decodeAddress(obj.Get(addressKey)),
		d.decodeType(obj.Get(borrowTypeKey)),
	)
}

// JSON types

type jsonObject map[string]interface{}

func (obj jsonObject) Get(key string) interface{} {
	v, hasKey := obj[key]
	if !hasKey {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func (obj jsonObject) GetBool(key string) bool {
	v := obj.Get(key)
	return toBool(v)
}

func (obj jsonObject) GetString(key string) string {
	v := obj.Get(key)
	return toString(v)
}

func (obj jsonObject) GetSlice(key string) []interface{} {
	v := obj.Get(key)
	return toSlice(v)
}

func (obj jsonObject) GetValue(d *Decoder, key string) cadence.Value {
	v := obj.Get(key)
	return d.decodeJSON(v)
}

// JSON conversion helpers

func toBool(valueJSON interface{}) bool {
	v, isBool := valueJSON.(bool)
	if !isBool {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toUInt(valueJSON interface{}) uint {
	v, isNum := valueJSON.(float64)
	if !isNum {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return uint(v)
}

func toString(valueJSON interface{}) string {
	v, isString := valueJSON.(string)
	if !isString {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toSlice(valueJSON interface{}) []interface{} {
	v, isSlice := valueJSON.([]interface{})
	if !isSlice {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toObject(valueJSON interface{}) jsonObject {
	v, isMap := valueJSON.(map[string]interface{})
	if !isMap {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}
