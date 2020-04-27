/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package runtime

import (
	"fmt"
	"sort"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// ConvertType converts a runtime type to its corresponding Go representation.
func ConvertType(typ Type) cadence.Type {
	switch t := typ.(type) {
	case *sema.AnyStructType:
		return cadence.AnyStructType{}
	case *sema.VoidType:
		return cadence.VoidType{}
	case *sema.OptionalType:
		return convertOptionalType(t)
	case *sema.BoolType:
		return cadence.BoolType{}
	case *sema.StringType:
		return cadence.StringType{}
	case *sema.IntType:
		return cadence.IntType{}
	case *sema.Int8Type:
		return cadence.Int8Type{}
	case *sema.Int16Type:
		return cadence.Int16Type{}
	case *sema.Int32Type:
		return cadence.Int32Type{}
	case *sema.Int64Type:
		return cadence.Int64Type{}
	case *sema.Int128Type:
		return cadence.Int128Type{}
	case *sema.Int256Type:
		return cadence.Int256Type{}
	case *sema.UIntType:
		return cadence.UIntType{}
	case *sema.UInt8Type:
		return cadence.UInt8Type{}
	case *sema.UInt16Type:
		return cadence.UInt16Type{}
	case *sema.UInt32Type:
		return cadence.UInt32Type{}
	case *sema.UInt64Type:
		return cadence.UInt64Type{}
	case *sema.UInt128Type:
		return cadence.UInt128Type{}
	case *sema.UInt256Type:
		return cadence.UInt256Type{}
	case *sema.Word8Type:
		return cadence.Word8Type{}
	case *sema.Word16Type:
		return cadence.Word16Type{}
	case *sema.Word32Type:
		return cadence.Word32Type{}
	case *sema.Word64Type:
		return cadence.Word64Type{}
	case *sema.Fix64Type:
		return cadence.Fix64Type{}
	case *sema.UFix64Type:
		return cadence.UFix64Type{}
	case *sema.VariableSizedType:
		return convertVariableSizedType(t)
	case *sema.ConstantSizedType:
		return convertConstantSizedType(t)
	case *sema.CompositeType:
		return convertCompositeType(t)
	case *sema.DictionaryType:
		return convertDictionaryType(t)
	case *sema.FunctionType:
		return convertFunctionType(t)
	case *sema.AddressType:
		return cadence.AddressType{}
	}

	panic(fmt.Sprintf("cannot convert type of type %T", typ))
}

func convertOptionalType(t *sema.OptionalType) cadence.Type {
	convertedType := ConvertType(t.Type)

	return cadence.OptionalType{Type: convertedType}
}

func convertVariableSizedType(t *sema.VariableSizedType) cadence.Type {
	convertedElement := ConvertType(t.Type)

	return cadence.VariableSizedArrayType{ElementType: convertedElement}
}

func convertConstantSizedType(t *sema.ConstantSizedType) cadence.Type {
	convertedElement := ConvertType(t.Type)

	return cadence.ConstantSizedArrayType{
		Size:        uint(t.Size),
		ElementType: convertedElement,
	}
}

func convertCompositeType(t *sema.CompositeType) cadence.Type {
	fields := make([]cadence.Field, 0, len(t.Members))

	// TODO: do not sort fields before export, store in order declared
	fieldNames := make([]string, 0, len(t.Members))
	for identifier, member := range t.Members {
		if member.IgnoreInSerialization {
			continue
		}
		fieldNames = append(fieldNames, identifier)
	}

	// sort field names in lexicographical order
	sort.Strings(fieldNames)

	for _, identifier := range fieldNames {
		field := t.Members[identifier]

		convertedFieldType := ConvertType(field.TypeAnnotation.Type)

		fields = append(fields, cadence.Field{
			Identifier: identifier,
			Type:       convertedFieldType,
		})
	}

	id := string(t.ID())

	switch t.Kind {
	case common.CompositeKindStructure:
		return cadence.StructType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindResource:
		return cadence.ResourceType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindEvent:
		return cadence.EventType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	}

	panic(fmt.Sprintf("cannot convert type %v of unknown kind %v", t, t.Kind))
}

func convertDictionaryType(t *sema.DictionaryType) cadence.Type {
	convertedKeyType := ConvertType(t.KeyType)
	convertedElementType := ConvertType(t.ValueType)

	return cadence.DictionaryType{
		KeyType:     convertedKeyType,
		ElementType: convertedElementType,
	}
}

func convertFunctionType(t *sema.FunctionType) cadence.Type {
	convertedReturnType := ConvertType(t.ReturnTypeAnnotation.Type)

	parameters := make([]cadence.Parameter, len(t.Parameters))

	for i, parameter := range t.Parameters {
		convertedParameterType := ConvertType(parameter.TypeAnnotation.Type)

		parameters[i] = cadence.Parameter{
			Label:      parameter.Label,
			Identifier: parameter.Identifier,
			Type:       convertedParameterType,
		}
	}

	return cadence.Function{
		Parameters: parameters,
		ReturnType: convertedReturnType,
	}.WithID(string(t.ID()))
}
