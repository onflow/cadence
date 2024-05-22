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
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
)

type SimpleCompositeValue struct {
	fields     map[string]Value
	typeID     common.TypeID
	staticType StaticType
	Kind       common.CompositeKind
}

var _ Value = &CompositeValue{}

func NewSimpleCompositeValue(
	kind common.CompositeKind,
	typeID common.TypeID,
	fields map[string]Value,
) *SimpleCompositeValue {

	return &SimpleCompositeValue{
		Kind:   kind,
		typeID: typeID,
		fields: fields,
	}
}

func (*SimpleCompositeValue) isValue() {}

func (v *SimpleCompositeValue) StaticType(memoryGauge common.MemoryGauge) StaticType {
	return v.staticType
}

func (v *SimpleCompositeValue) GetMember(_ *Config, name string) Value {
	return v.fields[name]
}

func (v *SimpleCompositeValue) SetMember(_ *Config, name string, value Value) {
	v.fields[name] = value
}

func (v *SimpleCompositeValue) IsResourceKinded() bool {
	return v.Kind == common.CompositeKindResource
}

func (v *SimpleCompositeValue) String() string {
	//TODO implement me
	panic("implement me")
}

func (v *SimpleCompositeValue) Transfer(
	conf *Config,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	return v
}

func (v *SimpleCompositeValue) Destroy(*Config) {

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
