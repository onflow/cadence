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

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// SimpleCompositeValue

type SimpleCompositeValue struct {
	TypeID      sema.TypeID
	staticType  StaticType
	dynamicType DynamicType
	// FieldNames are the names of the field members (i.e. not functions, and not computed fields), in order
	FieldNames      []string
	Fields          map[string]Value
	ComputedFields  map[string]ComputedField
	fieldFormatters map[string]func(Value, SeenReferences) string
	// stringer is an optional function that is used to produce the string representation of the value.
	// If nil, the FieldNames are used.
	stringer func(SeenReferences) string
}

var _ Value = &SimpleCompositeValue{}
var _ MemberAccessibleValue = &SimpleCompositeValue{}

func NewSimpleCompositeValue(
	typeID sema.TypeID,
	staticType StaticType,
	dynamicType DynamicType,
	fieldNames []string,
	fields map[string]Value,
	computedFields map[string]ComputedField,
	fieldFormatters map[string]func(Value, SeenReferences) string,
	stringer func(SeenReferences) string,
) *SimpleCompositeValue {
	return &SimpleCompositeValue{
		TypeID:          typeID,
		staticType:      staticType,
		dynamicType:     dynamicType,
		FieldNames:      fieldNames,
		Fields:          fields,
		ComputedFields:  computedFields,
		fieldFormatters: fieldFormatters,
		stringer:        stringer,
	}
}

func (*SimpleCompositeValue) IsValue() {}

func (v *SimpleCompositeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitSimpleCompositeValue(interpreter, v)
}

// ForEachField iterates over all field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
//
func (v *SimpleCompositeValue) ForEachField(f func(fieldName string, fieldValue Value)) {
	for _, fieldName := range v.FieldNames {
		fieldValue := v.Fields[fieldName]
		f(fieldName, fieldValue)
	}
}

// Walk iterates over all field values of the composite value.
// It does NOT walk the computed fields and functions!
//
func (v *SimpleCompositeValue) Walk(walkChild func(Value)) {
	v.ForEachField(func(_ string, fieldValue Value) {
		walkChild(fieldValue)
	})
}

func (v *SimpleCompositeValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return v.dynamicType
}

func (v *SimpleCompositeValue) StaticType() StaticType {
	return v.staticType
}

func (v *SimpleCompositeValue) GetMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
) Value {

	value, ok := v.Fields[name]
	if ok {
		return value
	}

	if v.ComputedFields != nil {
		computedField, ok := v.ComputedFields[name]
		if ok {
			return computedField(interpreter, getLocationRange)
		}
	}

	return nil
}

func (*SimpleCompositeValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Simple composite values have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *SimpleCompositeValue) SetMember(_ *Interpreter, _ func() LocationRange, name string, value Value) {
	v.Fields[name] = value
}

func (v *SimpleCompositeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *SimpleCompositeValue) RecursiveString(seenReferences SeenReferences) string {

	if v.stringer != nil {
		return v.stringer(seenReferences)
	}

	var fields []struct {
		Name  string
		Value string
	}

	for _, fieldName := range v.FieldNames {
		fieldValue := v.Fields[fieldName]

		var value string
		if v.fieldFormatters != nil {
			if fieldFormatter, ok := v.fieldFormatters[fieldName]; ok {
				value = fieldFormatter(fieldValue, seenReferences)
			}
		}
		if value == "" {
			value = fieldValue.RecursiveString(seenReferences)
		}

		fields = append(fields, struct {
			Name  string
			Value string
		}{
			Name:  fieldName,
			Value: value,
		})
	}

	return format.Composite(string(v.TypeID), fields)
}

func (v *SimpleCompositeValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	return dynamicType == v.dynamicType
}

func (v *SimpleCompositeValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: v}, nil
}

func (*SimpleCompositeValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (v *SimpleCompositeValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *SimpleCompositeValue) Transfer(
	interpreter *Interpreter,
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *SimpleCompositeValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}
