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

	"github.com/onflow/cadence/bbq"
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
	recoverErrors                 func(onError func(error))
	inStorageIteration            bool
	storageMutatedDuringIteration bool
	containerValueIteration       map[atree.ValueID]int
	destroyedResources            map[atree.ValueID]struct{}

	// semaTypeCache is a cache-alike for temporary storing sema-types by their ID,
	// to avoid repeated conversions from static-types to sema-types.
	// This cache-alike is maintained per execution.
	// TODO: Re-use the conversions from the compiler.
	// TODO: Maybe extend/share this between executions.
	semaTypeCache   map[sema.TypeID]sema.Type
	semaAccessCache map[interpreter.Authorization]sema.Access

	// linkedGlobalsCache is a local cache-alike that is being used to hold already linked imports.
	linkedGlobalsCache map[common.Location]LinkedGlobals

	getLocationRange func() interpreter.LocationRange

	attachmentIterationMap map[*interpreter.CompositeValue]struct{}
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

func (c *Context) newReusing() *Context {
	newContext := NewContext(c.Config)

	newContext.semaTypeCache = c.semaTypeCache
	newContext.linkedGlobalsCache = c.linkedGlobalsCache

	return newContext
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

func (c *Context) inAttachmentIteration(base *interpreter.CompositeValue) bool {
	_, ok := c.attachmentIterationMap[base]
	return ok
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
	oldState := c.inAttachmentIteration(composite)
	if c.attachmentIterationMap == nil {
		c.attachmentIterationMap = map[*interpreter.CompositeValue]struct{}{}
	}
	if state {
		c.attachmentIterationMap[composite] = struct{}{}
	} else {
		delete(c.attachmentIterationMap, composite)
	}
	return oldState
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
	c.ensureProgramInitialized(location)
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

func (c *Context) ValidateContainerMutation(valueID atree.ValueID) {
	_, present := c.containerValueIteration[valueID]
	if !present {
		return
	}
	panic(&interpreter.ContainerMutatedDuringIterationError{})
}

func (c *Context) GetCompositeValueFunctions(v *interpreter.CompositeValue) *interpreter.FunctionOrderedMap {
	//TODO
	return nil
}

func (c *Context) EnforceNotResourceDestruction(valueID atree.ValueID) {
	_, exists := c.destroyedResources[valueID]
	if exists {
		panic(&interpreter.DestroyedResourceError{})
	}
}

func (c *Context) GetMemberAccessContextForLocation(location common.Location) interpreter.MemberAccessibleContext {
	c.ensureProgramInitialized(location)
	return c
}

func (c *Context) WithResourceDestruction(valueID atree.ValueID, f func()) {
	c.EnforceNotResourceDestruction(valueID)

	if c.destroyedResources == nil {
		c.destroyedResources = make(map[atree.ValueID]struct{})
	}
	c.destroyedResources[valueID] = struct{}{}

	f()
}

func (c *Context) RecoverErrors(onError func(error)) {
	c.recoverErrors(onError)
}

func (c *Context) GetValueOfVariable(name string) interpreter.Value {
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Context) GetLocation() common.Location {
	//TODO
	return nil
}

// InvokeFunction function invokes a given function value with the given arguments.
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
	c.ensureProgramInitialized(location)
	return c
}

func AttachmentBaseAndSelfValues(
	c *Context,
	v *interpreter.CompositeValue,
	method FunctionValue,
) (base *interpreter.EphemeralReferenceValue, self *interpreter.EphemeralReferenceValue) {
	// CompositeValue.GetMethod, as in the interpreter we need an authorized reference to self
	var unqualifiedName string
	switch functionValue := method.(type) {
	case CompiledFunctionValue:
		unqualifiedName = functionValue.Function.Name
	case *NativeFunctionValue:
		unqualifiedName = functionValue.Name
	}

	fnAccess := interpreter.GetAccessOfMember(c, v, unqualifiedName)
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
	if fnAccess.IsPrimitiveAccess() {
		fnAccess = sema.UnauthorizedAccess
	}
	return interpreter.AttachmentBaseAndSelfValues(c, fnAccess, v)
}

func (c *Context) GetMethod(
	value interpreter.MemberAccessibleValue,
	name string,
) interpreter.FunctionValue {
	staticType := value.StaticType(c)

	var location common.Location

	switch staticType := staticType.(type) {
	case *interpreter.CompositeStaticType:
		location = staticType.Location
	case *interpreter.InterfaceStaticType:
		location = staticType.Location

		// TODO: Anything else?
	}

	qualifiedFuncName := commons.StaticTypeQualifiedName(staticType, name)

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

	var base *interpreter.EphemeralReferenceValue
	// If the value is an attachment, then we must create an authorized reference
	if v, ok := value.(*interpreter.CompositeValue); ok && v.Kind == common.CompositeKindAttachment {
		base, value = AttachmentBaseAndSelfValues(c, v, method)
	}

	return NewBoundFunctionValue(
		c,
		value,
		method,
		base,
	)
}

func (c *Context) GetFunction(
	location common.Location,
	name string,
) FunctionValue {
	return c.lookupFunction(location, name)
}

func (c *Context) DefaultDestroyEvents(resourceValue *interpreter.CompositeValue) []*interpreter.CompositeValue {
	method := c.GetMethod(resourceValue, commons.ResourceDestroyedEventsFunctionName)

	if method == nil {
		return nil
	}

	var arguments []Value

	if resourceValue.Kind == common.CompositeKindAttachment {
		base, _ := interpreter.AttachmentBaseAndSelfValues(
			c,
			sema.UnauthorizedAccess,
			resourceValue,
		)
		arguments = []Value{
			base,
		}
	}

	eventValues := make([]*interpreter.CompositeValue, 0)

	collectFunction := NewNativeFunctionValue(
		"", // anonymous function
		commons.CollectEventsFunctionType,
		func(
			context interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.Value,
			arguments []interpreter.Value,
		) interpreter.Value {
			for _, argument := range arguments {
				event := argument.(*interpreter.CompositeValue)
				eventValues = append(eventValues, event)
			}
			return interpreter.Void
		},
	)

	arguments = append(arguments, collectFunction)

	// The generated function takes no arguments unless its an attachment, and returns nothing.
	c.InvokeFunction(method, arguments)

	return eventValues
}

func (c *Context) SemaTypeFromStaticType(staticType interpreter.StaticType) (semaType sema.Type) {
	_, isPrimitiveType := staticType.(interpreter.PrimitiveStaticType)

	if !isPrimitiveType {
		typeID := staticType.ID()
		cachedSemaType, ok := c.semaTypeCache[typeID]
		if ok {
			return cachedSemaType
		}

		defer func() {
			if c.semaTypeCache == nil {
				c.semaTypeCache = make(map[sema.TypeID]sema.Type)
			}
			c.semaTypeCache[typeID] = semaType
		}()
	}

	// TODO: avoid the sema-type conversion
	return interpreter.MustConvertStaticToSemaType(staticType, c)
}

func (c *Context) SemaAccessFromStaticAuthorization(auth interpreter.Authorization) (sema.Access, error) {
	semaAccess, ok := c.semaAccessCache[auth]
	if ok {
		return semaAccess, nil
	}

	semaAccess, err := interpreter.ConvertStaticAuthorizationToSemaAccess(auth, c)
	if err != nil {
		return nil, err
	}

	if c.semaAccessCache == nil {
		c.semaAccessCache = make(map[interpreter.Authorization]sema.Access)
	}
	c.semaAccessCache[auth] = semaAccess

	return semaAccess, nil
}


func (c *Context) GetContractValue(contractLocation common.AddressLocation) *interpreter.CompositeValue {
	c.ensureProgramInitialized(contractLocation)
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

func (c *Context) ensureProgramInitialized(location common.Location) {
	if location == nil {
		return
	}

	c.linkLocation(location)
}

func (c *Context) linkLocation(location common.Location) LinkedGlobals {
	linkedGlobals, ok := c.linkedGlobalsCache[location]
	if ok {
		return linkedGlobals
	}

	program := c.ImportHandler(location)
	if program == nil {
		return LinkedGlobals{}
	}

	return c.linkGlobals(location, program)
}

func (c *Context) linkGlobals(location common.Location, program *bbq.InstructionProgram) LinkedGlobals {
	if c.linkedGlobalsCache == nil {
		c.linkedGlobalsCache = map[common.Location]LinkedGlobals{}
	}

	return LinkGlobals(
		c.MemoryGauge,
		location,
		program,
		c,
		c.linkedGlobalsCache,
	)
}

func (c *Context) GetCompositeType(
	location common.Location,
	qualifiedIdentifier string,
	typeID common.TypeID,
) (*sema.CompositeType, error) {
	c.ensureProgramInitialized(location)
	return c.Config.GetCompositeType(location, qualifiedIdentifier, typeID)
}

func (c *Context) GetInterfaceType(
	location common.Location,
	qualifiedIdentifier string,
	typeID common.TypeID,
) (*sema.InterfaceType, error) {
	c.ensureProgramInitialized(location)
	return c.Config.GetInterfaceType(location, qualifiedIdentifier, typeID)
}

func (c *Context) GetEntitlementType(
	typeID common.TypeID,
) (*sema.EntitlementType, error) {
	return c.Config.GetEntitlementType(typeID, c.ensureProgramInitialized)
}

func (c *Context) GetEntitlementMapType(
	typeID common.TypeID,
) (*sema.EntitlementMapType, error) {
	return c.Config.GetEntitlementMapType(typeID, c.ensureProgramInitialized)
}

func (c *Context) LocationRange() interpreter.LocationRange {
	return c.getLocationRange()
}
