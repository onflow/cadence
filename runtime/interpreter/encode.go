package interpreter

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	"github.com/fxamacker/cbor/v2"

	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/sema"
)

var cborTagSet cbor.TagSet

const cborTagBase = 2233623

const (
	cborTagDictionary = cborTagBase + iota
	cborTagComposite
	// TODO: add tags for remaining types
)

func init() {
	cborTagSet = cbor.NewTagSet()
	tagOptions := cbor.TagOptions{
		EncTag: cbor.EncTagRequired,
		DecTag: cbor.DecTagRequired,
	}

	register := func(tag uint64, ty interface{}) {
		err := cborTagSet.Add(
			tagOptions,
			reflect.TypeOf(ty),
			tag,
		)
		if err != nil {
			panic(err)
		}
	}

	register(cborTagDictionary, encodedDictionary{})
	register(cborTagComposite, encodedComposite{})
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
		//Sort: cbor.SortCanonical,
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

	case *CompositeValue:
		return e.prepareComposite(v)

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
	Keys    interface{}            `cbor:"0,keyasint"`
	Entries map[string]interface{} `cbor:"1,keyasint"`
}

func (e *Encoder) prepareDictionary(v *DictionaryValue) interface{} {
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

type encodedComposite struct {
	TypeID sema.TypeID            `cbor:"0,keyasint"`
	Kind   common.CompositeKind   `cbor:"1,keyasint"`
	Fields map[string]interface{} `cbor:"2,keyasint"`
}

func (e *Encoder) prepareComposite(v *CompositeValue) interface{} {

	// TODO: location

	fields := make(map[string]interface{}, len(v.Fields))

	for name, value := range v.Fields {
		fields[name] = e.prepare(value)
	}

	return encodedComposite{
		TypeID: v.TypeID,
		Kind:   v.Kind,
		Fields: fields,
	}
}
