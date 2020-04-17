package cadence

import (
	"fmt"
	"sort"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// ConvertType converts a runtime type to its corresponding Go representation.
func ConvertType(typ runtime.Type) Type {
	switch t := typ.(type) {
	case *sema.AnyStructType:
		return AnyStructType{}
	case *sema.VoidType:
		return VoidType{}
	case *sema.OptionalType:
		return convertOptionalType(t)
	case *sema.BoolType:
		return BoolType{}
	case *sema.StringType:
		return StringType{}
	case *sema.IntType:
		return IntType{}
	case *sema.Int8Type:
		return Int8Type{}
	case *sema.Int16Type:
		return Int16Type{}
	case *sema.Int32Type:
		return Int32Type{}
	case *sema.Int64Type:
		return Int64Type{}
	case *sema.Int128Type:
		return Int128Type{}
	case *sema.Int256Type:
		return Int256Type{}
	case *sema.UIntType:
		return UIntType{}
	case *sema.UInt8Type:
		return UInt8Type{}
	case *sema.UInt16Type:
		return UInt16Type{}
	case *sema.UInt32Type:
		return UInt32Type{}
	case *sema.UInt64Type:
		return UInt64Type{}
	case *sema.UInt128Type:
		return UInt128Type{}
	case *sema.UInt256Type:
		return UInt256Type{}
	case *sema.Word8Type:
		return Word8Type{}
	case *sema.Word16Type:
		return Word16Type{}
	case *sema.Word32Type:
		return Word32Type{}
	case *sema.Word64Type:
		return Word64Type{}
	case *sema.Fix64Type:
		return Fix64Type{}
	case *sema.UFix64Type:
		return UFix64Type{}
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
		return AddressType{}
	}

	panic(fmt.Sprintf("cannot convert type of type %T", typ))
}

func convertOptionalType(t *sema.OptionalType) Type {
	convertedType := ConvertType(t.Type)

	return OptionalType{Type: convertedType}
}

func convertVariableSizedType(t *sema.VariableSizedType) Type {
	convertedElement := ConvertType(t.Type)

	return VariableSizedArrayType{ElementType: convertedElement}
}

func convertConstantSizedType(t *sema.ConstantSizedType) Type {
	convertedElement := ConvertType(t.Type)

	return ConstantSizedArrayType{
		Size:        uint(t.Size),
		ElementType: convertedElement,
	}
}

func convertCompositeType(t *sema.CompositeType) Type {
	fields := make([]Field, 0, len(t.Members))

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

		fields = append(fields, Field{
			Identifier: identifier,
			Type:       convertedFieldType,
		})
	}

	id := string(t.ID())

	switch t.Kind {
	case common.CompositeKindStructure:
		return StructType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindResource:
		return ResourceType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindEvent:
		return EventType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	}

	panic(fmt.Sprintf("cannot convert type %v of unknown kind %v", t, t.Kind))
}

func convertDictionaryType(t *sema.DictionaryType) Type {
	convertedKeyType := ConvertType(t.KeyType)
	convertedElementType := ConvertType(t.ValueType)

	return DictionaryType{
		KeyType:     convertedKeyType,
		ElementType: convertedElementType,
	}
}

func convertFunctionType(t *sema.FunctionType) Type {
	convertedReturnType := ConvertType(t.ReturnTypeAnnotation.Type)

	parameters := make([]Parameter, len(t.Parameters))

	for i, parameter := range t.Parameters {
		convertedParameterType := ConvertType(parameter.TypeAnnotation.Type)

		parameters[i] = Parameter{
			Label:      parameter.Label,
			Identifier: parameter.Identifier,
			Type:       convertedParameterType,
		}
	}

	return Function{
		Parameters: parameters,
		ReturnType: convertedReturnType,
	}.WithID(string(t.ID()))
}
