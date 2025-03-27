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
	"github.com/onflow/cadence/test_utils/common_utils"
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

	// TODO: Move these to a 'shared state'?
	storage                                     interpreter.Storage
	CapabilityControllerIterations              map[interpreter.AddressPath]int
	mutationDuringCapabilityControllerIteration bool
	referencedResourceKindedValues              ReferencedResourceKindedValues

	// OnEventEmitted is triggered when an event is emitted by the program
	OnEventEmitted OnEventEmittedFunc

	// TODO: These are temporary. Remove once storing/reading is supported for VM values.
	inter      *interpreter.Interpreter
	TypeLoader func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType
}

var _ interpreter.ReferenceTracker = &Config{}
var _ interpreter.ValueStaticTypeContext = &Config{}
var _ interpreter.ValueTransferContext = &Config{}
var _ interpreter.StorageContext = &Config{}
var _ interpreter.StaticTypeConversionHandler = &Config{}
var _ interpreter.ValueComparisonContext = &Config{}

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

// TODO: This is temporary. Remove once storing/reading is supported for VM values.
func (c *Config) Interpreter() *interpreter.Interpreter {
	if c.inter == nil {
		inter, err := interpreter.NewInterpreter(
			nil,
			common_utils.TestLocation,
			&interpreter.Config{
				Storage: c.storage,

				// Interpreters are needed only to access interpreter-bound functions.
				// Hence, just return the same interpreter as-is, for now.
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.InterpreterImport{
						Interpreter: inter,
					}
				},
				CompositeTypeHandler: func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
					return c.TypeLoader(location, typeID).(*sema.CompositeType)
				},
				InterfaceTypeHandler: func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
					return c.TypeLoader(location, typeID).(*sema.InterfaceType)
				},
			},
		)

		if err != nil {
			panic(err)
		}

		c.inter = inter
	}

	return c.inter
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
		c.Interpreter(),
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
	inter := c.Interpreter()
	accountStorage := c.storage.GetDomainStorageMap(inter, storageAddress, domain, true)

	return accountStorage.WriteValue(
		inter,
		key,
		value,
	)
}

func (c *Config) IsSubType(subType interpreter.StaticType, superType interpreter.StaticType) bool {
	inter := c.Interpreter()
	return interpreter.IsSubType(inter, subType, superType)
}

func (c *Config) GetInterfaceType(
	location common.Location,
	_ string,
	typeID interpreter.TypeID,
) (*sema.InterfaceType, error) {

	// TODO: Lookup in built-in types

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
	_ string,
	typeID interpreter.TypeID,
) (*sema.CompositeType, error) {

	// TODO: Lookup in built-in types

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
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Config) GetEntitlementMapType(typeID interpreter.TypeID) (*sema.EntitlementMapType, error) {
	//TODO
	panic(errors.NewUnreachableError())
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

func (c *Config) IsRecovered(location common.Location) bool {
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
	//TODO
	return nil
}

func (c *Config) GetMemberAccessContextForLocation(_ common.Location) interpreter.MemberAccessibleContext {
	//TODO
	return c
}

func (c *Config) AccountHandler() interpreter.AccountHandlerFunc {
	// TODO:
	return nil
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

type ContractValueHandler func(conf *Config, location common.Location) *interpreter.CompositeValue
