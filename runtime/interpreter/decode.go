package interpreter

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/fxamacker/cbor/v2"

	"github.com/dapperlabs/cadence/runtime/ast"
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

	// CBOR Types

	case bool:
		return BoolValue(v), nil

	case string:
		return NewStringValue(v), nil

	case []interface{}:
		return d.decodeArray(v, owner)

	case cbor.Tag:
		switch v.Number {

		case cborTagNilValue:
			return d.decodeNil(v.Content, owner)

		case cborTagVoidValue:
			return d.decodeVoid(v.Content, owner)

		// Signed primitive integers

		case cborTagIntValue:
			return d.decodeInt(v.Content, owner)

		case cborTagInt8Value:
			return d.decodeInt8(v.Content, owner)

		case cborTagInt16Value:
			return d.decodeInt16(v.Content, owner)

		case cborTagInt32Value:
			return d.decodeInt32(v.Content, owner)

		case cborTagInt64Value:
			return d.decodeInt64(v.Content, owner)

		case cborTagInt128Value:
			return d.decodeInt128(v.Content, owner)

		case cborTagInt256Value:
			return d.decodeInt256(v.Content, owner)

		// Unsigned primitive integers

		case cborTagUIntValue:
			return d.decodeUInt(v.Content, owner)

		case cborTagUInt8Value:
			return d.decodeUInt8(v.Content, owner)

		case cborTagUInt16Value:
			return d.decodeUInt16(v.Content, owner)

		case cborTagUInt32Value:
			return d.decodeUInt32(v.Content, owner)

		case cborTagUInt64Value:
			return d.decodeUInt64(v.Content, owner)

		case cborTagUInt128Value:
			return d.decodeUInt128(v.Content, owner)

		case cborTagUInt256Value:
			return d.decodeUInt256(v.Content, owner)

		case cborTagWord8Value:
			return d.decodeWord8(v.Content, owner)

		case cborTagWord16Value:
			return d.decodeWord16(v.Content, owner)

		case cborTagWord32Value:
			return d.decodeWord32(v.Content, owner)

		case cborTagWord64Value:
			return d.decodeWord64(v.Content, owner)

		// Struct types

		case cborTagDictionaryValue:
			return d.decodeDictionary(v.Content, owner)

		case cborTagCompositeValue:
			return d.decodeComposite(v.Content, owner)

		case cborTagSomeValue:
			return d.decodeSome(v.Content, owner)

		case cborTagStorageReferenceValue:
			return d.decodeStorageReference(v.Content, owner)

		case cborTagEphemeralReferenceValue:
			return d.decodeEphemeralReference(v.Content, owner)

		case cborTagAddressValue:
			return d.decodeAddress(v.Content, owner)

		case cborTagPathValue:
			return d.decodePath(v.Content, owner)

		case cborTagCapabilityValue:
			return d.decodeCapability(v.Content, owner)

		case cborTagLinkValue:
			return d.decodeLink(v.Content, owner)

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

	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid composite encoding")
	}

	var location ast.Location

	encodedLocation, ok := encoded[uint64(0)].(cbor.Tag)
	if !ok {
		return nil, fmt.Errorf("invalid location encoding")
	}

	if encodedLocation.Number == cborTagStringLocation {
		location = ast.StringLocation(encodedLocation.Content.(string))
	} else if encodedLocation.Number == cborTagAddressLocation {
		location = ast.AddressLocation(encodedLocation.Content.([]byte))
	} else {
		return nil, fmt.Errorf("invalid location encoding tag: %d", encodedLocation.Number)
	}

	encodedTypeID, ok := encoded[uint64(1)].(string)
	if !ok {
		return nil, fmt.Errorf("invalid composite type ID encoding")
	}

	typeID := sema.TypeID(encodedTypeID)

	// TODO: common.CompositeKind is int, why is it encoded/decoded as uint64?
	encodedKind, ok := encoded[uint64(2)].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid composite kind encoding")
	}

	kind := common.CompositeKind(encodedKind)

	encodedFields, ok := encoded[uint64(3)].(map[interface{}]interface{})
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
		Location: location,
		TypeID:   typeID,
		Kind:     kind,
		Fields:   fields,
		Owner:    owner,
	}, nil
}

func (d *Decoder) decodeNil(v interface{}, owner *common.Address) (NilValue, error) {
	return NilValue{}, nil
}

func (d *Decoder) decodeVoid(v interface{}, owner *common.Address) (VoidValue, error) {
	return VoidValue{}, nil
}

func (d *Decoder) decodeInt(v interface{}, owner *common.Address) (IntValue, error) {
	var bigInt big.Int
	decodedBytes, ok := v.([]byte)
	if !ok {
		return IntValue{}, fmt.Errorf("invalid int encoding")
	}
	err := bigInt.GobDecode(decodedBytes)
	if err != nil {
		return IntValue{}, fmt.Errorf("invalid int encoding: gob")
	}
	return IntValue{Int: &bigInt}, nil
}

func (d *Decoder) decodeInt8(v interface{}, owner *common.Address) (Int8Value, error) {
	switch v.(type) {
	case uint64:
		return Int8Value(v.(uint64)), nil
	case int64:
		return Int8Value(v.(int64)), nil
	default:
		return Int8Value(0), fmt.Errorf("unknown int8 encoding")
	}
}

func (d *Decoder) decodeInt16(v interface{}, owner *common.Address) (Int16Value, error) {
	switch v.(type) {
	case uint64:
		return Int16Value(v.(uint64)), nil
	case int64:
		return Int16Value(v.(int64)), nil
	default:
		return Int16Value(0), fmt.Errorf("unknown int16 encoding")
	}
}

func (d *Decoder) decodeInt32(v interface{}, owner *common.Address) (Int32Value, error) {
	switch v.(type) {
	case uint64:
		return Int32Value(v.(uint64)), nil
	case int64:
		return Int32Value(v.(int64)), nil
	default:
		return Int32Value(0), fmt.Errorf("unknown int32 encoding")
	}
}

func (d *Decoder) decodeInt64(v interface{}, owner *common.Address) (Int64Value, error) {
	switch v.(type) {
	case uint64:
		return Int64Value(v.(uint64)), nil
	case int64:
		return Int64Value(v.(int64)), nil
	default:
		return Int64Value(0), fmt.Errorf("unknown int64 encoding")
	}
}

func (d *Decoder) decodeInt128(v interface{}, owner *common.Address) (Int128Value, error) {
	var bigInt big.Int
	decodedBytes, ok := v.([]byte)
	if !ok {
		return Int128Value{}, fmt.Errorf("invalid int128 encoding")
	}
	err := bigInt.GobDecode(decodedBytes)
	if err != nil {
		return Int128Value{}, fmt.Errorf("invalid int128 encoding: gob: %w", err)
	}
	return Int128Value{Int: &bigInt}, nil

}

func (d *Decoder) decodeInt256(v interface{}, owner *common.Address) (Int256Value, error) {
	var bigInt big.Int
	decodedBytes, ok := v.([]byte)
	if !ok {
		return Int256Value{}, fmt.Errorf("invalid int256 encoding")
	}
	err := bigInt.GobDecode(decodedBytes)
	if err != nil {
		return Int256Value{}, fmt.Errorf("invalid int256 encoding: gob: %w", err)
	}
	return Int256Value{Int: &bigInt}, nil
}

func (d *Decoder) decodeUInt(v interface{}, owner *common.Address) (UIntValue, error) {
	var bigInt big.Int
	decodedBytes, ok := v.([]byte)
	if !ok {
		return UIntValue{}, fmt.Errorf("invalid uint encoding")
	}
	err := bigInt.GobDecode(decodedBytes)
	if err != nil {
		return UIntValue{}, fmt.Errorf("invalid uint encoding: gob")
	}
	return UIntValue{Int: &bigInt}, nil
}

func (d *Decoder) decodeUInt8(v interface{}, owner *common.Address) (UInt8Value, error) {
	switch v.(type) {
	case uint64:
		return UInt8Value(v.(uint64)), nil
	case int64:
		return UInt8Value(v.(int64)), nil
	default:
		return UInt8Value(0), fmt.Errorf("unknown uint8 encoding")
	}
}

func (d *Decoder) decodeUInt16(v interface{}, owner *common.Address) (UInt16Value, error) {
	switch v.(type) {
	case uint64:
		return UInt16Value(v.(uint64)), nil
	case int64:
		return UInt16Value(v.(int64)), nil
	default:
		return UInt16Value(0), fmt.Errorf("unknown uint16 encoding")
	}
}

func (d *Decoder) decodeUInt32(v interface{}, owner *common.Address) (UInt32Value, error) {
	switch v.(type) {
	case uint64:
		return UInt32Value(v.(uint64)), nil
	case int64:
		return UInt32Value(v.(int64)), nil
	default:
		return UInt32Value(0), fmt.Errorf("unknown UInt32 encoding")
	}
}

func (d *Decoder) decodeUInt64(v interface{}, owner *common.Address) (UInt64Value, error) {
	switch v.(type) {
	case uint64:
		return UInt64Value(v.(uint64)), nil
	case int64:
		return UInt64Value(v.(int64)), nil
	default:
		return UInt64Value(0), fmt.Errorf("unknown UInt64 encoding")
	}
}

func (d *Decoder) decodeUInt128(v interface{}, owner *common.Address) (UInt128Value, error) {
	var bigInt big.Int
	decodedBytes, ok := v.([]byte)
	if !ok {
		return UInt128Value{}, fmt.Errorf("invalid uint128 encoding")
	}
	err := bigInt.GobDecode(decodedBytes)
	if err != nil {
		return UInt128Value{}, fmt.Errorf("invalid uint256 encoding: gob: %w", err)
	}
	return UInt128Value{Int: &bigInt}, nil

}

func (d *Decoder) decodeUInt256(v interface{}, owner *common.Address) (UInt256Value, error) {
	var bigInt big.Int
	decodedBytes, ok := v.([]byte)
	if !ok {
		return UInt256Value{}, fmt.Errorf("invalid uint256 encoding")
	}
	err := bigInt.GobDecode(decodedBytes)
	if err != nil {
		return UInt256Value{}, fmt.Errorf("invalid uint256 encoding: gob: %w", err)
	}
	return UInt256Value{Int: &bigInt}, nil
}

func (d *Decoder) decodeWord8(v interface{}, owner *common.Address) (Word8Value, error) {
	switch v.(type) {
	case uint64:
		return Word8Value(v.(uint64)), nil
	case int64:
		return Word8Value(v.(int64)), nil
	default:
		return Word8Value(0), fmt.Errorf("unknown word8 encoding")
	}
}

func (d *Decoder) decodeWord16(v interface{}, owner *common.Address) (Word16Value, error) {
	switch v.(type) {
	case uint64:
		return Word16Value(v.(uint64)), nil
	case int64:
		return Word16Value(v.(int64)), nil
	default:
		return Word16Value(0), fmt.Errorf("unknown word16 encoding")
	}
}

func (d *Decoder) decodeWord32(v interface{}, owner *common.Address) (Word32Value, error) {
	switch v.(type) {
	case uint64:
		return Word32Value(v.(uint64)), nil
	case int64:
		return Word32Value(v.(int64)), nil
	default:
		return Word32Value(0), fmt.Errorf("unknown word32 encoding")
	}
}

func (d *Decoder) decodeWord64(v interface{}, owner *common.Address) (Word64Value, error) {
	switch v.(type) {
	case uint64:
		return Word64Value(v.(uint64)), nil
	case int64:
		return Word64Value(v.(int64)), nil
	default:
		return Word64Value(0), fmt.Errorf("unknown word64 encoding")
	}
}

func (d *Decoder) decodeSome(v interface{}, owner *common.Address) (*SomeValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid some encoding")
	}

	value, err := d.decodeValue(encoded[uint64(0)], owner)
	if err != nil {
		return nil, fmt.Errorf("invalid some value encoding: %w", err)
	}

	return &SomeValue{
		Value: value,
		Owner: owner,
	}, nil
}

func (d *Decoder) decodeStorageReference(v interface{}, owner *common.Address) (*StorageReferenceValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid storagereference encoding")
	}

	authorized, ok := encoded[uint64(0)].(bool)
	if !ok {
		return nil, fmt.Errorf("invalid storagereference authorized encoding")
	}

	targetStorageAddress := common.BytesToAddress(encoded[uint64(1)].([]byte))

	targetKey, ok := encoded[uint64(2)].(string)
	if !ok {
		return nil, fmt.Errorf("invalid storagereference targetkey encoding")
	}

	return &StorageReferenceValue{
		Authorized:           authorized,
		TargetStorageAddress: targetStorageAddress,
		TargetKey:            targetKey,
		Owner:                owner,
	}, nil
}

func (d *Decoder) decodeEphemeralReference(v interface{}, owner *common.Address) (*EphemeralReferenceValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid ephemeralreference encoding")
	}

	authorized, ok := encoded[uint64(0)].(bool)
	if !ok {
		return nil, fmt.Errorf("invalid ephemeralreference authorized encoding")
	}

	value, err := d.decodeValue(encoded[uint64(1)], owner)
	if err != nil {
		return nil, fmt.Errorf("invalid ephemeralreference value encoding: %w", err)
	}

	return &EphemeralReferenceValue{
		Authorized: authorized,
		Value:      value,
	}, nil
}

func (d *Decoder) decodeAddress(v interface{}, owner *common.Address) (AddressValue, error) {
	addressBytes, ok := v.([]byte)
	if !ok {
		return AddressValue{}, fmt.Errorf("invalid address encoding")
	}
	address := NewAddressValueFromBytes(addressBytes)
	return address, nil
}

func (d *Decoder) decodePath(v interface{}, owner *common.Address) (*PathValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid path encoding")
	}

	domain, ok := encoded[uint64(0)].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid path domain encoding")
	}

	identifier, ok := encoded[uint64(1)].(string)
	if !ok {
		return nil, fmt.Errorf("invalid path identifier encodingw")
	}
	return &PathValue{Domain: common.PathDomain(domain), Identifier: identifier}, nil
}

func (d *Decoder) decodeCapability(v interface{}, owner *common.Address) (*CapabilityValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid capability encoding")
	}

	address, err := d.decodeAddress(encoded[uint64(0)].(cbor.Tag).Content, owner)
	if err != nil {
		return nil, fmt.Errorf("invalid capability address encoding: %w", err)
	}

	path, err := d.decodePath(encoded[uint64(1)].(cbor.Tag).Content, owner)
	if err != nil {
		return nil, fmt.Errorf("invalid capability path encoding: %w", err)
	}

	return &CapabilityValue{Address: address, Path: *path}, nil
}

func (d *Decoder) decodeLink(v interface{}, owner *common.Address) (*LinkValue, error) {
	encoded, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid link encoding")
	}

	path, err := d.decodePath(encoded[uint64(0)].(cbor.Tag).Content, owner)
	if err != nil {
		return nil, fmt.Errorf("invalid link targetpath encoding: %w", err)
	}

	staticType, err := d.decodeStaticType(encoded[uint64(1)], owner)
	if err != nil {
		return nil, fmt.Errorf("invalid link type encoding: %w", err)
	}
	return &LinkValue{TargetPath: *path, Type: staticType}, nil
}

func (d *Decoder) decodeStaticType(v interface{}, owner *common.Address) (StaticType, error) {
	// fmt.Println("decodeStaticType called")
	switch v := v.(type) {
	case cbor.Tag:
		switch v.Number {
		case cborTagStaticType:
			if v.Content == nil {
				return TypeStaticType{}, nil
			}
			// fmt.Println(reflect.TypeOf(v.Content))
			encoded, ok := v.Content.(map[interface{}]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid statictype encoding")
			}
			return d.decodeStaticType(encoded[uint64(0)], owner)
		case cborTagCompositeStaticType:
			encoded, ok := v.Content.(map[interface{}]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid statictype encoding")
			}

			var location ast.Location

			encodedLocation, ok := encoded[uint64(0)].(cbor.Tag)
			if encoded[uint64(0)] == nil {
				location = nil
			} else if !ok {
				return nil, fmt.Errorf("invalid location encoding")
			} else {
				if encodedLocation.Number == cborTagStringLocation {
					location = ast.StringLocation(encodedLocation.Content.(string))
				} else if encodedLocation.Number == cborTagAddressLocation {
					location = ast.AddressLocation(encodedLocation.Content.([]byte))
				} else {
					return nil, fmt.Errorf("invalid location encoding tag: %d", encodedLocation.Number)
				}
			}

			var typeID sema.TypeID
			encodedTypeID, ok := encoded[uint64(1)].(string)
			if encoded[uint64(1)] == nil {
				typeID = sema.TypeID("")
			} else if !ok {
				return nil, fmt.Errorf("invalid composite type ID encoding")
			}

			typeID = sema.TypeID(encodedTypeID)
			return CompositeStaticType{
				Location: location,
				TypeID:   typeID,
			}, nil
		case cborTagInterfaceStaticType:
			return nil, fmt.Errorf("interface static type not implemented")
		case cborTagVariableSizedStaticType:
			return nil, fmt.Errorf("variable sized static type not implemented")
		case cborTagConstantSizedStaticType:
			return nil, fmt.Errorf("constant sized static type not implemented")
		case cborTagDictionaryStaticType:
			return nil, fmt.Errorf("dictionary sized static type not implemented")
		}
	case nil:
		return TypeStaticType{}, nil
	default:
		// fmt.Println(reflect.TypeOf(v))
		return nil, fmt.Errorf("invalid statictype encoding (unrecognized type)")
	}
	return nil, fmt.Errorf("nottt implemented")
}
