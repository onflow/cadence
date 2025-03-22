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

package interpreter

import (
	"strings"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// TypeValue

type TypeValue struct {
	// Optional. nil represents "unknown"/"invalid" type
	Type StaticType
}

var EmptyTypeValue = TypeValue{}

var _ Value = TypeValue{}
var _ atree.Storable = TypeValue{}
var _ EquatableValue = TypeValue{}
var _ MemberAccessibleValue = TypeValue{}

func NewUnmeteredTypeValue(t StaticType) TypeValue {
	return TypeValue{Type: t}
}

func NewTypeValue(
	memoryGauge common.MemoryGauge,
	staticType StaticType,
) TypeValue {
	common.UseMemory(memoryGauge, common.TypeValueMemoryUsage)
	return NewUnmeteredTypeValue(staticType)
}

func (TypeValue) isValue() {}

func (v TypeValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitTypeValue(interpreter, v)
}

func (TypeValue) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (TypeValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeMetaType)
}

func (TypeValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return sema.MetaType.Importable
}

func (v TypeValue) String() string {
	var typeString string
	staticType := v.Type
	if staticType != nil {
		typeString = staticType.String()
	}

	return format.TypeValue(typeString)
}

func (v TypeValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v TypeValue) MeteredString(interpreter *Interpreter, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(interpreter, common.TypeValueStringMemoryUsage)

	var typeString string
	if v.Type != nil {
		typeString = v.Type.MeteredString(interpreter)
	}

	return format.TypeValue(typeString)
}

func (v TypeValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherTypeValue, ok := other.(TypeValue)
	if !ok {
		return false
	}

	// Unknown types are never equal to another type

	staticType := v.Type
	otherStaticType := otherTypeValue.Type

	if staticType == nil || otherStaticType == nil {
		return false
	}

	return staticType.Equal(otherStaticType)
}

func (v TypeValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.MetaTypeIdentifierFieldName:
		var typeID string
		staticType := v.Type
		if staticType != nil {
			typeID = string(staticType.ID())
		}
		memoryUsage := common.NewStringMemoryUsage(len(typeID))
		return NewStringValue(interpreter, memoryUsage, func() string {
			return typeID
		})

	case sema.MetaTypeIsSubtypeFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.MetaTypeIsSubtypeFunctionType,
			func(v TypeValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter

				staticType := v.Type
				otherTypeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				otherStaticType := otherTypeValue.Type

				// if either type is unknown, the subtype relation is false, as it doesn't make sense to even ask this question
				if staticType == nil || otherStaticType == nil {
					return FalseValue
				}

				result := sema.IsSubType(
					MustConvertStaticToSemaType(staticType, interpreter),
					MustConvertStaticToSemaType(otherStaticType, interpreter),
				)
				return AsBoolValue(result)
			},
		)

	case sema.MetaTypeIsRecoveredFieldName:
		staticType := v.Type
		if staticType == nil {
			return FalseValue
		}

		location, _, err := common.DecodeTypeID(interpreter, string(staticType.ID()))
		if err != nil || location == nil {
			return FalseValue
		}

		elaboration := interpreter.getElaboration(location)
		if elaboration == nil {
			return FalseValue
		}

		return AsBoolValue(elaboration.IsRecovered)

	case sema.MetaTypeAddressFieldName:
		staticType := v.Type
		if staticType == nil {
			return Nil
		}

		var location common.Location

		switch staticType := staticType.(type) {
		case *CompositeStaticType:
			location = staticType.Location

		case *InterfaceStaticType:
			location = staticType.Location

		default:
			return Nil
		}

		addressLocation, ok := location.(common.AddressLocation)
		if !ok {
			return Nil
		}

		addressValue := NewAddressValue(
			interpreter,
			addressLocation.Address,
		)
		return NewSomeValueNonCopying(
			interpreter,
			addressValue,
		)

	case sema.MetaTypeContractNameFieldName:
		staticType := v.Type
		if staticType == nil {
			return Nil
		}

		var location common.Location
		var qualifiedIdentifier string

		switch staticType := staticType.(type) {
		case *CompositeStaticType:
			location = staticType.Location
			qualifiedIdentifier = staticType.QualifiedIdentifier

		case *InterfaceStaticType:
			location = staticType.Location
			qualifiedIdentifier = staticType.QualifiedIdentifier

		default:
			return Nil
		}

		switch location.(type) {
		case common.AddressLocation,
			common.StringLocation:

			separatorIndex := strings.Index(qualifiedIdentifier, ".")
			contractNameLength := len(qualifiedIdentifier)
			if separatorIndex >= 0 {
				contractNameLength = separatorIndex
			}

			contractNameValue := NewStringValue(
				interpreter,
				common.NewStringMemoryUsage(contractNameLength),
				func() string {
					return qualifiedIdentifier[0:contractNameLength]
				},
			)

			return NewSomeValueNonCopying(interpreter, contractNameValue)

		default:
			return Nil
		}

	}

	return nil
}

func (TypeValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Types have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (TypeValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Types have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v TypeValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v TypeValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return maybeLargeImmutableStorable(
		v,
		storage,
		address,
		maxInlineSize,
	)
}

func (TypeValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (TypeValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v TypeValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		context.RemoveReferencedSlab(storable)
	}
	return v
}

func (v TypeValue) Clone(_ *Interpreter) Value {
	return v
}

func (TypeValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v TypeValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v TypeValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (TypeValue) ChildStorables() []atree.Storable {
	return nil
}

// HashInput returns a byte slice containing:
// - HashInputTypeType (1 byte)
// - type id (n bytes)
func (v TypeValue) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	typeID := v.Type.ID()

	length := 1 + len(typeID)
	var buf []byte
	if length <= len(scratch) {
		buf = scratch[:length]
	} else {
		buf = make([]byte, length)
	}

	buf[0] = byte(HashInputTypeType)
	copy(buf[1:], typeID)
	return buf
}
