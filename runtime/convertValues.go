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

package runtime

import (
	"math/big"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

// exportValue converts a runtime value to its native Go representation.
func exportValue(
	value exportableValue,
	locationRange interpreter.LocationRange,
) (
	cadence.Value,
	error,
) {
	return exportValueWithInterpreter(
		value.Value,
		value.Interpreter(),
		locationRange,
		seenReferences{},
	)
}

// ExportValue converts a runtime value to its native Go representation.
func ExportValue(
	value interpreter.Value,
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
) (cadence.Value, error) {
	return exportValueWithInterpreter(
		value,
		inter,
		locationRange,
		seenReferences{},
	)
}

// NOTE: Do not generalize to map[interpreter.Value],
// as not all values are Go hashable, i.e. this might lead to run-time panics
type seenReferences map[*interpreter.EphemeralReferenceValue]struct{}

// exportValueWithInterpreter exports the given internal (interpreter) value to an external value.
//
// The export is recursive, the results parameter prevents cycles:
// it is checked at the start of the recursively called function,
// and pre-set before a recursive call.
func exportValueWithInterpreter(
	value interpreter.Value,
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {

	switch v := value.(type) {
	case interpreter.VoidValue:
		return cadence.NewMeteredVoid(inter), nil
	case interpreter.NilValue:
		return cadence.NewMeteredOptional(inter, nil), nil
	case *interpreter.SomeValue:
		return exportSomeValue(v, inter, locationRange, seenReferences)
	case interpreter.BoolValue:
		return cadence.NewMeteredBool(inter, bool(v)), nil
	case *interpreter.StringValue:
		return cadence.NewMeteredString(
			inter,
			common.NewCadenceStringMemoryUsage(len(v.Str)),
			func() string {
				return v.Str
			},
		)
	case interpreter.CharacterValue:
		return cadence.NewMeteredCharacter(
			inter,
			common.NewCadenceCharacterMemoryUsage(len(v)),
			func() string {
				return string(v)
			},
		)
	case *interpreter.ArrayValue:
		return exportArrayValue(
			v,
			inter,
			locationRange,
			seenReferences,
		)
	case interpreter.IntValue:
		bigInt := v.ToBigInt(inter)
		return cadence.NewMeteredIntFromBig(
			inter,
			common.NewCadenceIntMemoryUsage(
				common.BigIntByteLength(bigInt),
			),
			func() *big.Int {
				return bigInt
			},
		), nil
	case interpreter.Int8Value:
		return cadence.NewMeteredInt8(inter, int8(v)), nil
	case interpreter.Int16Value:
		return cadence.NewMeteredInt16(inter, int16(v)), nil
	case interpreter.Int32Value:
		return cadence.NewMeteredInt32(inter, int32(v)), nil
	case interpreter.Int64Value:
		return cadence.NewMeteredInt64(inter, int64(v)), nil
	case interpreter.Int128Value:
		return cadence.NewMeteredInt128FromBig(
			inter,
			func() *big.Int {
				return v.ToBigInt(inter)
			},
		)
	case interpreter.Int256Value:
		return cadence.NewMeteredInt256FromBig(
			inter,
			func() *big.Int {
				return v.ToBigInt(inter)
			},
		)
	case interpreter.UIntValue:
		bigInt := v.ToBigInt(inter)
		return cadence.NewMeteredUIntFromBig(
			inter,
			common.NewCadenceIntMemoryUsage(
				common.BigIntByteLength(bigInt),
			),
			func() *big.Int {
				return bigInt
			},
		)
	case interpreter.UInt8Value:
		return cadence.NewMeteredUInt8(inter, uint8(v)), nil
	case interpreter.UInt16Value:
		return cadence.NewMeteredUInt16(inter, uint16(v)), nil
	case interpreter.UInt32Value:
		return cadence.NewMeteredUInt32(inter, uint32(v)), nil
	case interpreter.UInt64Value:
		return cadence.NewMeteredUInt64(inter, uint64(v)), nil
	case interpreter.UInt128Value:
		return cadence.NewMeteredUInt128FromBig(
			inter,
			func() *big.Int {
				return v.ToBigInt(inter)
			},
		)
	case interpreter.UInt256Value:
		return cadence.NewMeteredUInt256FromBig(
			inter,
			func() *big.Int {
				return v.ToBigInt(inter)
			},
		)
	case interpreter.Word8Value:
		return cadence.NewMeteredWord8(inter, uint8(v)), nil
	case interpreter.Word16Value:
		return cadence.NewMeteredWord16(inter, uint16(v)), nil
	case interpreter.Word32Value:
		return cadence.NewMeteredWord32(inter, uint32(v)), nil
	case interpreter.Word64Value:
		return cadence.NewMeteredWord64(inter, uint64(v)), nil
	case interpreter.Fix64Value:
		return cadence.Fix64(v), nil
	case interpreter.UFix64Value:
		return cadence.UFix64(v), nil
	case *interpreter.CompositeValue:
		return exportCompositeValue(
			v,
			inter,
			locationRange,
			seenReferences,
		)
	case *interpreter.SimpleCompositeValue:
		return exportCompositeValue(
			v,
			inter,
			locationRange,
			seenReferences,
		)
	case *interpreter.DictionaryValue:
		return exportDictionaryValue(
			v,
			inter,
			locationRange,
			seenReferences,
		)
	case interpreter.AddressValue:
		return cadence.NewMeteredAddress(inter, v), nil
	case interpreter.PathValue:
		return exportPathValue(inter, v), nil
	case interpreter.TypeValue:
		return exportTypeValue(v, inter), nil
	case *interpreter.StorageCapabilityValue:
		return exportStorageCapabilityValue(v, inter), nil
	case *interpreter.EphemeralReferenceValue:
		// Break recursion through ephemeral references
		if _, ok := seenReferences[v]; ok {
			return nil, nil
		}
		defer delete(seenReferences, v)
		seenReferences[v] = struct{}{}
		return exportValueWithInterpreter(
			v.Value,
			inter,
			locationRange,
			seenReferences,
		)
	case *interpreter.StorageReferenceValue:
		referencedValue := v.ReferencedValue(inter, interpreter.EmptyLocationRange, true)
		if referencedValue == nil {
			return nil, nil
		}
		return exportValueWithInterpreter(
			*referencedValue,
			inter,
			locationRange,
			seenReferences,
		)
	case interpreter.FunctionValue:
		return exportFunctionValue(v, inter), nil
	default:
		return nil, &ValueNotExportableError{
			Type: v.StaticType(inter),
		}
	}
}

func exportSomeValue(
	v *interpreter.SomeValue,
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Optional,
	error,
) {
	innerValue := v.InnerValue(inter, locationRange)

	if innerValue == nil {
		return cadence.NewMeteredOptional(inter, nil), nil
	}

	value, err := exportValueWithInterpreter(
		innerValue,
		inter,
		locationRange,
		seenReferences,
	)
	if err != nil {
		return cadence.Optional{}, err
	}

	return cadence.NewMeteredOptional(inter, value), nil
}

func exportArrayValue(
	v *interpreter.ArrayValue,
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Array,
	error,
) {
	array, err := cadence.NewMeteredArray(
		inter,
		v.Count(),
		func() ([]cadence.Value, error) {
			values := make([]cadence.Value, 0, v.Count())

			var err error
			v.Iterate(inter, func(value interpreter.Value) (resume bool) {
				var exportedValue cadence.Value
				exportedValue, err = exportValueWithInterpreter(
					value,
					inter,
					locationRange,
					seenReferences,
				)
				if err != nil {
					return false
				}
				values = append(
					values,
					exportedValue,
				)
				return true
			})

			if err != nil {
				return nil, err
			}
			return values, nil
		},
	)
	if err != nil {
		return cadence.Array{}, err
	}

	exportType := ExportType(v.SemaType(inter), map[sema.TypeID]cadence.Type{}).(cadence.ArrayType)

	return array.WithType(exportType), err
}

func exportCompositeValue(
	v interpreter.Value,
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {

	staticType := v.StaticType(inter)

	semaType, err := inter.ConvertStaticToSemaType(staticType)
	if err != nil {
		return nil, err
	}

	if !semaType.IsExportable(map[*sema.Member]bool{}) {
		return nil, &ValueNotExportableError{
			Type: staticType,
		}
	}

	compositeType, ok := semaType.(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// TODO: consider making the results map "global", by moving it up to exportValueWithInterpreter
	t := exportCompositeType(inter, compositeType, map[sema.TypeID]cadence.Type{})

	// NOTE: use the exported type's fields to ensure fields in type
	// and value are in sync

	fieldNames := t.CompositeFields()

	makeFields := func() ([]cadence.Value, error) {
		fields := make([]cadence.Value, len(fieldNames))

		for i, field := range fieldNames {
			fieldName := field.Identifier

			var fieldValue interpreter.Value

			switch v := v.(type) {
			case *interpreter.SimpleCompositeValue:
				fieldValue = v.Fields[fieldName]
				computeField := v.ComputeField

				if fieldValue == nil && computeField != nil {
					fieldValue = computeField(fieldName, inter, locationRange)
				}

			case *interpreter.CompositeValue:
				fieldValue = v.GetField(inter, locationRange, fieldName)
				if fieldValue == nil && v.ComputedFields != nil {
					if computedField, ok := v.ComputedFields[fieldName]; ok {
						fieldValue = computedField(inter, locationRange)
					}
				}
			}

			exportedFieldValue, err := exportValueWithInterpreter(
				fieldValue,
				inter,
				locationRange,
				seenReferences,
			)
			if err != nil {
				return nil, err
			}
			fields[i] = exportedFieldValue
		}

		return fields, nil
	}

	compositeKind := compositeType.Kind

	// NOTE: when modifying the cases below,
	// also update the error message below!

	switch compositeKind {
	case common.CompositeKindStructure:
		structure, err := cadence.NewMeteredStruct(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return structure.WithType(t.(*cadence.StructType)), nil
	case common.CompositeKindResource:
		resource, err := cadence.NewMeteredResource(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return resource.WithType(t.(*cadence.ResourceType)), nil
	case common.CompositeKindEvent:
		event, err := cadence.NewMeteredEvent(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return event.WithType(t.(*cadence.EventType)), nil
	case common.CompositeKindContract:
		contract, err := cadence.NewMeteredContract(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return contract.WithType(t.(*cadence.ContractType)), nil
	case common.CompositeKindEnum:
		enum, err := cadence.NewMeteredEnum(
			inter,
			len(fieldNames),
			func() ([]cadence.Value, error) {
				return makeFields()
			},
		)
		if err != nil {
			return nil, err
		}
		return enum.WithType(t.(*cadence.EnumType)), nil
	}

	return nil, errors.NewDefaultUserError(
		"invalid composite kind `%s`, must be %s",
		compositeKind,
		common.EnumerateWords(
			[]string{
				common.CompositeKindStructure.Name(),
				common.CompositeKindResource.Name(),
				common.CompositeKindEvent.Name(),
				common.CompositeKindContract.Name(),
				common.CompositeKindEnum.Name(),
			},
			"or",
		),
	)
}

func exportDictionaryValue(
	v *interpreter.DictionaryValue,
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Dictionary,
	error,
) {
	dictionary, err := cadence.NewMeteredDictionary(
		inter,
		v.Count(),
		func() ([]cadence.KeyValuePair, error) {
			var err error
			pairs := make([]cadence.KeyValuePair, 0, v.Count())

			v.Iterate(inter, func(key, value interpreter.Value) (resume bool) {

				var convertedKey cadence.Value
				convertedKey, err = exportValueWithInterpreter(
					key,
					inter,
					locationRange,
					seenReferences,
				)
				if err != nil {
					return false
				}

				var convertedValue cadence.Value
				convertedValue, err = exportValueWithInterpreter(
					value,
					inter,
					locationRange,
					seenReferences,
				)
				if err != nil {
					return false
				}

				pairs = append(
					pairs,
					cadence.KeyValuePair{
						Key:   convertedKey,
						Value: convertedValue,
					},
				)

				return true
			})

			if err != nil {
				return nil, err
			}

			return pairs, nil
		},
	)
	if err != nil {
		return cadence.Dictionary{}, err
	}

	exportType := ExportType(v.SemaType(inter), map[sema.TypeID]cadence.Type{}).(cadence.DictionaryType)

	return dictionary.WithType(exportType), err
}

func exportPathValue(gauge common.MemoryGauge, v interpreter.PathValue) cadence.Path {
	domain := v.Domain.Identifier()
	common.UseMemory(gauge, common.MemoryUsage{
		Kind: common.MemoryKindRawString,
		// no need to add 1 to account for empty string: string is metered in Path struct
		Amount: uint64(len(domain)),
	})

	return cadence.NewMeteredPath(
		gauge,
		domain,
		v.Identifier,
	)
}

func exportTypeValue(v interpreter.TypeValue, inter *interpreter.Interpreter) cadence.TypeValue {
	var typ sema.Type
	if v.Type != nil {
		typ = inter.MustConvertStaticToSemaType(v.Type)
	}
	return cadence.NewMeteredTypeValue(
		inter,
		ExportMeteredType(inter, typ, map[sema.TypeID]cadence.Type{}),
	)
}

func exportStorageCapabilityValue(v *interpreter.StorageCapabilityValue, inter *interpreter.Interpreter) cadence.StorageCapability {
	var borrowType sema.Type
	if v.BorrowType != nil {
		borrowType = inter.MustConvertStaticToSemaType(v.BorrowType)
	}

	return cadence.NewMeteredStorageCapability(
		inter,
		exportPathValue(inter, v.Path),
		cadence.NewMeteredAddress(inter, v.Address),
		ExportMeteredType(inter, borrowType, map[sema.TypeID]cadence.Type{}),
	)
}

// exportEvent converts a runtime event to its native Go representation.
func exportEvent(
	gauge common.MemoryGauge,
	event exportableEvent,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Event,
	error,
) {
	exported, err := cadence.NewMeteredEvent(
		gauge,
		len(event.Fields),
		func() ([]cadence.Value, error) {
			fields := make([]cadence.Value, len(event.Fields))

			for i, field := range event.Fields {
				value, err := exportValueWithInterpreter(
					field.Value,
					field.Interpreter(),
					locationRange,
					seenReferences,
				)
				if err != nil {
					return nil, err
				}
				fields[i] = value
			}

			return fields, nil
		},
	)

	if err != nil {
		return cadence.Event{}, err
	}

	eventType := ExportMeteredType(gauge, event.Type, map[sema.TypeID]cadence.Type{}).(*cadence.EventType)

	return exported.WithType(eventType), nil
}

func exportFunctionValue(
	v interpreter.FunctionValue,
	inter *interpreter.Interpreter,
) cadence.Function {
	return cadence.NewMeteredFunction(
		inter,
		ExportMeteredType(inter, v.FunctionType(), map[sema.TypeID]cadence.Type{}).(*cadence.FunctionType),
	)
}

type valueImporter struct {
	inter                  *interpreter.Interpreter
	locationRange          interpreter.LocationRange
	standardLibraryHandler stdlib.StandardLibraryHandler
}

// ImportValue converts a Cadence value to a runtime value.
func ImportValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	standardLibraryHandler stdlib.StandardLibraryHandler,
	value cadence.Value,
	expectedType sema.Type,
) (interpreter.Value, error) {
	return valueImporter{
		inter:                  inter,
		locationRange:          locationRange,
		standardLibraryHandler: standardLibraryHandler,
	}.importValue(value, expectedType)
}

func (i valueImporter) importValue(value cadence.Value, expectedType sema.Type) (interpreter.Value, error) {
	switch v := value.(type) {
	case cadence.Void:
		return interpreter.Void, nil
	case cadence.Optional:
		return i.importOptionalValue(v, expectedType)
	case cadence.Bool:
		return interpreter.AsBoolValue(bool(v)), nil
	case cadence.String:
		return i.importString(v), nil
	case cadence.Character:
		return i.importCharacter(v), nil
	case cadence.Bytes:
		return interpreter.ByteSliceToByteArrayValue(i.inter, v), nil
	case cadence.Address:
		return i.importAddress(v), nil
	case cadence.Int:
		return i.importInt(v), nil
	case cadence.Int8:
		return i.importInt8(v), nil
	case cadence.Int16:
		return i.importInt16(v), nil
	case cadence.Int32:
		return i.importInt32(v), nil
	case cadence.Int64:
		return i.importInt64(v), nil
	case cadence.Int128:
		return i.importInt128(v), nil
	case cadence.Int256:
		return i.importInt256(v), nil
	case cadence.UInt:
		return i.importUInt(v), nil
	case cadence.UInt8:
		return i.importUInt8(v), nil
	case cadence.UInt16:
		return i.importUInt16(v), nil
	case cadence.UInt32:
		return i.importUInt32(v), nil
	case cadence.UInt64:
		return i.importUInt64(v), nil
	case cadence.UInt128:
		return i.importUInt128(v), nil
	case cadence.UInt256:
		return i.importUInt256(v), nil
	case cadence.Word8:
		return i.importWord8(v), nil
	case cadence.Word16:
		return i.importWord16(v), nil
	case cadence.Word32:
		return i.importWord32(v), nil
	case cadence.Word64:
		return i.importWord64(v), nil
	case cadence.Fix64:
		return i.importFix64(v), nil
	case cadence.UFix64:
		return i.importUFix64(v), nil
	case cadence.Path:
		return i.importPathValue(v), nil
	case cadence.Array:
		return i.importArrayValue(v, expectedType)
	case cadence.Dictionary:
		return i.importDictionaryValue(v, expectedType)
	case cadence.Struct:
		return i.importCompositeValue(
			common.CompositeKindStructure,
			v.StructType.Location,
			v.StructType.QualifiedIdentifier,
			v.StructType.Fields,
			v.Fields,
		)
	case cadence.Resource:
		return i.importCompositeValue(
			common.CompositeKindResource,
			v.ResourceType.Location,
			v.ResourceType.QualifiedIdentifier,
			v.ResourceType.Fields,
			v.Fields,
		)
	case cadence.Event:
		return i.importCompositeValue(
			common.CompositeKindEvent,
			v.EventType.Location,
			v.EventType.QualifiedIdentifier,
			v.EventType.Fields,
			v.Fields,
		)
	case cadence.Enum:
		return i.importCompositeValue(
			common.CompositeKindEnum,
			v.EnumType.Location,
			v.EnumType.QualifiedIdentifier,
			v.EnumType.Fields,
			v.Fields,
		)
	case cadence.TypeValue:
		return i.importTypeValue(v.StaticType)
	case cadence.StorageCapability:
		return i.importStorageCapability(
			v.Path,
			v.Address,
			v.BorrowType,
		)
	case cadence.Contract:
		return nil, errors.NewDefaultUserError("cannot import contract")
	case cadence.Function:
		return nil, errors.NewDefaultUserError("cannot import function")
	default:
		// This means the implementation has unhandled types.
		// Hence, return an internal error
		return nil, errors.NewUnexpectedError("cannot import value of type %T", value)
	}
}
func (i valueImporter) importUInt8(v cadence.UInt8) interpreter.UInt8Value {
	return interpreter.NewUInt8Value(
		i.inter,
		func() uint8 {
			return uint8(v)
		},
	)
}

func (i valueImporter) importUInt16(v cadence.UInt16) interpreter.UInt16Value {
	return interpreter.NewUInt16Value(
		i.inter,
		func() uint16 {
			return uint16(v)
		},
	)
}

func (i valueImporter) importUInt32(v cadence.UInt32) interpreter.UInt32Value {
	return interpreter.NewUInt32Value(
		i.inter,
		func() uint32 {
			return uint32(v)
		},
	)
}

func (i valueImporter) importUInt64(v cadence.UInt64) interpreter.UInt64Value {
	return interpreter.NewUInt64Value(
		i.inter,
		func() uint64 {
			return uint64(v)
		},
	)
}

func (i valueImporter) importUInt128(v cadence.UInt128) interpreter.UInt128Value {
	return interpreter.NewUInt128ValueFromBigInt(
		i.inter,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importUInt256(v cadence.UInt256) interpreter.UInt256Value {
	return interpreter.NewUInt256ValueFromBigInt(
		i.inter,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importInt(v cadence.Int) interpreter.IntValue {
	memoryUsage := common.NewBigIntMemoryUsage(
		common.BigIntByteLength(v.Value),
	)
	return interpreter.NewIntValueFromBigInt(
		i.inter,
		memoryUsage,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importInt8(v cadence.Int8) interpreter.Int8Value {
	return interpreter.NewInt8Value(
		i.inter,
		func() int8 {
			return int8(v)
		},
	)
}

func (i valueImporter) importInt16(v cadence.Int16) interpreter.Int16Value {
	return interpreter.NewInt16Value(
		i.inter,
		func() int16 {
			return int16(v)
		},
	)
}

func (i valueImporter) importInt32(v cadence.Int32) interpreter.Int32Value {
	return interpreter.NewInt32Value(
		i.inter,
		func() int32 {
			return int32(v)
		},
	)
}

func (i valueImporter) importInt64(v cadence.Int64) interpreter.Int64Value {
	return interpreter.NewInt64Value(
		i.inter,
		func() int64 {
			return int64(v)
		},
	)
}

func (i valueImporter) importInt128(v cadence.Int128) interpreter.Int128Value {
	return interpreter.NewInt128ValueFromBigInt(
		i.inter,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importInt256(v cadence.Int256) interpreter.Int256Value {
	return interpreter.NewInt256ValueFromBigInt(
		i.inter,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importUInt(v cadence.UInt) interpreter.UIntValue {
	memoryUsage := common.NewBigIntMemoryUsage(
		common.BigIntByteLength(v.Value),
	)
	return interpreter.NewUIntValueFromBigInt(
		i.inter,
		memoryUsage,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importWord8(v cadence.Word8) interpreter.Word8Value {
	return interpreter.NewWord8Value(
		i.inter,
		func() uint8 {
			return uint8(v)
		},
	)
}

func (i valueImporter) importWord16(v cadence.Word16) interpreter.Word16Value {
	return interpreter.NewWord16Value(
		i.inter,
		func() uint16 {
			return uint16(v)
		},
	)
}

func (i valueImporter) importWord32(v cadence.Word32) interpreter.Word32Value {
	return interpreter.NewWord32Value(
		i.inter,
		func() uint32 {
			return uint32(v)
		},
	)
}

func (i valueImporter) importWord64(v cadence.Word64) interpreter.Word64Value {
	return interpreter.NewWord64Value(
		i.inter,
		func() uint64 {
			return uint64(v)
		},
	)
}

func (i valueImporter) importFix64(v cadence.Fix64) interpreter.Fix64Value {
	return interpreter.NewFix64Value(
		i.inter,
		func() int64 {
			return int64(v)
		},
	)
}

func (i valueImporter) importUFix64(v cadence.UFix64) interpreter.UFix64Value {
	return interpreter.NewUFix64Value(
		i.inter,
		func() uint64 {
			return uint64(v)
		},
	)
}

func (i valueImporter) importString(v cadence.String) *interpreter.StringValue {
	memoryUsage := common.NewStringMemoryUsage(len(v))
	return interpreter.NewStringValue(
		i.inter,
		memoryUsage,
		func() string {
			return string(v)
		},
	)
}

func (i valueImporter) importCharacter(v cadence.Character) interpreter.CharacterValue {
	s := string(v)
	memoryUsage := common.NewCharacterMemoryUsage(len(s))
	return interpreter.NewCharacterValue(
		i.inter,
		memoryUsage,
		func() string {
			return s
		},
	)
}

func (i valueImporter) importAddress(v cadence.Address) interpreter.AddressValue {
	return interpreter.NewAddressValue(
		i.inter,
		common.Address(v),
	)
}

func (i valueImporter) importPathValue(v cadence.Path) interpreter.PathValue {
	inter := i.inter

	// meter the Path's Identifier since path is just a container
	common.UseMemory(inter, common.NewRawStringMemoryUsage(len(v.Identifier)))

	return interpreter.NewPathValue(
		inter,
		common.PathDomainFromIdentifier(v.Domain),
		v.Identifier,
	)
}

func (i valueImporter) importTypeValue(v cadence.Type) (interpreter.TypeValue, error) {
	inter := i.inter

	typ := ImportType(inter, v)

	// Creating a static type performs no validation,
	// so in order to be sure the type we have created is legal,
	// we convert it to a sema type.
	//
	// If this fails, the import is invalid

	_, err := inter.ConvertStaticToSemaType(typ)
	if err != nil {
		// unmetered because when err != nil, value should be ignored
		return interpreter.EmptyTypeValue, err
	}

	return interpreter.NewTypeValue(inter, typ), nil
}

func (i valueImporter) importStorageCapability(
	path cadence.Path,
	address cadence.Address,
	borrowType cadence.Type,
) (
	*interpreter.StorageCapabilityValue,
	error,
) {
	_, ok := borrowType.(cadence.ReferenceType)
	if !ok {
		return nil, errors.NewDefaultUserError(
			"cannot import capability: expected reference, got '%s'",
			borrowType.ID(),
		)
	}

	inter := i.inter

	return interpreter.NewStorageCapabilityValue(
		inter,
		interpreter.NewAddressValue(
			inter,
			common.Address(address),
		),
		i.importPathValue(path),
		ImportType(inter, borrowType),
	), nil

}

func (i valueImporter) importOptionalValue(
	v cadence.Optional,
	expectedType sema.Type,
) (
	interpreter.Value,
	error,
) {
	if v.Value == nil {
		return interpreter.Nil, nil
	}

	var innerType sema.Type
	if optionalType, ok := expectedType.(*sema.OptionalType); ok {
		innerType = optionalType.Type
	}

	innerValue, err := i.importValue(v.Value, innerType)
	if err != nil {
		return nil, err
	}

	return interpreter.NewSomeValueNonCopying(i.inter, innerValue), nil
}

func (i valueImporter) importArrayValue(
	v cadence.Array,
	expectedType sema.Type,
) (
	*interpreter.ArrayValue,
	error,
) {
	values := make([]interpreter.Value, len(v.Values))

	var elementType sema.Type
	arrayType, ok := expectedType.(sema.ArrayType)
	if ok {
		elementType = arrayType.ElementType(false)
	}

	inter := i.inter
	locationRange := i.locationRange

	for elementIndex, element := range v.Values {
		value, err := i.importValue(
			element,
			elementType,
		)
		if err != nil {
			return nil, err
		}
		values[elementIndex] = value
	}

	var staticArrayType interpreter.ArrayStaticType
	if arrayType != nil {
		staticArrayType = interpreter.ConvertSemaArrayTypeToStaticArrayType(inter, arrayType)
	} else {
		types := make([]sema.Type, len(v.Values))

		for i, value := range values {
			typ, err := inter.ConvertStaticToSemaType(value.StaticType(inter))
			if err != nil {
				return nil, err
			}
			types[i] = typ
		}

		elementSuperType := sema.LeastCommonSuperType(types...)
		if elementSuperType == sema.InvalidType {
			return nil, errors.NewUnexpectedError("cannot import array: elements do not belong to the same type")
		}

		staticArrayType = interpreter.NewVariableSizedStaticType(
			inter,
			interpreter.ConvertSemaToStaticType(inter, elementSuperType),
		)
	}

	return interpreter.NewArrayValue(
		inter,
		locationRange,
		staticArrayType,
		common.ZeroAddress,
		values...,
	), nil
}

func (i valueImporter) importDictionaryValue(
	v cadence.Dictionary,
	expectedType sema.Type,
) (
	*interpreter.DictionaryValue,
	error,
) {
	keysAndValues := make([]interpreter.Value, len(v.Pairs)*2)

	var keyType sema.Type
	var valueType sema.Type

	dictionaryType, ok := expectedType.(*sema.DictionaryType)
	if ok {
		keyType = dictionaryType.KeyType
		valueType = dictionaryType.ValueType
	}

	inter := i.inter
	locationRange := i.locationRange

	for pairIndex, pair := range v.Pairs {
		key, err := i.importValue(pair.Key, keyType)
		if err != nil {
			return nil, err
		}
		keysAndValues[pairIndex*2] = key

		value, err := i.importValue(pair.Value, valueType)
		if err != nil {
			return nil, err
		}
		keysAndValues[pairIndex*2+1] = value
	}

	var dictionaryStaticType interpreter.DictionaryStaticType
	if dictionaryType != nil {
		dictionaryStaticType = interpreter.ConvertSemaDictionaryTypeToStaticDictionaryType(inter, dictionaryType)
	} else {
		size := len(v.Pairs)
		keyTypes := make([]sema.Type, size)
		valueTypes := make([]sema.Type, size)

		for i := 0; i < size; i++ {
			keyType, err := inter.ConvertStaticToSemaType(keysAndValues[i*2].StaticType(inter))
			if err != nil {
				return nil, err
			}
			keyTypes[i] = keyType

			valueType, err := inter.ConvertStaticToSemaType(keysAndValues[i*2+1].StaticType(inter))
			if err != nil {
				return nil, err
			}
			valueTypes[i] = valueType
		}

		keySuperType := sema.LeastCommonSuperType(keyTypes...)
		valueSuperType := sema.LeastCommonSuperType(valueTypes...)

		if !sema.IsValidDictionaryKeyType(keySuperType) {
			return nil, errors.NewDefaultUserError(
				"cannot import dictionary: keys does not belong to the same type",
			)
		}

		if valueSuperType == sema.InvalidType {
			return nil, errors.NewDefaultUserError("cannot import dictionary: values does not belong to the same type")
		}

		dictionaryStaticType = interpreter.NewDictionaryStaticType(
			inter,
			interpreter.ConvertSemaToStaticType(inter, keySuperType),
			interpreter.ConvertSemaToStaticType(inter, valueSuperType),
		)
	}

	return interpreter.NewDictionaryValue(
		inter,
		locationRange,
		dictionaryStaticType,
		keysAndValues...,
	), nil
}

func (i valueImporter) importCompositeValue(
	kind common.CompositeKind,
	location Location,
	qualifiedIdentifier string,
	fieldTypes []cadence.Field,
	fieldValues []cadence.Value,
) (
	interpreter.Value,
	error,
) {
	var fields []interpreter.CompositeField

	inter := i.inter
	locationRange := i.locationRange

	typeID := common.NewTypeIDFromQualifiedName(inter, location, qualifiedIdentifier)
	compositeType, typeErr := inter.GetCompositeType(location, qualifiedIdentifier, typeID)
	if typeErr != nil {
		return nil, typeErr
	}

	for fieldIndex := 0; fieldIndex < len(fieldTypes) && fieldIndex < len(fieldValues); fieldIndex++ {
		fieldType := fieldTypes[fieldIndex]
		fieldValue := fieldValues[fieldIndex]

		var expectedFieldType sema.Type

		member, ok := compositeType.Members.Get(fieldType.Identifier)
		if ok {
			expectedFieldType = member.TypeAnnotation.Type
		}

		importedFieldValue, err := i.importValue(fieldValue, expectedFieldType)
		if err != nil {
			return nil, err
		}

		fields = append(fields,
			interpreter.NewCompositeField(
				inter,
				fieldType.Identifier,
				importedFieldValue,
			),
		)
	}

	if location == nil {
		switch sema.NativeCompositeTypes[qualifiedIdentifier] {
		case sema.PublicKeyType:
			// PublicKey has a dedicated constructor
			// (e.g. it has computed fields that must be initialized)
			return i.importPublicKey(fields)

		case sema.HashAlgorithmType:
			// HashAlgorithmType has a dedicated constructor
			// (e.g. it has host functions)
			return i.importHashAlgorithm(fields)

		case sema.SignatureAlgorithmType:
			// SignatureAlgorithmType has a dedicated constructor
			// (e.g. it has host functions)
			return i.importSignatureAlgorithm(fields)

		default:
			return nil, errors.NewDefaultUserError(
				"cannot import value of type %s",
				qualifiedIdentifier,
			)
		}
	}

	return interpreter.NewCompositeValue(
		inter,
		locationRange,
		location,
		qualifiedIdentifier,
		kind,
		fields,
		common.ZeroAddress,
	), nil
}

func (i valueImporter) importPublicKey(
	fields []interpreter.CompositeField,
) (
	*interpreter.CompositeValue,
	error,
) {

	var publicKeyValue *interpreter.ArrayValue
	var signAlgoValue *interpreter.SimpleCompositeValue

	ty := sema.PublicKeyType

	for _, field := range fields {
		switch field.Name {
		case sema.PublicKeyTypePublicKeyFieldName:
			arrayValue, ok := field.Value.(*interpreter.ArrayValue)
			if !ok {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

			publicKeyValue = arrayValue

		case sema.PublicKeyTypeSignAlgoFieldName:
			compositeValue, ok := field.Value.(*interpreter.SimpleCompositeValue)
			if !ok {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

			signAlgoValue = compositeValue

		default:
			return nil, errors.NewDefaultUserError(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}

	}

	if publicKeyValue == nil {
		return nil, errors.NewDefaultUserError(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.PublicKeyTypePublicKeyFieldName,
		)
	}

	if signAlgoValue == nil {
		return nil, errors.NewDefaultUserError(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.PublicKeyTypeSignAlgoFieldName,
		)
	}

	return stdlib.NewPublicKeyFromFields(
		i.inter,
		i.locationRange,
		publicKeyValue,
		signAlgoValue,
		i.standardLibraryHandler,
		i.standardLibraryHandler,
		i.standardLibraryHandler,
	), nil
}

func (i valueImporter) importHashAlgorithm(
	fields []interpreter.CompositeField,
) (
	interpreter.MemberAccessibleValue,
	error,
) {

	var foundRawValue bool
	var rawValue interpreter.UInt8Value

	ty := sema.HashAlgorithmType

	for _, field := range fields {
		switch field.Name {
		case sema.EnumRawValueFieldName:
			rawValue, foundRawValue = field.Value.(interpreter.UInt8Value)
			if !foundRawValue {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

		default:
			return nil, errors.NewDefaultUserError(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}
	}

	if !foundRawValue {
		return nil, errors.NewDefaultUserError(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.EnumRawValueFieldName,
		)
	}

	return stdlib.NewHashAlgorithmCase(rawValue, i.standardLibraryHandler)
}

func (valueImporter) importSignatureAlgorithm(
	fields []interpreter.CompositeField,
) (
	interpreter.MemberAccessibleValue,
	error,
) {

	var foundRawValue bool
	var rawValue interpreter.UInt8Value

	ty := sema.SignatureAlgorithmType

	for _, field := range fields {
		switch field.Name {
		case sema.EnumRawValueFieldName:
			rawValue, foundRawValue = field.Value.(interpreter.UInt8Value)
			if !foundRawValue {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type '%s'. invalid value for field '%s': %v",
					ty,
					field.Name,
					field.Value,
				)
			}

		default:
			return nil, errors.NewDefaultUserError(
				"cannot import value of type '%s'. invalid field '%s'",
				ty,
				field.Name,
			)
		}
	}

	if !foundRawValue {
		return nil, errors.NewDefaultUserError(
			"cannot import value of type '%s'. missing field '%s'",
			ty,
			sema.EnumRawValueFieldName,
		)
	}

	caseValue, ok := stdlib.SignatureAlgorithmCaseValues[rawValue]
	if !ok {
		return nil, errors.NewDefaultUserError(
			"unknown SignatureAlgorithm with rawValue %d",
			rawValue,
		)
	}

	return caseValue, nil
}
