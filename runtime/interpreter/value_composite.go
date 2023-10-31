package interpreter

import (
	goerrors "errors"
	"strings"
	"time"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// CompositeValue

type CompositeValue struct {
	Destructor      FunctionValue
	Location        common.Location
	staticType      StaticType
	Stringer        func(gauge common.MemoryGauge, value *CompositeValue, seenReferences SeenReferences) string
	injectedFields  map[string]Value
	computedFields  map[string]ComputedField
	NestedVariables map[string]*Variable
	Functions       map[string]FunctionValue
	dictionary      *atree.OrderedMap
	typeID          TypeID

	// attachments also have a reference to their base value. This field is set in three cases:
	// 1) when an attachment `A` is accessed off `v` using `v[A]`, this is set to `&v`
	// 2) When a resource `r`'s destructor is invoked, all of `r`'s attachments' destructors will also run, and
	//    have their `base` fields set to `&r`
	// 3) When a value is transferred, this field is copied between its attachments
	base                *EphemeralReferenceValue
	QualifiedIdentifier string
	Kind                common.CompositeKind
	isDestroyed         bool
}

type ComputedField func(*Interpreter, LocationRange) Value

type CompositeField struct {
	Value Value
	Name  string
}

const attachmentNamePrefix = "$"

var _ TypeIndexableValue = &CompositeValue{}

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

func newCompositeValueFromOrderedMap(
	dict *atree.OrderedMap,
	typeInfo compositeTypeInfo,
) *CompositeValue {
	return &CompositeValue{
		dictionary:          dict,
		Location:            typeInfo.location,
		QualifiedIdentifier: typeInfo.qualifiedIdentifier,
		Kind:                typeInfo.kind,
	}
}

func newCompositeValueFromConstructor(
	gauge common.MemoryGauge,
	count uint64,
	typeInfo compositeTypeInfo,
	constructor func() *atree.OrderedMap,
) *CompositeValue {
	baseUse, elementOverhead, dataUse, metaDataUse := common.NewCompositeMemoryUsages(count, 0)
	common.UseMemory(gauge, baseUse)
	common.UseMemory(gauge, elementOverhead)
	common.UseMemory(gauge, dataUse)
	common.UseMemory(gauge, metaDataUse)
	return newCompositeValueFromOrderedMap(constructor(), typeInfo)
}

var _ Value = &CompositeValue{}
var _ EquatableValue = &CompositeValue{}
var _ HashableValue = &CompositeValue{}
var _ MemberAccessibleValue = &CompositeValue{}
var _ ReferenceTrackedResourceKindedValue = &CompositeValue{}
var _ ContractValue = &CompositeValue{}

func (*CompositeValue) isValue() {}

func (v *CompositeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitCompositeValue(interpreter, v)
	if !descend {
		return
	}

	v.ForEachField(interpreter, func(_ string, value Value) (resume bool) {
		value.Accept(interpreter, visitor)

		// continue iteration
		return true
	})
}

// Walk iterates over all field values of the composite value.
// It does NOT walk the computed field or functions!
func (v *CompositeValue) Walk(interpreter *Interpreter, walkChild func(Value)) {
	v.ForEachField(interpreter, func(_ string, value Value) (resume bool) {
		walkChild(value)

		// continue iteration
		return true
	})
}

func (v *CompositeValue) StaticType(interpreter *Interpreter) StaticType {
	if v.staticType == nil {
		// NOTE: Instead of using NewCompositeStaticType, which always generates the type ID,
		// use the TypeID accessor, which may return an already computed type ID
		v.staticType = NewCompositeStaticType(
			interpreter,
			v.Location,
			v.QualifiedIdentifier,
			v.TypeID(), // TODO TypeID metering
		)
	}
	return v.staticType
}

func (v *CompositeValue) IsImportable(inter *Interpreter) bool {
	// Check type is importable
	staticType := v.StaticType(inter)
	semaType := inter.MustConvertStaticToSemaType(staticType)
	if !semaType.IsImportable(map[*sema.Member]bool{}) {
		return false
	}

	// Check all field values are importable
	importable := true
	v.ForEachField(inter, func(_ string, value Value) (resume bool) {
		if !value.IsImportable(inter) {
			importable = false
			// stop iteration
			return false
		}

		// continue iteration
		return true
	})

	return importable
}

func (v *CompositeValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *CompositeValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyCompositeValue, 1)

	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

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

	storageID := v.StorageID()

	interpreter.withResourceDestruction(
		storageID,
		locationRange,
		func() {
			// if this type has attachments, destroy all of them before invoking the destructor
			v.forEachAttachment(interpreter, locationRange, func(attachment *CompositeValue) {
				// an attachment's destructor may make reference to `base`, so we must set the base value
				// for the attachment before invoking its destructor. For other functions, this happens
				// automatically when the attachment is accessed with the access expression `v[A]`, which
				// is a necessary pre-requisite for calling any members of the attachment. However, in
				// the case of a destructor, this is called implicitly, and thus must have its `base`
				// set manually
				attachment.setBaseValue(interpreter, v)
				attachment.Destroy(interpreter, locationRange)
			})

			interpreter = v.getInterpreter(interpreter)

			// if composite was deserialized, dynamically link in the destructor
			if v.Destructor == nil {
				v.Destructor = interpreter.SharedState.typeCodes.CompositeCodes[v.TypeID()].DestructorFunction
			}

			destructor := v.Destructor

			if destructor != nil {
				var base *EphemeralReferenceValue
				var self MemberAccessibleValue = v
				if v.Kind == common.CompositeKindAttachment {
					base, self = attachmentBaseAndSelfValues(interpreter, v)
				}
				invocation := NewInvocation(
					interpreter,
					&self,
					base,
					nil,
					nil,
					nil,
					locationRange,
				)

				destructor.invoke(invocation)
			}
		},
	)

	v.isDestroyed = true

	if config.InvalidatedResourceValidationEnabled {
		v.dictionary = nil
	}

	interpreter.updateReferencedResource(
		storageID,
		storageID,
		func(value ReferenceTrackedResourceKindedValue) {
			compositeValue, ok := value.(*CompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			compositeValue.isDestroyed = true

			if config.InvalidatedResourceValidationEnabled {
				compositeValue.dictionary = nil
			}
		},
	)

}

func (v *CompositeValue) getBuiltinMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {

	switch name {
	case sema.ResourceOwnerFieldName:
		if v.Kind == common.CompositeKindResource {
			return v.OwnerValue(interpreter, locationRange)
		}
	}

	return nil
}

func (v *CompositeValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

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
		return builtin
	}

	// Give computed fields precedence over stored fields for built-in types
	if v.Location == nil {
		if computedField := v.GetComputedField(interpreter, locationRange, name); computedField != nil {
			return computedField
		}
	}

	if field := v.GetField(interpreter, locationRange, name); field != nil {
		return field
	}

	if v.NestedVariables != nil {
		variable, ok := v.NestedVariables[name]
		if ok {
			return variable.GetValue()
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

func (v *CompositeValue) checkInvalidatedResourceUse(locationRange LocationRange) {
	if v.isDestroyed || (v.dictionary == nil && v.Kind == common.CompositeKindResource) {
		panic(InvalidatedResourceError{
			LocationRange: locationRange,
		})
	}
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

	return computedField(interpreter, locationRange)
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

	function, ok := v.Functions[name]
	if !ok {
		return nil
	}

	var base *EphemeralReferenceValue
	var self MemberAccessibleValue = v
	if v.Kind == common.CompositeKindAttachment {
		base, self = attachmentBaseAndSelfValues(interpreter, v)
	}
	return NewBoundFunctionValue(interpreter, function, &self, base)
}

func (v *CompositeValue) OwnerValue(interpreter *Interpreter, locationRange LocationRange) OptionalValue {
	address := v.StorageAddress()

	if address == (atree.Address{}) {
		return NilOptionalValue
	}

	config := interpreter.SharedState.Config

	ownerAccount := config.PublicAccountHandler(AddressValue(address))

	// Owner must be of `PublicAccount` type.
	interpreter.ExpectType(ownerAccount, sema.PublicAccountType, locationRange)

	return NewSomeValueNonCopying(interpreter, ownerAccount)
}

func (v *CompositeValue) RemoveMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {

	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

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
		)
}

func (v *CompositeValue) SetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
	value Value,
) bool {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

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

	address := v.StorageAddress()

	value = value.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		map[atree.StorageID]struct{}{
			v.StorageID(): {},
		},
	)

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

	if existingStorable != nil {
		existingValue := StoredValue(interpreter, existingStorable, config.Storage)

		existingValue.DeepRemove(interpreter)

		interpreter.RemoveReferencedSlab(existingStorable)
		return true
	}

	return false
}

func (v *CompositeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *CompositeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences)
}

var emptyCompositeStringLen = len(format.Composite("", nil))

func (v *CompositeValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {

	if v.Stringer != nil {
		return v.Stringer(memoryGauge, v, seenReferences)
	}

	strLen := emptyCompositeStringLen

	var fields []CompositeField
	_ = v.dictionary.Iterate(func(key atree.Value, value atree.Value) (resume bool, err error) {
		field := NewCompositeField(
			memoryGauge,
			string(key.(StringAtreeValue)),
			MustConvertStoredValue(memoryGauge, value),
		)

		fields = append(fields, field)

		strLen += len(field.Name)

		return true, nil
	})

	typeId := string(v.TypeID())

	// bodyLen = len(fieldNames) + len(typeId) + (n times colon+space) + ((n-1) times comma+space)
	//         = len(fieldNames) + len(typeId) + 2n + 2n - 2
	//         = len(fieldNames) + len(typeId) + 4n - 2
	//
	// Since (-2) only occurs if its non-empty, ignore the (-2). i.e: overestimate
	// 		bodyLen = len(fieldNames) + len(typeId) + 4n
	//
	strLen = strLen + len(typeId) + len(fields)*4

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))

	return formatComposite(memoryGauge, typeId, fields, seenReferences)
}

func formatComposite(memoryGauge common.MemoryGauge, typeId string, fields []CompositeField, seenReferences SeenReferences) string {
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
				Value: field.Value.MeteredString(memoryGauge, seenReferences),
			},
		)
	}

	return format.Composite(typeId, preparedFields)
}

func (v *CompositeValue) GetField(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

	storable, err := v.dictionary.Get(
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

	return StoredValue(interpreter, storable, v.dictionary.Storage)
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

	iterator, err := v.dictionary.Iterator()
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

	staticType := v.StaticType(interpreter).(CompositeStaticType)

	semaType := interpreter.MustConvertStaticToSemaType(staticType)

	compositeType, ok := semaType.(*sema.CompositeType)
	if !ok ||
		v.Kind != compositeType.Kind ||
		v.TypeID() != compositeType.ID() {

		return false
	}

	if compositeType.Kind == common.CompositeKindAttachment {
		base := v.getBaseValue().Value
		if base == nil || !base.ConformsToStaticType(interpreter, locationRange, results) {
			return false
		}
	}

	fieldsLen := int(v.dictionary.Count())

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

			value = fieldGetter(interpreter, locationRange)
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

	return v.dictionary.Storable(storage, address, maxInlineSize)
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
	preventTransfer map[atree.StorageID]struct{},
) Value {

	baseUse, elementOverhead, dataUse, metaDataUse := common.NewCompositeMemoryUsages(v.dictionary.Count(), 0)
	common.UseMemory(interpreter, baseUse)
	common.UseMemory(interpreter, elementOverhead)
	common.UseMemory(interpreter, dataUse)
	common.UseMemory(interpreter, metaDataUse)

	interpreter.ReportComputation(common.ComputationKindTransferCompositeValue, 1)

	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

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

	currentStorageID := v.StorageID()
	currentAddress := v.StorageAddress()

	if preventTransfer == nil {
		preventTransfer = map[atree.StorageID]struct{}{}
	} else if _, ok := preventTransfer[currentStorageID]; ok {
		panic(RecursiveTransferError{
			LocationRange: locationRange,
		})
	}
	preventTransfer[currentStorageID] = struct{}{}
	defer delete(preventTransfer, currentStorageID)

	dictionary := v.dictionary

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo && v.Kind == common.CompositeKindContract {
		panic(NonTransferableValueError{
			Value: v,
		})
	}

	if needsStoreTo || !isResourceKinded {
		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(v.dictionary.Count(), 0)
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

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	var res *CompositeValue

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource as invalidated, by unsetting the backing dictionary.
		// This allows raising an error when the resource is attempted
		// to be transferred/moved again (see beginning of this function)

		if config.InvalidatedResourceValidationEnabled {
			v.dictionary = nil
		} else {
			v.dictionary = dictionary
			res = v
		}

		newStorageID := dictionary.StorageID()

		interpreter.updateReferencedResource(
			currentStorageID,
			newStorageID,
			func(value ReferenceTrackedResourceKindedValue) {
				compositeValue, ok := value.(*CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				compositeValue.dictionary = dictionary
			},
		)
	}

	if res == nil {
		info := NewCompositeTypeInfo(
			interpreter,
			v.Location,
			v.QualifiedIdentifier,
			v.Kind,
		)
		res = newCompositeValueFromOrderedMap(dictionary, info)
		res.injectedFields = v.injectedFields
		res.computedFields = v.computedFields
		res.NestedVariables = v.NestedVariables
		res.Functions = v.Functions
		res.Destructor = v.Destructor
		res.Stringer = v.Stringer
		res.isDestroyed = v.isDestroyed
		res.typeID = v.typeID
		res.staticType = v.staticType
		res.base = v.base
	}

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

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	config := interpreter.SharedState.Config

	elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(v.dictionary.Count(), 0)
	common.UseMemory(config.MemoryGauge, elementMemoryUse)

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
		Destructor:          v.Destructor,
		Stringer:            v.Stringer,
		isDestroyed:         v.isDestroyed,
		typeID:              v.typeID,
		staticType:          v.staticType,
		base:                v.base,
	}
}

func (v *CompositeValue) DeepRemove(interpreter *Interpreter) {
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
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)
}

func (v *CompositeValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

// ForEachField iterates over all field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
func (v *CompositeValue) ForEachField(
	gauge common.MemoryGauge,
	f func(fieldName string, fieldValue Value) (resume bool),
) {

	err := v.dictionary.Iterate(func(key atree.Value, value atree.Value) (resume bool, err error) {
		resume = f(
			string(key.(StringAtreeValue)),
			MustConvertStoredValue(gauge, value),
		)
		return
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}

func (v *CompositeValue) StorageID() atree.StorageID {
	return v.dictionary.StorageID()
}

func (v *CompositeValue) StorageAddress() atree.Address {
	return v.dictionary.Address()
}

func (v *CompositeValue) RemoveField(
	interpreter *Interpreter,
	_ LocationRange,
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

	// Key

	// NOTE: key / field name is stringAtreeValue,
	// and not a Value, so no need to deep remove
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value
	existingValue := StoredValue(interpreter, existingValueStorable, interpreter.Storage())
	existingValue.DeepRemove(interpreter)
	interpreter.RemoveReferencedSlab(existingValueStorable)
}

func (v *CompositeValue) SetNestedVariables(variables map[string]*Variable) {
	v.NestedVariables = variables
}

func NewEnumCaseValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	enumType *sema.CompositeType,
	rawValue NumberValue,
	functions map[string]FunctionValue,
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

func (v *CompositeValue) getBaseValue() *EphemeralReferenceValue {
	return v.base
}

func (v *CompositeValue) setBaseValue(interpreter *Interpreter, base *CompositeValue) {
	attachmentType, ok := interpreter.MustSemaTypeOfValue(v).(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var baseType sema.Type
	switch ty := attachmentType.GetBaseType().(type) {
	case *sema.InterfaceType:
		baseType, _ = ty.RewriteWithRestrictedTypes()
	default:
		baseType = ty
	}

	// the base reference can only be borrowed with the declared type of the attachment's base
	v.base = NewEphemeralReferenceValue(interpreter, false, base, baseType)

	interpreter.trackReferencedResourceKindedValue(base.StorageID(), base)
}

func attachmentMemberName(ty sema.Type) string {
	return attachmentNamePrefix + string(ty.ID())
}

func (v *CompositeValue) getAttachmentValue(interpreter *Interpreter, locationRange LocationRange, ty sema.Type) *CompositeValue {
	if attachment := v.GetMember(interpreter, locationRange, attachmentMemberName(ty)); attachment != nil {
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

func attachmentBaseAndSelfValues(
	interpreter *Interpreter,
	v *CompositeValue,
) (base *EphemeralReferenceValue, self *EphemeralReferenceValue) {
	base = v.getBaseValue()
	// in attachment functions, self is a reference value
	self = NewEphemeralReferenceValue(interpreter, false, v, interpreter.MustSemaTypeOfValue(v))
	interpreter.trackReferencedResourceKindedValue(v.StorageID(), v)
	return
}

func (v *CompositeValue) forEachAttachment(interpreter *Interpreter, _ LocationRange, f func(*CompositeValue)) {
	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	oldSharedState := interpreter.SharedState.inAttachmentIteration(v)
	interpreter.SharedState.setAttachmentIteration(v, true)
	defer func() {
		interpreter.SharedState.setAttachmentIteration(v, oldSharedState)
	}()

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			break
		}
		if strings.HasPrefix(string(key.(StringAtreeValue)), attachmentNamePrefix) {
			attachment, ok := MustConvertStoredValue(interpreter, value).(*CompositeValue)
			if !ok {
				panic(errors.NewExternalError(err))
			}
			// `f` takes the `attachment` value directly, but if a method to later iterate over
			// attachments is added that takes a `fun (&Attachment): Void` callback, the `f` provided here
			// should convert the provided attachment value into a reference before passing it to the user
			// callback
			f(attachment)
		}
	}
}

func (v *CompositeValue) GetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	ty sema.Type,
) Value {
	attachment := v.getAttachmentValue(interpreter, locationRange, ty)
	if attachment == nil {
		return NilValue{}
	}
	// dynamically set the attachment's base to this composite
	attachment.setBaseValue(interpreter, v)

	attachmentRef := NewEphemeralReferenceValue(interpreter, false, attachment, ty)
	interpreter.trackReferencedResourceKindedValue(attachment.StorageID(), attachment)

	return NewSomeValueNonCopying(interpreter, attachmentRef)
}

func (v *CompositeValue) SetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	attachmentType sema.Type,
	attachment Value,
) {
	if v.SetMember(interpreter, locationRange, attachmentMemberName(attachmentType), attachment) {
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
	return v.RemoveMember(interpreter, locationRange, attachmentMemberName(attachmentType))
}
