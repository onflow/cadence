/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
	_ "unsafe"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type pathElement interface {
	Append(w io.Writer)
}

type indexPathElement int

var _ pathElement = indexPathElement(0)

func (e indexPathElement) Append(w io.Writer) {
	_, _ = fmt.Fprintf(w, "[%d]", int(e))
}

type propertyPathElement string

var _ pathElement = propertyPathElement("")

func (e propertyPathElement) Append(w io.Writer) {
	_, _ = fmt.Fprintf(w, ".%s", e)
}

// A Decoder decodes JSON-encoded representations of Cadence values.
type Decoder struct {
	dec   *json.Decoder
	gauge common.MemoryGauge
	// allowUnstructuredStaticTypes controls if the decoding
	// of a static type as a type ID (cadence.TypeID) is allowed
	allowUnstructuredStaticTypes bool
	// backwardsCompatible controls if the decoder can decode old versions of the JSON encoding
	backwardsCompatible bool
	pathContext         []pathElement
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

// WithBackwardsCompatibility returns a new Decoder Option
// which enables backwards compatibility mode, where the decoding
// of old versions of the JSON encoding is allowed
func WithBackwardsCompatibility() Option {
	return func(decoder *Decoder) {
		decoder.backwardsCompatible = true
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
		dec:         json.NewDecoder(r),
		gauge:       gauge,
		pathContext: make([]pathElement, 0, 8),
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

			format := "failed to decode JSON-Cadence value: %w"

			path := d.getPathString()
			if path != "" {
				format += fmt.Sprintf(" (at %s)", path)
			}

			err = errors.NewDefaultUserError(format, panicErr)
		}
	}()

	value = d.decodeValue(jsonMap)
	return value, nil
}

const (
	typeKey              = "type"
	kindKey              = "kind"
	valueKey             = "value"
	keyKey               = "key"
	nameKey              = "name"
	fieldsKey            = "fields"
	initializersKey      = "initializers"
	idKey                = "id"
	targetPathKey        = "targetPath"
	borrowTypeKey        = "borrowType"
	domainKey            = "domain"
	identifierKey        = "identifier"
	staticTypeKey        = "staticType"
	addressKey           = "address"
	pathKey              = "path"
	authorizationKey     = "authorization"
	authorizedKey        = "authorized" // Deprecated. The authorized flag got replaced by the authorization field.
	entitlementsKey      = "entitlements"
	sizeKey              = "size"
	typeIDKey            = "typeID"
	restrictionsKey      = "restrictions" // Deprecated. Restricted types are removed in v1.0.0
	intersectionTypesKey = "types"
	labelKey             = "label"
	parametersKey        = "parameters"
	typeParametersKey    = "typeParameters"
	returnKey            = "return"
	typeBoundKey         = "typeBound"
	purityKey            = "purity"
	functionTypeKey      = "functionType"
	elementKey           = "element"
	startKey             = "start"
	endKey               = "end"
	stepKey              = "step"
)

func (d *Decoder) pushPath(element pathElement) {
	d.pathContext = append(d.pathContext, element)
}

func (d *Decoder) popPath() {
	if len(d.pathContext) > 0 {
		d.pathContext = d.pathContext[:len(d.pathContext)-1]
	}
}

func (d *Decoder) getPathString() string {
	if len(d.pathContext) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, element := range d.pathContext {
		element.Append(&builder)
	}
	return builder.String()
}

func (d *Decoder) decodeValue(v any) cadence.Value {
	obj := toObject(v)

	typeStr := get(d, obj, typeKey, toString)

	// void is a special case, does not have "value" field
	if typeStr == voidTypeStr {
		return d.decodeVoid(obj)
	}

	// object should only contain two keys: "type", "value"
	if len(obj) != 2 {
		panic(errors.NewDefaultUserError(
			"expected JSON object with keys `%s` and `%s`",
			typeKey,
			valueKey,
		))
	}

	return get(d, obj, valueKey, func(valueJSON any) cadence.Value {
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
		case word128TypeStr:
			return d.decodeWord128(valueJSON)
		case word256TypeStr:
			return d.decodeWord256(valueJSON)
		case fix64TypeStr:
			return d.decodeFix64(valueJSON)
		case fix128TypeStr:
			return d.decodeFix128(valueJSON)
		case ufix64TypeStr:
			return d.decodeUFix64(valueJSON)
		case ufix128TypeStr:
			return d.decodeUFix128(valueJSON)
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
		case inclusiveRangeTypeStr:
			return d.decodeInclusiveRange(valueJSON)
		case pathTypeStr:
			return d.decodePath(valueJSON)
		case typeTypeStr:
			return d.decodeTypeValue(valueJSON)
		case capabilityTypeStr:
			return d.decodeCapability(valueJSON)
		case enumTypeStr:
			return d.decodeEnum(valueJSON)
		case functionTypeStr:
			return d.decodeFunction(valueJSON)
		}

		panic(errors.NewDefaultUserError("invalid type: %s", typeStr))
	})
}

func (d *Decoder) decodeVoid(m map[string]any) cadence.Void {
	// object should not contain fields other than "type"
	if len(m) != 1 {
		panic(errors.NewDefaultUserError("invalid additional fields in Void value"))
	}

	return cadence.NewMeteredVoid(d.gauge)
}

func (d *Decoder) decodeOptional(valueJSON any) cadence.Optional {
	if valueJSON == nil {
		return cadence.NewMeteredOptional(d.gauge, nil)
	}

	return cadence.NewMeteredOptional(d.gauge, d.decodeValue(valueJSON))
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
		if utf8.ValidString(actualPrefix) {
			panic(errors.NewDefaultUserError(
				"invalid address prefix: expected %s, got %s",
				addressPrefix,
				actualPrefix,
			))
		} else {
			panic(errors.NewDefaultUserError(
				"invalid address prefix: (shown as hex) expected %x, got %x", // hex encoding user input (actualPrefix) avoids invalid UTF-8.
				addressPrefix,
				actualPrefix,
			))
		}
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

func (d *Decoder) decodeWord128(valueJSON any) cadence.Word128 {
	value, err := cadence.NewMeteredWord128FromBig(
		d.gauge,
		func() *big.Int {
			bigInt := d.decodeBigInt(valueJSON)
			if bigInt == nil {
				// TODO: propagate toString error from decodeBigInt
				panic(errors.NewDefaultUserError("invalid Word128: %s", valueJSON))
			}
			return bigInt
		},
	)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Word128: %w", err))
	}
	return value
}

func (d *Decoder) decodeWord256(valueJSON any) cadence.Word256 {
	value, err := cadence.NewMeteredWord256FromBig(
		d.gauge,
		func() *big.Int {
			bigInt := d.decodeBigInt(valueJSON)
			if bigInt == nil {
				// TODO: propagate toString error from decodeBigInt
				panic(errors.NewDefaultUserError("invalid Word256: %s", valueJSON))
			}
			return bigInt
		},
	)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Word256: %w", err))
	}
	return value
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

func (d *Decoder) decodeFix128(valueJSON any) cadence.Fix128 {
	v, err := cadence.NewFix128FromString(d.gauge, func() (string, error) {
		return toString(valueJSON), nil
	})
	if err != nil {
		panic(errors.NewDefaultUserError("invalid Fix128: %w", err))
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

func (d *Decoder) decodeUFix128(valueJSON any) cadence.UFix128 {
	v, err := cadence.NewUFix128FromString(d.gauge, func() (string, error) {
		return toString(valueJSON), nil
	})
	if err != nil {
		panic(errors.NewDefaultUserError("invalid UFix128: %w", err))
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
				d.pushPath(indexPathElement(i))
				values[i] = d.decodeValue(val)
				d.popPath()
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
				d.pushPath(indexPathElement(i))
				pairs[i] = d.decodeKeyValuePair(val)
				d.popPath()
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

	key := get(d, obj, keyKey, d.decodeValue)
	value := get(d, obj, valueKey, d.decodeValue)

	return cadence.NewMeteredKeyValuePair(
		d.gauge,
		key,
		value,
	)
}

type compositeTypeID struct {
	typeID              string
	location            common.Location
	qualifiedIdentifier string
}

type compositeFields struct {
	fieldValues []cadence.Value
	fieldTypes  []cadence.Field
}

type composite struct {
	compositeTypeID
	compositeFields
}

func (d *Decoder) decodeComposite(valueJSON any) composite {
	obj := toObject(valueJSON)

	return composite{
		compositeTypeID: get(d, obj, idKey, d.decodeCompositeTypeID),
		compositeFields: get(d, obj, fieldsKey, d.decodeCompositeFields),
	}
}

func (d *Decoder) decodeCompositeTypeID(valueJSON any) compositeTypeID {
	typeID := toString(valueJSON)

	location, qualifiedIdentifier, err := common.DecodeTypeID(d.gauge, typeID)
	if err != nil {
		panic(errors.NewDefaultUserError("invalid type ID `%s`: %s", typeID, err.Error()))
	} else if location == nil && sema.NativeCompositeTypes[typeID] == nil {

		// If the location is nil, and there is no native composite type with this ID, then it's an invalid type.
		// Note: This is moved out from the common.DecodeTypeID() to avoid the circular dependency.
		panic(errors.NewDefaultUserError("invalid type ID for built-in: `%s`", typeID))
	}

	return compositeTypeID{
		typeID:              typeID,
		location:            location,
		qualifiedIdentifier: qualifiedIdentifier,
	}
}

func (d *Decoder) decodeCompositeFields(valueJSON any) compositeFields {
	fields := toSlice(valueJSON)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceField,
		Amount: uint64(len(fields)),
	})

	fieldValues := make([]cadence.Value, len(fields))
	fieldTypes := make([]cadence.Field, len(fields))

	for i, field := range fields {
		d.pushPath(indexPathElement(i))
		value, fieldType := d.decodeCompositeField(field)
		d.popPath()

		fieldValues[i] = value
		fieldTypes[i] = fieldType
	}

	return compositeFields{
		fieldValues: fieldValues,
		fieldTypes:  fieldTypes,
	}
}

func (d *Decoder) decodeCompositeField(valueJSON any) (cadence.Value, cadence.Field) {
	obj := toObject(valueJSON)

	name := get(d, obj, nameKey, toString)
	value := get(d, obj, valueKey, d.decodeValue)

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

func (d *Decoder) decodeInclusiveRange(valueJSON any) *cadence.InclusiveRange {
	obj := toObject(valueJSON)

	start := get(d, obj, startKey, d.decodeValue)
	end := get(d, obj, endKey, d.decodeValue)
	step := get(d, obj, stepKey, d.decodeValue)

	value := cadence.NewMeteredInclusiveRange(
		d.gauge,
		start,
		end,
		step,
	)

	return value.WithType(cadence.NewMeteredInclusiveRangeType(
		d.gauge,
		start.Type(),
	))
}

func (d *Decoder) decodePath(valueJSON any) cadence.Path {
	obj := toObject(valueJSON)

	domain := get(d, obj, domainKey, func(valueJSON any) common.PathDomain {
		return common.PathDomainFromIdentifier(toString(valueJSON))
	})

	identifier := get(d, obj, identifierKey, toString)
	common.UseMemory(d.gauge, common.NewRawStringMemoryUsage(len(identifier)))

	path, err := cadence.NewMeteredPath(
		d.gauge,
		domain,
		identifier,
	)
	if err != nil {
		panic(errors.NewDefaultUserError("failed to decode path: %w", err))
	}
	return path
}

func (d *Decoder) decodeFunction(valueJSON any) cadence.Function {
	obj := toObject(valueJSON)

	functionType := get(d, obj, functionTypeKey, func(valueJSON any) *cadence.FunctionType {
		functionType, ok := d.decodeType(valueJSON, typeDecodingResults{}).(*cadence.FunctionType)
		if !ok {
			panic(errors.NewDefaultUserError("invalid function: invalid function type"))
		}

		return functionType
	})

	return cadence.NewMeteredFunction(
		d.gauge,
		functionType,
	)
}

func (d *Decoder) decodeTypeParameter(valueJSON any, results typeDecodingResults) cadence.TypeParameter {
	obj := toObject(valueJSON)

	name := get(d, obj, nameKey, toString)

	// Unmetered because decodeTypeParameter is metered in decodeTypeParameters and called nowhere else
	// TODO: getOpt
	var typeBound cadence.Type
	typeBoundObj, ok := obj[typeBoundKey]
	if ok {
		d.pushPath(propertyPathElement(typeBoundKey))
		typeBound = d.decodeType(typeBoundObj, results)
		d.popPath()
	}

	return cadence.NewTypeParameter(
		name,
		typeBound,
	)
}

func (d *Decoder) decodeTypeParameters(valueJSON any, results typeDecodingResults) []cadence.TypeParameter {
	typeParams := toSlice(valueJSON)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceTypeParameter,
		Amount: uint64(len(typeParams)),
	})

	typeParameters := make([]cadence.TypeParameter, len(typeParams))

	for i, param := range typeParams {
		d.pushPath(indexPathElement(i))
		typeParameters[i] = d.decodeTypeParameter(param, results)
		d.popPath()
	}

	return typeParameters
}

func (d *Decoder) decodeParameter(valueJSON any, results typeDecodingResults) cadence.Parameter {
	obj := toObject(valueJSON)

	label := get(d, obj, labelKey, toString)
	id := get(d, obj, idKey, toString)
	ty := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})

	// Unmetered because decodeParameter is metered in decodeParameters and called nowhere else
	return cadence.NewParameter(
		label,
		id,
		ty,
	)
}

func (d *Decoder) decodeParameters(valueJSON any, results typeDecodingResults) []cadence.Parameter {
	params := toSlice(valueJSON)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceParameter,
		Amount: uint64(len(params)),
	})

	parameters := make([]cadence.Parameter, len(params))

	for i, param := range params {
		d.pushPath(indexPathElement(i))
		parameters[i] = d.decodeParameter(param, results)
		d.popPath()
	}

	return parameters
}

func (d *Decoder) decodeInitializers(valueJSON any, results typeDecodingResults) [][]cadence.Parameter {
	initializers := toSlice(valueJSON)

	// Unmetered because this is created as an array of nil arrays, not Parameter structs
	inits := make([][]cadence.Parameter, len(initializers))

	for i, params := range initializers {
		d.pushPath(indexPathElement(i))
		inits[i] = d.decodeParameters(params, results)
		d.popPath()
	}

	return inits
}

func (d *Decoder) decodeFieldTypes(valueJSON any, results typeDecodingResults) []cadence.Field {
	fs := toSlice(valueJSON)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceField,
		Amount: uint64(len(fs)),
	})

	fields := make([]cadence.Field, len(fs))

	for i, field := range fs {
		d.pushPath(indexPathElement(i))
		fields[i] = d.decodeFieldType(field, results)
		d.popPath()
	}

	return fields
}

func (d *Decoder) decodeFieldType(valueJSON any, results typeDecodingResults) cadence.Field {
	obj := toObject(valueJSON)

	id := get(d, obj, idKey, toString)
	ty := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})

	// Unmetered because decodeFieldType is metered in decodeFieldTypes and called nowhere else
	return cadence.NewField(id, ty)
}

func (d *Decoder) decodePurity(purity any) cadence.FunctionPurity {
	functionPurity := toString(purity)
	if functionPurity == "view" {
		return cadence.FunctionPurityView
	}
	return cadence.FunctionPurityImpure
}

func (d *Decoder) decodeFunctionType(obj jsonObject, results typeDecodingResults) cadence.Type {
	// TODO: getOpt
	functionPurity := cadence.FunctionPurityImpure
	purity, ok := obj[purityKey]
	if ok {
		d.pushPath(propertyPathElement(purityKey))
		functionPurity = d.decodePurity(purity)
		d.popPath()
	}

	// TODO: getOpt
	var typeParameters []cadence.TypeParameter
	typeParametersValue := obj[typeParametersKey]
	if typeParametersValue != nil {
		d.pushPath(propertyPathElement(typeParametersKey))
		typeParameters = d.decodeTypeParameters(typeParametersValue, results)
		d.popPath()
	}

	parameters := get(d, obj, parametersKey, func(valueJSON any) []cadence.Parameter {
		return d.decodeParameters(valueJSON, results)
	})

	returnType := get(d, obj, returnKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})

	return cadence.NewMeteredFunctionType(
		d.gauge,
		functionPurity,
		typeParameters,
		parameters,
		returnType,
	)
}

func (d *Decoder) decodeAuthorization(authorizationJSON any) cadence.Authorization {
	obj := toObject(authorizationJSON)

	kind := get(d, obj, kindKey, toString)

	switch kind {
	case "Unauthorized":
		return cadence.UnauthorizedAccess

	case "EntitlementMapAuthorization":
		return d.decodeEntitlementMapAuthorization(obj)

	case "EntitlementConjunctionSet":
		return d.decodeEntitlementConjunctionSetAuthorization(obj)

	case "EntitlementDisjunctionSet":
		return d.decodeEntitlementDisjunctionSetAuthorization(obj)
	}

	panic(errors.NewDefaultUserError("invalid kind in authorization: %s", kind))
}

func (d *Decoder) decodeEntitlementMapAuthorization(obj jsonObject) cadence.Authorization {
	typeIDs := get(d, obj, entitlementsKey, d.decodeEntitlementTypeIDs)
	if len(typeIDs) != 1 {
		panic(errors.NewDefaultUserError(
			"invalid entitlement map authorization: exactly one entitlement type ID expected",
		))
	}

	return cadence.NewEntitlementMapAuthorization(d.gauge, typeIDs[0])
}

func (d *Decoder) decodeEntitlementConjunctionSetAuthorization(obj jsonObject) cadence.Authorization {
	typeIDs := get(d, obj, entitlementsKey, d.decodeEntitlementTypeIDs)

	return cadence.NewEntitlementSetAuthorization(
		d.gauge,
		typeIDs,
		cadence.Conjunction,
	)
}

func (d *Decoder) decodeEntitlementDisjunctionSetAuthorization(obj jsonObject) cadence.Authorization {
	typeIDs := get(d, obj, entitlementsKey, d.decodeEntitlementTypeIDs)

	return cadence.NewEntitlementSetAuthorization(
		d.gauge,
		typeIDs,
		cadence.Disjunction,
	)
}

func (d *Decoder) decodeEntitlementTypeIDs(valueJSON any) []common.TypeID {
	entitlements := toSlice(valueJSON)

	typeIDs := make([]common.TypeID, len(entitlements))

	for i, entitlement := range entitlements {
		d.pushPath(indexPathElement(i))
		typeIDs[i] = d.decodeEntitlementTypeID(entitlement)
		d.popPath()
	}

	return typeIDs
}

func (d *Decoder) decodeEntitlementTypeID(valueJSON any) common.TypeID {
	obj := toObject(valueJSON)
	id := get(d, obj, typeIDKey, toString)
	return common.TypeID(id)
}

//go:linkname setCompositeTypeFields github.com/onflow/cadence.setCompositeTypeFields
func setCompositeTypeFields(cadence.CompositeType, []cadence.Field)

//go:linkname setInterfaceTypeFields github.com/onflow/cadence.setInterfaceTypeFields
func setInterfaceTypeFields(cadence.InterfaceType, []cadence.Field)

func (d *Decoder) decodeNominalType(obj jsonObject, kind string, results typeDecodingResults) cadence.Type {

	inits := get(d, obj, initializersKey, func(valueJSON any) [][]cadence.Parameter {
		return d.decodeInitializers(valueJSON, results)
	})

	compositeTypeID := get(d, obj, typeIDKey, d.decodeCompositeTypeID)

	var result cadence.Type
	var interfaceType cadence.InterfaceType
	var compositeType cadence.CompositeType

	switch kind {
	case "Struct":
		compositeType = cadence.NewMeteredStructType(
			d.gauge,
			compositeTypeID.location,
			compositeTypeID.qualifiedIdentifier,
			nil,
			inits,
		)
		result = compositeType

	case "Resource":
		compositeType = cadence.NewMeteredResourceType(
			d.gauge,
			compositeTypeID.location,
			compositeTypeID.qualifiedIdentifier,
			nil,
			inits,
		)
		result = compositeType

	case "Event":
		if len(inits) != 1 {
			panic(errors.NewDefaultUserError(
				"invalid event: exactly one initializer expected, got %d",
				len(inits),
			))
		}

		compositeType = cadence.NewMeteredEventType(
			d.gauge,
			compositeTypeID.location,
			compositeTypeID.qualifiedIdentifier,
			nil,
			inits[0],
		)
		result = compositeType

	case "Contract":
		compositeType = cadence.NewMeteredContractType(
			d.gauge,
			compositeTypeID.location,
			compositeTypeID.qualifiedIdentifier,
			nil,
			inits,
		)
		result = compositeType

	case "StructInterface":
		interfaceType = cadence.NewMeteredStructInterfaceType(
			d.gauge,
			compositeTypeID.location,
			compositeTypeID.qualifiedIdentifier,
			nil,
			inits,
		)
		result = interfaceType

	case "ResourceInterface":
		interfaceType = cadence.NewMeteredResourceInterfaceType(
			d.gauge,
			compositeTypeID.location,
			compositeTypeID.qualifiedIdentifier,
			nil,
			inits,
		)
		result = interfaceType

	case "ContractInterface":
		interfaceType = cadence.NewMeteredContractInterfaceType(
			d.gauge,
			compositeTypeID.location,
			compositeTypeID.qualifiedIdentifier,
			nil,
			inits,
		)
		result = interfaceType

	case "Enum":
		rawType := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
			return d.decodeType(valueJSON, results)
		})

		compositeType = cadence.NewMeteredEnumType(
			d.gauge,
			compositeTypeID.location,
			compositeTypeID.qualifiedIdentifier,
			rawType,
			nil,
			inits,
		)
		result = compositeType

	default:
		panic(errors.NewDefaultUserError("invalid kind: %s", kind))
	}

	results[compositeTypeID.typeID] = result

	fields := get(d, obj, fieldsKey, func(valueJSON any) []cadence.Field {
		return d.decodeFieldTypes(valueJSON, results)
	})

	switch {
	case compositeType != nil:
		setCompositeTypeFields(compositeType, fields)
	case interfaceType != nil:
		setInterfaceTypeFields(interfaceType, fields)
	}

	return result
}

func (d *Decoder) decodeIntersectionType(
	obj jsonObject,
	results typeDecodingResults,
) cadence.Type {
	types := get(d, obj, intersectionTypesKey, func(valueJSON any) []cadence.Type {
		return d.decodeTypes(valueJSON, results)
	})

	return cadence.NewMeteredIntersectionType(
		d.gauge,
		types,
	)
}

func (d *Decoder) decodeTypes(valueJSON any, results typeDecodingResults) []cadence.Type {
	v := toSlice(valueJSON)

	types := make([]cadence.Type, len(v))

	for i, typ := range v {
		d.pushPath(indexPathElement(i))
		types[i] = d.decodeType(typ, results)
		d.popPath()
	}

	return types
}

type typeDecodingResults map[string]cadence.Type

var simpleTypes = func() map[string]cadence.Type {
	typeMap := make(map[string]cadence.Type, interpreter.PrimitiveStaticType_Count)

	// Bytes is not a primitive static type
	typeMap["Bytes"] = cadence.TheBytesType

	for ty := interpreter.PrimitiveStaticType(1); ty < interpreter.PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() || ty.IsDeprecated() { //nolint:staticcheck
			continue
		}

		cadenceType := cadence.PrimitiveType(ty)
		if !canEncodeAsSimpleType(cadenceType) {
			continue
		}

		semaType := ty.SemaType()

		typeMap[string(semaType.ID())] = cadenceType
	}

	return typeMap
}()

func canEncodeAsSimpleType(primitiveType cadence.PrimitiveType) bool {
	return primitiveType != cadence.PrimitiveType(interpreter.PrimitiveStaticTypeCapability)
}

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
	kindValue := get(d, obj, kindKey, toString)

	switch kindValue {
	case "Function":
		return d.decodeFunctionType(obj, results)

	case "Intersection":
		return d.decodeIntersectionType(obj, results)

	case "Optional":
		return d.decodeOptionalType(obj, results)

	case "Restriction":
		return d.decodeDeprecatedRestrictedType(obj, results)

	case "VariableSizedArray":
		return d.decodeVariableSizedArrayType(obj, results)

	case "Capability":
		return d.decodeCapabilityType(obj, results)

	case "Dictionary":
		return d.decodeDictionaryType(obj, results)

	case "InclusiveRange":
		return d.decodeInclusiveRangeType(obj, results)

	case "ConstantSizedArray":
		return d.decodeConstantSizedArrayType(obj, results)

	case "Reference":
		return d.decodeReferenceType(obj, results)

	default:
		simpleType, ok := simpleTypes[kindValue]
		if ok {
			return simpleType
		}

		return d.decodeNominalType(obj, kindValue, results)
	}
}

func (d *Decoder) decodeReferenceType(obj jsonObject, results typeDecodingResults) cadence.Type {

	ty := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})

	// Backwards-compatibility for format <v1.0.0:
	if d.backwardsCompatible {
		if _, ok := obj[authorizedKey]; ok {
			authorized := get(d, obj, authorizedKey, toBool)

			return cadence.NewDeprecatedMeteredReferenceType(
				d.gauge,
				authorized,
				ty,
			)
		}
	}

	authorization := get(d, obj, authorizationKey, d.decodeAuthorization)

	return cadence.NewMeteredReferenceType(
		d.gauge,
		authorization,
		ty,
	)
}

func (d *Decoder) decodeConstantSizedArrayType(obj jsonObject, results typeDecodingResults) cadence.Type {
	size := get(d, obj, sizeKey, toUInt)
	ty := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})

	return cadence.NewMeteredConstantSizedArrayType(
		d.gauge,
		size,
		ty,
	)
}

func (d *Decoder) decodeInclusiveRangeType(obj jsonObject, results typeDecodingResults) cadence.Type {
	elementType := get(d, obj, elementKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})
	return cadence.NewMeteredInclusiveRangeType(d.gauge, elementType)
}

func (d *Decoder) decodeDictionaryType(obj jsonObject, results typeDecodingResults) cadence.Type {
	keyType := get(d, obj, keyKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})
	valueType := get(d, obj, valueKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})
	return cadence.NewMeteredDictionaryType(
		d.gauge,
		keyType,
		valueType,
	)
}

func (d *Decoder) decodeCapabilityType(obj jsonObject, results typeDecodingResults) cadence.Type {
	ty := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})
	return cadence.NewMeteredCapabilityType(d.gauge, ty)
}

func (d *Decoder) decodeVariableSizedArrayType(obj jsonObject, results typeDecodingResults) cadence.Type {
	ty := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})
	return cadence.NewMeteredVariableSizedArrayType(d.gauge, ty)
}

func (d *Decoder) decodeOptionalType(obj jsonObject, results typeDecodingResults) cadence.Type {
	ty := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})
	return cadence.NewMeteredOptionalType(d.gauge, ty)
}

func (d *Decoder) decodeTypeValue(valueJSON any) cadence.TypeValue {
	obj := toObject(valueJSON)

	ty := get(d, obj, staticTypeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, typeDecodingResults{})
	})

	return cadence.NewMeteredTypeValue(d.gauge, ty)
}

func (d *Decoder) decodeCapability(valueJSON any) cadence.Capability {
	obj := toObject(valueJSON)

	address := get(d, obj, addressKey, d.decodeAddress)
	borrowType := get(d, obj, borrowTypeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, typeDecodingResults{})
	})

	if d.backwardsCompatible {
		if _, ok := obj[idKey]; !ok {
			path := get(d, obj, pathKey, func(valueJSON any) cadence.Path {
				path, ok := d.decodeValue(valueJSON).(cadence.Path)
				if !ok {
					panic(errors.NewDefaultUserError("invalid capability: missing or invalid path"))
				}

				return path
			})

			return cadence.NewDeprecatedMeteredPathCapability(
				d.gauge,
				address,
				path,
				borrowType,
			)
		}
	} else if _, ok := obj[pathKey]; ok {
		panic(errors.NewDefaultUserError("invalid capability: path is not supported"))
	}

	id := get(d, obj, idKey, d.decodeUInt64)

	return cadence.NewMeteredCapability(
		d.gauge,
		id,
		address,
		borrowType,
	)
}

// Deprecated: do not use in new code, only for backwards compatibility
// Restricted types got removed in v1.0.0
func (d *Decoder) decodeDeprecatedRestrictedType(
	obj jsonObject,
	results typeDecodingResults,
) cadence.Type {
	// Backwards-compatibility for format <v1.0.0:
	if !d.backwardsCompatible {
		panic("Restriction kind is not supported")
	}

	ty := get(d, obj, typeKey, func(valueJSON any) cadence.Type {
		return d.decodeType(valueJSON, results)
	})

	restrictions := get(d, obj, restrictionsKey, func(valueJSON any) []cadence.Type {
		return d.decodeTypes(valueJSON, results)
	})

	return cadence.NewDeprecatedMeteredRestrictedType(
		d.gauge,
		ty,
		restrictions,
	)
}

// JSON types

type jsonObject map[string]any

func get[T any](d *Decoder, obj jsonObject, key string, f func(valueJSON any) T) T {
	v, ok := obj[key]
	if !ok {
		panic(errors.NewDefaultUserError("missing property: %s", key))
	}

	d.pushPath(propertyPathElement(key))
	result := f(v)
	d.popPath()
	return result
}

// JSON conversion helpers

func toBool(valueJSON any) bool {
	v, ok := valueJSON.(bool)
	if !ok {
		panic(errors.NewDefaultUserError("expected JSON bool, got %s", valueJSON))
	}

	return v
}

func toUInt(valueJSON any) uint {
	v, ok := valueJSON.(float64)
	if !ok {
		panic(errors.NewDefaultUserError("expected JSON number, got %s", valueJSON))
	}

	return uint(v)
}

func toString(valueJSON any) string {
	v, ok := valueJSON.(string)
	if !ok {
		panic(errors.NewDefaultUserError("expected JSON string, got %s", valueJSON))
	}

	return v
}

func toSlice(valueJSON any) []any {
	v, ok := valueJSON.([]any)
	if !ok {
		panic(errors.NewDefaultUserError("expected JSON array, got %s", valueJSON))
	}

	return v
}

func toObject(valueJSON any) jsonObject {
	v, ok := valueJSON.(map[string]any)
	if !ok {
		panic(errors.NewDefaultUserError("expected JSON object, got %s", valueJSON))
	}

	return v
}
