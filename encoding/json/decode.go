/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// A Decoder decodes JSON-encoded representations of Cadence values.
type Decoder struct {
	dec *json.Decoder
}

// Decode returns a Cadence value decoded from its JSON-encoded representation.
//
// This function returns an error if the bytes represent JSON that is malformed
// or does not conform to the JSON Cadence specification.
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
// or does not conform to the JSON Cadence specification.
func (d *Decoder) Decode() (value cadence.Value, err error) {
	jsonMap := make(map[string]interface{})

	err = d.dec.Decode(&jsonMap)
	if err != nil {
		return nil, fmt.Errorf("json-cdc: failed to decode valid JSON structure: %w", err)
	}

	// capture panics that occur during decoding
	defer func() {
		if r := recover(); r != nil {
			panicErr, isError := r.(error)
			if !isError {
				panic(r)
			}

			err = fmt.Errorf("failed to decode value: %w", panicErr)
		}
	}()

	value = decodeJSON(jsonMap)
	return value, nil
}

const (
	typeKey         = "type"
	kindKey         = "kind"
	valueKey        = "value"
	keyKey          = "key"
	nameKey         = "name"
	fieldsKey       = "fields"
	initializersKey = "initializers"
	idKey           = "id"
	targetPathKey   = "targetPath"
	borrowTypeKey   = "borrowType"
	domainKey       = "domain"
	identifierKey   = "identifier"
	staticTypeKey   = "staticType"
	addressKey      = "address"
	pathKey         = "path"
	authorizedKey   = "authorized"
	sizeKey         = "size"
	typeIDKey       = "typeID"
	restrictionsKey = "restrictions"
	labelKey        = "label"
	parametersKey   = "parameters"
	returnKey       = "return"
)

var ErrInvalidJSONCadence = errors.New("invalid JSON Cadence structure")

func decodeJSON(v interface{}) cadence.Value {
	obj := toObject(v)

	typeStr := obj.GetString(typeKey)

	// void is a special case, does not have "value" field
	if typeStr == voidTypeStr {
		return decodeVoid(obj)
	}

	// object should only contain two keys: "type", "value"
	if len(obj) != 2 {
		panic(ErrInvalidJSONCadence)
	}

	valueJSON := obj.Get(valueKey)

	switch typeStr {
	case optionalTypeStr:
		return decodeOptional(valueJSON)
	case boolTypeStr:
		return decodeBool(valueJSON)
	case characterTypeStr:
		return decodeCharacter(valueJSON)
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
	case contractTypeStr:
		return decodeContract(valueJSON)
	case linkTypeStr:
		return decodeLink(valueJSON)
	case pathTypeStr:
		return decodePath(valueJSON)
	case typeTypeStr:
		return decodeTypeValue(valueJSON)
	case capabilityTypeStr:
		return decodeCapability(valueJSON)
	case enumTypeStr:
		return decodeEnum(valueJSON)
	}

	panic(ErrInvalidJSONCadence)
}

func decodeVoid(m map[string]interface{}) cadence.Void {
	// object should not contain fields other than "type"
	if len(m) != 1 {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewVoid()
}

func decodeOptional(valueJSON interface{}) cadence.Optional {
	if valueJSON == nil {
		return cadence.NewOptional(nil)
	}

	return cadence.NewOptional(decodeJSON(valueJSON))
}

func decodeBool(valueJSON interface{}) cadence.Bool {
	return cadence.NewBool(toBool(valueJSON))
}

func decodeCharacter(valueJSON interface{}) cadence.Character {
	char, err := cadence.NewCharacter(toString(valueJSON))
	if err != nil {
		panic(err)
	}
	return char
}

func decodeString(valueJSON interface{}) cadence.String {
	str, err := cadence.NewString(toString(valueJSON))
	if err != nil {
		panic(err)
	}
	return str
}

func decodeAddress(valueJSON interface{}) cadence.Address {
	v := toString(valueJSON)

	// must include 0x prefix
	if v[:2] != "0x" {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	b, err := hex.DecodeString(v[2:])
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.BytesToAddress(b)
}

func decodeBigInt(valueJSON interface{}) *big.Int {
	v := toString(valueJSON)

	i := new(big.Int)
	i, ok := i.SetString(v, 10)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return i
}

func decodeInt(valueJSON interface{}) cadence.Int {
	return cadence.NewIntFromBig(decodeBigInt(valueJSON))
}

func decodeInt8(valueJSON interface{}) cadence.Int8 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt8(int8(i))
}

func decodeInt16(valueJSON interface{}) cadence.Int16 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt16(int16(i))
}

func decodeInt32(valueJSON interface{}) cadence.Int32 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt32(int32(i))
}

func decodeInt64(valueJSON interface{}) cadence.Int64 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt64(i)
}

func decodeInt128(valueJSON interface{}) cadence.Int128 {
	bigInt := decodeBigInt(valueJSON)
	value, err := cadence.NewInt128FromBig(bigInt)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func decodeInt256(valueJSON interface{}) cadence.Int256 {
	bigInt := decodeBigInt(valueJSON)
	value, err := cadence.NewInt256FromBig(bigInt)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func decodeUInt(valueJSON interface{}) cadence.UInt {
	bigInt := decodeBigInt(valueJSON)
	value, err := cadence.NewUIntFromBig(bigInt)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func decodeUInt8(valueJSON interface{}) cadence.UInt8 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt8(uint8(i))
}

func decodeUInt16(valueJSON interface{}) cadence.UInt16 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt16(uint16(i))
}

func decodeUInt32(valueJSON interface{}) cadence.UInt32 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt32(uint32(i))
}

func decodeUInt64(valueJSON interface{}) cadence.UInt64 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt64(i)
}

func decodeUInt128(valueJSON interface{}) cadence.UInt128 {
	bigInt := decodeBigInt(valueJSON)
	value, err := cadence.NewUInt128FromBig(bigInt)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func decodeUInt256(valueJSON interface{}) cadence.UInt256 {
	bigInt := decodeBigInt(valueJSON)
	value, err := cadence.NewUInt256FromBig(bigInt)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func decodeWord8(valueJSON interface{}) cadence.Word8 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord8(uint8(i))
}

func decodeWord16(valueJSON interface{}) cadence.Word16 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord16(uint16(i))
}

func decodeWord32(valueJSON interface{}) cadence.Word32 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord32(uint32(i))
}

func decodeWord64(valueJSON interface{}) cadence.Word64 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord64(i)
}

func decodeFix64(valueJSON interface{}) cadence.Fix64 {
	v, err := cadence.NewFix64(toString(valueJSON))
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return v
}

func decodeUFix64(valueJSON interface{}) cadence.UFix64 {
	v, err := cadence.NewUFix64(toString(valueJSON))
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return v
}

func decodeValues(valueJSON interface{}) []cadence.Value {
	v := toSlice(valueJSON)

	values := make([]cadence.Value, len(v))

	for i, val := range v {
		values[i] = decodeJSON(val)
	}

	return values
}

func decodeArray(valueJSON interface{}) cadence.Array {
	return cadence.NewArray(decodeValues(valueJSON))
}

func decodeDictionary(valueJSON interface{}) cadence.Dictionary {
	v := toSlice(valueJSON)

	pairs := make([]cadence.KeyValuePair, len(v))

	for i, val := range v {
		pairs[i] = decodeKeyValuePair(val)
	}

	return cadence.NewDictionary(pairs)
}

func decodeKeyValuePair(valueJSON interface{}) cadence.KeyValuePair {
	obj := toObject(valueJSON)

	key := obj.GetValue(keyKey)
	value := obj.GetValue(valueKey)

	return cadence.KeyValuePair{
		Key:   key,
		Value: value,
	}
}

type composite struct {
	location            common.Location
	qualifiedIdentifier string
	fieldValues         []cadence.Value
	fieldTypes          []cadence.Field
}

func decodeComposite(valueJSON interface{}) composite {
	obj := toObject(valueJSON)

	typeID := obj.GetString(idKey)
	location, qualifiedIdentifier, err := common.DecodeTypeID(typeID)

	if err != nil ||
		location == nil && sema.NativeCompositeTypes[typeID] == nil {

		// If the location is nil, and there is no native composite type with this ID, then its an invalid type.
		// Note: This is moved out from the common.DecodeTypeID() to avoid the circular dependency.
		panic(fmt.Errorf("%s. invalid type ID: `%s`", ErrInvalidJSONCadence, typeID))
	}

	fields := obj.GetSlice(fieldsKey)

	fieldValues := make([]cadence.Value, len(fields))
	fieldTypes := make([]cadence.Field, len(fields))

	for i, field := range fields {
		value, fieldType := decodeCompositeField(field)

		fieldValues[i] = value
		fieldTypes[i] = fieldType
	}

	return composite{
		location:            location,
		qualifiedIdentifier: qualifiedIdentifier,
		fieldValues:         fieldValues,
		fieldTypes:          fieldTypes,
	}
}

func decodeCompositeField(valueJSON interface{}) (cadence.Value, cadence.Field) {
	obj := toObject(valueJSON)

	name := obj.GetString(nameKey)
	value := obj.GetValue(valueKey)

	field := cadence.Field{
		Identifier: name,
		Type:       value.Type(),
	}

	return value, field
}

func decodeStruct(valueJSON interface{}) cadence.Struct {
	comp := decodeComposite(valueJSON)

	return cadence.NewStruct(comp.fieldValues).WithType(&cadence.StructType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func decodeResource(valueJSON interface{}) cadence.Resource {
	comp := decodeComposite(valueJSON)

	return cadence.NewResource(comp.fieldValues).WithType(&cadence.ResourceType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func decodeEvent(valueJSON interface{}) cadence.Event {
	comp := decodeComposite(valueJSON)

	return cadence.NewEvent(comp.fieldValues).WithType(&cadence.EventType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func decodeContract(valueJSON interface{}) cadence.Contract {
	comp := decodeComposite(valueJSON)

	return cadence.NewContract(comp.fieldValues).WithType(&cadence.ContractType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func decodeEnum(valueJSON interface{}) cadence.Enum {
	comp := decodeComposite(valueJSON)

	return cadence.NewEnum(comp.fieldValues).WithType(&cadence.EnumType{
		Location:            comp.location,
		QualifiedIdentifier: comp.qualifiedIdentifier,
		Fields:              comp.fieldTypes,
	})
}

func decodeLink(valueJSON interface{}) cadence.Link {
	obj := toObject(valueJSON)

	targetPath, ok := decodeJSON(obj.Get(targetPathKey)).(cadence.Path)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return cadence.NewLink(
		targetPath,
		obj.GetString(borrowTypeKey),
	)
}

func decodePath(valueJSON interface{}) cadence.Path {
	obj := toObject(valueJSON)

	return cadence.Path{
		Domain:     obj.GetString(domainKey),
		Identifier: obj.GetString(identifierKey),
	}
}

func decodeParamType(valueJSON interface{}) cadence.Parameter {
	obj := toObject(valueJSON)
	return cadence.Parameter{
		Label:      toString(obj.Get(labelKey)),
		Identifier: toString(obj.Get(idKey)),
		Type:       decodeType(obj.Get(typeKey)),
	}
}

func decodeParamTypes(params []interface{}) []cadence.Parameter {
	parameters := make([]cadence.Parameter, 0, len(params))

	for _, param := range params {
		parameters = append(parameters, decodeParamType(param))
	}

	return parameters
}

func decodeFieldTypes(fs []interface{}) []cadence.Field {
	fields := make([]cadence.Field, 0, len(fs))

	for _, field := range fs {
		fields = append(fields, decodeFieldType(field))
	}

	return fields
}

func decodeFieldType(valueJSON interface{}) cadence.Field {
	obj := toObject(valueJSON)
	return cadence.Field{
		Identifier: toString(obj.Get(idKey)),
		Type:       decodeType(obj.Get(typeKey)),
	}
}

func decodeFunctionType(returnValue, parametersValue, id interface{}) cadence.Type {
	parameters := decodeParamTypes(toSlice(parametersValue))
	returnType := decodeType(returnValue)

	return cadence.FunctionType{
		Parameters: parameters,
		ReturnType: returnType,
	}.WithID(toString(id))
}

func decodeNominalType(obj jsonObject, kind, typeID string, fs, initializers []interface{}) cadence.Type {
	fields := decodeFieldTypes(fs)
	inits := make([][]cadence.Parameter, 0, len(initializers))

	for _, params := range initializers {
		inits = append(inits, decodeParamTypes(toSlice(params)))
	}

	location, id, err := common.DecodeTypeID(typeID)
	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	switch kind {
	case "Struct":
		return &cadence.StructType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "Resource":
		return &cadence.ResourceType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "Event":
		return &cadence.EventType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializer:         inits[0],
		}
	case "Contract":
		return &cadence.ContractType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "StructInterface":
		return &cadence.StructInterfaceType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "ResourceInterface":
		return &cadence.ResourceInterfaceType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "ContractInterface":
		return &cadence.ContractInterfaceType{
			Location:            location,
			QualifiedIdentifier: id,
			Fields:              fields,
			Initializers:        inits,
		}
	case "Enum":
		return &cadence.EnumType{
			Location:            location,
			QualifiedIdentifier: id,
			RawType:             decodeType(obj.Get(typeKey)),
			Fields:              fields,
			Initializers:        inits,
		}
	}

	panic(ErrInvalidJSONCadence)
}

func decodeRestrictedType(
	typeValue interface{},
	restrictionsValue []interface{},
	typeIDValue string,
) cadence.Type {
	typ := decodeType(typeValue)
	restrictions := make([]cadence.Type, 0, len(restrictionsValue))
	for _, restriction := range restrictionsValue {
		restrictions = append(restrictions, decodeType(restriction))
	}

	return cadence.RestrictedType{
		Type:         typ,
		Restrictions: restrictions,
	}.WithID(typeIDValue)
}

func decodeType(valueJSON interface{}) cadence.Type {
	if valueJSON == "" {
		return nil
	}
	obj := toObject(valueJSON)
	kindValue := toString(obj.Get(kindKey))

	switch kindValue {
	case "Function":
		returnValue := obj.Get(returnKey)
		parametersValue := obj.Get(parametersKey)
		idValue := obj.Get(typeIDKey)
		return decodeFunctionType(returnValue, parametersValue, idValue)
	case "Restriction":
		restrictionsValue := obj.Get(restrictionsKey)
		typeIDValue := toString(obj.Get(typeIDKey))
		typeValue := obj.Get(typeKey)
		return decodeRestrictedType(typeValue, toSlice(restrictionsValue), typeIDValue)
	case "Optional":
		return cadence.OptionalType{
			Type: decodeType(obj.Get(typeKey)),
		}
	case "VariableSizedArray":
		return cadence.VariableSizedArrayType{
			ElementType: decodeType(obj.Get(typeKey)),
		}
	case "Capability":
		return cadence.CapabilityType{
			BorrowType: decodeType(obj.Get(typeKey)),
		}
	case "Dictionary":
		return cadence.DictionaryType{
			KeyType:     decodeType(obj.Get(keyKey)),
			ElementType: decodeType(obj.Get(valueKey)),
		}
	case "ConstantSizedArray":
		size := toUInt(obj.Get(sizeKey))
		return cadence.ConstantSizedArrayType{
			ElementType: decodeType(obj.Get(typeKey)),
			Size:        size,
		}
	case "Reference":
		auth := toBool(obj.Get(authorizedKey))
		return cadence.ReferenceType{
			Type:       decodeType(obj.Get(typeKey)),
			Authorized: auth,
		}
	case "Any":
		return cadence.AnyType{}
	case "AnyStruct":
		return cadence.AnyStructType{}
	case "AnyResource":
		return cadence.AnyResourceType{}
	case "Type":
		return cadence.MetaType{}
	case "Void":
		return cadence.VoidType{}
	case "Never":
		return cadence.NeverType{}
	case "Bool":
		return cadence.BoolType{}
	case "String":
		return cadence.StringType{}
	case "Character":
		return cadence.CharacterType{}
	case "Bytes":
		return cadence.BytesType{}
	case "Address":
		return cadence.AddressType{}
	case "Number":
		return cadence.NumberType{}
	case "SignedNumber":
		return cadence.SignedNumberType{}
	case "Integer":
		return cadence.IntegerType{}
	case "SignedInteger":
		return cadence.SignedIntegerType{}
	case "FixedPoint":
		return cadence.FixedPointType{}
	case "SignedFixedPoint":
		return cadence.SignedFixedPointType{}
	case "Int":
		return cadence.IntType{}
	case "Int8":
		return cadence.Int8Type{}
	case "Int16":
		return cadence.Int16Type{}
	case "Int32":
		return cadence.Int32Type{}
	case "Int64":
		return cadence.Int64Type{}
	case "Int128":
		return cadence.Int128Type{}
	case "Int256":
		return cadence.Int256Type{}
	case "UInt":
		return cadence.UIntType{}
	case "UInt8":
		return cadence.UInt8Type{}
	case "UInt16":
		return cadence.UInt16Type{}
	case "UInt32":
		return cadence.UInt32Type{}
	case "UInt64":
		return cadence.UInt64Type{}
	case "UInt128":
		return cadence.UInt128Type{}
	case "UInt256":
		return cadence.UInt256Type{}
	case "Word8":
		return cadence.Word8Type{}
	case "Word16":
		return cadence.Word16Type{}
	case "Word32":
		return cadence.Word32Type{}
	case "Word64":
		return cadence.Word64Type{}
	case "Fix64":
		return cadence.Fix64Type{}
	case "UFix64":
		return cadence.UFix64Type{}
	case "Path":
		return cadence.PathType{}
	case "CapabilityPath":
		return cadence.CapabilityPathType{}
	case "StoragePath":
		return cadence.StoragePathType{}
	case "PublicPath":
		return cadence.PublicPathType{}
	case "PrivatePath":
		return cadence.PrivatePathType{}
	case "AuthAccount":
		return cadence.AuthAccountType{}
	case "PublicAccount":
		return cadence.PublicAccountType{}
	case "AuthAccount.Keys":
		return cadence.AuthAccountKeysType{}
	case "PublicAccount.Keys":
		return cadence.PublicAccountKeysType{}
	case "AuthAccount.Contracts":
		return cadence.AuthAccountContractsType{}
	case "PublicAccount.Contracts":
		return cadence.PublicAccountContractsType{}
	case "DeployedContract":
		return cadence.DeployedContractType{}
	case "AccountKey":
		return cadence.AccountKeyType{}
	default:
		fieldsValue := obj.Get(fieldsKey)
		typeIDValue := toString(obj.Get(typeIDKey))
		initValue := obj.Get(initializersKey)
		return decodeNominalType(obj, kindValue, typeIDValue, toSlice(fieldsValue), toSlice(initValue))
	}
}

func decodeTypeValue(valueJSON interface{}) cadence.TypeValue {
	obj := toObject(valueJSON)

	return cadence.TypeValue{
		StaticType: decodeType(obj.Get(staticTypeKey)),
	}
}

func decodeCapability(valueJSON interface{}) cadence.Capability {
	obj := toObject(valueJSON)

	path, ok := decodeJSON(obj.Get(pathKey)).(cadence.Path)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.Capability{
		Path:       path,
		Address:    decodeAddress(obj.Get(addressKey)),
		BorrowType: decodeType(obj.Get(borrowTypeKey)),
	}
}

// JSON types

type jsonObject map[string]interface{}

func (obj jsonObject) Get(key string) interface{} {
	v, hasKey := obj[key]
	if !hasKey {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func (obj jsonObject) GetBool(key string) bool {
	v := obj.Get(key)
	return toBool(v)
}

func (obj jsonObject) GetString(key string) string {
	v := obj.Get(key)
	return toString(v)
}

func (obj jsonObject) GetSlice(key string) []interface{} {
	v := obj.Get(key)
	return toSlice(v)
}

func (obj jsonObject) GetValue(key string) cadence.Value {
	v := obj.Get(key)
	return decodeJSON(v)
}

// JSON conversion helpers

func toBool(valueJSON interface{}) bool {
	v, isBool := valueJSON.(bool)
	if !isBool {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toUInt(valueJSON interface{}) uint {
	v, isNum := valueJSON.(float64)
	if !isNum {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return uint(v)
}

func toString(valueJSON interface{}) string {
	v, isString := valueJSON.(string)
	if !isString {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toSlice(valueJSON interface{}) []interface{} {
	v, isSlice := valueJSON.([]interface{})
	if !isSlice {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toObject(valueJSON interface{}) jsonObject {
	v, isMap := valueJSON.(map[string]interface{})
	if !isMap {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}
