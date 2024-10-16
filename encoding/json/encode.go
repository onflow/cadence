/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	_ "unsafe"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
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

type jsonValue any

type jsonValueObject struct {
	Value jsonValue `json:"value"`
	Type  string    `json:"type"`
}

type jsonEmptyValueObject struct {
	Type string `json:"type"`
}

type jsonDictionaryItem struct {
	Key   jsonValue `json:"key"`
	Value jsonValue `json:"value"`
}

type jsonInclusiveRangeValue struct {
	Start jsonValue `json:"start"`
	End   jsonValue `json:"end"`
	Step  jsonValue `json:"step"`
}

type jsonCompositeValue struct {
	ID     string               `json:"id"`
	Fields []jsonCompositeField `json:"fields"`
}

type jsonCompositeField struct {
	Value jsonValue `json:"value"`
	Name  string    `json:"name"`
}

type jsonPathValue struct {
	Domain     string `json:"domain"`
	Identifier string `json:"identifier"`
}

type jsonFieldType struct {
	Type jsonValue `json:"type"`
	Id   string    `json:"id"`
}

type jsonNominalType struct {
	Type         jsonValue             `json:"type"`
	Kind         string                `json:"kind"`
	TypeID       string                `json:"typeID"`
	Fields       []jsonFieldType       `json:"fields"`
	Initializers [][]jsonParameterType `json:"initializers"`
}

type jsonSimpleType struct {
	Kind string `json:"kind"`
}

type jsonUnaryType struct {
	Type jsonValue `json:"type"`
	Kind string    `json:"kind"`
}

type jsonConstantSizedArrayType struct {
	Type jsonValue `json:"type"`
	Kind string    `json:"kind"`
	Size uint      `json:"size"`
}

type jsonDictionaryType struct {
	KeyType   jsonValue `json:"key"`
	ValueType jsonValue `json:"value"`
	Kind      string    `json:"kind"`
}

type jsonAuthorization struct {
	Kind         string            `json:"kind"`
	Entitlements []jsonNominalType `json:"entitlements"`
}

type jsonReferenceType struct {
	Type          jsonValue         `json:"type"`
	Kind          string            `json:"kind"`
	Authorization jsonAuthorization `json:"authorization"`
}

type jsonIntersectionType struct {
	Kind   string      `json:"kind"`
	TypeID string      `json:"typeID"`
	Types  []jsonValue `json:"types"`
}

type jsonTypeParameter struct {
	Name      string    `json:"name"`
	TypeBound jsonValue `json:"typeBound"`
}

type jsonParameterType struct {
	Type  jsonValue `json:"type"`
	Label string    `json:"label"`
	Id    string    `json:"id"`
}

type jsonFunctionType struct {
	Kind           string              `json:"kind"`
	TypeID         string              `json:"typeID"`
	TypeParameters []jsonTypeParameter `json:"typeParameters"`
	Parameters     []jsonParameterType `json:"parameters"`
	Return         jsonValue           `json:"return"`
	Purity         string              `json:"purity"`
}

type jsonTypeValue struct {
	StaticType jsonValue `json:"staticType"`
}

type jsonCapabilityValue struct {
	BorrowType     jsonValue `json:"borrowType"`
	Address        string    `json:"address"`
	ID             string    `json:"id"`
	DeprecatedPath jsonValue `json:"path,omitempty"` // Deprecated
}

type jsonFunctionValue struct {
	FunctionType jsonValue `json:"functionType"`
}

type jsonInclusiveRangeType struct {
	ElementType jsonValue `json:"element"`
	Kind        string    `json:"kind"`
}

const (
	voidTypeStr           = "Void"
	optionalTypeStr       = "Optional"
	boolTypeStr           = "Bool"
	characterTypeStr      = "Character"
	stringTypeStr         = "String"
	addressTypeStr        = "Address"
	intTypeStr            = "Int"
	int8TypeStr           = "Int8"
	int16TypeStr          = "Int16"
	int32TypeStr          = "Int32"
	int64TypeStr          = "Int64"
	int128TypeStr         = "Int128"
	int256TypeStr         = "Int256"
	uintTypeStr           = "UInt"
	uint8TypeStr          = "UInt8"
	uint16TypeStr         = "UInt16"
	uint32TypeStr         = "UInt32"
	uint64TypeStr         = "UInt64"
	uint128TypeStr        = "UInt128"
	uint256TypeStr        = "UInt256"
	word8TypeStr          = "Word8"
	word16TypeStr         = "Word16"
	word32TypeStr         = "Word32"
	word64TypeStr         = "Word64"
	word128TypeStr        = "Word128"
	word256TypeStr        = "Word256"
	fix64TypeStr          = "Fix64"
	ufix64TypeStr         = "UFix64"
	arrayTypeStr          = "Array"
	dictionaryTypeStr     = "Dictionary"
	structTypeStr         = "Struct"
	resourceTypeStr       = "Resource"
	attachmentTypeStr     = "Attachment"
	eventTypeStr          = "Event"
	contractTypeStr       = "Contract"
	pathTypeStr           = "Path"
	typeTypeStr           = "Type"
	capabilityTypeStr     = "Capability"
	enumTypeStr           = "Enum"
	functionTypeStr       = "Function"
	inclusiveRangeTypeStr = "InclusiveRange"
)

// Prepare traverses the object graph of the provided value and constructs
// a struct representation that can be marshalled to JSON.
func Prepare(v cadence.Value) jsonValue {
	switch v := v.(type) {
	case cadence.Void:
		return prepareVoid()
	case cadence.Optional:
		return prepareOptional(v)
	case cadence.Bool:
		return prepareBool(v)
	case cadence.Character:
		return prepareCharacter(v)
	case cadence.String:
		return prepareString(v)
	case cadence.Address:
		return prepareAddress(v)
	case cadence.Int:
		return prepareInt(v)
	case cadence.Int8:
		return prepareInt8(v)
	case cadence.Int16:
		return prepareInt16(v)
	case cadence.Int32:
		return prepareInt32(v)
	case cadence.Int64:
		return prepareInt64(v)
	case cadence.Int128:
		return prepareInt128(v)
	case cadence.Int256:
		return prepareInt256(v)
	case cadence.UInt:
		return prepareUInt(v)
	case cadence.UInt8:
		return prepareUInt8(v)
	case cadence.UInt16:
		return prepareUInt16(v)
	case cadence.UInt32:
		return prepareUInt32(v)
	case cadence.UInt64:
		return prepareUInt64(v)
	case cadence.UInt128:
		return prepareUInt128(v)
	case cadence.UInt256:
		return prepareUInt256(v)
	case cadence.Word8:
		return prepareWord8(v)
	case cadence.Word16:
		return prepareWord16(v)
	case cadence.Word32:
		return prepareWord32(v)
	case cadence.Word64:
		return prepareWord64(v)
	case cadence.Word128:
		return prepareWord128(v)
	case cadence.Word256:
		return prepareWord256(v)
	case cadence.Fix64:
		return prepareFix64(v)
	case cadence.UFix64:
		return prepareUFix64(v)
	case cadence.Array:
		return prepareArray(v)
	case cadence.Dictionary:
		return prepareDictionary(v)
	case *cadence.InclusiveRange:
		return prepareInclusiveRange(v)
	case cadence.Struct:
		return prepareStruct(v)
	case cadence.Resource:
		return prepareResource(v)
	case cadence.Event:
		return prepareEvent(v)
	case cadence.Contract:
		return prepareContract(v)
	case cadence.Path:
		return preparePath(v)
	case cadence.TypeValue:
		return prepareTypeValue(v)
	case cadence.Capability:
		return prepareCapability(v)
	case cadence.Enum:
		return prepareEnum(v)
	case cadence.Attachment:
		return prepareAttachment(v)
	case cadence.Function:
		return prepareFunction(v)
	case nil:
		return nil
	default:
		panic(fmt.Errorf("unsupported value: %T, %v", v, v))
	}
}

func prepareVoid() jsonValue {
	return jsonEmptyValueObject{Type: voidTypeStr}
}

func prepareOptional(v cadence.Optional) jsonValue {
	var value any

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

func prepareCharacter(v cadence.Character) jsonValue {
	return jsonValueObject{
		Type:  characterTypeStr,
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

func prepareWord128(v cadence.Word128) jsonValue {
	return jsonValueObject{
		Type:  word128TypeStr,
		Value: encodeBig(v.Big()),
	}
}

func prepareWord256(v cadence.Word256) jsonValue {
	return jsonValueObject{
		Type:  word256TypeStr,
		Value: encodeBig(v.Big()),
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

func prepareInclusiveRange(v *cadence.InclusiveRange) jsonValue {
	return jsonValueObject{
		Type: inclusiveRangeTypeStr,
		Value: jsonInclusiveRangeValue{
			Start: Prepare(v.Start),
			End:   Prepare(v.End),
			Step:  Prepare(v.Step),
		},
	}
}

//go:linkname getCompositeFieldValues github.com/onflow/cadence.getCompositeFieldValues
func getCompositeFieldValues(cadence.Composite) []cadence.Value

//go:linkname getCompositeTypeFields github.com/onflow/cadence.getCompositeTypeFields
func getCompositeTypeFields(cadence.CompositeType) []cadence.Field

//go:linkname getInterfaceTypeFields github.com/onflow/cadence.getInterfaceTypeFields
func getInterfaceTypeFields(cadence.InterfaceType) []cadence.Field

func prepareStruct(v cadence.Struct) jsonValue {
	return prepareComposite(
		structTypeStr,
		v.StructType.ID(),
		getCompositeTypeFields(v.StructType),
		getCompositeFieldValues(v),
	)
}

func prepareResource(v cadence.Resource) jsonValue {
	return prepareComposite(
		resourceTypeStr,
		v.ResourceType.ID(),
		getCompositeTypeFields(v.ResourceType),
		getCompositeFieldValues(v),
	)
}

func prepareEvent(v cadence.Event) jsonValue {
	return prepareComposite(
		eventTypeStr,
		v.EventType.ID(),
		getCompositeTypeFields(v.EventType),
		getCompositeFieldValues(v),
	)
}

func prepareContract(v cadence.Contract) jsonValue {
	return prepareComposite(
		contractTypeStr,
		v.ContractType.ID(),
		getCompositeTypeFields(v.ContractType),
		getCompositeFieldValues(v),
	)
}

func prepareEnum(v cadence.Enum) jsonValue {
	return prepareComposite(
		enumTypeStr,
		v.EnumType.ID(),
		getCompositeTypeFields(v.EnumType),
		getCompositeFieldValues(v),
	)
}

func prepareAttachment(v cadence.Attachment) jsonValue {
	return prepareComposite(
		attachmentTypeStr,
		v.AttachmentType.ID(),
		getCompositeTypeFields(v.AttachmentType),
		getCompositeFieldValues(v),
	)
}

func prepareComposite(kind, id string, fieldTypes []cadence.Field, fields []cadence.Value) jsonValue {
	// Ensure there are _at least_ as many field values as field types.
	// There might be more field values in the case of attachments.
	if len(fields) < len(fieldTypes) {
		panic(fmt.Errorf(
			"%s field count (%d) does not match declared type (%d)",
			kind,
			len(fields),
			len(fieldTypes),
		))
	}

	compositeFields := make([]jsonCompositeField, len(fields))

	for i, value := range fields {
		var name string
		// Provide the field name, if the field type is available.
		// In the case of attachments, they are provided as field values,
		// but there is no corresponding field type.
		if i < len(fieldTypes) {
			fieldType := fieldTypes[i]
			name = fieldType.Identifier
		}

		compositeFields[i] = jsonCompositeField{
			Name:  name,
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

func prepareAuthorization(auth cadence.Authorization) jsonAuthorization {
	var kind string
	var entitlements []jsonNominalType
	switch auth := auth.(type) {
	case cadence.Unauthorized:
		kind = "Unauthorized"
	case cadence.EntitlementMapAuthorization:
		kind = "EntitlementMapAuthorization"
		entitlements = []jsonNominalType{
			{
				Kind:   "EntitlementMap",
				TypeID: string(auth.TypeID),
			},
		}
	case *cadence.EntitlementSetAuthorization:
		for _, entitlement := range auth.Entitlements {
			entitlements = append(entitlements, jsonNominalType{
				Kind:   "Entitlement",
				TypeID: string(entitlement),
			})
		}
		switch auth.Kind {
		case cadence.Conjunction:
			kind = "EntitlementConjunctionSet"
		case cadence.Disjunction:
			kind = "EntitlementDisjunctionSet"
		}
	}
	return jsonAuthorization{
		Kind:         kind,
		Entitlements: entitlements,
	}
}

func preparePath(x cadence.Path) jsonValue {
	return jsonValueObject{
		Type: pathTypeStr,
		Value: jsonPathValue{
			Domain:     x.Domain.Identifier(),
			Identifier: x.Identifier,
		},
	}
}

func prepareTypeParameter(typeParameter cadence.TypeParameter, results TypePreparationResults) jsonTypeParameter {
	typeBound := typeParameter.TypeBound
	var preparedTypeBound jsonValue
	if typeBound != nil {
		preparedTypeBound = PrepareType(typeBound, results)
	}
	return jsonTypeParameter{
		Name:      typeParameter.Name,
		TypeBound: preparedTypeBound,
	}
}

func prepareParameter(parameterType cadence.Parameter, results TypePreparationResults) jsonParameterType {
	return jsonParameterType{
		Label: parameterType.Label,
		Id:    parameterType.Identifier,
		Type:  PrepareType(parameterType.Type, results),
	}
}

func prepareFieldType(fieldType cadence.Field, results TypePreparationResults) jsonFieldType {
	return jsonFieldType{
		Id:   fieldType.Identifier,
		Type: PrepareType(fieldType.Type, results),
	}
}

func prepareFields(fieldTypes []cadence.Field, results TypePreparationResults) []jsonFieldType {
	fields := make([]jsonFieldType, len(fieldTypes))
	for i, fieldType := range fieldTypes {
		fields[i] = prepareFieldType(fieldType, results)
	}
	return fields
}

func prepareTypeParameters(typeParameters []cadence.TypeParameter, results TypePreparationResults) []jsonTypeParameter {
	result := make([]jsonTypeParameter, len(typeParameters))
	for i, typeParameter := range typeParameters {
		result[i] = prepareTypeParameter(typeParameter, results)
	}
	return result
}

func prepareParameters(parameters []cadence.Parameter, results TypePreparationResults) []jsonParameterType {
	result := make([]jsonParameterType, len(parameters))
	for i, param := range parameters {
		result[i] = prepareParameter(param, results)
	}
	return result
}

func prepareInitializers(initializers [][]cadence.Parameter, results TypePreparationResults) [][]jsonParameterType {
	result := make([][]jsonParameterType, len(initializers))
	for i, params := range initializers {
		result[i] = prepareParameters(params, results)
	}
	return result
}

func PrepareType(typ cadence.Type, results TypePreparationResults) jsonValue {

	var supportedRecursiveType bool
	switch typ.(type) {
	case cadence.CompositeType, cadence.InterfaceType:
		supportedRecursiveType = true
	}

	if _, ok := results[typ]; ok {
		if !supportedRecursiveType {
			panic(fmt.Errorf("failed to prepare type: unsupported recursive type: %T", typ))
		}

		return typ.ID()
	}

	if supportedRecursiveType {
		results[typ] = struct{}{}
	}

	switch typ := typ.(type) {
	case cadence.BytesType:
		return jsonSimpleType{
			Kind: typ.ID(),
		}
	case *cadence.OptionalType:
		return jsonUnaryType{
			Kind: "Optional",
			Type: PrepareType(typ.Type, results),
		}
	case *cadence.VariableSizedArrayType:
		return jsonUnaryType{
			Kind: "VariableSizedArray",
			Type: PrepareType(typ.ElementType, results),
		}
	case *cadence.ConstantSizedArrayType:
		return jsonConstantSizedArrayType{
			Kind: "ConstantSizedArray",
			Type: PrepareType(typ.ElementType, results),
			Size: typ.Size,
		}
	case *cadence.DictionaryType:
		return jsonDictionaryType{
			Kind:      "Dictionary",
			KeyType:   PrepareType(typ.KeyType, results),
			ValueType: PrepareType(typ.ElementType, results),
		}
	case *cadence.InclusiveRangeType:
		return jsonInclusiveRangeType{
			Kind:        "InclusiveRange",
			ElementType: PrepareType(typ.ElementType, results),
		}
	case *cadence.StructType:
		return jsonNominalType{
			Kind:         "Struct",
			Type:         "",
			TypeID:       string(common.NewTypeIDFromQualifiedName(nil, typ.Location, typ.QualifiedIdentifier)),
			Fields:       prepareFields(getCompositeTypeFields(typ), results),
			Initializers: prepareInitializers(typ.Initializers, results),
		}
	case *cadence.ResourceType:
		return jsonNominalType{
			Kind:         "Resource",
			Type:         "",
			TypeID:       string(common.NewTypeIDFromQualifiedName(nil, typ.Location, typ.QualifiedIdentifier)),
			Fields:       prepareFields(getCompositeTypeFields(typ), results),
			Initializers: prepareInitializers(typ.Initializers, results),
		}
	case *cadence.EventType:
		return jsonNominalType{
			Kind:         "Event",
			Type:         "",
			TypeID:       string(common.NewTypeIDFromQualifiedName(nil, typ.Location, typ.QualifiedIdentifier)),
			Fields:       prepareFields(getCompositeTypeFields(typ), results),
			Initializers: [][]jsonParameterType{prepareParameters(typ.Initializer, results)},
		}
	case *cadence.ContractType:
		return jsonNominalType{
			Kind:         "Contract",
			Type:         "",
			TypeID:       string(common.NewTypeIDFromQualifiedName(nil, typ.Location, typ.QualifiedIdentifier)),
			Fields:       prepareFields(getCompositeTypeFields(typ), results),
			Initializers: prepareInitializers(typ.Initializers, results),
		}
	case *cadence.StructInterfaceType:
		return jsonNominalType{
			Kind:         "StructInterface",
			Type:         "",
			TypeID:       string(common.NewTypeIDFromQualifiedName(nil, typ.Location, typ.QualifiedIdentifier)),
			Fields:       prepareFields(getInterfaceTypeFields(typ), results),
			Initializers: prepareInitializers(typ.Initializers, results),
		}
	case *cadence.ResourceInterfaceType:
		return jsonNominalType{
			Kind:         "ResourceInterface",
			Type:         "",
			TypeID:       string(common.NewTypeIDFromQualifiedName(nil, typ.Location, typ.QualifiedIdentifier)),
			Fields:       prepareFields(getInterfaceTypeFields(typ), results),
			Initializers: prepareInitializers(typ.Initializers, results),
		}
	case *cadence.ContractInterfaceType:
		return jsonNominalType{
			Kind:         "ContractInterface",
			Type:         "",
			TypeID:       string(common.NewTypeIDFromQualifiedName(nil, typ.Location, typ.QualifiedIdentifier)),
			Fields:       prepareFields(getInterfaceTypeFields(typ), results),
			Initializers: prepareInitializers(typ.Initializers, results),
		}
	case *cadence.FunctionType:
		typeJson := jsonFunctionType{
			Kind:           "Function",
			TypeID:         typ.ID(),
			TypeParameters: prepareTypeParameters(typ.TypeParameters, results),
			Parameters:     prepareParameters(typ.Parameters, results),
			Return:         PrepareType(typ.ReturnType, results),
		}
		if typ.Purity == cadence.FunctionPurityView {
			typeJson.Purity = "view"
		}
		return typeJson
	case *cadence.ReferenceType:
		return jsonReferenceType{
			Kind:          "Reference",
			Authorization: prepareAuthorization(typ.Authorization),
			Type:          PrepareType(typ.Type, results),
		}
	case *cadence.IntersectionType:
		types := make([]jsonValue, len(typ.Types))
		for i, typ := range typ.Types {
			types[i] = PrepareType(typ, results)
		}
		return jsonIntersectionType{
			Kind:   "Intersection",
			Types:  types,
			TypeID: typ.ID(),
		}
	case *cadence.CapabilityType:
		return jsonUnaryType{
			Kind: "Capability",
			Type: PrepareType(typ.BorrowType, results),
		}
	case *cadence.EnumType:
		return jsonNominalType{
			Kind:         "Enum",
			TypeID:       string(common.NewTypeIDFromQualifiedName(nil, typ.Location, typ.QualifiedIdentifier)),
			Fields:       prepareFields(getCompositeTypeFields(typ), results),
			Initializers: prepareInitializers(typ.Initializers, results),
			Type:         PrepareType(typ.RawType, results),
		}
	case cadence.PrimitiveType:
		return jsonSimpleType{
			Kind: typ.ID(),
		}
	case nil:
		return ""
	}

	panic(fmt.Errorf("unsupported type: %T, %s", typ, typ))
}

type TypePreparationResults map[cadence.Type]struct{}

func prepareTypeValue(typeValue cadence.TypeValue) jsonValue {
	return jsonValueObject{
		Type: typeTypeStr,
		Value: jsonTypeValue{
			StaticType: PrepareType(typeValue.StaticType, TypePreparationResults{}),
		},
	}
}

func prepareCapability(capability cadence.Capability) jsonValue {
	capabilityJson := jsonCapabilityValue{
		ID:         encodeUInt(uint64(capability.ID)),
		Address:    encodeBytes(capability.Address.Bytes()),
		BorrowType: PrepareType(capability.BorrowType, TypePreparationResults{}),
	}

	if capability.DeprecatedPath != nil {
		capabilityJson.DeprecatedPath = preparePath(*capability.DeprecatedPath)
	}

	return jsonValueObject{
		Type:  capabilityTypeStr,
		Value: capabilityJson,
	}
}

func prepareFunction(function cadence.Function) jsonValue {
	return jsonValueObject{
		Type: functionTypeStr,
		Value: jsonFunctionValue{
			FunctionType: PrepareType(function.FunctionType, TypePreparationResults{}),
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
			builder.WriteByte('-')
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
