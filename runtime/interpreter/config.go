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

package interpreter

import (
	"github.com/onflow/cadence/runtime/common"
)

type Config struct {
	MemoryGauge common.MemoryGauge
	Storage     Storage
	// ImportLocationHandler is used to handle imports of locations
	ImportLocationHandler ImportLocationHandlerFunc
	// OnInvokedFunctionReturn is triggered when an invoked function returned
	OnInvokedFunctionReturn OnInvokedFunctionReturnFunc
	// OnRecordTrace is triggered when a trace is recorded
	OnRecordTrace OnRecordTraceFunc
	// OnResourceOwnerChange is triggered when the owner of a resource changes
	OnResourceOwnerChange OnResourceOwnerChangeFunc
	// OnMeterComputation is triggered when a computation is about to happen
	OnMeterComputation OnMeterComputationFunc
	// InjectedCompositeFieldsHandler is used to initialize new composite values' fields
	InjectedCompositeFieldsHandler InjectedCompositeFieldsHandlerFunc
	// ContractValueHandler is used to handle imports of values
	ContractValueHandler ContractValueHandlerFunc
	// OnEventEmitted is triggered when an event is emitted by the program
	OnEventEmitted OnEventEmittedFunc
	// OnFunctionInvocation is triggered when a function invocation is about to be executed
	OnFunctionInvocation OnFunctionInvocationFunc
	// AccountHandler is used to handle accounts
	AccountHandler AccountHandlerFunc
	// UUIDHandler is used to handle the generation of UUIDs
	UUIDHandler UUIDHandlerFunc
	// CompositeTypeHandler is used to load composite types
	CompositeTypeHandler CompositeTypeHandlerFunc
	// InterfaceTypeHandler is used to load interface types
	InterfaceTypeHandler InterfaceTypeHandlerFunc
	// CompositeValueFunctionsHandler is used to load composite value functions
	CompositeValueFunctionsHandler CompositeValueFunctionsHandlerFunc
	BaseActivationHandler          func(location common.Location) *VariableActivation
	Debugger                       *Debugger
	// OnStatement is triggered when a statement is about to be executed
	OnStatement OnStatementFunc
	// OnLoopIteration is triggered when a loop iteration is about to be executed
	OnLoopIteration OnLoopIterationFunc
	// TracingEnabled determines if tracing is enabled.
	// Tracing reports certain operations, e.g. composite value transfers
	TracingEnabled bool
	// AtreeStorageValidationEnabled determines if the validation of atree storage is enabled
	AtreeStorageValidationEnabled bool
	// AtreeValueValidationEnabled determines if the validation of atree values is enabled
	AtreeValueValidationEnabled bool
	// CapabilityCheckHandler is used to check ID capabilities
	CapabilityCheckHandler CapabilityCheckHandlerFunc
	// CapabilityBorrowHandler is used to borrow ID capabilities
	CapabilityBorrowHandler CapabilityBorrowHandlerFunc
	// LegacyContractUpgradeEnabled specifies whether to fall back to the old parser when attempting a contract upgrade
	LegacyContractUpgradeEnabled bool
}
