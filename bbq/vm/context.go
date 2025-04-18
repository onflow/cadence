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
	"github.com/onflow/cadence/stdlib"
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

	invokeFunction func(function Value, arguments []Value) (Value, error)

	// TODO: stack-trace, location, etc.
}

var _ interpreter.ReferenceTracker = &Context{}
var _ interpreter.ValueStaticTypeContext = &Context{}
var _ interpreter.ValueTransferContext = &Context{}
var _ interpreter.StorageContext = &Context{}
var _ interpreter.StaticTypeConversionHandler = &Context{}
var _ interpreter.ValueComparisonContext = &Context{}
var _ interpreter.InvocationContext = &Context{}
var _ stdlib.Logger = &Context{}

func NewContext(config *Config) *Context {
	return &Context{
		Config:                         config,
		CapabilityControllerIterations: make(map[interpreter.AddressPath]int),
		mutationDuringCapabilityControllerIteration: false,
		referencedResourceKindedValues:              ReferencedResourceKindedValues{},
	}
}

func (c *Context) StorageMutatedDuringIteration() bool {
	//TODO
	return false
}

func (c *Context) InStorageIteration() bool {
	//TODO
	return false
}

func (c *Context) SetInStorageIteration(b bool) {
	//TODO
}

func (c *Context) GetCapabilityControllerIterations() map[interpreter.AddressPath]int {
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

func (c *Context) ReadStored(storageAddress common.Address, domain common.StorageDomain, identifier interpreter.StorageMapKey) interpreter.Value {
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

func (c *Context) RecordStorageMutation() {
	// TODO
	// NO-OP
}

func (c *Context) IsTypeInfoRecovered(location common.Location) bool {
	//TODO
	return false
}

func (c *Context) WithMutationPrevention(valueID atree.ValueID, f func()) {
	f()
	//TODO
}

func (c *Context) ValidateMutation(valueID atree.ValueID, locationRange interpreter.LocationRange) {
	//TODO
}

func (c *Context) GetCompositeValueFunctions(v *interpreter.CompositeValue, locationRange interpreter.LocationRange) *interpreter.FunctionOrderedMap {
	//TODO
	return nil
}

func (c *Context) EnforceNotResourceDestruction(valueID atree.ValueID, locationRange interpreter.LocationRange) {
	//TODO
}

func (c *Context) GetMemberAccessContextForLocation(_ common.Location) interpreter.MemberAccessibleContext {
	//TODO
	return c
}

func (c *Context) WithResourceDestruction(valueID atree.ValueID, locationRange interpreter.LocationRange, f func()) {
	f()
	//TODO
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
	panic(errors.NewUnreachableError())
}

func (c *Context) CallStack() []interpreter.Invocation {
	//TODO
	return nil
}

func (c *Context) InvokeFunction(
	fn interpreter.FunctionValue,
	arguments []interpreter.Value,
	_ []sema.Type,
	_ interpreter.LocationRange,
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
	locationRange interpreter.LocationRange,
) interpreter.FunctionValue {
	staticType := value.StaticType(c)

	// TODO: avoid the sema-type conversion
	semaType := interpreter.MustConvertStaticToSemaType(staticType, c)

	var location common.Location
	if locatedType, ok := semaType.(sema.LocatedType); ok {
		location = locatedType.GetLocation()
	}

	typeQualifier := commons.TypeQualifier(semaType)
	qualifiedFuncName := commons.TypeQualifiedName(typeQualifier, name)

	return c.lookupFunction(location, qualifiedFuncName)
}
