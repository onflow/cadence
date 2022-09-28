/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// SimpleCompositeValue

type SimpleCompositeValue struct {
	TypeID     sema.TypeID
	staticType StaticType
	// FieldNames are the names of the field members (i.e. not functions, and not computed fields), in order
	FieldNames      []string
	Fields          map[string]Value
	ComputeField    func(name string, interpreter *Interpreter, getLocationRange func() LocationRange) Value
	fieldFormatters map[string]func(common.MemoryGauge, Value, SeenReferences) string
	// stringer is an optional function that is used to produce the string representation of the value.
	// If nil, the FieldNames are used.
	stringer func(common.MemoryGauge, SeenReferences) string
}

var _ Value = &SimpleCompositeValue{}
var _ MemberAccessibleValue = &SimpleCompositeValue{}

func NewSimpleCompositeValue(
	gauge common.MemoryGauge,
	typeID sema.TypeID,
	staticType StaticType,
	fieldNames []string,
	fields map[string]Value,
	computeField func(name string, interpreter *Interpreter, getLocationRange func() LocationRange) Value,
	fieldFormatters map[string]func(common.MemoryGauge, Value, SeenReferences) string,
	stringer func(common.MemoryGauge, SeenReferences) string,
) *SimpleCompositeValue {

	common.UseMemory(gauge, common.SimpleCompositeValueBaseMemoryUsage)
	common.UseMemory(gauge, common.NewSimpleCompositeMemoryUsage(len(fields)))

	return &SimpleCompositeValue{
		TypeID:          typeID,
		staticType:      staticType,
		FieldNames:      fieldNames,
		Fields:          fields,
		ComputeField:    computeField,
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
func (v *SimpleCompositeValue) ForEachField(_ *Interpreter, f func(fieldName string, fieldValue Value)) {
	for _, fieldName := range v.FieldNames {
		fieldValue := v.Fields[fieldName]
		f(fieldName, fieldValue)
	}
}

// Walk iterates over all field values of the composite value.
// It does NOT walk the computed fields and functions!
func (v *SimpleCompositeValue) Walk(interpreter *Interpreter, walkChild func(Value)) {
	v.ForEachField(interpreter, func(_ string, fieldValue Value) {
		walkChild(fieldValue)
	})
}

func (v *SimpleCompositeValue) StaticType(_ *Interpreter) StaticType {
	return v.staticType
}

func (v *SimpleCompositeValue) IsImportable(inter *Interpreter) bool {
	staticType := v.StaticType(inter)
	semaType := inter.MustConvertStaticToSemaType(staticType)
	return semaType.IsImportable(map[*sema.Member]bool{})
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

	computeField := v.ComputeField
	if computeField != nil {
		return computeField(name, interpreter, getLocationRange)
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
	return v.MeteredString(nil, seenReferences)
}

func (v *SimpleCompositeValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {

	if v.stringer != nil {
		return v.stringer(memoryGauge, seenReferences)
	}

	var fields []struct {
		Name  string
		Value string
	}

	strLen := emptyCompositeStringLen

	for _, fieldName := range v.FieldNames {
		fieldValue := v.Fields[fieldName]

		var value string
		if v.fieldFormatters != nil {
			if fieldFormatter, ok := v.fieldFormatters[fieldName]; ok {
				value = fieldFormatter(memoryGauge, fieldValue, seenReferences)
			}
		}
		if value == "" {
			value = fieldValue.MeteredString(memoryGauge, seenReferences)
		}

		fields = append(fields, struct {
			Name  string
			Value string
		}{
			Name:  fieldName,
			Value: value,
		})

		strLen += len(fieldName)
	}

	typeId := string(v.TypeID)

	// bodyLen = len(fieldNames) + len(typeId) + (n times colon+space) + ((n-1) times comma+space)
	//         = len(fieldNames) + len(typeId) + 2n + 2n - 2
	//         = len(fieldNames) + len(typeId) + 4n - 2
	//
	// Since (-2) only occurs if its non-empty, ignore the (-2). i.e: overestimate
	// 		bodyLen = len(fieldNames) + len(typeId) + 4n
	//
	// Value of each field is metered separately.
	strLen = strLen + len(typeId) + len(fields)*4

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))

	return format.Composite(typeId, fields)
}

func (v *SimpleCompositeValue) ConformsToStaticType(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	results TypeConformanceResults,
) bool {

	for _, fieldName := range v.FieldNames {
		value, ok := v.Fields[fieldName]
		if !ok {
			continue
		}
		if !value.ConformsToStaticType(
			interpreter,
			getLocationRange,
			results,
		) {
			return false
		}
	}

	return true
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

func (v *SimpleCompositeValue) Clone(interpreter *Interpreter) Value {

	clonedFields := make(map[string]Value, len(v.Fields))

	for _, fieldName := range v.FieldNames {
		fieldValue := v.Fields[fieldName]

		clonedFields[fieldName] = fieldValue.Clone(interpreter)
	}

	return &SimpleCompositeValue{
		TypeID:          v.TypeID,
		staticType:      v.staticType,
		FieldNames:      v.FieldNames,
		Fields:          clonedFields,
		ComputeField:    v.ComputeField,
		fieldFormatters: v.fieldFormatters,
		stringer:        v.stringer,
	}
}

func (v *SimpleCompositeValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}
