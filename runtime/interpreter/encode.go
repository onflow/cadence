package interpreter

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	"github.com/fxamacker/cbor/v2"
)

var cborTagSet cbor.TagSet

const cborTagBase = 2233623

const (
	cborTagDictionary = cborTagBase + iota
	// TODO: add tags for remaining types
)

func init() {
	cborTagSet = cbor.NewTagSet()
	tagOptions := cbor.TagOptions{
		EncTag: cbor.EncTagRequired,
		DecTag: cbor.DecTagRequired,
	}
	err := cborTagSet.Add(
		tagOptions,
		reflect.TypeOf(encodedDictionary{}),
		cborTagDictionary,
	)
	if err != nil {
		panic(err)
	}
}

// Encoder converts Values into CBOR-encoded bytes.
//
type Encoder struct {
	enc *cbor.Encoder
}

// EncodeValue returns the CBOR-encoded representation of the given value.
//
func EncodeValue(value Value) ([]byte, error) {
	var w bytes.Buffer
	enc, err := NewEncoder(&w)
	if err != nil {
		return nil, err
	}

	err = enc.Encode(value)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// NewEncoder initializes an Encoder that will write CBOR-encoded bytes
// to the given io.Writer.
//
func NewEncoder(w io.Writer) (*Encoder, error) {
	encMode, err := cbor.EncOptions{
		Sort: cbor.SortCanonical,
	}.EncModeWithTags(cborTagSet)
	if err != nil {
		return nil, err
	}
	enc := encMode.NewEncoder(w)
	return &Encoder{enc: enc}, nil
}

// Encode writes the CBOR-encoded representation of the given value to this
// encoder's io.Writer.
//
// This function returns an error if the given value's type is not supported
// by this encoder.
//
func (e *Encoder) Encode(v Value) error {
	return e.enc.Encode(e.prepare(v))
}

// prepare traverses the object graph of the provided value and returns
// the representation for the value that can be marshalled to CBOR.
//
func (e *Encoder) prepare(v Value) interface{} {
	switch v := v.(type) {
	case BoolValue:
		return e.prepareBool(v)

	case *StringValue:
		return e.prepareString(v)

	case *ArrayValue:
		return e.prepareArray(v)

	case *DictionaryValue:
		return e.prepareDictionary(v)

	// TODO: support remaining types

	default:
		return fmt.Errorf("unsupported value: %[1]T, %[1]v", v)
	}
}

func (e *Encoder) prepareBool(v BoolValue) bool {
	return bool(v)
}

func (e *Encoder) prepareString(v *StringValue) string {
	return v.Str
}

func (e *Encoder) prepareArray(v *ArrayValue) []interface{} {
	result := make([]interface{}, len(v.Values))

	for i, value := range v.Values {
		result[i] = e.prepare(value)
	}

	return result
}

type encodedDictionary struct {
	_       struct{} `cbor:",toarray"`
	Keys    interface{}
	Entries map[string]interface{}
}

func (e *Encoder) prepareDictionary(v *DictionaryValue) encodedDictionary {
	keys := e.prepareArray(v.Keys)

	entries := make(map[string]interface{}, len(v.Entries))

	for _, keyValue := range v.Keys.Values {
		key := dictionaryKey(keyValue)
		entries[key] = e.prepare(v.Entries[key])
	}

	return encodedDictionary{
		Keys:    keys,
		Entries: entries,
	}
}
