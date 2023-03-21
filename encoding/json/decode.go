/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"io"
	"math/big"
	"strconv"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// A Decoder decodes JSON-encoded representations of Cadence values.
type Decoder struct {
	dec   *json.Decoder
	gauge common.MemoryGauge
	// allowUnstructuredStaticTypes controls if the decoding
	// of a static type as a type ID (cadence.TypeID) is allowed
	allowUnstructuredStaticTypes bool
}

type Option func(*Decoder)

// WithAllowUnstructuredStaticTypes returns a new Decoder Option
// which enables or disables if the decoding of a static type
// as a type ID (cadence.TypeID) is allowed
func WithAllowUnstructuredStaticTypes(allow bool) Option {
	return func(decoder *Decoder) {
		decoder.allowUnstructuredStaticTypes = allow
	}
}

// Decode returns a Cadence value decoded from its JSON-encoded representation.
//
// This function returns an error if the bytes represent JSON that is malformed
// or does not conform to the JSON Cadence specification.
func Decode(gauge common.MemoryGauge, b []byte, options ...Option) (cadence.Value, error) {
	r := bytes.NewReader(b)
	dec := NewDecoder(gauge, r)

	for _, option := range options {
		option(dec)
	}

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
	jsonMap := make(map[string]any)

	err = d.dec.Decode(&jsonMap)
	if err != nil {
		return nil, errors.NewDefaultUserError("failed to decode JSON: %w", err)
	}

	// capture panics that occur during decoding
	defer func() {
		if r := recover(); r != nil {
			panicErr, isError := r.(error)
			if !isError {
				panic(r)
			}

			err = errors.NewDefaultUserError("failed to decode JSON-Cadence value: %w", panicErr)
		}
	}()

	value = d.decodeJSON(jsonMap)
	return value, nil
}

const (
	typeKey          = "type"
	kindKey          = "kind"
	valueKey         = "value"
	keyKey           = "key"
	nameKey          = "name"
	fieldsKey        = "fields"
	initializersKey  = "initializers"
	idKey            = "id"
	targetPathKey    = "targetPath"
	borrowTypeKey    = "borrowType"
	domainKey        = "domain"
	identifierKey    = "identifier"
	staticTypeKey    = "staticType"
	addressKey       = "address"
	pathKey          = "path"
	authorizationKey = "authorization"
	entitlementsKey  = "entitlements"
	sizeKey          = "size"
	typeIDKey        = "typeID"
	restrictionsKey  = "restrictions"
	labelKey         = "label"
	parametersKey    = "parameters"
	returnKey        = "return"
	purityKey        = "purity"
)

func (d *Decoder) decodeJSON(v any) cadence.Value {
	obj := toObject(v)

	typeStr := obj.GetString(typeKey)

	// void is a special case, does not have "value" field
	if typeStr == voidTypeStr {
		return d.decodeVoid(obj)
	}

	// object should only contain two keys: "type", "value"
	if len(obj) != 2 {
		panic(errors.NewDefaultUserError("expected JSON object with keys `%s` and `%s`", typeKey, valueKey))
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
	case pathTypeStr:
		return d.decodePath(valueJSON)
	case typeTypeStr:
		return d.decodeTypeValue(valueJSON)
	case capabilityTypeStr:
		return d.decodeCapability(valueJSON)
	case enumTypeStr:
		return d.decodeEnum(valueJSON)
	}

	panic(errors.NewDefaultUserError("invalid type: %s", typeStr))
}

func (d *Decoder) decodeVoid(m map[string]any) cadence.Void {
	// object should not contain fields other than "type"
	if len(m) != 1 {
		panic(errors.NewDefaultUserError("invalid additional fields in void value"))
	}

	return cadence.NewMeteredVoid(d.gauge)
}

func (d *Decoder) decodeOptional(valueJSON any) cadence.Optional {
	if valueJSON == nil {
		return cadence.NewMeteredOptional(d.gauge, nil)
	}

	return cadence.NewMeteredOptional(d.gauge, d.decodeJSON(valueJSON))
}

func (d *Decoder) decodeBool(valueJSON any) cadence.Bool {
	return cadence.NewMeteredBool(d.gauge, toBool(valueJSON))
}

func (d *Decoder) decodeCharacter(valueJSON any) cadence.Character {
	asString := toString(valueJSON)
	char, err := cadence.NewMeteredCharacter(
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

func (d *Decoder) decodeString(valueJSON any) cadence.String {
	asString := toString(valueJSON)
	str, err := cadence.NewMeteredString(
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

const addressPrefix = "0x"

func (d *Decoder) decodeAddress(valueJSON any) cadence.Address {
	v := toString(valueJSON)

	prefixLength := len(addressPrefix)
	if len(v) < prefixLength {
		panic(errors.NewDefaultUserError("missing address prefix: `%s`", addressPrefix))
	}

	// must include 0x prefix
	actualPrefix := v[:prefixLength]
	if actualPrefix != addressPrefix {
		panic(errors.NewDefaultUserError(
			"invalid address prefix: expected `%s`, got `%s`",
			addressPrefix,
			actualPrefix,
		))
	}

	b, err := hex.DecodeString(v[prefixLength:])
	if err != nil {
		panic(errors.NewDefaultUserError("invalid address: %w", err))
	}

	return cadence.BytesToMeteredAddress(d.gauge, b)
}

func (d *Decoder) decodeBigInt(valueJSON any) *big.Int {
	v := toString(valueJSON)

	i := new(big.Int)
	i, ok := i.SetString(v, 10)
	if !ok {
		return nil
	}

	return i
}

func (d *Decoder) decodeInt(valueJSON any) cadence.Int {
	bigInt := d.decodeBigInt(valueJSON)
	if bigInt == nil {
		// TODO: propagate toString error from decodeBigInt
		panic(errors.NewDefaultUserError("invalid Int: %s", valueJSON))
	}

	return cadence.NewMeteredIntFromBig(
		d.gauge,
		common.NewCadenceIntMemoryUsage(
			common.BigIntByteLength(bigInt),
		),
		func() *big.Int {
			return bigInt
		},
	)
}

func (d *Decoder) decodeInt8(valueJSON any) cadence.Int8 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 8)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Int8: %s", v))
	}

	return cadence.NewMeteredInt8(d.gauge, int8(i))
}

func (d *Decoder) decodeInt16(valueJSON any) cadence.Int16 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 16)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Int16: %s", v))
	}

	return cadence.NewMeteredInt16(d.gauge, int16(i))
}

func (d *Decoder) decodeInt32(valueJSON any) cadence.Int32 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Int32: %s", v))
	}

	return cadence.NewMeteredInt32(d.gauge, int32(i))
}

func (d *Decoder) decodeInt64(valueJSON any) cadence.Int64 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Int64: %s", v))
	}

	return cadence.NewMeteredInt64(d.gauge, i)
}

func (d *Decoder) decodeInt128(valueJSON any) cadence.Int128 {
	value, err := cadence.NewMeteredInt128FromBig(
		d.gauge,
		func() *big.Int {
			bigInt := d.decodeBigInt(valueJSON)
			if bigInt == nil {
				// TODO: propagate toString error from decodeBigInt
				panic(errors.NewDefaultUserError("invalid Int128: %s", valueJSON))
			}
			return bigInt
		},
	)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid Int128: %w", err))
	}
	return value
}

func (d *Decoder) decodeInt256(valueJSON any) cadence.Int256 {
	value, err := cadence.NewMeteredInt256FromBig(
		d.gauge,
		func() *big.Int {
			bigInt := d.decodeBigInt(valueJSON)
			if bigInt == nil {
				// TODO: propagate toString error from decodeBigInt
				panic(errors.NewDefaultUserError("invalid Int256: %s", valueJSON))
			}
			return bigInt
		},
	)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid Int256: %w", err))
	}
	return value
}

func (d *Decoder) decodeUInt(valueJSON any) cadence.UInt {
	bigInt := d.decodeBigInt(valueJSON)
	if bigInt == nil {
		// TODO: propagate toString error from decodeBigInt
		panic(errors.NewDefaultUserError("invalid UInt: %s", valueJSON))
	}
	value, err := cadence.NewMeteredUIntFromBig(
		d.gauge,
		common.NewCadenceIntMemoryUsage(
			common.BigIntByteLength(bigInt),
		),
		func() *big.Int {
			return bigInt
		},
	)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid UInt: %w", err))
	}
	return value
}

func (d *Decoder) decodeUInt8(valueJSON any) cadence.UInt8 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid UInt8: %w", err))
	}

	return cadence.NewMeteredUInt8(d.gauge, uint8(i))
}

func (d *Decoder) decodeUInt16(valueJSON any) cadence.UInt16 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid UInt16: %w", err))
	}

	return cadence.NewMeteredUInt16(d.gauge, uint16(i))
}

func (d *Decoder) decodeUInt32(valueJSON any) cadence.UInt32 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid UInt32: %w", err))
	}

	return cadence.NewMeteredUInt32(d.gauge, uint32(i))
}

func (d *Decoder) decodeUInt64(valueJSON any) cadence.UInt64 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid UInt64: %w", err))
	}

	return cadence.NewMeteredUInt64(d.gauge, i)
}

func (d *Decoder) decodeUInt128(valueJSON any) cadence.UInt128 {
	value, err := cadence.NewMeteredUInt128FromBig(
		d.gauge,
		func() *big.Int {
			bigInt := d.decodeBigInt(valueJSON)
			if bigInt == nil {
				// TODO: propagate toString error from decodeBigInt
				panic(errors.NewDefaultUserError("invalid UInt128: %s", valueJSON))
			}
			return bigInt
		},
	)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid UInt128: %w", err))
	}
	return value
}

func (d *Decoder) decodeUInt256(valueJSON any) cadence.UInt256 {
	value, err := cadence.NewMeteredUInt256FromBig(
		d.gauge,
		func() *big.Int {
			bigInt := d.decodeBigInt(valueJSON)
			if bigInt == nil {
				panic(errors.NewDefaultUserError("invalid UInt256: %s", valueJSON))
			}
			return bigInt
		},
	)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid UInt256: %w", err))
	}
	return value
}

func (d *Decoder) decodeWord8(valueJSON any) cadence.Word8 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Word8: %w", err))
	}

	return cadence.NewMeteredWord8(d.gauge, uint8(i))
}

func (d *Decoder) decodeWord16(valueJSON any) cadence.Word16 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Word16: %w", err))
	}

	return cadence.NewMeteredWord16(d.gauge, uint16(i))
}

func (d *Decoder) decodeWord32(valueJSON any) cadence.Word32 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Word32: %w", err))
	}

	return cadence.NewMeteredWord32(d.gauge, uint32(i))
}

func (d *Decoder) decodeWord64(valueJSON any) cadence.Word64 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Word64: %w", err))
	}

	return cadence.NewMeteredWord64(d.gauge, i)
}

func (d *Decoder) decodeFix64(valueJSON any) cadence.Fix64 {
	v, err := cadence.NewMeteredFix64(d.gauge, func() (string, error) {
		return toString(valueJSON), nil
	})
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Fix64: %w", err))
	}
	return v
}

func (d *Decoder) decodeUFix64(valueJSON any) cadence.UFix64 {
	v, err := cadence.NewMeteredUFix64(d.gauge, func() (string, error) {
		return toString(valueJSON), nil
	})
	if err != nil {
		panic(errors.NewDefaultUserError("invalid UFix64: %w", err))
	}
	return v
}

func (d *Decoder) decodeArray(valueJSON any) cadence.Array {
	v := toSlice(valueJSON)

	value, err := cadence.NewMeteredArray(
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
		panic(errors.NewDefaultUserError("invalid array: %w", err))
	}
	return value
}

func (d *Decoder) decodeDictionary(valueJSON any) cadence.Dictionary {
	v := toSlice(valueJSON)

	value, err := cadence.NewMeteredDictionary(
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
		panic(errors.NewDefaultUserError("invalid dictionary: %w", err))
	}

	return value
}

func (d *Decoder) decodeKeyValuePair(valueJSON any) cadence.KeyValuePair {
	obj := toObject(valueJSON)

	key := obj.GetValue(d, keyKey)
	value := obj.GetValue(d, valueKey)

	return cadence.NewMeteredKeyValuePair(
		d.gauge,
		key,
		value,
	)
}

type composite struct {
	location            common.Location
	qualifiedIdentifier string
	fieldValues         []cadence.Value
	fieldTypes          []cadence.Field
}

func (d *Decoder) decodeComposite(valueJSON any) composite {
	obj := toObject(valueJSON)

	typeID := obj.GetString(idKey)
	location, qualifiedIdentifier, err := common.DecodeTypeID(d.gauge, typeID)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid type ID `%s`: %w", typeID, err))
	} else if location == nil && sema.NativeCompositeTypes[typeID] == nil {

		// If the location is nil, and there is no native composite type with this ID, then it's an invalid type.
		// Note: This is moved out from the common.DecodeTypeID() to avoid the circular dependency.
		panic(errors.NewDefaultUserError("invalid type ID for built-in: `%s`", typeID))
	}

	fields := obj.GetSlice(fieldsKey)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceField,
		Amount: uint64(len(fields)),
	})

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

func (d *Decoder) decodeCompositeField(valueJSON any) (cadence.Value, cadence.Field) {
	obj := toObject(valueJSON)

	name := obj.GetString(nameKey)
	value := obj.GetValue(d, valueKey)

	// Unmetered because decodeCompositeField is metered in decodeComposite and called nowhere else
	// Type is still metered.
	field := cadence.NewField(name, value.MeteredType(d.gauge))

	return value, field
}

func (d *Decoder) decodeStruct(valueJSON any) cadence.Struct {
	comp := d.decodeComposite(valueJSON)

	structure, err := cadence.NewMeteredStruct(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid struct: %w", err))
	}

	return structure.WithType(cadence.NewMeteredStructType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeResource(valueJSON any) cadence.Resource {
	comp := d.decodeComposite(valueJSON)

	resource, err := cadence.NewMeteredResource(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid resource: %w", err))
	}
	return resource.WithType(cadence.NewMeteredResourceType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeEvent(valueJSON any) cadence.Event {
	comp := d.decodeComposite(valueJSON)

	event, err := cadence.NewMeteredEvent(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid event: %w", err))
	}

	return event.WithType(cadence.NewMeteredEventType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeContract(valueJSON any) cadence.Contract {
	comp := d.decodeComposite(valueJSON)

	contract, err := cadence.NewMeteredContract(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid contract: %w", err))
	}

	return contract.WithType(cadence.NewMeteredContractType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeEnum(valueJSON any) cadence.Enum {
	comp := d.decodeComposite(valueJSON)

	enum, err := cadence.NewMeteredEnum(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(errors.NewDefaultUserError("invalid enum: %w", err))
	}

	return enum.WithType(cadence.NewMeteredEnumType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		nil,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodePath(valueJSON any) cadence.Path {
	obj := toObject(valueJSON)

	domain := obj.GetString(domainKey)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind: common.MemoryKindRawString,
		// no need to add 1 to account for empty string: string is metered in Path struct
		Amount: uint64(len(domain)),
	})

	identifier := obj.GetString(identifierKey)
	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind: common.MemoryKindRawString,
		// no need to add 1 to account for empty string: string is metered in Path struct
		Amount: uint64(len(identifier)),
	})

	return cadence.NewMeteredPath(
		d.gauge,
		domain,
		identifier,
	)
}

func (d *Decoder) decodeParamType(valueJSON any, results typeDecodingResults) cadence.Parameter {
	obj := toObject(valueJSON)
	// Unmetered because decodeParamType is metered in decodeParamTypes and called nowhere else
	return cadence.NewParameter(
		toString(obj.Get(labelKey)),
		toString(obj.Get(idKey)),
		d.decodeType(obj.Get(typeKey), results),
	)
}

func (d *Decoder) decodeParamTypes(params []any, results typeDecodingResults) []cadence.Parameter {
	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceParameter,
		Amount: uint64(len(params)),
	})
	parameters := make([]cadence.Parameter, 0, len(params))

	for _, param := range params {
		parameters = append(parameters, d.decodeParamType(param, results))
	}

	return parameters
}

func (d *Decoder) decodeFieldTypes(fs []any, results typeDecodingResults) []cadence.Field {
	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceField,
		Amount: uint64(len(fs)),
	})

	fields := make([]cadence.Field, 0, len(fs))

	for _, field := range fs {
		fields = append(fields, d.decodeFieldType(field, results))
	}

	return fields
}

func (d *Decoder) decodeFieldType(valueJSON any, results typeDecodingResults) cadence.Field {
	obj := toObject(valueJSON)
	// Unmetered because decodeFieldType is metered in decodeFieldTypes and called nowhere else
	return cadence.NewField(
		toString(obj.Get(idKey)),
		d.decodeType(obj.Get(typeKey), results),
	)
}

func (d *Decoder) decodePurity(purity any) cadence.FunctionPurity {
	functionPurity := toString(purity)
	if functionPurity == "view" {
		return cadence.FunctionPurityView
	}
	return cadence.FunctionPurityUnspecified
}

func (d *Decoder) decodeFunctionType(returnValue, parametersValue, id any, purity any, results typeDecodingResults) cadence.Type {
	parameters := d.decodeParamTypes(toSlice(parametersValue), results)
	returnType := d.decodeType(returnValue, results)
	functionPurity := d.decodePurity(purity)

	return cadence.NewMeteredFunctionType(
		d.gauge,
		"",
		functionPurity,
		parameters,
		returnType,
	).WithID(toString(id))
}

func (d *Decoder) decodeAuthorization(authorizationJSON any) cadence.Authorization {
	obj := toObject(authorizationJSON)
	kind := obj.Get(kindKey)
	entitlements := toSlice(obj.Get(entitlementsKey))

	switch kind {
	case "Unauthorized":
		return cadence.UnauthorizedAccess
	case "EntitlementMapAuthorization":
		m := toString(toObject(entitlements[0]).Get("typeID"))
		return cadence.NewEntitlementMapAuthorization(d.gauge, common.TypeID(m))
	case "EntitlementConjunctionSet":
		var typeIDs []common.TypeID
		for _, entitlement := range entitlements {
			id := toString(toObject(entitlement).Get("typeID"))
			typeIDs = append(typeIDs, common.TypeID(id))
		}
		return cadence.NewEntitlementSetAuthorization(d.gauge, typeIDs, cadence.Conjunction)
	case "EntitlementDisjunctionSet":
		var typeIDs []common.TypeID
		for _, entitlement := range entitlements {
			id := toString(toObject(entitlement).Get("typeID"))
			typeIDs = append(typeIDs, common.TypeID(id))
		}
		return cadence.NewEntitlementSetAuthorization(d.gauge, typeIDs, cadence.Disjunction)
	}

	panic(errors.NewDefaultUserError("invalid kind in authorization: %s", kind))
}

func (d *Decoder) decodeNominalType(
	obj jsonObject,
	kind, typeID string,
	fs, initializers []any,
	results typeDecodingResults,
) cadence.Type {

	// Unmetered because this is created as an array of nil arrays, not Parameter structs
	inits := make([][]cadence.Parameter, 0, len(initializers))
	for _, params := range initializers {
		inits = append(
			inits,
			d.decodeParamTypes(toSlice(params), results),
		)
	}

	location, qualifiedIdentifier, err := common.DecodeTypeID(d.gauge, typeID)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid type ID in nominal type: %w", err))
	}

	var result cadence.Type
	var interfaceType cadence.InterfaceType
	var compositeType cadence.CompositeType

	switch kind {
	case "Struct":
		compositeType = cadence.NewMeteredStructType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			inits,
		)
		result = compositeType
	case "Resource":
		compositeType = cadence.NewMeteredResourceType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			inits,
		)
		result = compositeType
	case "Event":
		compositeType = cadence.NewMeteredEventType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			inits[0],
		)
		result = compositeType
	case "Contract":
		compositeType = cadence.NewMeteredContractType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			inits,
		)
		result = compositeType
	case "StructInterface":
		interfaceType = cadence.NewMeteredStructInterfaceType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			inits,
		)
		result = interfaceType
	case "ResourceInterface":
		interfaceType = cadence.NewMeteredResourceInterfaceType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			inits,
		)
		result = interfaceType
	case "ContractInterface":
		interfaceType = cadence.NewMeteredContractInterfaceType(
			d.gauge,
			location,
			qualifiedIdentifier,
			nil,
			inits,
		)
		result = interfaceType
	case "Enum":
		compositeType = cadence.NewMeteredEnumType(
			d.gauge,
			location,
			qualifiedIdentifier,
			d.decodeType(obj.Get(typeKey), results),
			nil,
			inits,
		)
		result = compositeType
	default:
		panic(errors.NewDefaultUserError("invalid kind: %s", kind))
	}

	results[typeID] = result

	fields := d.decodeFieldTypes(fs, results)

	switch {
	case compositeType != nil:
		compositeType.SetCompositeFields(fields)
	case interfaceType != nil:
		interfaceType.SetInterfaceFields(fields)
	}

	return result
}

func (d *Decoder) decodeRestrictedType(
	typeValue any,
	restrictionsValue []any,
	typeIDValue string,
	results typeDecodingResults,
) cadence.Type {
	typ := d.decodeType(typeValue, results)
	restrictions := make([]cadence.Type, 0, len(restrictionsValue))
	for _, restriction := range restrictionsValue {
		restrictions = append(restrictions, d.decodeType(restriction, results))
	}

	return cadence.NewMeteredRestrictedType(
		d.gauge,
		"",
		typ,
		restrictions,
	).WithID(typeIDValue)
}

type typeDecodingResults map[string]cadence.Type

func (d *Decoder) decodeType(valueJSON any, results typeDecodingResults) cadence.Type {
	if valueJSON == "" {
		return nil
	}

	if typeID, ok := valueJSON.(string); ok {
		if result, ok := results[typeID]; ok {
			return result
		}

		// Backwards-compatibility for format <0.3.0:
		// static types were encoded as
		if d.allowUnstructuredStaticTypes {
			return cadence.TypeID(typeID)
		}
	}

	obj := toObject(valueJSON)
	kindValue := toString(obj.Get(kindKey))

	switch kindValue {
	case "Function":
		returnValue := obj.Get(returnKey)
		parametersValue := obj.Get(parametersKey)
		idValue := obj.Get(typeIDKey)
		purity, hasPurity := obj[purityKey]
		if !hasPurity {
			purity = "impure"
		}
		return d.decodeFunctionType(returnValue, parametersValue, idValue, purity, results)
	case "Restriction":
		restrictionsValue := obj.Get(restrictionsKey)
		typeIDValue := toString(obj.Get(typeIDKey))
		typeValue := obj.Get(typeKey)
		return d.decodeRestrictedType(
			typeValue,
			toSlice(restrictionsValue),
			typeIDValue,
			results,
		)
	case "Optional":
		return cadence.NewMeteredOptionalType(
			d.gauge,
			d.decodeType(obj.Get(typeKey), results),
		)
	case "VariableSizedArray":
		return cadence.NewMeteredVariableSizedArrayType(
			d.gauge,
			d.decodeType(obj.Get(typeKey), results),
		)
	case "Capability":
		return cadence.NewMeteredCapabilityType(
			d.gauge,
			d.decodeType(obj.Get(typeKey), results),
		)
	case "Dictionary":
		return cadence.NewMeteredDictionaryType(
			d.gauge,
			d.decodeType(obj.Get(keyKey), results),
			d.decodeType(obj.Get(valueKey), results),
		)
	case "ConstantSizedArray":
		size := toUInt(obj.Get(sizeKey))
		return cadence.NewMeteredConstantSizedArrayType(
			d.gauge,
			size,
			d.decodeType(obj.Get(typeKey), results),
		)
	case "Reference":
		return cadence.NewMeteredReferenceType(
			d.gauge,
			d.decodeAuthorization(obj.Get(authorizationKey)),
			d.decodeType(obj.Get(typeKey), results),
		)
	case "Any":
		return cadence.TheAnyType
	case "AnyStruct":
		return cadence.TheAnyStructType
	case "AnyStructAttachment":
		return cadence.TheAnyStructAttachmentType
	case "AnyResource":
		return cadence.TheAnyResourceType
	case "AnyResourceAttachment":
		return cadence.TheAnyResourceAttachmentType
	case "Type":
		return cadence.TheMetaType
	case "Void":
		return cadence.TheVoidType
	case "Never":
		return cadence.TheNeverType
	case "Bool":
		return cadence.TheBoolType
	case "String":
		return cadence.TheStringType
	case "Character":
		return cadence.TheCharacterType
	case "Bytes":
		return cadence.TheBytesType
	case "Address":
		return cadence.TheAddressType
	case "Number":
		return cadence.TheNumberType
	case "SignedNumber":
		return cadence.TheSignedNumberType
	case "Integer":
		return cadence.TheIntegerType
	case "SignedInteger":
		return cadence.TheSignedIntegerType
	case "FixedPoint":
		return cadence.TheFixedPointType
	case "SignedFixedPoint":
		return cadence.TheSignedFixedPointType
	case "Int":
		return cadence.TheIntType
	case "Int8":
		return cadence.TheInt8Type
	case "Int16":
		return cadence.TheInt16Type
	case "Int32":
		return cadence.TheInt32Type
	case "Int64":
		return cadence.TheInt64Type
	case "Int128":
		return cadence.TheInt128Type
	case "Int256":
		return cadence.TheInt256Type
	case "UInt":
		return cadence.TheUIntType
	case "UInt8":
		return cadence.TheUInt8Type
	case "UInt16":
		return cadence.TheUInt16Type
	case "UInt32":
		return cadence.TheUInt32Type
	case "UInt64":
		return cadence.TheUInt64Type
	case "UInt128":
		return cadence.TheUInt128Type
	case "UInt256":
		return cadence.TheUInt256Type
	case "Word8":
		return cadence.TheWord8Type
	case "Word16":
		return cadence.TheWord16Type
	case "Word32":
		return cadence.TheWord32Type
	case "Word64":
		return cadence.TheWord64Type
	case "Fix64":
		return cadence.TheFix64Type
	case "UFix64":
		return cadence.TheUFix64Type
	case "Path":
		return cadence.ThePathType
	case "CapabilityPath":
		return cadence.TheCapabilityPathType
	case "StoragePath":
		return cadence.TheStoragePathType
	case "PublicPath":
		return cadence.ThePublicPathType
	case "PrivatePath":
		return cadence.ThePrivatePathType
	case "AuthAccount":
		return cadence.TheAuthAccountType
	case "PublicAccount":
		return cadence.ThePublicAccountType
	case "AuthAccount.Keys":
		return cadence.TheAuthAccountKeysType
	case "PublicAccount.Keys":
		return cadence.ThePublicAccountKeysType
	case "AuthAccount.Contracts":
		return cadence.TheAuthAccountContractsType
	case "PublicAccount.Contracts":
		return cadence.ThePublicAccountContractsType
	case "DeployedContract":
		return cadence.TheDeployedContractType
	case "AccountKey":
		return cadence.TheAccountKeyType
	case "Block":
		return cadence.TheBlockType
	default:
		fieldsValue := obj.Get(fieldsKey)
		typeIDValue := toString(obj.Get(typeIDKey))
		initValue := obj.Get(initializersKey)
		return d.decodeNominalType(
			obj,
			kindValue,
			typeIDValue,
			toSlice(fieldsValue),
			toSlice(initValue),
			results,
		)
	}
}

func (d *Decoder) decodeTypeValue(valueJSON any) cadence.TypeValue {
	obj := toObject(valueJSON)

	return cadence.NewMeteredTypeValue(
		d.gauge,
		d.decodeType(obj.Get(staticTypeKey), typeDecodingResults{}),
	)
}

func (d *Decoder) decodeCapability(valueJSON any) cadence.StorageCapability {
	obj := toObject(valueJSON)

	path, ok := d.decodeJSON(obj.Get(pathKey)).(cadence.Path)
	if !ok {
		panic(errors.NewDefaultUserError("invalid capability: missing or invalid path"))
	}

	return cadence.NewMeteredStorageCapability(
		d.gauge,
		path,
		d.decodeAddress(obj.Get(addressKey)),
		d.decodeType(obj.Get(borrowTypeKey), typeDecodingResults{}),
	)
}

// JSON types

type jsonObject map[string]any

func (obj jsonObject) Get(key string) any {
	v, hasKey := obj[key]
	if !hasKey {
		panic(errors.NewDefaultUserError("missing property: %s", key))
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

func (obj jsonObject) GetSlice(key string) []any {
	v := obj.Get(key)
	return toSlice(v)
}

func (obj jsonObject) GetValue(d *Decoder, key string) cadence.Value {
	v := obj.Get(key)
	return d.decodeJSON(v)
}

// JSON conversion helpers

func toBool(valueJSON any) bool {
	v, isBool := valueJSON.(bool)
	if !isBool {
		panic(errors.NewDefaultUserError("expected JSON bool, got %s", valueJSON))
	}

	return v
}

func toUInt(valueJSON any) uint {
	v, isNum := valueJSON.(float64)
	if !isNum {
		panic(errors.NewDefaultUserError("expected JSON number, got %s", valueJSON))
	}

	return uint(v)
}

func toString(valueJSON any) string {
	v, isString := valueJSON.(string)
	if !isString {
		panic(errors.NewDefaultUserError("expected JSON string, got %s", valueJSON))

	}

	return v
}

func toSlice(valueJSON any) []any {
	v, isSlice := valueJSON.([]any)
	if !isSlice {
		panic(errors.NewDefaultUserError("expected JSON array, got %s", valueJSON))
	}

	return v
}

func toObject(valueJSON any) jsonObject {
	v, isMap := valueJSON.(map[string]any)
	if !isMap {
		panic(errors.NewDefaultUserError("expecte JSON object, got %s", valueJSON))
	}

	return v
}
