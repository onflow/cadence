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

package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
)

type CompositeValue struct {
	//dictionary    *atree.OrderedMap
	dictionary    map[string]Value
	CompositeType *interpreter.CompositeStaticType
	Kind          common.CompositeKind
	address       common.Address
}

var _ Value = &CompositeValue{}
var _ MemberAccessibleValue = &CompositeValue{}

//var _ ReferenceTrackedResourceKindedValue = &CompositeValue{}

func NewCompositeValue(
	kind common.CompositeKind,
	staticType *interpreter.CompositeStaticType,
) *CompositeValue {

	return &CompositeValue{
		CompositeType: staticType,
		dictionary:    make(map[string]Value),
		Kind:          kind,
		address:       common.ZeroAddress,
	}
}

func newCompositeValueFromOrderedMap(
	dict map[string]Value,
	staticType *interpreter.CompositeStaticType,
	kind common.CompositeKind,
) *CompositeValue {
	return &CompositeValue{
		dictionary:    dict,
		CompositeType: staticType,
		Kind:          kind,
	}
}

func (*CompositeValue) isValue() {}

func (v *CompositeValue) StaticType(*Config) StaticType {
	return v.CompositeType
}

func (v *CompositeValue) GetMember(config *Config, name string) Value {
	return v.GetField(config, name)
}

func (v *CompositeValue) GetField(config *Config, name string) Value {
	storedValue, ok := v.dictionary[name]
	if !ok {
		return nil
	}

	return storedValue
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

	v.dictionary[name] = value
}

//func (v *CompositeValue) SlabID() atree.SlabID {
//	return v.dictionary.SlabID()
//}

func (v *CompositeValue) TypeID() common.TypeID {
	return v.CompositeType.TypeID
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
		newDictionary := make(map[string]Value, len(dictionary))

		for key, value := range dictionary {
			newDictionary[key] = value.Transfer(config, address, remove, storable)
		}

		dictionary = newDictionary
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

		//invalidateReferencedResources(config, v)

		v.dictionary = nil
	}

	if res == nil {
		res = newCompositeValueFromOrderedMap(
			dictionary,
			v.CompositeType,
			v.Kind,
		)

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

	res.address = common.Address(address)

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
	return atree.Address(v.address)
}

func (v *CompositeValue) IsReferenceTrackedResourceKindedValue() {}

//func (v *CompositeValue) ValueID() atree.ValueID {
//	v.dictionary.
//	return v.dictionary.ValueID()
//}

func (v *CompositeValue) IsStaleResource() bool {
	return v.dictionary == nil && v.IsResourceKinded()
}

// ForEachField iterates over all field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
func (v *CompositeValue) ForEachField(
	config *Config,
	f func(fieldName string, fieldValue Value) (resume bool),
) {

	for key, value := range v.dictionary {
		if !f(key, value) {
			break
		}
	}
}

// ForEachReadOnlyLoadedField iterates over all LOADED field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
// DO NOT perform storage mutations in the callback!
func (v *CompositeValue) ForEachReadOnlyLoadedField(
	config *Config,
	f func(fieldName string, fieldValue Value) (resume bool),
) {
	for key, value := range v.dictionary {
		if !f(key, value) {
			break
		}
	}
}

//func (v *CompositeValue) forEachField(
//	config *Config,
//	atreeIterate func(fn atree.MapEntryIterationFunc) error,
//	f func(fieldName string, fieldValue Value) (resume bool),
//) {
//	err := atreeIterate(func(key atree.Value, atreeValue atree.Value) (resume bool, err error) {
//		value := MustConvertStoredValue(
//			config.MemoryGauge,
//			config.Storage,
//			atreeValue,
//		)
//
//		checkInvalidatedResourceOrResourceReference(value)
//
//		resume = f(
//			string(key.(interpreter.StringAtreeValue)),
//			value,
//		)
//		return
//	})
//
//	if err != nil {
//		panic(errors.NewExternalError(err))
//	}
//}
