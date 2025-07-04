/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"strings"
	_ "unsafe"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

// ExportValue converts a runtime value to its native Go representation.
func ExportValue(
	value interpreter.Value,
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
) (cadence.Value, error) {
	return exportValue(
		value,
		context,
		locationRange,
		seenReferences{},
	)
}

// NOTE: Do not generalize to map[interpreter.Value],
// as not all values are Go hashable, i.e. this might lead to run-time panics
type seenReferences map[interpreter.ReferenceValue]struct{}

// exportValue exports the given internal (interpreter) value to an external value.
//
// The export is recursive, the results parameter prevents cycles:
// it is checked at the start of the recursively called function,
// and pre-set before a recursive call.
func exportValue(
	value interpreter.Value,
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {

	switch v := value.(type) {
	case interpreter.VoidValue:
		return cadence.NewMeteredVoid(context), nil
	case interpreter.NilValue:
		return cadence.NewMeteredOptional(context, nil), nil
	case *interpreter.SomeValue:
		return exportSomeValue(v, context, locationRange, seenReferences)
	case interpreter.BoolValue:
		return cadence.NewMeteredBool(context, bool(v)), nil
	case *interpreter.StringValue:
		return cadence.NewMeteredString(
			context,
			common.NewCadenceStringMemoryUsage(len(v.Str)),
			func() string {
				return v.Str
			},
		)
	case interpreter.CharacterValue:
		return cadence.NewMeteredCharacter(
			context,
			common.NewCadenceCharacterMemoryUsage(len(v.Str)),
			func() string {
				return v.Str
			},
		)
	case *interpreter.ArrayValue:
		return exportArrayValue(
			v,
			context,
			locationRange,
			seenReferences,
		)
	case interpreter.IntValue:
		bigInt := v.ToBigInt(context)
		return cadence.NewMeteredIntFromBig(
			context,
			common.NewCadenceIntMemoryUsage(
				common.BigIntByteLength(bigInt),
			),
			func() *big.Int {
				return bigInt
			},
		), nil
	case interpreter.Int8Value:
		return cadence.NewMeteredInt8(context, int8(v)), nil
	case interpreter.Int16Value:
		return cadence.NewMeteredInt16(context, int16(v)), nil
	case interpreter.Int32Value:
		return cadence.NewMeteredInt32(context, int32(v)), nil
	case interpreter.Int64Value:
		return cadence.NewMeteredInt64(context, int64(v)), nil
	case interpreter.Int128Value:
		return cadence.NewMeteredInt128FromBig(
			context,
			func() *big.Int {
				return v.ToBigInt(context)
			},
		)
	case interpreter.Int256Value:
		return cadence.NewMeteredInt256FromBig(
			context,
			func() *big.Int {
				return v.ToBigInt(context)
			},
		)
	case interpreter.UIntValue:
		bigInt := v.ToBigInt(context)
		return cadence.NewMeteredUIntFromBig(
			context,
			common.NewCadenceIntMemoryUsage(
				common.BigIntByteLength(bigInt),
			),
			func() *big.Int {
				return bigInt
			},
		)
	case interpreter.UInt8Value:
		return cadence.NewMeteredUInt8(context, uint8(v)), nil
	case interpreter.UInt16Value:
		return cadence.NewMeteredUInt16(context, uint16(v)), nil
	case interpreter.UInt32Value:
		return cadence.NewMeteredUInt32(context, uint32(v)), nil
	case interpreter.UInt64Value:
		return cadence.NewMeteredUInt64(context, uint64(v)), nil
	case interpreter.UInt128Value:
		return cadence.NewMeteredUInt128FromBig(
			context,
			func() *big.Int {
				return v.ToBigInt(context)
			},
		)
	case interpreter.UInt256Value:
		return cadence.NewMeteredUInt256FromBig(
			context,
			func() *big.Int {
				return v.ToBigInt(context)
			},
		)
	case interpreter.Word8Value:
		return cadence.NewMeteredWord8(context, uint8(v)), nil
	case interpreter.Word16Value:
		return cadence.NewMeteredWord16(context, uint16(v)), nil
	case interpreter.Word32Value:
		return cadence.NewMeteredWord32(context, uint32(v)), nil
	case interpreter.Word64Value:
		return cadence.NewMeteredWord64(context, uint64(v)), nil
	case interpreter.Word128Value:
		return cadence.NewMeteredWord128FromBig(
			context,
			func() *big.Int {
				return v.ToBigInt(context)
			},
		)
	case interpreter.Word256Value:
		return cadence.NewMeteredWord256FromBig(
			context,
			func() *big.Int {
				return v.ToBigInt(context)
			},
		)
	case interpreter.Fix64Value:
		return cadence.Fix64(v), nil
	case interpreter.UFix64Value:
		return cadence.UFix64(v.UFix64Value), nil
	case *interpreter.CompositeValue:
		return exportCompositeValue(
			v,
			context,
			locationRange,
			seenReferences,
		)
	case *interpreter.SimpleCompositeValue:
		return exportCompositeValue(
			v,
			context,
			locationRange,
			seenReferences,
		)
	case *interpreter.DictionaryValue:
		return exportDictionaryValue(
			v,
			context,
			locationRange,
			seenReferences,
		)
	case interpreter.AddressValue:
		return cadence.NewMeteredAddress(context, v), nil
	case interpreter.PathValue:
		return exportPathValue(context, v)
	case interpreter.TypeValue:
		return exportTypeValue(v, context), nil
	case *interpreter.IDCapabilityValue:
		return exportCapabilityValue(v, context)
	case *interpreter.PathCapabilityValue: //nolint:staticcheck
		return exportPathCapabilityValue(v, context)
	case *interpreter.EphemeralReferenceValue:
		if v.Value == nil {
			return nil, nil
		}

		// Break recursion through references
		if _, ok := seenReferences[v]; ok {
			return nil, nil
		}
		defer delete(seenReferences, v)
		seenReferences[v] = struct{}{}

		return exportValue(
			v.Value,
			context,
			locationRange,
			seenReferences,
		)
	case *interpreter.StorageReferenceValue:
		// Break recursion through references
		if _, ok := seenReferences[v]; ok {
			return nil, nil
		}
		defer delete(seenReferences, v)
		seenReferences[v] = struct{}{}

		referencedValue := v.ReferencedValue(context, interpreter.EmptyLocationRange, true)
		if referencedValue == nil {
			return nil, nil
		}

		return exportValue(
			*referencedValue,
			context,
			locationRange,
			seenReferences,
		)
	case interpreter.FunctionValue:
		return exportFunctionValue(v, context), nil
	case nil:
		return nil, nil
	}
	return nil, &ValueNotExportableError{
		Type: value.StaticType(context),
	}
}

func exportSomeValue(
	v *interpreter.SomeValue,
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Optional,
	error,
) {
	innerValue := v.InnerValue()

	if innerValue == nil {
		return cadence.NewMeteredOptional(context, nil), nil
	}

	value, err := exportValue(
		innerValue,
		context,
		locationRange,
		seenReferences,
	)
	if err != nil {
		return cadence.Optional{}, err
	}

	return cadence.NewMeteredOptional(context, value), nil
}

func exportArrayValue(
	v *interpreter.ArrayValue,
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Array,
	error,
) {
	array, err := cadence.NewMeteredArray(
		context,
		v.Count(),
		func() ([]cadence.Value, error) {
			values := make([]cadence.Value, 0, v.Count())

			var err error
			v.Iterate(
				context,
				func(value interpreter.Value) (resume bool) {
					var exportedValue cadence.Value
					exportedValue, err = exportValue(
						value,
						context,
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
				},
				false,
				locationRange,
			)

			if err != nil {
				return nil, err
			}
			return values, nil
		},
	)
	if err != nil {
		return cadence.Array{}, err
	}

	exportType := ExportType(v.SemaType(context), map[sema.TypeID]cadence.Type{}).(cadence.ArrayType)

	return array.WithType(exportType), err
}

func exportCompositeValue(
	v interpreter.Value,
	context interpreter.CompositeValueExportContext,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Value,
	error,
) {

	staticType := v.StaticType(context)

	semaType, err := interpreter.ConvertStaticToSemaType(context, staticType)
	if err != nil {
		return nil, err
	}

	if !semaType.IsExportable(map[*sema.Member]bool{}) {
		return nil, &ValueNotExportableError{
			Type: staticType,
		}
	}

	switch semaType := semaType.(type) {
	case *sema.CompositeType:
		// Continue.
	case *sema.InclusiveRangeType:
		// InclusiveRange is stored as a CompositeValue but isn't a CompositeType.
		return exportCompositeValueAsInclusiveRange(v, semaType, context, locationRange, seenReferences)
	default:
		panic(errors.NewUnreachableError())
	}

	compositeType, ok := semaType.(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// TODO: consider making the results map "global", by moving it up to exportValue
	t := exportCompositeType(context, compositeType, map[sema.TypeID]cadence.Type{})

	// NOTE: use the exported type's fields to ensure fields in type
	// and value are in sync

	fields := getCompositeTypeFields(t)

	makeFieldsValues := func() ([]cadence.Value, error) {
		fieldValues := make([]cadence.Value, len(fields))

		for i, field := range fields {
			fieldName := field.Identifier

			var fieldValue interpreter.Value

			switch v := v.(type) {
			case *interpreter.SimpleCompositeValue:
				fieldValue = v.Fields[fieldName]
				computeField := v.ComputeField

				if fieldValue == nil && computeField != nil {
					fieldValue = computeField(fieldName, context, locationRange)
				}

			case *interpreter.CompositeValue:
				fieldValue = v.GetField(context, fieldName)
				if fieldValue == nil {
					fieldValue = v.GetComputedField(context, locationRange, fieldName)
				}
			}

			exportedFieldValue, err := exportValue(
				fieldValue,
				context,
				locationRange,
				seenReferences,
			)
			if err != nil {
				return nil, err
			}
			fieldValues[i] = exportedFieldValue
		}

		if composite, ok := v.(*interpreter.CompositeValue); ok {
			for _, attachment := range composite.GetAttachments(context, locationRange) {
				exportedAttachmentValue, err := exportValue(
					attachment,
					context,
					locationRange,
					seenReferences,
				)
				if err != nil {
					return nil, err
				}
				fieldValues = append(fieldValues, exportedAttachmentValue)
			}
		}

		return fieldValues, nil
	}

	compositeKind := compositeType.Kind

	// NOTE: when modifying the cases below,
	// also update the error message below!

	switch compositeKind {
	case common.CompositeKindStructure:
		structure, err := cadence.NewMeteredStruct(
			context,
			len(fields),
			makeFieldsValues,
		)
		if err != nil {
			return nil, err
		}
		return structure.WithType(t.(*cadence.StructType)), nil

	case common.CompositeKindResource:
		resource, err := cadence.NewMeteredResource(
			context,
			len(fields),
			makeFieldsValues,
		)
		if err != nil {
			return nil, err
		}
		return resource.WithType(t.(*cadence.ResourceType)), nil

	case common.CompositeKindAttachment:
		attachment, err := cadence.NewMeteredAttachment(
			context,
			len(fields),
			makeFieldsValues,
		)
		if err != nil {
			return nil, err
		}
		return attachment.WithType(t.(*cadence.AttachmentType)), nil

	case common.CompositeKindEvent:
		event, err := cadence.NewMeteredEvent(
			context,
			len(fields),
			makeFieldsValues,
		)
		if err != nil {
			return nil, err
		}
		return event.WithType(t.(*cadence.EventType)), nil

	case common.CompositeKindContract:
		contract, err := cadence.NewMeteredContract(
			context,
			len(fields),
			makeFieldsValues,
		)
		if err != nil {
			return nil, err
		}
		return contract.WithType(t.(*cadence.ContractType)), nil

	case common.CompositeKindEnum:
		enum, err := cadence.NewMeteredEnum(
			context,
			len(fields),
			makeFieldsValues,
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
				common.CompositeKindAttachment.Name(),
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
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Dictionary,
	error,
) {
	dictionary, err := cadence.NewMeteredDictionary(
		context,
		v.Count(),
		func() ([]cadence.KeyValuePair, error) {
			var err error
			pairs := make([]cadence.KeyValuePair, 0, v.Count())

			v.Iterate(
				context,
				locationRange,
				func(key, value interpreter.Value) (resume bool) {

					var convertedKey cadence.Value
					convertedKey, err = exportValue(
						key,
						context,
						locationRange,
						seenReferences,
					)
					if err != nil {
						return false
					}

					var convertedValue cadence.Value
					convertedValue, err = exportValue(
						value,
						context,
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
				},
			)

			if err != nil {
				return nil, err
			}

			return pairs, nil
		},
	)
	if err != nil {
		return cadence.Dictionary{}, err
	}

	exportType := ExportType(v.SemaType(context), map[sema.TypeID]cadence.Type{}).(*cadence.DictionaryType)

	return dictionary.WithType(exportType), err
}

func exportCompositeValueAsInclusiveRange(
	v interpreter.Value,
	inclusiveRangeType *sema.InclusiveRangeType,
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	*cadence.InclusiveRange,
	error,
) {
	compositeValue, ok := v.(*interpreter.CompositeValue)
	if !ok {
		// InclusiveRange is stored as a CompositeValue.
		panic(errors.NewUnreachableError())
	}

	getNonComputedField := func(fieldName string) (cadence.Value, error) {
		fieldValue := compositeValue.GetField(context, fieldName)
		if fieldValue == nil {
			// Bug if the field is absent.
			panic(errors.NewUnreachableError())
		}

		return exportValue(
			fieldValue,
			context,
			locationRange,
			seenReferences,
		)
	}

	startValue, err := getNonComputedField(sema.InclusiveRangeTypeStartFieldName)
	if err != nil {
		return &cadence.InclusiveRange{}, err
	}

	endValue, err := getNonComputedField(sema.InclusiveRangeTypeEndFieldName)
	if err != nil {
		return &cadence.InclusiveRange{}, err
	}

	stepValue, err := getNonComputedField(sema.InclusiveRangeTypeStepFieldName)
	if err != nil {
		return &cadence.InclusiveRange{}, err
	}

	inclusiveRange := cadence.NewMeteredInclusiveRange(
		context,
		startValue,
		endValue,
		stepValue,
	)

	t := exportInclusiveRangeType(context, inclusiveRangeType, map[sema.TypeID]cadence.Type{})
	return inclusiveRange.WithType(t), err
}

func exportPathValue(gauge common.MemoryGauge, v interpreter.PathValue) (cadence.Path, error) {
	return cadence.NewMeteredPath(
		gauge,
		v.Domain,
		v.Identifier,
	)
}

func exportTypeValue(v interpreter.TypeValue, converter interpreter.TypeConverter) cadence.TypeValue {
	var typ sema.Type
	if v.Type != nil {
		typ = interpreter.MustConvertStaticToSemaType(v.Type, converter)
	}
	return cadence.NewMeteredTypeValue(
		converter,
		ExportMeteredType(converter, typ, map[sema.TypeID]cadence.Type{}),
	)
}

func exportCapabilityValue(
	v *interpreter.IDCapabilityValue,
	typeConverter interpreter.TypeConverter,
) (cadence.Capability, error) {
	borrowType := interpreter.MustConvertStaticToSemaType(v.BorrowType, typeConverter)
	exportedBorrowType := ExportMeteredType(typeConverter, borrowType, map[sema.TypeID]cadence.Type{})

	return cadence.NewMeteredCapability(
		typeConverter,
		cadence.NewMeteredUInt64(typeConverter, uint64(v.ID)),
		cadence.NewMeteredAddress(typeConverter, v.Address()),
		exportedBorrowType,
	), nil
}

func exportPathCapabilityValue(
	v *interpreter.PathCapabilityValue, //nolint:staticcheck
	typeConverter interpreter.TypeConverter,
) (cadence.Capability, error) {
	var exportedBorrowType cadence.Type

	if v.BorrowType != nil {
		borrowType := interpreter.MustConvertStaticToSemaType(v.BorrowType, typeConverter)
		exportedBorrowType = ExportMeteredType(typeConverter, borrowType, map[sema.TypeID]cadence.Type{})
	}

	capability := cadence.NewMeteredCapability(
		typeConverter,
		cadence.NewMeteredUInt64(typeConverter, uint64(interpreter.InvalidCapabilityID)),
		cadence.NewMeteredAddress(typeConverter, v.Address()),
		exportedBorrowType,
	)

	path, err := exportPathValue(typeConverter, v.Path)
	if err != nil {
		return cadence.Capability{}, err
	}
	capability.DeprecatedPath = &path

	return capability, nil
}

// exportEvent converts a runtime event to its native Go representation.
func exportEvent(
	context interpreter.ValueExportContext,
	event exportableEvent,
	locationRange interpreter.LocationRange,
	seenReferences seenReferences,
) (
	cadence.Event,
	error,
) {
	exported, err := cadence.NewMeteredEvent(
		context,
		len(event.Fields),
		func() ([]cadence.Value, error) {
			fields := make([]cadence.Value, len(event.Fields))

			for i, field := range event.Fields {
				value, err := exportValue(
					field,
					context,
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

	eventType := ExportMeteredType(context, event.Type, map[sema.TypeID]cadence.Type{}).(*cadence.EventType)

	return exported.WithType(eventType), nil
}

func exportFunctionValue(
	v interpreter.FunctionValue,
	context interpreter.ValueStaticTypeContext,
) cadence.Function {
	return cadence.NewMeteredFunction(
		context,
		ExportMeteredType(
			context,
			v.FunctionType(context),
			map[sema.TypeID]cadence.Type{},
		).(*cadence.FunctionType),
	)
}

type ValueImportContext interface {
	common.MemoryGauge
	interpreter.ArrayCreationContext
	interpreter.MemberAccessibleContext
}

type valueImporter struct {
	context                ValueImportContext
	locationRange          interpreter.LocationRange
	standardLibraryHandler stdlib.StandardLibraryHandler
	resolveLocation        sema.LocationHandlerFunc
}

// ImportValue converts a Cadence value to a runtime value.
func ImportValue(
	context ValueImportContext,
	locationRange interpreter.LocationRange,
	standardLibraryHandler stdlib.StandardLibraryHandler,
	resolveLocation sema.LocationHandlerFunc,
	value cadence.Value,
	expectedType sema.Type,
) (interpreter.Value, error) {
	return valueImporter{
		context:                context,
		locationRange:          locationRange,
		standardLibraryHandler: standardLibraryHandler,
		resolveLocation:        resolveLocation,
	}.importValue(value, expectedType)
}

//go:linkname getCompositeFieldValues github.com/onflow/cadence.getCompositeFieldValues
func getCompositeFieldValues(cadence.Composite) []cadence.Value

//go:linkname getCompositeTypeFields github.com/onflow/cadence.getCompositeTypeFields
func getCompositeTypeFields(cadence.CompositeType) []cadence.Field

func (i valueImporter) importValue(value cadence.Value, expectedType sema.Type) (interpreter.Value, error) {
	switch v := value.(type) {
	case cadence.Void:
		return interpreter.Void, nil
	case cadence.Optional:
		return i.importOptionalValue(v, expectedType)
	case cadence.Bool:
		return interpreter.BoolValue(v), nil
	case cadence.String:
		return i.importString(v), nil
	case cadence.Character:
		return i.importCharacter(v), nil
	case cadence.Bytes:
		return interpreter.ByteSliceToByteArrayValue(i.context, v), nil
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
	case cadence.Word128:
		return i.importWord128(v), nil
	case cadence.Word256:
		return i.importWord256(v), nil
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
			getCompositeTypeFields(v.StructType),
			getCompositeFieldValues(v),
		)
	case cadence.Resource:
		return i.importCompositeValue(
			common.CompositeKindResource,
			v.ResourceType.Location,
			v.ResourceType.QualifiedIdentifier,
			getCompositeTypeFields(v.ResourceType),
			getCompositeFieldValues(v),
		)
	case cadence.Event:
		return i.importCompositeValue(
			common.CompositeKindEvent,
			v.EventType.Location,
			v.EventType.QualifiedIdentifier,
			getCompositeTypeFields(v.EventType),
			getCompositeFieldValues(v),
		)
	case cadence.Enum:
		return i.importCompositeValue(
			common.CompositeKindEnum,
			v.EnumType.Location,
			v.EnumType.QualifiedIdentifier,
			getCompositeTypeFields(v.EnumType),
			getCompositeFieldValues(v),
		)
	case *cadence.InclusiveRange:
		return i.importInclusiveRangeValue(v, expectedType)
	case cadence.TypeValue:
		return i.importTypeValue(v.StaticType)
	case cadence.Capability:
		return i.importCapability(
			v.ID,
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
		i.context,
		func() uint8 {
			return uint8(v)
		},
	)
}

func (i valueImporter) importUInt16(v cadence.UInt16) interpreter.UInt16Value {
	return interpreter.NewUInt16Value(
		i.context,
		func() uint16 {
			return uint16(v)
		},
	)
}

func (i valueImporter) importUInt32(v cadence.UInt32) interpreter.UInt32Value {
	return interpreter.NewUInt32Value(
		i.context,
		func() uint32 {
			return uint32(v)
		},
	)
}

func (i valueImporter) importUInt64(v cadence.UInt64) interpreter.UInt64Value {
	return interpreter.NewUInt64Value(
		i.context,
		func() uint64 {
			return uint64(v)
		},
	)
}

func (i valueImporter) importUInt128(v cadence.UInt128) interpreter.UInt128Value {
	return interpreter.NewUInt128ValueFromBigInt(
		i.context,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importUInt256(v cadence.UInt256) interpreter.UInt256Value {
	return interpreter.NewUInt256ValueFromBigInt(
		i.context,
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
		i.context,
		memoryUsage,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importInt8(v cadence.Int8) interpreter.Int8Value {
	return interpreter.NewInt8Value(
		i.context,
		func() int8 {
			return int8(v)
		},
	)
}

func (i valueImporter) importInt16(v cadence.Int16) interpreter.Int16Value {
	return interpreter.NewInt16Value(
		i.context,
		func() int16 {
			return int16(v)
		},
	)
}

func (i valueImporter) importInt32(v cadence.Int32) interpreter.Int32Value {
	return interpreter.NewInt32Value(
		i.context,
		func() int32 {
			return int32(v)
		},
	)
}

func (i valueImporter) importInt64(v cadence.Int64) interpreter.Int64Value {
	return interpreter.NewInt64Value(
		i.context,
		func() int64 {
			return int64(v)
		},
	)
}

func (i valueImporter) importInt128(v cadence.Int128) interpreter.Int128Value {
	return interpreter.NewInt128ValueFromBigInt(
		i.context,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importInt256(v cadence.Int256) interpreter.Int256Value {
	return interpreter.NewInt256ValueFromBigInt(
		i.context,
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
		i.context,
		memoryUsage,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importWord8(v cadence.Word8) interpreter.Word8Value {
	return interpreter.NewWord8Value(
		i.context,
		func() uint8 {
			return uint8(v)
		},
	)
}

func (i valueImporter) importWord16(v cadence.Word16) interpreter.Word16Value {
	return interpreter.NewWord16Value(
		i.context,
		func() uint16 {
			return uint16(v)
		},
	)
}

func (i valueImporter) importWord32(v cadence.Word32) interpreter.Word32Value {
	return interpreter.NewWord32Value(
		i.context,
		func() uint32 {
			return uint32(v)
		},
	)
}

func (i valueImporter) importWord64(v cadence.Word64) interpreter.Word64Value {
	return interpreter.NewWord64Value(
		i.context,
		func() uint64 {
			return uint64(v)
		},
	)
}

func (i valueImporter) importWord128(v cadence.Word128) interpreter.Word128Value {
	return interpreter.NewWord128ValueFromBigInt(
		i.context,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importWord256(v cadence.Word256) interpreter.Word256Value {
	return interpreter.NewWord256ValueFromBigInt(
		i.context,
		func() *big.Int {
			return v.Value
		},
	)
}

func (i valueImporter) importFix64(v cadence.Fix64) interpreter.Fix64Value {
	return interpreter.NewFix64Value(
		i.context,
		func() int64 {
			return int64(v)
		},
	)
}

func (i valueImporter) importUFix64(v cadence.UFix64) interpreter.UFix64Value {
	return interpreter.NewUFix64Value(
		i.context,
		func() uint64 {
			return uint64(v)
		},
	)
}

func (i valueImporter) importString(v cadence.String) *interpreter.StringValue {
	memoryUsage := common.NewStringMemoryUsage(len(v))
	return interpreter.NewStringValue(
		i.context,
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
		i.context,
		memoryUsage,
		func() string {
			return s
		},
	)
}

func (i valueImporter) importAddress(v cadence.Address) interpreter.AddressValue {
	return interpreter.NewAddressValue(
		i.context,
		common.Address(v),
	)
}

func (i valueImporter) importPathValue(v cadence.Path) interpreter.PathValue {
	inter := i.context

	// meter the Path's Identifier since path is just a container
	common.UseMemory(inter, common.NewRawStringMemoryUsage(len(v.Identifier)))

	return interpreter.NewPathValue(
		inter,
		v.Domain,
		v.Identifier,
	)
}

func (i valueImporter) importTypeValue(v cadence.Type) (interpreter.TypeValue, error) {
	inter := i.context

	typ := ImportType(inter, v)

	// Creating a static type performs no validation,
	// so in order to be sure the type we have created is legal,
	// we convert it to a sema type.
	//
	// If this fails, the import is invalid

	_, err := interpreter.ConvertStaticToSemaType(inter, typ)
	if err != nil {
		// unmetered because when err != nil, value should be ignored
		return interpreter.EmptyTypeValue, err
	}

	return interpreter.NewTypeValue(inter, typ), nil
}

func (i valueImporter) importCapability(
	id cadence.UInt64,
	address cadence.Address,
	borrowType cadence.Type,
) (
	*interpreter.IDCapabilityValue,
	error,
) {
	_, ok := borrowType.(*cadence.ReferenceType)
	if !ok {
		return nil, errors.NewDefaultUserError(
			"cannot import capability: expected reference, got '%s'",
			borrowType.ID(),
		)
	}

	inter := i.context

	addressValue := interpreter.NewAddressValue(
		inter,
		common.Address(address),
	)

	return interpreter.NewCapabilityValue(
		inter,
		i.importUInt64(id),
		addressValue,
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

	return interpreter.NewSomeValueNonCopying(i.context, innerValue), nil
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

	inter := i.context
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
			typ, err := interpreter.ConvertStaticToSemaType(inter, value.StaticType(inter))
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

	inter := i.context
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

	var dictionaryStaticType *interpreter.DictionaryStaticType
	if dictionaryType != nil {
		dictionaryStaticType = interpreter.ConvertSemaDictionaryTypeToStaticDictionaryType(inter, dictionaryType)
	} else {
		size := len(v.Pairs)
		keyTypes := make([]sema.Type, size)
		valueTypes := make([]sema.Type, size)

		for i := 0; i < size; i++ {
			keyType, err := interpreter.ConvertStaticToSemaType(inter, keysAndValues[i*2].StaticType(inter))
			if err != nil {
				return nil, err
			}
			keyTypes[i] = keyType

			valueType, err := interpreter.ConvertStaticToSemaType(inter, keysAndValues[i*2+1].StaticType(inter))
			if err != nil {
				return nil, err
			}
			valueTypes[i] = valueType
		}

		keySuperType := sema.LeastCommonSuperType(keyTypes...)
		valueSuperType := sema.LeastCommonSuperType(valueTypes...)

		if !sema.IsSubType(keySuperType, sema.HashableStructType) {
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

func (i valueImporter) importInclusiveRangeValue(
	v *cadence.InclusiveRange,
	expectedType sema.Type,
) (
	*interpreter.CompositeValue,
	error,
) {

	var memberType sema.Type

	inclusiveRangeType, ok := expectedType.(*sema.InclusiveRangeType)
	if ok {
		memberType = inclusiveRangeType.MemberType
	}

	inter := i.context
	locationRange := i.locationRange

	// start, end, and step. The order matters.
	members := make([]interpreter.IntegerValue, 3)

	// import members.
	for index, value := range []cadence.Value{v.Start, v.End, v.Step} {
		importedValue, err := i.importValue(value, memberType)
		if err != nil {
			return nil, err
		}
		importedIntegerValue, ok := importedValue.(interpreter.IntegerValue)
		if !ok {
			return nil, errors.NewDefaultUserError(
				"cannot import InclusiveRange: start, end and step must be integers",
			)
		}

		members[index] = importedIntegerValue
	}

	startValue := members[0]
	endValue := members[1]
	stepValue := members[2]

	startType := startValue.StaticType(inter)

	if inclusiveRangeType == nil {
		memberSemaType, err := interpreter.ConvertStaticToSemaType(inter, startType)
		if err != nil {
			return nil, err
		}

		memberType = memberSemaType
		inclusiveRangeType = sema.NewInclusiveRangeType(
			inter,
			memberType,
		)
	}

	inclusiveRangeStaticType, ok := interpreter.ConvertSemaToStaticType(
		inter,
		inclusiveRangeType,
	).(interpreter.InclusiveRangeStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Ensure that start, end and step have the same static type.
	// Usually this validation would be done outside of this function in ConformsToStaticType but
	// we do it here because the NewInclusiveRangeValueWithStep constructor performs validations
	// which involve comparisons between these values and hence they need to be of the same static
	// type.

	if !startType.Equal(endValue.StaticType(inter)) ||
		!startType.Equal(stepValue.StaticType(inter)) {

		return nil, errors.NewDefaultUserError(
			"cannot import InclusiveRange: start, end and step must be of the same type",
		)
	}

	return interpreter.NewInclusiveRangeValueWithStep(
		inter,
		locationRange,
		startValue,
		endValue,
		stepValue,
		inclusiveRangeStaticType,
		inclusiveRangeType,
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

	inter := i.context
	locationRange := i.locationRange

	// Resolve the location if it is not nil (not a built-in type)

	if location != nil {
		resolveLocation := i.resolveLocation
		if resolveLocation != nil {

			rootIdentifier := strings.SplitN(qualifiedIdentifier, ".", 2)[0]

			resolvedLocations, err := resolveLocation(
				[]ast.Identifier{{Identifier: rootIdentifier}},
				location,
			)
			if err != nil {
				return nil, err
			}

			if len(resolvedLocations) != 1 {
				return nil, errors.NewDefaultUserError(
					"cannot import value of type %s: location resolution failed",
					qualifiedIdentifier,
				)
			}

			location = resolvedLocations[0].Location
		}
	}

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

		case sema.PublicKeyTypeSignatureAlgorithmFieldName:
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
			sema.PublicKeyTypeSignatureAlgorithmFieldName,
		)
	}

	return stdlib.NewPublicKeyFromFields(
		i.context,
		i.locationRange,
		publicKeyValue,
		signAlgoValue,
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
