/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	goRuntime "runtime"
	"strconv"
	"strings"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/sema"
)

// An Encoder converts Cadence values into JSON-encoded bytes.
type Encoder struct {
	enc *json.Encoder
}

// Encode returns the JSON-encoded representation of the given value.
//
// This function returns an error if the Cadence value cannot be represented as JSON.
func Encode(value cadence.Value) ([]byte, error) {
	var w bytes.Buffer
	enc := NewEncoder(&w)

	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MustEncode returns the JSON-encoded representation of the given value, or panics
// if the value cannot be represented as JSON.
func MustEncode(value cadence.Value) []byte {
	b, err := Encode(value)
	if err != nil {
		panic(err)
	}
	return b
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
			// don't recover Go errors
			goErr, ok := r.(goRuntime.Error)
			if ok {
				panic(goErr)
			}

			panicErr, isError := r.(error)
			if !isError {
				panic(r)
			}

			err = fmt.Errorf("failed to encode value: %w", panicErr)
		}
	}()

	preparedValue := Prepare(value)

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

type jsonLinkValue struct {
	TargetPath jsonValue `json:"targetPath"`
	BorrowType string    `json:"borrowType"`
}

type jsonPathValue struct {
	Domain     string `json:"domain"`
	Identifier string `json:"identifier"`
}

type jsonTypeValue struct {
	StaticType string `json:"staticType"`
}

type jsonCapabilityValue struct {
	Path       jsonValue `json:"path"`
	Address    string    `json:"address"`
	BorrowType string    `json:"borrowType"`
}

const (
	voidTypeStr       = "Void"
	optionalTypeStr   = "Optional"
	boolTypeStr       = "Bool"
	stringTypeStr     = "String"
	addressTypeStr    = "Address"
	intTypeStr        = "Int"
	int8TypeStr       = "Int8"
	int16TypeStr      = "Int16"
	int32TypeStr      = "Int32"
	int64TypeStr      = "Int64"
	int128TypeStr     = "Int128"
	int256TypeStr     = "Int256"
	uintTypeStr       = "UInt"
	uint8TypeStr      = "UInt8"
	uint16TypeStr     = "UInt16"
	uint32TypeStr     = "UInt32"
	uint64TypeStr     = "UInt64"
	uint128TypeStr    = "UInt128"
	uint256TypeStr    = "UInt256"
	word8TypeStr      = "Word8"
	word16TypeStr     = "Word16"
	word32TypeStr     = "Word32"
	word64TypeStr     = "Word64"
	fix64TypeStr      = "Fix64"
	ufix64TypeStr     = "UFix64"
	arrayTypeStr      = "Array"
	dictionaryTypeStr = "Dictionary"
	structTypeStr     = "Struct"
	resourceTypeStr   = "Resource"
	eventTypeStr      = "Event"
	contractTypeStr   = "Contract"
	linkTypeStr       = "Link"
	pathTypeStr       = "Path"
	typeTypeStr       = "Type"
	capabilityTypeStr = "Capability"
	enumTypeStr       = "Enum"
)

// prepare traverses the object graph of the provided value and constructs
// a struct representation that can be marshalled to JSON.
func Prepare(v cadence.Value) jsonValue {
	switch x := v.(type) {
	case cadence.Void:
		return prepareVoid()
	case cadence.Optional:
		return prepareOptional(x)
	case cadence.Bool:
		return prepareBool(x)
	case cadence.String:
		return prepareString(x)
	case cadence.Address:
		return prepareAddress(x)
	case cadence.Int:
		return prepareInt(x)
	case cadence.Int8:
		return prepareInt8(x)
	case cadence.Int16:
		return prepareInt16(x)
	case cadence.Int32:
		return prepareInt32(x)
	case cadence.Int64:
		return prepareInt64(x)
	case cadence.Int128:
		return prepareInt128(x)
	case cadence.Int256:
		return prepareInt256(x)
	case cadence.UInt:
		return prepareUInt(x)
	case cadence.UInt8:
		return prepareUInt8(x)
	case cadence.UInt16:
		return prepareUInt16(x)
	case cadence.UInt32:
		return prepareUInt32(x)
	case cadence.UInt64:
		return prepareUInt64(x)
	case cadence.UInt128:
		return prepareUInt128(x)
	case cadence.UInt256:
		return prepareUInt256(x)
	case cadence.Word8:
		return prepareWord8(x)
	case cadence.Word16:
		return prepareWord16(x)
	case cadence.Word32:
		return prepareWord32(x)
	case cadence.Word64:
		return prepareWord64(x)
	case cadence.Fix64:
		return prepareFix64(x)
	case cadence.UFix64:
		return prepareUFix64(x)
	case cadence.Array:
		return prepareArray(x)
	case cadence.Dictionary:
		return prepareDictionary(x)
	case cadence.Struct:
		return prepareStruct(x)
	case cadence.Resource:
		return prepareResource(x)
	case cadence.Event:
		return prepareEvent(x)
	case cadence.Contract:
		return prepareContract(x)
	case cadence.Link:
		return prepareLink(x)
	case cadence.Path:
		return preparePath(x)
	case cadence.TypeValue:
		return prepareTypeValue(x)
	case cadence.Capability:
		return prepareCapability(x)
	case cadence.Enum:
		return prepareEnum(x)
	default:
		panic(fmt.Errorf("unsupported value: %T, %v", v, v))
	}
}

func prepareVoid() jsonValue {
	return jsonEmptyValueObject{Type: voidTypeStr}
}

func prepareOptional(v cadence.Optional) jsonValue {
	var value interface{}

	if v.Value != nil {
		value = Prepare(v.Value)
	}

	return jsonValueObject{
		Type:  optionalTypeStr,
		Value: value,
	}
}

func prepareBool(v cadence.Bool) jsonValue {
	return jsonValueObject{
		Type:  boolTypeStr,
		Value: v,
	}
}

func prepareString(v cadence.String) jsonValue {
	return jsonValueObject{
		Type:  stringTypeStr,
		Value: v,
	}
}

func prepareAddress(v cadence.Address) jsonValue {
	return jsonValueObject{
		Type:  addressTypeStr,
		Value: encodeBytes(v.Bytes()),
	}
}

func prepareInt(v cadence.Int) jsonValue {
	return jsonValueObject{
		Type:  intTypeStr,
		Value: encodeBig(v.Big()),
	}
}

func prepareInt8(v cadence.Int8) jsonValue {
	return jsonValueObject{
		Type:  int8TypeStr,
		Value: encodeInt(int64(v)),
	}
}

func prepareInt16(v cadence.Int16) jsonValue {
	return jsonValueObject{
		Type:  int16TypeStr,
		Value: encodeInt(int64(v)),
	}
}

func prepareInt32(v cadence.Int32) jsonValue {
	return jsonValueObject{
		Type:  int32TypeStr,
		Value: encodeInt(int64(v)),
	}
}

func prepareInt64(v cadence.Int64) jsonValue {
	return jsonValueObject{
		Type:  int64TypeStr,
		Value: encodeInt(int64(v)),
	}
}

func prepareInt128(v cadence.Int128) jsonValue {
	return jsonValueObject{
		Type:  int128TypeStr,
		Value: encodeBig(v.Big()),
	}
}

func prepareInt256(v cadence.Int256) jsonValue {
	return jsonValueObject{
		Type:  int256TypeStr,
		Value: encodeBig(v.Big()),
	}
}

func prepareUInt(v cadence.UInt) jsonValue {
	return jsonValueObject{
		Type:  uintTypeStr,
		Value: encodeBig(v.Big()),
	}
}

func prepareUInt8(v cadence.UInt8) jsonValue {
	return jsonValueObject{
		Type:  uint8TypeStr,
		Value: encodeUInt(uint64(v)),
	}
}

func prepareUInt16(v cadence.UInt16) jsonValue {
	return jsonValueObject{
		Type:  uint16TypeStr,
		Value: encodeUInt(uint64(v)),
	}
}

func prepareUInt32(v cadence.UInt32) jsonValue {
	return jsonValueObject{
		Type:  uint32TypeStr,
		Value: encodeUInt(uint64(v)),
	}
}

func prepareUInt64(v cadence.UInt64) jsonValue {
	return jsonValueObject{
		Type:  uint64TypeStr,
		Value: encodeUInt(uint64(v)),
	}
}

func prepareUInt128(v cadence.UInt128) jsonValue {
	return jsonValueObject{
		Type:  uint128TypeStr,
		Value: encodeBig(v.Big()),
	}
}

func prepareUInt256(v cadence.UInt256) jsonValue {
	return jsonValueObject{
		Type:  uint256TypeStr,
		Value: encodeBig(v.Big()),
	}
}

func prepareWord8(v cadence.Word8) jsonValue {
	return jsonValueObject{
		Type:  word8TypeStr,
		Value: encodeUInt(uint64(v)),
	}
}

func prepareWord16(v cadence.Word16) jsonValue {
	return jsonValueObject{
		Type:  word16TypeStr,
		Value: encodeUInt(uint64(v)),
	}
}

func prepareWord32(v cadence.Word32) jsonValue {
	return jsonValueObject{
		Type:  word32TypeStr,
		Value: encodeUInt(uint64(v)),
	}
}

func prepareWord64(v cadence.Word64) jsonValue {
	return jsonValueObject{
		Type:  word64TypeStr,
		Value: encodeUInt(uint64(v)),
	}
}

func prepareFix64(v cadence.Fix64) jsonValue {
	return jsonValueObject{
		Type:  fix64TypeStr,
		Value: encodeFix64(int64(v)),
	}
}

func prepareUFix64(v cadence.UFix64) jsonValue {
	return jsonValueObject{
		Type:  ufix64TypeStr,
		Value: encodeUFix64(uint64(v)),
	}
}

func prepareArray(v cadence.Array) jsonValue {
	values := make([]jsonValue, len(v.Values))

	for i, value := range v.Values {
		values[i] = Prepare(value)
	}

	return jsonValueObject{
		Type:  arrayTypeStr,
		Value: values,
	}
}

func prepareDictionary(v cadence.Dictionary) jsonValue {
	items := make([]jsonDictionaryItem, len(v.Pairs))

	for i, pair := range v.Pairs {
		items[i] = jsonDictionaryItem{
			Key:   Prepare(pair.Key),
			Value: Prepare(pair.Value),
		}
	}

	return jsonValueObject{
		Type:  dictionaryTypeStr,
		Value: items,
	}
}

func prepareStruct(v cadence.Struct) jsonValue {
	return prepareComposite(structTypeStr, v.StructType.ID(), v.StructType.Fields, v.Fields)
}

func prepareResource(v cadence.Resource) jsonValue {
	return prepareComposite(resourceTypeStr, v.ResourceType.ID(), v.ResourceType.Fields, v.Fields)
}

func prepareEvent(v cadence.Event) jsonValue {
	return prepareComposite(eventTypeStr, v.EventType.ID(), v.EventType.Fields, v.Fields)
}

func prepareContract(v cadence.Contract) jsonValue {
	return prepareComposite(contractTypeStr, v.ContractType.ID(), v.ContractType.Fields, v.Fields)
}

func prepareEnum(v cadence.Enum) jsonValue {
	return prepareComposite(enumTypeStr, v.EnumType.ID(), v.EnumType.Fields, v.Fields)
}

func prepareComposite(kind, id string, fieldTypes []cadence.Field, fields []cadence.Value) jsonValue {
	nonFunctionFieldTypes := make([]cadence.Field, 0)

	for _, field := range fieldTypes {
		if _, ok := field.Type.(cadence.Function); !ok {
			nonFunctionFieldTypes = append(nonFunctionFieldTypes, field)
		}
	}

	if len(nonFunctionFieldTypes) != len(fields) {
		panic(fmt.Errorf(
			"%s field count (%d) does not match declared type (%d)",
			kind,
			len(fields),
			len(nonFunctionFieldTypes),
		))
	}

	compositeFields := make([]jsonCompositeField, len(fields))

	for i, value := range fields {
		fieldType := nonFunctionFieldTypes[i]

		compositeFields[i] = jsonCompositeField{
			Name:  fieldType.Identifier,
			Value: Prepare(value),
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

func prepareLink(x cadence.Link) jsonValue {
	return jsonValueObject{
		Type: linkTypeStr,
		Value: jsonLinkValue{
			TargetPath: preparePath(x.TargetPath),
			BorrowType: x.BorrowType,
		},
	}
}

func preparePath(x cadence.Path) jsonValue {
	return jsonValueObject{
		Type: pathTypeStr,
		Value: jsonPathValue{
			Domain:     x.Domain,
			Identifier: x.Identifier,
		},
	}
}

func prepareTypeValue(x cadence.TypeValue) jsonValue {
	return jsonValueObject{
		Type: typeTypeStr,
		Value: jsonTypeValue{
			StaticType: x.StaticType,
		},
	}
}

func prepareCapability(x cadence.Capability) jsonValue {
	return jsonValueObject{
		Type: capabilityTypeStr,
		Value: jsonCapabilityValue{
			Path:       preparePath(x.Path),
			Address:    encodeBytes(x.Address.Bytes()),
			BorrowType: x.BorrowType,
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
