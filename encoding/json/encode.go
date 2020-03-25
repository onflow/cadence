package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"

	"github.com/dapperlabs/cadence"
	"github.com/dapperlabs/cadence/runtime/sema"
)

// An Encoder converts Cadence values into JSON-encoded bytes.
type Encoder struct {
	enc *json.Encoder
}

// Encode returns the JSON-encoded representation of the given value.
func Encode(value cadence.Value) ([]byte, error) {
	var w bytes.Buffer
	enc := NewEncoder(&w)

	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// NewEncoder initializes an Encoder that will write JSON-encoded bytes to the
// given io.Writer.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{enc: json.NewEncoder(w)}
}

// Encode writes the JSON-encoded representation of the given value to this
// encoder's io.Writer.
//
// This function returns an error if the given value's type is not supported
// by this encoder.
func (e *Encoder) Encode(value cadence.Value) (err error) {
	// capture panics that occur during struct preparation
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}

			err = fmt.Errorf("failed to encode value: %w", err)
		}
	}()

	preparedValue := e.prepare(value)

	return e.enc.Encode(&preparedValue)
}

// JSON struct definitions

type jsonValue interface{}

type jsonValueObject struct {
	Type  string    `json:"type"`
	Value jsonValue `json:"value"`
}

type jsonEmptyValueObject struct {
	Type string `json:"type"`
}

type jsonDictionaryItem struct {
	Key   jsonValue `json:"key"`
	Value jsonValue `json:"value"`
}

type jsonCompositeValue struct {
	ID     string               `json:"id"`
	Fields []jsonCompositeField `json:"fields"`
}

type jsonCompositeField struct {
	Name  string    `json:"name"`
	Value jsonValue `json:"value"`
}

// prepare traverses the object graph of the provided value and constructs
// a struct representation that can be marshalled to JSON.
func (e *Encoder) prepare(v cadence.Value) jsonValue {
	switch x := v.(type) {
	case cadence.Void:
		return e.prepareVoid()
	case cadence.Optional:
		return e.prepareOptional(x)
	case cadence.Bool:
		return e.prepareBool(x)
	case cadence.String:
		return e.prepareString(x)
	case cadence.Address:
		return e.prepareAddress(x)
	case cadence.Int:
		return e.prepareInt(x)
	case cadence.Int8:
		return e.prepareInt8(x)
	case cadence.Int16:
		return e.prepareInt16(x)
	case cadence.Int32:
		return e.prepareInt32(x)
	case cadence.Int64:
		return e.prepareInt64(x)
	case cadence.Int128:
		return e.prepareInt128(x)
	case cadence.Int256:
		return e.prepareInt256(x)
	case cadence.UInt:
		return e.prepareUInt(x)
	case cadence.UInt8:
		return e.prepareUInt8(x)
	case cadence.UInt16:
		return e.prepareUInt16(x)
	case cadence.UInt32:
		return e.prepareUInt32(x)
	case cadence.UInt64:
		return e.prepareUInt64(x)
	case cadence.UInt128:
		return e.prepareUInt128(x)
	case cadence.UInt256:
		return e.prepareUInt256(x)
	case cadence.Word8:
		return e.prepareWord8(x)
	case cadence.Word16:
		return e.prepareWord16(x)
	case cadence.Word32:
		return e.prepareWord32(x)
	case cadence.Word64:
		return e.prepareWord64(x)
	case cadence.Fix64:
		return e.prepareFix64(x)
	case cadence.UFix64:
		return e.prepareUFix64(x)
	case cadence.Array:
		return e.prepareArray(x)
	case cadence.Dictionary:
		return e.prepareDictionary(x)
	case cadence.Struct:
		return e.prepareStruct(x)
	case cadence.Resource:
		return e.prepareResource(x)
	case cadence.Event:
		return e.prepareEvent(x)
	default:
		return fmt.Errorf("unsupported value: %T, %v", v, v)
	}
}

func (e *Encoder) prepareVoid() jsonValue {
	return jsonEmptyValueObject{Type: "Void"}
}

func (e *Encoder) prepareOptional(v cadence.Optional) jsonValue {
	var value interface{}

	if v.Value != nil {
		value = e.prepare(v.Value)
	}

	return jsonValueObject{
		Type:  "Optional",
		Value: value,
	}
}

func (e *Encoder) prepareBool(v cadence.Bool) jsonValue {
	return jsonValueObject{
		Type:  "Bool",
		Value: v,
	}
}

func (e *Encoder) prepareString(v cadence.String) jsonValue {
	return jsonValueObject{
		Type:  "String",
		Value: v,
	}
}

func (e *Encoder) prepareAddress(v cadence.Address) jsonValue {
	return jsonValueObject{
		Type:  "Address",
		Value: encodeBytes(v.Bytes()),
	}
}

func (e *Encoder) prepareInt(v cadence.Int) jsonValue {
	return jsonValueObject{
		Type:  "Int",
		Value: encodeBig(v.Big()),
	}
}

func (e *Encoder) prepareInt8(v cadence.Int8) jsonValue {
	return jsonValueObject{
		Type:  "Int8",
		Value: encodeInt(int64(v)),
	}
}

func (e *Encoder) prepareInt16(v cadence.Int16) jsonValue {
	return jsonValueObject{
		Type:  "Int16",
		Value: encodeInt(int64(v)),
	}
}

func (e *Encoder) prepareInt32(v cadence.Int32) jsonValue {
	return jsonValueObject{
		Type:  "Int32",
		Value: encodeInt(int64(v)),
	}
}

func (e *Encoder) prepareInt64(v cadence.Int64) jsonValue {
	return jsonValueObject{
		Type:  "Int64",
		Value: encodeInt(int64(v)),
	}
}

func (e *Encoder) prepareInt128(v cadence.Int128) jsonValue {
	return jsonValueObject{
		Type:  "Int128",
		Value: encodeBig(v.Big()),
	}
}

func (e *Encoder) prepareInt256(v cadence.Int256) jsonValue {
	return jsonValueObject{
		Type:  "Int256",
		Value: encodeBig(v.Big()),
	}
}

func (e *Encoder) prepareUInt(v cadence.UInt) jsonValue {
	return jsonValueObject{
		Type:  "UInt",
		Value: encodeBig(v.Big()),
	}
}

func (e *Encoder) prepareUInt8(v cadence.UInt8) jsonValue {
	return jsonValueObject{
		Type:  "UInt8",
		Value: encodeUInt(uint64(v)),
	}
}

func (e *Encoder) prepareUInt16(v cadence.UInt16) jsonValue {
	return jsonValueObject{
		Type:  "UInt16",
		Value: encodeUInt(uint64(v)),
	}
}

func (e *Encoder) prepareUInt32(v cadence.UInt32) jsonValue {
	return jsonValueObject{
		Type:  "UInt32",
		Value: encodeUInt(uint64(v)),
	}
}

func (e *Encoder) prepareUInt64(v cadence.UInt64) jsonValue {
	return jsonValueObject{
		Type:  "UInt64",
		Value: encodeUInt(uint64(v)),
	}
}

func (e *Encoder) prepareUInt128(v cadence.UInt128) jsonValue {
	return jsonValueObject{
		Type:  "UInt128",
		Value: encodeBig(v.Big()),
	}
}

func (e *Encoder) prepareUInt256(v cadence.UInt256) jsonValue {
	return jsonValueObject{
		Type:  "UInt256",
		Value: encodeBig(v.Big()),
	}
}

func (e *Encoder) prepareWord8(v cadence.Word8) jsonValue {
	return jsonValueObject{
		Type:  "Word8",
		Value: encodeUInt(uint64(v)),
	}
}

func (e *Encoder) prepareWord16(v cadence.Word16) jsonValue {
	return jsonValueObject{
		Type:  "Word16",
		Value: encodeUInt(uint64(v)),
	}
}

func (e *Encoder) prepareWord32(v cadence.Word32) jsonValue {
	return jsonValueObject{
		Type:  "Word32",
		Value: encodeUInt(uint64(v)),
	}
}

func (e *Encoder) prepareWord64(v cadence.Word64) jsonValue {
	return jsonValueObject{
		Type:  "Word64",
		Value: encodeUInt(uint64(v)),
	}
}

func (e *Encoder) prepareFix64(v cadence.Fix64) jsonValue {
	return jsonValueObject{
		Type:  "Fix64",
		Value: encodeFix64(int64(v)),
	}
}

func (e *Encoder) prepareUFix64(v cadence.UFix64) jsonValue {
	return jsonValueObject{
		Type:  "UFix64",
		Value: encodeUFix64(uint64(v)),
	}
}

func (e *Encoder) prepareArray(v cadence.Array) jsonValue {
	values := make([]jsonValue, len(v.Values))

	for i, value := range v.Values {
		values[i] = e.prepare(value)
	}

	return jsonValueObject{
		Type:  "Array",
		Value: values,
	}
}

func (e *Encoder) prepareDictionary(v cadence.Dictionary) jsonValue {
	items := make([]jsonDictionaryItem, len(v.Pairs))

	for i, pair := range v.Pairs {
		items[i] = jsonDictionaryItem{
			Key:   e.prepare(pair.Key),
			Value: e.prepare(pair.Value),
		}
	}

	return jsonValueObject{
		Type:  "Dictionary",
		Value: items,
	}
}

func (e *Encoder) prepareStruct(v cadence.Struct) jsonValue {
	return e.prepareComposite("Struct", v.StructType.ID(), v.StructType.Fields, v.Fields)
}

func (e *Encoder) prepareResource(v cadence.Resource) jsonValue {
	return e.prepareComposite("Resource", v.ResourceType.ID(), v.ResourceType.Fields, v.Fields)
}

func (e *Encoder) prepareEvent(v cadence.Event) jsonValue {
	return e.prepareComposite("Event", v.EventType.ID(), v.EventType.Fields, v.Fields)
}

func (e *Encoder) prepareComposite(kind, id string, fieldTypes []cadence.Field, fields []cadence.Value) jsonValue {
	compositeFields := make([]jsonCompositeField, len(fields))

	for i, value := range fields {
		fieldType := fieldTypes[i]

		compositeFields[i] = jsonCompositeField{
			Name:  fieldType.Identifier,
			Value: e.prepare(value),
		}
	}

	return jsonValueObject{
		Type: kind,
		Value: jsonCompositeValue{
			ID:     id,
			Fields: compositeFields,
		},
	}
}

func encodeBytes(v []byte) string {
	return fmt.Sprintf("0x%x", v)
}

func encodeBig(v *big.Int) string {
	return v.String()
}

func encodeInt(v int64) string {
	return strconv.FormatInt(v, 10)
}

func encodeUInt(v uint64) string {
	return strconv.FormatUint(v, 10)
}

func encodeFix64(v int64) string {
	integer := v / sema.Fix64Factor
	fraction := v % sema.Fix64Factor

	negative := fraction < 0

	var builder strings.Builder

	if negative {
		fraction = -fraction
		if integer == 0 {
			builder.WriteRune('-')
		}
	}

	builder.WriteString(fmt.Sprintf(
		"%d.%08d",
		integer,
		fraction,
	))

	return builder.String()
}

func encodeUFix64(v uint64) string {
	integer := v / sema.Fix64Factor
	fraction := v % sema.Fix64Factor

	return fmt.Sprintf(
		"%d.%08d",
		integer,
		fraction,
	)
}
