package interpreter

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	"github.com/fxamacker/cbor/v2"

	"github.com/dapperlabs/cadence/runtime/ast"
	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/sema"
)

var cborTagSet cbor.TagSet

const cborTagBase = 2233623

const (
	cborTagNilValue = cborTagBase + iota
	cborTagVoidValue
	cborTagBoolValue
	cborTagIntValue
	cborTagUIntValue
	cborTagInt8Value
	cborTagInt16Value
	cborTagInt32Value
	cborTagInt64Value
	cborTagUInt8Value
	cborTagUInt16Value
	cborTagUInt32Value
	cborTagUInt64Value
	cborTagWord8Value
	cborTagWord16Value
	cborTagWord32Value
	cborTagWord64Value
	cborTagFix64Value
	cborTagUFix64Value
	cborTagDictionaryValue
	cborTagCompositeValue
	cborTagSomeValue
	cborTagStorageValue
	cborTagPublishedValue
	cborTagStorageReferenceValue
	cborTagEphemeralReferenceValue
	cborTagAddressValue
	cborTagAuthAccountValue
	cborTagPublicAccountValue
	cborTagPathValue
	cborTagCapabilityValue
	cborTagLinkValue
	cborTagStringLocation
	cborTagAddressLocation
)

func init() {
	cborTagSet = cbor.NewTagSet()
	tagOptions := cbor.TagOptions{
		EncTag: cbor.EncTagRequired,
		DecTag: cbor.DecTagRequired,
	}

	register := func(tag uint64, encodingType interface{}) {
		err := cborTagSet.Add(
			tagOptions,
			reflect.TypeOf(encodingType),
			tag,
		)
		if err != nil {
			panic(err)
		}
	}

	types := map[uint64]interface{}{
		cborTagNilValue:  encodedNilValue{},
		cborTagVoidValue: encodedVoidValue{},
		// cborTagBoolValue:               false,
		cborTagIntValue:                encodedIntValue([]byte{}),
		cborTagUIntValue:               encodedUIntValue([]byte{}),
		cborTagInt8Value:               encodedInt8Value(0),
		cborTagInt16Value:              encodedInt16Value(0),
		cborTagInt32Value:              encodedInt32Value(0),
		cborTagInt64Value:              encodedInt64Value(0),
		cborTagUInt8Value:              encodedUInt8Value(0),
		cborTagUInt16Value:             encodedUInt16Value(0),
		cborTagUInt32Value:             encodedUInt32Value(0),
		cborTagUInt64Value:             encodedUInt64Value(0),
		cborTagWord8Value:              encodedWord8Value(0),
		cborTagWord16Value:             encodedWord16Value(0),
		cborTagWord32Value:             encodedWord32Value(0),
		cborTagWord64Value:             encodedWord64Value(0),
		cborTagDictionaryValue:         encodedDictionaryValue{},
		cborTagCompositeValue:          encodedCompositeValue{},
		cborTagSomeValue:               encodedSomeValue{},
		cborTagStorageValue:            encodedStorageValue{},
		cborTagPublishedValue:          encodedPublishedValue{},
		cborTagStorageReferenceValue:   encodedStorageReferenceValue{},
		cborTagEphemeralReferenceValue: encodedEphemeralReferenceValue{},
		cborTagAddressValue:            encodedAddressValue{},
		cborTagAuthAccountValue:        encodedAuthAccountValue{},
		cborTagPublicAccountValue:      encodedPublicAccountValue{},
		cborTagPathValue:               encodedPathValue{},
		cborTagCapabilityValue:         encodedCapabilityValue{},
		cborTagLinkValue:               encodedLinkValue{},
		cborTagStringLocation:          encodedStringLocation(""),
		cborTagAddressLocation:         encodedAddressLocation{},
	}

	// Register types
	for tag, encodedType := range types {
		register(tag, encodedType)
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
	encMode, err := cbor.CanonicalEncOptions().EncModeWithTags(cborTagSet)
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

	case NilValue:
		return e.prepareNil(v)

	case VoidValue:
		return e.prepareVoid(v)

	case BoolValue:
		return e.prepareBool(v)

	// Signed Types

	case IntValue:
		return e.prepareInt(v)

	case Int8Value:
		return e.prepareInt8(v)

	case Int16Value:
		return e.prepareInt16(v)

	case Int32Value:
		return e.prepareInt32(v)

	case Int64Value:
		return e.prepareInt64(v)

	// case Int128Value:
	// 	return e.prepareInt128(v)

	// case Int256Value:
	// 	return e.prepareInt256(v)

	// Unsigned Types

	case UIntValue:
		return e.prepareUInt(v)

	case UInt8Value:
		return e.prepareUInt8(v)

	case UInt16Value:
		return e.prepareUInt16(v)

	case UInt32Value:
		return e.prepareUInt32(v)

	case UInt64Value:
		return e.prepareUInt64(v)

	// case UInt128Value:
	// 	return e.prepareUInt128(v)

	// case UInt256Value:
	// 	return e.prepareUInt256(v)

	// Words

	case Word8Value:
		return e.prepareWord8(v)

	case Word16Value:
		return e.prepareWord16(v)

	case Word32Value:
		return e.prepareWord32(v)

	case Word64Value:
		return e.prepareWord64(v)

	// Fixed Point

	case Fix64Value:
		return e.prepareFix64(v)

	case UFix64Value:
		return e.prepareUFix64(v)

	// String

	case *StringValue:
		return e.prepareString(v)

	// Collections

	case *ArrayValue:
		return e.prepareArray(v)

	case *DictionaryValue:
		return e.prepareDictionaryValue(v)

	// Composites

	case *CompositeValue:
		return e.prepareCompositeValue(v)

	// Some

	case *SomeValue:
		return e.prepareSomeValue(v)

	// Storage

	case *StorageValue:
		return e.prepareStorageValue(v)

	case *PublishedValue:
		return e.preparePublishedValue(v)

	case *StorageReferenceValue:
		return e.prepareStorageReferenceValue(v)

	case *EphemeralReferenceValue:
		return e.prepareEphemeralReferenceValue(v)

	case AddressValue:
		return e.prepareAddressValue(v)

	case *AuthAccountValue:
		return e.prepareAuthAcccountValue(v)

	case *PublicAccountValue:
		return e.preparePublicAccountValue(v)

	case *PathValue:
		return e.preparePathValue(v)

	// Capability

	case *CapabilityValue:
		return e.prepareCapabilityValue(v)

	case *LinkValue:
		return e.prepareLinkValue(v)

	default:
		return fmt.Errorf("unsupported value: %[1]T, %[1]v", v)
	}
}

type encodedNilValue struct{}

// TODO Implement properly
func (e *Encoder) prepareNil(v NilValue) interface{} {
	return encodedNilValue{}
}

type encodedVoidValue struct{}
type encodedIntValue []byte
type encodedUIntValue []byte
type encodedInt8Value int8
type encodedInt16Value int16
type encodedInt32Value int32
type encodedInt64Value int64
type encodedUInt8Value uint8
type encodedUInt16Value uint16
type encodedUInt32Value uint32
type encodedUInt64Value uint64
type encodedWord8Value uint8
type encodedWord16Value uint16
type encodedWord32Value uint32
type encodedWord64Value uint64

// TODO Implement properly
func (e *Encoder) prepareVoid(v VoidValue) interface{} {
	return encodedVoidValue{}
}

func (e *Encoder) prepareBool(v BoolValue) bool {
	return bool(v)
}

func (e *Encoder) prepareInt(v IntValue) interface{} {
	text, err := v.Int.MarshalText()
	if err != nil {
		return encodedIntValue{}
	}
	return encodedIntValue(text)
}

func (e *Encoder) prepareInt8(v Int8Value) interface{} {
	return encodedInt8Value(v)
}

func (e *Encoder) prepareInt16(v Int16Value) interface{} {
	return encodedInt16Value(v)
}

func (e *Encoder) prepareInt32(v Int32Value) interface{} {
	return encodedInt32Value(v)
}

func (e *Encoder) prepareInt64(v Int64Value) interface{} {
	return encodedInt64Value(v)
}

// func (e *Encoder) prepareInt128(v Int128Valuecbor.Tag {
// 	return cbor.Tag{Number: , Content: v.Int}
// }

// func (e *Encoder) prepareInt256(v Int256Valuecbor.Tag {
// 	return cbor.Tag{Number: , Content: v.Int}
// }

func (e *Encoder) prepareUInt(v UIntValue) interface{} {
	encodedIntBytes, err := v.Int.GobEncode()
	if err != nil {
		return encodedUIntValue{}
	}
	return encodedUIntValue(encodedIntBytes)
}

func (e *Encoder) prepareUInt8(v UInt8Value) interface{} {
	return encodedUInt8Value(v)
}

func (e *Encoder) prepareUInt16(v UInt16Value) interface{} {
	return encodedUInt16Value(v)
}

func (e *Encoder) prepareUInt32(v UInt32Value) interface{} {
	return encodedUInt32Value(v)
}

func (e *Encoder) prepareUInt64(v UInt64Value) interface{} {
	return encodedUInt64Value(v)
}

// func (e *Encoder) prepareUInt128(v *IntValueinterface{} {
// 	return cbor.Tag{Number: UInt, Content: v.Int}
// }

// func (e *Encoder) prepareUInt256(v *IntValueinterface{} {
// 	return cbor.Tag{Number: UInt, Content: v.Int}
// }

func (e *Encoder) prepareWord8(v Word8Value) interface{} {
	return encodedWord8Value(v)
}

func (e *Encoder) prepareWord16(v Word16Value) interface{} {
	return encodedWord16Value(v)
}

func (e *Encoder) prepareWord32(v Word32Value) interface{} {
	return encodedWord32Value(v)
}

func (e *Encoder) prepareWord64(v Word64Value) interface{} {
	return encodedWord64Value(v)
}

func (e *Encoder) prepareFix64(v Fix64Value) cbor.Tag {
	return cbor.Tag{Number: cborTagFix64Value, Content: v}
}

func (e *Encoder) prepareUFix64(v UFix64Value) cbor.Tag {
	return cbor.Tag{Number: cborTagUFix64Value, Content: v}
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

type encodedDictionaryValue struct {
	Keys    interface{}            `cbor:"0,keyasint"`
	Entries map[string]interface{} `cbor:"1,keyasint"`
}

func (e *Encoder) prepareDictionaryValue(v *DictionaryValue) interface{} {
	keys := e.prepareArray(v.Keys)

	entries := make(map[string]interface{}, len(v.Entries))

	for _, keyValue := range v.Keys.Values {
		key := dictionaryKey(keyValue)
		entries[key] = e.prepare(v.Entries[key])
	}

	return encodedDictionaryValue{
		Keys:    keys,
		Entries: entries,
	}
}

type encodedCompositeValue struct {
	Location interface{}            `cbor:"0,keyasint"`
	TypeID   sema.TypeID            `cbor:"1,keyasint"`
	Kind     common.CompositeKind   `cbor:"2,keyasint"`
	Fields   map[string]interface{} `cbor:"3,keyasint"`
}

func (e *Encoder) prepareCompositeValue(v *CompositeValue) interface{} {

	fields := make(map[string]interface{}, len(v.Fields))

	for name, value := range v.Fields {
		fields[name] = e.prepare(value)
	}

	return encodedCompositeValue{
		Location: e.prepareLocation(v.Location),
		TypeID:   v.TypeID,
		Kind:     v.Kind,
		Fields:   fields,
	}
}

type encodedSomeValue struct {
	Value interface{} `cbor:"0,keyasint"`
}

func (e *Encoder) prepareSomeValue(v *SomeValue) interface{} {
	return encodedSomeValue{
		Value: e.prepare(v.Value),
	}
}

type encodedStorageValue struct {
	Address [common.AddressLength]byte `cbor:"0,keyasint"`
}

func (e *Encoder) prepareStorageValue(v *StorageValue) interface{} {
	fmt.Println("Preparing storagevalue")
	if len(v.Address) == 0 {
		return &encodedStorageValue{Address: common.Address{}}
	}
	return &encodedStorageValue{Address: v.Address}
}

type encodedPublishedValue struct {
	Address [common.AddressLength]byte `cbor:"0,keyasint"`
}

func (e *Encoder) preparePublishedValue(v *PublishedValue) interface{} {
	return &encodedPublishedValue{Address: v.Address}
}

type encodedStorageReferenceValue struct {
	Authorized           bool                       `cbor:"0,keyasint"`
	TargetStorageAddress [common.AddressLength]byte `cbor:"1,keyasint"`
	TargetKey            string                     `cbor:"2,keyasint"`
	Owner                [common.AddressLength]byte `cbor:"3,keyasint"`
}

func (e *Encoder) prepareStorageReferenceValue(v *StorageReferenceValue) interface{} {
	return &encodedStorageReferenceValue{
		Authorized:           v.Authorized,
		TargetStorageAddress: v.TargetStorageAddress,
		TargetKey:            v.TargetKey,
		Owner:                *v.Owner,
	}
}

type encodedEphemeralReferenceValue struct {
	Authorized bool        `cbor:"0,keyasint"`
	Value      interface{} `cbor:"1,keyasint"`
}

func (e *Encoder) prepareEphemeralReferenceValue(v *EphemeralReferenceValue) interface{} {
	return &encodedEphemeralReferenceValue{Authorized: v.Authorized, Value: e.prepare(v.Value)}
}

type encodedAddressValue [common.AddressLength]byte

func (e *Encoder) prepareAddressValue(v AddressValue) interface{} {
	result := encodedAddressValue{}
	return result
}

type encodedAuthAccountValue struct {
	Address interface{} `cbor:"0,keyasint"`
}

func (e *Encoder) prepareAuthAcccountValue(v *AuthAccountValue) interface{} {
	return encodedAuthAccountValue{}
}

type encodedPublicAccountValue struct {
	Address    interface{} `cbor:"0,keyasint"`
	Identifier string      `cbor:"1,keyasint"`
}

func (e *Encoder) preparePublicAccountValue(v *PublicAccountValue) interface{} {
	return &encodedPublicAccountValue{Identifier: v.Identifier}
}

type encodedPathValue struct {
	Domain     int    `cbor:"0,keyasint"`
	Identifier string `cbor:"1,keyasint"`
}

func (e *Encoder) preparePathValue(v *PathValue) interface{} {
	return &encodedPathValue{Domain: int(v.Domain), Identifier: v.Identifier}
}

type encodedCapabilityValue struct {
	Address interface{} `cbor:"0,keyasint"`
	Path    interface{} `cbor:"1,keyasint"`
}

func (e *Encoder) prepareCapabilityValue(v *CapabilityValue) interface{} {
	return encodedCapabilityValue{
		Address: e.prepareAddressValue(v.Address),
		Path:    e.preparePathValue(&v.Path),
	}
}

type encodedLinkValue struct {
	TargetPath interface{} `cbor:"0,keyasint"`
	Type       interface{} `cbor:"1,keyasint"`
}

func (e *Encoder) prepareLinkValue(v *LinkValue) interface{} {
	return encodedLinkValue{}
}

func (e *Encoder) prepareLocation(l ast.Location) interface{} {
	switch l := l.(type) {
	case ast.StringLocation:
		return e.prepareStringLocation(l)
	case ast.AddressLocation:
		return e.prepareAddressLocation(l)
	default:
		return nil
	}
}

type encodedStringLocation string

func (e *Encoder) prepareStringLocation(l ast.StringLocation) interface{} {
	return encodedStringLocation(l.ID())
}

type encodedAddressLocation []byte

func (e *Encoder) prepareAddressLocation(l ast.AddressLocation) interface{} {
	return encodedAddressLocation(l)
}
