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
	"sort"

	"github.com/onflow/cadence"
	cadenceErrors "github.com/onflow/cadence/runtime/errors"
)

type encodeTypeFn func(typ cadence.Type, tids ccfTypeIDByCadenceType) error

// encodeInlineType encodes cadence.Type as
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
// All exported Cadence types need to be supported by this function,
// including abstract and interface types.
func (e *Encoder) encodeInlineType(typ cadence.Type, tids ccfTypeIDByCadenceType) error {
	simpleTypeID, ok := simpleTypeIDByType(typ)
	if ok {
		return e.encodeSimpleType(simpleTypeID)
	}

	switch typ := typ.(type) {
	case *cadence.OptionalType:
		return e.encodeOptionalType(typ, tids)

	case *cadence.VariableSizedArrayType:
		return e.encodeVarSizedArrayType(typ, tids)

	case *cadence.ConstantSizedArrayType:
		return e.encodeConstantSizedArrayType(typ, tids)

	case *cadence.DictionaryType:
		return e.encodeDictType(typ, tids)

	case *cadence.InclusiveRangeType:
		return e.encodeInclusiveRangeType(typ, tids)

	case cadence.CompositeType, cadence.InterfaceType:
		id, err := tids.id(typ)
		if err != nil {
			panic(cadenceErrors.NewUnexpectedErrorFromCause(err))
		}
		return e.encodeTypeRef(id)

	case *cadence.ReferenceType:
		return e.encodeReferenceType(typ, tids)

	case *cadence.IntersectionType:
		return e.encodeIntersectionType(typ, tids)

	case *cadence.CapabilityType:
		return e.encodeCapabilityType(typ, tids)

	case *cadence.FunctionType:
		return e.encodeSimpleType(SimpleTypeFunction)

	default:
		panic(cadenceErrors.NewUnexpectedError("unsupported type %s (%T)", typ.ID(), typ))
	}
}

func (e *Encoder) encodeNullableInlineType(typ cadence.Type, tids ccfTypeIDByCadenceType) error {
	if typ == nil {
		return e.enc.EncodeNil()
	}
	return e.encodeInlineType(typ, tids)
}

// encodeSimpleType encodes cadence simple type as
// language=CDDL
// simple-type =
//
//	; cbor-tag-simple-type
//	#6.137(simple-type-id)
func (e *Encoder) encodeSimpleType(id SimpleType) error {
	rawTagNum := []byte{0xd8, CBORTagSimpleType}
	return e.encodeSimpleTypeWithRawTag(uint64(id), rawTagNum)
}

// encodeSimpleTypeWithRawTag encodes simple type with given tag number as
// language=CDDL
// simple-type-id = uint
func (e *Encoder) encodeSimpleTypeWithRawTag(id uint64, rawTagNumber []byte) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// Encode simple-type-id as uint.
	return e.enc.EncodeUint64(id)
}

// encodeOptionalType encodes cadence.OptionalType as
// language=CDDL
// optional-type =
//
//	; cbor-tag-optional-type
//	#6.138(inline-type)
func (e *Encoder) encodeOptionalType(
	typ *cadence.OptionalType,
	tids ccfTypeIDByCadenceType,
) error {
	rawTagNum := []byte{0xd8, CBORTagOptionalType}
	return e.encodeOptionalTypeWithRawTag(
		typ,
		tids,
		e.encodeInlineType,
		rawTagNum,
	)
}

// encodeOptionalTypeWithRawTag encodes cadence.OptionalType
// with given tag number and encode type function.
func (e *Encoder) encodeOptionalTypeWithRawTag(
	typ *cadence.OptionalType,
	tids ccfTypeIDByCadenceType,
	encodeTypeFn encodeTypeFn,
	rawTagNumber []byte,
) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// Encode non-optional type with given encodeTypeFn.
	return encodeTypeFn(typ.Type, tids)
}

// encodeVarSizedArrayType encodes cadence.VariableSizedArrayType as
// language=CDDL
// varsized-array-type =
//
//	; cbor-tag-varsized-array-type
//	#6.139(inline-type)
func (e *Encoder) encodeVarSizedArrayType(
	typ *cadence.VariableSizedArrayType,
	tids ccfTypeIDByCadenceType,
) error {
	rawTagNum := []byte{0xd8, CBORTagVarsizedArrayType}
	return e.encodeVarSizedArrayTypeWithRawTag(
		typ,
		tids,
		e.encodeInlineType,
		rawTagNum,
	)
}

// encodeVarSizedArrayTypeWithRawTag encodes cadence.VariableSizedArrayType
// with given tag number and encode type function.
func (e *Encoder) encodeVarSizedArrayTypeWithRawTag(
	typ *cadence.VariableSizedArrayType,
	tids ccfTypeIDByCadenceType,
	encodeTypeFn encodeTypeFn,
	rawTagNumber []byte,
) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// Encode array element type with given encodeTypeFn.
	return encodeTypeFn(typ.ElementType, tids)
}

// encodeConstantSizedArrayType encodes cadence.ConstantSizedArrayType as
// language=CDDL
// constsized-array-type =
//
//	; cbor-tag-constsized-array-type
//	#6.140([
//	    array-size: uint,
//	    element-type: inline-type
//	])
func (e *Encoder) encodeConstantSizedArrayType(
	typ *cadence.ConstantSizedArrayType,
	tids ccfTypeIDByCadenceType,
) error {
	rawTagNum := []byte{0xd8, CBORTagConstsizedArrayType}
	return e.encodeConstantSizedArrayTypeWithRawTag(
		typ,
		tids,
		e.encodeInlineType,
		rawTagNum,
	)
}

// encodeConstantSizedArrayTypeWithRawTag encodes cadence.ConstantSizedArrayType
// with given tag number and encode type function as
func (e *Encoder) encodeConstantSizedArrayTypeWithRawTag(
	typ *cadence.ConstantSizedArrayType,
	tids ccfTypeIDByCadenceType,
	encodeTypeFn encodeTypeFn,
	rawTagNumber []byte,
) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// Encode array of length 2.
	err = e.enc.EncodeArrayHead(2)
	if err != nil {
		return err
	}

	// element 0: array size as uint
	err = e.enc.EncodeUint(typ.Size)
	if err != nil {
		return err
	}

	// element 1: array element type with given encodeTypeFn
	return encodeTypeFn(typ.ElementType, tids)
}

// encodeDictType encodes cadence.DictionaryType as
// language=CDDL
// dict-type =
//
//	; cbor-tag-dict-type
//	#6.141([
//	    key-type: inline-type,
//	    element-type: inline-type
//	])
func (e *Encoder) encodeDictType(
	typ *cadence.DictionaryType,
	tids ccfTypeIDByCadenceType,
) error {
	rawTagNum := []byte{0xd8, CBORTagDictType}
	return e.encodeDictTypeWithRawTag(
		typ,
		tids,
		e.encodeInlineType,
		rawTagNum,
	)
}

// encodeDictTypeWithRawTag encodes cadence.DictionaryType
// with given tag number and encode type function.
func (e *Encoder) encodeDictTypeWithRawTag(
	typ *cadence.DictionaryType,
	tids ccfTypeIDByCadenceType,
	encodeTypeFn encodeTypeFn,
	rawTagNumber []byte,
) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// Encode array head of length 2.
	err = e.enc.EncodeArrayHead(2)
	if err != nil {
		return err
	}

	// element 0: key type with given encodeTypeFn
	err = encodeTypeFn(typ.KeyType, tids)
	if err != nil {
		return err
	}

	// element 1: element type with given encodeTypeFn
	return encodeTypeFn(typ.ElementType, tids)
}

// encodeInclusiveRangeType encodes cadence.InclusiveRangeType as
// language=CDDL
// inclusiverange-type =
//
// ; cbor-tag-inclusiverange-type
// #6.145(inline-type)
func (e *Encoder) encodeInclusiveRangeType(
	typ *cadence.InclusiveRangeType,
	tids ccfTypeIDByCadenceType,
) error {
	rawTagNum := []byte{0xd8, CBORTagInclusiveRangeType}
	return e.encodeInclusiveRangeTypeWithRawTag(
		typ,
		tids,
		e.encodeInlineType,
		rawTagNum,
	)
}

// encodeInclusiveRangeTypeWithRawTag encodes cadence.InclusiveRangeType
// with given tag number and encode type function.
func (e *Encoder) encodeInclusiveRangeTypeWithRawTag(
	typ *cadence.InclusiveRangeType,
	tids ccfTypeIDByCadenceType,
	encodeTypeFn encodeTypeFn,
	rawTagNumber []byte,
) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// Encode element type with given encodeTypeFn
	return encodeTypeFn(typ.ElementType, tids)
}

// encodeReferenceType encodes cadence.ReferenceType as
// language=CDDL
// reference-type =
//
//	; cbor-tag-reference-type
//	#6.142([
//	  authorized: bool,
//	  type: inline-type,
//	])
func (e *Encoder) encodeReferenceType(
	typ *cadence.ReferenceType,
	tids ccfTypeIDByCadenceType,
) error {
	rawTagNum := []byte{0xd8, CBORTagReferenceType}
	return e.encodeReferenceTypeWithRawTag(
		typ,
		tids,
		e.encodeInlineType,
		rawTagNum,
	)
}

func (e *Encoder) encodeAuthorization(
	auth cadence.Authorization,
) error {
	// TODO: implement this
	return e.enc.EncodeNil()
}

// encodeReferenceTypeWithRawTag encodes cadence.ReferenceType
// with given tag number and encode type function.
func (e *Encoder) encodeReferenceTypeWithRawTag(
	typ *cadence.ReferenceType,
	tids ccfTypeIDByCadenceType,
	encodeTypeFn encodeTypeFn,
	rawTagNumber []byte,
) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// Encode array head of length 2.
	err = e.enc.EncodeArrayHead(2)
	if err != nil {
		return err
	}

	// element 0: authorization
	err = e.encodeAuthorization(typ.Authorization)
	if err != nil {
		return err
	}

	// element 1: referenced type with given encodeTypeFn
	return encodeTypeFn(typ.Type, tids)
}

// encodeIntersectionType encodes cadence.IntersectionType as
// language=CDDL
// intersection-type =
//
//	; cbor-tag-intersection-type
//	#6.143([
//	  type: inline-type / nil,
//	  types: [* inline-type]
//	])
func (e *Encoder) encodeIntersectionType(typ *cadence.IntersectionType, tids ccfTypeIDByCadenceType) error {
	rawTagNum := []byte{0xd8, CBORTagIntersectionType}
	return e.encodeIntersectionTypeWithRawTag(
		typ,
		tids,
		e.encodeNullableInlineType,
		e.encodeInlineType,
		rawTagNum,
	)
}

// encodeIntersectionTypeWithRawTag encodes cadence.IntersectionType
// with given tag number and encode type function.
func (e *Encoder) encodeIntersectionTypeWithRawTag(
	typ *cadence.IntersectionType,
	tids ccfTypeIDByCadenceType,
	encodeTypeFn encodeTypeFn,
	encodeIntersectionTypeFn encodeTypeFn,
	rawTagNumber []byte,
) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// types as array.

	// Encode array head with number of types.
	intersectionTypes := typ.Types
	err = e.enc.EncodeArrayHead(uint64(len(intersectionTypes)))
	if err != nil {
		return err
	}

	switch e.em.sortIntersectionTypes {
	case SortNone:
		for _, res := range intersectionTypes {
			// Encode restriction type with given encodeTypeFn.
			err = encodeIntersectionTypeFn(res, tids)
			if err != nil {
				return err
			}
		}
		return nil

	case SortBytewiseLexical:
		switch len(intersectionTypes) {
		case 0:
			// Short-circuit if there are no types.
			return nil

		case 1:
			// Avoid overhead of sorting if there is only one type.
			// Encode intersection type with given encodeTypeFn.
			return encodeTypeFn(intersectionTypes[0], tids)

		default:
			// "Deterministic CCF Encoding Requirements" in CCF specs:
			//
			//   "intersection-type.types MUST be sorted by intersection's cadence-type-id"
			//   "intersection-type-value.types MUST be sorted by intersection's cadence-type-id."
			sorter := newBytewiseCadenceTypeSorter(intersectionTypes)

			sort.Sort(sorter)

			for _, index := range sorter.indexes {
				// Encode intersection type with given encodeTypeFn.
				err = encodeIntersectionTypeFn(intersectionTypes[index], tids)
				if err != nil {
					return err
				}
			}

			return nil
		}

	default:
		panic(cadenceErrors.NewUnexpectedError("unsupported sort option for intersection types: %d", e.em.sortIntersectionTypes))
	}
}

// encodeCapabilityType encodes cadence.CapabilityType as
// language=CDDL
// capability-type =
//
//	; cbor-tag-capability-type
//	; use an array as an extension point
//	#6.144([
//	    ; borrow-type
//	    inline-type / nil
//	])
func (e *Encoder) encodeCapabilityType(
	typ *cadence.CapabilityType,
	tids ccfTypeIDByCadenceType,
) error {
	rawTagNum := []byte{0xd8, CBORTagCapabilityType}
	return e.encodeCapabilityTypeWithRawTag(
		typ,
		tids,
		e.encodeNullableInlineType,
		rawTagNum,
	)
}

// encodeCapabilityTypeWithRawTag encodes cadence.CapabilityType
// with given tag number and encode type function.
func (e *Encoder) encodeCapabilityTypeWithRawTag(
	typ *cadence.CapabilityType,
	tids ccfTypeIDByCadenceType,
	encodeTypeFn encodeTypeFn,
	rawTagNumber []byte,
) error {
	// Encode CBOR tag number.
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}

	// Encode array head of length 1.
	err = e.enc.EncodeArrayHead(1)
	if err != nil {
		return err
	}

	// element 0: borrow type with given encodeTypeFn.
	return encodeTypeFn(typ.BorrowType, tids)
}

// encodeTypeRef encodes CCF type id as
// language=CDDL
// type-ref =
//
//	; cbor-tag-type-ref
//	#6.136(id)
func (e *Encoder) encodeTypeRef(ref ccfTypeID) error {
	rawTagNum := []byte{0xd8, CBORTagTypeRef}
	return e.encodeTypeRefWithRawTag(ref, rawTagNum)
}

// encodeTypeRefWithRawTag encodes CCF type ID as
// with given tag number.
func (e *Encoder) encodeTypeRefWithRawTag(ref ccfTypeID, rawTagNumber []byte) error {
	err := e.enc.EncodeRawBytes(rawTagNumber)
	if err != nil {
		return err
	}
	return e.encodeCCFTypeID(ref)
}

// encodeCCFTypeID encodes CCF type ID as
// language=CDDL
// id = bstr
func (e *Encoder) encodeCCFTypeID(id ccfTypeID) error {
	return e.enc.EncodeBytes(id.Bytes())
}

// encodeCadenceTypeID encodes Cadence type ID as
// language=CDDL
// cadence-type-id = tstr
func (e *Encoder) encodeCadenceTypeID(id string) error {
	return e.enc.EncodeString(id)
}
