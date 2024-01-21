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
//	/ intersection-type
//	/ capability-type
//	/ inclusiverange-type
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

	case CBORTagInclusiveRangeType:
		return d.decodeInclusiveRangeType(types, d.decodeInlineType)

	case CBORTagReferenceType:
		return d.decodeReferenceType(types, d.decodeInlineType)

	case CBORTagIntersectionType:
		return d.decodeIntersectionType(types, d.decodeInlineType)

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

	ty := typeBySimpleTypeID(SimpleType(simpleTypeID))
	if ty == nil {
		return nil, fmt.Errorf("unsupported encoded simple type ID %d", simpleTypeID)
	}

	return ty, nil
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

// decodeInclusiveRangeType decodes inclusiverange-type or inclusiverange-type-value as
// language=CDDL
// inclusiverange-type =
//
//	; cbor-tag-inclusiverange-type
//	#6.145(inline-type)
//
// inclusiverange-type-value =
//
//	; cbor-tag-inclusiverange-type-value
//	#6.194(type-value)
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeInclusiveRangeType(
	types *cadenceTypeByCCFTypeID,
	decodeTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// element 0: element type (inline-type or type-value)
	elementType, err := decodeTypeFn(types)
	if err != nil {
		return nil, err
	}

	if elementType == nil {
		return nil, errors.New("unexpected nil type as InclusiveRange element type")
	}

	return cadence.NewMeteredInclusiveRangeType(d.gauge, elementType), nil
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

func (d *Decoder) decodeAuthorization() (cadence.Authorization, error) {
	err := d.dec.DecodeNil()
	return cadence.UnauthorizedAccess, err
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

	// element 0: authorization
	authorization, err := d.decodeAuthorization()
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

	return cadence.NewMeteredReferenceType(d.gauge, authorization, elementType), nil
}

// decodeIntersectionType decodes intersection-type or intersection-type-value as
// language=CDDL
// intersection-type =
//
//	; cbor-tag-intersection-type
//	#6.143([
//	  type: inline-type / nil,
//	  types: [* inline-type]
//	])
//
// intersection-type-value =
//
//	; cbor-tag-intersection-type-value
//	#6.191([
//	  type: type-value / nil,
//	  types: [* type-value]
//	])
//
// NOTE: decodeTypeFn is responsible for decoding inline-type or type-value.
func (d *Decoder) decodeIntersectionType(
	types *cadenceTypeByCCFTypeID,
	decodeIntersectionTypeFn decodeTypeFn,
) (cadence.Type, error) {
	// types
	typeCount, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}
	if typeCount == 0 {
		return nil, errors.New("unexpected empty intersection type")
	}

	intersectionTypeIDs := make(map[string]struct{}, typeCount)
	var previousIntersectionTypeID string

	intersectionTypes := make([]cadence.Type, typeCount)
	for i := 0; i < int(typeCount); i++ {
		// Decode type.
		intersectedType, err := decodeIntersectionTypeFn(types)
		if err != nil {
			return nil, err
		}

		if intersectedType == nil {
			return nil, errors.New("unexpected nil type as intersection type")
		}

		intersectionTypeID := intersectedType.ID()

		// "Valid CCF Encoding Requirements" in CCF specs:
		//
		//   "Elements MUST be unique in intersection-type or intersection-type-value."
		if _, ok := intersectionTypeIDs[intersectionTypeID]; ok {
			return nil, fmt.Errorf("found duplicate intersection type %s", intersectionTypeID)
		}

		if d.dm.enforceSortRestrictedTypes == EnforceSortBytewiseLexical {
			// "Deterministic CCF Encoding Requirements" in CCF specs:
			//
			//   "intersection-type.types MUST be sorted by intersection's cadence-type-id"
			//   "intersection-type-value.types MUST be sorted by intersection's cadence-type-id."
			if !stringsAreSortedBytewise(previousIntersectionTypeID, intersectionTypeID) {
				return nil, fmt.Errorf("restricted types are not sorted (%s, %s)", previousIntersectionTypeID, intersectionTypeID)
			}
		}

		intersectionTypeIDs[intersectionTypeID] = struct{}{}
		previousIntersectionTypeID = intersectionTypeID

		intersectionTypes[i] = intersectedType
	}

	if len(intersectionTypes) == 0 {
		return nil, errors.New("unexpected empty intersection type")
	}

	return cadence.NewMeteredIntersectionType(
		d.gauge,
		intersectionTypes,
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
