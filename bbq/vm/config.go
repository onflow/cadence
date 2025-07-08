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
	"math"

	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// Config contains the VM configurations that is safe to be re-used across VMs/executions.
// It does not hold data specific to a single execution. i.e: No state is maintained.
type Config struct {
	Tracer
	storage              interpreter.Storage
	ImportHandler        commons.ImportHandler
	ContractValueHandler ContractValueHandler
	// BuiltinGlobalsProvider provides the built-in globals for a given location.
	// NOTE: all global must be defined for location nil!
	BuiltinGlobalsProvider BuiltinGlobalsProvider
	TypeLoader             func(location common.Location, typeID interpreter.TypeID) sema.Type

	MemoryGauge      common.MemoryGauge
	ComputationGauge common.ComputationGauge
	// CapabilityCheckHandler is used to check ID capabilities
	CapabilityCheckHandler interpreter.CapabilityCheckHandlerFunc
	// CapabilityBorrowHandler is used to borrow ID capabilities
	CapabilityBorrowHandler interpreter.CapabilityBorrowHandlerFunc
	// ValidateAccountCapabilitiesGetHandler is used to handle when a capability of an account is got.
	ValidateAccountCapabilitiesGetHandler interpreter.ValidateAccountCapabilitiesGetHandlerFunc
	// ValidateAccountCapabilitiesPublishHandler is used to handle when a capability of an account is got.
	ValidateAccountCapabilitiesPublishHandler interpreter.ValidateAccountCapabilitiesPublishHandlerFunc
	// OnEventEmitted is triggered when an event is emitted by the program
	OnEventEmitted interpreter.OnEventEmittedFunc
	// AccountHandlerFunc is used to handle accounts
	AccountHandlerFunc interpreter.AccountHandlerFunc
	// InjectedCompositeFieldsHandler is used to initialize new composite values' fields
	InjectedCompositeFieldsHandler interpreter.InjectedCompositeFieldsHandlerFunc
	// UUIDHandler is used to handle the generation of UUIDs
	UUIDHandler interpreter.UUIDHandlerFunc
	// AtreeStorageValidationEnabled determines if the validation of atree storage is enabled
	AtreeStorageValidationEnabled bool
	// AtreeValueValidationEnabled determines if the validation of atree values is enabled
	AtreeValueValidationEnabled bool
	// StackDepthLimit is the maximum depth of the call stack
	StackDepthLimit uint64

	debugEnabled bool
}

func NewConfig(storage interpreter.Storage) *Config {
	return &Config{
		storage:         storage,
		StackDepthLimit: math.MaxInt,
	}
}

func (c *Config) WithDebugEnabled() *Config {
	c.debugEnabled = true
	return c
}

func (c *Config) MeterMemory(usage common.MemoryUsage) error {
	gauge := c.MemoryGauge
	if gauge == nil {
		return nil
	}

	return gauge.MeterMemory(usage)
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

	ty := c.TypeLoader(location, typeID)
	interfaceType, ok := ty.(*sema.InterfaceType)
	if !ok {
		return nil, interpreter.TypeLoadingError{
			TypeID: typeID,
		}
	}

	return interfaceType, nil
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

	ty := c.TypeLoader(location, typeID)
	compositeType, ok := ty.(*sema.CompositeType)
	if !ok {
		return nil, interpreter.TypeLoadingError{
			TypeID: typeID,
		}
	}

	return compositeType, nil
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

	ty := c.TypeLoader(location, typeID)
	entitlementType, ok := ty.(*sema.EntitlementType)
	if !ok {
		return nil, interpreter.TypeLoadingError{
			TypeID: typeID,
		}
	}

	return entitlementType, nil
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

	ty := c.TypeLoader(location, typeID)
	entitlementMapType, ok := ty.(*sema.EntitlementMapType)
	if !ok {
		return nil, interpreter.TypeLoadingError{
			TypeID: typeID,
		}
	}

	return entitlementMapType, nil
}

func (c *Config) MeterComputation(usage common.ComputationUsage) error {
	if c.ComputationGauge == nil {
		return nil
	}

	return c.ComputationGauge.MeterComputation(usage)
}

func (c *Config) GetInjectedCompositeFieldsHandler() interpreter.InjectedCompositeFieldsHandlerFunc {
	return c.InjectedCompositeFieldsHandler
}

func (c *Config) GetAccountHandlerFunc() interpreter.AccountHandlerFunc {
	return c.AccountHandlerFunc
}

func (c *Config) GetValidateAccountCapabilitiesGetHandler() interpreter.ValidateAccountCapabilitiesGetHandlerFunc {
	return c.ValidateAccountCapabilitiesGetHandler
}

func (c *Config) GetValidateAccountCapabilitiesPublishHandler() interpreter.ValidateAccountCapabilitiesPublishHandlerFunc {
	return c.ValidateAccountCapabilitiesPublishHandler
}

func (c *Config) GetCapabilityBorrowHandler() interpreter.CapabilityBorrowHandlerFunc {
	return c.CapabilityBorrowHandler
}

func (c *Config) GetCapabilityCheckHandler() interpreter.CapabilityCheckHandlerFunc {
	return c.CapabilityCheckHandler
}

func (c *Config) EmitEvent(
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	eventFields []Value,
) {
	onEventEmitted := c.OnEventEmitted
	if onEventEmitted == nil {
		panic(&interpreter.EventEmissionUnavailableError{
			LocationRange: locationRange,
		})
	}

	err := onEventEmitted(
		context,
		locationRange,
		eventType,
		eventFields,
	)
	if err != nil {
		panic(err)
	}
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

type ContractValueHandler func(
	context *Context,
	location common.Location,
) *interpreter.CompositeValue
