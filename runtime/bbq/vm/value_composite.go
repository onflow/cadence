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

package vm

import (
	goerrors "errors"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type CompositeValue struct {
	dictionary          *atree.OrderedMap
	Location            common.Location
	QualifiedIdentifier string
	typeID              common.TypeID
	staticType          StaticType
	Kind                common.CompositeKind
}

var _ Value = &CompositeValue{}
var _ MemberAccessibleValue = &CompositeValue{}

func NewCompositeValue(
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
	address common.Address,
	storage atree.SlabStorage,
) *CompositeValue {

	dictionary, err := atree.NewMap(
		storage,
		atree.Address(address),
		atree.NewDefaultDigesterBuilder(),
		interpreter.NewCompositeTypeInfo(
			nil,
			location,
			qualifiedIdentifier,
			kind,
		),
	)

	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &CompositeValue{
		QualifiedIdentifier: qualifiedIdentifier,
		Location:            location,
		dictionary:          dictionary,
		Kind:                kind,
	}
}

func newCompositeValueFromOrderedMap(
	dict *atree.OrderedMap,
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
) *CompositeValue {
	return &CompositeValue{
		dictionary:          dict,
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Kind:                kind,
	}
}

func (*CompositeValue) isValue() {}

func (v *CompositeValue) StaticType(memoryGauge common.MemoryGauge) StaticType {
	if v.staticType == nil {
		// NOTE: Instead of using NewCompositeStaticType, which always generates the type ID,
		// use the TypeID accessor, which may return an already computed type ID
		v.staticType = interpreter.NewCompositeStaticType(
			memoryGauge,
			v.Location,
			v.QualifiedIdentifier,
			v.TypeID(),
		)
	}
	return v.staticType
}

func (v *CompositeValue) GetMember(config *Config, name string) Value {
	return v.GetField(config, name)
}

func (v *CompositeValue) GetField(config *Config, name string) Value {
	storedValue, err := v.dictionary.Get(
		interpreter.StringAtreeValueComparator,
		interpreter.StringAtreeValueHashInput,
		interpreter.StringAtreeValue(name),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil
		}
		panic(errors.NewExternalError(err))
	}

	return MustConvertStoredValue(config.MemoryGauge, config.Storage, storedValue)
}

func (v *CompositeValue) SetMember(config *Config, name string, value Value) {

	// TODO:
	//address := v.StorageID().Address
	//value = value.Transfer(
	//	interpreter,
	//	locationRange,
	//	address,
	//	true,
	//	nil,
	//)

	interpreterValue := VMValueToInterpreterValue(config, value)

	existingStorable, err := v.dictionary.Set(
		interpreter.StringAtreeValueComparator,
		interpreter.StringAtreeValueHashInput,
		interpreter.NewStringAtreeValue(config.MemoryGauge, name),
		interpreterValue,
	)

	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if existingStorable != nil {
		inter := config.interpreter()
		existingValue := interpreter.StoredValue(nil, existingStorable, config.Storage)

		existingValue.DeepRemove(inter, true) // existingValue is standalone because it was overwritten in parent container.

		RemoveReferencedSlab(config.Storage, existingStorable)
	}
}

func (v *CompositeValue) SlabID() atree.SlabID {
	return v.dictionary.SlabID()
}

func (v *CompositeValue) TypeID() common.TypeID {
	if v.typeID == "" {
		location := v.Location
		qualifiedIdentifier := v.QualifiedIdentifier
		if location == nil {
			return common.TypeID(qualifiedIdentifier)
		}

		// TODO: TypeID metering
		v.typeID = location.TypeID(nil, qualifiedIdentifier)
	}
	return v.typeID
}

func (v *CompositeValue) IsResourceKinded() bool {
	return v.Kind == common.CompositeKindResource
}

func (v *CompositeValue) String() string {
	//TODO implement me
	panic("implement me")
}

func (v *CompositeValue) Transfer(
	config *Config,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {

	//baseUse, elementOverhead, dataUse, metaDataUse := common.NewCompositeMemoryUsages(v.dictionary.Count(), 0)
	//common.UseMemory(interpreter, baseUse)
	//common.UseMemory(interpreter, elementOverhead)
	//common.UseMemory(interpreter, dataUse)
	//common.UseMemory(interpreter, metaDataUse)
	//
	//interpreter.ReportComputation(common.ComputationKindTransferCompositeValue, 1)

	//if interpreter.Config.InvalidatedResourceValidationEnabled {
	//	v.checkInvalidatedResourceUse(locationRange)
	//}

	//if interpreter.Config.TracingEnabled {
	//	startTime := time.Now()
	//
	//	owner := v.GetOwner().String()
	//	typeID := string(v.TypeID())
	//	kind := v.Kind.String()
	//
	//	defer func() {
	//		interpreter.reportCompositeValueTransferTrace(
	//			owner,
	//			typeID,
	//			kind,
	//			time.Since(startTime),
	//		)
	//	}()
	//}

	dictionary := v.dictionary

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded()

	if needsStoreTo && v.Kind == common.CompositeKindContract {
		panic(interpreter.NonTransferableValueError{
			Value: VMValueToInterpreterValue(config, v),
		})
	}

	if needsStoreTo || !isResourceKinded {
		iterator, err := v.dictionary.Iterator(
			interpreter.StringAtreeValueComparator,
			interpreter.StringAtreeValueHashInput,
		)
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
			interpreter.StringAtreeValueComparator,
			interpreter.StringAtreeValueHashInput,
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

				value := interpreter.MustConvertStoredValue(config.MemoryGauge, atreeValue)

				// TODO:
				vmValue := InterpreterValueToVMValue(config.Storage, value)
				vmValue.Transfer(config, address, remove, nil)

				return atreeKey, value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.dictionary.PopIterate(func(nameStorable atree.Storable, valueStorable atree.Storable) {
				RemoveReferencedSlab(config.Storage, nameStorable)
				RemoveReferencedSlab(config.Storage, valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}
			//interpreter.maybeValidateAtreeValue(v.dictionary)

			RemoveReferencedSlab(config.Storage, storable)
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

		//if interpreter.Config.InvalidatedResourceValidationEnabled {
		//	v.dictionary = nil
		//} else {
		//	v.dictionary = dictionary
		//	res = v
		//}

		//newStorageID := dictionary.StorageID()
		//
		//interpreter.updateReferencedResource(
		//	currentStorageID,
		//	newStorageID,
		//	func(value ReferenceTrackedResourceKindedValue) {
		//		compositeValue, ok := value.(*CompositeValue)
		//		if !ok {
		//			panic(errors.NewUnreachableError())
		//		}
		//		compositeValue.dictionary = dictionary
		//	},
		//)
	}

	if res == nil {
		typeInfo := interpreter.NewCompositeTypeInfo(
			config.MemoryGauge,
			v.Location,
			v.QualifiedIdentifier,
			v.Kind,
		)
		res = &CompositeValue{
			dictionary:          dictionary,
			Location:            typeInfo.Location,
			QualifiedIdentifier: typeInfo.QualifiedIdentifier,
			Kind:                typeInfo.Kind,
			typeID:              v.typeID,
			staticType:          v.staticType,
		}

		//res.InjectedFields = v.InjectedFields
		//res.ComputedFields = v.ComputedFields
		//res.NestedVariables = v.NestedVariables
		//res.Functions = v.Functions
		//res.Destructor = v.Destructor
		//res.Stringer = v.Stringer
		//res.isDestroyed = v.isDestroyed
	}

	//onResourceOwnerChange := interpreter.Config.OnResourceOwnerChange
	//
	//if needsStoreTo &&
	//	res.Kind == common.CompositeKindResource &&
	//	onResourceOwnerChange != nil {
	//
	//	onResourceOwnerChange(
	//		interpreter,
	//		res,
	//		common.Address(currentAddress),
	//		common.Address(address),
	//	)
	//}

	return res
}

func (v *CompositeValue) Destroy(*Config) {

	//interpreter.ReportComputation(common.ComputationKindDestroyCompositeValue, 1)
	//
	//if interpreter.Config.InvalidatedResourceValidationEnabled {
	//	v.checkInvalidatedResourceUse(locationRange)
	//}
	//
	//storageID := v.StorageID()
	//
	//if interpreter.Config.TracingEnabled {
	//	startTime := time.Now()
	//
	//	owner := v.GetOwner().String()
	//	typeID := string(v.TypeID())
	//	kind := v.Kind.String()
	//
	//	defer func() {
	//
	//		interpreter.reportCompositeValueDestroyTrace(
	//			owner,
	//			typeID,
	//			kind,
	//			time.Since(startTime),
	//		)
	//	}()
	//}

	//interpreter = v.getInterpreter(interpreter)

	//// if composite was deserialized, dynamically link in the destructor
	//if v.Destructor == nil {
	//	v.Destructor = interpreter.sharedState.typeCodes.CompositeCodes[v.TypeID()].DestructorFunction
	//}
	//
	//destructor := v.Destructor
	//
	//if destructor != nil {
	//	invocation := NewInvocation(
	//		interpreter,
	//		v,
	//		nil,
	//		nil,
	//		nil,
	//		locationRange,
	//	)
	//
	//	destructor.invoke(invocation)
	//}

	//v.isDestroyed = true

	//if interpreter.Config.InvalidatedResourceValidationEnabled {
	//	v.dictionary = nil
	//}

	//interpreter.updateReferencedResource(
	//	storageID,
	//	storageID,
	//	func(value ReferenceTrackedResourceKindedValue) {
	//		compositeValue, ok := value.(*CompositeValue)
	//		if !ok {
	//			panic(errors.NewUnreachableError())
	//		}
	//
	//		compositeValue.isDestroyed = true
	//
	//		if interpreter.Config.InvalidatedResourceValidationEnabled {
	//			compositeValue.dictionary = nil
	//		}
	//	},
	//)
}

func (v *CompositeValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *CompositeValue) StorageAddress() atree.Address {
	return v.dictionary.Address()
}
