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
	ids   ccfTypeIDByCadenceType
	types []cadence.Type
}

// compositeTypesFromValue returns all composite/interface types for value v.
// Returned types are sorted unique list of static and runtime composite/interface types.
// NOTE: nested composite/interface types are included in the returned types.
func compositeTypesFromValue(v cadence.Value) ([]cadence.Type, ccfTypeIDByCadenceType) {
	ct := &compositeTypes{
		ids:   make(ccfTypeIDByCadenceType),
		types: make([]cadence.Type, 0, 1),
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
	for i := 0; i < len(ct.types); i++ {
		ct.ids[ct.types[i].ID()] = ccfTypeID(i)
	}

	return ct.types, ct.ids
}

func (ct *compositeTypes) traverseValue(v cadence.Value) {

	// Traverse type for composite/interface types.
	checkRuntimeType := ct.traverseType(v.Type())

	if !checkRuntimeType {
		// Return without traversing value for runtime types.
		return
	}

	// Traverse v's elements for runtime types.
	switch x := v.(type) {

	case cadence.Optional:
		ct.traverseValue(x.Value)

	case cadence.Array:
		for i := 0; i < len(x.Values); i++ {
			ct.traverseValue(x.Values[i])
		}

	case cadence.Dictionary:
		for i := 0; i < len(x.Pairs); i++ {
			ct.traverseValue(x.Pairs[i].Key)
			ct.traverseValue(x.Pairs[i].Value)
		}

	case cadence.Struct:
		for i := 0; i < len(x.Fields); i++ {
			ct.traverseValue(x.Fields[i])
		}

	case cadence.Resource:
		for i := 0; i < len(x.Fields); i++ {
			ct.traverseValue(x.Fields[i])
		}

	case cadence.Event:
		for i := 0; i < len(x.Fields); i++ {
			ct.traverseValue(x.Fields[i])
		}

	case cadence.Contract:
		for i := 0; i < len(x.Fields); i++ {
			ct.traverseValue(x.Fields[i])
		}

	case cadence.Enum:
		for i := 0; i < len(x.Fields); i++ {
			ct.traverseValue(x.Fields[i])
		}
	}
}

// traverseType traverses cadence type typ to find composite/interface types and
// returns true if typ contains any abstract type and runtime type needs to be checked.
// It recurisvely traverse cadence.Type if the type contains other cadence.Type,
// such as OptionalType.
// Runtime needs to be checked when typ contains any abstract type.
func (ct *compositeTypes) traverseType(typ cadence.Type) (checkRuntimeType bool) {
	switch t := typ.(type) {

	case *cadence.OptionalType:
		return ct.traverseType(t.Type)

	case cadence.ArrayType:
		return ct.traverseType(t.Element())

	case *cadence.DictionaryType:
		checkKeyRuntimeType := ct.traverseType(t.KeyType)
		checkValueRuntimeType := ct.traverseType(t.ElementType)
		return checkKeyRuntimeType || checkValueRuntimeType

	case *cadence.CapabilityType:
		return ct.traverseType(t.BorrowType)

	case *cadence.ReferenceType:
		return ct.traverseType(t.Type)

	case *cadence.RestrictedType:
		check := ct.traverseType(t.Type)
		for i := 0; i < len(t.Restrictions); i++ {
			checkRestriction := ct.traverseType(t.Restrictions[i])
			check = check || checkRestriction
		}
		return check

	case cadence.CompositeType: // struct, resource, event, contract, enum
		check := false

		newType := ct.add(t)
		if newType {
			fields := t.CompositeFields()
			for i := 0; i < len(fields); i++ {
				checkField := ct.traverseType(fields[i].Type)
				check = check || checkField
			}

			// Don't need to traverse initializers because
			// they are not encoded and their types aren't needed.
		}

		return check

	case cadence.InterfaceType: // struct interface, resource interface, contract interface
		ct.add(t)
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
		cadence.Fix64Type,
		cadence.UFix64Type,
		cadence.PathType,
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
