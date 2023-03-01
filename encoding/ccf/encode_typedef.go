/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022-2023 Dapper Labs, Inc.
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
	"fmt"

	"github.com/onflow/cadence"
)

// encodeCompositeType encodes cadence.CompositeType in type definition as
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
// composite-type = [
//
//	id: id,
//	cadence-type-id: cadence-type-id,
//	fields: [
//	    + [
//	        field-name: tstr,
//	        field-type: inline-type
//	    ]
//	]
//
// ]
func (e *Encoder) encodeCompositeType(typ cadence.CompositeType, tids ccfTypeIDByCadenceType) error {
	ccfID, err := tids.id(typ)
	if err != nil {
		return fmt.Errorf("CCF type ID not found for composite type %s (%T)", typ.ID(), typ)
	}

	var cborTagNum uint64

	switch t := typ.(type) {
	case *cadence.StructType:
		cborTagNum = CBORTagStructType

	case *cadence.ResourceType:
		cborTagNum = CBORTagResourceType

	case *cadence.EventType:
		cborTagNum = CBORTagEventType

	case *cadence.ContractType:
		cborTagNum = CBORTagContractType

	case *cadence.EnumType:
		cborTagNum = CBORTagEnumType

	default:
		panic(fmt.Errorf("unexpected composite type %s (%T)", t.ID(), t))
	}

	// Encode tag number indicating composite type.
	err = e.enc.EncodeTagHead(cborTagNum)
	if err != nil {
		return err
	}

	// Encode array head of length 3.
	err = e.enc.EncodeArrayHead(3)
	if err != nil {
		return err
	}

	// element 0: CCF type id
	err = e.encodeCCFTypeID(ccfID)
	if err != nil {
		return err
	}

	// element 1: cadence-type-id
	err = e.encodeCadenceTypeID(typ.ID())
	if err != nil {
		return err
	}

	// element 2: fields as array
	return e.encodeCompositeTypeFields(typ, tids)
}

// encodeCompositeTypeFields encodes field types as
// language=CDDL
//
//	fields: [
//	    + [
//	        field-name: tstr,
//	        field-type: inline-type
//	    ]
//	]
func (e *Encoder) encodeCompositeTypeFields(typ cadence.CompositeType, tids ccfTypeIDByCadenceType) error {
	fieldTypes := typ.CompositeFields()

	// Encode array head with number of fields.
	err := e.enc.EncodeArrayHead(uint64(len(fieldTypes)))
	if err != nil {
		return err
	}

	if len(fieldTypes) == 1 {
		// Avoid overhead of sorting if there is only one field.
		return e.encodeCompositeTypeField(fieldTypes[0], tids)
	}

	// "Deterministic CCF Encoding Requirements" in CCF specs:
	//
	//   "composite-type.fields MUST be sorted by name"
	sortedIndexes := getSortedFieldIndex(typ)

	for i := 0; i < len(sortedIndexes); i++ {
		index := sortedIndexes[i]

		// Encode field
		err = e.encodeCompositeTypeField(fieldTypes[index], tids)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeCompositeTypeField encodes field type as
// language=CDDL
//
//	[
//	    field-name: tstr,
//	    field-type: inline-type
//	]
func (e *Encoder) encodeCompositeTypeField(typ cadence.Field, tids ccfTypeIDByCadenceType) error {
	// Encode array head of length 2.
	err := e.enc.EncodeArrayHead(2)
	if err != nil {
		return err
	}

	// element 0: field identifier as tstr
	err = e.enc.EncodeString(typ.Identifier)
	if err != nil {
		return err
	}

	// element 1: field type as inline-type
	return e.encodeInlineType(typ.Type, tids)
}

// encodeInterfaceType encodes cadence.InterfaceType as
// language=CDDL
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
//
// interface-type = [
//
//	id: id,
//	cadence-type-id: tstr,
//
// ]
func (e *Encoder) encodeInterfaceType(typ cadence.InterfaceType, tids ccfTypeIDByCadenceType) error {
	ccfID, err := tids.id(typ)
	if err != nil {
		return fmt.Errorf("CCF type ID not found for interface type %s (%T)", typ.ID(), typ)
	}

	var cborTagNum uint64

	switch t := typ.(type) {
	case *cadence.StructInterfaceType:
		cborTagNum = CBORTagStructInterfaceType

	case *cadence.ResourceInterfaceType:
		cborTagNum = CBORTagResourceInterfaceType

	case *cadence.ContractInterfaceType:
		cborTagNum = CBORTagContractInterfaceType

	default:
		panic(fmt.Errorf("unexpected interface type %s (%T)", t.ID(), t))
	}

	// Encode tag number indicating interface type.
	err = e.enc.EncodeTagHead(cborTagNum)
	if err != nil {
		return err
	}

	// Encode array head with length 2.
	err = e.enc.EncodeArrayHead(2)
	if err != nil {
		return err
	}

	// element 0: CCf type ID
	err = e.encodeCCFTypeID(ccfID)
	if err != nil {
		return err
	}

	// element 1: cadence-type-id
	return e.encodeCadenceTypeID(typ.ID())
}
