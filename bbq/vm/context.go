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

	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// Context holds the information about the current execution at any given point of time.
// It consists of:
//   - Re-usable configurations (Config).
//   - State maintained for the current execution.
//
// Should not be re-used across executions.
type Context struct {
	*Config

	CapabilityControllerIterations              map[interpreter.AddressPath]int
	mutationDuringCapabilityControllerIteration bool
	referencedResourceKindedValues              ReferencedResourceKindedValues

	invokeFunction                func(function Value, arguments []Value) (Value, error)
	lookupFunction                func(location common.Location, name string) FunctionValue
	inStorageIteration            bool
	storageMutatedDuringIteration bool
	containerValueIteration       map[atree.ValueID]int
	destroyedResources            map[atree.ValueID]struct{}

	// semaTypes is a cache-alike for temporary storing sema-types by their ID,
	// to avoid repeated conversions from static-types to sema-types.
	// This cache-alike is maintained per execution.
	// TODO: Re-use the conversions from the compiler.
	// TODO: Maybe extend/share this between executions.
	semaTypes map[sema.TypeID]sema.Type

	// TODO: stack-trace, location, etc.
}

var _ interpreter.ReferenceTracker = &Context{}
var _ interpreter.ValueStaticTypeContext = &Context{}
var _ interpreter.ValueTransferContext = &Context{}
var _ interpreter.StorageContext = &Context{}
var _ interpreter.StaticTypeConversionHandler = &Context{}
var _ interpreter.ValueComparisonContext = &Context{}
var _ interpreter.InvocationContext = &Context{}

func NewContext(config *Config) *Context {
	return &Context{
		Config: config,
	}
}

func (c *Context) RecordStorageMutation() {
	if c.inStorageIteration {
		c.storageMutatedDuringIteration = true
	}
}

func (c *Context) StorageMutatedDuringIteration() bool {
	return c.storageMutatedDuringIteration
}

func (c *Context) InStorageIteration() bool {
	return c.inStorageIteration
}

func (c *Context) SetInStorageIteration(inStorageIteration bool) {
	c.inStorageIteration = inStorageIteration
}

func (c *Context) GetCapabilityControllerIterations() map[interpreter.AddressPath]int {
	if c.CapabilityControllerIterations == nil {
		c.CapabilityControllerIterations = make(map[interpreter.AddressPath]int)
	}
	return c.CapabilityControllerIterations
}

func (c *Context) SetMutationDuringCapabilityControllerIteration() {
	c.mutationDuringCapabilityControllerIteration = true
}

func (c *Context) MutationDuringCapabilityControllerIteration() bool {
	return c.mutationDuringCapabilityControllerIteration
}

func (c *Context) SetAttachmentIteration(composite *interpreter.CompositeValue, state bool) bool {
	//TODO
	return false
}

func (c *Context) ReadStored(
	storageAddress common.Address,
	domain common.StorageDomain,
	identifier interpreter.StorageMapKey,
) interpreter.Value {
	accountStorage := c.storage.GetDomainStorageMap(
		c,
		storageAddress,
		domain,
		false,
	)
	if accountStorage == nil {
		return nil
	}

	return accountStorage.ReadValue(c, identifier)
}

func (c *Context) WriteStored(
	storageAddress common.Address,
	domain common.StorageDomain,
	key interpreter.StorageMapKey,
	value interpreter.Value,
) (existed bool) {
	accountStorage := c.storage.GetDomainStorageMap(c, storageAddress, domain, true)

	return accountStorage.WriteValue(
		c,
		key,
		value,
	)
}

func (c *Context) IsSubType(subType interpreter.StaticType, superType interpreter.StaticType) bool {
	return interpreter.IsSubType(c, subType, superType)
}

func (c *Context) MaybeValidateAtreeValue(v atree.Value) {
	//TODO
	// NO-OP: no validation happens for now
}

func (c *Context) MaybeValidateAtreeStorage() {
	//TODO
	// NO-OP: no validation happens for now
}

func (c *Context) IsTypeInfoRecovered(location common.Location) bool {
	elaboration, err := c.ElaborationResolver(location)
	if err != nil {
		return false
	}

	return elaboration.IsRecovered
}

func (c *Context) WithContainerMutationPrevention(valueID atree.ValueID, f func()) {
	if c == nil {
		f()
		return
	}

	c.startContainerValueIteration(valueID)
	f()
	c.endContainerValueIteration(valueID)
}

func (c *Context) endContainerValueIteration(valueID atree.ValueID) {
	c.containerValueIteration[valueID]--
	if c.containerValueIteration[valueID] <= 0 {
		delete(c.containerValueIteration, valueID)
	}
}

func (c *Context) startContainerValueIteration(valueID atree.ValueID) {
	if c.containerValueIteration == nil {
		c.containerValueIteration = make(map[atree.ValueID]int)
	}
	c.containerValueIteration[valueID]++
}

func (c *Context) ValidateContainerMutation(valueID atree.ValueID, locationRange interpreter.LocationRange) {
	_, present := c.containerValueIteration[valueID]
	if !present {
		return
	}
	panic(&interpreter.ContainerMutatedDuringIterationError{
		LocationRange: locationRange,
	})
}

func (c *Context) GetCompositeValueFunctions(v *interpreter.CompositeValue, locationRange interpreter.LocationRange) *interpreter.FunctionOrderedMap {
	//TODO
	return nil
}

func (c *Context) EnforceNotResourceDestruction(valueID atree.ValueID, locationRange interpreter.LocationRange) {
	_, exists := c.destroyedResources[valueID]
	if exists {
		panic(&interpreter.DestroyedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (c *Context) GetMemberAccessContextForLocation(_ common.Location) interpreter.MemberAccessibleContext {
	//TODO
	return c
}

func (c *Context) WithResourceDestruction(valueID atree.ValueID, locationRange interpreter.LocationRange, f func()) {
	c.EnforceNotResourceDestruction(valueID, locationRange)

	if c.destroyedResources == nil {
		c.destroyedResources = make(map[atree.ValueID]struct{})
	}
	c.destroyedResources[valueID] = struct{}{}

	f()
}

func (c *Context) RecoverErrors(onError func(error)) {
	//TODO
}

func (c *Context) GetValueOfVariable(name string) interpreter.Value {
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Context) GetLocation() common.Location {
	//TODO
	return nil
}

func (c *Context) CallStack() []interpreter.Invocation {
	//TODO
	return nil
}

// InvokeFunction function invokes a given function value with the given arguments.
// For bound functions, it expects the first argument to be the receiver.
// i.e: The caller is responsible for preparing the arguments.
func (c *Context) InvokeFunction(
	fn interpreter.FunctionValue,
	arguments []interpreter.Value,
) interpreter.Value {
	result, err := c.invokeFunction(fn, arguments)
	if err != nil {
		panic(err)
	}

	return result
}

func (c *Context) GetResourceDestructionContextForLocation(location common.Location) interpreter.ResourceDestructionContext {
	return c
}

func (c *Context) GetMethod(
	value interpreter.MemberAccessibleValue,
	name string,
	_ interpreter.LocationRange,
) interpreter.FunctionValue {
	staticType := value.StaticType(c)

	semaType := c.SemaTypeFromStaticType(staticType)

	var location common.Location
	if locatedType, ok := semaType.(sema.LocatedType); ok {
		location = locatedType.GetLocation()
	}

	qualifiedFuncName := commons.TypeQualifiedName(semaType, name)

	method := c.GetFunction(location, qualifiedFuncName)
	if method == nil {
		return nil
	}

	// If the value is a "type function" (e.g., `String`, `Int`, etc.),
	// then return the method directly, as the method is essentially "static"
	// and does not expect a receiver.
	if functionValue, ok := value.(FunctionValue); ok &&
		functionValue.FunctionType(c).TypeFunctionType != nil {

		return method
	}

	return NewBoundFunctionValue(
		c,
		value,
		method,
	)
}

func (c *Context) GetFunction(
	location common.Location,
	name string,
) FunctionValue {
	return c.lookupFunction(location, name)
}

func (c *Context) DefaultDestroyEvents(
	resourceValue *interpreter.CompositeValue,
	locationRange interpreter.LocationRange,
) []*interpreter.CompositeValue {
	method := c.GetMethod(
		resourceValue,
		commons.ResourceDestroyedEventsFunctionName,
		EmptyLocationRange,
	)

	if method == nil {
		return nil
	}

	//Always have the receiver as the first argument.
	//arguments := []Value{resourceValue}

	events := c.InvokeFunction(method, nil)
	eventsArray, ok := events.(*interpreter.ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	length := eventsArray.Count()
	common.UseMemory(c, common.NewGoSliceMemoryUsages(length))

	eventValues := make([]*interpreter.CompositeValue, 0, eventsArray.Count())

	eventsArray.Iterate(
		c,
		func(element interpreter.Value) (resume bool) {
			event := element.(*interpreter.CompositeValue)
			eventValues = append(eventValues, event)
			return true
		},
		false,
		locationRange,
	)

	return eventValues
}

func (c *Context) SemaTypeFromStaticType(staticType interpreter.StaticType) sema.Type {
	typeID := staticType.ID()
	semaType, ok := c.semaTypes[typeID]
	if ok {
		return semaType
	}

	// TODO: avoid the sema-type conversion
	semaType = interpreter.MustConvertStaticToSemaType(staticType, c)

	if c.semaTypes == nil {
		c.semaTypes = make(map[sema.TypeID]sema.Type)
	}
	c.semaTypes[typeID] = semaType

	return semaType
}

func (c *Context) GetContractValue(contractLocation common.AddressLocation) *interpreter.CompositeValue {
	return c.ContractValueHandler(c, contractLocation)
}

func (c *Context) MaybeUpdateStorageReferenceMemberReceiver(
	storageReference *interpreter.StorageReferenceValue,
	referencedValue Value,
	member Value,
) Value {
	if boundFunction, isBoundFunction := member.(*BoundFunctionValue); isBoundFunction {
		boundFunction.ReceiverReference = interpreter.StorageReference(
			c,
			storageReference,
			referencedValue,
		)
		return boundFunction
	}

	return member
}
