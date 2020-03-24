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

	preparedValue := e.encode(value)

	return e.enc.Encode(&preparedValue)
}

// JSON struct definitions

type value interface{}

type emptyValueObject struct {
	Type string `json:"type"`
}

type valueObject struct {
	Type  string `json:"type"`
	Value value  `json:"value"`
}

type dictionaryItem struct {
	Key   value `json:"key"`
	Value value `json:"value"`
}

type compositeValue struct {
	ID     string           `json:"id"`
	Fields []compositeField `json:"fields"`
}

type compositeField struct {
	Name  string `json:"name"`
	Value value  `json:"value"`
}

// encode traverses the object graph of the provided value and constructs
// a struct representation that can be marshalled to JSON.
func (e *Encoder) encode(v cadence.Value) value {
	switch x := v.(type) {
	case cadence.Void:
		return e.encodeVoid()
	case cadence.Optional:
		return e.encodeOptional(x)
	case cadence.Bool:
		return e.encodeBool(x)
	case cadence.String:
		return e.encodeString(x)
	case cadence.Address:
		return e.encodeAddress(x)
	case cadence.Int:
		return e.encodeInt(x)
	case cadence.Int8:
		return e.encodeInt8(x)
	case cadence.Int16:
		return e.encodeInt16(x)
	case cadence.Int32:
		return e.encodeInt32(x)
	case cadence.Int64:
		return e.encodeInt64(x)
	case cadence.Int128:
		return e.encodeInt128(x)
	case cadence.Int256:
		return e.encodeInt256(x)
	case cadence.UInt:
		return e.encodeUInt(x)
	case cadence.UInt8:
		return e.encodeUInt8(x)
	case cadence.UInt16:
		return e.encodeUInt16(x)
	case cadence.UInt32:
		return e.encodeUInt32(x)
	case cadence.UInt64:
		return e.encodeUInt64(x)
	case cadence.UInt128:
		return e.encodeUInt128(x)
	case cadence.UInt256:
		return e.encodeUInt256(x)
	case cadence.Word8:
		return e.encodeWord8(x)
	case cadence.Word16:
		return e.encodeWord16(x)
	case cadence.Word32:
		return e.encodeWord32(x)
	case cadence.Word64:
		return e.encodeWord64(x)
	case cadence.Array:
		return e.encodeArray(x)
	case cadence.Dictionary:
		return e.encodeDictionary(x)
	case cadence.Composite:
		return e.encodeComposite(x)
	default:
		return fmt.Errorf("unsupported value: %T, %v", v, v)
	}
}

func (e *Encoder) encodeVoid() value {
	return emptyValueObject{Type: "Void"}
}

func (e *Encoder) encodeOptional(v cadence.Optional) value {
	return valueObject{
		Type:  "Optional",
		Value: v.Value,
	}
}

func (e *Encoder) encodeBool(v cadence.Bool) value {
	return valueObject{
		Type:  "Optional",
		Value: v,
	}
}

func (e *Encoder) encodeString(v cadence.String) value {
	return valueObject{
		Type:  "String",
		Value: v,
	}
}

func (e *Encoder) encodeAddress(v cadence.Address) value {
	return valueObject{
		Type:  "Address",
		Value: v.Hex(),
	}
}

func (e *Encoder) encodeInt(v cadence.Int) value {
	return valueObject{
		Type:  "Int",
		Value: e.encodeBig(v.Big()),
	}
}

func (e *Encoder) encodeBig(v *big.Int) string {
	return v.String()
}

func (e *Encoder) encodeInt8(v cadence.Int8) value {
	return valueObject{
		Type:  "Int8",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeInt16(v cadence.Int16) value {
	return valueObject{
		Type:  "Int16",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeInt32(v cadence.Int32) value {
	return valueObject{
		Type:  "Int32",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeInt64(v cadence.Int64) value {
	return valueObject{
		Type:  "Int64",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeInt128(v cadence.Int128) value {
	return valueObject{
		Type:  "Int128",
		Value: e.encodeBig(v.Big()),
	}
}

func (e *Encoder) encodeInt256(v cadence.Int256) value {
	return valueObject{
		Type:  "Int256",
		Value: e.encodeBig(v.Big()),
	}
}

func (e *Encoder) encodeUInt(v cadence.UInt) value {
	return valueObject{
		Type:  "UInt",
		Value: e.encodeBig(v.Big()),
	}
}

func (e *Encoder) encodeUInt8(v cadence.UInt8) value {
	return valueObject{
		Type:  "UInt8",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeUInt16(v cadence.UInt16) value {
	return valueObject{
		Type:  "UInt16",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeUInt32(v cadence.UInt32) value {
	return valueObject{
		Type:  "UInt32",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeUInt64(v cadence.UInt64) value {
	return valueObject{
		Type:  "UInt64",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeUInt128(v cadence.UInt128) value {
	return valueObject{
		Type:  "UInt128",
		Value: e.encodeBig(v.Big()),
	}
}

func (e *Encoder) encodeUInt256(v cadence.UInt256) value {
	return valueObject{
		Type:  "UInt256",
		Value: e.encodeBig(v.Big()),
	}
}

func (e *Encoder) encodeWord8(v cadence.Word8) value {
	return valueObject{
		Type:  "Word8",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeWord16(v cadence.Word16) value {
	return valueObject{
		Type:  "Word16",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeWord32(v cadence.Word32) value {
	return valueObject{
		Type:  "Word32",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeWord64(v cadence.Word64) value {
	return valueObject{
		Type:  "Word64",
		Value: strconv.Itoa(int(v)),
	}
}

func (e *Encoder) encodeArray(v cadence.Array) value {
	values := make([]value, len(v.Values))

	for i, value := range v.Values {
		values[i] = e.encode(value)
	}

	return valueObject{
		Type:  "Array",
		Value: values,
	}
}

func (e *Encoder) encodeDictionary(v cadence.Dictionary) value {
	items := make([]dictionaryItem, len(v.Pairs))

	for i, pair := range v.Pairs {
		items[i] = dictionaryItem{
			Key:   e.encode(pair.Key),
			Value: e.encode(pair.Value),
		}
	}

	return valueObject{
		Type:  "Dictionary",
		Value: items,
	}
}

func (e *Encoder) encodeComposite(v cadence.Composite) value {
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

	fields := make([]compositeField, len(v.Fields))

	for i, value := range v.Fields {
		fieldType := fieldTypes[i]

		fields[i] = compositeField{
			Name:  fieldType.Identifier,
			Value: e.encode(value),
		}
	}

	return valueObject{
		Type: kind,
		Value: compositeValue{
			ID:     v.Type().ID(),
			Fields: fields,
		},
	}
}
