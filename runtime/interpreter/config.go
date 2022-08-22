/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	// OnEventEmitted is triggered when an event is emitted by the program.
	OnEventEmitted OnEventEmittedFunc
	// OnStatement is triggered when a statement is about to be executed.
	OnStatement OnStatementFunc
	// OnLoopIteration is triggered when a loop iteration is about to be executed.
	OnLoopIteration OnLoopIterationFunc
	// OnFunctionInvocation is triggered when a function invocation is about to be executed.
	OnFunctionInvocation OnFunctionInvocationFunc
	// OnInvokedFunctionReturn is triggered when an invoked function returned.
	OnInvokedFunctionReturn OnInvokedFunctionReturnFunc
	// OnRecordTrace is triggered when a trace is recorded.
	OnRecordTrace OnRecordTraceFunc
	// OnResourceOwnerChange is triggered when the owner of a resource changes.
	OnResourceOwnerChange OnResourceOwnerChangeFunc
	// OnMeterComputation sets the function that is triggered when a computation is about to happen.
	OnMeterComputation OnMeterComputationFunc
	// InjectedCompositeFieldsHandler is used to initialize new composite values' fields
	InjectedCompositeFieldsHandler InjectedCompositeFieldsHandlerFunc
	// ContractValueHandler is used to handle imports of values.
	ContractValueHandler ContractValueHandlerFunc
	// ImportLocationHandler is used to handle imports of locations.
	ImportLocationHandler ImportLocationHandlerFunc
	// PublicAccountHandler is used to handle accounts.
	PublicAccountHandler PublicAccountHandlerFunc
	// UUIDHandler is used to handle the generation of UUIDs.
	UUIDHandler UUIDHandlerFunc
	// PublicKeyValidationHandler is used to handle public key validation.
	PublicKeyValidationHandler PublicKeyValidationHandlerFunc
	// SignatureVerificationHandler is used to handle signature validation.
	SignatureVerificationHandler SignatureVerificationHandlerFunc
	// BLSVerifyPoPHandler is used to verify BLS PoPs.
	BLSVerifyPoPHandler BLSVerifyPoPHandlerFunc
	// BLSAggregateSignaturesHandler is used to aggregate BLS signatures.
	BLSAggregateSignaturesHandler BLSAggregateSignaturesHandlerFunc
	// BLSAggregatePublicKeysHandler is used to aggregate BLS public keys.
	BLSAggregatePublicKeysHandler BLSAggregatePublicKeysHandlerFunc
	// HashHandler is used to hash.
	HashHandler HashHandlerFunc
	// AtreeValueValidationEnabled sets the atree value validation option.
	AtreeValueValidationEnabled bool
	// AtreeStorageValidationEnabled sets the atree storage validation option.
	AtreeStorageValidationEnabled        bool
	TracingEnabled                       bool
	InvalidatedResourceValidationEnabled bool
	BaseActivation                       *VariableActivation
	Debugger                             *Debugger
	MemoryGauge                          common.MemoryGauge
	Storage                              Storage
}
