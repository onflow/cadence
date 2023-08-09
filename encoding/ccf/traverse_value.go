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
)

type compositeTypes struct {
	ids           ccfTypeIDByCadenceType
	abstractTypes map[string]bool
	types         []cadence.Type
}

// compositeTypesFromValue returns all composite/interface types for value v.
// Returned types are sorted unique list of static and runtime composite/interface types.
// NOTE: nested composite/interface types are included in the returned types.
func compositeTypesFromValue(v cadence.Value) ([]cadence.Type, ccfTypeIDByCadenceType) {
	ct := &compositeTypes{
		ids:           make(ccfTypeIDByCadenceType),
		abstractTypes: make(map[string]bool),
		types:         make([]cadence.Type, 0, 1),
	}

	// Traverse v to get all unique:
	// - static composite types
	// - static interface types
	// - runtime composite types
	// - runtime interface types
	ct.traverseValue(v)

	if len(ct.ids) < 2 {
		// No need to reassign ccf id, nor sort types.
		return ct.types, ct.ids
	}

	// Sort Cadence types by Cadence type ID.
	sort.Sort(bytewiseCadenceTypeInPlaceSorter(ct.types))

	// Assign sorted array index as local ccf ID.
	for i, typ := range ct.types {
		ct.ids[typ.ID()] = ccfTypeID(i)
	}

	return ct.types, ct.ids
}

func (ct *compositeTypes) traverseValue(v cadence.Value) {

	if v == nil {
		return
	}

	// Traverse type for composite/interface types.
	checkRuntimeType := ct.traverseType(v.Type())

	if !checkRuntimeType {
		// Return without traversing value for runtime types.
		return
	}

	// Traverse v's elements for runtime types.
	// Note: don't need to traverse fields of cadence.Enum
	// because enum's field is an integer subtype.
	switch v := v.(type) {

	case cadence.Optional:
		ct.traverseValue(v.Value)

	case cadence.Array:
		for _, element := range v.Values {
			ct.traverseValue(element)
		}

	case cadence.Dictionary:
		for _, pair := range v.Pairs {
			ct.traverseValue(pair.Key)
			ct.traverseValue(pair.Value)
		}

	case cadence.Struct:
		for _, field := range v.Fields {
			ct.traverseValue(field)
		}

	case cadence.Resource:
		for _, field := range v.Fields {
			ct.traverseValue(field)
		}

	case cadence.Event:
		for _, field := range v.Fields {
			ct.traverseValue(field)
		}

	case cadence.Contract:
		for _, field := range v.Fields {
			ct.traverseValue(field)
		}

	}
}

// traverseType traverses cadence type typ to find composite/interface types and
// returns true if typ contains any abstract type and runtime type needs to be checked.
// It recurisvely traverse cadence.Type if the type contains other cadence.Type,
// such as OptionalType.
// Runtime needs to be checked when typ contains any abstract type.
func (ct *compositeTypes) traverseType(typ cadence.Type) (checkRuntimeType bool) {
	switch typ := typ.(type) {

	case *cadence.OptionalType:
		return ct.traverseType(typ.Type)

	case cadence.ArrayType:
		return ct.traverseType(typ.Element())

	case *cadence.DictionaryType:
		checkKeyRuntimeType := ct.traverseType(typ.KeyType)
		checkValueRuntimeType := ct.traverseType(typ.ElementType)
		return checkKeyRuntimeType || checkValueRuntimeType

	case *cadence.CapabilityType:
		return ct.traverseType(typ.BorrowType)

	case *cadence.ReferenceType:
		return ct.traverseType(typ.Type)

	case *cadence.IntersectionType:
		check := false
		for _, typ := range typ.Types {
			checkTyp := ct.traverseType(typ)
			check = check || checkTyp
		}
		return check

	case cadence.CompositeType: // struct, resource, event, contract, enum
		newType := ct.add(typ)
		if !newType {
			return ct.abstractTypes[typ.ID()]
		}

		check := false
		fields := typ.CompositeFields()
		for _, field := range fields {
			checkField := ct.traverseType(field.Type)
			check = check || checkField
		}

		// Don't need to traverse initializers because
		// they are not encoded and their types aren't needed.

		ct.abstractTypes[typ.ID()] = check

		return check

	case cadence.InterfaceType: // struct interface, resource interface, contract interface
		ct.add(typ)
		// Don't need to traverse fields or initializers because
		// they are not encoded and their types aren't needed.

		// Return true to check runtime type.
		return true

	case cadence.VoidType,
		cadence.BoolType,
		cadence.NeverType,
		cadence.CharacterType,
		cadence.StringType,
		cadence.BytesType,
		cadence.AddressType,
		cadence.IntType,
		cadence.Int8Type,
		cadence.Int16Type,
		cadence.Int32Type,
		cadence.Int64Type,
		cadence.Int128Type,
		cadence.Int256Type,
		cadence.UIntType,
		cadence.UInt8Type,
		cadence.UInt16Type,
		cadence.UInt32Type,
		cadence.UInt64Type,
		cadence.UInt128Type,
		cadence.UInt256Type,
		cadence.Word8Type,
		cadence.Word16Type,
		cadence.Word32Type,
		cadence.Word64Type,
		cadence.Word128Type,
		cadence.Word256Type,
		cadence.Fix64Type,
		cadence.UFix64Type,
		cadence.PathType,
		cadence.StoragePathType,
		cadence.PublicPathType,
		cadence.PrivatePathType,
		cadence.MetaType,
		*cadence.FunctionType,
		cadence.NumberType,
		cadence.SignedNumberType,
		cadence.IntegerType,
		cadence.SignedIntegerType,
		cadence.FixedPointType,
		cadence.SignedFixedPointType:
		// TODO: Maybe there are more types that we can skip checking runtime type for composite type.

		return false

	default:
		return true
	}
}

func (ct *compositeTypes) add(t cadence.Type) bool {
	cadenceTypeID := t.ID()
	if _, ok := ct.ids[cadenceTypeID]; ok {
		return false
	}
	ct.ids[cadenceTypeID] = 0
	ct.types = append(ct.types, t)
	return true
}
