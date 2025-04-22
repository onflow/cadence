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

package runtime

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

// vmEnvironmentReconfigured is the portion of vmEnvironment
// that gets reconfigured by vmEnvironment.Configure
type vmEnvironmentReconfigured struct {
	Interface
	storage        *Storage
	coverageReport *CoverageReport
}

type vmEnvironment struct {
	vmEnvironmentReconfigured

	checkingEnvironment *checkingEnvironment

	deployedContractConstructorInvocation *stdlib.DeployedContractConstructorInvocation
	config                                Config
	*stdlib.SimpleContractAdditionTracker
}

var _ Environment = &vmEnvironment{}
var _ stdlib.Logger = &vmEnvironment{}
var _ stdlib.RandomGenerator = &vmEnvironment{}
var _ stdlib.BlockAtHeightProvider = &vmEnvironment{}
var _ stdlib.CurrentBlockProvider = &vmEnvironment{}
var _ stdlib.AccountHandler = &vmEnvironment{}
var _ stdlib.AccountCreator = &vmEnvironment{}
var _ stdlib.EventEmitter = &vmEnvironment{}
var _ stdlib.PublicKeyValidator = &vmEnvironment{}
var _ stdlib.PublicKeySignatureVerifier = &vmEnvironment{}
var _ stdlib.BLSPoPVerifier = &vmEnvironment{}
var _ stdlib.BLSPublicKeyAggregator = &vmEnvironment{}
var _ stdlib.BLSSignatureAggregator = &vmEnvironment{}
var _ stdlib.Hasher = &vmEnvironment{}
var _ ArgumentDecoder = &vmEnvironment{}
var _ common.MemoryGauge = &vmEnvironment{}

func newVMEnvironment(config Config) *vmEnvironment {
	env := &vmEnvironment{
		config:                        config,
		checkingEnvironment:           newCheckingEnvironment(),
		SimpleContractAdditionTracker: stdlib.NewSimpleContractAdditionTracker(),
	}
	return env
}

func NewBaseVMEnvironment(config Config) *vmEnvironment {
	env := newVMEnvironment(config)
	for _, valueDeclaration := range stdlib.DefaultStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func NewScriptVMEnvironment(config Config) Environment {
	env := newVMEnvironment(config)
	for _, valueDeclaration := range stdlib.DefaultScriptStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func (e *vmEnvironment) Configure(
	runtimeInterface Interface,
	codesAndPrograms CodesAndPrograms,
	storage *Storage,
	coverageReport *CoverageReport,
) {
	e.Interface = runtimeInterface
	e.storage = storage
	// TODO:
	//e.InterpreterConfig.Storage = storage
	e.coverageReport = coverageReport

	e.checkingEnvironment.configure(
		runtimeInterface,
		codesAndPrograms,
	)

	configureVersionedFeatures(runtimeInterface)
}

func (e *vmEnvironment) DeclareValue(valueDeclaration stdlib.StandardLibraryValue, location common.Location) {
	e.checkingEnvironment.declareValue(valueDeclaration, location)

	// TODO: declare in VM
}

func (e *vmEnvironment) DeclareType(typeDeclaration stdlib.StandardLibraryType, location common.Location) {
	e.checkingEnvironment.declareType(typeDeclaration, location)
}

func (e *vmEnvironment) SetCompositeValueFunctionsHandler(
	typeID common.TypeID,
	handler stdlib.CompositeValueFunctionsHandler,
) {
	// TODO:
	panic(errors.NewUnreachableError())
}

func (e *vmEnvironment) CommitStorageTemporarily(context interpreter.ValueTransferContext) error {
	const commitContractUpdates = false
	return e.storage.Commit(context, commitContractUpdates)
}

func (e *vmEnvironment) EmitEvent(
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	values []interpreter.Value,
) {
	EmitEventFields(
		context,
		locationRange,
		eventType,
		values,
		e.Interface.EmitEvent,
	)
}

func (e *vmEnvironment) RecordContractRemoval(location common.AddressLocation) {
	e.storage.recordContractUpdate(location, nil)
}

func (e *vmEnvironment) RecordContractUpdate(
	location common.AddressLocation,
	contractValue *interpreter.CompositeValue,
) {
	e.storage.recordContractUpdate(location, contractValue)
}

func (e *vmEnvironment) ContractUpdateRecorded(location common.AddressLocation) bool {
	return e.storage.contractUpdateRecorded(location)
}

func (e *vmEnvironment) TemporarilyRecordCode(location common.AddressLocation, code []byte) {
	e.checkingEnvironment.temporarilyRecordCode(location, code)
}

func (e *vmEnvironment) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	return e.checkingEnvironment.ParseAndCheckProgram(code, location, getAndSetProgram)
}

func (e *vmEnvironment) ResolveLocation(
	identifiers []Identifier,
	location Location,
) (
	res []ResolvedLocation,
	err error,
) {
	return e.checkingEnvironment.resolveLocation(identifiers, location)
}

func (e *vmEnvironment) LoadContractValue(
	location common.AddressLocation,
	program *interpreter.Program,
	name string,
	invocation stdlib.DeployedContractConstructorInvocation,
) (
	contract *interpreter.CompositeValue,
	err error,
) {
	e.deployedContractConstructorInvocation = &invocation
	defer func() {
		e.deployedContractConstructorInvocation = nil
	}()

	// TODO:
	panic(errors.NewUnreachableError())
}

func (e *vmEnvironment) newAccountValue(
	context interpreter.AccountCreationContext,
	address interpreter.AddressValue,
) interpreter.Value {
	return stdlib.NewAccountValue(context, e, address)
}

func (e *vmEnvironment) commitStorage(context interpreter.ValueTransferContext) error {
	checkStorageHealth := e.config.AtreeValidationEnabled
	return CommitStorage(context, e.storage, checkStorageHealth)
}

func (e *vmEnvironment) ProgramLog(message string, _ interpreter.LocationRange) error {
	return e.Interface.ProgramLog(message)
}
