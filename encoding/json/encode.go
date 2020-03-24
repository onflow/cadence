package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/dapperlabs/cadence"
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

type jsonEmptyjsonValueObject struct {
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
	case cadence.Array:
		return e.prepareArray(x)
	case cadence.Dictionary:
		return e.prepareDictionary(x)
	case cadence.Composite:
		return e.prepareComposite(x)
	default:
		return fmt.Errorf("unsupported value: %T, %v", v, v)
	}
}

func (e *Encoder) prepareVoid() jsonValue {
	return jsonEmptyjsonValueObject{Type: "Void"}
}

func (e *Encoder) prepareOptional(v cadence.Optional) jsonValue {
	return jsonValueObject{
		Type:  "Optional",
		Value: v.Value,
	}
}

func (e *Encoder) prepareBool(v cadence.Bool) jsonValue {
	return jsonValueObject{
		Type:  "Optional",
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
		Value: v.Hex(),
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
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareInt16(v cadence.Int16) jsonValue {
	return jsonValueObject{
		Type:  "Int16",
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareInt32(v cadence.Int32) jsonValue {
	return jsonValueObject{
		Type:  "Int32",
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareInt64(v cadence.Int64) jsonValue {
	return jsonValueObject{
		Type:  "Int64",
		Value: encodeInt(int(v)),
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
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareUInt16(v cadence.UInt16) jsonValue {
	return jsonValueObject{
		Type:  "UInt16",
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareUInt32(v cadence.UInt32) jsonValue {
	return jsonValueObject{
		Type:  "UInt32",
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareUInt64(v cadence.UInt64) jsonValue {
	return jsonValueObject{
		Type:  "UInt64",
		Value: encodeInt(int(v)),
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
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareWord16(v cadence.Word16) jsonValue {
	return jsonValueObject{
		Type:  "Word16",
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareWord32(v cadence.Word32) jsonValue {
	return jsonValueObject{
		Type:  "Word32",
		Value: encodeInt(int(v)),
	}
}

func (e *Encoder) prepareWord64(v cadence.Word64) jsonValue {
	return jsonValueObject{
		Type:  "Word64",
		Value: encodeInt(int(v)),
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

func (e *Encoder) prepareComposite(v cadence.Composite) jsonValue {
	var kind string
	var compositeType cadence.CompositeType

	switch c := v.Type().(type) {
	case cadence.StructType:
		kind = "Struct"
		compositeType = c.CompositeType
	case cadence.ResourceType:
		kind = "Resource"
		compositeType = c.CompositeType
	case cadence.EventType:
		kind = "Event"
		compositeType = c.CompositeType
	default:
		panic(fmt.Errorf("invalid composite type %T, must be Struct, Resource or Event", c))
	}

	fieldTypes := compositeType.Fields

	fields := make([]jsonCompositeField, len(v.Fields))

	for i, value := range v.Fields {
		fieldType := fieldTypes[i]

		fields[i] = jsonCompositeField{
			Name:  fieldType.Identifier,
			Value: e.prepare(value),
		}
	}

	return jsonValueObject{
		Type: kind,
		Value: jsonCompositeValue{
			ID:     v.Type().ID(),
			Fields: fields,
		},
	}
}

func encodeBig(v *big.Int) string {
	return v.String()
}

func encodeInt(v int) string {
	return strconv.Itoa(v)
}
