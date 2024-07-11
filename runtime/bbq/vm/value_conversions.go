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

package vm

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

// Utility methods to convert between old and new values.
// These are temporary until all parts of the interpreter are migrated to the vm.

func InterpreterValueToVMValue(value interpreter.Value) Value {
	switch value := value.(type) {
	case nil:
		return nil
	case interpreter.NilValue:
		return Nil
	case interpreter.IntValue:
		return IntValue{value.BigInt.Int64()}
	case *interpreter.StringValue:
		return StringValue{Str: []byte(value.Str)}
	case *interpreter.CompositeValue:
		return newCompositeValueFromOrderedMap(
			value.Dictionary,
			value.Location,
			value.QualifiedIdentifier,
			value.Kind,
		)
	//case interpreter.LinkValue:
	//	return NewLinkValue(
	//		InterpreterValueToVMValue(value.TargetPath).(PathValue),
	//		value.Type,
	//	)
	case interpreter.PathValue:
		return PathValue{
			Domain:     value.Domain,
			Identifier: value.Identifier,
		}
	case interpreter.AddressValue:
		return AddressValue(value)
	case *interpreter.SimpleCompositeValue:
		fields := make(map[string]Value)
		var fieldNames []string

		for name, field := range value.Fields {
			fields[name] = InterpreterValueToVMValue(field)
			fieldNames = append(fieldNames, name)
		}

		return NewSimpleCompositeValue(
			common.CompositeKindStructure,
			value.TypeID,
			fields,
		)
	case *interpreter.DictionaryValue:
		return newDictionaryValueFromAtreeMap(
			value.Type,
			value.ElementSize(),
			value.AtreeMap(),
		)
	case *interpreter.IDCapabilityValue:
		return NewCapabilityValue(
			AddressValue(value.Address),
			NewIntValue(int64(value.ID.ToInt(interpreter.EmptyLocationRange))),
			value.BorrowType,
		)
	case *interpreter.StorageCapabilityControllerValue:
		return NewStorageCapabilityControllerValue(
			value.BorrowType,
			NewIntValue(int64(value.CapabilityID.ToInt(interpreter.EmptyLocationRange))),
			InterpreterValueToVMValue(value.TargetPath).(PathValue),
		)
	default:
		panic(errors.NewUnreachableError())
	}
}

func VMValueToInterpreterValue(value Value) interpreter.Value {
	switch value := value.(type) {
	case nil:
		return nil
	case NilValue:
		return interpreter.Nil
	case IntValue:
		return interpreter.NewIntValueFromInt64(nil, value.SmallInt)
	case StringValue:
		return interpreter.NewUnmeteredStringValue(string(value.Str))
	case *CompositeValue:
		return interpreter.NewCompositeValueFromAtreeMap(
			nil,
			interpreter.CompositeTypeInfo{
				Location:            value.Location,
				QualifiedIdentifier: value.QualifiedIdentifier,
				Kind:                value.Kind,
			},
			value.dictionary,
		)
	case *DictionaryValue:
		staticType := value.Type
		elementSize := interpreter.DictionaryElementSize(staticType)
		return interpreter.NewDictionaryValueFromAtreeMap(
			nil,
			staticType,
			elementSize,
			value.dictionary,
		)
	case CapabilityValue:
		return interpreter.NewCapabilityValue(
			nil,
			interpreter.NewUnmeteredUInt64Value(uint64(value.ID.SmallInt)), // TODO: properly convert
			VMValueToInterpreterValue(value.Address).(interpreter.AddressValue),
			value.BorrowType,
		)
	//case LinkValue:
	//	return interpreter.LinkValue{
	//		TargetPath: VMValueToInterpreterValue(storage, value.TargetPath).(interpreter.PathValue),
	//		Type:       value.StaticType(nil),
	//	}
	case AddressValue:
		return interpreter.AddressValue(value)
	case PathValue:
		return interpreter.PathValue{
			Domain:     value.Domain,
			Identifier: value.Identifier,
		}
	case *SimpleCompositeValue:
		fields := make(map[string]interpreter.Value)
		var fieldNames []string

		for name, field := range value.fields {
			fields[name] = VMValueToInterpreterValue(field)
			fieldNames = append(fieldNames, name)
		}

		return interpreter.NewSimpleCompositeValue(
			nil,
			value.typeID,
			nil,
			fieldNames,
			fields,
			nil,
			nil,
			nil,
		)
	//case *StorageReferenceValue:
	//	return interpreter.NewStorageReferenceValue(
	//		nil,
	//		value.Authorized,
	//		value.TargetStorageAddress,
	//		value.TargetPath,
	//		value.BorrowedType,
	//	)
	case *StorageCapabilityControllerValue:
		return interpreter.NewStorageCapabilityControllerValue(
			nil,
			value.BorrowType,
			interpreter.NewUnmeteredUInt64Value(uint64(value.CapabilityID.SmallInt)),
			VMValueToInterpreterValue(value.TargetPath).(interpreter.PathValue),
		)
	default:
		panic(errors.NewUnreachableError())
	}
}
