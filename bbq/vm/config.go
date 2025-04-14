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
	"fmt"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

// OnEventEmittedFunc is a function that is triggered when an event is emitted by the program.
type OnEventEmittedFunc func(
	event *interpreter.CompositeValue,
	eventType *interpreter.CompositeStaticType,
) error

type Config struct {
	common.MemoryGauge
	commons.ImportHandler
	ContractValueHandler
	Tracer

	accountHandler stdlib.AccountHandler

	NativeFunctionsProvider
	interpreterConfig *interpreter.Config

	invokeFunction func(function Value, arguments []Value) (Value, error)

	// TODO: Move these to a 'shared state'?
	storage                                     interpreter.Storage
	CapabilityControllerIterations              map[interpreter.AddressPath]int
	mutationDuringCapabilityControllerIteration bool
	referencedResourceKindedValues              ReferencedResourceKindedValues

	// OnEventEmitted is triggered when an event is emitted by the program
	OnEventEmitted OnEventEmittedFunc

	TypeLoader func(location common.Location, typeID interpreter.TypeID) sema.ContainedType

	debugEnabled bool
}

var _ interpreter.ReferenceTracker = &Config{}
var _ interpreter.ValueStaticTypeContext = &Config{}
var _ interpreter.ValueTransferContext = &Config{}
var _ interpreter.StorageContext = &Config{}
var _ interpreter.StaticTypeConversionHandler = &Config{}
var _ interpreter.ValueComparisonContext = &Config{}
var _ interpreter.InvocationContext = &Config{}
var _ stdlib.Logger = &Config{}

func NewConfig(storage interpreter.Storage) *Config {
	return &Config{
		storage:              storage,
		MemoryGauge:          nil,
		ImportHandler:        nil,
		ContractValueHandler: nil,
		accountHandler:       nil,

		CapabilityControllerIterations:              make(map[interpreter.AddressPath]int),
		mutationDuringCapabilityControllerIteration: false,
		referencedResourceKindedValues:              ReferencedResourceKindedValues{},
	}
}

func (c *Config) WithAccountHandler(handler stdlib.AccountHandler) *Config {
	c.accountHandler = handler
	return c
}

func (c *Config) InterpreterConfig() *interpreter.Config {
	return c.interpreterConfig
}

func (c *Config) WithInterpreterConfig(config *interpreter.Config) *Config {
	c.interpreterConfig = config
	return c
}

func (c *Config) WithDebugEnabled() *Config {
	c.debugEnabled = true
	return c
}

func (c *Config) MeterMemory(usage common.MemoryUsage) error {
	if c.MemoryGauge == nil {
		return nil
	}

	return c.MemoryGauge.MeterMemory(usage)
}

func (c *Config) Storage() interpreter.Storage {
	return c.storage
}

func (c *Config) ReadStored(storageAddress common.Address, domain common.StorageDomain, identifier interpreter.StorageMapKey) interpreter.Value {
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

func (c *Config) WriteStored(
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

func (c *Config) IsSubType(subType interpreter.StaticType, superType interpreter.StaticType) bool {
	return interpreter.IsSubType(c, subType, superType)
}

func (c *Config) GetInterfaceType(
	location common.Location,
	qualifiedIdentifier string,
	typeID interpreter.TypeID,
) (*sema.InterfaceType, error) {

	if location == nil {
		interfaceType := sema.NativeInterfaceTypes[qualifiedIdentifier]
		if interfaceType != nil {
			return interfaceType, nil
		}
	}

	compositeKindedType := c.TypeLoader(location, typeID)
	if compositeKindedType != nil {

		inter, ok := compositeKindedType.(*sema.InterfaceType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return inter, nil
	}

	return nil, interpreter.TypeLoadingError{
		TypeID: typeID,
	}
}

func (c *Config) GetCompositeType(
	location common.Location,
	qualifiedIdentifier string,
	typeID interpreter.TypeID,
) (*sema.CompositeType, error) {

	if location == nil {
		compositeType := sema.NativeCompositeTypes[qualifiedIdentifier]
		if compositeType != nil {
			return compositeType, nil
		}
	}

	compositeKindedType := c.TypeLoader(location, typeID)
	if compositeKindedType != nil {

		compositeType, ok := compositeKindedType.(*sema.CompositeType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return compositeType, nil
	}

	return nil, interpreter.TypeLoadingError{
		TypeID: typeID,
	}
}

func (c *Config) GetEntitlementType(typeID interpreter.TypeID) (*sema.EntitlementType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(c, string(typeID))
	if err != nil {
		return nil, err
	}

	if location == nil {
		ty := sema.BuiltinEntitlements[qualifiedIdentifier]
		if ty == nil {
			return nil, interpreter.TypeLoadingError{
				TypeID: typeID,
			}
		}

		return ty, nil
	}

	typ := c.TypeLoader(location, typeID)
	if typ != nil {

		entitlementType, ok := typ.(*sema.EntitlementType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return entitlementType, nil
	}

	return nil, interpreter.TypeLoadingError{
		TypeID: typeID,
	}
}

func (c *Config) GetEntitlementMapType(typeID interpreter.TypeID) (*sema.EntitlementMapType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(c, string(typeID))
	if err != nil {
		return nil, err
	}

	if location == nil {
		ty := sema.BuiltinEntitlementMappings[qualifiedIdentifier]
		if ty == nil {
			return nil, interpreter.TypeLoadingError{
				TypeID: typeID,
			}
		}

		return ty, nil
	}

	typ := c.TypeLoader(location, typeID)
	if typ != nil {

		entitlementMapType, ok := typ.(*sema.EntitlementMapType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return entitlementMapType, nil
	}

	return nil, interpreter.TypeLoadingError{
		TypeID: typeID,
	}
}

func (c *Config) MaybeValidateAtreeValue(v atree.Value) {
	//TODO
	// NO-OP: no validation happens for now
}

func (c *Config) MaybeValidateAtreeStorage() {
	//TODO
	// NO-OP: no validation happens for now
}

func (c *Config) RecordStorageMutation() {
	// TODO
	// NO-OP
}

func (c *Config) IsTypeInfoRecovered(location common.Location) bool {
	//TODO
	return false
}

func (c *Config) ReportComputation(compKind common.ComputationKind, intensity uint) {
	//TODO
}

func (c *Config) OnResourceOwnerChange(resource *interpreter.CompositeValue, oldOwner common.Address, newOwner common.Address) {
	//TODO
}

func (c *Config) WithMutationPrevention(valueID atree.ValueID, f func()) {
	f()
	//TODO
}

func (c *Config) ValidateMutation(valueID atree.ValueID, locationRange interpreter.LocationRange) {
	//TODO
}

func (c *Config) GetCompositeValueFunctions(v *interpreter.CompositeValue, locationRange interpreter.LocationRange) *interpreter.FunctionOrderedMap {
	//TODO
	return nil
}

func (c *Config) EnforceNotResourceDestruction(valueID atree.ValueID, locationRange interpreter.LocationRange) {
	//TODO
}

func (c *Config) InjectedCompositeFieldsHandler() interpreter.InjectedCompositeFieldsHandlerFunc {
	if c.interpreterConfig == nil {
		return nil
	}
	return c.interpreterConfig.InjectedCompositeFieldsHandler
}

func (c *Config) GetMemberAccessContextForLocation(_ common.Location) interpreter.MemberAccessibleContext {
	//TODO
	return c
}

func (c *Config) AccountHandler() interpreter.AccountHandlerFunc {
	return c.interpreterConfig.AccountHandler
}

func (c *Config) GetAccountHandler() stdlib.AccountHandler {
	return c.accountHandler
}

func (c *Config) StorageMutatedDuringIteration() bool {
	//TODO
	return false
}

func (c *Config) InStorageIteration() bool {
	//TODO
	return false
}

func (c *Config) SetInStorageIteration(b bool) {
	//TODO
}

func (c *Config) WithResourceDestruction(valueID atree.ValueID, locationRange interpreter.LocationRange, f func()) {
	f()
	//TODO
}

func (c *Config) GetCapabilityControllerIterations() map[interpreter.AddressPath]int {
	return c.CapabilityControllerIterations
}

func (c *Config) SetMutationDuringCapabilityControllerIteration() {
	c.mutationDuringCapabilityControllerIteration = true
}

func (c *Config) MutationDuringCapabilityControllerIteration() bool {
	return c.mutationDuringCapabilityControllerIteration
}

func (c *Config) GetContractValue(contractLocation common.AddressLocation) (*interpreter.CompositeValue, error) {
	//TODO
	return nil, nil
}

func (c *Config) SetAttachmentIteration(composite *interpreter.CompositeValue, state bool) bool {
	//TODO
	return false
}

func (c *Config) RecoverErrors(onError func(error)) {
	//TODO
}

func (c *Config) CurrentEntitlementMappedValue() interpreter.Authorization {
	//TODO
	return nil
}

func (c *Config) ValidateAccountCapabilitiesGetHandler() interpreter.ValidateAccountCapabilitiesGetHandlerFunc {
	return c.interpreterConfig.ValidateAccountCapabilitiesGetHandler
}

func (c *Config) ValidateAccountCapabilitiesPublishHandler() interpreter.ValidateAccountCapabilitiesPublishHandlerFunc {
	return c.interpreterConfig.ValidateAccountCapabilitiesPublishHandler
}

func (c *Config) CapabilityBorrowHandler() interpreter.CapabilityBorrowHandlerFunc {
	return c.interpreterConfig.CapabilityBorrowHandler
}

func (c *Config) GetCapabilityCheckHandler() interpreter.CapabilityCheckHandlerFunc {
	return c.interpreterConfig.CapabilityCheckHandler
}

func (c *Config) GetValueOfVariable(name string) interpreter.Value {
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Config) GetLocation() common.Location {
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Config) CallStack() []interpreter.Invocation {
	//TODO
	return nil
}

func (c *Config) EmitEventValue(event *interpreter.CompositeValue, eventType *sema.CompositeType, locationRange interpreter.LocationRange) {
	//TODO
}

func (c *Config) GetResourceDestructionContextForLocation(location common.Location) interpreter.ResourceDestructionContext {
	return c
}

func (c *Config) EmitEvent(
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	values []interpreter.Value,
) {
	//TODO
}

func (c *Config) ProgramLog(message string, locationRange interpreter.LocationRange) error {
	//TODO implement properly
	fmt.Println(message)
	return nil
}

func (c *Config) InvokeFunction(
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

type ContractValueHandler func(conf *Config, location common.Location) *interpreter.CompositeValue
