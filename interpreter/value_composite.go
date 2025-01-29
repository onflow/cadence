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

type ComputedField func(*Interpreter, LocationRange, *CompositeValue) Value

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
	interpreter *Interpreter,
	locationRange LocationRange,
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
	fields []CompositeField,
	address common.Address,
	staticType StaticType,
) *CompositeValue {
	value := NewCompositeValue(
		interpreter,
		locationRange,
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
	interpreter *Interpreter,
	locationRange LocationRange,
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
	fields []CompositeField,
	address common.Address,
) *CompositeValue {

	interpreter.ReportComputation(common.ComputationKindCreateCompositeValue, 1)

	config := interpreter.SharedState.Config

	var v *CompositeValue

	if config.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			owner := v.GetOwner().String()
			typeID := string(v.TypeID())
			kind := v.Kind.String()

			interpreter.reportCompositeValueConstructTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	constructor := func() *atree.OrderedMap {
		dictionary, err := atree.NewMap(
			config.Storage,
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			NewCompositeTypeInfo(
				interpreter,
				location,
				qualifiedIdentifier,
				kind,
			),
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return dictionary
	}

	typeInfo := NewCompositeTypeInfo(
		interpreter,
		location,
		qualifiedIdentifier,
		kind,
	)

	v = newCompositeValueFromConstructor(interpreter, uint64(len(fields)), typeInfo, constructor)

	for _, field := range fields {
		v.SetMember(
			interpreter,
			locationRange,
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

func (*CompositeValue) isValue() {}

func (*CompositeValue) isAtreeContainerBackedValue() {}

func (v *CompositeValue) Accept(interpreter *Interpreter, visitor Visitor, locationRange LocationRange) {
	descend := visitor.VisitCompositeValue(interpreter, v)
	if !descend {
		return
	}

	v.ForEachField(interpreter, func(_ string, value Value) (resume bool) {
		value.Accept(interpreter, visitor, locationRange)

		// continue iteration
		return true
	}, locationRange)
}

// Walk iterates over all field values of the composite value.
// It does NOT walk the computed field or functions!
func (v *CompositeValue) Walk(interpreter *Interpreter, walkChild func(Value), locationRange LocationRange) {
	v.ForEachField(interpreter, func(_ string, value Value) (resume bool) {
		walkChild(value)

		// continue iteration
		return true
	}, locationRange)
}

func (v *CompositeValue) StaticType(interpreter *Interpreter) StaticType {
	if v.staticType == nil {
		// NOTE: Instead of using NewCompositeStaticType, which always generates the type ID,
		// use the TypeID accessor, which may return an already computed type ID
		v.staticType = NewCompositeStaticType(
			interpreter,
			v.Location,
			v.QualifiedIdentifier,
			v.TypeID(),
		)
	}
	return v.staticType
}

func (v *CompositeValue) IsImportable(inter *Interpreter, locationRange LocationRange) bool {
	// Check type is importable
	staticType := v.StaticType(inter)
	semaType := inter.MustConvertStaticToSemaType(staticType)
	if !semaType.IsImportable(map[*sema.Member]bool{}) {
		return false
	}

	// Check all field values are importable
	importable := true
	v.ForEachField(inter, func(_ string, value Value) (resume bool) {
		if !value.IsImportable(inter, locationRange) {
			importable = false
			// stop iteration
			return false
		}

		// continue iteration
		return true
	}, locationRange)

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

func (v *CompositeValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyCompositeValue, 1)

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {

			interpreter.reportCompositeValueDestroyTrace(
				owner,
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
		v.Functions = interpreter.SharedState.typeCodes.CompositeCodes[v.TypeID()].CompositeFunctions
	}
	for _, constructor := range v.defaultDestroyEventConstructors() {

		// pass the container value to the creation of the default event as an implicit argument, so that
		// its fields are accessible in the body of the event constructor
		eventConstructorInvocation := NewInvocation(
			interpreter,
			nil,
			nil,
			nil,
			[]Value{v},
			[]sema.Type{},
			nil,
			locationRange,
		)

		event := constructor.invoke(eventConstructorInvocation).(*CompositeValue)
		eventType := interpreter.MustSemaTypeOfValue(event).(*sema.CompositeType)

		// emit the event once destruction is complete
		defer interpreter.emitEvent(event, eventType, locationRange)
	}

	valueID := v.ValueID()

	interpreter.withResourceDestruction(
		valueID,
		locationRange,
		func() {
			interpreter = v.getInterpreter(interpreter)

			// destroy every nested resource in this composite; note that this iteration includes attachments
			v.ForEachField(interpreter, func(_ string, fieldValue Value) bool {
				if compositeFieldValue, ok := fieldValue.(*CompositeValue); ok && compositeFieldValue.Kind == common.CompositeKindAttachment {
					compositeFieldValue.setBaseValue(interpreter, v)
				}
				maybeDestroy(interpreter, locationRange, fieldValue)
				return true
			}, locationRange)
		},
	)

	v.isDestroyed = true

	interpreter.invalidateReferencedResources(v, locationRange)

	v.dictionary = nil
}

func (v *CompositeValue) getBuiltinMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {

	switch name {
	case sema.ResourceOwnerFieldName:
		if v.Kind == common.CompositeKindResource {
			return v.OwnerValue(interpreter, locationRange)
		}
	case sema.CompositeForEachAttachmentFunctionName:
		if v.Kind.SupportsAttachments() {
			return v.forEachAttachmentFunction(interpreter, locationRange)
		}
	}

	return nil
}

func (v *CompositeValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueGetMemberTrace(
				owner,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

	if builtin := v.getBuiltinMember(interpreter, locationRange, name); builtin != nil {
		return compositeMember(interpreter, v, builtin)
	}

	// Give computed fields precedence over stored fields for built-in types
	if v.Location == nil {
		if computedField := v.GetComputedField(interpreter, locationRange, name); computedField != nil {
			return computedField
		}
	}

	if field := v.GetField(interpreter, locationRange, name); field != nil {
		return compositeMember(interpreter, v, field)
	}

	if v.NestedVariables != nil {
		variable, ok := v.NestedVariables[name]
		if ok {
			return variable.GetValue(interpreter)
		}
	}

	interpreter = v.getInterpreter(interpreter)

	// Dynamically link in the computed fields, injected fields, and functions

	if computedField := v.GetComputedField(interpreter, locationRange, name); computedField != nil {
		return computedField
	}

	if injectedField := v.GetInjectedField(interpreter, name); injectedField != nil {
		return injectedField
	}

	if function := v.GetFunction(interpreter, locationRange, name); function != nil {
		return function
	}

	return nil
}

func compositeMember(interpreter *Interpreter, compositeValue Value, memberValue Value) Value {
	hostFunc, isHostFunc := memberValue.(*HostFunctionValue)
	if isHostFunc {
		return NewBoundFunctionValue(interpreter, hostFunc, &compositeValue, nil, nil)
	}

	return memberValue
}

func (v *CompositeValue) isInvalidatedResource(_ *Interpreter) bool {
	return v.isDestroyed || (v.dictionary == nil && v.Kind == common.CompositeKindResource)
}

func (v *CompositeValue) IsStaleResource(inter *Interpreter) bool {
	return v.dictionary == nil && v.IsResourceKinded(inter)
}

func (v *CompositeValue) getInterpreter(interpreter *Interpreter) *Interpreter {

	// Get the correct interpreter. The program code might need to be loaded.
	// NOTE: standard library values have no location

	location := v.Location

	if location == nil || interpreter.Location == location {
		return interpreter
	}

	return interpreter.EnsureLoaded(v.Location)
}

func (v *CompositeValue) GetComputedFields(interpreter *Interpreter) map[string]ComputedField {
	if v.computedFields == nil {
		v.computedFields = interpreter.GetCompositeValueComputedFields(v)
	}
	return v.computedFields
}

func (v *CompositeValue) GetComputedField(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	computedFields := v.GetComputedFields(interpreter)

	computedField, ok := computedFields[name]
	if !ok {
		return nil
	}

	return computedField(interpreter, locationRange, v)
}

func (v *CompositeValue) GetInjectedField(interpreter *Interpreter, name string) Value {
	if v.injectedFields == nil {
		v.injectedFields = interpreter.GetCompositeValueInjectedFields(v)
	}

	value, ok := v.injectedFields[name]
	if !ok {
		return nil
	}

	return value
}

func (v *CompositeValue) GetFunction(interpreter *Interpreter, locationRange LocationRange, name string) FunctionValue {
	if v.Functions == nil {
		v.Functions = interpreter.GetCompositeValueFunctions(v, locationRange)
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
		functionAccess := interpreter.getAccessOfMember(v, name)

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
		base, self = attachmentBaseAndSelfValues(interpreter, functionAccess, v, locationRange)
	}

	// If the function is already a bound function, then do not re-wrap.
	// `NewBoundFunctionValue` already handles this.
	return NewBoundFunctionValue(interpreter, function, &self, base, nil)
}

func (v *CompositeValue) OwnerValue(interpreter *Interpreter, locationRange LocationRange) OptionalValue {
	address := v.StorageAddress()

	if address == (atree.Address{}) {
		return NilOptionalValue
	}

	config := interpreter.SharedState.Config

	ownerAccount := config.AccountHandler(interpreter, AddressValue(address))

	// Owner must be of `Account` type.
	interpreter.ExpectType(
		ownerAccount,
		sema.AccountType,
		locationRange,
	)

	reference := NewEphemeralReferenceValue(
		interpreter,
		UnauthorizedAccess,
		ownerAccount,
		sema.AccountType,
		locationRange,
	)

	return NewSomeValueNonCopying(interpreter, reference)
}

func (v *CompositeValue) RemoveMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueRemoveMemberTrace(
				owner,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

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

	interpreter.maybeValidateAtreeValue(v.dictionary)
	interpreter.maybeValidateAtreeStorage()

	// Key
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	storedValue := StoredValue(
		interpreter,
		existingValueStorable,
		config.Storage,
	)
	return storedValue.
		Transfer(
			interpreter,
			locationRange,
			atree.Address{},
			true,
			existingValueStorable,
			nil,
			true, // value is standalone because it was removed from parent container.
		)
}

func (v *CompositeValue) SetMemberWithoutTransfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
	value Value,
) bool {
	config := interpreter.SharedState.Config

	interpreter.enforceNotResourceDestruction(v.ValueID(), locationRange)

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueSetMemberTrace(
				owner,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

	existingStorable, err := v.dictionary.Set(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		NewStringAtreeValue(interpreter, name),
		value,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	interpreter.maybeValidateAtreeValue(v.dictionary)
	interpreter.maybeValidateAtreeStorage()

	if existingStorable != nil {
		existingValue := StoredValue(interpreter, existingStorable, config.Storage)

		interpreter.checkResourceLoss(existingValue, locationRange)

		existingValue.DeepRemove(interpreter, true) // existingValue is standalone because it was overwritten in parent container.

		interpreter.RemoveReferencedSlab(existingStorable)
		return true
	}

	return false
}

func (v *CompositeValue) SetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
	value Value,
) bool {
	address := v.StorageAddress()

	value = value.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		map[atree.ValueID]struct{}{
			v.ValueID(): {},
		},
		true, // value is standalone before being set in parent container.
	)

	return v.SetMemberWithoutTransfer(
		interpreter,
		locationRange,
		name,
		value,
	)
}

func (v *CompositeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *CompositeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences, EmptyLocationRange)
}

var emptyCompositeStringLen = len(format.Composite("", nil))

func (v *CompositeValue) MeteredString(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {

	if v.Stringer != nil {
		return v.Stringer(interpreter, v, seenReferences)
	}

	strLen := emptyCompositeStringLen

	var fields []CompositeField

	v.ForEachField(
		interpreter,
		func(fieldName string, fieldValue Value) (resume bool) {
			field := NewCompositeField(
				interpreter,
				fieldName,
				fieldValue,
			)

			fields = append(fields, field)

			strLen += len(field.Name)

			return true
		},
		locationRange,
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

	common.UseMemory(interpreter, common.NewRawStringMemoryUsage(strLen))

	return formatComposite(interpreter, typeId, fields, seenReferences, locationRange)
}

func formatComposite(
	interpreter *Interpreter,
	typeId string,
	fields []CompositeField,
	seenReferences SeenReferences,
	locationRange LocationRange,
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
				Value: field.Value.MeteredString(interpreter, seenReferences, locationRange),
			},
		)
	}

	return format.Composite(typeId, preparedFields)
}

func (v *CompositeValue) GetField(interpreter *Interpreter, locationRange LocationRange, name string) Value {
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

	return MustConvertStoredValue(interpreter, storedValue)
}

func (v *CompositeValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherComposite, ok := other.(*CompositeValue)
	if !ok {
		return false
	}

	if !v.StaticType(interpreter).Equal(otherComposite.StaticType(interpreter)) ||
		v.Kind != otherComposite.Kind ||
		v.dictionary.Count() != otherComposite.dictionary.Count() {

		return false
	}

	iterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	for {
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
		otherValue := otherComposite.GetField(interpreter, locationRange, fieldName)

		equatableValue, ok := MustConvertStoredValue(interpreter, value).(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, locationRange, otherValue) {
			return false
		}
	}
}

// HashInput returns a byte slice containing:
// - HashInputTypeEnum (1 byte)
// - type id (n bytes)
// - hash input of raw value field name (n bytes)
func (v *CompositeValue) HashInput(interpreter *Interpreter, locationRange LocationRange, scratch []byte) []byte {
	if v.Kind == common.CompositeKindEnum {
		typeID := v.TypeID()

		rawValue := v.GetField(interpreter, locationRange, sema.EnumRawValueFieldName)
		rawValueHashInput := rawValue.(HashableValue).
			HashInput(interpreter, locationRange, scratch)

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
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueConformsToStaticTypeTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	staticType := v.StaticType(interpreter)
	semaType := interpreter.MustConvertStaticToSemaType(staticType)

	switch staticType.(type) {
	case *CompositeStaticType:
		return v.CompositeStaticTypeConformsToStaticType(interpreter, locationRange, results, semaType)

	// CompositeValue is also used for storing types which aren't CompositeStaticType.
	// E.g. InclusiveRange.
	case InclusiveRangeStaticType:
		return v.InclusiveRangeStaticTypeConformsToStaticType(interpreter, locationRange, results, semaType)

	default:
		return false
	}
}

func (v *CompositeValue) CompositeStaticTypeConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
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
		base := v.getBaseValue(interpreter, UnauthorizedAccess, locationRange).Value
		if base == nil || !base.ConformsToStaticType(interpreter, locationRange, results) {
			return false
		}
	}

	fieldsLen := v.FieldCount()

	computedFields := v.GetComputedFields(interpreter)
	if computedFields != nil {
		fieldsLen += len(computedFields)
	}

	// The composite might store additional fields
	// which are not statically declared in the composite type.
	if fieldsLen < len(compositeType.Fields) {
		return false
	}

	for _, fieldName := range compositeType.Fields {
		value := v.GetField(interpreter, locationRange, fieldName)
		if value == nil {
			if computedFields == nil {
				return false
			}

			fieldGetter, ok := computedFields[fieldName]
			if !ok {
				return false
			}

			value = fieldGetter(interpreter, locationRange, v)
		}

		member, ok := compositeType.Members.Get(fieldName)
		if !ok {
			return false
		}

		fieldStaticType := value.StaticType(interpreter)

		if !interpreter.IsSubTypeOfSemaType(fieldStaticType, member.TypeAnnotation.Type) {
			return false
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

func (v *CompositeValue) InclusiveRangeStaticTypeConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
	semaType sema.Type,
) bool {
	inclusiveRangeType, ok := semaType.(*sema.InclusiveRangeType)
	if !ok {
		return false
	}

	expectedMemberStaticType := ConvertSemaToStaticType(interpreter, inclusiveRangeType.MemberType)
	for _, fieldName := range sema.InclusiveRangeTypeFieldNames {
		value := v.GetField(interpreter, locationRange, fieldName)

		fieldStaticType := value.StaticType(interpreter)

		// InclusiveRange is non-covariant.
		// For e.g. we disallow assigning InclusiveRange<Int> to an InclusiveRange<Integer>.
		// Hence we do an exact equality check instead of a sub-type check.
		if !fieldStaticType.Equal(expectedMemberStaticType) {
			return false
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
	maxInlineSize uint64,
) (atree.Storable, error) {
	if !v.IsStorable() {
		return NonStorable{Value: v}, nil
	}

	// NOTE: Need to change CompositeValue.UnwrapAtreeValue()
	// if CompositeValue is stored with wrapping.

	return v.dictionary.Storable(storage, address, maxInlineSize)
}

func (v *CompositeValue) UnwrapAtreeValue() (atree.Value, uint64) {
	// Wrapper size is 0 because CompositeValue is stored as
	// atree.OrderedMap without any physical wrapping (see CompositeValue.Storable()).
	return v.dictionary, 0
}

func (v *CompositeValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *CompositeValue) IsResourceKinded(interpreter *Interpreter) bool {
	if v.Kind == common.CompositeKindAttachment {
		return interpreter.MustSemaTypeOfValue(v).IsResourceType()
	}
	return v.Kind == common.CompositeKindResource
}

func (v *CompositeValue) IsReferenceTrackedResourceKindedValue() {}

func (v *CompositeValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.ValueID]struct{},
	hasNoParentContainer bool,
) Value {

	config := interpreter.SharedState.Config

	interpreter.ReportComputation(common.ComputationKindTransferCompositeValue, 1)

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueTransferTrace(
				owner,
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
		panic(RecursiveTransferError{
			LocationRange: locationRange,
		})
	}
	preventTransfer[currentValueID] = struct{}{}
	defer delete(preventTransfer, currentValueID)

	dictionary := v.dictionary

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo && v.Kind == common.CompositeKindContract {
		panic(NonTransferableValueError{
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
		common.UseMemory(interpreter, elementOverhead)
		common.UseMemory(interpreter, dataUse)
		common.UseMemory(interpreter, metaDataUse)

		elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(elementCount, 0)
		common.UseMemory(config.MemoryGauge, elementMemoryUse)

		dictionary, err = atree.NewMapFromBatchData(
			config.Storage,
			address,
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

				// NOTE: key is stringAtreeValue
				// and does not need to be converted or copied

				value := MustConvertStoredValue(interpreter, atreeValue)
				// the base of an attachment is not stored in the atree, so in order to make the
				// transfer happen properly, we set the base value here if this field is an attachment
				if compositeValue, ok := value.(*CompositeValue); ok &&
					compositeValue.Kind == common.CompositeKindAttachment {

					compositeValue.setBaseValue(interpreter, v)
				}

				value = value.Transfer(
					interpreter,
					locationRange,
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

		if remove {
			err = v.dictionary.PopIterate(func(nameStorable atree.Storable, valueStorable atree.Storable) {
				interpreter.RemoveReferencedSlab(nameStorable)
				interpreter.RemoveReferencedSlab(valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			interpreter.maybeValidateAtreeValue(v.dictionary)
			if hasNoParentContainer {
				interpreter.maybeValidateAtreeStorage()
			}

			interpreter.RemoveReferencedSlab(storable)
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

		interpreter.invalidateReferencedResources(v, locationRange)

		v.dictionary = nil
	}

	info := NewCompositeTypeInfo(
		interpreter,
		v.Location,
		v.QualifiedIdentifier,
		v.Kind,
	)

	res := NewCompositeValueFromAtreeMap(
		interpreter,
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

	onResourceOwnerChange := config.OnResourceOwnerChange

	if needsStoreTo &&
		res.Kind == common.CompositeKindResource &&
		onResourceOwnerChange != nil {

		onResourceOwnerChange(
			interpreter,
			res,
			common.Address(currentAddress),
			common.Address(address),
		)
	}

	return res
}

func (v *CompositeValue) ResourceUUID(interpreter *Interpreter, locationRange LocationRange) *UInt64Value {
	fieldValue := v.GetField(interpreter, locationRange, sema.ResourceUUIDFieldName)
	uuid, ok := fieldValue.(UInt64Value)
	if !ok {
		return nil
	}
	return &uuid
}

func (v *CompositeValue) Clone(interpreter *Interpreter) Value {

	iterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	config := interpreter.SharedState.Config

	dictionary, err := atree.NewMapFromBatchData(
		config.Storage,
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
			value := MustConvertStoredValue(interpreter, atreeValue).Clone(interpreter)

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

func (v *CompositeValue) DeepRemove(interpreter *Interpreter, hasNoParentContainer bool) {
	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueDeepRemoveTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.dictionary.Storage

	err := v.dictionary.PopIterate(func(nameStorable atree.Storable, valueStorable atree.Storable) {
		// NOTE: key / field name is stringAtreeValue,
		// and not a Value, so no need to deep remove
		interpreter.RemoveReferencedSlab(nameStorable)

		value := StoredValue(interpreter, valueStorable, storage)
		value.DeepRemove(interpreter, false) // value is an element of v.dictionary because it is from PopIterate() callback.
		interpreter.RemoveReferencedSlab(valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	interpreter.maybeValidateAtreeValue(v.dictionary)
	if hasNoParentContainer {
		interpreter.maybeValidateAtreeStorage()
	}
}

func (v *CompositeValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

// ForEachFieldName iterates over all field names of the composite value.
// It does NOT iterate over computed fields and functions!
func (v *CompositeValue) ForEachFieldName(
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
	v.forEachFieldName(iterate, f)
}

func (v *CompositeValue) forEachFieldName(
	atreeIterate func(fn atree.MapElementIterationFunc) error,
	f func(fieldName string) (resume bool),
) {
	err := atreeIterate(func(key atree.Value) (resume bool, err error) {
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
	interpreter *Interpreter,
	f func(fieldName string, fieldValue Value) (resume bool),
	locationRange LocationRange,
) {
	iterate := func(fn atree.MapEntryIterationFunc) error {
		return v.dictionary.Iterate(
			StringAtreeValueComparator,
			StringAtreeValueHashInput,
			fn,
		)
	}
	v.forEachField(
		interpreter,
		iterate,
		f,
		locationRange,
	)
}

// ForEachReadOnlyLoadedField iterates over all LOADED field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
// DO NOT perform storage mutations in the callback!
func (v *CompositeValue) ForEachReadOnlyLoadedField(
	interpreter *Interpreter,
	f func(fieldName string, fieldValue Value) (resume bool),
	locationRange LocationRange,
) {
	v.forEachField(
		interpreter,
		v.dictionary.IterateReadOnlyLoadedValues,
		f,
		locationRange,
	)
}

func (v *CompositeValue) forEachField(
	interpreter *Interpreter,
	atreeIterate func(fn atree.MapEntryIterationFunc) error,
	f func(fieldName string, fieldValue Value) (resume bool),
	locationRange LocationRange,
) {
	err := atreeIterate(func(key atree.Value, atreeValue atree.Value) (resume bool, err error) {
		value := MustConvertStoredValue(interpreter, atreeValue)
		interpreter.checkInvalidatedResourceOrResourceReference(value, locationRange)

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
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) {

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

	interpreter.maybeValidateAtreeValue(v.dictionary)
	interpreter.maybeValidateAtreeStorage()

	// Key

	// NOTE: key / field name is stringAtreeValue,
	// and not a Value, so no need to deep remove
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value
	existingValue := StoredValue(interpreter, existingValueStorable, interpreter.Storage())
	interpreter.checkResourceLoss(existingValue, locationRange)
	existingValue.DeepRemove(interpreter, true) // existingValue is standalone because it was removed from parent container.
	interpreter.RemoveReferencedSlab(existingValueStorable)
}

func (v *CompositeValue) SetNestedVariables(variables map[string]Variable) {
	v.NestedVariables = variables
}

func NewEnumCaseValue(
	interpreter *Interpreter,
	locationRange LocationRange,
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
		interpreter,
		locationRange,
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
	interpreter *Interpreter,
	functionAuthorization Authorization,
	locationRange LocationRange,
) *EphemeralReferenceValue {
	attachmentType, ok := interpreter.MustSemaTypeOfValue(v).(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var baseType sema.Type
	switch ty := attachmentType.GetBaseType().(type) {
	case *sema.InterfaceType:
		baseType, _ = ty.RewriteWithIntersectionTypes()
	default:
		baseType = ty
	}

	return NewEphemeralReferenceValue(interpreter, functionAuthorization, v.base, baseType, locationRange)
}

func (v *CompositeValue) setBaseValue(_ *Interpreter, base *CompositeValue) {
	v.base = base
}

func AttachmentMemberName(typeID string) string {
	return unrepresentableNamePrefix + typeID
}

func (v *CompositeValue) getAttachmentValue(interpreter *Interpreter, locationRange LocationRange, ty sema.Type) *CompositeValue {
	attachment := v.GetMember(
		interpreter,
		locationRange,
		AttachmentMemberName(string(ty.ID())),
	)
	if attachment != nil {
		return attachment.(*CompositeValue)
	}
	return nil
}

func (v *CompositeValue) GetAttachments(interpreter *Interpreter, locationRange LocationRange) []*CompositeValue {
	var attachments []*CompositeValue
	v.forEachAttachment(interpreter, locationRange, func(attachment *CompositeValue) {
		attachments = append(attachments, attachment)
	})
	return attachments
}

func (v *CompositeValue) forEachAttachmentFunction(interpreter *Interpreter, locationRange LocationRange) Value {
	compositeType := interpreter.MustSemaTypeOfValue(v).(*sema.CompositeType)
	return NewBoundHostFunctionValue(
		interpreter,
		v,
		sema.CompositeForEachAttachmentFunctionType(
			compositeType.GetCompositeKind(),
		),
		func(v *CompositeValue, invocation Invocation) Value {
			inter := invocation.Interpreter

			functionValue, ok := invocation.Arguments[0].(FunctionValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			functionValueType := functionValue.FunctionType()
			parameterTypes := functionValueType.ParameterTypes()
			returnType := functionValueType.ReturnTypeAnnotation.Type

			fn := func(attachment *CompositeValue) {

				attachmentType := inter.MustSemaTypeOfValue(attachment).(*sema.CompositeType)

				attachmentReference := NewEphemeralReferenceValue(
					inter,
					// attachments are unauthorized during iteration
					UnauthorizedAccess,
					attachment,
					attachmentType,
					locationRange,
				)

				referenceType := sema.NewReferenceType(
					inter,
					// attachments are unauthorized during iteration
					sema.UnauthorizedAccess,
					attachmentType,
				)

				inter.invokeFunctionValue(
					functionValue,
					[]Value{attachmentReference},
					nil,
					[]sema.Type{referenceType},
					parameterTypes,
					returnType,
					nil,
					locationRange,
				)
			}

			v.forEachAttachment(inter, locationRange, fn)
			return Void
		},
	)
}

func attachmentBaseAndSelfValues(
	interpreter *Interpreter,
	fnAccess sema.Access,
	v *CompositeValue,
	locationRange LocationRange,
) (base *EphemeralReferenceValue, self *EphemeralReferenceValue) {
	attachmentReferenceAuth := ConvertSemaAccessToStaticAuthorization(interpreter, fnAccess)

	base = v.getBaseValue(interpreter, attachmentReferenceAuth, locationRange)
	// in attachment functions, self is a reference value
	self = NewEphemeralReferenceValue(
		interpreter,
		attachmentReferenceAuth,
		v,
		interpreter.MustSemaTypeOfValue(v),
		locationRange,
	)

	return
}

func (v *CompositeValue) forEachAttachment(
	interpreter *Interpreter,
	locationRange LocationRange,
	f func(*CompositeValue),
) {
	// The attachment iteration creates an implicit reference to the composite, and holds onto that referenced-value.
	// But the reference could get invalidated during the iteration, making that referenced-value invalid.
	// We create a reference here for the purposes of tracking it during iteration.
	vType := interpreter.MustSemaTypeOfValue(v)
	compositeReference := NewEphemeralReferenceValue(interpreter, UnauthorizedAccess, v, vType, locationRange)
	interpreter.maybeTrackReferencedResourceKindedValue(compositeReference)
	forEachAttachment(interpreter, compositeReference, locationRange, f)
}

func forEachAttachment(
	interpreter *Interpreter,
	compositeReference *EphemeralReferenceValue,
	locationRange LocationRange,
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

	oldSharedState := interpreter.SharedState.inAttachmentIteration(composite)
	interpreter.SharedState.setAttachmentIteration(composite, true)
	defer func() {
		interpreter.SharedState.setAttachmentIteration(composite, oldSharedState)
	}()

	for {
		// Check that the implicit composite reference was not invalidated during iteration
		interpreter.checkInvalidatedResourceOrResourceReference(compositeReference, locationRange)
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			break
		}
		if strings.HasPrefix(string(key.(StringAtreeValue)), unrepresentableNamePrefix) {
			attachment, ok := MustConvertStoredValue(interpreter, value).(*CompositeValue)
			if !ok {
				panic(errors.NewExternalError(err))
			}
			// `f` takes the `attachment` value directly, but if a method to later iterate over
			// attachments is added that takes a `fun (&Attachment): Void` callback, the `f` provided here
			// should convert the provided attachment value into a reference before passing it to the user
			// callback
			attachment.setBaseValue(interpreter, composite)
			f(attachment)
		}
	}
}

func (v *CompositeValue) getTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyType sema.Type,
	baseAccess sema.Access,
) Value {
	attachment := v.getAttachmentValue(interpreter, locationRange, keyType)
	if attachment == nil {
		return Nil
	}
	attachmentType := keyType.(*sema.CompositeType)
	// dynamically set the attachment's base to this composite
	attachment.setBaseValue(interpreter, v)

	// The attachment reference has the same entitlements as the base access
	attachmentRef := NewEphemeralReferenceValue(
		interpreter,
		ConvertSemaAccessToStaticAuthorization(interpreter, baseAccess),
		attachment,
		attachmentType,
		locationRange,
	)

	return NewSomeValueNonCopying(interpreter, attachmentRef)
}

func (v *CompositeValue) GetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	ty sema.Type,
) Value {
	access := sema.UnauthorizedAccess
	attachmentTyp, isAttachmentType := ty.(*sema.CompositeType)
	if isAttachmentType {
		access = attachmentTyp.SupportedEntitlements().Access()
	}
	return v.getTypeKey(interpreter, locationRange, ty, access)
}

func (v *CompositeValue) SetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	attachmentType sema.Type,
	attachment Value,
) {
	memberName := AttachmentMemberName(string(attachmentType.ID()))
	if v.SetMember(interpreter, locationRange, memberName, attachment) {
		panic(DuplicateAttachmentError{
			AttachmentType: attachmentType,
			Value:          v,
			LocationRange:  locationRange,
		})
	}
}

func (v *CompositeValue) RemoveTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	attachmentType sema.Type,
) Value {
	memberName := AttachmentMemberName(string(attachmentType.ID()))
	return v.RemoveMember(interpreter, locationRange, memberName)
}

func (v *CompositeValue) Iterator(interpreter *Interpreter, locationRange LocationRange) ValueIterator {
	staticType := v.StaticType(interpreter)

	switch typ := staticType.(type) {
	case InclusiveRangeStaticType:
		return NewInclusiveRangeIterator(interpreter, locationRange, v, typ)

	default:
		// Must be caught in the checker.
		panic(errors.NewUnreachableError())
	}
}

func (v *CompositeValue) ForEach(
	interpreter *Interpreter,
	_ sema.Type,
	function func(value Value) (resume bool),
	transferElements bool,
	locationRange LocationRange,
) {
	iterator := v.Iterator(interpreter, locationRange)
	for {
		value := iterator.Next(interpreter, locationRange)
		if value == nil {
			return
		}

		if transferElements {
			// Each element must be transferred before passing onto the function.
			value = value.Transfer(
				interpreter,
				locationRange,
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
