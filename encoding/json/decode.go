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
	"strings"

	"github.com/dapperlabs/cadence"
)

// A Decoder decodes JSON-encoded representations of Cadence values.
type Decoder struct {
	dec *json.Decoder
}

// Decode returns a Cadence value decoded from its JSON-encoded representation.
//
// This function returns an error if the bytes do not match the given type
// definition.
func Decode(b []byte) (cadence.Value, error) {
	r := bytes.NewReader(b)
	dec := NewDecoder(r)

	v, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	return v, nil
}

// NewDecoder initializes a Decoder that will decode JSON-encoded bytes from the
// given io.Reader.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{json.NewDecoder(r)}
}

// Decode reads JSON-encoded bytes from the io.Reader and decodes them to a
// Cadence value.
//
// This function returns an error if the bytes represent JSON that is malformed
// or does not conform to the Cadence JSON specification.
func (d *Decoder) Decode() (cadence.Value, error) {
	jsonMap := make(map[string]interface{})

	err := d.dec.Decode(&jsonMap)
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

	value := decodeJSON(jsonMap)
	return value, nil
}

const (
	typeKey  = "type"
	valueKey = "value"
)

var ErrDecode = errors.New("failed to decode")

func decodeJSON(v interface{}) cadence.Value {
	obj := toObject(v)

	typeStr := obj.GetString(typeKey)

	// void is a special case, does not have "value" field
	if typeStr == voidTypeStr {
		return decodeVoid(obj)
	}

	// object should only contain two keys: "type", "value"
	if len(obj) != 2 {
		panic(ErrDecode)
	}

	valueJSON := obj.Get(valueKey)

	switch typeStr {
	case optionalTypeStr:
		return decodeOptional(valueJSON)
	case boolTypeStr:
		return decodeBool(valueJSON)
	case stringTypeStr:
		return decodeString(valueJSON)
	case addressTypeStr:
		return decodeAddress(valueJSON)
	case intTypeStr:
		return decodeInt(valueJSON)
	case int8TypeStr:
		return decodeInt8(valueJSON)
	case int16TypeStr:
		return decodeInt16(valueJSON)
	case int32TypeStr:
		return decodeInt32(valueJSON)
	case int64TypeStr:
		return decodeInt64(valueJSON)
	case int128TypeStr:
		return decodeInt128(valueJSON)
	case int256TypeStr:
		return decodeInt256(valueJSON)
	case uintTypeStr:
		return decodeUInt(valueJSON)
	case uint8TypeStr:
		return decodeUInt8(valueJSON)
	case uint16TypeStr:
		return decodeUInt16(valueJSON)
	case uint32TypeStr:
		return decodeUInt32(valueJSON)
	case uint64TypeStr:
		return decodeUInt64(valueJSON)
	case uint128TypeStr:
		return decodeUInt128(valueJSON)
	case uint256TypeStr:
		return decodeUInt256(valueJSON)
	case word8TypeStr:
		return decodeWord8(valueJSON)
	case word16TypeStr:
		return decodeWord16(valueJSON)
	case word32TypeStr:
		return decodeWord32(valueJSON)
	case word64TypeStr:
		return decodeWord64(valueJSON)
	case fix64TypeStr:
		return decodeFix64(valueJSON)
	case ufix64TypeStr:
		return decodeUFix64(valueJSON)
	case arrayTypeStr:
		return decodeArray(valueJSON)
	case dictionaryTypeStr:
		return decodeDictionary(valueJSON)
	case resourceTypeStr:
		return decodeResource(valueJSON)
	case structTypeStr:
		return decodeStruct(valueJSON)
	case eventTypeStr:
		return decodeEvent(valueJSON)
	}

	panic(ErrDecode)
}

func decodeVoid(m map[string]interface{}) cadence.Value {
	// object should not contain fields other than "type"
	if len(m) != 1 {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewVoid()
}

func decodeOptional(valueJSON interface{}) cadence.Value {
	if valueJSON == nil {
		return cadence.NewOptional(nil)
	}

	return cadence.NewOptional(decodeJSON(valueJSON))
}

func decodeBool(valueJSON interface{}) cadence.Value {
	return cadence.NewBool(toBool(valueJSON))
}

func decodeString(valueJSON interface{}) cadence.Value {
	return cadence.NewString(toString(valueJSON))
}

func decodeAddress(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	// must include 0x prefix
	if v[:2] != "0x" {
		// TODO: improve error message
		panic(ErrDecode)
	}

	b, err := hex.DecodeString(v[2:])
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewAddressFromBytes(b)
}

func decodeBigInt(valueJSON interface{}) *big.Int {
	v := toString(valueJSON)

	i := new(big.Int)
	i, ok := i.SetString(v, 10)
	if !ok {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return i
}

func decodeInt(valueJSON interface{}) cadence.Value {
	return cadence.NewIntFromBig(decodeBigInt(valueJSON))
}

func decodeInt8(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewInt8(int8(i))
}

func decodeInt16(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewInt16(int16(i))
}

func decodeInt32(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewInt32(int32(i))
}

func decodeInt64(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewInt64(i)
}

func decodeInt128(valueJSON interface{}) cadence.Value {
	return cadence.NewInt128FromBig(decodeBigInt(valueJSON))
}

func decodeInt256(valueJSON interface{}) cadence.Value {
	return cadence.NewInt256FromBig(decodeBigInt(valueJSON))
}

func decodeUInt(valueJSON interface{}) cadence.Value {
	return cadence.NewUIntFromBig(decodeBigInt(valueJSON))
}

func decodeUInt8(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewUInt8(uint8(i))
}

func decodeUInt16(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewUInt16(uint16(i))
}

func decodeUInt32(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewUInt32(uint32(i))
}

func decodeUInt64(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewUInt64(i)
}

func decodeUInt128(valueJSON interface{}) cadence.Value {
	return cadence.NewUInt128FromBig(decodeBigInt(valueJSON))
}

func decodeUInt256(valueJSON interface{}) cadence.Value {
	return cadence.NewUInt256FromBig(decodeBigInt(valueJSON))
}

func decodeWord8(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewWord8(uint8(i))
}

func decodeWord16(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewWord16(uint16(i))
}

func decodeWord32(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewWord32(uint32(i))
}

func decodeWord64(valueJSON interface{}) cadence.Value {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewWord64(i)
}

func decodeFix64(valueJSON interface{}) cadence.Value {
	v := decodeFixString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewFix64(i)
}

func decodeUFix64(valueJSON interface{}) cadence.Value {
	v := decodeFixString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return cadence.NewUFix64(i)
}

func decodeFixString(valueJSON interface{}) string {
	v := toString(valueJSON)

	// must contain single decimal point
	parts := strings.Split(v, ".")
	if len(parts) != 2 {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return parts[0] + parts[1]
}

func decodeValues(valueJSON interface{}) []cadence.Value {
	v := toSlice(valueJSON)

	values := make([]cadence.Value, len(v))

	for i, val := range v {
		values[i] = decodeJSON(val)
	}

	return values
}

func decodeArray(valueJSON interface{}) cadence.Value {
	return cadence.NewArray(decodeValues(valueJSON))
}

func decodeDictionary(valueJSON interface{}) cadence.Value {
	v := toSlice(valueJSON)

	pairs := make([]cadence.KeyValuePair, len(v))

	for i, val := range v {
		pairs[i] = decodeKeyValuePair(val)
	}

	return cadence.NewDictionary(pairs)
}

func decodeKeyValuePair(valueJSON interface{}) cadence.KeyValuePair {
	obj := toObject(valueJSON)

	key := obj.GetValue("key")
	value := obj.GetValue("value")

	return cadence.KeyValuePair{
		Key:   key,
		Value: value,
	}
}

type composite struct {
	typeID      string
	identifier  string
	fieldValues []cadence.Value
	fieldTypes  []cadence.Field
}

func decodeComposite(valueJSON interface{}) composite {
	obj := toObject(valueJSON)

	typeID := obj.GetString("id")

	identifier := identifierFromTypeID(typeID)

	fields := obj.GetSlice("fields")

	fieldValues := make([]cadence.Value, len(fields))
	fieldTypes := make([]cadence.Field, len(fields))

	for i, field := range fields {
		value, fieldType := decodeCompositeField(field)

		fieldValues[i] = value
		fieldTypes[i] = fieldType
	}

	return composite{
		typeID:      typeID,
		identifier:  identifier,
		fieldValues: fieldValues,
		fieldTypes:  fieldTypes,
	}
}

func decodeCompositeField(valueJSON interface{}) (cadence.Value, cadence.Field) {
	obj := toObject(valueJSON)

	name := obj.GetString("name")
	value := obj.GetValue("value")

	field := cadence.Field{
		Identifier: name,
		Type:       value.Type(),
	}

	return value, field
}

func decodeStruct(valueJSON interface{}) cadence.Value {
	comp := decodeComposite(valueJSON)

	return cadence.NewStruct(comp.fieldValues).WithType(cadence.StructType{
		TypeID:     comp.typeID,
		Identifier: comp.identifier,
		Fields:     comp.fieldTypes,
	})
}

func decodeResource(valueJSON interface{}) cadence.Value {
	comp := decodeComposite(valueJSON)

	return cadence.NewResource(comp.fieldValues).WithType(cadence.ResourceType{
		TypeID:     comp.typeID,
		Identifier: comp.identifier,
		Fields:     comp.fieldTypes,
	})
}

func decodeEvent(valueJSON interface{}) cadence.Value {
	comp := decodeComposite(valueJSON)

	return cadence.NewEvent(comp.fieldValues).WithType(cadence.EventType{
		TypeID:     comp.typeID,
		Identifier: comp.identifier,
		Fields:     comp.fieldTypes,
	})
}

// JSON types

type jsonObject map[string]interface{}

func (obj jsonObject) Get(key string) interface{} {
	v, hasKey := obj[key]
	if !hasKey {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return v
}

func (obj jsonObject) GetString(key string) string {
	v := obj.Get(key)
	return toString(v)
}

func (obj jsonObject) GetSlice(key string) []interface{} {
	v := obj.Get(key)
	return toSlice(v)
}

func (obj jsonObject) GetValue(key string) cadence.Value {
	v := obj.Get(key)
	return decodeJSON(v)
}

// JSON conversion helpers

func toBool(valueJSON interface{}) bool {
	v, isBool := valueJSON.(bool)
	if !isBool {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return v
}

func toString(valueJSON interface{}) string {
	v, isString := valueJSON.(string)
	if !isString {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return v
}

func toSlice(valueJSON interface{}) []interface{} {
	v, isSlice := valueJSON.([]interface{})
	if !isSlice {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return v
}

func toObject(valueJSON interface{}) jsonObject {
	v, isMap := valueJSON.(map[string]interface{})
	if !isMap {
		// TODO: improve error message
		panic(ErrDecode)
	}

	return v
}

func identifierFromTypeID(typeID string) string {
	// fully-qualified type ID must have at least two parts
	// (namespace + ID)
	// e.g. foo.Bar
	parts := strings.Split(typeID, ".")
	if len(parts) < 2 {
		// TODO: improve error message
		panic(ErrDecode)
	}

	// parse ID from fully-qualified type ID
	return parts[len(parts)-1]
}
