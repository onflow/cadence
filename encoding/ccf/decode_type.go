/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package ccf

import (
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

type cadenceTypeID string

type decodeTypeFn func(types *cadenceTypeByCCFTypeID) (cadence.Type, error)

// decodeInlineType decodes inline-type as
// language=CDDL
// inline-type =
//
//	simple-type
//	/ optional-type
//	/ varsized-array-type
//	/ constsized-array-type
//	/ dict-type
//	/ reference-type
//	/ restricted-type
//	/ capability-type
//	/ type-ref
//
// All exported Cadence types needs to be handled in this function,
// including abstract and interface types.
func (d *Decoder) decodeInlineType(types *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	tagNum, err := d.dec.DecodeTagNumber()
	if err != nil {
		return nil, err
	}

	switch tagNum {
	case CBORTagSimpleType:
		return d.decodeSimpleTypeID()

	case CBORTagOptionalType:
		return d.decodeOptionalType(types, d.decodeInlineType)

	case CBORTagVarsizedArrayType:
		return d.decodeVarSizedArrayType(types, d.decodeInlineType)

	case CBORTagConstsizedArrayType:
		return d.decodeConstantSizedArrayType(types, d.decodeInlineType)

	case CBORTagDictType:
		return d.decodeDictType(types, d.decodeInlineType)

	case CBORTagReferenceType:
		return d.decodeReferenceType(types, d.decodeInlineType)

	case CBORTagRestrictedType:
		return d.decodeRestrictedType(types, d.decodeNullableInlineType, d.decodeInlineType)

	case CBORTagCapabilityType:
		return d.decodeCapabilityType(types, d.decodeNullableInlineType)

	case CBORTagTypeRef:
		return d.decodeTypeRef(types)

	default:
		return nil, fmt.Errorf("unsupported encoded inline type with CBOR tag number %d", tagNum)
	}
}

// decodeNullableInlineType decodes encoded inline-type or nil.
func (d *Decoder) decodeNullableInlineType(types *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	cborType, err := d.dec.NextType()
	if err != nil {
		return nil, err
	}
	if cborType == cbor.NilType {
		err = d.dec.DecodeNil()
		return nil, err
	}
	return d.decodeInlineType(types)
}

// decodeSimpleTypeID decodes encoded simple-type-id.
// See CCF specification for complete list of simple-type-id.
func (d *Decoder) decodeSimpleTypeID() (cadence.Type, error) {
	simpleTypeID, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}

	switch simpleTypeID {
	case TypeBool:
		return cadence.TheBoolType, nil

	case TypeString:
		return cadence.TheStringType, nil

	case TypeCharacter:
		return cadence.TheCharacterType, nil

	case TypeAddress:
		return cadence.TheAddressType, nil

	case TypeInt:
		return cadence.TheIntType, nil

	case TypeInt8:
		return cadence.TheInt8Type, nil

	case TypeInt16:
		return cadence.TheInt16Type, nil

	case TypeInt32:
		return cadence.TheInt32Type, nil

	case TypeInt64:
		return cadence.TheInt64Type, nil

	case TypeInt128:
		return cadence.TheInt128Type, nil

	case TypeInt256:
		return cadence.TheInt256Type, nil

	case TypeUInt:
		return cadence.TheUIntType, nil

	case TypeUInt8:
		return cadence.TheUInt8Type, nil

	case TypeUInt16:
		return cadence.TheUInt16Type, nil

	case TypeUInt32:
		return cadence.TheUInt32Type, nil

	case TypeUInt64:
		return cadence.TheUInt64Type, nil

	case TypeUInt128:
		return cadence.TheUInt128Type, nil

	case TypeUInt256:
		return cadence.TheUInt256Type, nil

	case TypeWord8:
		return cadence.TheWord8Type, nil

	case TypeWord16:
		return cadence.TheWord16Type, nil

	case TypeWord32:
		return cadence.TheWord32Type, nil

	case TypeWord64:
		return cadence.TheWord64Type, nil

	case TypeFix64:
		return cadence.TheFix64Type, nil

	case TypeUFix64:
		return cadence.TheUFix64Type, nil

	case TypePath:
		return cadence.ThePathType, nil

	case TypeCapabilityPath:
		return cadence.TheCapabilityPathType, nil

	case TypeStoragePath:
		return cadence.TheStoragePathType, nil

	case TypePublicPath:
		return cadence.ThePublicPathType, nil

	case TypePrivatePath:
		return cadence.ThePrivatePathType, nil

	case TypeAuthAccount:
		return cadence.TheAuthAccountType, nil

	case TypePublicAccount:
		return cadence.ThePublicAccountType, nil

	case TypeAuthAccountKeys:
		return cadence.TheAuthAccountKeysType, nil

	case TypePublicAccountKeys:
		return cadence.ThePublicAccountKeysType, nil

	case TypeAuthAccountContracts:
		return cadence.TheAuthAccountContractsType, nil

	case TypePublicAccountContracts:
		return cadence.ThePublicAccountContractsType, nil

	case TypeDeployedContract:
		return cadence.TheDeployedContractType, nil

	case TypeAccountKey:
		return cadence.TheAccountKeyType, nil

	case TypeBlock:
		return cadence.TheBlockType, nil

	case TypeAny:
		return cadence.TheAnyType, nil

	case TypeAnyStruct:
		return cadence.TheAnyStructType, nil

	case TypeAnyResource:
		return cadence.TheAnyResourceType, nil

	case TypeMetaType:
		return cadence.TheMetaType, nil

	case TypeNever:
		return cadence.TheNeverType, nil

	case TypeNumber:
		return cadence.TheNumberType, nil

	case TypeSignedNumber:
		return cadence.TheSignedNumberType, nil

	case TypeInteger:
		return cadence.TheIntegerType, nil

	case TypeSignedInteger:
		return cadence.TheSignedIntegerType, nil

	case TypeFixedPoint:
		return cadence.TheFixedPointType, nil

	case TypeSignedFixedPoint:
		return cadence.TheSignedFixedPointType, nil

	case TypeBytes:
		return cadence.TheBytesType, nil

	case TypeVoid:
		return cadence.TheVoidType, nil

	default:
		return nil, fmt.Errorf("unsupported encoded simple type ID %d", simpleTypeID)
	}
}

// decodeOptionalType decodes optional-type or optional-type-value as
// language=CDDL
// optional-type =
//
//	; cbor-tag-optional-type
//	#6.138(inline-type)
//
// optional-type-value =
//
//	; cbor-tag-optional-type-value
//	#6.186(type-value)
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeOptionalType(
	types *cadenceTypeByCCFTypeID,
	decodeTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// Decode inline-type or type-value.
	elementType, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}
	if elementType == nil {
		return nil, errors.New("unexpected nil type as optional inner type")
	}
	return cadence.NewMeteredOptionalType(d.gauge, elementType), nil
}

// decodeVarSizedArrayType decodes varsized-array-type or varsized-array-type-value as
// language=CDDL
// varsized-array-type =
//
//	; cbor-tag-varsized-array-type
//	#6.139(inline-type)
//
// varsized-array-type-value =
//
//	; cbor-tag-varsized-array-type-value
//	#6.187(type-value)
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeVarSizedArrayType(
	types *cadenceTypeByCCFTypeID,
	decodeTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// Decode inline-type or type-value.
	elementType, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}
	if elementType == nil {
		return nil, errors.New("unexpected nil type as variable sized array element type")
	}
	return cadence.NewMeteredVariableSizedArrayType(d.gauge, elementType), nil
}

// decodeConstantSizedArrayType decodes constsized-array-type or constsized-array-type-value as
// language=CDDL
// constsized-array-type =
//
//	; cbor-tag-constsized-array-type
//	#6.140([
//	    array-size: uint,
//	    element-type: inline-type
//	])
//
// constsized-array-type-value =
//
//	; cbor-tag-constsized-array-type-value
//	#6.188([
//	    array-size: uint,
//	    element-type: type-value
//	])
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeConstantSizedArrayType(
	types *cadenceTypeByCCFTypeID,
	decodeTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// Decode array head of length 2.
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return nil, err
	}

	// element 0: array-size
	size, err := d.dec.DecodeUint64()
	if err != nil {
		return nil, err
	}

	// element 1: element-type (inline-type or type-value)
	elementType, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}

	if elementType == nil {
		return nil, errors.New("unexpected nil type as constant sized array element type")
	}

	return cadence.NewMeteredConstantSizedArrayType(d.gauge, uint(size), elementType), nil
}

// decodeDictType decodes dict-type or dict-type-value as
// language=CDDL
// dict-type =
//
//	; cbor-tag-dict-type
//	#6.141([
//	    key-type: inline-type,
//	    element-type: inline-type
//	])
//
// dict-type-value =
//
//	; cbor-tag-dict-type-value
//	#6.189([
//	    key-type: type-value,
//	    element-type: type-value
//	])
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeDictType(
	types *cadenceTypeByCCFTypeID,
	decodeTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// Decode array head of length 2.
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return nil, err
	}

	// element 0: key type (inline-type or type-value)
	keyType, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}

	if keyType == nil {
		return nil, errors.New("unexpected nil type as dictionary key type")
	}

	// element 1: element type (inline-type or type-value)
	elementType, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}

	if elementType == nil {
		return nil, errors.New("unexpected nil type as dictionary element type")
	}

	return cadence.NewMeteredDictionaryType(d.gauge, keyType, elementType), nil
}

// decodeCapabilityType decodes capability-type or capability-type-value as
// language=CDDL
// capability-type =
//
//	; cbor-tag-capability-type
//	; use an array as an extension point
//	#6.144([
//	    ; borrow-type
//	    inline-type / nil
//	])
//
// capability-type-value =
//
//	; cbor-tag-capability-type-value
//	; use an array as an extension point
//	#6.192([
//	  ; borrow-type
//	  type-value / nil
//	])
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeCapabilityType(
	types *cadenceTypeByCCFTypeID,
	decodeTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// Decode array head of length 1
	err := decodeCBORArrayWithKnownSize(d.dec, 1)
	if err != nil {
		return nil, err
	}

	// element 0: borrow-type (inline-type or type-value)
	borrowType, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}

	return cadence.NewMeteredCapabilityType(d.gauge, borrowType), nil
}

// decodeReferenceType decodes reference-type or reference-type-value as
// language=CDDL
// reference-type =
//
//	; cbor-tag-reference-type
//	#6.142([
//	  authorized: bool,
//	  type: inline-type,
//	])
//
// reference-type-value =
//
//	; cbor-tag-reference-type-value
//	#6.190([
//	  authorized: bool,
//	  type: type-value,
//	])
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeReferenceType(
	types *cadenceTypeByCCFTypeID,
	decodeTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// Decode array head of length 2
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return nil, err
	}

	// element 0: authorized
	// TODO: implement in later PR
	// authorized, err := d.dec.DecodeBool()
	if err != nil {
		return nil, err
	}

	// element 0: type (inline-type or type-value)
	elementType, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}

	if elementType == nil {
		return nil, errors.New("unexpected nil type as reference type")
	}
	// TODO: implement in later PR
	// return cadence.NewMeteredReferenceType(d.gauge, authorized, elementType), nil
	return nil, nil
}

// decodeRestrictedType decodes restricted-type or restricted-type-value as
// language=CDDL
// restricted-type =
//
//	; cbor-tag-restricted-type
//	#6.143([
//	  type: inline-type / nil,
//	  restrictions: [* inline-type]
//	])
//
// restricted-type-value =
//
//	; cbor-tag-restricted-type-value
//	#6.191([
//	  type: type-value / nil,
//	  restrictions: [* type-value]
//	])
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeRestrictedType(
	types *cadenceTypeByCCFTypeID,
	decodeTypeFn decodeTypeFn,
	decodeRestrictionTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// Decode array of length 2.
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return nil, err
	}

	// element 0: type
	typ, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}

	// element 1: restrictions
	restrictionCount, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	restrictionTypeIDs := make(map[string]struct{}, restrictionCount)
	var previousRestrictedTypeID string

	restrictions := make([]cadence.Type, restrictionCount)
	for i := 0; i < int(restrictionCount); i++ {
		// Decode restriction.
		restrictedType, err := decodeRestrictionTypeFn(types)
		if err != nil {
			return nil, err
		}

		if restrictedType == nil {
			return nil, errors.New("unexpected nil type as restriction type")
		}

		restrictedTypeID := restrictedType.ID()

		// "Valid CCF Encoding Requirements" in CCF specs:
		//
		//   "Elements MUST be unique in restricted-type or restricted-type-value."
		if _, ok := restrictionTypeIDs[restrictedTypeID]; ok {
			return nil, fmt.Errorf("found duplicate restricted type %s", restrictedTypeID)
		}

		// "Deterministic CCF Encoding Requirements" in CCF specs:
		//
		//   "restricted-type.restrictions MUST be sorted by restriction's cadence-type-id"
		//   "restricted-type-value.restrictions MUST be sorted by restriction's cadence-type-id."
		if !stringsAreSortedBytewise(previousRestrictedTypeID, restrictedTypeID) {
			return nil, fmt.Errorf("restricted types are not sorted (%s, %s)", previousRestrictedTypeID, restrictedTypeID)
		}

		restrictionTypeIDs[restrictedTypeID] = struct{}{}
		previousRestrictedTypeID = restrictedTypeID

		restrictions[i] = restrictedType
	}

	return cadence.NewMeteredRestrictedType(
		d.gauge,
		typ,
		restrictions,
	), nil
}

// decodeCCFTypeID decodes encoded id as
// language=CDDL
// id = bstr
func (d *Decoder) decodeCCFTypeID() (ccfTypeID, error) {
	b, err := d.dec.DecodeBytes()
	if err != nil {
		return 0, err
	}
	return newCCFTypeID(b), nil
}

// decodeCadenceTypeID decodes encoded cadence-type-id as
// language=CDDL
// cadence-type-id = tstr
func (d *Decoder) decodeCadenceTypeID() (cadenceTypeID, common.Location, string, error) {
	typeID, err := d.dec.DecodeString()
	if err != nil {
		return "", nil, "", err
	}

	location, identifier, err := common.DecodeTypeID(d.gauge, typeID)
	if err != nil {
		return cadenceTypeID(typeID), nil, "", fmt.Errorf("invalid type ID `%s`: %w", typeID, err)
	} else if location == nil && sema.NativeCompositeTypes[typeID] == nil {
		// If the location is nil and there is no native composite type with this ID, then it's an invalid type.
		// Note: This was moved out from the common.DecodeTypeID() to avoid the circular dependency.
		return cadenceTypeID(typeID), nil, "", fmt.Errorf("invalid type ID for built-in: `%s`", typeID)
	}

	return cadenceTypeID(typeID), location, identifier, nil
}

// decodeTypeRef decodes encoded type-ref as
// language=CDDL
// type-ref =
//
//	; cbor-tag-type-ref
//	#6.136(id)
func (d *Decoder) decodeTypeRef(types *cadenceTypeByCCFTypeID) (cadence.Type, error) {
	id, err := d.decodeCCFTypeID()
	if err != nil {
		return nil, err
	}

	// "Valid CCF Encoding Requirements" in CCF specs:
	//
	//   "type-ref.id MUST refer to composite-type.id."
	//   "type-value-ref.id MUST refer to composite-type-value.id in the same composite-type-value data item."
	t, err := types.typ(id)
	if err != nil {
		return nil, err
	}

	// Track referenced type definition so the decoder can detect
	// encoded but not referenced type definition (extraneous data).
	types.reference(id)

	return t, nil
}
