/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	"encoding/hex"
	goErrors "errors"
	"fmt"
	"math"
	goRuntime "runtime"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"
	"github.com/opentracing/opentracing-go"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

type controlReturn interface {
	isControlReturn()
}

type controlBreak struct{}

func (controlBreak) isControlReturn() {}

type controlContinue struct{}

func (controlContinue) isControlReturn() {}

type functionReturn struct {
	Value Value
}

func (functionReturn) isControlReturn() {}

type ExpressionStatementResult struct {
	Value Value
}

//

var emptyFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: &sema.TypeAnnotation{
		Type: sema.VoidType,
	},
}

//

type getterSetter struct {
	target Value
	// allowMissing may be true when the got value is nil.
	// For example, this is the case when a field is initialized
	// with the force-assignment operator (which checks the existing value)
	get func(allowMissing bool) Value
	set func(Value)
}

// Visit-methods for statement which return a non-nil value
// are treated like they are returning a value.

// OnEventEmittedFunc is a function that is triggered when an event is emitted by the program.
//
type OnEventEmittedFunc func(
	inter *Interpreter,
	getLocationRange func() LocationRange,
	event *CompositeValue,
	eventType *sema.CompositeType,
) error

// OnStatementFunc is a function that is triggered when a statement is about to be executed.
//
type OnStatementFunc func(
	inter *Interpreter,
	statement ast.Statement,
)

// OnLoopIterationFunc is a function that is triggered when a loop iteration is about to be executed.
//
type OnLoopIterationFunc func(
	inter *Interpreter,
	line int,
)

// OnFunctionInvocationFunc is a function that is triggered when a function is about to be invoked.
//
type OnFunctionInvocationFunc func(
	inter *Interpreter,
	line int,
)

// OnInvokedFunctionReturnFunc is a function that is triggered when an invoked function returned.
//
type OnInvokedFunctionReturnFunc func(
	inter *Interpreter,
	line int,
)

// OnRecordTraceFunc is a function thats records a trace.
type OnRecordTraceFunc func(
	inter *Interpreter,
	operationName string,
	duration time.Duration,
	logs []opentracing.LogRecord,
)

// OnResourceOwnerChangeFunc is a function that is triggered when a resource's owner changes.
type OnResourceOwnerChangeFunc func(
	inter *Interpreter,
	resource *CompositeValue,
	oldOwner common.Address,
	newOwner common.Address,
)

// OnMeterComputationFunc is a function that is called when some computation is about to happen.
// intensity captures the intensity of the computation and can be set using input sizes
// complexity of computation given input sizes, or any other factors that could help the upper levels
// to differentiate same kind of computation with different level (and time) of execution.
type OnMeterComputationFunc func(
	compKind common.ComputationKind,
	intensity uint,
)

// InjectedCompositeFieldsHandlerFunc is a function that handles storage reads.
//
type InjectedCompositeFieldsHandlerFunc func(
	inter *Interpreter,
	location common.Location,
	qualifiedIdentifier string,
	compositeKind common.CompositeKind,
) map[string]Value

// ContractValueHandlerFunc is a function that handles contract values.
//
type ContractValueHandlerFunc func(
	inter *Interpreter,
	compositeType *sema.CompositeType,
	constructorGenerator func(common.Address) *HostFunctionValue,
	invocationRange ast.Range,
) *CompositeValue

// ImportLocationHandlerFunc is a function that handles imports of locations.
//
type ImportLocationHandlerFunc func(
	inter *Interpreter,
	location common.Location,
) Import

// PublicAccountHandlerFunc is a function that handles retrieving a public account at a given address.
// The account returned must be of type `PublicAccount`.
//
type PublicAccountHandlerFunc func(
	inter *Interpreter,
	address AddressValue,
) Value

// UUIDHandlerFunc is a function that handles the generation of UUIDs.
type UUIDHandlerFunc func() (uint64, error)

// PublicKeyValidationHandlerFunc is a function that validates a given public key.
// Parameter types:
// - publicKey: PublicKey
//
type PublicKeyValidationHandlerFunc func(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	publicKey *CompositeValue,
) error

// BLSVerifyPoPHandlerFunc is a function that verifies a BLS proof of possession.
// Parameter types:
// - publicKey: PublicKey
// - signature: [UInt8]
// Expected result type: Bool
//
type BLSVerifyPoPHandlerFunc func(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	publicKey MemberAccessibleValue,
	signature *ArrayValue,
) BoolValue

// BLSAggregateSignaturesHandlerFunc is a function that aggregates multiple BLS signatures.
// Parameter types:
// - signatures: [[UInt8]]
// Expected result type: [UInt8]?
//
type BLSAggregateSignaturesHandlerFunc func(
	inter *Interpreter,
	getLocationRange func() LocationRange,
	signatures *ArrayValue,
) OptionalValue

// BLSAggregatePublicKeysHandlerFunc is a function that aggregates multiple BLS public keys.
// Parameter types:
// - publicKeys: [PublicKey]
// Expected result type: PublicKey?
//
type BLSAggregatePublicKeysHandlerFunc func(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	publicKeys *ArrayValue,
) OptionalValue

// SignatureVerificationHandlerFunc is a function that validates a signature.
// Parameter types:
// - signature: [UInt8]
// - signedData: [UInt8]
// - domainSeparationTag: String
// - hashAlgorithm: HashAlgorithm
// - publicKey: PublicKey
// Expected result type: Bool
//
type SignatureVerificationHandlerFunc func(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	signature *ArrayValue,
	signedData *ArrayValue,
	domainSeparationTag *StringValue,
	hashAlgorithm *CompositeValue,
	publicKey MemberAccessibleValue,
) BoolValue

// HashHandlerFunc is a function that hashes.
// Parameter types:
// - data: [UInt8]
// - domainSeparationTag: [UInt8]
// - hashAlgorithm: HashAlgorithm
// Expected result type: [UInt8]
//
type HashHandlerFunc func(
	inter *Interpreter,
	getLocationRange func() LocationRange,
	data *ArrayValue,
	domainSeparationTag *StringValue,
	hashAlgorithm MemberAccessibleValue,
) *ArrayValue

// ExitHandlerFunc is a function that is called at the end of execution
type ExitHandlerFunc func() error

// CompositeTypeCode contains the "prepared" / "callable" "code"
// for the functions and the destructor of a composite
// (contract, struct, resource, event).
//
// As there is no support for inheritance of concrete types,
// these are the "leaf" nodes in the call chain, and are functions.
//
type CompositeTypeCode struct {
	CompositeFunctions map[string]FunctionValue
	DestructorFunction FunctionValue
}

type FunctionWrapper = func(inner FunctionValue) FunctionValue

// WrapperCode contains the "prepared" / "callable" "code"
// for inherited types (interfaces and type requirements).
//
// These are "branch" nodes in the call chain, and are function wrappers,
// i.e. they wrap the functions / function wrappers that inherit them.
//
type WrapperCode struct {
	InitializerFunctionWrapper FunctionWrapper
	DestructorFunctionWrapper  FunctionWrapper
	FunctionWrappers           map[string]FunctionWrapper
}

// TypeCodes is the value which stores the "prepared" / "callable" "code"
// of all composite types, interface types, and type requirements.
//
type TypeCodes struct {
	CompositeCodes       map[sema.TypeID]CompositeTypeCode
	InterfaceCodes       map[sema.TypeID]WrapperCode
	TypeRequirementCodes map[sema.TypeID]WrapperCode
}

func (c TypeCodes) Merge(codes TypeCodes) {

	// Iterating over the maps in a non-deterministic way is OK,
	// we only copy the values over.

	for typeID, code := range codes.CompositeCodes { //nolint:maprangecheck
		c.CompositeCodes[typeID] = code
	}

	for typeID, code := range codes.InterfaceCodes { //nolint:maprangecheck
		c.InterfaceCodes[typeID] = code
	}

	for typeID, code := range codes.TypeRequirementCodes { //nolint:maprangecheck
		c.TypeRequirementCodes[typeID] = code
	}
}

type Storage interface {
	atree.SlabStorage
	GetStorageMap(address common.Address, domain string) *StorageMap
	CheckHealth() error
}

type ReferencedResourceKindedValues map[atree.StorageID]map[ReferenceTrackedResourceKindedValue]struct{}

type Interpreter struct {
	Program                        *Program
	Location                       common.Location
	PredeclaredValues              []ValueDeclaration
	effectivePredeclaredValues     map[string]ValueDeclaration
	activations                    *VariableActivations
	Globals                        GlobalVariables
	allInterpreters                map[common.LocationID]*Interpreter
	typeCodes                      TypeCodes
	Transactions                   []*HostFunctionValue
	Storage                        Storage
	onEventEmitted                 OnEventEmittedFunc
	onStatement                    OnStatementFunc
	onLoopIteration                OnLoopIterationFunc
	onFunctionInvocation           OnFunctionInvocationFunc
	onInvokedFunctionReturn        OnInvokedFunctionReturnFunc
	onRecordTrace                  OnRecordTraceFunc
	onResourceOwnerChange          OnResourceOwnerChangeFunc
	onMeterComputation             OnMeterComputationFunc
	injectedCompositeFieldsHandler InjectedCompositeFieldsHandlerFunc
	contractValueHandler           ContractValueHandlerFunc
	importLocationHandler          ImportLocationHandlerFunc
	publicAccountHandler           PublicAccountHandlerFunc
	uuidHandler                    UUIDHandlerFunc
	PublicKeyValidationHandler     PublicKeyValidationHandlerFunc
	SignatureVerificationHandler   SignatureVerificationHandlerFunc
	BLSVerifyPoPHandler            BLSVerifyPoPHandlerFunc
	BLSAggregateSignaturesHandler  BLSAggregateSignaturesHandlerFunc
	BLSAggregatePublicKeysHandler  BLSAggregatePublicKeysHandlerFunc
	HashHandler                    HashHandlerFunc
	ExitHandler                    ExitHandlerFunc
	interpreted                    bool
	statement                      ast.Statement
	debugger                       *Debugger
	atreeValueValidationEnabled    bool
	atreeStorageValidationEnabled  bool
	tracingEnabled                 bool
	// TODO: ideally this would be a weak map, but Go has no weak references
	referencedResourceKindedValues       ReferencedResourceKindedValues
	invalidatedResourceValidationEnabled bool
	resourceVariables                    map[ResourceKindedValue]*Variable
	memoryGauge                          common.MemoryGauge
}

var _ common.MemoryGauge = &Interpreter{}

type Option func(*Interpreter) error

// WithOnEventEmittedHandler returns an interpreter option which sets
// the given function as the event handler.
//
func WithOnEventEmittedHandler(handler OnEventEmittedFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnEventEmittedHandler(handler)
		return nil
	}
}

// WithOnStatementHandler returns an interpreter option which sets
// the given function as the statement handler.
//
func WithOnStatementHandler(handler OnStatementFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnStatementHandler(handler)
		return nil
	}
}

// WithOnLoopIterationHandler returns an interpreter option which sets
// the given function as the loop iteration handler.
//
func WithOnLoopIterationHandler(handler OnLoopIterationFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnLoopIterationHandler(handler)
		return nil
	}
}

// WithOnFunctionInvocationHandler returns an interpreter option which sets
// the given function as the function invocation handler.
//
func WithOnFunctionInvocationHandler(handler OnFunctionInvocationFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnFunctionInvocationHandler(handler)
		return nil
	}
}

// WithOnInvokedFunctionReturnHandler returns an interpreter option which sets
// the given function as the invoked function return handler.
//
func WithOnInvokedFunctionReturnHandler(handler OnInvokedFunctionReturnFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnInvokedFunctionReturnHandler(handler)
		return nil
	}
}

// WithMemoryGauge returns an interpreter option which sets
// the given object as the memory gauge.
//
func WithMemoryGauge(memoryGauge common.MemoryGauge) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetMemoryGauge(memoryGauge)
		return nil
	}
}

// WithOnRecordTraceHandler returns an interpreter option which sets
// the given function as the record trace handler.
//
func WithOnRecordTraceHandler(handler OnRecordTraceFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnRecordTraceHandler(handler)
		return nil
	}
}

// WithOnResourceOwnerChangeHandler returns an interpreter option which sets
// the given function as the resource owner change handler.
//
func WithOnResourceOwnerChangeHandler(handler OnResourceOwnerChangeFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnResourceOwnerChangeHandler(handler)
		return nil
	}
}

// WithOnMeterComputationFuncHandler returns an interpreter option which sets
// the given function as the meter computation handler.
//
func WithOnMeterComputationFuncHandler(handler OnMeterComputationFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnMeterComputationHandler(handler)
		return nil
	}
}

// WithPredeclaredValues returns an interpreter option which declares
// the given the predeclared values.
//
func WithPredeclaredValues(predeclaredValues []ValueDeclaration) Option {
	return func(interpreter *Interpreter) error {
		interpreter.PredeclaredValues = predeclaredValues

		for _, declaration := range predeclaredValues {
			variable := interpreter.declareValue(declaration)
			if variable == nil {
				continue
			}
			name := declaration.ValueDeclarationName()
			interpreter.Globals.Set(name, variable)
			interpreter.effectivePredeclaredValues[name] = declaration
		}

		return nil
	}
}

// WithStorage returns an interpreter option which sets the given value
// as the function that is used for storage operations.
//
func WithStorage(storage Storage) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetStorage(storage)
		return nil
	}
}

// WithInjectedCompositeFieldsHandler returns an interpreter option which sets the given function
// as the function that is used to initialize new composite values' fields
//
func WithInjectedCompositeFieldsHandler(handler InjectedCompositeFieldsHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetInjectedCompositeFieldsHandler(handler)
		return nil
	}
}

// WithContractValueHandler returns an interpreter option which sets the given function
// as the function that is used to handle imports of values.
//
func WithContractValueHandler(handler ContractValueHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetContractValueHandler(handler)
		return nil
	}
}

// WithImportLocationHandler returns an interpreter option which sets the given function
// as the function that is used to handle the imports of locations.
//
func WithImportLocationHandler(handler ImportLocationHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetImportLocationHandler(handler)
		return nil
	}
}

// WithPublicAccountHandler returns an interpreter option which sets the given function
// as the function that is used to handle public accounts.
//
func WithPublicAccountHandler(handler PublicAccountHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetPublicAccountHandler(handler)
		return nil
	}
}

// WithUUIDHandler returns an interpreter option which sets the given function
// as the function that is used to generate UUIDs.
//
func WithUUIDHandler(handler UUIDHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetUUIDHandler(handler)
		return nil
	}
}

// WithPublicKeyValidationHandler returns an interpreter option which sets the given
// function as the function that is used to handle public key validation.
//
func WithPublicKeyValidationHandler(handler PublicKeyValidationHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetPublicKeyValidationHandler(handler)
		return nil
	}
}

// WithBLSCryptoFunctions returns an interpreter option which sets the given
// functions as the functions used to handle certain BLS-specific crypto functions.
//
func WithBLSCryptoFunctions(
	verifyPoP BLSVerifyPoPHandlerFunc,
	aggregateSignatures BLSAggregateSignaturesHandlerFunc,
	aggregatePublicKeys BLSAggregatePublicKeysHandlerFunc,
) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetBLSCryptoFunctions(
			verifyPoP,
			aggregateSignatures,
			aggregatePublicKeys,
		)
		return nil
	}
}

// WithSignatureVerificationHandler returns an interpreter option which sets the given
// function as the function that is used to handle signature validation.
//
func WithSignatureVerificationHandler(handler SignatureVerificationHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetSignatureVerificationHandler(handler)
		return nil
	}
}

// WithHashHandler returns an interpreter option which sets the given
// function as the function that is used to hash.
//
func WithHashHandler(handler HashHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetHashHandler(handler)
		return nil
	}
}

// WithExitHandler returns an interpreter option which sets the given
// function as the function that is used when execution is complete.
//
func WithExitHandler(handler ExitHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetExitHandler(handler)
		return nil
	}
}

// WithAllInterpreters returns an interpreter option which sets
// the given map of interpreters as the map of all interpreters.
//
func WithAllInterpreters(allInterpreters map[common.LocationID]*Interpreter) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetAllInterpreters(allInterpreters)
		return nil
	}
}

// WithAtreeValueValidationEnabled returns an interpreter option which sets
// the atree validation option.
//
func WithAtreeValueValidationEnabled(enabled bool) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetAtreeValueValidationEnabled(enabled)
		return nil
	}
}

// WithAtreeStorageValidationEnabled returns an interpreter option which sets
// the atree validation option.
//
func WithAtreeStorageValidationEnabled(enabled bool) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetAtreeStorageValidationEnabled(enabled)
		return nil
	}
}

// WithTracingEnabled returns an interpreter option which sets
// the tracing option.
//
func WithTracingEnabled(enabled bool) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetTracingEnabled(enabled)
		return nil
	}
}

// WithInvalidatedResourceValidationEnabled returns an interpreter option which sets
// the resource validation option.
//
func WithInvalidatedResourceValidationEnabled(enabled bool) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetInvalidatedResourceValidationEnabled(enabled)
		return nil
	}
}

// withTypeCodes returns an interpreter option which sets the type codes.
//
func withTypeCodes(typeCodes TypeCodes) Option {
	return func(interpreter *Interpreter) error {
		interpreter.setTypeCodes(typeCodes)
		return nil
	}
}

// withReferencedResourceKindedValues returns an interpreter option which sets the referenced values.
//
func withReferencedResourceKindedValues(referencedResourceKindedValues ReferencedResourceKindedValues) Option {
	return func(interpreter *Interpreter) error {
		interpreter.referencedResourceKindedValues = referencedResourceKindedValues
		return nil
	}
}

// WithDebugger returns an interpreter option which sets the given debugger
//
func WithDebugger(debugger *Debugger) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetDebugger(debugger)
		return nil
	}
}

// Create a base-activation so that it can be reused across all interpreters.
//
var baseActivation = func() *VariableActivation {
	activation := NewVariableActivation(nil)
	defineBaseFunctions(activation)
	return activation
}()

func NewInterpreter(program *Program, location common.Location, options ...Option) (*Interpreter, error) {

	interpreter := &Interpreter{
		Program:                    program,
		Location:                   location,
		activations:                &VariableActivations{},
		Globals:                    map[string]*Variable{},
		effectivePredeclaredValues: map[string]ValueDeclaration{},
		resourceVariables:          map[ResourceKindedValue]*Variable{},
	}

	// Start a new activation/scope for the current program.
	// Use the base activation as the parent.
	interpreter.activations.PushNewWithParent(baseActivation)

	defaultOptions := []Option{
		WithAllInterpreters(map[common.LocationID]*Interpreter{}),
		withTypeCodes(TypeCodes{
			CompositeCodes:       map[sema.TypeID]CompositeTypeCode{},
			InterfaceCodes:       map[sema.TypeID]WrapperCode{},
			TypeRequirementCodes: map[sema.TypeID]WrapperCode{},
		}),
		withReferencedResourceKindedValues(map[atree.StorageID]map[ReferenceTrackedResourceKindedValue]struct{}{}),
		WithInvalidatedResourceValidationEnabled(true),
	}

	for _, option := range defaultOptions {
		err := option(interpreter)
		if err != nil {
			return nil, err
		}
	}

	for _, option := range options {
		err := option(interpreter)
		if err != nil {
			return nil, err
		}
	}

	return interpreter, nil
}

// SetOnEventEmittedHandler sets the function that is triggered when an event is emitted by the program.
//
func (interpreter *Interpreter) SetOnEventEmittedHandler(function OnEventEmittedFunc) {
	interpreter.onEventEmitted = function
}

// SetOnStatementHandler sets the function that is triggered when a statement is about to be executed.
//
func (interpreter *Interpreter) SetOnStatementHandler(function OnStatementFunc) {
	interpreter.onStatement = function
}

// SetOnLoopIterationHandler sets the function that is triggered when a loop iteration is about to be executed.
//
func (interpreter *Interpreter) SetOnLoopIterationHandler(function OnLoopIterationFunc) {
	interpreter.onLoopIteration = function
}

// SetOnFunctionInvocationHandler sets the function that is triggered when a function invocation is about to be executed.
//
func (interpreter *Interpreter) SetOnFunctionInvocationHandler(function OnFunctionInvocationFunc) {
	interpreter.onFunctionInvocation = function
}

// SetOnInvokedFunctionReturnHandler sets the function that is triggered when an invoked function returned.
//
func (interpreter *Interpreter) SetOnInvokedFunctionReturnHandler(function OnInvokedFunctionReturnFunc) {
	interpreter.onInvokedFunctionReturn = function
}

// SetMemoryGauge sets the object as the memory gauge.
//
func (interpreter *Interpreter) SetMemoryGauge(memoryGauge common.MemoryGauge) {
	interpreter.memoryGauge = memoryGauge
}

// SetOnRecordTraceHandler sets the function that is triggered when a trace is recorded.
//
func (interpreter *Interpreter) SetOnRecordTraceHandler(function OnRecordTraceFunc) {
	interpreter.onRecordTrace = function
}

// SetOnResourceOwnerChangeHandler sets the function that is triggered when the owner of a resource changes.
//
func (interpreter *Interpreter) SetOnResourceOwnerChangeHandler(function OnResourceOwnerChangeFunc) {
	interpreter.onResourceOwnerChange = function
}

// SetOnMeterComputationFuncHandler sets the function that is triggered when a computation is about to happen.
//
func (interpreter *Interpreter) SetOnMeterComputationHandler(function OnMeterComputationFunc) {
	interpreter.onMeterComputation = function
}

// SetStorage sets the value that is used for storage operations.
func (interpreter *Interpreter) SetStorage(storage Storage) {
	interpreter.Storage = storage
}

// SetInjectedCompositeFieldsHandler sets the function that is used to initialize
// new composite values' fields
//
func (interpreter *Interpreter) SetInjectedCompositeFieldsHandler(function InjectedCompositeFieldsHandlerFunc) {
	interpreter.injectedCompositeFieldsHandler = function
}

// SetContractValueHandler sets the function that is used to handle imports of values
//
func (interpreter *Interpreter) SetContractValueHandler(function ContractValueHandlerFunc) {
	interpreter.contractValueHandler = function
}

// SetImportLocationHandler sets the function that is used to handle imports of locations.
//
func (interpreter *Interpreter) SetImportLocationHandler(function ImportLocationHandlerFunc) {
	interpreter.importLocationHandler = function
}

// SetPublicAccountHandler sets the function that is used to handle accounts.
//
func (interpreter *Interpreter) SetPublicAccountHandler(function PublicAccountHandlerFunc) {
	interpreter.publicAccountHandler = function
}

// SetUUIDHandler sets the function that is used to handle the generation of UUIDs.
//
func (interpreter *Interpreter) SetUUIDHandler(function UUIDHandlerFunc) {
	interpreter.uuidHandler = function
}

// SetPublicKeyValidationHandler sets the function that is used to handle public key validation.
//
func (interpreter *Interpreter) SetPublicKeyValidationHandler(function PublicKeyValidationHandlerFunc) {
	interpreter.PublicKeyValidationHandler = function
}

// SetBLSCryptoFunctions sets the functions that are used to handle certain BLS specific crypt functions.
//
func (interpreter *Interpreter) SetBLSCryptoFunctions(
	verifyPoP BLSVerifyPoPHandlerFunc,
	aggregateSignatures BLSAggregateSignaturesHandlerFunc,
	aggregatePublicKeys BLSAggregatePublicKeysHandlerFunc,
) {
	interpreter.BLSVerifyPoPHandler = verifyPoP
	interpreter.BLSAggregateSignaturesHandler = aggregateSignatures
	interpreter.BLSAggregatePublicKeysHandler = aggregatePublicKeys
}

// SetSignatureVerificationHandler sets the function that is used to handle signature validation.
//
func (interpreter *Interpreter) SetSignatureVerificationHandler(function SignatureVerificationHandlerFunc) {
	interpreter.SignatureVerificationHandler = function
}

// SetHashHandler sets the function that is used to hash.
//
func (interpreter *Interpreter) SetHashHandler(function HashHandlerFunc) {
	interpreter.HashHandler = function
}

// SetExitHandler sets the function that is used to handle end of execution.
//
func (interpreter *Interpreter) SetExitHandler(function ExitHandlerFunc) {
	interpreter.ExitHandler = function
}

// SetAllInterpreters sets the given map of interpreters as the map of all interpreters.
//
func (interpreter *Interpreter) SetAllInterpreters(allInterpreters map[common.LocationID]*Interpreter) {
	interpreter.allInterpreters = allInterpreters

	// Register self
	if interpreter.Location != nil {
		locationID := interpreter.Location.ID()
		interpreter.allInterpreters[locationID] = interpreter
	}
}

// SetAtreeValueValidationEnabled sets the atree value validation option.
//
func (interpreter *Interpreter) SetAtreeValueValidationEnabled(enabled bool) {
	interpreter.atreeValueValidationEnabled = enabled
}

// SetAtreeStorageValidationEnabled sets the atree storage validation option.
//
func (interpreter *Interpreter) SetAtreeStorageValidationEnabled(enabled bool) {
	interpreter.atreeStorageValidationEnabled = enabled
}

// SetTracingEnabled sets the tracing option.
//
func (interpreter *Interpreter) SetTracingEnabled(enabled bool) {
	interpreter.tracingEnabled = enabled
}

// SetInvalidatedResourceValidationEnabled sets the invalidated resource validation option.
//
func (interpreter *Interpreter) SetInvalidatedResourceValidationEnabled(enabled bool) {
	interpreter.invalidatedResourceValidationEnabled = enabled
}

// setTypeCodes sets the type codes.
//
func (interpreter *Interpreter) setTypeCodes(typeCodes TypeCodes) {
	interpreter.typeCodes = typeCodes
}

// SetDebugger sets the debugger.
//
func (interpreter *Interpreter) SetDebugger(debugger *Debugger) {
	interpreter.debugger = debugger
}

// locationRangeGetter returns a function that returns the location range
// for the given location and positioned element.
//
func locationRangeGetter(location common.Location, hasPosition ast.HasPosition) func() LocationRange {
	return func() LocationRange {
		return LocationRange{
			Location: location,
			Range:    ast.NewRangeFromPositioned(hasPosition),
		}
	}
}

func (interpreter *Interpreter) findVariable(name string) *Variable {
	return interpreter.activations.Find(name)
}

func (interpreter *Interpreter) findOrDeclareVariable(name string) *Variable {
	variable := interpreter.findVariable(name)
	if variable == nil {
		variable = interpreter.declareVariable(name, nil)
	}
	return variable
}

func (interpreter *Interpreter) setVariable(name string, variable *Variable) {
	interpreter.activations.Set(name, variable)
}

func (interpreter *Interpreter) Interpret() (err error) {
	if interpreter.interpreted {
		return
	}

	// recover internal panics and return them as an error
	defer interpreter.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	if interpreter.Program != nil {
		interpreter.Program.Program.Accept(interpreter)
	}

	interpreter.interpreted = true

	return nil
}

// visitGlobalDeclaration firsts interprets the global declaration,
// then finds the declaration and adds it to the globals
//
func (interpreter *Interpreter) visitGlobalDeclaration(declaration ast.Declaration) {
	declaration.Accept(interpreter)
	interpreter.declareGlobal(declaration)
}

func (interpreter *Interpreter) declareGlobal(declaration ast.Declaration) {
	identifier := declaration.DeclarationIdentifier()
	if identifier == nil {
		return
	}
	name := identifier.Identifier
	// NOTE: semantic analysis already checked possible invalid redeclaration
	interpreter.Globals.Set(name, interpreter.findVariable(name))
}

// invokeVariable looks up the function by the given name from global variables,
// checks the function type, and executes the function with the given arguments
func (interpreter *Interpreter) invokeVariable(
	functionName string,
	arguments []Value,
) (
	value Value,
	err error,
) {

	// function must be defined as a global variable
	variable, ok := interpreter.Globals.Get(functionName)
	if !ok {
		return nil, NotDeclaredError{
			ExpectedKind: common.DeclarationKindFunction,
			Name:         functionName,
		}
	}

	variableValue := variable.GetValue()

	// the global variable must be declared as a function
	functionValue, ok := variableValue.(FunctionValue)
	if !ok {
		return nil, NotInvokableError{
			Value: variableValue,
		}
	}

	functionVariable, ok := interpreter.Program.Elaboration.GlobalValues.Get(functionName)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	ty := functionVariable.Type

	// function must be invokable
	functionType, ok := ty.(*sema.FunctionType)

	if !ok {
		return nil, NotInvokableError{
			Value: variableValue,
		}
	}

	return interpreter.prepareInvoke(functionValue, functionType, arguments)
}

func (interpreter *Interpreter) prepareInvokeTransaction(
	index int,
	arguments []Value,
) (value Value, err error) {
	if index >= len(interpreter.Transactions) {
		return nil, TransactionNotDeclaredError{Index: index}
	}

	functionValue := interpreter.Transactions[index]

	transactionType := interpreter.Program.Elaboration.TransactionTypes[index]
	functionType := transactionType.EntryPointFunctionType()

	return interpreter.prepareInvoke(functionValue, functionType, arguments)
}

func (interpreter *Interpreter) prepareInvoke(
	functionValue FunctionValue,
	functionType *sema.FunctionType,
	arguments []Value,
) (
	result Value,
	err error,
) {

	// ensures the invocation's argument count matches the function's parameter count

	parameters := functionType.Parameters
	parameterCount := len(parameters)
	argumentCount := len(arguments)

	if argumentCount != parameterCount {

		// if the function has defined optional parameters,
		// then the provided arguments must be equal to or greater than
		// the number of required parameters.
		if functionType.RequiredArgumentCount == nil ||
			argumentCount < *functionType.RequiredArgumentCount {

			return nil, ArgumentCountError{
				ParameterCount: parameterCount,
				ArgumentCount:  argumentCount,
			}
		}
	}

	getLocationRange := ReturnEmptyLocationRange

	preparedArguments := make([]Value, len(arguments))
	for i, argument := range arguments {
		parameterType := parameters[i].TypeAnnotation.Type

		// converts the argument into the parameter type declared by the function
		preparedArguments[i] = interpreter.ConvertAndBox(getLocationRange, argument, nil, parameterType)
	}

	// NOTE: can't fill argument types, as they are unknown
	invocation := Invocation{
		Arguments:        preparedArguments,
		GetLocationRange: getLocationRange,
		Interpreter:      interpreter,
	}

	return functionValue.invoke(invocation), nil
}

// Invoke invokes a global function with the given arguments
func (interpreter *Interpreter) Invoke(functionName string, arguments ...Value) (value Value, err error) {

	// recover internal panics and return them as an error
	defer interpreter.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	return interpreter.invokeVariable(functionName, arguments)
}

// InvokeFunction invokes a function value with the given invocation
func (interpreter *Interpreter) InvokeFunction(function FunctionValue, invocation Invocation) (value Value, err error) {

	// recover internal panics and return them as an error
	defer interpreter.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	value = function.invoke(invocation)
	return
}

func (interpreter *Interpreter) InvokeTransaction(index int, arguments ...Value) (err error) {

	// recover internal panics and return them as an error
	defer interpreter.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	_, err = interpreter.prepareInvokeTransaction(index, arguments)
	return err
}

func (interpreter *Interpreter) RecoverErrors(onError func(error)) {
	if r := recover(); r != nil {
		var err error
		switch r := r.(type) {
		case goRuntime.Error, ExternalError:
			// Don't recover Go's or external panics
			panic(r)
		case error:
			err = r
		default:
			err = fmt.Errorf("%s", r)
		}

		// if the error is not yet an interpreter error, wrap it
		if _, ok := err.(Error); !ok {

			// wrap the error with position information if needed

			_, ok := err.(ast.HasPosition)
			if !ok && interpreter.statement != nil {
				r := ast.NewRangeFromPositioned(interpreter.statement)

				err = PositionedError{
					Err:   err,
					Range: r,
				}
			}

			err = Error{
				Err:      err,
				Location: interpreter.Location,
			}
		}

		onError(err)
	}
}

func (interpreter *Interpreter) VisitProgram(program *ast.Program) ast.Repr {

	for _, declaration := range program.ImportDeclarations() {
		interpreter.visitGlobalDeclaration(declaration)
	}

	for _, declaration := range program.InterfaceDeclarations() {
		interpreter.visitGlobalDeclaration(declaration)
	}

	for _, declaration := range program.CompositeDeclarations() {
		interpreter.visitGlobalDeclaration(declaration)
	}

	for _, declaration := range program.FunctionDeclarations() {
		interpreter.visitGlobalDeclaration(declaration)
	}

	for _, declaration := range program.TransactionDeclarations() {
		interpreter.visitGlobalDeclaration(declaration)
	}

	// Finally, evaluate the global variable declarations,
	// which are effectively lazy declarations,
	// i.e. the value is evaluated on first access.
	//
	// This enables forward references, especially indirect ones
	// through functions, for example:
	//
	// ```
	// fun f(): Int {
	//    return g()
	// }
	//
	// let x = f()
	// let y = 0
	//
	// fun g(): Int {
	//     return y
	// }
	// ```
	//
	// Here, the variable `x` has an indirect forward reference
	// to variable `y`, through functions `f` and `g`.
	// When variable `x` is evaluated, it forces the evaluation of variable `y`.
	//
	// Variable declarations are still eagerly evaluated,
	// in the order they are declared.

	// First, for each variable declaration, declare a variable with a getter
	// which will evaluate the variable declaration. The resulting value
	// is reused for subsequent reads of the variable.

	variableDeclarationVariables := make([]*Variable, 0, len(program.VariableDeclarations()))

	for _, declaration := range program.VariableDeclarations() {

		// Rebind declaration, so the closure captures to current iteration's value,
		// i.e. the next iteration doesn't override `declaration`

		declaration := declaration

		identifier := declaration.Identifier.Identifier

		var variable *Variable

		variable = NewVariableWithGetter(func() Value {
			var result Value
			interpreter.visitVariableDeclaration(declaration, func(_ string, value Value) {
				result = value
			})

			// Global variables are lazily loaded. Therefore, start resource tracking also
			// lazily when the resource is used for the first time.
			// This is needed to support forward referencing.
			interpreter.startResourceTracking(
				result,
				variable,
				identifier,
				declaration.Identifier,
			)

			return result
		})
		interpreter.setVariable(identifier, variable)
		interpreter.Globals.Set(identifier, variable)

		variableDeclarationVariables = append(variableDeclarationVariables, variable)
	}

	// Second, force the evaluation of all variable declarations,
	// in the order they were declared.

	for _, variable := range variableDeclarationVariables {
		_ = variable.GetValue()
	}

	return nil
}

func (interpreter *Interpreter) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) ast.Repr {

	identifier := declaration.Identifier.Identifier

	functionType := interpreter.Program.Elaboration.FunctionDeclarationFunctionTypes[declaration]

	// NOTE: find *or* declare, as the function might have not been pre-declared (e.g. in the REPL)
	variable := interpreter.findOrDeclareVariable(identifier)

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	// make the function itself available inside the function
	lexicalScope.Set(identifier, variable)

	variable.SetValue(
		interpreter.functionDeclarationValue(
			declaration,
			functionType,
			lexicalScope,
		),
	)

	return nil
}

func (interpreter *Interpreter) functionDeclarationValue(
	declaration *ast.FunctionDeclaration,
	functionType *sema.FunctionType,
	lexicalScope *VariableActivation,
) *InterpretedFunctionValue {

	var preConditions ast.Conditions
	if declaration.FunctionBlock.PreConditions != nil {
		preConditions = *declaration.FunctionBlock.PreConditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

	if declaration.FunctionBlock.PostConditions != nil {
		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite[declaration.FunctionBlock.PostConditions]

		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
		beforeStatements = postConditionsRewrite.BeforeStatements
	}

	return NewInterpretedFunctionValue(
		interpreter,
		declaration.ParameterList,
		functionType,
		lexicalScope,
		beforeStatements,
		preConditions,
		declaration.FunctionBlock.Block.Statements,
		rewrittenPostConditions,
	)
}

func (interpreter *Interpreter) VisitBlock(block *ast.Block) ast.Repr {
	// block scope: each block gets an activation record
	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	return interpreter.visitStatements(block.Statements)
}

func (interpreter *Interpreter) VisitFunctionBlock(_ *ast.FunctionBlock) ast.Repr {
	// NOTE: see visitBlock
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) visitFunctionBody(
	beforeStatements []ast.Statement,
	preConditions ast.Conditions,
	body func() controlReturn,
	postConditions ast.Conditions,
	returnType sema.Type,
) Value {

	// block scope: each function block gets an activation record
	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	result := interpreter.visitStatements(beforeStatements)
	if ret, ok := result.(functionReturn); ok {
		return ret.Value
	}

	interpreter.visitConditions(preConditions)

	var returnValue Value

	if body != nil {
		result = body()
		if ret, ok := result.(functionReturn); ok {
			returnValue = ret.Value
		} else {
			returnValue = NewVoidValue(interpreter)
		}
	} else {
		returnValue = NewVoidValue(interpreter)
	}

	// If there is a return type, declare the constant `result`.
	// If it is a resource type, the constant has the same type as a reference to the return type.
	// If it is not a resource type, the constant has the same type as the return type.

	if returnType != sema.VoidType {
		var resultValue Value
		if returnType.IsResourceType() {
			resultValue = NewEphemeralReferenceValue(interpreter, false, returnValue, returnType)
		} else {
			resultValue = returnValue
		}
		interpreter.declareVariable(
			sema.ResultIdentifier,
			resultValue,
		)
	}

	interpreter.visitConditions(postConditions)

	return returnValue
}

func (interpreter *Interpreter) visitConditions(conditions []*ast.Condition) {
	for _, condition := range conditions {
		interpreter.visitCondition(condition)
	}
}

func (interpreter *Interpreter) visitCondition(condition *ast.Condition) {

	// Evaluate the condition as a statement, so we get position information in case of an error

	statement := &ast.ExpressionStatement{
		Expression: condition.Test,
	}

	result, ok := interpreter.evalStatement(statement).(ExpressionStatementResult)

	value, valueOk := result.Value.(BoolValue)

	if ok && valueOk && bool(value) {
		return
	}

	var message string
	if condition.Message != nil {
		messageValue := interpreter.evalExpression(condition.Message)
		message = messageValue.(*StringValue).Str
	}

	panic(ConditionError{
		ConditionKind: condition.Kind,
		Message:       message,
		LocationRange: locationRangeGetter(interpreter.Location, condition.Test)(),
	})
}

func (interpreter *Interpreter) declareValue(declaration ValueDeclaration) *Variable {

	if !declaration.ValueDeclarationAvailable(interpreter.Location) {
		return nil
	}

	return interpreter.declareVariable(
		declaration.ValueDeclarationName(),
		declaration.ValueDeclarationValue(interpreter),
	)
}

// declareVariable declares a variable in the latest scope
func (interpreter *Interpreter) declareVariable(identifier string, value Value) *Variable {
	// NOTE: semantic analysis already checked possible invalid redeclaration
	variable := NewVariableWithValue(value)
	interpreter.setVariable(identifier, variable)

	// TODO: add proper location info
	interpreter.startResourceTracking(value, variable, identifier, nil)

	return variable
}

func (interpreter *Interpreter) visitAssignment(
	transferOperation ast.TransferOperation,
	targetExpression ast.Expression, targetType sema.Type,
	valueExpression ast.Expression, valueType sema.Type,
	position ast.HasPosition,
) {
	// First evaluate the target, which results in a getter/setter function pair
	getterSetter := interpreter.assignmentGetterSetter(targetExpression)

	getLocationRange := locationRangeGetter(interpreter.Location, position)

	// If the assignment is a forced move,
	// ensure that the target is nil,
	// otherwise panic

	if transferOperation == ast.TransferOperationMoveForced {

		// If the force-move assignment is used for the initialization of a field,
		// then there is no prior value for the field, so allow missing

		const allowMissing = true

		target := getterSetter.get(allowMissing)

		if _, ok := target.(NilValue); !ok && target != nil {
			getLocationRange := locationRangeGetter(interpreter.Location, position)
			panic(ForceAssignmentToNonNilResourceError{
				LocationRange: getLocationRange(),
			})
		}
	}

	// Finally, evaluate the value, and assign it using the setter function

	value := interpreter.evalExpression(valueExpression)

	transferredValue := interpreter.transferAndConvert(value, valueType, targetType, getLocationRange)

	getterSetter.set(transferredValue)
}

// NOTE: only called for top-level composite declarations
func (interpreter *Interpreter) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) ast.Repr {

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	_, _ = interpreter.declareCompositeValue(declaration, lexicalScope)

	return nil
}

// declareCompositeValue creates and declares the value for
// the composite declaration.
//
// For all composite kinds a constructor function is created.
//
// The constructor is a host function which creates a new composite,
// calls the initializer (interpreted function), if any,
// and then returns the composite.
//
// Inside the initializer and all functions, `self` is bound to
// the new composite value, and the constructor itself is bound
//
// For contracts, `contractValueHandler` is used to declare
// a contract value / instance (singleton).
//
// For all other composite kinds the constructor function is declared.
//
func (interpreter *Interpreter) declareCompositeValue(
	declaration *ast.CompositeDeclaration,
	lexicalScope *VariableActivation,
) (
	scope *VariableActivation,
	variable *Variable,
) {
	if declaration.CompositeKind == common.CompositeKindEnum {
		return interpreter.declareEnumConstructor(declaration, lexicalScope)
	} else {
		return interpreter.declareNonEnumCompositeValue(declaration, lexicalScope)
	}
}

func (interpreter *Interpreter) declareNonEnumCompositeValue(
	declaration *ast.CompositeDeclaration,
	lexicalScope *VariableActivation,
) (
	scope *VariableActivation,
	variable *Variable,
) {
	identifier := declaration.Identifier.Identifier
	// NOTE: find *or* declare, as the function might have not been pre-declared (e.g. in the REPL)
	variable = interpreter.findOrDeclareVariable(identifier)

	// Make the value available in the initializer
	lexicalScope.Set(identifier, variable)

	// Evaluate nested declarations in a new scope, so values
	// of nested declarations won't be visible after the containing declaration

	nestedVariables := map[string]*Variable{}

	(func() {
		interpreter.activations.PushNewWithCurrent()
		defer interpreter.activations.Pop()

		// Pre-declare empty variables for all interfaces, composites, and function declarations
		predeclare := func(identifier ast.Identifier) {
			name := identifier.Identifier
			lexicalScope.Set(
				name,
				interpreter.declareVariable(name, nil),
			)
		}

		for _, nestedInterfaceDeclaration := range declaration.Members.Interfaces() {
			predeclare(nestedInterfaceDeclaration.Identifier)
		}

		for _, nestedCompositeDeclaration := range declaration.Members.Composites() {
			predeclare(nestedCompositeDeclaration.Identifier)
		}

		for _, nestedInterfaceDeclaration := range declaration.Members.Interfaces() {
			interpreter.declareInterface(nestedInterfaceDeclaration, lexicalScope)
		}

		for _, nestedCompositeDeclaration := range declaration.Members.Composites() {

			// Pass the lexical scope, which has the containing composite's value declared,
			// to the nested declarations so they can refer to it, and update the lexical scope
			// so the container's functions can refer to the nested composite's value

			var nestedVariable *Variable
			lexicalScope, nestedVariable =
				interpreter.declareCompositeValue(nestedCompositeDeclaration, lexicalScope)

			memberIdentifier := nestedCompositeDeclaration.Identifier.Identifier
			nestedVariables[memberIdentifier] = nestedVariable
		}
	})()

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationTypes[declaration]

	constructorType := &sema.FunctionType{
		IsConstructor: true,
		Parameters:    compositeType.ConstructorParameters,
		ReturnTypeAnnotation: &sema.TypeAnnotation{
			Type: compositeType,
		},
		RequiredArgumentCount: nil,
	}

	var initializerFunction FunctionValue
	if declaration.CompositeKind == common.CompositeKindEvent {
		initializerFunction = NewHostFunctionValue(
			interpreter,
			func(invocation Invocation) Value {
				for i, argument := range invocation.Arguments {
					parameter := compositeType.ConstructorParameters[i]
					invocation.Self.SetMember(
						invocation.Interpreter,
						invocation.GetLocationRange,
						parameter.Identifier,
						argument,
					)
				}
				return nil
			},
			constructorType,
		)
	} else {
		compositeInitializerFunction := interpreter.compositeInitializerFunction(declaration, lexicalScope)
		if compositeInitializerFunction != nil {
			initializerFunction = compositeInitializerFunction
		}
	}

	var destructorFunction FunctionValue
	compositeDestructorFunction := interpreter.compositeDestructorFunction(declaration, lexicalScope)
	if compositeDestructorFunction != nil {
		destructorFunction = compositeDestructorFunction
	}

	functions := interpreter.compositeFunctions(declaration, lexicalScope)

	wrapFunctions := func(code WrapperCode) {

		// Wrap initializer

		initializerFunctionWrapper :=
			code.InitializerFunctionWrapper

		if initializerFunctionWrapper != nil {
			initializerFunction = initializerFunctionWrapper(initializerFunction)
		}

		// Wrap destructor

		destructorFunctionWrapper :=
			code.DestructorFunctionWrapper

		if destructorFunctionWrapper != nil {
			destructorFunction = destructorFunctionWrapper(destructorFunction)
		}

		// Wrap functions

		// Iterating over the map in a non-deterministic way is OK,
		// we only apply the function wrapper to each function,
		// the order does not matter.

		for name, functionWrapper := range code.FunctionWrappers { //nolint:maprangecheck
			functions[name] = functionWrapper(functions[name])
		}
	}

	// NOTE: First the conditions of the type requirements are evaluated,
	//  then the conditions of this composite's conformances
	//
	// Because the conditions are wrappers, they have to be applied
	// in reverse order: first the conformances, then the type requirements;
	// each conformances and type requirements in reverse order as well.

	for i := len(compositeType.ExplicitInterfaceConformances) - 1; i >= 0; i-- {
		conformance := compositeType.ExplicitInterfaceConformances[i]

		wrapFunctions(interpreter.typeCodes.InterfaceCodes[conformance.ID()])
	}

	typeRequirements := compositeType.TypeRequirements()

	for i := len(typeRequirements) - 1; i >= 0; i-- {
		typeRequirement := typeRequirements[i]

		wrapFunctions(interpreter.typeCodes.TypeRequirementCodes[typeRequirement.ID()])
	}

	interpreter.typeCodes.CompositeCodes[compositeType.ID()] = CompositeTypeCode{
		DestructorFunction: destructorFunction,
		CompositeFunctions: functions,
	}

	location := interpreter.Location

	qualifiedIdentifier := compositeType.QualifiedIdentifier()

	constructorGenerator := func(address common.Address) *HostFunctionValue {
		return NewHostFunctionValue(
			interpreter,
			func(invocation Invocation) Value {

				// Check that the resource is constructed
				// in the same location as it was declared

				if compositeType.Kind == common.CompositeKindResource &&
					!common.LocationsMatch(invocation.Interpreter.Location, compositeType.Location) {

					panic(ResourceConstructionError{
						CompositeType: compositeType,
						LocationRange: invocation.GetLocationRange(),
					})
				}

				// Load injected fields
				var injectedFields map[string]Value
				if interpreter.injectedCompositeFieldsHandler != nil {
					injectedFields = interpreter.injectedCompositeFieldsHandler(
						interpreter,
						location,
						qualifiedIdentifier,
						declaration.CompositeKind,
					)
				}

				var fields []CompositeField

				if declaration.CompositeKind == common.CompositeKindResource {

					if interpreter.uuidHandler == nil {
						panic(UUIDUnavailableError{
							LocationRange: invocation.GetLocationRange(),
						})
					}

					uuid, err := interpreter.uuidHandler()
					if err != nil {
						panic(err)
					}

					fields = append(
						fields,
						CompositeField{
							Name: sema.ResourceUUIDFieldName,
							Value: NewUInt64Value(
								interpreter,
								func() uint64 {
									return uuid
								},
							),
						},
					)
				}

				value := NewCompositeValue(
					interpreter,
					location,
					qualifiedIdentifier,
					declaration.CompositeKind,
					fields,
					address,
				)

				value.InjectedFields = injectedFields
				value.Functions = functions
				value.Destructor = destructorFunction

				invocation.Self = value

				if declaration.CompositeKind == common.CompositeKindContract {
					// NOTE: set the variable value immediately, as the contract value
					// needs to be available for nested declarations

					variable.SetValue(value)

					// Also, immediately set the nested values,
					// as the initializer of the contract may use nested declarations

					value.NestedVariables = nestedVariables
				}

				if initializerFunction != nil {
					// NOTE: arguments are already properly boxed by invocation expression

					_ = initializerFunction.invoke(invocation)
				}
				return value
			},
			constructorType,
		)
	}

	// Contract declarations declare a value / instance (singleton),
	// for all other composite kinds, the constructor is declared

	if declaration.CompositeKind == common.CompositeKindContract {
		variable.getter = func() Value {
			positioned := ast.NewRangeFromPositioned(declaration.Identifier)
			contract := interpreter.contractValueHandler(
				interpreter,
				compositeType,
				constructorGenerator,
				positioned,
			)
			contract.NestedVariables = nestedVariables
			return contract
		}
	} else {
		constructor := constructorGenerator(common.Address{})
		constructor.NestedVariables = nestedVariables
		variable.SetValue(constructor)
	}

	return lexicalScope, variable
}

func (interpreter *Interpreter) declareEnumConstructor(
	declaration *ast.CompositeDeclaration,
	lexicalScope *VariableActivation,
) (
	scope *VariableActivation,
	variable *Variable,
) {
	identifier := declaration.Identifier.Identifier
	// NOTE: find *or* declare, as the function might have not been pre-declared (e.g. in the REPL)
	variable = interpreter.findOrDeclareVariable(identifier)

	lexicalScope.Set(identifier, variable)

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationTypes[declaration]
	qualifiedIdentifier := compositeType.QualifiedIdentifier()

	location := interpreter.Location

	intType := sema.IntType

	enumCases := declaration.Members.EnumCases()
	caseValues := make([]*CompositeValue, len(enumCases))

	constructorNestedVariables := map[string]*Variable{}

	for i, enumCase := range enumCases {

		// TODO: replace, avoid conversion
		rawValue := interpreter.convert(
			NewIntValueFromInt64(interpreter, int64(i)),
			intType,
			compositeType.EnumRawType,
		)

		caseValueFields := []CompositeField{
			{
				Name:  sema.EnumRawValueFieldName,
				Value: rawValue,
			},
		}

		caseValue := NewCompositeValue(
			interpreter,
			location,
			qualifiedIdentifier,
			declaration.CompositeKind,
			caseValueFields,
			common.Address{},
		)
		caseValues[i] = caseValue

		constructorNestedVariables[enumCase.Identifier.Identifier] =
			NewVariableWithValue(caseValue)
	}

	getLocationRange := locationRangeGetter(location, declaration)

	value := EnumConstructorFunction(
		interpreter,
		getLocationRange,
		compositeType,
		caseValues,
		constructorNestedVariables,
	)
	variable.SetValue(value)

	return lexicalScope, variable
}

func EnumConstructorFunction(
	inter *Interpreter,
	getLocationRange func() LocationRange,
	enumType *sema.CompositeType,
	caseValues []*CompositeValue,
	nestedVariables map[string]*Variable,
) *HostFunctionValue {

	// Prepare a lookup table based on the big-endian byte representation

	lookupTable := make(map[string]*CompositeValue)

	for _, caseValue := range caseValues {
		rawValue := caseValue.GetField(inter, getLocationRange, sema.EnumRawValueFieldName)
		rawValueBigEndianBytes := rawValue.(IntegerValue).ToBigEndianBytes()
		lookupTable[string(rawValueBigEndianBytes)] = caseValue
	}

	// Prepare the constructor function which performs a lookup in the lookup table

	constructor := NewHostFunctionValue(
		inter,
		func(invocation Invocation) Value {
			rawValue, ok := invocation.Arguments[0].(IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			rawValueArgumentBigEndianBytes := rawValue.ToBigEndianBytes()

			caseValue, ok := lookupTable[string(rawValueArgumentBigEndianBytes)]
			if !ok {
				return NewNilValue(inter)
			}

			return NewSomeValueNonCopying(invocation.Interpreter, caseValue)
		},
		sema.EnumConstructorType(enumType),
	)

	constructor.NestedVariables = nestedVariables

	return constructor
}

func (interpreter *Interpreter) compositeInitializerFunction(
	compositeDeclaration *ast.CompositeDeclaration,
	lexicalScope *VariableActivation,
) *InterpretedFunctionValue {

	// TODO: support multiple overloaded initializers

	initializers := compositeDeclaration.Members.Initializers()
	var initializer *ast.SpecialFunctionDeclaration
	if len(initializers) == 0 {
		return nil
	}

	initializer = initializers[0]
	functionType := interpreter.Program.Elaboration.ConstructorFunctionTypes[initializer]

	parameterList := initializer.FunctionDeclaration.ParameterList

	var preConditions ast.Conditions
	if initializer.FunctionDeclaration.FunctionBlock.PreConditions != nil {
		preConditions = *initializer.FunctionDeclaration.FunctionBlock.PreConditions
	}

	statements := initializer.FunctionDeclaration.FunctionBlock.Block.Statements

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

	postConditions := initializer.FunctionDeclaration.FunctionBlock.PostConditions
	if postConditions != nil {
		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite[postConditions]

		beforeStatements = postConditionsRewrite.BeforeStatements
		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
	}

	return NewInterpretedFunctionValue(
		interpreter,
		parameterList,
		functionType,
		lexicalScope,
		beforeStatements,
		preConditions,
		statements,
		rewrittenPostConditions,
	)
}

func (interpreter *Interpreter) compositeDestructorFunction(
	compositeDeclaration *ast.CompositeDeclaration,
	lexicalScope *VariableActivation,
) *InterpretedFunctionValue {

	destructor := compositeDeclaration.Members.Destructor()
	if destructor == nil {
		return nil
	}

	statements := destructor.FunctionDeclaration.FunctionBlock.Block.Statements

	var preConditions ast.Conditions

	conditions := destructor.FunctionDeclaration.FunctionBlock.PreConditions
	if conditions != nil {
		preConditions = *conditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

	postConditions := destructor.FunctionDeclaration.FunctionBlock.PostConditions
	if postConditions != nil {
		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite[postConditions]

		beforeStatements = postConditionsRewrite.BeforeStatements
		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
	}

	return NewInterpretedFunctionValue(
		interpreter,
		nil,
		emptyFunctionType,
		lexicalScope,
		beforeStatements,
		preConditions,
		statements,
		rewrittenPostConditions,
	)
}

func (interpreter *Interpreter) compositeFunctions(
	compositeDeclaration *ast.CompositeDeclaration,
	lexicalScope *VariableActivation,
) map[string]FunctionValue {

	functions := map[string]FunctionValue{}

	for _, functionDeclaration := range compositeDeclaration.Members.Functions() {
		name := functionDeclaration.Identifier.Identifier
		functions[name] =
			interpreter.compositeFunction(
				functionDeclaration,
				lexicalScope,
			)
	}

	return functions
}

func (interpreter *Interpreter) functionWrappers(
	members *ast.Members,
	lexicalScope *VariableActivation,
) map[string]FunctionWrapper {

	functionWrappers := map[string]FunctionWrapper{}

	for _, functionDeclaration := range members.Functions() {

		functionType := interpreter.Program.Elaboration.FunctionDeclarationFunctionTypes[functionDeclaration]

		name := functionDeclaration.Identifier.Identifier
		functionWrapper := interpreter.functionConditionsWrapper(
			functionDeclaration,
			functionType.ReturnTypeAnnotation.Type,
			lexicalScope,
		)
		if functionWrapper == nil {
			continue
		}
		functionWrappers[name] = functionWrapper
	}

	return functionWrappers
}

func (interpreter *Interpreter) compositeFunction(
	functionDeclaration *ast.FunctionDeclaration,
	lexicalScope *VariableActivation,
) *InterpretedFunctionValue {

	functionType := interpreter.Program.Elaboration.FunctionDeclarationFunctionTypes[functionDeclaration]

	var preConditions ast.Conditions

	if functionDeclaration.FunctionBlock.PreConditions != nil {
		preConditions = *functionDeclaration.FunctionBlock.PreConditions
	}

	var beforeStatements []ast.Statement
	var postConditions ast.Conditions

	if functionDeclaration.FunctionBlock.PostConditions != nil {

		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite[functionDeclaration.FunctionBlock.PostConditions]

		beforeStatements = postConditionsRewrite.BeforeStatements
		postConditions = postConditionsRewrite.RewrittenPostConditions
	}

	parameterList := functionDeclaration.ParameterList
	statements := functionDeclaration.FunctionBlock.Block.Statements

	return NewInterpretedFunctionValue(
		interpreter,
		parameterList,
		functionType,
		lexicalScope,
		beforeStatements,
		preConditions,
		statements,
		postConditions,
	)
}

func (interpreter *Interpreter) VisitFieldDeclaration(_ *ast.FieldDeclaration) ast.Repr {
	// fields aren't interpreted
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) ast.Repr {
	// enum cases aren't interpreted
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) checkValueTransferTargetType(value Value, targetType sema.Type) bool {

	if targetType == nil {
		return true
	}

	valueDynamicType := value.DynamicType(interpreter, SeenReferences{})
	if interpreter.IsSubType(valueDynamicType, targetType) {
		return true
	}

	// Handle function types:
	//
	// Static function types have parameter and return type information.
	// Dynamic function types do not (yet) have parameter and return types information.
	// Therefore, IsSubType currently returns false even in cases where
	// the function value is valid.
	//
	// For now, make this check more lenient and accept any function type (or Any/AnyStruct)

	unwrappedValueDynamicType := UnwrapOptionalDynamicType(valueDynamicType)
	if _, ok := unwrappedValueDynamicType.(FunctionDynamicType); ok {
		unwrappedTargetType := sema.UnwrapOptionalType(targetType)
		if _, ok := unwrappedTargetType.(*sema.FunctionType); ok {
			return true
		}

		switch unwrappedTargetType {
		case sema.AnyStructType, sema.AnyType:
			return true
		}
	}

	return false
}

func (interpreter *Interpreter) transferAndConvert(
	value Value,
	valueType, targetType sema.Type,
	getLocationRange func() LocationRange,
) Value {

	transferredValue := value.Transfer(
		interpreter,
		getLocationRange,
		atree.Address{},
		false,
		nil,
	)

	result := interpreter.ConvertAndBox(
		getLocationRange,
		transferredValue,
		valueType,
		targetType,
	)

	if !interpreter.checkValueTransferTargetType(result, targetType) {
		panic(ValueTransferTypeError{
			TargetType:    targetType,
			LocationRange: getLocationRange(),
		})
	}

	return result
}

// ConvertAndBox converts a value to a target type, and boxes in optionals and any value, if necessary
func (interpreter *Interpreter) ConvertAndBox(
	getLocationRange func() LocationRange,
	value Value,
	valueType, targetType sema.Type,
) Value {
	value = interpreter.convert(value, valueType, targetType)
	return interpreter.BoxOptional(getLocationRange, value, targetType)
}

func (interpreter *Interpreter) convert(value Value, valueType, targetType sema.Type) Value {
	if valueType == nil {
		return value
	}

	if _, valueIsOptional := valueType.(*sema.OptionalType); valueIsOptional {
		return value
	}

	unwrappedTargetType := sema.UnwrapOptionalType(targetType)

	switch unwrappedTargetType {
	case sema.IntType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt(interpreter, value)
		}

	case sema.UIntType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt(interpreter, value)
		}

	// Int*
	case sema.Int8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt8(interpreter, value)
		}

	case sema.Int16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt16(interpreter, value)
		}

	case sema.Int32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt32(interpreter, value)
		}

	case sema.Int64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt64(interpreter, value)
		}

	case sema.Int128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt128(interpreter, value)
		}

	case sema.Int256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt256(interpreter, value)
		}

	// UInt*
	case sema.UInt8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt8(interpreter, value)
		}

	case sema.UInt16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt16(interpreter, value)
		}

	case sema.UInt32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt32(interpreter, value)
		}

	case sema.UInt64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt64(interpreter, value)
		}

	case sema.UInt128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt128(interpreter, value)
		}

	case sema.UInt256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt256(interpreter, value)
		}

	// Word*
	case sema.Word8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord8(interpreter, value)
		}

	case sema.Word16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord16(interpreter, value)
		}

	case sema.Word32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord32(interpreter, value)
		}

	case sema.Word64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord64(interpreter, value)
		}

	// Fix*

	case sema.Fix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertFix64(interpreter, value)
		}

	case sema.UFix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUFix64(interpreter, value)
		}
	}

	switch unwrappedTargetType.(type) {
	case *sema.AddressType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertAddress(interpreter, value)
		}
	}

	return value
}

// BoxOptional boxes a value in optionals, if necessary
func (interpreter *Interpreter) BoxOptional(
	getLocationRange func() LocationRange,
	value Value,
	targetType sema.Type,
) Value {

	inner := value

	for {
		optionalType, ok := targetType.(*sema.OptionalType)
		if !ok {
			break
		}

		switch typedInner := inner.(type) {
		case *SomeValue:
			inner = typedInner.InnerValue(interpreter, getLocationRange)

		case NilValue:
			// NOTE: nested nil will be unboxed!
			return inner

		default:
			value = NewSomeValueNonCopying(interpreter, value)
		}

		targetType = optionalType.Type
	}
	return value
}

func (interpreter *Interpreter) Unbox(getLocationRange func() LocationRange, value Value) Value {
	for {
		some, ok := value.(*SomeValue)
		if !ok {
			return value
		}

		value = some.InnerValue(interpreter, getLocationRange)
	}
}

// NOTE: only called for top-level interface declarations
func (interpreter *Interpreter) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) ast.Repr {

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	interpreter.declareInterface(declaration, lexicalScope)

	return nil
}

func (interpreter *Interpreter) declareInterface(
	declaration *ast.InterfaceDeclaration,
	lexicalScope *VariableActivation,
) {
	// Evaluate nested declarations in a new scope, so values
	// of nested declarations won't be visible after the containing declaration

	(func() {
		interpreter.activations.PushNewWithCurrent()
		defer interpreter.activations.Pop()

		for _, nestedInterfaceDeclaration := range declaration.Members.Interfaces() {
			interpreter.declareInterface(nestedInterfaceDeclaration, lexicalScope)
		}

		for _, nestedCompositeDeclaration := range declaration.Members.Composites() {
			interpreter.declareTypeRequirement(nestedCompositeDeclaration, lexicalScope)
		}
	})()

	interfaceType := interpreter.Program.Elaboration.InterfaceDeclarationTypes[declaration]
	typeID := interfaceType.ID()

	initializerFunctionWrapper := interpreter.initializerFunctionWrapper(declaration.Members, lexicalScope)
	destructorFunctionWrapper := interpreter.destructorFunctionWrapper(declaration.Members, lexicalScope)
	functionWrappers := interpreter.functionWrappers(declaration.Members, lexicalScope)

	interpreter.typeCodes.InterfaceCodes[typeID] = WrapperCode{
		InitializerFunctionWrapper: initializerFunctionWrapper,
		DestructorFunctionWrapper:  destructorFunctionWrapper,
		FunctionWrappers:           functionWrappers,
	}
}

func (interpreter *Interpreter) declareTypeRequirement(
	declaration *ast.CompositeDeclaration,
	lexicalScope *VariableActivation,
) {
	// Evaluate nested declarations in a new scope, so values
	// of nested declarations won't be visible after the containing declaration

	(func() {
		interpreter.activations.PushNewWithCurrent()
		defer interpreter.activations.Pop()

		for _, nestedInterfaceDeclaration := range declaration.Members.Interfaces() {
			interpreter.declareInterface(nestedInterfaceDeclaration, lexicalScope)
		}

		for _, nestedCompositeDeclaration := range declaration.Members.Composites() {
			interpreter.declareTypeRequirement(nestedCompositeDeclaration, lexicalScope)
		}
	})()

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationTypes[declaration]
	typeID := compositeType.ID()

	initializerFunctionWrapper := interpreter.initializerFunctionWrapper(declaration.Members, lexicalScope)
	destructorFunctionWrapper := interpreter.destructorFunctionWrapper(declaration.Members, lexicalScope)
	functionWrappers := interpreter.functionWrappers(declaration.Members, lexicalScope)

	interpreter.typeCodes.TypeRequirementCodes[typeID] = WrapperCode{
		InitializerFunctionWrapper: initializerFunctionWrapper,
		DestructorFunctionWrapper:  destructorFunctionWrapper,
		FunctionWrappers:           functionWrappers,
	}
}

func (interpreter *Interpreter) initializerFunctionWrapper(
	members *ast.Members,
	lexicalScope *VariableActivation,
) FunctionWrapper {

	// TODO: support multiple overloaded initializers

	initializers := members.Initializers()
	if len(initializers) == 0 {
		return nil
	}

	firstInitializer := initializers[0]
	if firstInitializer.FunctionDeclaration.FunctionBlock == nil {
		return nil
	}

	return interpreter.functionConditionsWrapper(
		firstInitializer.FunctionDeclaration,
		sema.VoidType,
		lexicalScope,
	)
}

func (interpreter *Interpreter) destructorFunctionWrapper(
	members *ast.Members,
	lexicalScope *VariableActivation,
) FunctionWrapper {

	destructor := members.Destructor()
	if destructor == nil {
		return nil
	}

	return interpreter.functionConditionsWrapper(
		destructor.FunctionDeclaration,
		sema.VoidType,
		lexicalScope,
	)
}

func (interpreter *Interpreter) functionConditionsWrapper(
	declaration *ast.FunctionDeclaration,
	returnType sema.Type,
	lexicalScope *VariableActivation,
) FunctionWrapper {

	if declaration.FunctionBlock == nil {
		return nil
	}

	var preConditions ast.Conditions
	if declaration.FunctionBlock.PreConditions != nil {
		preConditions = *declaration.FunctionBlock.PreConditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

	if declaration.FunctionBlock.PostConditions != nil {

		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite[declaration.FunctionBlock.PostConditions]

		beforeStatements = postConditionsRewrite.BeforeStatements
		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
	}

	return func(inner FunctionValue) FunctionValue {
		// Construct a raw HostFunctionValue without a type,
		// instead of using NewHostFunctionValue, which requires a type.
		//
		// This host function value is an internally created and used function,
		// and can never be passed around as a value.
		// Hence, the type is not required.

		return &HostFunctionValue{
			Function: func(invocation Invocation) Value {
				// Start a new activation record.
				// Lexical scope: use the function declaration's activation record,
				// not the current one (which would be dynamic scope)
				interpreter.activations.PushNewWithParent(lexicalScope)
				defer interpreter.activations.Pop()

				if declaration.ParameterList != nil {
					interpreter.bindParameterArguments(
						declaration.ParameterList,
						invocation.Arguments,
					)
				}

				if invocation.Self != nil {
					interpreter.declareVariable(sema.SelfIdentifier, invocation.Self)
				}

				// NOTE: The `inner` function might be nil.
				//   This is the case if the conforming type did not declare a function.

				var body func() controlReturn
				if inner != nil {
					// NOTE: It is important to wrap the invocation in a function,
					//  so the inner function isn't invoked here

					body = func() controlReturn {

						// NOTE: It is important to actually return the value returned
						//   from the inner function, otherwise it is lost

						returnValue := inner.invoke(invocation)
						return functionReturn{returnValue}
					}
				}

				return interpreter.visitFunctionBody(
					beforeStatements,
					preConditions,
					body,
					rewrittenPostConditions,
					returnType,
				)
			},
		}
	}
}

func (interpreter *Interpreter) EnsureLoaded(
	location common.Location,
) *Interpreter {
	return interpreter.ensureLoadedWithLocationHandler(
		location,
		func() Import {
			return interpreter.importLocationHandler(interpreter, location)
		},
	)
}

func (interpreter *Interpreter) ensureLoadedWithLocationHandler(
	location common.Location,
	loadLocation func() Import,
) *Interpreter {

	locationID := location.ID()

	// If a sub-interpreter already exists, return it

	subInterpreter := interpreter.allInterpreters[locationID]
	if subInterpreter != nil {
		return subInterpreter
	}

	// Load the import

	var virtualImport *VirtualImport

	imported := loadLocation()

	switch imported := imported.(type) {
	case InterpreterImport:
		subInterpreter = imported.Interpreter
		err := subInterpreter.Interpret()
		if err != nil {
			panic(err)
		}

		return subInterpreter

	case VirtualImport:
		virtualImport = &imported

		var err error
		// NOTE: virtual import, no program
		subInterpreter, err = interpreter.NewSubInterpreter(nil, location)
		if err != nil {
			panic(err)
		}

		// If the imported location is a virtual import,
		// prepare the interpreter

		for _, global := range virtualImport.Globals {
			variable := NewVariableWithValue(global.Value)
			subInterpreter.setVariable(global.Name, variable)
			subInterpreter.Globals.Set(global.Name, variable)
		}

		subInterpreter.typeCodes.
			Merge(virtualImport.TypeCodes)

		// Virtual import does not register interpreter itself,
		// unlike InterpreterImport
		interpreter.allInterpreters[locationID] = subInterpreter

		subInterpreter.Program = &Program{
			Elaboration: virtualImport.Elaboration,
		}

		return subInterpreter

	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) NewSubInterpreter(
	program *Program,
	location common.Location,
	options ...Option,
) (
	*Interpreter,
	error,
) {

	defaultOptions := []Option{
		WithStorage(interpreter.Storage),
		WithPredeclaredValues(interpreter.PredeclaredValues),
		WithOnEventEmittedHandler(interpreter.onEventEmitted),
		WithOnStatementHandler(interpreter.onStatement),
		WithOnLoopIterationHandler(interpreter.onLoopIteration),
		WithOnFunctionInvocationHandler(interpreter.onFunctionInvocation),
		WithOnInvokedFunctionReturnHandler(interpreter.onInvokedFunctionReturn),
		WithInjectedCompositeFieldsHandler(interpreter.injectedCompositeFieldsHandler),
		WithContractValueHandler(interpreter.contractValueHandler),
		WithImportLocationHandler(interpreter.importLocationHandler),
		WithUUIDHandler(interpreter.uuidHandler),
		WithAllInterpreters(interpreter.allInterpreters),
		WithAtreeValueValidationEnabled(interpreter.atreeValueValidationEnabled),
		WithAtreeStorageValidationEnabled(interpreter.atreeStorageValidationEnabled),
		withTypeCodes(interpreter.typeCodes),
		withReferencedResourceKindedValues(interpreter.referencedResourceKindedValues),
		WithPublicAccountHandler(interpreter.publicAccountHandler),
		WithPublicKeyValidationHandler(interpreter.PublicKeyValidationHandler),
		WithSignatureVerificationHandler(interpreter.SignatureVerificationHandler),
		WithHashHandler(interpreter.HashHandler),
		WithBLSCryptoFunctions(
			interpreter.BLSVerifyPoPHandler,
			interpreter.BLSAggregateSignaturesHandler,
			interpreter.BLSAggregatePublicKeysHandler,
		),
		WithDebugger(interpreter.debugger),
		WithExitHandler(interpreter.ExitHandler),
		WithTracingEnabled(interpreter.tracingEnabled),
		WithOnRecordTraceHandler(interpreter.onRecordTrace),
		WithOnResourceOwnerChangeHandler(interpreter.onResourceOwnerChange),
		WithOnMeterComputationFuncHandler(interpreter.onMeterComputation),
		WithMemoryGauge(interpreter.memoryGauge),
	}

	return NewInterpreter(
		program,
		location,
		append(
			defaultOptions,
			options...,
		)...,
	)
}

func (interpreter *Interpreter) storedValueExists(
	storageAddress common.Address,
	domain string,
	identifier string,
) bool {
	accountStorage := interpreter.Storage.GetStorageMap(storageAddress, domain)
	return accountStorage.ValueExists(identifier)
}

func (interpreter *Interpreter) ReadStored(
	storageAddress common.Address,
	domain string,
	identifier string,
) Value {
	accountStorage := interpreter.Storage.GetStorageMap(storageAddress, domain)
	return accountStorage.ReadValue(interpreter, identifier)
}

func (interpreter *Interpreter) writeStored(
	storageAddress common.Address,
	domain string,
	identifier string,
	value Value,
) {
	accountStorage := interpreter.Storage.GetStorageMap(storageAddress, domain)
	accountStorage.WriteValue(interpreter, identifier, value)
}

type ValueConverterDeclaration struct {
	name         string
	convert      func(*Interpreter, Value) Value
	min          Value
	max          Value
	functionType *sema.FunctionType
}

// It would be nice if return types in Go's function types would be covariant
//
var ConverterDeclarations = []ValueConverterDeclaration{
	{
		name:         sema.IntTypeName,
		functionType: sema.NumberConversionFunctionType(sema.IntType),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertInt(interpreter, value)
		},
	},
	{
		name:         sema.UIntTypeName,
		functionType: sema.NumberConversionFunctionType(sema.UIntType),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertUInt(interpreter, value)
		},
		min: NewUnmeteredUIntValueFromBigInt(sema.UIntTypeMin),
	},
	{
		name:         sema.Int8TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int8Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertInt8(interpreter, value)
		},
		min: NewUnmeteredInt8Value(math.MinInt8),
		max: NewUnmeteredInt8Value(math.MaxInt8),
	},
	{
		name:         sema.Int16TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int16Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertInt16(interpreter, value)
		},
		min: NewUnmeteredInt16Value(math.MinInt16),
		max: NewUnmeteredInt16Value(math.MaxInt16),
	},
	{
		name:         sema.Int32TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int32Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertInt32(interpreter, value)
		},
		min: NewUnmeteredInt32Value(math.MinInt32),
		max: NewUnmeteredInt32Value(math.MaxInt32),
	},
	{
		name:         sema.Int64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int64Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertInt64(interpreter, value)
		},
		min: NewUnmeteredInt64Value(math.MinInt64),
		max: NewUnmeteredInt64Value(math.MaxInt64),
	},
	{
		name:         sema.Int128TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int128Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertInt128(interpreter, value)
		},
		min: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
		max: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
	},
	{
		name:         sema.Int256TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int256Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertInt256(interpreter, value)
		},
		min: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
		max: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
	},
	{
		name:         sema.UInt8TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt8Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertUInt8(interpreter, value)
		},
		min: NewUnmeteredUInt8Value(0),
		max: NewUnmeteredUInt8Value(math.MaxUint8),
	},
	{
		name:         sema.UInt16TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt16Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertUInt16(interpreter, value)
		},
		min: NewUnmeteredUInt16Value(0),
		max: NewUnmeteredUInt16Value(math.MaxUint16),
	},
	{
		name:         sema.UInt32TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt32Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertUInt32(interpreter, value)
		},
		min: NewUnmeteredUInt32Value(0),
		max: NewUnmeteredUInt32Value(math.MaxUint32),
	},
	{
		name:         sema.UInt64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt64Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertUInt64(interpreter, value)
		},
		min: NewUnmeteredUInt64Value(0),
		max: NewUnmeteredUInt64Value(math.MaxUint64),
	},
	{
		name:         sema.UInt128TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt128Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertUInt128(interpreter, value)
		},
		min: NewUnmeteredUInt128ValueFromUint64(0),
		max: NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
	},
	{
		name:         sema.UInt256TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt256Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertUInt256(interpreter, value)
		},
		min: NewUnmeteredUInt256ValueFromUint64(0),
		max: NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
	},
	{
		name:         sema.Word8TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Word8Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertWord8(interpreter, value)
		},
		min: NewUnmeteredWord8Value(0),
		max: NewUnmeteredWord8Value(math.MaxUint8),
	},
	{
		name:         sema.Word16TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Word16Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertWord16(interpreter, value)
		},
		min: NewUnmeteredWord16Value(0),
		max: NewUnmeteredWord16Value(math.MaxUint16),
	},
	{
		name:         sema.Word32TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Word32Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertWord32(interpreter, value)
		},
		min: NewUnmeteredWord32Value(0),
		max: NewUnmeteredWord32Value(math.MaxUint32),
	},
	{
		name:         sema.Word64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Word64Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertWord64(interpreter, value)
		},
		min: NewUnmeteredWord64Value(0),
		max: NewUnmeteredWord64Value(math.MaxUint64),
	},
	{
		name:         sema.Fix64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Fix64Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertFix64(interpreter, value)
		},
		min: NewUnmeteredFix64Value(math.MinInt64),
		max: NewUnmeteredFix64Value(math.MaxInt64),
	},
	{
		name:         sema.UFix64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UFix64Type),
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertUFix64(interpreter, value)
		},
		min: NewUnmeteredUFix64Value(0),
		max: NewUnmeteredUFix64Value(math.MaxUint64),
	},
	{
		name:         sema.AddressTypeName,
		functionType: sema.AddressConversionFunctionType,
		convert: func(interpreter *Interpreter, value Value) Value {
			return ConvertAddress(interpreter, value)
		},
	},
	{
		name:         sema.PublicPathType.Name,
		functionType: sema.PublicPathConversionFunctionType,
		convert:      ConvertPublicPath,
	},
	{
		name:         sema.PrivatePathType.Name,
		functionType: sema.PrivatePathConversionFunctionType,
		convert:      ConvertPrivatePath,
	},
	{
		name:         sema.StoragePathType.Name,
		functionType: sema.StoragePathConversionFunctionType,
		convert:      ConvertStoragePath,
	},
}

func lookupInterface(interpreter *Interpreter, typeID string) (*sema.InterfaceType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(typeID)
	// if the typeID is invalid, return nil
	if err != nil {
		return nil, err
	}

	typ, err := interpreter.getInterfaceType(location, qualifiedIdentifier)
	if err != nil {
		return nil, err
	}

	return typ, nil
}

func lookupComposite(interpreter *Interpreter, typeID string) (*sema.CompositeType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(typeID)
	// if the typeID is invalid, return nil
	if err != nil {
		return nil, err
	}

	typ, err := interpreter.GetCompositeType(location, qualifiedIdentifier, common.TypeID(typeID))
	if err != nil {
		return nil, err
	}

	return typ, nil
}

func init() {

	converterNames := make(map[string]struct{}, len(ConverterDeclarations))

	for _, converterDeclaration := range ConverterDeclarations {
		converterNames[converterDeclaration.name] = struct{}{}
	}

	for _, numberType := range sema.AllNumberTypes {

		// Only leaf number types require a converter,
		// "hierarchy" number types don't need one

		switch numberType {
		case sema.NumberType, sema.SignedNumberType,
			sema.IntegerType, sema.SignedIntegerType,
			sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := converterNames[numberType.String()]; !ok {
			panic(fmt.Sprintf("missing converter for number type: %s", numberType))
		}
	}

	// We assign this here because it depends on the interpreter, so this breaks the initialization cycle
	defineBaseValue(
		baseActivation,
		"DictionaryType",
		NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				keyTypeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				valueTypeValue, ok := invocation.Arguments[1].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				keyType := keyTypeValue.Type
				valueType := valueTypeValue.Type

				// if the given key is not a valid dictionary key, it wouldn't make sense to create this type
				if keyType == nil ||
					!sema.IsValidDictionaryKeyType(invocation.Interpreter.MustConvertStaticToSemaType(keyType)) {
					return NewNilValue(invocation.Interpreter)
				}

				return NewSomeValueNonCopying(
					invocation.Interpreter,
					TypeValue{
						Type: DictionaryStaticType{
							KeyType:   keyType,
							ValueType: valueType,
						},
					},
				)
			},
			sema.DictionaryTypeFunctionType,
		))

	defineBaseValue(
		baseActivation,
		"CompositeType",
		NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				typeIDValue, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				typeID := typeIDValue.Str

				composite, err := lookupComposite(invocation.Interpreter, typeID)
				if err != nil {
					return NewNilValue(invocation.Interpreter)
				}

				return NewSomeValueNonCopying(
					invocation.Interpreter,
					TypeValue{
						Type: ConvertSemaToStaticType(composite),
					},
				)
			},
			sema.CompositeTypeFunctionType,
		),
	)

	defineBaseValue(
		baseActivation,
		"InterfaceType",
		NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				typeIDValue, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				typeID := typeIDValue.Str

				interfaceType, err := lookupInterface(invocation.Interpreter, typeID)
				if err != nil {
					return NewNilValue(invocation.Interpreter)
				}

				return NewSomeValueNonCopying(
					invocation.Interpreter,
					TypeValue{
						Type: ConvertSemaToStaticType(interfaceType),
					},
				)
			},
			sema.InterfaceTypeFunctionType,
		),
	)

	defineBaseValue(
		baseActivation,
		"FunctionType",
		NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				parameters, ok := invocation.Arguments[0].(*ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				typeValue, ok := invocation.Arguments[1].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				returnType := invocation.Interpreter.MustConvertStaticToSemaType(typeValue.Type)
				parameterTypes := make([]*sema.Parameter, 0, parameters.Count())
				parameters.Iterate(invocation.Interpreter, func(param Value) bool {
					semaType := invocation.Interpreter.MustConvertStaticToSemaType(param.(TypeValue).Type)
					parameterTypes = append(
						parameterTypes,
						&sema.Parameter{
							TypeAnnotation: sema.NewTypeAnnotation(semaType),
						},
					)

					// Continue iteration
					return true
				})
				functionStaticType := FunctionStaticType{
					Type: &sema.FunctionType{
						ReturnTypeAnnotation: sema.NewTypeAnnotation(returnType),
						Parameters:           parameterTypes,
					},
				}
				return NewUnmeteredTypeValue(functionStaticType)
			},
			sema.FunctionTypeFunctionType,
		),
	)

	defineBaseValue(
		baseActivation,
		"RestrictedType",
		NewUnmeteredHostFunctionValue(
			RestrictedTypeFunction,
			sema.RestrictedTypeFunctionType,
		),
	)
}

func RestrictedTypeFunction(invocation Invocation) Value {
	interpreter := invocation.Interpreter

	restrictionIDs, ok := invocation.Arguments[1].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	staticRestrictions := make([]InterfaceStaticType, 0, restrictionIDs.Count())
	semaRestrictions := make([]*sema.InterfaceType, 0, restrictionIDs.Count())

	var invalidRestrictionID bool
	restrictionIDs.Iterate(invocation.Interpreter, func(typeID Value) bool {
		typeIDValue, ok := typeID.(*StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		restrictionInterface, err := lookupInterface(interpreter, typeIDValue.Str)
		if err != nil {
			invalidRestrictionID = true
			return true
		}

		staticRestrictions = append(
			staticRestrictions,
			ConvertSemaToStaticType(restrictionInterface).(InterfaceStaticType),
		)
		semaRestrictions = append(semaRestrictions, restrictionInterface)

		// Continue iteration
		return true
	})

	// If there are any invalid restrictions,
	// then return nil
	if invalidRestrictionID {
		return NewNilValue(invocation.Interpreter)
	}

	var semaType sema.Type
	var err error

	switch typeID := invocation.Arguments[0].(type) {
	case NilValue:
		semaType = nil
	case *SomeValue:
		innerValue := typeID.InnerValue(interpreter, invocation.GetLocationRange)
		semaType, err = lookupComposite(interpreter, innerValue.(*StringValue).Str)
		if err != nil {
			return NewNilValue(invocation.Interpreter)
		}
	default:
		panic(errors.NewUnreachableError())
	}

	var invalidRestrictedType bool
	ty := sema.CheckRestrictedType(
		semaType,
		semaRestrictions,
		func(_ func(*ast.RestrictedType) error) {
			invalidRestrictedType = true
		},
	)

	// If the restricted type would have failed to type-check statically,
	// then return nil
	if invalidRestrictedType {
		return NewNilValue(invocation.Interpreter)
	}

	return NewSomeValueNonCopying(
		interpreter,
		TypeValue{
			Type: &RestrictedStaticType{
				Type:         ConvertSemaToStaticType(ty),
				Restrictions: staticRestrictions,
			},
		},
	)
}

func defineBaseFunctions(activation *VariableActivation) {
	defineConverterFunctions(activation)
	defineTypeFunction(activation)
	defineRuntimeTypeConstructorFunctions(activation)
	defineStringFunction(activation)
}

type converterFunction struct {
	name      string
	converter *HostFunctionValue
}

// Converter functions are stateless functions. Hence they can be re-used across interpreters.
//
var converterFunctionValues = func() []converterFunction {

	converterFuncValues := make([]converterFunction, len(ConverterDeclarations))

	for index, declaration := range ConverterDeclarations {
		// NOTE: declare in loop, as captured in closure below
		convert := declaration.convert
		converterFunctionValue := NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				return convert(invocation.Interpreter, invocation.Arguments[0])
			},
			declaration.functionType,
		)

		addMember := func(name string, value Value) {
			if converterFunctionValue.NestedVariables == nil {
				converterFunctionValue.NestedVariables = map[string]*Variable{}
			}
			converterFunctionValue.NestedVariables[name] = NewVariableWithValue(value)
		}

		if declaration.min != nil {
			addMember(sema.NumberTypeMinFieldName, declaration.min)
		}

		if declaration.max != nil {
			addMember(sema.NumberTypeMaxFieldName, declaration.max)
		}

		converterFuncValues[index] = converterFunction{
			name:      declaration.name,
			converter: converterFunctionValue,
		}
	}

	return converterFuncValues
}()

func defineConverterFunctions(activation *VariableActivation) {
	for _, converterFunc := range converterFunctionValues {
		defineBaseValue(activation, converterFunc.name, converterFunc.converter)
	}
}

type runtimeTypeConstructor struct {
	name      string
	converter *HostFunctionValue
}

// Constructor functions are stateless functions. Hence they can be re-used across interpreters.
//
var runtimeTypeConstructors = []runtimeTypeConstructor{
	{
		name: "OptionalType",
		converter: NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return TypeValue{
					//nolint:gosimple
					Type: OptionalStaticType{
						Type: typeValue.Type,
					},
				}
			},
			sema.OptionalTypeFunctionType,
		),
	},
	{
		name: "VariableSizedArrayType",
		converter: NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return TypeValue{
					//nolint:gosimple
					Type: VariableSizedStaticType{
						Type: typeValue.Type,
					},
				}
			},
			sema.VariableSizedArrayTypeFunctionType,
		),
	},
	{
		name: "ConstantSizedArrayType",
		converter: NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				sizeValue, ok := invocation.Arguments[1].(IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return TypeValue{
					Type: ConstantSizedStaticType{
						Type: typeValue.Type,
						Size: int64(sizeValue.ToInt()),
					},
				}
			},
			sema.ConstantSizedArrayTypeFunctionType,
		),
	},
	{
		name: "ReferenceType",
		converter: NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				authorizedValue, ok := invocation.Arguments[0].(BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				typeValue, ok := invocation.Arguments[1].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return TypeValue{
					Type: ReferenceStaticType{
						Authorized: bool(authorizedValue),
						Type:       typeValue.Type,
					},
				}
			},
			sema.ReferenceTypeFunctionType,
		),
	},
	{
		name: "CapabilityType",
		converter: NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				ty := typeValue.Type
				// Capabilities must hold references
				_, ok = ty.(ReferenceStaticType)
				if !ok {
					return NewNilValue(invocation.Interpreter)
				}

				return NewSomeValueNonCopying(
					invocation.Interpreter,
					TypeValue{
						Type: CapabilityStaticType{
							BorrowType: ty,
						},
					},
				)
			},
			sema.CapabilityTypeFunctionType,
		),
	},
}

func defineRuntimeTypeConstructorFunctions(activation *VariableActivation) {
	for _, constructorFunc := range runtimeTypeConstructors {
		defineBaseValue(activation, constructorFunc.name, constructorFunc.converter)
	}
}

// typeFunction is the `Type` function. It is stateless, hence it can be re-used across interpreters.
//
var typeFunction = NewUnmeteredHostFunctionValue(
	func(invocation Invocation) Value {

		typeParameterPair := invocation.TypeParameterTypes.Oldest()
		if typeParameterPair == nil {
			panic(errors.NewUnreachableError())
		}

		ty := typeParameterPair.Value

		// TODO TypeValue metering is more complicated.
		// 	    Here, staticType conversion should be delayed but can't be.
		staticType := ConvertSemaToStaticType(ty)
		return NewUnmeteredTypeValue(staticType)
	},
	&sema.FunctionType{
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.MetaType),
	},
)

func defineTypeFunction(activation *VariableActivation) {
	defineBaseValue(activation, sema.MetaTypeName, typeFunction)
}

func defineBaseValue(activation *VariableActivation, name string, value Value) {
	if activation.Find(name) != nil {
		panic(errors.NewUnreachableError())
	}
	activation.Set(name, NewVariableWithValue(value))
}

// stringFunction is the `String` function. It is stateless, hence it can be re-used across interpreters.
//
var stringFunction = func() Value {
	functionValue := NewUnmeteredHostFunctionValue(
		func(invocation Invocation) Value {
			return emptyString
		},
		&sema.FunctionType{
			ReturnTypeAnnotation: sema.NewTypeAnnotation(
				sema.StringType,
			),
		},
	)

	addMember := func(name string, value Value) {
		if functionValue.NestedVariables == nil {
			functionValue.NestedVariables = map[string]*Variable{}
		}
		functionValue.NestedVariables[name] = NewVariableWithValue(value)
	}

	addMember(
		sema.StringTypeEncodeHexFunctionName,
		NewUnmeteredHostFunctionValue(
			func(invocation Invocation) Value {
				argument, ok := invocation.Arguments[0].(*ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				inter := invocation.Interpreter
				memoryUsage := common.NewStringMemoryUsage(
					safeMul(argument.Count(), 2),
				)
				return NewStringValue(
					inter,
					memoryUsage,
					func() string {
						// TODO: meter
						bytes, _ := ByteArrayValueToByteSlice(inter, argument)
						return hex.EncodeToString(bytes)
					},
				)
			},
			sema.StringTypeEncodeHexFunctionType,
		),
	)

	return functionValue
}()

func defineStringFunction(activation *VariableActivation) {
	defineBaseValue(activation, sema.StringType.String(), stringFunction)
}

// TODO:
// - FunctionType
//
// - Character
// - Block

func (interpreter *Interpreter) IsSubType(subType DynamicType, superType sema.Type) bool {
	if superType == sema.AnyType {
		return true
	}

	switch typedSubType := subType.(type) {
	case MetaTypeDynamicType:
		switch superType {
		case sema.AnyStructType, sema.MetaType:
			return true
		}

	case VoidDynamicType:
		switch superType {
		case sema.AnyStructType, sema.VoidType:
			return true
		}

	case CharacterDynamicType:
		switch superType {
		case sema.AnyStructType, sema.CharacterType:
			return true
		}

	case StringDynamicType:
		switch superType {
		case sema.AnyStructType, sema.StringType:
			return true
		}

	case BoolDynamicType:
		switch superType {
		case sema.AnyStructType, sema.BoolType:
			return true
		}

	case AddressDynamicType:
		if _, ok := superType.(*sema.AddressType); ok {
			return true
		}

		return superType == sema.AnyStructType

	case NumberDynamicType:
		return sema.IsSubType(typedSubType.StaticType, superType)

	case FunctionDynamicType:
		if superType == sema.AnyStructType {
			return true
		}

		return sema.IsSubType(typedSubType.FuncType, superType)

	case CompositeDynamicType:
		return sema.IsSubType(typedSubType.StaticType, superType)

	case *ArrayDynamicType:
		var superTypeElementType sema.Type

		switch typedSuperType := superType.(type) {
		case *sema.VariableSizedType:
			superTypeElementType = typedSuperType.Type

			subTypeStaticType := interpreter.MustConvertStaticToSemaType(typedSubType.StaticType)
			if !sema.IsSubType(subTypeStaticType, typedSuperType) {
				return false
			}

		case *sema.ConstantSizedType:
			superTypeElementType = typedSuperType.Type

			subTypeStaticType := interpreter.MustConvertStaticToSemaType(typedSubType.StaticType)
			if !sema.IsSubType(subTypeStaticType, typedSuperType) {
				return false
			}

			if typedSuperType.Size != int64(len(typedSubType.ElementTypes)) {
				return false
			}

		default:
			switch superType {
			case sema.AnyStructType, sema.AnyResourceType:
				return true
			default:
				return false
			}
		}

		for _, elementType := range typedSubType.ElementTypes {
			if !interpreter.IsSubType(elementType, superTypeElementType) {
				return false
			}
		}

		return true

	case *DictionaryDynamicType:

		if typedSuperType, ok := superType.(*sema.DictionaryType); ok {

			subTypeStaticType := interpreter.MustConvertStaticToSemaType(typedSubType.StaticType)
			if !sema.IsSubType(subTypeStaticType, typedSuperType) {
				return false
			}

			for _, entryTypes := range typedSubType.EntryTypes {
				if !interpreter.IsSubType(entryTypes.KeyType, typedSuperType.KeyType) ||
					!interpreter.IsSubType(entryTypes.ValueType, typedSuperType.ValueType) {

					return false
				}
			}

			return true
		}

		switch superType {
		case sema.AnyStructType, sema.AnyResourceType:
			return true
		}

	case NilDynamicType:
		if _, ok := superType.(*sema.OptionalType); ok {
			return true
		}

		switch superType {
		case sema.AnyStructType, sema.AnyResourceType:
			return true
		}

	case SomeDynamicType:
		if typedSuperType, ok := superType.(*sema.OptionalType); ok {
			return interpreter.IsSubType(typedSubType.InnerType, typedSuperType.Type)
		}

		switch superType {
		case sema.AnyStructType, sema.AnyResourceType:
			return true
		}

	case ReferenceDynamicType:
		if typedSuperType, ok := superType.(*sema.ReferenceType); ok {

			// First, check that the dynamic type of the referenced value
			// is a subtype of the super type

			if !interpreter.IsSubType(typedSubType.InnerType(), typedSuperType.Type) {
				return false
			}

			// If the reference value is authorized it may be downcasted

			authorized := typedSubType.Authorized()

			if authorized {
				return true
			}

			// If the reference value is not authorized,
			// it may not be downcasted

			return sema.IsSubType(
				&sema.ReferenceType{
					Authorized: authorized,
					Type:       typedSubType.BorrowedType(),
				},
				typedSuperType,
			)
		}

		return superType == sema.AnyStructType

	case CapabilityDynamicType:
		if typedSuperType, ok := superType.(*sema.CapabilityType); ok {

			if typedSuperType.BorrowType != nil {

				// Capability <: Capability<T>:
				// never

				if typedSubType.BorrowType == nil {
					return false
				}

				// Capability<T> <: Capability<U>:
				// if T <: U

				return sema.IsSubType(
					typedSubType.BorrowType,
					typedSuperType.BorrowType,
				)
			}

			// Capability<T> <: Capability || Capability <: Capability:
			// always

			return true

		}

		return superType == sema.AnyStructType

	case PublicPathDynamicType:
		switch superType {
		case sema.PublicPathType, sema.CapabilityPathType, sema.PathType, sema.AnyStructType:
			return true
		}

	case PrivatePathDynamicType:
		switch superType {
		case sema.PrivatePathType, sema.CapabilityPathType, sema.PathType, sema.AnyStructType:
			return true
		}

	case StoragePathDynamicType:
		switch superType {
		case sema.StoragePathType, sema.PathType, sema.AnyStructType:
			return true
		}

	case DeployedContractDynamicType:
		switch superType {
		case sema.AnyStructType, sema.DeployedContractType:
			return true
		}

	case BlockDynamicType:
		switch superType {
		case sema.AnyStructType, sema.BlockType:
			return true
		}
	}

	return false
}

func (interpreter *Interpreter) authAccountSaveFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {
			value := invocation.Arguments[0]

			path, ok := invocation.Arguments[1].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := path.Domain.Identifier()
			identifier := path.Identifier

			// Prevent an overwrite

			getLocationRange := invocation.GetLocationRange

			if interpreter.storedValueExists(
				address,
				domain,
				identifier,
			) {
				panic(
					OverwriteError{
						Address:       addressValue,
						Path:          path,
						LocationRange: getLocationRange(),
					},
				)
			}

			value = value.Transfer(
				interpreter,
				getLocationRange,
				atree.Address(address),
				true,
				nil,
			)

			// Write new value

			interpreter.writeStored(address, domain, identifier, value)

			return NewVoidValue(invocation.Interpreter)
		},
		sema.AuthAccountTypeSaveFunctionType,
	)
}

func (interpreter *Interpreter) authAccountTypeFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {
			path, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := path.Domain.Identifier()
			identifier := path.Identifier

			value := interpreter.ReadStored(address, domain, identifier)

			if value == nil {
				return NewNilValue(invocation.Interpreter)
			}

			return NewSomeValueNonCopying(
				invocation.Interpreter,
				TypeValue{
					Type: value.StaticType(),
				},
			)
		},

		sema.AuthAccountTypeTypeFunctionType,
	)
}

func (interpreter *Interpreter) authAccountLoadFunction(addressValue AddressValue) *HostFunctionValue {
	return interpreter.authAccountReadFunction(addressValue, true)
}

func (interpreter *Interpreter) authAccountCopyFunction(addressValue AddressValue) *HostFunctionValue {
	return interpreter.authAccountReadFunction(addressValue, false)
}

func (interpreter *Interpreter) authAccountReadFunction(addressValue AddressValue, clear bool) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {
			path, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := path.Domain.Identifier()
			identifier := path.Identifier

			value := interpreter.ReadStored(address, domain, identifier)

			if value == nil {
				return NewNilValue(invocation.Interpreter)
			}

			// If there is value stored for the given path,
			// check that it satisfies the type given as the type argument.

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := typeParameterPair.Value

			dynamicType := value.DynamicType(interpreter, SeenReferences{})
			if !interpreter.IsSubType(dynamicType, ty) {
				panic(ForceCastTypeMismatchError{
					ExpectedType:  ty,
					LocationRange: invocation.GetLocationRange(),
				})
			}

			inter := invocation.Interpreter
			getLocationRange := invocation.GetLocationRange

			// We could also pass remove=true and the storable stored in storage,
			// but passing remove=false here and writing nil below has the same effect
			// TODO: potentially refactor and get storable in storage, pass it and remove=true
			transferredValue := value.Transfer(
				inter,
				getLocationRange,
				atree.Address{},
				false,
				nil,
			)

			// Remove the value from storage,
			// but only if the type check succeeded.
			if clear {
				interpreter.writeStored(address, domain, identifier, nil)
			}

			return NewSomeValueNonCopying(invocation.Interpreter, transferredValue)
		},

		// same as sema.AuthAccountTypeCopyFunctionType
		sema.AuthAccountTypeLoadFunctionType,
	)
}

func (interpreter *Interpreter) authAccountBorrowFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {
			path, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := typeParameterPair.Value

			referenceType, ok := ty.(*sema.ReferenceType)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			reference := NewStorageReferenceValue(
				invocation.Interpreter,
				referenceType.Authorized,
				address,
				path,
				referenceType.Type,
			)

			// Attempt to dereference,
			// which reads the stored value
			// and performs a dynamic type check

			value, err := reference.dereference(interpreter, invocation.GetLocationRange)
			if err != nil {
				panic(err)
			}
			if value == nil {
				return NewNilValue(invocation.Interpreter)
			}

			return NewSomeValueNonCopying(invocation.Interpreter, reference)
		},
		sema.AuthAccountTypeBorrowFunctionType,
	)
}

func (interpreter *Interpreter) authAccountLinkFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			borrowType, ok := typeParameterPair.Value.(*sema.ReferenceType)

			if !ok {
				panic(errors.NewUnreachableError())
			}

			newCapabilityPath, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			targetPath, ok := invocation.Arguments[1].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			newCapabilityDomain := newCapabilityPath.Domain.Identifier()
			newCapabilityIdentifier := newCapabilityPath.Identifier

			if interpreter.storedValueExists(
				address,
				newCapabilityDomain,
				newCapabilityIdentifier,
			) {
				return NewNilValue(invocation.Interpreter)
			}

			// Write new value

			borrowStaticType := ConvertSemaToStaticType(borrowType)

			// Note that this will be metered twice if Atree validation is enabled.
			linkValue := NewLinkValue(interpreter, targetPath, borrowStaticType)

			interpreter.writeStored(
				address,
				newCapabilityDomain,
				newCapabilityIdentifier,
				linkValue,
			)

			return NewSomeValueNonCopying(
				invocation.Interpreter,
				NewCapabilityValue(
					invocation.Interpreter,
					addressValue,
					newCapabilityPath,
					borrowStaticType,
				),
			)

		},
		sema.AuthAccountTypeLinkFunctionType,
	)
}

func (interpreter *Interpreter) accountGetLinkTargetFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {

			capabilityPath, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := capabilityPath.Domain.Identifier()
			identifier := capabilityPath.Identifier

			value := interpreter.ReadStored(address, domain, identifier)

			if value == nil {
				return NewNilValue(invocation.Interpreter)
			}

			link, ok := value.(LinkValue)
			if !ok {
				return NewNilValue(invocation.Interpreter)
			}

			return NewSomeValueNonCopying(invocation.Interpreter, link.TargetPath)
		},
		sema.AccountTypeGetLinkTargetFunctionType,
	)
}

func (interpreter *Interpreter) authAccountUnlinkFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {

			capabilityPath, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := capabilityPath.Domain.Identifier()
			identifier := capabilityPath.Identifier

			// Write new value

			interpreter.writeStored(address, domain, identifier, nil)

			return NewVoidValue(invocation.Interpreter)
		},
		sema.AuthAccountTypeUnlinkFunctionType,
	)
}

func (interpreter *Interpreter) capabilityBorrowFunction(
	addressValue AddressValue,
	pathValue PathValue,
	borrowType *sema.ReferenceType,
) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {

			if borrowType == nil {
				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair != nil {
					ty := typeParameterPair.Value
					var ok bool
					borrowType, ok = ty.(*sema.ReferenceType)
					if !ok {
						panic(errors.NewUnreachableError())
					}

				}
			}

			if borrowType == nil {
				panic(errors.NewUnreachableError())
			}

			targetPath, authorized, err :=
				interpreter.GetCapabilityFinalTargetPath(
					address,
					pathValue,
					borrowType,
					invocation.GetLocationRange,
				)
			if err != nil {
				panic(err)
			}

			if targetPath == EmptyPathValue {
				return NewNilValue(invocation.Interpreter)
			}

			reference := NewStorageReferenceValue(
				invocation.Interpreter,
				authorized,
				address,
				targetPath,
				borrowType.Type,
			)

			// Attempt to dereference,
			// which reads the stored value
			// and performs a dynamic type check

			value, err := reference.dereference(interpreter, invocation.GetLocationRange)
			if err != nil {
				panic(err)
			}
			if value == nil {
				return NewNilValue(invocation.Interpreter)
			}

			return NewSomeValueNonCopying(invocation.Interpreter, reference)
		},
		sema.CapabilityTypeBorrowFunctionType(borrowType),
	)
}

func (interpreter *Interpreter) capabilityCheckFunction(
	addressValue AddressValue,
	pathValue PathValue,
	borrowType *sema.ReferenceType,
) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {

			if borrowType == nil {

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair != nil {
					ty := typeParameterPair.Value
					var ok bool
					borrowType, ok = ty.(*sema.ReferenceType)
					if !ok {
						panic(errors.NewUnreachableError())
					}

				}
			}

			if borrowType == nil {
				panic(errors.NewUnreachableError())
			}

			targetPath, authorized, err :=
				interpreter.GetCapabilityFinalTargetPath(
					address,
					pathValue,
					borrowType,
					invocation.GetLocationRange,
				)
			if err != nil {
				panic(err)
			}

			if targetPath == EmptyPathValue {
				return NewBoolValue(invocation.Interpreter, false)
			}

			reference := NewStorageReferenceValue(
				invocation.Interpreter,
				authorized,
				address,
				targetPath,
				borrowType.Type,
			)

			// Attempt to dereference,
			// which reads the stored value
			// and performs a dynamic type check

			if reference.ReferencedValue(interpreter) == nil {
				return NewBoolValue(invocation.Interpreter, false)
			}

			return NewBoolValue(invocation.Interpreter, true)
		},
		sema.CapabilityTypeCheckFunctionType(borrowType),
	)
}

func (interpreter *Interpreter) GetCapabilityFinalTargetPath(
	address common.Address,
	path PathValue,
	wantedBorrowType *sema.ReferenceType,
	getLocationRange func() LocationRange,
) (
	finalPath PathValue,
	authorized bool,
	err error,
) {
	wantedReferenceType := wantedBorrowType

	seenPaths := map[PathValue]struct{}{}
	paths := []PathValue{path}

	for {
		// Detect cyclic links

		if _, ok := seenPaths[path]; ok {
			return EmptyPathValue, false, CyclicLinkError{
				Address:       address,
				Paths:         paths,
				LocationRange: getLocationRange(),
			}
		} else {
			seenPaths[path] = struct{}{}
		}

		value := interpreter.ReadStored(
			address,
			path.Domain.Identifier(),
			path.Identifier,
		)

		if value == nil {
			return EmptyPathValue, false, nil
		}

		if link, ok := value.(LinkValue); ok {

			allowedType := interpreter.MustConvertStaticToSemaType(link.Type)

			if !sema.IsSubType(allowedType, wantedBorrowType) {
				return EmptyPathValue, false, nil
			}

			targetPath := link.TargetPath
			paths = append(paths, targetPath)
			path = targetPath

		} else {
			return path, wantedReferenceType.Authorized, nil
		}
	}
}

func (interpreter *Interpreter) ConvertStaticToSemaType(staticType StaticType) (sema.Type, error) {
	return ConvertStaticToSemaType(
		staticType,
		func(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error) {
			return interpreter.getInterfaceType(location, qualifiedIdentifier)
		},
		func(location common.Location, qualifiedIdentifier string, typeID common.TypeID) (*sema.CompositeType, error) {
			return interpreter.GetCompositeType(location, qualifiedIdentifier, typeID)
		},
	)
}

func (interpreter *Interpreter) MustConvertStaticToSemaType(staticType StaticType) sema.Type {
	semaType, err := interpreter.ConvertStaticToSemaType(staticType)
	if err != nil {
		panic(err)
	}
	return semaType
}

func (interpreter *Interpreter) getElaboration(location common.Location) *sema.Elaboration {

	// Ensure the program for this location is loaded,
	// so its checker is available

	inter := interpreter.EnsureLoaded(location)

	locationID := location.ID()

	subInterpreter := inter.allInterpreters[locationID]
	if subInterpreter == nil || subInterpreter.Program == nil {
		return nil
	}

	return subInterpreter.Program.Elaboration
}

// GetContractComposite gets the composite value of the contract at the address location.
func (interpreter *Interpreter) GetContractComposite(contractLocation common.AddressLocation) (*CompositeValue, error) {
	contractGlobal, ok := interpreter.Globals.Get(contractLocation.Name)
	if !ok {
		return nil, NotDeclaredError{
			ExpectedKind: common.DeclarationKindContract,
			Name:         contractLocation.Name,
		}
	}

	// get contract value
	contractValue, ok := contractGlobal.GetValue().(*CompositeValue)
	if !ok {
		return nil, NotDeclaredError{
			ExpectedKind: common.DeclarationKindContract,
			Name:         contractLocation.Name,
		}
	}

	return contractValue, nil
}

func (interpreter *Interpreter) GetCompositeType(
	location common.Location,
	qualifiedIdentifier string,
	typeID common.TypeID,
) (*sema.CompositeType, error) {
	if location == nil {
		return interpreter.getNativeCompositeType(qualifiedIdentifier)
	}

	return interpreter.getUserCompositeType(location, typeID)
}

func (interpreter *Interpreter) getUserCompositeType(location common.Location, typeID common.TypeID) (*sema.CompositeType, error) {
	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	ty := elaboration.CompositeTypes[typeID]
	if ty == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	return ty, nil
}

func (interpreter *Interpreter) getNativeCompositeType(qualifiedIdentifier string) (*sema.CompositeType, error) {
	ty := sema.NativeCompositeTypes[qualifiedIdentifier]
	if ty == nil {
		return ty, TypeLoadingError{
			TypeID: common.TypeID(qualifiedIdentifier),
		}
	}

	return ty, nil
}

func (interpreter *Interpreter) getInterfaceType(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error) {
	if location == nil {
		return nil, InterfaceMissingLocationError{QualifiedIdentifier: qualifiedIdentifier}
	}

	typeID := location.TypeID(qualifiedIdentifier)

	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	ty := elaboration.InterfaceTypes[typeID]
	if ty == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}
	return ty, nil
}

func (interpreter *Interpreter) reportLoopIteration(pos ast.HasPosition) {
	if interpreter.onMeterComputation != nil {
		interpreter.onMeterComputation(common.ComputationKindLoop, 1)
	}

	if interpreter.onLoopIteration != nil {
		line := pos.StartPosition().Line
		interpreter.onLoopIteration(interpreter, line)
	}
}

func (interpreter *Interpreter) reportFunctionInvocation(line int) {
	if interpreter.onMeterComputation != nil {
		interpreter.onMeterComputation(common.ComputationKindFunctionInvocation, 1)
	}
	if interpreter.onFunctionInvocation != nil {
		interpreter.onFunctionInvocation(interpreter, line)
	}
}

func (interpreter *Interpreter) reportInvokedFunctionReturn(line int) {
	if interpreter.onInvokedFunctionReturn == nil {
		return
	}

	interpreter.onInvokedFunctionReturn(interpreter, line)
}

func (interpreter *Interpreter) ReportComputation(compKind common.ComputationKind, intensity uint) {
	if interpreter.onMeterComputation != nil {
		interpreter.onMeterComputation(compKind, intensity)
	}
}

// getMember gets the member value by the given identifier from the given Value depending on its type.
// May return nil if the member does not exist.
func (interpreter *Interpreter) getMember(self Value, getLocationRange func() LocationRange, identifier string) Value {
	var result Value
	// When the accessed value has a type that supports the declaration of members
	// or is a built-in type that has members (`MemberAccessibleValue`),
	// then try to get the member for the given identifier.
	// For example, the built-in type `String` has a member "length",
	// and composite declarations may contain member declarations
	if memberAccessibleValue, ok := self.(MemberAccessibleValue); ok {
		result = memberAccessibleValue.GetMember(interpreter, getLocationRange, identifier)
	}
	if result == nil {
		switch identifier {
		case sema.IsInstanceFunctionName:
			return interpreter.isInstanceFunction(self)
		case sema.GetTypeFunctionName:
			return interpreter.getTypeFunction(self)
		}
	}

	// NOTE: do not panic if the member is nil. This is a valid state.
	// For example, when a composite field is initialized with a force-assignment, the field's value is read.

	return result
}

func (interpreter *Interpreter) isInstanceFunction(self Value) *HostFunctionValue {
	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {
			firstArgument := invocation.Arguments[0]
			typeValue, ok := firstArgument.(TypeValue)

			if !ok {
				panic(errors.NewUnreachableError())
			}

			staticType := typeValue.Type

			// Values are never instances of unknown types
			if staticType == nil {
				return NewBoolValue(invocation.Interpreter, false)
			}

			semaType := interpreter.MustConvertStaticToSemaType(staticType)
			// NOTE: not invocation.Self, as that is only set for composite values
			dynamicType := self.DynamicType(invocation.Interpreter, SeenReferences{})
			result := interpreter.IsSubType(dynamicType, semaType)
			return NewBoolValue(invocation.Interpreter, result)
		},
		sema.IsInstanceFunctionType,
	)
}

func (interpreter *Interpreter) getTypeFunction(self Value) *HostFunctionValue {
	return NewHostFunctionValue(
		interpreter,
		func(invocation Invocation) Value {
			staticType := self.StaticType()
			return NewUnmeteredTypeValue(staticType)
		},
		sema.GetTypeFunctionType,
	)
}

func (interpreter *Interpreter) setMember(self Value, getLocationRange func() LocationRange, identifier string, value Value) {
	self.(MemberAccessibleValue).SetMember(interpreter, getLocationRange, identifier, value)
}

func (interpreter *Interpreter) ExpectType(
	value Value,
	expectedType sema.Type,
	getLocationRange func() LocationRange,
) {
	dynamicType := value.DynamicType(interpreter, SeenReferences{})
	if !interpreter.IsSubType(dynamicType, expectedType) {
		var locationRange LocationRange
		if getLocationRange != nil {
			locationRange = getLocationRange()
		}
		panic(TypeMismatchError{
			ExpectedType:  expectedType,
			LocationRange: locationRange,
		})
	}
}

func (interpreter *Interpreter) checkContainerMutation(
	elementType StaticType,
	element Value,
	getLocationRange func() LocationRange,
) {
	expectedType := interpreter.MustConvertStaticToSemaType(elementType)
	actualType := element.DynamicType(interpreter, SeenReferences{})

	if !interpreter.IsSubType(actualType, expectedType) {
		panic(ContainerMutationError{
			ExpectedType:  expectedType,
			ActualType:    interpreter.MustConvertStaticToSemaType(element.StaticType()),
			LocationRange: getLocationRange(),
		})
	}
}

func (interpreter *Interpreter) checkResourceNotDestroyed(value Value, getLocationRange func() LocationRange) {
	resourceKindedValue, ok := value.(ResourceKindedValue)
	if !ok || !resourceKindedValue.IsDestroyed() {
		return
	}

	panic(InvalidatedResourceError{
		LocationRange: getLocationRange(),
	})
}

func (interpreter *Interpreter) RemoveReferencedSlab(storable atree.Storable) {
	storageIDStorable, ok := storable.(atree.StorageIDStorable)
	if !ok {
		return
	}

	storageID := atree.StorageID(storageIDStorable)
	err := interpreter.Storage.Remove(storageID)
	if err != nil {
		panic(ExternalError{err})
	}
}

func (interpreter *Interpreter) maybeValidateAtreeValue(v atree.Value) {
	if interpreter.atreeValueValidationEnabled {
		interpreter.ValidateAtreeValue(v)
	}
	if interpreter.atreeStorageValidationEnabled {
		err := interpreter.Storage.CheckHealth()
		if err != nil {
			panic(ExternalError{err})
		}
	}
}

func (interpreter *Interpreter) ValidateAtreeValue(value atree.Value) {
	tic := func(info atree.TypeInfo, other atree.TypeInfo) bool {
		switch info := info.(type) {
		case ConstantSizedStaticType:
			return info.Equal(other.(StaticType))
		case VariableSizedStaticType:
			return info.Equal(other.(StaticType))
		case DictionaryStaticType:
			return info.Equal(other.(StaticType))
		case compositeTypeInfo:
			return info.Equal(other)
		case EmptyTypeInfo:
			_, ok := other.(EmptyTypeInfo)
			return ok
		}
		panic(errors.NewUnreachableError())
	}

	defaultHIP := newHashInputProvider(interpreter, ReturnEmptyLocationRange)

	hip := func(value atree.Value, buffer []byte) ([]byte, error) {
		if _, ok := value.(StringAtreeValue); ok {
			return StringAtreeHashInput(value, buffer)
		}

		return defaultHIP(value, buffer)
	}

	compare := func(storable, otherStorable atree.Storable) bool {
		value, err := storable.StoredValue(interpreter.Storage)
		if err != nil {
			panic(err)
		}

		if _, ok := value.(StringAtreeValue); ok {
			equal, err := StringAtreeComparator(interpreter.Storage, value, otherStorable)
			if err != nil {
				panic(err)
			}

			return equal
		}

		if equatableValue, ok := value.(EquatableValue); ok {
			otherValue := StoredValue(interpreter, otherStorable, interpreter.Storage)
			return equatableValue.Equal(interpreter, ReturnEmptyLocationRange, otherValue)
		}

		// Not all values are comparable, assume valid for now
		return true
	}

	switch value := value.(type) {
	case *atree.Array:
		err := atree.ValidArray(value, value.Type(), tic, hip)
		if err != nil {
			panic(ExternalError{err})
		}

		err = atree.ValidArraySerialization(
			value,
			CBORDecMode,
			CBOREncMode,
			interpreter.DecodeStorable,
			interpreter.DecodeTypeInfo,
			compare,
		)
		if err != nil {
			var nonStorableValueErr NonStorableValueError
			var nonStorableStaticTypeErr NonStorableStaticTypeError

			if !(goErrors.As(err, &nonStorableValueErr) ||
				goErrors.As(err, &nonStorableStaticTypeErr)) {

				atree.PrintArray(value)
				panic(ExternalError{err})
			}
		}

	case *atree.OrderedMap:
		err := atree.ValidMap(value, value.Type(), tic, hip)
		if err != nil {
			panic(ExternalError{err})
		}

		err = atree.ValidMapSerialization(
			value,
			CBORDecMode,
			CBOREncMode,
			interpreter.DecodeStorable,
			interpreter.DecodeTypeInfo,
			compare,
		)
		if err != nil {
			var nonStorableValueErr NonStorableValueError
			var nonStorableStaticTypeErr NonStorableStaticTypeError

			if !(goErrors.As(err, &nonStorableValueErr) ||
				goErrors.As(err, &nonStorableStaticTypeErr)) {

				atree.PrintMap(value)
				panic(ExternalError{err})
			}
		}
	}
}

func (interpreter *Interpreter) trackReferencedResourceKindedValue(
	id atree.StorageID,
	value ReferenceTrackedResourceKindedValue,
) {
	values := interpreter.referencedResourceKindedValues[id]
	if values == nil {
		values = map[ReferenceTrackedResourceKindedValue]struct{}{}
		interpreter.referencedResourceKindedValues[id] = values
	}
	values[value] = struct{}{}
}

func (interpreter *Interpreter) updateReferencedResource(
	currentStorageID atree.StorageID,
	newStorageID atree.StorageID,
	updateFunc func(value ReferenceTrackedResourceKindedValue),
) {
	values := interpreter.referencedResourceKindedValues[currentStorageID]
	if values == nil {
		return
	}
	for value := range values { //nolint:maprangecheck
		updateFunc(value)
	}
	if newStorageID != currentStorageID {
		interpreter.referencedResourceKindedValues[newStorageID] = values
		interpreter.referencedResourceKindedValues[currentStorageID] = nil
	}
}

// startResourceTracking starts tracking the life-span of a resource.
// A resource can only be associated with one variable at most, at a given time.
func (interpreter *Interpreter) startResourceTracking(
	value Value,
	variable *Variable,
	identifier string,
	hasPosition ast.HasPosition,
) {

	if !interpreter.invalidatedResourceValidationEnabled ||
		identifier == sema.SelfIdentifier {
		return
	}

	resourceKindedValue := interpreter.resourceForValidation(value)
	if resourceKindedValue == nil {
		return
	}

	// A resource value can be associated with only one variable at a time.
	// If the resource already has a variable-association, that means there is a
	// resource variable that has not been invalidated properly.
	// This should not be allowed, and must have been caught by the checker ideally.
	if _, exists := interpreter.resourceVariables[resourceKindedValue]; exists {
		var astRange ast.Range
		if hasPosition != nil {
			astRange = ast.NewRangeFromPositioned(hasPosition)
		}

		panic(InvalidatedResourceError{
			LocationRange: LocationRange{
				Location: interpreter.Location,
				Range:    astRange,
			},
		})
	}

	interpreter.resourceVariables[resourceKindedValue] = variable
}

// checkInvalidatedResourceUse checks whether a resource variable is used after invalidation.
func (interpreter *Interpreter) checkInvalidatedResourceUse(
	value Value,
	variable *Variable,
	identifier string,
	hasPosition ast.HasPosition,
) {

	if !interpreter.invalidatedResourceValidationEnabled ||
		identifier == sema.SelfIdentifier {
		return
	}

	resourceKindedValue := interpreter.resourceForValidation(value)
	if resourceKindedValue == nil {
		return
	}

	// A resource value can be associated with only one variable at a time.
	// If the resource already has a variable-association other than the current variable,
	// that means two variables are referring to the same resource at the same time.
	// This should not be allowed, and must have been caught by the checker ideally.
	//
	// Note: if the `resourceVariables` doesn't have a mapping, that implies an invalidated resource.
	if existingVar, exists := interpreter.resourceVariables[resourceKindedValue]; !exists || existingVar != variable {
		panic(InvalidatedResourceError{
			LocationRange: LocationRange{
				Location: interpreter.Location,
				Range:    ast.NewRangeFromPositioned(hasPosition),
			},
		})
	}
}

func (interpreter *Interpreter) resourceForValidation(value Value) ResourceKindedValue {
	switch typedValue := value.(type) {
	case *SomeValue:
		// Optional value's inner value could be nil, if it was a resource
		// and has been invalidated.
		if typedValue.value == nil || value.IsResourceKinded(interpreter) {
			return typedValue
		}
	case ResourceKindedValue:
		if value.IsResourceKinded(interpreter) {
			return typedValue
		}
	}

	return nil
}

func (interpreter *Interpreter) invalidateResource(value Value) {
	if !interpreter.invalidatedResourceValidationEnabled {
		return
	}

	if value == nil || !value.IsResourceKinded(interpreter) {
		return
	}

	resourceKindedValue, ok := value.(ResourceKindedValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Remove the resource-to-variable mapping.
	delete(interpreter.resourceVariables, resourceKindedValue)
}

// UseMemory delegates the memory usage to the interpreter's memory gauge, if any.
//
func (interpreter *Interpreter) MeterMemory(usage common.MemoryUsage) error {
	common.UseMemory(interpreter.memoryGauge, usage)
	return nil
}

// UseConstantMemory uses a pre-determined amount of memory
//
func (interpreter *Interpreter) UseConstantMemory(kind common.MemoryKind) {
	common.UseMemory(interpreter.memoryGauge, common.NewConstantMemoryUsage(kind))
}

func (interpreter *Interpreter) DecodeStorable(
	decoder *cbor.StreamDecoder,
	storageID atree.StorageID,
) (
	atree.Storable,
	error,
) {
	return DecodeStorable(decoder, storageID, interpreter)
}

func (interpreter *Interpreter) DecodeTypeInfo(decoder *cbor.StreamDecoder) (atree.TypeInfo, error) {
	return DecodeTypeInfo(decoder, interpreter)
}
