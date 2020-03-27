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

	value := decodeJSON(jsonMap)
	return value, nil
}

const (
	typeKey  = "type"
	valueKey = "value"
)

var ErrDecode = errors.New("failed to decode")

func getType(m map[string]interface{}) string {
	t, hasType := m[typeKey]
	if !hasType {
		panic(ErrDecode)
	}

	typeStr, isString := t.(string)
	if !isString {
		panic(ErrDecode)
	}

	return typeStr
}

func getValue(m map[string]interface{}) interface{} {
	valueJSON, hasValue := m[valueKey]
	if !hasValue {
		panic(ErrDecode)
	}

	return valueJSON
}

func decodeJSON(v interface{}) cadence.Value {
	m, isMap := v.(map[string]interface{})
	if !isMap {
		panic(ErrDecode)
	}

	typeStr := getType(m)

	// void is a special case, does not have "value" field
	if typeStr == voidTypeStr {
		return decodeVoid(m)
	}

	// object should only contain two keys: "type", "value"
	if len(m) != 2 {
		panic(ErrDecode)
	}

	valueJSON := getValue(m)

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
	v, isBool := valueJSON.(bool)
	if !isBool {
		panic(ErrDecode)
	}

	return cadence.NewBool(v)
}

func decodeString(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	return cadence.NewString(v)
}

func decodeAddress(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	if v[:2] != "0x" {
		panic(ErrDecode)
	}

	b, err := hex.DecodeString(v[2:])
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewAddressFromBytes(b)
}

func decodeBigInt(valueJSON interface{}) *big.Int {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i := new(big.Int)
	i, ok := i.SetString(v, 10)
	if !ok {
		panic(ErrDecode)
	}

	return i
}

func decodeInt(valueJSON interface{}) cadence.Value {
	return cadence.NewIntFromBig(decodeBigInt(valueJSON))
}

func decodeInt8(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseInt(v, 10, 8)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewInt8(int8(i))
}

func decodeInt16(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseInt(v, 10, 16)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewInt16(int16(i))
}

func decodeInt32(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewInt32(int32(i))
}

func decodeInt64(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
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
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewUInt8(uint8(i))
}

func decodeUInt16(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewUInt16(uint16(i))
}

func decodeUInt32(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewUInt32(uint32(i))
}

func decodeUInt64(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
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
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewWord8(uint8(i))
}

func decodeWord16(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewWord16(uint16(i))
}

func decodeWord32(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewWord32(uint32(i))
}

func decodeWord64(valueJSON interface{}) cadence.Value {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewWord64(i)
}

func decodeFix64(valueJSON interface{}) cadence.Value {
	str := decodeFixString(valueJSON)

	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewFix64(i)
}

func decodeUFix64(valueJSON interface{}) cadence.Value {
	str := decodeFixString(valueJSON)

	i, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		panic(ErrDecode)
	}

	return cadence.NewUFix64(i)
}

func decodeFixString(valueJSON interface{}) string {
	v, isString := valueJSON.(string)
	if !isString {
		panic(ErrDecode)
	}

	pieces := strings.Split(v, ".")
	if len(pieces) != 2 {
		panic(ErrDecode)
	}

	return pieces[0] + pieces[1]
}

func decodeValues(valueJSON interface{}) []cadence.Value {
	v, isSlice := valueJSON.([]interface{})
	if !isSlice {
		panic(ErrDecode)
	}

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
	v, isSlice := valueJSON.([]interface{})
	if !isSlice {
		panic(ErrDecode)
	}

	pairs := make([]cadence.KeyValuePair, len(v))

	for i, val := range v {
		m, isMap := val.(map[string]interface{})
		if !isMap {
			panic(ErrDecode)
		}

		key, hasKey := m["key"]
		if !hasKey {
			panic(ErrDecode)
		}

		value, hasValue := m["value"]
		if !hasValue {
			panic(ErrDecode)
		}

		pairs[i] = cadence.KeyValuePair{
			Key:   decodeJSON(key),
			Value: decodeJSON(value),
		}
	}

	return cadence.NewDictionary(pairs)
}

type composite struct {
	typeID      string
	identifier  string
	fieldValues []cadence.Value
	fieldTypes  []cadence.Field
}

func decodeComposite(valueJSON interface{}) composite {
	m, isMap := valueJSON.(map[string]interface{})
	if !isMap {
		panic(ErrDecode)
	}

	typeID, hasID := m["id"]
	if !hasID {
		panic(ErrDecode)
	}

	typeIDStr, isString := typeID.(string)
	if !isString {
		panic(ErrDecode)
	}

	pieces := strings.Split(typeIDStr, ".")
	if len(pieces) < 2 {
		panic(ErrDecode)
	}

	identifier := pieces[len(pieces)-1]

	fields, hasFields := m["fields"]
	if !hasFields {
		panic(ErrDecode)
	}

	v, isSlice := fields.([]interface{})
	if !isSlice {
		panic(ErrDecode)
	}

	fieldValues := make([]cadence.Value, len(v))
	fieldTypes := make([]cadence.Field, len(v))

	for i, field := range v {
		m, isMap := field.(map[string]interface{})
		if !isMap {
			panic(ErrDecode)
		}

		name, hasName := m["name"]
		if !hasName {
			panic(ErrDecode)
		}

		nameStr, isString := name.(string)
		if !isString {
			panic(ErrDecode)
		}

		value, hasValue := m["value"]
		if !hasValue {
			panic(ErrDecode)
		}

		decodedValue := decodeJSON(value)

		fieldValues[i] = decodedValue
		fieldTypes[i] = cadence.Field{
			Identifier: nameStr,
			Type:       decodedValue.Type(),
		}
	}
	return composite{
		typeID:      typeIDStr,
		identifier:  identifier,
		fieldValues: fieldValues,
		fieldTypes:  fieldTypes,
	}
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
