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

// Config contains the VM configurations that is safe to be re-used across VMs/executions.
// It does not hold data specific to a single execution. i.e: No state is maintained.
type Config struct {
	common.MemoryGauge
	commons.ImportHandler
	ContractValueHandler
	NativeFunctionsProvider
	Tracer
	stdlib.Logger

	storage           interpreter.Storage
	interpreterConfig *interpreter.Config

	accountHandler stdlib.AccountHandler
	TypeLoader     func(location common.Location, typeID interpreter.TypeID) sema.ContainedType
	// OnEventEmitted is triggered when an event is emitted by the program
	OnEventEmitted OnEventEmittedFunc

	debugEnabled bool
}

func NewConfig(storage interpreter.Storage) *Config {
	return &Config{
		storage: storage,
	}
}

func (c *Config) WithAccountHandler(handler stdlib.AccountHandler) *Config {
	c.accountHandler = handler
	return c
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

func (c *Config) InterpreterConfig() *interpreter.Config {
	return c.interpreterConfig
}

func (c *Config) Storage() interpreter.Storage {
	return c.storage
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

func (c *Config) ReportComputation(compKind common.ComputationKind, intensity uint) {
	//TODO
}

func (c *Config) InjectedCompositeFieldsHandler() interpreter.InjectedCompositeFieldsHandlerFunc {
	if c.interpreterConfig == nil {
		return nil
	}
	return c.interpreterConfig.InjectedCompositeFieldsHandler
}

func (c *Config) AccountHandler() interpreter.AccountHandlerFunc {
	return c.interpreterConfig.AccountHandler
}

func (c *Config) GetAccountHandler() stdlib.AccountHandler {
	return c.accountHandler
}

func (c *Config) GetContractValue(contractLocation common.AddressLocation) (*interpreter.CompositeValue, error) {
	//TODO
	return nil, nil
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

func (c *Config) EmitEventValue(
	event *interpreter.CompositeValue,
	eventType *sema.CompositeType,
	locationRange interpreter.LocationRange,
) {
	//TODO
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
	if c.Logger == nil {
		return errors.NewDefaultUserError("logging is not supported in this environment")
	}
	return c.Logger.ProgramLog(message, locationRange)
}

func (c *Config) OnResourceOwnerChange(
	resource *interpreter.CompositeValue,
	oldOwner common.Address,
	newOwner common.Address,
) {
	//TODO
}

func (c *Config) SetStorage(storage interpreter.Storage) {
	c.storage = storage
}

type ContractValueHandler func(conf *Config, location common.Location) *interpreter.CompositeValue
