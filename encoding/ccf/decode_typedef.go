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

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	cadenceErrors "github.com/onflow/cadence/runtime/errors"
)

type rawFieldsWithCCFTypeID struct {
	typeID    ccfTypeID
	rawFields []byte
}

// decodeTypeDefs decodes composite/interface type definitions as
// language=CDDL
// composite-typedef = [
//
//	 ; one-or-more instead of zero-or-more because:
//	 ; - when encoding a primitive type, such as boolean or string, `ccf-type-and-value-message` is used (no `composite-typedef` at all)
//	 ; - when encoding a composite type, such as event, `ccf-typedef-and-value-message` is used, which encodes at least one `composite-typedef`
//		+ (
//		  struct-type
//		  / resource-type
//		  / contract-type
//		  / event-type
//		  / enum-type
//		  / struct-interface-type
//		  / resource-interface-type
//		  / contract-interface-type
//		  )]
func (d *Decoder) decodeTypeDefs() (*cadenceTypeByCCFTypeID, error) {
	// Decode number of type definitions.
	count, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, errors.New("found 0 type definition in composite-typedef (expected at least 1 type definition)")
	}

	types := newCadenceTypeByCCFTypeID()

	// NOTE: composite fields are not decoded while composite types are decoded
	// because field type might reference composite type that hasn't decoded yet.
	rawFieldsOfTypes := make([]rawFieldsWithCCFTypeID, 0, count)

	// cadenceTypeIDs is used to check if cadence type IDs are unique in type definitions.
	cadenceTypeIDs := make(map[cadenceTypeID]struct{}, count)

	// previousCadenceID is used to check if type definitions are sorted by cadence type IDs.
	var previousCadenceID cadenceTypeID

	for i := uint64(0); i < count; i++ {
		ccfID, cadenceID, rawFields, err := d.decodeTypeDef(types)
		if err != nil {
			return nil, err
		}

		// "Valid CCF Encoding Requirements" in CCF specs:
		//
		//   "composite-type.cadence-type-id MUST be unique in
		//    ccf-typedef-message or ccf-typedef-and-value-message."
		if _, ok := cadenceTypeIDs[cadenceID]; ok {
			return nil, fmt.Errorf("found duplicate Cadence type ID %s in composite-typedef", cadenceID)
		}

		// "Deterministic CCF Encoding Requirements" in CCF specs:
		//
		//   "composite-type.id in ccf-typedef-and-value-message MUST
		//    be identical to its zero-based index in composite-typedef."
		if !ccfID.Equal(newCCFTypeIDFromUint64(i)) {
			return nil, fmt.Errorf(
				"CCF type ID %d doesn't match composite-typedef index %d in composite-typedef",
				ccfID,
				i,
			)
		}

		// "Deterministic CCF Encoding Requirements" in CCF specs:
		//
		//   "Type definitions MUST be sorted by cadence-type-id in composite-typedef."
		if !stringsAreSortedBytewise(string(previousCadenceID), string(cadenceID)) {
			return nil, fmt.Errorf(
				"Cadence type ID (%s, %s) isn't sorted in composite-typedef",
				string(previousCadenceID),
				string(cadenceID),
			)
		}

		if len(rawFields) > 0 {
			rawFieldsOfTypes = append(rawFieldsOfTypes, rawFieldsWithCCFTypeID{typeID: ccfID, rawFields: rawFields})
		}

		cadenceTypeIDs[cadenceID] = struct{}{}
		previousCadenceID = cadenceID
	}

	// Decode fields after all high-level type definitions are resolved.
	for _, rawFields := range rawFieldsOfTypes {
		typ, err := types.typ(rawFields.typeID)
		if err != nil {
			panic(cadenceErrors.NewUnexpectedErrorFromCause(err))
		}

		dec := d.dm.NewDecoder(d.gauge, rawFields.rawFields)
		fields, err := dec.decodeCompositeFields(types, dec.decodeInlineType)
		if err != nil {
			return nil, err
		}

		switch typ := typ.(type) {
		case cadence.CompositeType:
			typ.SetCompositeFields(fields)

		default:
			return nil, fmt.Errorf("unsupported type %s (%T) in composite-typedef", typ.ID(), typ)
		}
	}

	return types, nil
}

// decodeTypeDef decodes composite/interface type in type definition as
// language=CDDL
// struct-type =
//
//	; cbor-tag-struct-type
//	#6.160(composite-type)
//
// resource-type =
//
//	; cbor-tag-resource-type
//	#6.161(composite-type)
//
// event-type =
//
//	; cbor-tag-event-type
//	#6.162(composite-type)
//
// contract-type =
//
//	; cbor-tag-contract-type
//	#6.163(composite-type)
//
// enum-type =
//
//	; cbor-tag-enum-type
//	#6.164(composite-type)
//
// struct-interface-type =
//
//	; cbor-tag-struct-interface-type
//	#6.176(interface-type)
//
// resource-interface-type =
//
//	; cbor-tag-resource-interface-type
//	#6.177(interface-type)
//
// contract-interface-type =
//
//	; cbor-tag-contract-interface-type
//	#6.178(interface-type)
func (d *Decoder) decodeTypeDef(
	types *cadenceTypeByCCFTypeID,
) (
	ccfTypeID,
	cadenceTypeID,
	[]byte,
	error,
) {
	tagNum, err := d.dec.DecodeTagNumber()
	if err != nil {
		return ccfTypeID(0), cadenceTypeID(""), nil, err
	}

	switch tagNum {
	case CBORTagStructType:
		ctr := func(location common.Location, identifier string) cadence.Type {
			return cadence.NewMeteredStructType(
				d.gauge,
				location,
				identifier,
				nil,
				nil,
			)
		}
		return d.decodeCompositeType(types, ctr)

	case CBORTagResourceType:
		ctr := func(location common.Location, identifier string) cadence.Type {
			return cadence.NewMeteredResourceType(
				d.gauge,
				location,
				identifier,
				nil,
				nil,
			)
		}
		return d.decodeCompositeType(types, ctr)

	case CBORTagEventType:
		ctr := func(location common.Location, identifier string) cadence.Type {
			return cadence.NewMeteredEventType(
				d.gauge,
				location,
				identifier,
				nil,
				nil,
			)
		}
		return d.decodeCompositeType(types, ctr)

	case CBORTagContractType:
		ctr := func(location common.Location, identifier string) cadence.Type {
			return cadence.NewMeteredContractType(
				d.gauge,
				location,
				identifier,
				nil,
				nil,
			)
		}
		return d.decodeCompositeType(types, ctr)

	case CBORTagEnumType:
		ctr := func(location common.Location, identifier string) cadence.Type {
			return cadence.NewMeteredEnumType(
				d.gauge,
				location,
				identifier,
				nil,
				nil,
				nil,
			)
		}
		return d.decodeCompositeType(types, ctr)

	case CBORTagStructInterfaceType:
		ctr := func(location common.Location, identifier string) cadence.Type {
			return cadence.NewMeteredStructInterfaceType(
				d.gauge,
				location,
				identifier,
				nil,
				nil,
			)
		}
		return d.decodeInterfaceType(types, ctr)

	case CBORTagResourceInterfaceType:
		ctr := func(location common.Location, identifier string) cadence.Type {
			return cadence.NewMeteredResourceInterfaceType(
				d.gauge,
				location,
				identifier,
				nil,
				nil,
			)
		}
		return d.decodeInterfaceType(types, ctr)

	case CBORTagContractInterfaceType:
		ctr := func(location common.Location, identifier string) cadence.Type {
			return cadence.NewMeteredContractInterfaceType(
				d.gauge,
				location,
				identifier,
				nil,
				nil,
			)
		}
		return d.decodeInterfaceType(types, ctr)

	default:
		return ccfTypeID(0),
			cadenceTypeID(""),
			nil,
			fmt.Errorf("unsupported type definition with CBOR tag number %d", tagNum)
	}
}

// decodeCompositeType decodes composite type in type definition as
// language=CDDL
// composite-type = [
//
//	id: id,
//	cadence-type-id: cadence-type-id,
//	fields: [
//	    * [
//	        field-name: tstr,
//	        field-type: inline-type
//	    ]
//	]
//
// ]
func (d *Decoder) decodeCompositeType(
	types *cadenceTypeByCCFTypeID,
	constructor func(common.Location, string) cadence.Type,
) (
	ccfTypeID,
	cadenceTypeID,
	[]byte,
	error,
) {

	// Decode array head of length 3.
	err := decodeCBORArrayWithKnownSize(d.dec, 3)
	if err != nil {
		return ccfTypeID(0), cadenceTypeID(""), nil, err
	}

	// element 0: id
	ccfID, err := d.decodeCCFTypeID()
	if err != nil {
		return ccfTypeID(0), cadenceTypeID(""), nil, err
	}

	// "Valid CCF Encoding Requirements" in CCF specs:
	//
	//   "composite-type.id MUST be unique in ccf-typedef-message or ccf-typedef-and-value-message."
	if types.has(ccfID) {
		return ccfTypeID(0), cadenceTypeID(""), nil, fmt.Errorf("found duplicate CCF type ID %d in composite-type", ccfID)
	}

	// element 1: cadence-type-id
	cadenceID, location, identifier, err := d.decodeCadenceTypeID()
	if err != nil {
		return ccfTypeID(0), cadenceTypeID(""), nil, err
	}

	// element 2: fields
	rawField, err := d.dec.DecodeRawBytes()
	if err != nil {
		return ccfTypeID(0), cadenceTypeID(""), nil, err
	}

	// The return value can be ignored, because its non-existence was already checked above
	_ = types.add(ccfID, constructor(location, identifier))

	return ccfID, cadenceID, rawField, nil
}

// decodeCompositeFields decodes field types as
// language=CDDL
//
//	fields: [
//	    * [
//	        field-name: tstr,
//	        field-type: inline-type
//	    ]
//	]
func (d *Decoder) decodeCompositeFields(types *cadenceTypeByCCFTypeID, decodeTypeFn decodeTypeFn) ([]cadence.Field, error) {
	// Decode number of fields.
	fieldCount, err := d.dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	fields := make([]cadence.Field, fieldCount)
	fieldNames := make(map[string]struct{}, fieldCount)
	var previousFieldName string

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceField,
		Amount: fieldCount,
	})

	for i := 0; i < int(fieldCount); i++ {
		field, err := d.decodeCompositeField(types, decodeTypeFn)
		if err != nil {
			return nil, err
		}

		// "Valid CCF Encoding Requirements" in CCF specs:
		//
		//   "field-name MUST be unique in composite-type."
		//   "name MUST be unique in composite-type-value.fields."
		if _, ok := fieldNames[field.Identifier]; ok {
			return nil, fmt.Errorf("found duplicate field name %s in composite-type", field.Identifier)
		}

		if d.dm.enforceSortCompositeFields == EnforceSortBytewiseLexical {
			// "Deterministic CCF Encoding Requirements" in CCF specs:
			//
			//   "composite-type.fields MUST be sorted by name"
			//   "composite-type-value.fields MUST be sorted by name."
			if !stringsAreSortedBytewise(previousFieldName, field.Identifier) {
				return nil, fmt.Errorf("field names are not sorted in composite-type (%s, %s)", previousFieldName, field.Identifier)
			}
		}

		fieldNames[field.Identifier] = struct{}{}
		previousFieldName = field.Identifier
		fields[i] = field
	}

	return fields, nil
}

// decodeCompositeField decodes field type as
// language=CDDL
//
//	[
//	    field-name: tstr,
//	    field-type: inline-type
//	]
func (d *Decoder) decodeCompositeField(types *cadenceTypeByCCFTypeID, decodeTypeFn decodeTypeFn) (cadence.Field, error) {
	// Decode array head of length 2
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return cadence.Field{}, err
	}

	// element 0: field-name
	fieldName, err := d.dec.DecodeString()
	if err != nil {
		return cadence.Field{}, err
	}

	// element 1: field-type
	fieldType, err := decodeTypeFn(types)
	if err != nil {
		return cadence.Field{}, err
	}

	if fieldType == nil {
		return cadence.Field{}, errors.New("unexpected nil type as composite field type")
	}

	// Unmetered because decodeCompositeField is metered in decodeCompositeFields and called nowhere else
	// fieldType is still metered.
	return cadence.NewField(fieldName, fieldType), nil
}

// decodeInterfaceType decodes interface type as
// language=CDDL
// interface-type = [
//
//	id: id,
//	cadence-type-id: tstr,
//
// ]
func (d *Decoder) decodeInterfaceType(
	types *cadenceTypeByCCFTypeID,
	constructor func(common.Location, string) cadence.Type,
) (
	ccfTypeID,
	cadenceTypeID,
	[]byte, // always nil since interface type definition doesn't contain fields
	error,
) {

	// Decode array head of length 2.
	err := decodeCBORArrayWithKnownSize(d.dec, 2)
	if err != nil {
		return ccfTypeID(0), cadenceTypeID(""), nil, err
	}

	// element 0: id
	ccfID, err := d.decodeCCFTypeID()
	if err != nil {
		return ccfTypeID(0), cadenceTypeID(""), nil, err
	}

	// "Valid CCF Encoding Requirements" in CCF specs:
	//
	//   "composite-type.id MUST be unique in ccf-typedef-message or ccf-typedef-and-value-message."
	if types.has(ccfID) {
		return ccfTypeID(0), cadenceTypeID(""), nil, fmt.Errorf("found duplicate CCF type ID %d in interface-type", ccfID)
	}

	// element 1: cadence-type-id
	cadenceID, location, identifier, err := d.decodeCadenceTypeID()
	if err != nil {
		return ccfTypeID(0), cadenceTypeID(""), nil, err
	}

	// The return value can be ignored, because its non-existence was already checked above
	_ = types.add(ccfID, constructor(location, identifier))

	return ccfID, cadenceID, nil, nil
}
