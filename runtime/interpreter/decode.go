package interpreter

import (
	"bytes"
	"fmt"
	"io"

	"github.com/fxamacker/cbor/v2"

	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/sema"
)

// A Decoder decodes CBOR-encoded representations of values.
//
type Decoder struct {
	dec *cbor.Decoder
}

// Decode returns a value decoded from its CBOR-encoded representation,
// for the given owner (can be `nil`).
//
func DecodeValue(b []byte, owner *common.Address) (Value, error) {
	r := bytes.NewReader(b)

	dec, err := NewDecoder(r)
	if err != nil {
		return nil, err
	}

	v, err := dec.Decode(owner)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// NewDecoder initializes a Decoder that will decode CBOR-encoded bytes from the
// given io.Reader.
//
func NewDecoder(r io.Reader) (*Decoder, error) {
	decMode, err := cbor.DecOptions{}.DecModeWithTags(cborTagSet)
	if err != nil {
		return nil, err
	}

	return &Decoder{decMode.NewDecoder(r)}, nil
}

// Decode reads CBOR-encoded bytes from the io.Reader and decodes them to a value.
//
// It sets the given address as the owner (can be `nil`).
//
func (d *Decoder) Decode(owner *common.Address) (Value, error) {
	var v interface{}
	err := d.dec.Decode(&v)
	if err != nil {
		return nil, err
	}

	return d.decodeValue(v, owner)
}

func (d *Decoder) decodeValue(v interface{}, owner *common.Address) (Value, error) {
	switch v := v.(type) {
	case bool:
		return BoolValue(v), nil

	case string:
		return NewStringValue(v), nil

	case []interface{}:
		return d.decodeArray(v, owner)

	case cbor.Tag:
		switch v.Number {
		case cborTagDictionary:
			return d.decodeDictionary(v.Content, owner)

		case cborTagComposite:
			return d.decodeComposite(v.Content, owner)

		default:
			return nil, fmt.Errorf("unsupported decoded tag: %d, %v", v.Number, v.Content)
		}

	default:
		return nil, fmt.Errorf("unsupported decoded type: %[1]T, %[1]v", v)
	}
}

func (d *Decoder) decodeArray(v []interface{}, owner *common.Address) (*ArrayValue, error) {
	values := make([]Value, len(v))
	for i, value := range v {
		res, err := d.decodeValue(value, owner)
		if err != nil {
			return nil, fmt.Errorf("invalid array element encoding: %w", err)
		}
		values[i] = res
	}

	return &ArrayValue{
		Values: values,
		Owner:  owner,
	}, nil
}

func (d *Decoder) decodeDictionary(v interface{}, owner *common.Address) (*DictionaryValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dictionary encoding: %T", v)
	}

	encodedKeys, ok := encoded[uint64(0)].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dictionary keys encoding")
	}

	keys, err := d.decodeArray(encodedKeys, owner)
	if err != nil {
		return nil, fmt.Errorf("invalid dictionary keys encoding: %w", err)
	}

	encodedEntries, ok := encoded[uint64(1)].(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dictionary entries encoding")
	}

	entries := make(map[string]Value, len(encodedEntries))

	for key, value := range encodedEntries {

		keyString, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("invalid dictionary key encoding")
		}

		decodedValue, err := d.decodeValue(value, owner)
		if err != nil {
			return nil, fmt.Errorf("invalid dictionary value encoding: %w", err)
		}

		entries[keyString] = decodedValue
	}

	return &DictionaryValue{
		Keys:    keys,
		Entries: entries,
		Owner:   owner,
	}, nil
}

func (d *Decoder) decodeComposite(v interface{}, owner *common.Address) (*CompositeValue, error) {

	// TODO: location

	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid composite encoding")
	}

	encodedTypeID, ok := encoded[uint64(0)].(string)
	if !ok {
		return nil, fmt.Errorf("invalid composite type ID encoding")
	}

	typeID := sema.TypeID(encodedTypeID)

	// TODO: common.CompositeKind is int, why is it encoded/decoded as uint64?
	encodedKind, ok := encoded[uint64(1)].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid composite kind encoding")
	}

	kind := common.CompositeKind(encodedKind)

	encodedFields, ok := encoded[uint64(2)].(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid composite fields encoding")
	}

	fields := make(map[string]Value, len(encodedFields))

	for name, value := range encodedFields {
		nameString, ok := name.(string)
		if !ok {
			return nil, fmt.Errorf("invalid dictionary field name encoding")
		}

		decodedValue, err := d.decodeValue(value, owner)
		if err != nil {
			return nil, fmt.Errorf("invalid dictionary field value encoding: %w", err)
		}

		fields[nameString] = decodedValue
	}

	return &CompositeValue{
		TypeID: typeID,
		Kind:   kind,
		Fields: fields,
		Owner:  owner,
	}, nil
}
