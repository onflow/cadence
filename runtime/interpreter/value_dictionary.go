package interpreter

import (
	goerrors "errors"
	"time"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// DictionaryValue

type DictionaryValue struct {
	Type             DictionaryStaticType
	semaType         *sema.DictionaryType
	isResourceKinded *bool
	dictionary       *atree.OrderedMap
	isDestroyed      bool
	elementSize      uint
}

func NewDictionaryValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	dictionaryType DictionaryStaticType,
	keysAndValues ...Value,
) *DictionaryValue {
	return NewDictionaryValueWithAddress(
		interpreter,
		locationRange,
		dictionaryType,
		common.ZeroAddress,
		keysAndValues...,
	)
}

func NewDictionaryValueWithAddress(
	interpreter *Interpreter,
	locationRange LocationRange,
	dictionaryType DictionaryStaticType,
	address common.Address,
	keysAndValues ...Value,
) *DictionaryValue {

	interpreter.ReportComputation(common.ComputationKindCreateDictionaryValue, 1)

	var v *DictionaryValue

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			interpreter.reportDictionaryValueConstructTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	keysAndValuesCount := len(keysAndValues)
	if keysAndValuesCount%2 != 0 {
		panic("uneven number of keys and values")
	}

	constructor := func() *atree.OrderedMap {
		dictionary, err := atree.NewMap(
			config.Storage,
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			dictionaryType,
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return dictionary
	}

	// values are added to the dictionary after creation, not here
	v = newDictionaryValueFromConstructor(interpreter, dictionaryType, 0, constructor)

	// NOTE: lazily initialized when needed for performance reasons
	var lazyIsResourceTyped *bool

	for i := 0; i < keysAndValuesCount; i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		existingValue := v.Insert(interpreter, locationRange, key, value)
		// If the dictionary already contained a value for the key,
		// and the dictionary is resource-typed,
		// then we need to prevent a resource loss
		if _, ok := existingValue.(*SomeValue); ok {
			// Lazily determine if the dictionary is resource-typed, once
			if lazyIsResourceTyped == nil {
				isResourceTyped := v.SemaType(interpreter).IsResourceType()
				lazyIsResourceTyped = &isResourceTyped
			}
			if *lazyIsResourceTyped {
				panic(DuplicateKeyInResourceDictionaryError{
					LocationRange: locationRange,
				})
			}
		}
	}

	return v
}

func newDictionaryValueFromOrderedMap(
	dict *atree.OrderedMap,
	staticType DictionaryStaticType,
) *DictionaryValue {
	return &DictionaryValue{
		Type:       staticType,
		dictionary: dict,
	}
}

func newDictionaryValueFromConstructor(
	gauge common.MemoryGauge,
	staticType DictionaryStaticType,
	count uint64,
	constructor func() *atree.OrderedMap,
) (dict *DictionaryValue) {

	keySize := staticType.KeyType.elementSize()
	valueSize := staticType.ValueType.elementSize()
	var elementSize uint
	if keySize != 0 && valueSize != 0 {
		elementSize = keySize + valueSize
	}
	baseUsage, overheadUsage, dataSlabs, metaDataSlabs := common.NewDictionaryMemoryUsages(count, elementSize)
	common.UseMemory(gauge, baseUsage)
	common.UseMemory(gauge, overheadUsage)
	common.UseMemory(gauge, dataSlabs)
	common.UseMemory(gauge, metaDataSlabs)

	dict = newDictionaryValueFromOrderedMap(constructor(), staticType)
	dict.elementSize = elementSize
	return
}

var _ Value = &DictionaryValue{}
var _ atree.Value = &DictionaryValue{}
var _ EquatableValue = &DictionaryValue{}
var _ ValueIndexableValue = &DictionaryValue{}
var _ MemberAccessibleValue = &DictionaryValue{}
var _ ReferenceTrackedResourceKindedValue = &DictionaryValue{}

func (*DictionaryValue) isValue() {}

func (v *DictionaryValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitDictionaryValue(interpreter, v)
	if !descend {
		return
	}

	v.Walk(interpreter, func(value Value) {
		value.Accept(interpreter, visitor)
	})
}

func (v *DictionaryValue) Iterate(interpreter *Interpreter, f func(key, value Value) (resume bool)) {
	iterate := func() {
		err := v.dictionary.Iterate(func(key, value atree.Value) (resume bool, err error) {
			// atree.OrderedMap iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			resume = f(
				MustConvertStoredValue(interpreter, key),
				MustConvertStoredValue(interpreter, value),
			)

			return resume, nil
		})
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}
	if v.IsResourceKinded(interpreter) {
		interpreter.withMutationPrevention(v.StorageID(), iterate)
	} else {
		iterate()
	}
}

type DictionaryIterator struct {
	mapIterator *atree.MapIterator
}

func (i DictionaryIterator) NextKey(gauge common.MemoryGauge) Value {
	atreeValue, err := i.mapIterator.NextKey()
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	if atreeValue == nil {
		return nil
	}
	return MustConvertStoredValue(gauge, atreeValue)
}

func (v *DictionaryValue) Iterator() DictionaryIterator {
	mapIterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return DictionaryIterator{
		mapIterator: mapIterator,
	}
}

func (v *DictionaryValue) Walk(interpreter *Interpreter, walkChild func(Value)) {
	v.Iterate(interpreter, func(key, value Value) (resume bool) {
		walkChild(key)
		walkChild(value)
		return true
	})
}

func (v *DictionaryValue) StaticType(_ *Interpreter) StaticType {
	// TODO meter
	return v.Type
}

func (v *DictionaryValue) IsImportable(inter *Interpreter) bool {
	importable := true
	v.Iterate(inter, func(key, value Value) (resume bool) {
		if !key.IsImportable(inter) || !value.IsImportable(inter) {
			importable = false
			// stop iteration
			return false
		}

		// continue iteration
		return true
	})

	return importable
}

func (v *DictionaryValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *DictionaryValue) checkInvalidatedResourceUse(interpreter *Interpreter, locationRange LocationRange) {
	if v.isDestroyed || (v.dictionary == nil && v.IsResourceKinded(interpreter)) {
		panic(InvalidatedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (v *DictionaryValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyDictionaryValue, 1)

	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportDictionaryValueDestroyTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	storageID := v.StorageID()

	interpreter.withResourceDestruction(
		storageID,
		locationRange,
		func() {
			v.Iterate(interpreter, func(key, value Value) (resume bool) {
				// Resources cannot be keys at the moment, so should theoretically not be needed
				maybeDestroy(interpreter, locationRange, key)
				maybeDestroy(interpreter, locationRange, value)

				return true
			})
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
			dictionaryValue, ok := value.(*DictionaryValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			dictionaryValue.isDestroyed = true

			if config.InvalidatedResourceValidationEnabled {
				dictionaryValue.dictionary = nil
			}
		},
	)
}

func (v *DictionaryValue) ForEachKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
) {
	keyType := v.SemaType(interpreter).KeyType

	iterationInvocation := func(key Value) Invocation {
		return NewInvocation(
			interpreter,
			nil,
			nil,
			[]Value{key},
			[]sema.Type{keyType},
			nil,
			locationRange,
		)
	}

	iterate := func() {
		err := v.dictionary.IterateKeys(
			func(item atree.Value) (bool, error) {
				key := MustConvertStoredValue(interpreter, item)

				shouldContinue, ok := procedure.invoke(iterationInvocation(key)).(BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return bool(shouldContinue), nil
			},
		)

		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	if v.IsResourceKinded(interpreter) {
		interpreter.withMutationPrevention(v.StorageID(), iterate)
	} else {
		iterate()
	}
}

func (v *DictionaryValue) ContainsKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) BoolValue {

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	exists, err := v.dictionary.Has(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	return AsBoolValue(exists)
}

func (v *DictionaryValue) Get(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) (Value, bool) {

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	storable, err := v.dictionary.Get(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil, false
		}
		panic(errors.NewExternalError(err))
	}

	storage := v.dictionary.Storage
	value := StoredValue(interpreter, storable, storage)
	return value, true
}

func (v *DictionaryValue) GetKey(interpreter *Interpreter, locationRange LocationRange, keyValue Value) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	value, ok := v.Get(interpreter, locationRange, keyValue)
	if ok {
		return NewSomeValueNonCopying(interpreter, value)
	}

	return Nil
}

func (v *DictionaryValue) SetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
	value Value,
) {
	interpreter.validateMutation(v.StorageID(), locationRange)

	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	interpreter.checkContainerMutation(v.Type.KeyType, keyValue, locationRange)
	interpreter.checkContainerMutation(
		OptionalStaticType{ // intentionally unmetered
			Type: v.Type.ValueType,
		},
		value,
		locationRange,
	)

	switch value := value.(type) {
	case *SomeValue:
		innerValue := value.InnerValue(interpreter, locationRange)
		_ = v.Insert(interpreter, locationRange, keyValue, innerValue)

	case NilValue:
		_ = v.Remove(interpreter, locationRange, keyValue)

	case placeholderValue:
		// NO-OP

	default:
		panic(errors.NewUnreachableError())
	}
}

func (v *DictionaryValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *DictionaryValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences)
}

func (v *DictionaryValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {

	pairs := make([]struct {
		Key   string
		Value string
	}, v.Count())

	index := 0
	_ = v.dictionary.Iterate(func(key, value atree.Value) (resume bool, err error) {
		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value

		pairs[index] = struct {
			Key   string
			Value string
		}{
			Key:   MustConvertStoredValue(memoryGauge, key).MeteredString(memoryGauge, seenReferences),
			Value: MustConvertStoredValue(memoryGauge, value).MeteredString(memoryGauge, seenReferences),
		}
		index++
		return true, nil
	})

	// len = len(open-brace) + len(close-brace) + (n times colon+space) + ((n-1) times comma+space)
	//     = 2 + 2n + 2n - 2
	//     = 4n + 2 - 2
	//
	// Since (-2) only occurs if its non-empty (i.e: n>0), ignore the (-2). i.e: overestimate
	//    len = 4n + 2
	//
	// String of each key and value are metered separately.
	strLen := len(pairs)*4 + 2

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))

	return format.Dictionary(pairs)
}

func (v *DictionaryValue) GetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportDictionaryValueGetMemberTrace(
				typeInfo,
				count,
				name,
				time.Since(startTime),
			)
		}()
	}

	switch name {
	case "length":
		return NewIntValueFromInt64(interpreter, int64(v.Count()))

	case "keys":

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		return NewArrayValueWithIterator(
			interpreter,
			NewVariableSizedStaticType(interpreter, v.Type.KeyType),
			common.ZeroAddress,
			v.dictionary.Count(),
			func() Value {

				key, err := iterator.NextKey()
				if err != nil {
					panic(errors.NewExternalError(err))
				}
				if key == nil {
					return nil
				}

				return MustConvertStoredValue(interpreter, key).
					Transfer(
						interpreter,
						locationRange,
						atree.Address{},
						false,
						nil,
						nil,
					)
			},
		)

	case "values":

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		return NewArrayValueWithIterator(
			interpreter,
			NewVariableSizedStaticType(interpreter, v.Type.ValueType),
			common.ZeroAddress,
			v.dictionary.Count(),
			func() Value {

				value, err := iterator.NextValue()
				if err != nil {
					panic(errors.NewExternalError(err))
				}
				if value == nil {
					return nil
				}

				return MustConvertStoredValue(interpreter, value).
					Transfer(
						interpreter,
						locationRange,
						atree.Address{},
						false,
						nil,
						nil,
					)
			})

	case "remove":
		return NewHostFunctionValue(
			interpreter,
			sema.DictionaryRemoveFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				keyValue := invocation.Arguments[0]

				return v.Remove(
					invocation.Interpreter,
					invocation.LocationRange,
					keyValue,
				)
			},
		)

	case "insert":
		return NewHostFunctionValue(
			interpreter,
			sema.DictionaryInsertFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				keyValue := invocation.Arguments[0]
				newValue := invocation.Arguments[1]

				return v.Insert(
					invocation.Interpreter,
					invocation.LocationRange,
					keyValue,
					newValue,
				)
			},
		)

	case "containsKey":
		return NewHostFunctionValue(
			interpreter,
			sema.DictionaryContainsKeyFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				return v.ContainsKey(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)
	case "forEachKey":
		return NewHostFunctionValue(
			interpreter,
			sema.DictionaryForEachKeyFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				v.ForEachKey(
					interpreter,
					invocation.LocationRange,
					funcArgument,
				)

				return Void
			},
		)
	}

	return nil
}

func (v *DictionaryValue) RemoveMember(interpreter *Interpreter, locationRange LocationRange, _ string) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	// Dictionaries have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) SetMember(interpreter *Interpreter, locationRange LocationRange, _ string, _ Value) bool {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	// Dictionaries have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) Count() int {
	return int(v.dictionary.Count())
}

func (v *DictionaryValue) RemoveKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	return v.Remove(interpreter, locationRange, key)
}

func (v *DictionaryValue) Remove(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) OptionalValue {

	interpreter.validateMutation(v.StorageID(), locationRange)

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	// No need to clean up storable for passed-in key value,
	// as atree never calls Storable()
	existingKeyStorable, existingValueStorable, err := v.dictionary.Remove(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return NilOptionalValue
		}
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	storage := interpreter.Storage()

	// Key

	existingKeyValue := StoredValue(interpreter, existingKeyStorable, storage)
	existingKeyValue.DeepRemove(interpreter)
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	existingValue := StoredValue(interpreter, existingValueStorable, storage).
		Transfer(
			interpreter,
			locationRange,
			atree.Address{},
			true,
			existingValueStorable,
			nil,
		)

	return NewSomeValueNonCopying(interpreter, existingValue)
}

func (v *DictionaryValue) InsertKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key, value Value,
) {
	v.SetKey(interpreter, locationRange, key, value)
}

func (v *DictionaryValue) Insert(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue, value Value,
) OptionalValue {

	interpreter.validateMutation(v.StorageID(), locationRange)

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(v.dictionary.Count(), v.elementSize, false)
	common.UseMemory(interpreter, common.AtreeMapElementOverhead)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)

	interpreter.checkContainerMutation(v.Type.KeyType, keyValue, locationRange)
	interpreter.checkContainerMutation(v.Type.ValueType, value, locationRange)

	address := v.dictionary.Address()

	preventTransfer := map[atree.StorageID]struct{}{
		v.StorageID(): {},
	}

	keyValue = keyValue.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
	)

	value = value.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
	)

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	// atree only calls Storable() on keyValue if needed,
	// i.e., if the key is a new key
	existingValueStorable, err := v.dictionary.Set(
		valueComparator,
		hashInputProvider,
		keyValue,
		value,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	if existingValueStorable == nil {
		return NilOptionalValue
	}

	storage := interpreter.Storage()

	existingValue := StoredValue(
		interpreter,
		existingValueStorable,
		storage,
	).Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		true,
		existingValueStorable,
		nil,
	)

	return NewSomeValueNonCopying(interpreter, existingValue)
}

type DictionaryEntryValues struct {
	Key   Value
	Value Value
}

func (v *DictionaryValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {

	count := v.Count()

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()

		defer func() {
			interpreter.reportDictionaryValueConformsToStaticTypeTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	staticType, ok := v.StaticType(interpreter).(DictionaryStaticType)
	if !ok {
		return false
	}

	keyType := staticType.KeyType
	valueType := staticType.ValueType

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

		// Check the key

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryKey := MustConvertStoredValue(interpreter, key)

		if !interpreter.IsSubType(entryKey.StaticType(interpreter), keyType) {
			return false
		}

		if !entryKey.ConformsToStaticType(
			interpreter,
			locationRange,
			results,
		) {
			return false
		}

		// Check the value

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryValue := MustConvertStoredValue(interpreter, value)

		if !interpreter.IsSubType(entryValue.StaticType(interpreter), valueType) {
			return false
		}

		if !entryValue.ConformsToStaticType(
			interpreter,
			locationRange,
			results,
		) {
			return false
		}
	}
}

func (v *DictionaryValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {

	otherDictionary, ok := other.(*DictionaryValue)
	if !ok {
		return false
	}

	if v.Count() != otherDictionary.Count() {
		return false
	}

	if !v.Type.Equal(otherDictionary.Type) {
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

		// Do NOT use an iterator, as other value may be stored in another account,
		// leading to a different iteration order, as the storage ID is used in the seed
		otherValue, otherValueExists :=
			otherDictionary.Get(
				interpreter,
				locationRange,
				MustConvertStoredValue(interpreter, key),
			)

		if !otherValueExists {
			return false
		}

		equatableValue, ok := MustConvertStoredValue(interpreter, value).(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, locationRange, otherValue) {
			return false
		}
	}
}

func (v *DictionaryValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return v.dictionary.Storable(storage, address, maxInlineSize)
}

func (v *DictionaryValue) IsReferenceTrackedResourceKindedValue() {}

func (v *DictionaryValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.StorageID]struct{},
) Value {
	baseUse, elementOverhead, dataUse, metaDataUse := common.NewDictionaryMemoryUsages(
		v.dictionary.Count(),
		v.elementSize,
	)
	common.UseMemory(interpreter, baseUse)
	common.UseMemory(interpreter, elementOverhead)
	common.UseMemory(interpreter, dataUse)
	common.UseMemory(interpreter, metaDataUse)

	interpreter.ReportComputation(common.ComputationKindTransferDictionaryValue, uint(v.Count()))

	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportDictionaryValueTransferTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	currentStorageID := v.StorageID()

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

	if needsStoreTo || !isResourceKinded {

		valueComparator := newValueComparator(interpreter, locationRange)
		hashInputProvider := newHashInputProvider(interpreter, locationRange)

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(v.dictionary.Count(), v.elementSize)
		common.UseMemory(config.MemoryGauge, elementMemoryUse)

		dictionary, err = atree.NewMapFromBatchData(
			config.Storage,
			address,
			atree.NewDefaultDigesterBuilder(),
			v.dictionary.Type(),
			valueComparator,
			hashInputProvider,
			v.dictionary.Seed(),
			func() (atree.Value, atree.Value, error) {

				atreeKey, atreeValue, err := iterator.Next()
				if err != nil {
					return nil, nil, err
				}
				if atreeKey == nil || atreeValue == nil {
					return nil, nil, nil
				}

				key := MustConvertStoredValue(interpreter, atreeKey).
					Transfer(interpreter, locationRange, address, remove, nil, preventTransfer)

				value := MustConvertStoredValue(interpreter, atreeValue).
					Transfer(interpreter, locationRange, address, remove, nil, preventTransfer)

				return key, value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {
				interpreter.RemoveReferencedSlab(keyStorable)
				interpreter.RemoveReferencedSlab(valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}
			interpreter.maybeValidateAtreeValue(v.dictionary)

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	var res *DictionaryValue

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource array as invalidated, by unsetting the backing array.
		// This allows raising an error when the resource array is attempted
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
				dictionaryValue, ok := value.(*DictionaryValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				dictionaryValue.dictionary = dictionary
			},
		)
	}

	if res == nil {
		res = newDictionaryValueFromOrderedMap(dictionary, v.Type)
		res.elementSize = v.elementSize
		res.semaType = v.semaType
		res.isResourceKinded = v.isResourceKinded
		res.isDestroyed = v.isDestroyed
	}

	return res
}

func (v *DictionaryValue) Clone(interpreter *Interpreter) Value {
	config := interpreter.SharedState.Config

	valueComparator := newValueComparator(interpreter, EmptyLocationRange)
	hashInputProvider := newHashInputProvider(interpreter, EmptyLocationRange)

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(v.dictionary.Count(), v.elementSize)
	common.UseMemory(config.MemoryGauge, elementMemoryUse)

	dictionary, err := atree.NewMapFromBatchData(
		config.Storage,
		v.StorageAddress(),
		atree.NewDefaultDigesterBuilder(),
		v.dictionary.Type(),
		valueComparator,
		hashInputProvider,
		v.dictionary.Seed(),
		func() (atree.Value, atree.Value, error) {

			atreeKey, atreeValue, err := iterator.Next()
			if err != nil {
				return nil, nil, err
			}
			if atreeKey == nil || atreeValue == nil {
				return nil, nil, nil
			}

			key := MustConvertStoredValue(interpreter, atreeKey).
				Clone(interpreter)

			value := MustConvertStoredValue(interpreter, atreeValue).
				Clone(interpreter)

			return key, value, nil
		},
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &DictionaryValue{
		Type:             v.Type,
		semaType:         v.semaType,
		isResourceKinded: v.isResourceKinded,
		dictionary:       dictionary,
		isDestroyed:      v.isDestroyed,
	}
}

func (v *DictionaryValue) DeepRemove(interpreter *Interpreter) {

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportDictionaryValueDeepRemoveTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.dictionary.Storage

	err := v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {

		key := StoredValue(interpreter, keyStorable, storage)
		key.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(keyStorable)

		value := StoredValue(interpreter, valueStorable, storage)
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)
}

func (v *DictionaryValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

func (v *DictionaryValue) StorageID() atree.StorageID {
	return v.dictionary.StorageID()
}

func (v *DictionaryValue) StorageAddress() atree.Address {
	return v.dictionary.Address()
}

func (v *DictionaryValue) SemaType(interpreter *Interpreter) *sema.DictionaryType {
	if v.semaType == nil {
		// this function will panic already if this conversion fails
		v.semaType, _ = interpreter.MustConvertStaticToSemaType(v.Type).(*sema.DictionaryType)
	}
	return v.semaType
}

func (v *DictionaryValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *DictionaryValue) IsResourceKinded(interpreter *Interpreter) bool {
	if v.isResourceKinded == nil {
		isResourceKinded := v.SemaType(interpreter).IsResourceType()
		v.isResourceKinded = &isResourceKinded
	}
	return *v.isResourceKinded
}
