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
	goerrors "errors"
	"strings"
	"time"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// CompositeValue

type FunctionOrderedMap = orderedmap.OrderedMap[string, FunctionValue]

type CompositeValue struct {
	Location common.Location

	// note that the staticType is not guaranteed to be a CompositeStaticType as there can be types
	// which are non-composite but their values are treated as CompositeValue.
	// For e.g. InclusiveRangeValue
	staticType StaticType

	Stringer        func(gauge common.MemoryGauge, value *CompositeValue, seenReferences SeenReferences) string
	injectedFields  map[string]Value
	computedFields  map[string]ComputedField
	NestedVariables map[string]Variable
	Functions       *FunctionOrderedMap
	dictionary      *atree.OrderedMap
	typeID          TypeID

	// attachments also have a reference to their base value. This field is set in three cases:
	// 1) when an attachment `A` is accessed off `v` using `v[A]`, this is set to `&v`
	// 2) When a resource `r`'s destructor is invoked, all of `r`'s attachments' destructors will also run, and
	//    have their `base` fields set to `&r`
	// 3) When a value is transferred, this field is copied between its attachments
	base                *CompositeValue
	QualifiedIdentifier string
	Kind                common.CompositeKind
	isDestroyed         bool
}

type ComputedField func(ValueTransferContext, *CompositeValue) Value

type CompositeField struct {
	Value Value
	Name  string
}

const unrepresentableNamePrefix = "$"
const resourceDefaultDestroyEventPrefix = ast.ResourceDestructionDefaultEventName + unrepresentableNamePrefix

var _ TypeIndexableValue = &CompositeValue{}
var _ IterableValue = &CompositeValue{}

func NewCompositeField(memoryGauge common.MemoryGauge, name string, value Value) CompositeField {
	common.UseMemory(memoryGauge, common.CompositeFieldMemoryUsage)
	return NewUnmeteredCompositeField(name, value)
}

func NewUnmeteredCompositeField(name string, value Value) CompositeField {
	return CompositeField{
		Name:  name,
		Value: value,
	}
}

// Create a CompositeValue with the provided StaticType.
// Useful when we wish to utilize CompositeValue as the value
// for a type which isn't CompositeType.
// For e.g. InclusiveRangeType
func NewCompositeValueWithStaticType(
	context MemberAccessibleContext,
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
	fields []CompositeField,
	address common.Address,
	staticType StaticType,
) *CompositeValue {
	value := NewCompositeValue(
		context,
		location,
		qualifiedIdentifier,
		kind,
		fields,
		address,
	)
	value.staticType = staticType
	return value
}

func NewCompositeValue(
	context MemberAccessibleContext,
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
	fields []CompositeField,
	address common.Address,
) *CompositeValue {

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindCreateCompositeValue,
			Intensity: 1,
		},
	)

	var v *CompositeValue

	if TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			valueID := v.ValueID().String()
			typeID := string(v.TypeID())
			kind := v.Kind.String()

			context.ReportCompositeValueConstructTrace(
				valueID,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	typeInfo := NewCompositeTypeInfo(
		context,
		location,
		qualifiedIdentifier,
		kind,
	)

	constructor := func() (dictionary *atree.OrderedMap) {

		if TracingEnabled {
			startTime := time.Now()

			defer func() {
				valueID := dictionary.ValueID().String()
				typeID := string(common.NewTypeIDFromQualifiedName(
					context,
					location,
					qualifiedIdentifier,
				))
				seed := dictionary.Seed()

				context.ReportAtreeNewMapTrace(
					valueID,
					typeID,
					seed,
					time.Since(startTime),
				)
			}()
		}

		common.UseComputation(
			context,
			common.ComputationUsage{
				Kind:      common.ComputationKindAtreeMapConstruction,
				Intensity: 1,
			},
		)

		var err error
		dictionary, err = atree.NewMap(
			context.Storage(),
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			typeInfo,
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return dictionary
	}

	v = newCompositeValueFromConstructor(context, uint64(len(fields)), typeInfo, constructor)

	for _, field := range fields {
		v.SetMember(
			context,
			field.Name,
			field.Value,
		)
	}

	return v
}

func newCompositeValueFromConstructor(
	gauge common.MemoryGauge,
	count uint64,
	typeInfo CompositeTypeInfo,
	constructor func() *atree.OrderedMap,
) *CompositeValue {

	elementOverhead, dataUse, metaDataUse :=
		common.NewAtreeMapMemoryUsages(count, 0)
	common.UseMemory(gauge, elementOverhead)
	common.UseMemory(gauge, dataUse)
	common.UseMemory(gauge, metaDataUse)

	return NewCompositeValueFromAtreeMap(
		gauge,
		typeInfo,
		constructor(),
	)
}

func NewCompositeValueFromAtreeMap(
	gauge common.MemoryGauge,
	typeInfo CompositeTypeInfo,
	atreeOrderedMap *atree.OrderedMap,
) *CompositeValue {

	common.UseMemory(gauge, common.CompositeValueBaseMemoryUsage)

	return &CompositeValue{
		dictionary:          atreeOrderedMap,
		Location:            typeInfo.Location,
		QualifiedIdentifier: typeInfo.QualifiedIdentifier,
		Kind:                typeInfo.Kind,
	}
}

var _ Value = &CompositeValue{}
var _ EquatableValue = &CompositeValue{}
var _ HashableValue = &CompositeValue{}
var _ MemberAccessibleValue = &CompositeValue{}
var _ ReferenceTrackedResourceKindedValue = &CompositeValue{}
var _ ContractValue = &CompositeValue{}
var _ atree.Value = &CompositeValue{}
var _ atree.WrapperValue = &CompositeValue{}
var _ atreeContainerBackedValue = &CompositeValue{}

func (*CompositeValue) IsValue() {}

func (*CompositeValue) isAtreeContainerBackedValue() {}

func (v *CompositeValue) Accept(context ValueVisitContext, visitor Visitor) {
	descend := visitor.VisitCompositeValue(context, v)
	if !descend {
		return
	}

	v.ForEachField(
		context,
		func(_ string, value Value) (resume bool) {
			value.Accept(context, visitor)

			// continue iteration
			return true
		},
	)
}

// Walk iterates over all field values of the composite value.
// It does NOT walk the computed field or functions!
func (v *CompositeValue) Walk(context ValueWalkContext, walkChild func(Value)) {
	v.ForEachField(
		context,
		func(_ string, value Value) (resume bool) {
			walkChild(value)

			// continue iteration
			return true
		},
	)
}

func (v *CompositeValue) StaticType(context ValueStaticTypeContext) StaticType {
	if v.staticType == nil {
		// NOTE: Instead of using NewCompositeStaticType, which always generates the type ID,
		// use the TypeID accessor, which may return an already computed type ID
		v.staticType = NewCompositeStaticType(
			context,
			v.Location,
			v.QualifiedIdentifier,
			v.TypeID(),
		)
	}
	return v.staticType
}

func (v *CompositeValue) IsImportable(context ValueImportableContext) bool {
	// Check type is importable
	staticType := v.StaticType(context)
	semaType := context.SemaTypeFromStaticType(staticType)
	if !semaType.IsImportable(map[*sema.Member]bool{}) {
		return false
	}

	// Check all field values are importable
	importable := true
	v.ForEachField(
		context,
		func(fieldName string, value Value) (resume bool) {

			if strings.HasPrefix(fieldName, unrepresentableNamePrefix) {
				importable = false
				return false
			}

			if !value.IsImportable(context) {
				importable = false
				// stop iteration
				return false
			}

			// continue iteration
			return true
		},
	)

	return importable
}

func (v *CompositeValue) IsDestroyed() bool {
	return v.isDestroyed
}

func resourceDefaultDestroyEventName(t sema.ContainerType) string {
	return resourceDefaultDestroyEventPrefix + string(t.ID())
}

// get all the default destroy event constructors associated with this composite value.
// note that there can be more than one in the case where a resource inherits from an interface
// that also defines a default destroy event. When that composite is destroyed, all of these
// events will need to be emitted.
func (v *CompositeValue) defaultDestroyEventConstructors() (constructors []FunctionValue) {
	if v.Functions == nil {
		return
	}
	v.Functions.Foreach(func(name string, f FunctionValue) {
		if strings.HasPrefix(name, resourceDefaultDestroyEventPrefix) {
			constructors = append(constructors, f)
		}
	})
	return
}

func (v *CompositeValue) Destroy(context ResourceDestructionContext) {

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindDestroyCompositeValue,
			Intensity: 1,
		},
	)

	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			context.ReportCompositeValueDestroyTrace(
				valueID,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	// before actually performing the destruction (i.e. so that any fields are still available),
	// compute the default arguments of the default destruction events (if any exist). However,
	// wait until after the destruction completes to actually emit the events, so that the correct order
	// is preserved and nested resource destroy events happen first

	// default destroy event constructors are encoded as functions on the resource (with an unrepresentable name)
	// so that we can leverage existing atree encoding and decoding. However, we need to make sure functions are initialized
	// if the composite was recently loaded from storage
	if v.Functions == nil {
		v.Functions = context.GetCompositeValueFunctions(v)
	}

	events := context.DefaultDestroyEvents(v)
	for _, event := range events {
		// emit the event once destruction is complete
		eventType := MustSemaTypeOfValue(event, context).(*sema.CompositeType)

		eventFields := extractEventFields(context, event, eventType)

		defer context.EmitEvent(context, eventType, eventFields)
	}

	valueID := v.ValueID()

	context.WithResourceDestruction(
		valueID,
		func() {
			contextForLocation := context.GetResourceDestructionContextForLocation(v.Location)

			// destroy every nested resource in this composite; note that this iteration includes attachments
			v.ForEachField(
				contextForLocation,
				func(_ string, fieldValue Value) bool {
					if compositeFieldValue, ok := fieldValue.(*CompositeValue); ok &&
						compositeFieldValue.Kind == common.CompositeKindAttachment {

						compositeFieldValue.SetBaseValue(v)
					}

					maybeDestroy(contextForLocation, fieldValue)

					return true
				},
			)
		},
	)

	v.isDestroyed = true

	InvalidateReferencedResources(context, v)

	v.dictionary = nil
}

func (v *CompositeValue) DefaultDestroyEvents(
	context ResourceDestructionContext,
) []*CompositeValue {
	eventConstructors := v.defaultDestroyEventConstructors()

	length := len(eventConstructors)
	common.UseMemory(context, common.NewGoSliceMemoryUsages(length))

	events := make([]*CompositeValue, 0, length)
	for _, constructor := range eventConstructors {

		returnType := constructor.FunctionType(context).ReturnTypeAnnotation.Type

		// pass the container value to the creation of the default event as an implicit argument, so that
		// its fields are accessible in the body of the event constructor
		eventConstructorInvocation := NewInvocation(
			context,
			nil,
			nil,
			[]Value{v},
			[]sema.Type{},
			nil,
			returnType,
			context.LocationRange(),
		)

		event := constructor.Invoke(eventConstructorInvocation).(*CompositeValue)
		events = append(events, event)
	}

	return events
}

func (v *CompositeValue) getBuiltinMember(context MemberAccessibleContext, name string) Value {

	switch name {
	case sema.ResourceOwnerFieldName:
		if v.Kind == common.CompositeKindResource {
			return v.OwnerValue(context)
		}
	case sema.CompositeForEachAttachmentFunctionName:
		if v.Kind.SupportsAttachments() {
			return v.forEachAttachmentFunction(context)
		}
	}

	return nil
}

func (v *CompositeValue) GetMember(context MemberAccessibleContext, name string) Value {

	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			context.ReportCompositeValueGetMemberTrace(
				valueID,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

	if builtin := v.getBuiltinMember(context, name); builtin != nil {
		return compositeMember(context, v, builtin)
	}

	// Give computed fields precedence over stored fields for built-in types
	if v.Location == nil {
		if computedField := v.GetComputedField(context, name); computedField != nil {
			return computedField
		}
	}

	if field := v.GetField(context, name); field != nil {
		return compositeMember(context, v, field)
	}

	if v.NestedVariables != nil {
		variable, ok := v.NestedVariables[name]
		if ok {
			return variable.GetValue(context)
		}
	}

	context = context.GetMemberAccessContextForLocation(v.Location)

	// Dynamically link in the computed fields, injected fields, and functions

	if computedField := v.GetComputedField(context, name); computedField != nil {
		return computedField
	}

	if injectedField := v.GetInjectedField(context, name); injectedField != nil {
		return injectedField
	}

	if function := context.GetMethod(v, name); function != nil {
		return function
	}

	return nil
}

func compositeMember(context FunctionCreationContext, compositeValue Value, memberValue Value) Value {
	hostFunc, isHostFunc := memberValue.(*HostFunctionValue)
	if isHostFunc {
		return NewBoundFunctionValue(context, hostFunc, &compositeValue, nil)
	}

	return memberValue
}

func (v *CompositeValue) isInvalidatedResource(_ ValueStaticTypeContext) bool {
	return v.isDestroyed || (v.dictionary == nil && v.Kind == common.CompositeKindResource)
}

func (v *CompositeValue) IsStaleResource(context ValueStaticTypeContext) bool {
	return v.dictionary == nil && v.IsResourceKinded(context)
}

func (v *CompositeValue) GetComputedFields() map[string]ComputedField {
	if v.computedFields == nil {
		v.computedFields = GetCompositeValueComputedFields(v)
	}
	return v.computedFields
}

func (v *CompositeValue) GetComputedField(context ValueTransferContext, name string) Value {
	computedFields := v.GetComputedFields()

	computedField, ok := computedFields[name]
	if !ok {
		return nil
	}

	return computedField(context, v)
}

func (v *CompositeValue) GetInjectedField(context MemberAccessibleContext, name string) Value {
	if v.injectedFields == nil {
		v.injectedFields = GetCompositeValueInjectedFields(context, v)
	}

	value, ok := v.injectedFields[name]
	if !ok {
		return nil
	}

	return value
}

func (v *CompositeValue) GetMethod(context MemberAccessibleContext, name string) FunctionValue {
	if v.Functions == nil {
		v.Functions = context.GetCompositeValueFunctions(v)
	}
	// if no functions were produced, the `Get` below will be nil
	if v.Functions == nil {
		return nil
	}

	function, present := v.Functions.Get(name)
	if !present {
		return nil
	}

	var base *EphemeralReferenceValue
	var self Value = v
	if v.Kind == common.CompositeKindAttachment {
		functionAccess := GetAccessOfMember(context, v, name)

		// with respect to entitlements, any access inside an attachment that is not an entitlement access
		// does not provide any entitlements to base and self
		// E.g. consider:
		//
		//    access(E) fun foo() {}
		//    access(self) fun bar() {
		//        self.foo()
		//    }
		//    access(all) fun baz() {
		//        self.bar()
		//    }
		//
		// clearly `bar` should be callable within `baz`, but we cannot allow `foo`
		// to be callable within `bar`, or it will be possible to access `E` entitled
		// methods on `base`
		if functionAccess.IsPrimitiveAccess() {
			functionAccess = sema.UnauthorizedAccess
		}
		base, self = AttachmentBaseAndSelfValues(context, functionAccess, v)
	}

	// If the function is already a bound function, then do not re-wrap.
	// `NewBoundFunctionValue` already handles this.
	return NewBoundFunctionValue(context, function, &self, base)
}

func (v *CompositeValue) OwnerValue(context MemberAccessibleContext) OptionalValue {
	address := v.StorageAddress()

	if address == (atree.Address{}) {
		return NilOptionalValue
	}

	accountHandler := context.GetAccountHandlerFunc()

	ownerAccount := accountHandler(context, AddressValue(address))

	// Owner must be of `Account` type.
	ExpectType(
		context,
		ownerAccount,
		sema.AccountType,
	)

	reference := NewEphemeralReferenceValue(
		context,
		UnauthorizedAccess,
		ownerAccount,
		sema.AccountType,
	)

	return NewSomeValueNonCopying(context, reference)
}

func (v *CompositeValue) RemoveMember(context ValueTransferContext, name string) Value {

	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			context.ReportCompositeValueRemoveMemberTrace(
				valueID,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeMapRemove,
			Intensity: 1,
		},
	)

	// No need to clean up storable for passed-in key value,
	// as atree never calls Storable()
	existingKeyStorable, existingValueStorable, err := v.dictionary.Remove(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		StringAtreeValue(name),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil
		}
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.dictionary)
	context.MaybeValidateAtreeStorage()

	// Key
	RemoveReferencedSlab(context, existingKeyStorable)

	// Value

	storedValue := StoredValue(
		context,
		existingValueStorable,
		context.Storage(),
	)
	return storedValue.
		Transfer(
			context,
			atree.Address{},
			true,
			existingValueStorable,
			nil,
			true, // value is standalone because it was removed from parent container.
		)
}

func (v *CompositeValue) SetMemberWithoutTransfer(
	context ValueTransferContext,
	name string,
	value Value,
) bool {

	context.EnforceNotResourceDestruction(v.ValueID())

	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			context.ReportCompositeValueSetMemberTrace(
				valueID,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeMapSet,
			Intensity: 1,
		},
	)

	existingStorable, err := v.dictionary.Set(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		NewStringAtreeValue(context, name),
		value,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.dictionary)
	context.MaybeValidateAtreeStorage()

	if existingStorable != nil {
		existingValue := StoredValue(context, existingStorable, context.Storage())

		CheckResourceLoss(context, existingValue)

		existingValue.DeepRemove(context, true) // existingValue is standalone because it was overwritten in parent container.

		RemoveReferencedSlab(context, existingStorable)
		return true
	}

	return false
}

func (v *CompositeValue) SetMember(context ValueTransferContext, name string, value Value) bool {
	address := v.StorageAddress()

	value = value.Transfer(
		context,
		address,
		true,
		nil,
		map[atree.ValueID]struct{}{
			v.ValueID(): {},
		},
		true, // value is standalone before being set in parent container.
	)

	return v.SetMemberWithoutTransfer(
		context,
		name,
		value,
	)
}

func (v *CompositeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *CompositeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(NoOpStringContext{}, seenReferences)
}

var emptyCompositeStringLen = len(format.Composite("", nil))

func (v *CompositeValue) MeteredString(
	context ValueStringContext,
	seenReferences SeenReferences,
) string {

	if v.Stringer != nil {
		return v.Stringer(context, v, seenReferences)
	}

	strLen := emptyCompositeStringLen

	var fields []CompositeField

	v.ForEachField(
		context,
		func(fieldName string, fieldValue Value) (resume bool) {
			field := NewCompositeField(
				context,
				fieldName,
				fieldValue,
			)

			fields = append(fields, field)

			strLen += len(field.Name)

			return true
		},
	)

	typeId := string(v.TypeID())

	// bodyLen = len(fieldNames) + len(typeId) + (n times colon+space) + ((n-1) times comma+space)
	//         = len(fieldNames) + len(typeId) + 2n + 2n - 2
	//         = len(fieldNames) + len(typeId) + 4n - 2
	//
	// Since (-2) only occurs if its non-empty, ignore the (-2). i.e: overestimate
	// 		bodyLen = len(fieldNames) + len(typeId) + 4n
	//
	strLen = strLen + len(typeId) + len(fields)*4

	common.UseMemory(context, common.NewRawStringMemoryUsage(strLen))

	return formatComposite(context, typeId, fields, seenReferences)
}

func formatComposite(
	context ValueStringContext,
	typeId string,
	fields []CompositeField,
	seenReferences SeenReferences,
) string {
	preparedFields := make(
		[]struct {
			Name  string
			Value string
		},
		0,
		len(fields),
	)

	for _, field := range fields {
		preparedFields = append(
			preparedFields,
			struct {
				Name  string
				Value string
			}{
				Name:  field.Name,
				Value: field.Value.MeteredString(context, seenReferences),
			},
		)
	}

	return format.Composite(typeId, preparedFields)
}

func (v *CompositeValue) GetField(gauge common.Gauge, name string) Value {

	common.UseComputation(
		gauge,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeMapGet,
			Intensity: 1,
		},
	)

	storedValue, err := v.dictionary.Get(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		StringAtreeValue(name),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil
		}
		panic(errors.NewExternalError(err))
	}

	return MustConvertStoredValue(gauge, storedValue)
}

func (v *CompositeValue) Equal(context ValueComparisonContext, other Value) bool {
	otherComposite, ok := other.(*CompositeValue)
	if !ok {
		return false
	}

	if !v.StaticType(context).Equal(otherComposite.StaticType(context)) ||
		v.Kind != otherComposite.Kind ||
		v.dictionary.Count() != otherComposite.dictionary.Count() {

		return false
	}

	iterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	for {
		common.UseComputation(
			context,
			common.ComputationUsage{
				Kind:      common.ComputationKindAtreeMapReadIteration,
				Intensity: 1,
			},
		)

		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			return true
		}

		fieldName := string(key.(StringAtreeValue))

		// NOTE: Do NOT use an iterator, iteration order of fields may be different
		// (if stored in different account, as storage ID is used as hash seed)
		otherValue := otherComposite.GetField(context, fieldName)

		equatableValue, ok := MustConvertStoredValue(context, value).(EquatableValue)
		if !ok || !equatableValue.Equal(context, otherValue) {
			return false
		}
	}
}

// HashInput returns a byte slice containing:
// - HashInputTypeEnum (1 byte)
// - type id (n bytes)
// - hash input of raw value field name (n bytes)
func (v *CompositeValue) HashInput(gauge common.Gauge, scratch []byte) []byte {
	if v.Kind == common.CompositeKindEnum {
		typeID := v.TypeID()

		rawValue := v.GetField(gauge, sema.EnumRawValueFieldName)
		rawValueHashInput := rawValue.(HashableValue).
			HashInput(gauge, scratch)

		length := 1 + len(typeID) + len(rawValueHashInput)
		if length <= len(scratch) {
			// Copy rawValueHashInput first because
			// rawValueHashInput and scratch can point to the same underlying scratch buffer
			copy(scratch[1+len(typeID):], rawValueHashInput)

			scratch[0] = byte(HashInputTypeEnum)
			copy(scratch[1:], typeID)
			return scratch[:length]
		}

		buffer := make([]byte, length)
		buffer[0] = byte(HashInputTypeEnum)
		copy(buffer[1:], typeID)
		copy(buffer[1+len(typeID):], rawValueHashInput)
		return buffer
	}

	panic(errors.NewUnreachableError())
}

func (v *CompositeValue) TypeID() TypeID {
	if v.typeID == "" {
		v.typeID = common.NewTypeIDFromQualifiedName(nil, v.Location, v.QualifiedIdentifier)
	}
	return v.typeID
}

func (v *CompositeValue) ConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	results TypeConformanceResults,
) bool {
	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			context.ReportCompositeValueConformsToStaticTypeTrace(
				valueID,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	staticType := v.StaticType(context)
	semaType := context.SemaTypeFromStaticType(staticType)

	switch staticType.(type) {
	case *CompositeStaticType:
		return v.CompositeStaticTypeConformsToStaticType(
			context,
			results,
			semaType,
		)

	// CompositeValue is also used for storing types which aren't CompositeStaticType.
	// E.g. InclusiveRange.
	case InclusiveRangeStaticType:
		return v.InclusiveRangeStaticTypeConformsToStaticType(
			context,
			results,
			semaType,
		)

	default:
		return false
	}
}

func (v *CompositeValue) CompositeStaticTypeConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	results TypeConformanceResults,
	semaType sema.Type,
) bool {
	compositeType, ok := semaType.(*sema.CompositeType)
	if !ok ||
		v.Kind != compositeType.Kind ||
		v.TypeID() != compositeType.ID() {

		return false
	}

	if compositeType.Kind == common.CompositeKindAttachment {
		base := v.getBaseValue(context, UnauthorizedAccess).Value
		if base == nil || !base.ConformsToStaticType(context, results) {
			return false
		}
	}

	fieldsLen := v.FieldCount()

	computedFields := v.GetComputedFields()
	//if computedFields != nil {
	//	fieldsLen += len(computedFields)
	//}

	// Values for all declared fields must be present
	if fieldsLen != len(compositeType.Fields) {
		return false
	}

	for _, fieldName := range compositeType.Fields {
		value := v.GetField(context, fieldName)
		if value == nil {
			if computedFields == nil {
				return false
			}

			fieldGetter, ok := computedFields[fieldName]
			if !ok {
				return false
			}

			value = fieldGetter(context, v)
		}

		member, ok := compositeType.Members.Get(fieldName)
		if !ok {
			return false
		}

		fieldStaticType := value.StaticType(context)

		if !IsSubTypeOfSemaType(context, fieldStaticType, member.TypeAnnotation.Type) {
			return false
		}

		if !value.ConformsToStaticType(context, results) {
			return false
		}
	}

	return true
}

func (v *CompositeValue) InclusiveRangeStaticTypeConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	results TypeConformanceResults,
	semaType sema.Type,
) bool {
	inclusiveRangeType, ok := semaType.(*sema.InclusiveRangeType)
	if !ok {
		return false
	}

	expectedMemberStaticType := ConvertSemaToStaticType(context, inclusiveRangeType.MemberType)
	for _, fieldName := range sema.InclusiveRangeTypeFieldNames {
		value := v.GetField(context, fieldName)

		fieldStaticType := value.StaticType(context)

		// InclusiveRange is non-covariant.
		// For e.g. we disallow assigning InclusiveRange<Int> to an InclusiveRange<Integer>.
		// Hence, we do an exact equality check instead of a subtype check.
		if !fieldStaticType.Equal(expectedMemberStaticType) {
			return false
		}

		if !value.ConformsToStaticType(context, results) {
			return false
		}
	}

	return true
}

func (v *CompositeValue) FieldCount() int {
	return int(v.dictionary.Count())
}

func (v *CompositeValue) IsStorable() bool {

	// Only structures, resources, enums, and contracts can be stored.
	// Contracts are not directly storable by programs,
	// but they are still stored in storage by the interpreter

	switch v.Kind {
	case common.CompositeKindStructure,
		common.CompositeKindResource,
		common.CompositeKindEnum,
		common.CompositeKindAttachment,
		common.CompositeKindContract:
		break
	default:
		return false
	}

	// Composite value's of native/built-in types are not storable for now
	return v.Location != nil
}

func (v *CompositeValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint32,
) (atree.Storable, error) {
	if !v.IsStorable() {
		return NonStorable{Value: v}, nil
	}

	// NOTE: Need to change CompositeValue.UnwrapAtreeValue()
	// if CompositeValue is stored with wrapping.

	return v.dictionary.Storable(storage, address, maxInlineSize)
}

func (v *CompositeValue) UnwrapAtreeValue() (atree.Value, uint32) {
	// Wrapper size is 0 because CompositeValue is stored as
	// atree.OrderedMap without any physical wrapping (see CompositeValue.Storable()).
	return v.dictionary, 0
}

func (v *CompositeValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *CompositeValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	if v.Kind == common.CompositeKindAttachment {
		return MustSemaTypeOfValue(v, context).IsResourceType()
	}
	return v.Kind == common.CompositeKindResource
}

func (v *CompositeValue) IsReferenceTrackedResourceKindedValue() {}

func (v *CompositeValue) Transfer(
	context ValueTransferContext,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.ValueID]struct{},
	hasNoParentContainer bool,
) Value {

	count := v.FieldCount()

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindTransferCompositeValue,
			Intensity: uint64(count),
		},
	)

	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			context.ReportCompositeValueTransferTrace(
				valueID,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	currentValueID := v.ValueID()
	currentAddress := v.StorageAddress()

	if preventTransfer == nil {
		preventTransfer = map[atree.ValueID]struct{}{}
	} else if _, ok := preventTransfer[currentValueID]; ok {
		panic(&RecursiveTransferError{})
	}
	preventTransfer[currentValueID] = struct{}{}
	defer delete(preventTransfer, currentValueID)

	dictionary := v.dictionary

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(context)

	if needsStoreTo && v.Kind == common.CompositeKindContract {
		panic(&NonTransferableValueError{
			Value: v,
		})
	}

	if needsStoreTo || !isResourceKinded {
		// Use non-readonly iterator here because iterated
		// value can be removed if remove parameter is true.
		iterator, err := v.dictionary.Iterator(
			StringAtreeValueComparator,
			StringAtreeValueHashInput,
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		elementCount := v.dictionary.Count()

		elementOverhead, dataUse, metaDataUse := common.NewAtreeMapMemoryUsages(elementCount, 0)
		common.UseMemory(context, elementOverhead)
		common.UseMemory(context, dataUse)
		common.UseMemory(context, metaDataUse)

		elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(elementCount, 0)
		common.UseMemory(context, elementMemoryUse)

		func() {
			seed := v.dictionary.Seed()

			if TracingEnabled {
				startTime := time.Now()

				defer func() {
					valueID := dictionary.ValueID().String()
					typeID := string(v.TypeID())

					context.ReportAtreeNewMapFromBatchDataTrace(
						valueID,
						typeID,
						seed,
						time.Since(startTime),
					)
				}()
			}

			common.UseComputation(
				context,
				common.ComputationUsage{
					Kind:      common.ComputationKindAtreeMapBatchConstruction,
					Intensity: uint64(count),
				},
			)

			common.UseComputation(
				context,
				common.ComputationUsage{
					Kind:      common.ComputationKindAtreeMapReadIteration,
					Intensity: uint64(count),
				},
			)

			dictionary, err = atree.NewMapFromBatchData(
				context.Storage(),
				address,
				atree.NewDefaultDigesterBuilder(),
				v.dictionary.Type(),
				StringAtreeValueComparator,
				StringAtreeValueHashInput,
				seed,
				func() (atree.Value, atree.Value, error) {

					// Computation was already metered above

					atreeKey, atreeValue, err := iterator.Next()
					if err != nil {
						return nil, nil, err
					}
					if atreeKey == nil || atreeValue == nil {
						return nil, nil, nil
					}

					// NOTE: key is stringAtreeValue
					// and does not need to be converted or copied

					value := MustConvertStoredValue(context, atreeValue)
					// the base of an attachment is not stored in the atree, so in order to make the
					// transfer happen properly, we set the base value here if this field is an attachment
					if compositeValue, ok := value.(*CompositeValue); ok &&
						compositeValue.Kind == common.CompositeKindAttachment {

						compositeValue.SetBaseValue(v)
					}

					value = value.Transfer(
						context,
						address,
						remove,
						nil,
						preventTransfer,
						false, // value is an element of parent container because it is returned from iterator.
					)

					return atreeKey, value, nil
				},
			)
			if err != nil {
				panic(errors.NewExternalError(err))
			}
		}()

		if remove {

			common.UseComputation(
				context,
				common.ComputationUsage{
					Kind:      common.ComputationKindAtreeMapPopIteration,
					Intensity: v.dictionary.Count(),
				},
			)

			err = v.dictionary.PopIterate(func(nameStorable atree.Storable, valueStorable atree.Storable) {
				RemoveReferencedSlab(context, nameStorable)
				RemoveReferencedSlab(context, valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			context.MaybeValidateAtreeValue(v.dictionary)
			if hasNoParentContainer {
				context.MaybeValidateAtreeStorage()
			}

			RemoveReferencedSlab(context, storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource as invalidated, by unsetting the backing dictionary.
		// This allows raising an error when the resource is attempted
		// to be transferred/moved again (see beginning of this function)

		InvalidateReferencedResources(context, v)

		v.dictionary = nil
	}

	info := NewCompositeTypeInfo(
		context,
		v.Location,
		v.QualifiedIdentifier,
		v.Kind,
	)

	res := NewCompositeValueFromAtreeMap(
		context,
		info,
		dictionary,
	)

	res.injectedFields = v.injectedFields
	res.computedFields = v.computedFields
	res.NestedVariables = v.NestedVariables
	res.Functions = v.Functions
	res.Stringer = v.Stringer
	res.isDestroyed = v.isDestroyed
	res.typeID = v.typeID
	res.staticType = v.staticType
	res.base = v.base

	if needsStoreTo &&
		res.Kind == common.CompositeKindResource {

		context.OnResourceOwnerChange(
			res,
			common.Address(currentAddress),
			common.Address(address),
		)
	}

	return res
}

func (v *CompositeValue) ResourceUUID(interpreter *Interpreter) *UInt64Value {
	fieldValue := v.GetField(interpreter, sema.ResourceUUIDFieldName)
	uuid, ok := fieldValue.(UInt64Value)
	if !ok {
		return nil
	}
	return &uuid
}

func (v *CompositeValue) Clone(context ValueCloneContext) Value {

	iterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	dictionary, err := atree.NewMapFromBatchData(
		context.Storage(),
		v.StorageAddress(),
		atree.NewDefaultDigesterBuilder(),
		v.dictionary.Type(),
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		v.dictionary.Seed(),
		func() (atree.Value, atree.Value, error) {

			atreeKey, atreeValue, err := iterator.Next()
			if err != nil {
				return nil, nil, err
			}
			if atreeKey == nil || atreeValue == nil {
				return nil, nil, nil
			}

			// The key is always interpreter.StringAtreeValue,
			// an "atree-level string", not an interpreter.Value.
			// Thus, we do not, and cannot, convert.
			key := atreeKey
			value := MustConvertStoredValue(context, atreeValue).Clone(context)

			return key, value, nil
		},
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &CompositeValue{
		dictionary:          dictionary,
		Location:            v.Location,
		QualifiedIdentifier: v.QualifiedIdentifier,
		Kind:                v.Kind,
		injectedFields:      v.injectedFields,
		computedFields:      v.computedFields,
		NestedVariables:     v.NestedVariables,
		Functions:           v.Functions,
		Stringer:            v.Stringer,
		isDestroyed:         v.isDestroyed,
		typeID:              v.typeID,
		staticType:          v.staticType,
		base:                v.base,
	}
}

func (v *CompositeValue) DeepRemove(context ValueRemoveContext, hasNoParentContainer bool) {
	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			context.ReportCompositeValueDeepRemoveTrace(
				valueID,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.dictionary.Storage

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeMapPopIteration,
			Intensity: v.dictionary.Count(),
		},
	)

	err := v.dictionary.PopIterate(func(nameStorable atree.Storable, valueStorable atree.Storable) {
		// NOTE: key / field name is stringAtreeValue,
		// and not a Value, so no need to deep remove
		RemoveReferencedSlab(context, nameStorable)

		value := StoredValue(context, valueStorable, storage)
		value.DeepRemove(context, false) // value is an element of v.dictionary because it is from PopIterate() callback.
		RemoveReferencedSlab(context, valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.dictionary)
	if hasNoParentContainer {
		context.MaybeValidateAtreeStorage()
	}
}

func (v *CompositeValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

// ForEachFieldName iterates over all field names of the composite value.
// It does NOT iterate over computed fields and functions!
func (v *CompositeValue) ForEachFieldName(
	gauge common.ComputationGauge,
	f func(fieldName string) (resume bool),
) {
	iterate := func(fn atree.MapElementIterationFunc) error {
		// Use NonReadOnlyIterator because we are not sure if it's guaranteed that
		// all uses of CompositeValue.ForEachFieldName are only read-only.
		// TODO: determine if all uses of CompositeValue.ForEachFieldName are read-only.
		return v.dictionary.IterateKeys(
			StringAtreeValueComparator,
			StringAtreeValueHashInput,
			fn,
		)
	}
	v.forEachFieldName(gauge, iterate, f)
}

func (v *CompositeValue) forEachFieldName(
	gauge common.ComputationGauge,
	atreeIterate func(fn atree.MapElementIterationFunc) error,
	f func(fieldName string) (resume bool),
) {
	err := atreeIterate(func(key atree.Value) (resume bool, err error) {

		common.UseComputation(
			gauge,
			common.ComputationUsage{
				Kind:      common.ComputationKindAtreeMapReadIteration,
				Intensity: 1,
			},
		)

		resume = f(
			string(key.(StringAtreeValue)),
		)
		return
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}

// ForEachField iterates over all field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
func (v *CompositeValue) ForEachField(
	context ContainerMutationContext,
	f func(fieldName string, fieldValue Value) (resume bool),
) {
	iterate := func(fn atree.MapEntryIterationFunc) error {
		return v.dictionary.Iterate(
			StringAtreeValueComparator,
			StringAtreeValueHashInput,
			fn,
		)
	}
	v.forEachField(
		context,
		iterate,
		f,
	)
}

// ForEachReadOnlyLoadedField iterates over all LOADED field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
// DO NOT perform storage mutations in the callback!
func (v *CompositeValue) ForEachReadOnlyLoadedField(
	context ContainerMutationContext,
	f func(fieldName string, fieldValue Value) (resume bool),
) {
	v.forEachField(
		context,
		v.dictionary.IterateReadOnlyLoadedValues,
		f,
	)
}

func (v *CompositeValue) forEachField(
	context ContainerMutationContext,
	atreeIterate func(fn atree.MapEntryIterationFunc) error,
	f func(fieldName string, fieldValue Value) (resume bool),
) {
	err := atreeIterate(func(key atree.Value, atreeValue atree.Value) (resume bool, err error) {
		value := MustConvertStoredValue(context, atreeValue)
		CheckInvalidatedResourceOrResourceReference(value, context)

		resume = f(
			string(key.(StringAtreeValue)),
			value,
		)
		return
	})

	if err != nil {
		panic(errors.NewExternalError(err))
	}
}

func (v *CompositeValue) SlabID() atree.SlabID {
	return v.dictionary.SlabID()
}

func (v *CompositeValue) StorageAddress() atree.Address {
	return v.dictionary.Address()
}

func (v *CompositeValue) ValueID() atree.ValueID {
	return v.dictionary.ValueID()
}

func (v *CompositeValue) RemoveField(
	context ValueRemoveContext,
	name string,
) {

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeMapRemove,
			Intensity: 1,
		},
	)

	existingKeyStorable, existingValueStorable, err := v.dictionary.Remove(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		StringAtreeValue(name),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return
		}
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.dictionary)
	context.MaybeValidateAtreeStorage()

	// Key

	// NOTE: key / field name is stringAtreeValue,
	// and not a Value, so no need to deep remove
	RemoveReferencedSlab(context, existingKeyStorable)

	// Value
	existingValue := StoredValue(context, existingValueStorable, context.Storage())
	CheckResourceLoss(context, existingValue)
	existingValue.DeepRemove(context, true) // existingValue is standalone because it was removed from parent container.
	RemoveReferencedSlab(context, existingValueStorable)
}

func (v *CompositeValue) SetNestedVariables(variables map[string]Variable) {
	v.NestedVariables = variables
}

func NewEnumCaseValue(
	context MemberAccessibleContext,
	enumType *sema.CompositeType,
	rawValue NumberValue,
	functions *FunctionOrderedMap,
) *CompositeValue {

	fields := []CompositeField{
		{
			Name:  sema.EnumRawValueFieldName,
			Value: rawValue,
		},
	}

	v := NewCompositeValue(
		context,
		enumType.Location,
		enumType.QualifiedIdentifier(),
		enumType.Kind,
		fields,
		common.ZeroAddress,
	)

	v.Functions = functions

	return v
}

func (v *CompositeValue) getBaseValue(
	context StaticTypeAndReferenceContext,
	functionAuthorization Authorization,
) *EphemeralReferenceValue {
	attachmentType, ok := MustSemaTypeOfValue(v, context).(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var baseType sema.Type
	switch ty := attachmentType.GetBaseType().(type) {
	case *sema.InterfaceType:
		baseType, _ = sema.RewriteWithIntersectionTypes(ty)
	default:
		baseType = ty
	}

	return NewEphemeralReferenceValue(context, functionAuthorization, v.base, baseType)
}

func (v *CompositeValue) SetBaseValue(base *CompositeValue) {
	v.base = base
}

func AttachmentMemberName(typeID string) string {
	return unrepresentableNamePrefix + typeID
}

func (v *CompositeValue) getAttachmentValue(
	context MemberAccessibleContext,
	ty sema.Type,
) *CompositeValue {
	attachment := v.GetMember(context, AttachmentMemberName(string(ty.ID())))
	if attachment != nil {
		return attachment.(*CompositeValue)
	}
	return nil
}

func (v *CompositeValue) GetAttachments(context AttachmentContext) []*CompositeValue {
	var attachments []*CompositeValue
	v.forEachAttachment(
		context,
		func(attachment *CompositeValue) {
			attachments = append(attachments, attachment)
		},
	)
	return attachments
}

func (v *CompositeValue) forEachAttachmentFunction(context FunctionCreationContext) Value {
	compositeType := MustSemaTypeOfValue(v, context).(*sema.CompositeType)
	return NewBoundHostFunctionValue(
		context,
		v,
		sema.CompositeForEachAttachmentFunctionType(
			compositeType.GetCompositeKind(),
		),
		NativeForEachAttachmentFunction,
	)
}

var NativeForEachAttachmentFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		v := AssertValueOfType[*CompositeValue](receiver)
		functionValue := AssertValueOfType[FunctionValue](args[0])

		v.ForEachAttachment(context, functionValue)

		return Void
	},
)

func (v *CompositeValue) ForEachAttachment(
	context InvocationContext,
	functionValue FunctionValue,
) {
	functionValueType := functionValue.FunctionType(context)
	parameterTypes := functionValueType.ParameterTypes()
	returnType := functionValueType.ReturnTypeAnnotation.Type

	fn := func(attachment *CompositeValue) {
		attachmentType := MustSemaTypeOfValue(attachment, context).(*sema.CompositeType)

		attachmentReference := NewEphemeralReferenceValue(
			context,
			// attachments are unauthorized during iteration
			UnauthorizedAccess,
			attachment,
			attachmentType,
		)

		referenceType := sema.NewReferenceType(
			context,
			// attachments are unauthorized during iteration
			sema.UnauthorizedAccess,
			attachmentType,
		)

		invokeFunctionValue(
			context,
			functionValue,
			[]Value{attachmentReference},
			[]sema.Type{referenceType},
			parameterTypes,
			returnType,
			nil,
		)
	}

	v.forEachAttachment(context, fn)
}

func AttachmentBaseAndSelfValues(
	context StaticTypeAndReferenceContext,
	fnAccess sema.Access,
	v *CompositeValue,
) (base *EphemeralReferenceValue, self *EphemeralReferenceValue) {
	attachmentReferenceAuth := ConvertSemaAccessToStaticAuthorization(context, fnAccess)

	base = v.getBaseValue(context, attachmentReferenceAuth)
	// in attachment functions, self is a reference value
	self = NewEphemeralReferenceValue(
		context,
		attachmentReferenceAuth,
		v,
		MustSemaTypeOfValue(v, context),
	)

	return
}

func (v *CompositeValue) forEachAttachment(
	context AttachmentContext,
	f func(*CompositeValue),
) {
	// The attachment iteration creates an implicit reference to the composite, and holds onto that referenced-value.
	// But the reference could get invalidated during the iteration, making that referenced-value invalid.
	// We create a reference here for the purposes of tracking it during iteration.
	vType := MustSemaTypeOfValue(v, context)
	compositeReference := NewEphemeralReferenceValue(context, UnauthorizedAccess, v, vType)
	forEachAttachment(context, compositeReference, f)
}

func forEachAttachment(
	context AttachmentContext,
	compositeReference *EphemeralReferenceValue,
	f func(*CompositeValue),
) {
	composite, ok := compositeReference.Value.(*CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	iterator, err := composite.dictionary.Iterator(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	oldSharedState := context.SetAttachmentIteration(composite, true)
	defer func() {
		context.SetAttachmentIteration(composite, oldSharedState)
	}()

	for {
		// Check that the implicit composite reference was not invalidated during iteration
		CheckInvalidatedResourceOrResourceReference(compositeReference, context)
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			break
		}
		if strings.HasPrefix(string(key.(StringAtreeValue)), unrepresentableNamePrefix) {
			attachment, ok := MustConvertStoredValue(context, value).(*CompositeValue)
			if !ok {
				panic(errors.NewExternalError(err))
			}
			// `f` takes the `attachment` value directly, but if a method to later iterate over
			// attachments is added that takes a `fun (&Attachment): Void` callback, the `f` provided here
			// should convert the provided attachment value into a reference before passing it to the user
			// callback
			attachment.SetBaseValue(composite)
			f(attachment)
		}
	}
}

func (v *CompositeValue) getTypeKey(
	context MemberAccessibleContext,
	keyType sema.Type,
	baseAccess sema.Access,
) Value {
	attachment := v.getAttachmentValue(context, keyType)
	if attachment == nil {
		return Nil
	}
	attachmentType := keyType.(*sema.CompositeType)
	// dynamically set the attachment's base to this composite
	attachment.SetBaseValue(v)

	// The attachment reference has the same entitlements as the base access
	attachmentRef := NewEphemeralReferenceValue(
		context,
		ConvertSemaAccessToStaticAuthorization(context, baseAccess),
		attachment,
		attachmentType,
	)

	return NewSomeValueNonCopying(context, attachmentRef)
}

func (v *CompositeValue) GetTypeKey(context MemberAccessibleContext, ty sema.Type) Value {
	access := sema.UnauthorizedAccess
	attachmentTyp, isAttachmentType := ty.(*sema.CompositeType)
	if isAttachmentType {
		access = attachmentTyp.SupportedEntitlements().Access()
	}
	return v.getTypeKey(context, ty, access)
}

func (v *CompositeValue) SetTypeKey(
	context ValueTransferContext,
	attachmentType sema.Type,
	attachment Value,
) {
	memberName := AttachmentMemberName(string(attachmentType.ID()))
	if v.SetMember(context, memberName, attachment) {
		panic(&DuplicateAttachmentError{
			AttachmentType: attachmentType,
			Value:          v,
		})
	}
}

func (v *CompositeValue) RemoveTypeKey(
	context ValueTransferContext,
	attachmentType sema.Type,
) Value {
	memberName := AttachmentMemberName(string(attachmentType.ID()))
	return v.RemoveMember(context, memberName)
}

func (v *CompositeValue) Iterator(context ValueStaticTypeContext) ValueIterator {
	staticType := v.StaticType(context)

	switch typ := staticType.(type) {
	case InclusiveRangeStaticType:
		return NewInclusiveRangeIterator(context, v, typ)

	default:
		// Must be caught in the checker.
		panic(errors.NewUnreachableError())
	}
}

func (v *CompositeValue) ForEach(
	context IterableValueForeachContext,
	_ sema.Type,
	function func(value Value) (resume bool),
	transferElements bool,
) {
	iterator := v.Iterator(context)
	for {
		value := iterator.Next(context)
		if value == nil {
			return
		}

		if transferElements {
			// Each element must be transferred before passing onto the function.
			value = value.Transfer(
				context,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is from iterator.
			)
		}

		if !function(value) {
			return
		}
	}
}

func (v *CompositeValue) AtreeMap() *atree.OrderedMap {
	return v.dictionary
}

func (v *CompositeValue) Inlined() bool {
	return v.dictionary.Inlined()
}
