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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// SimpleCompositeValue

type SimpleCompositeValue struct {
	staticType      StaticType
	Fields          map[string]Value
	ComputeField    func(name string, context MemberAccessibleContext, locationRange LocationRange) Value
	fieldFormatters map[string]func(common.MemoryGauge, Value, SeenReferences) string
	// stringer is an optional function that is used to produce the string representation of the value.
	// If nil, the FieldNames are used.
	stringer func(ValueStringContext, SeenReferences, LocationRange) string
	TypeID   sema.TypeID
	// FieldNames are the names of the field members (i.e. not functions, and not computed fields), in order
	FieldNames []string

	// This is used for distinguishing between transaction values and other composite values.
	// TODO: maybe cleanup if there is an alternative/better way.
	isTransaction bool

	// privateFields is a property bag to carry internal data
	// that are not visible to cadence users.
	// TODO: any better way to pass down information?
	privateFields map[string]Value
}

var _ Value = &SimpleCompositeValue{}
var _ MemberAccessibleValue = &SimpleCompositeValue{}

func NewSimpleCompositeValue(
	gauge common.MemoryGauge,
	typeID sema.TypeID,
	staticType StaticType,
	fieldNames []string,
	fields map[string]Value,
	computeField func(name string, context MemberAccessibleContext, locationRange LocationRange) Value,
	fieldFormatters map[string]func(common.MemoryGauge, Value, SeenReferences) string,
	stringer func(ValueStringContext, SeenReferences, LocationRange) string,
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

func (v *SimpleCompositeValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitSimpleCompositeValue(interpreter, v)
}

// ForEachField iterates over all field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
func (v *SimpleCompositeValue) ForEachField(
	f func(fieldName string, fieldValue Value) (resume bool),
) {
	for _, fieldName := range v.FieldNames {
		fieldValue := v.Fields[fieldName]
		if !f(fieldName, fieldValue) {
			break
		}
	}
}

// Walk iterates over all field values of the composite value.
// It does NOT walk the computed fields and functions!
func (v *SimpleCompositeValue) Walk(_ ValueWalkContext, walkChild func(Value), _ LocationRange) {
	v.ForEachField(func(_ string, fieldValue Value) (resume bool) {
		walkChild(fieldValue)

		// continue iteration
		return true
	})
}

func (v *SimpleCompositeValue) StaticType(_ ValueStaticTypeContext) StaticType {
	return v.staticType
}

func (v *SimpleCompositeValue) IsImportable(inter *Interpreter, locationRange LocationRange) bool {
	// Check type is importable
	staticType := v.StaticType(inter)
	semaType := MustConvertStaticToSemaType(staticType, inter)
	if !semaType.IsImportable(map[*sema.Member]bool{}) {
		return false
	}

	// Check all field values are importable
	importable := true
	v.ForEachField(func(_ string, value Value) (resume bool) {
		if !value.IsImportable(inter, locationRange) {
			importable = false
			// stop iteration
			return false
		}

		// continue iteration
		return true
	})

	return importable
}

func (v *SimpleCompositeValue) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {

	value, ok := v.Fields[name]
	if ok {
		return value
	}

	computeField := v.ComputeField
	if computeField != nil {
		return computeField(name, context, locationRange)
	}

	return nil
}

func (v *SimpleCompositeValue) RemoveMember(_ *Interpreter, _ LocationRange, name string) Value {
	value := v.Fields[name]
	delete(v.Fields, name)
	return value
}

func (v *SimpleCompositeValue) SetMember(_ MemberAccessibleContext, _ LocationRange, name string, value Value) bool {
	_, hasField := v.Fields[name]
	v.Fields[name] = value
	return hasField
}

func (v *SimpleCompositeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *SimpleCompositeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(NoOpStringContext{}, seenReferences, EmptyLocationRange)
}

func (v *SimpleCompositeValue) MeteredString(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {

	if v.stringer != nil {
		return v.stringer(context, seenReferences, locationRange)
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
				value = fieldFormatter(context, fieldValue, seenReferences)
			}
		}
		if value == "" {
			value = fieldValue.MeteredString(context, seenReferences, locationRange)
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

	common.UseMemory(context, common.NewRawStringMemoryUsage(strLen))

	return format.Composite(typeId, fields)
}

func (v *SimpleCompositeValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {

	for _, fieldName := range v.FieldNames {
		value, ok := v.Fields[fieldName]
		if !ok {
			continue
		}
		if !value.ConformsToStaticType(
			interpreter,
			locationRange,
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

func (v *SimpleCompositeValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v *SimpleCompositeValue) Transfer(
	transferContext ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		RemoveReferencedSlab(transferContext, storable)
	}

	if v.isTransaction {
		panic(NonTransferableValueError{
			Value: v,
		})
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

func (v *SimpleCompositeValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v *SimpleCompositeValue) WithPrivateField(key string, value Value) *SimpleCompositeValue {
	if v.privateFields == nil {
		v.privateFields = make(map[string]Value)
	}

	v.privateFields[key] = value
	return v
}

func (v *SimpleCompositeValue) PrivateField(key string) Value {
	if v.privateFields == nil {
		return nil
	}
	return v.privateFields[key]
}
