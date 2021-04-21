/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"fmt"
	"math"
	goRuntime "runtime"

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
	get func() Value
	set func(Value)
}

// Visit-methods for statement which return a non-nil value
// are treated like they are returning a value.

// OnEventEmittedFunc is a function that is triggered when an event is emitted by the program.
//
type OnEventEmittedFunc func(
	inter *Interpreter,
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

// StorageExistenceHandlerFunc is a function that handles storage existence checks.
//
type StorageExistenceHandlerFunc func(
	inter *Interpreter,
	storageAddress common.Address,
	key string,
) bool

// StorageReadHandlerFunc is a function that handles storage reads.
//
type StorageReadHandlerFunc func(
	inter *Interpreter,
	storageAddress common.Address,
	key string,
	deferred bool,
) OptionalValue

// StorageWriteHandlerFunc is a function that handles storage writes.
//
type StorageWriteHandlerFunc func(
	inter *Interpreter,
	storageAddress common.Address,
	key string,
	value OptionalValue,
)

// InjectedCompositeFieldsHandlerFunc is a function that handles storage reads.
//
type InjectedCompositeFieldsHandlerFunc func(
	inter *Interpreter,
	location common.Location,
	qualifiedIdentifier string,
	compositeKind common.CompositeKind,
) *StringValueOrderedMap

// ContractValueHandlerFunc is a function that handles contract values.
//
type ContractValueHandlerFunc func(
	inter *Interpreter,
	compositeType *sema.CompositeType,
	constructor FunctionValue,
	invocationRange ast.Range,
) *CompositeValue

// ImportLocationFunc is a function that handles imports of locations.
//
type ImportLocationHandlerFunc func(
	inter *Interpreter,
	location common.Location,
) Import

// AccountHandlerFunc is a function that handles retrieving a public account at a given address.
// The account returned must be of type `PublicAccount`.
//
type AccountHandlerFunc func(
	address AddressValue,
) *CompositeValue

// UUIDHandlerFunc is a function that handles the generation of UUIDs.
type UUIDHandlerFunc func() (uint64, error)

// PublicKeyValidationHandlerFunc is a function that validates a given public key.
type PublicKeyValidationHandlerFunc func(publicKey *CompositeValue) bool

// CompositeTypeCode contains the the "prepared" / "callable" "code"
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

type Interpreter struct {
	Program                        *Program
	Location                       common.Location
	PredeclaredValues              []ValueDeclaration
	effectivePredeclaredValues     map[string]ValueDeclaration
	activations                    *VariableActivations
	Globals                        map[string]*Variable
	allInterpreters                map[common.LocationID]*Interpreter
	typeCodes                      TypeCodes
	Transactions                   []*HostFunctionValue
	onEventEmitted                 OnEventEmittedFunc
	onStatement                    OnStatementFunc
	onLoopIteration                OnLoopIterationFunc
	onFunctionInvocation           OnFunctionInvocationFunc
	storageExistenceHandler        StorageExistenceHandlerFunc
	storageReadHandler             StorageReadHandlerFunc
	storageWriteHandler            StorageWriteHandlerFunc
	injectedCompositeFieldsHandler InjectedCompositeFieldsHandlerFunc
	contractValueHandler           ContractValueHandlerFunc
	importLocationHandler          ImportLocationHandlerFunc
	accountHandler                 AccountHandlerFunc
	uuidHandler                    UUIDHandlerFunc
	PublicKeyValidationHandler     PublicKeyValidationHandlerFunc
	interpreted                    bool
	statement                      ast.Statement
}

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

// WithOnLoopIterationHandler returns an interpreter option which sets
// the given function as the loop iteration handler.
//
func WithOnFunctionInvocationHandler(handler OnFunctionInvocationFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetOnFunctionInvocationHandler(handler)
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
			interpreter.Globals[name] = variable
			interpreter.effectivePredeclaredValues[name] = declaration
		}

		return nil
	}
}

// WithStorageExistenceHandler returns an interpreter option which sets the given function
// as the function that is used when a storage key is checked for existence.
//
func WithStorageExistenceHandler(handler StorageExistenceHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetStorageExistenceHandler(handler)
		return nil
	}
}

// WithStorageReadHandler returns an interpreter option which sets the given function
// as the function that is used when a stored value is read.
//
func WithStorageReadHandler(handler StorageReadHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetStorageReadHandler(handler)
		return nil
	}
}

// WithStorageWriteHandler returns an interpreter option which sets the given function
// as the function that is used when a stored value is written.
//
func WithStorageWriteHandler(handler StorageWriteHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetStorageWriteHandler(handler)
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

// WithAccountHandlerFunc returns an interpreter option which sets the given function
// as the function that is used to handle public accounts.
//
func WithAccountHandlerFunc(handler AccountHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetAccountHandler(handler)
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

// WithAllInterpreters returns an interpreter option which sets
// the given map of interpreters as the map of all interpreters.
//
func WithAllInterpreters(allInterpreters map[common.LocationID]*Interpreter) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetAllInterpreters(allInterpreters)
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

func NewInterpreter(program *Program, location common.Location, options ...Option) (*Interpreter, error) {

	interpreter := &Interpreter{
		Program:                    program,
		Location:                   location,
		activations:                &VariableActivations{},
		Globals:                    map[string]*Variable{},
		effectivePredeclaredValues: map[string]ValueDeclaration{},
	}

	defaultOptions := []Option{
		WithAllInterpreters(map[common.LocationID]*Interpreter{}),
		withTypeCodes(TypeCodes{
			CompositeCodes:       map[sema.TypeID]CompositeTypeCode{},
			InterfaceCodes:       map[sema.TypeID]WrapperCode{},
			TypeRequirementCodes: map[sema.TypeID]WrapperCode{},
		}),
	}

	interpreter.defineBaseFunctions()

	for _, option := range append(defaultOptions, options...) {
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

// SetOnFunctionInvocationHandler sets the function that is triggered when a loop iteration is about to be executed.
//
func (interpreter *Interpreter) SetOnFunctionInvocationHandler(function OnFunctionInvocationFunc) {
	interpreter.onFunctionInvocation = function
}

// SetStorageExistenceHandler sets the function that is used when a storage key is checked for existence.
//
func (interpreter *Interpreter) SetStorageExistenceHandler(function StorageExistenceHandlerFunc) {
	interpreter.storageExistenceHandler = function
}

// SetStorageReadHandler sets the function that is used when a stored value is read.
//
func (interpreter *Interpreter) SetStorageReadHandler(function StorageReadHandlerFunc) {
	interpreter.storageReadHandler = function
}

// SetStorageWriteHandler sets the function that is used when a stored value is written.
//
func (interpreter *Interpreter) SetStorageWriteHandler(function StorageWriteHandlerFunc) {
	interpreter.storageWriteHandler = function
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

// SetAccountHandler sets the function that is used to handle accounts.
//
func (interpreter *Interpreter) SetAccountHandler(function AccountHandlerFunc) {
	interpreter.accountHandler = function
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

// SetAllInterpreters sets the given map of interpreters as the map of all interpreters.
//
func (interpreter *Interpreter) SetAllInterpreters(allInterpreters map[common.LocationID]*Interpreter) {
	interpreter.allInterpreters = allInterpreters

	// Register self
	interpreter.allInterpreters[interpreter.Location.ID()] = interpreter
}

// setTypeCodes sets the type codes.
//
func (interpreter *Interpreter) setTypeCodes(typeCodes TypeCodes) {
	interpreter.typeCodes = typeCodes
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
	defer interpreter.recoverErrors(func(internalErr error) {
		err = internalErr
	})

	if interpreter.Program != nil {
		interpreter.Program.Program.Accept(interpreter)
	}

	interpreter.interpreted = true

	return nil
}

func (interpreter *Interpreter) prepareInterpretation() {
	program := interpreter.Program.Program

	// Pre-declare empty variables for all interfaces, composites, and function declarations
	for _, declaration := range program.InterfaceDeclarations() {
		interpreter.declareVariable(declaration.Identifier.Identifier, nil)
	}

	for _, declaration := range program.CompositeDeclarations() {
		interpreter.declareVariable(declaration.Identifier.Identifier, nil)
	}

	for _, declaration := range program.FunctionDeclarations() {
		interpreter.declareVariable(declaration.Identifier.Identifier, nil)
	}

	// TODO:
	// Register top-level interface declarations, as their functions' conditions
	// need to be included in conforming composites' functions
}

func (interpreter *Interpreter) visitGlobalDeclarations(declarations []ast.Declaration) {
	for _, declaration := range declarations {
		interpreter.visitGlobalDeclaration(declaration)
	}
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
	interpreter.Globals[name] = interpreter.findVariable(name)
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
	variable, ok := interpreter.Globals[functionName]
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
	invokableType, ok := ty.(sema.InvokableType)

	if !ok {
		return nil, NotInvokableError{
			Value: variableValue,
		}
	}

	functionType := invokableType.InvocationFunctionType()

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

	preparedArguments := make([]Value, len(arguments))
	for i, argument := range arguments {
		parameterType := parameters[i].TypeAnnotation.Type
		// TODO: value type is not known, reject for now
		switch parameterType {
		case sema.AnyStructType, sema.AnyResourceType:
			return nil, NotInvokableError{
				Value: functionValue,
			}
		}

		// converts the argument into the parameter type declared by the function
		preparedArguments[i] = interpreter.convertAndBox(argument, nil, parameterType)
	}

	// NOTE: can't fill argument types, as they are unknown
	invocation := Invocation{
		Arguments:        preparedArguments,
		GetLocationRange: ReturnEmptyLocationRange,
		Interpreter:      interpreter,
	}

	return functionValue.Invoke(invocation), nil
}

// Invoke invokes a global function with the given arguments
func (interpreter *Interpreter) Invoke(functionName string, arguments ...Value) (value Value, err error) {

	// recover internal panics and return them as an error
	defer interpreter.recoverErrors(func(internalErr error) {
		err = internalErr
	})

	return interpreter.invokeVariable(functionName, arguments)
}

func (interpreter *Interpreter) InvokeTransaction(index int, arguments ...Value) (err error) {

	// recover internal panics and return them as an error
	defer interpreter.recoverErrors(func(internalErr error) {
		err = internalErr
	})

	_, err = interpreter.prepareInvokeTransaction(index, arguments)
	return err
}

func (interpreter *Interpreter) recoverErrors(onError func(error)) {
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
	interpreter.prepareInterpretation()

	interpreter.visitGlobalDeclarations(program.Declarations())

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
) InterpretedFunctionValue {

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

	return InterpretedFunctionValue{
		Interpreter:      interpreter,
		ParameterList:    declaration.ParameterList,
		Type:             functionType,
		Activation:       lexicalScope,
		BeforeStatements: beforeStatements,
		PreConditions:    preConditions,
		Statements:       declaration.FunctionBlock.Block.Statements,
		PostConditions:   rewrittenPostConditions,
	}
}

// NOTE: consider using NewInterpreter and WithPredeclaredValues if the value should be predeclared in all locations
//
func (interpreter *Interpreter) ImportValue(name string, value Value) error {
	if _, ok := interpreter.Globals[name]; ok {
		return RedeclarationError{
			Name: name,
		}
	}

	variable := interpreter.declareVariable(name, value)
	interpreter.Globals[name] = variable
	return nil
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

	var resultValue Value

	if body != nil {
		result = body()
		if ret, ok := result.(functionReturn); ok {
			resultValue = ret.Value
		} else {
			resultValue = VoidValue{}
		}
	} else {
		resultValue = VoidValue{}
	}

	// If there is a return type, declare the constant `result`
	// which has the return value

	if returnType != sema.VoidType {
		interpreter.declareVariable(sema.ResultIdentifier, resultValue)
	}

	interpreter.visitConditions(postConditions)

	return resultValue
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

	result := interpreter.evalStatement(statement).(ExpressionStatementResult)

	value := result.Value.(BoolValue)

	if value {
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
		declaration.ValueDeclarationValue(),
	)
}

// declareVariable declares a variable in the latest scope
func (interpreter *Interpreter) declareVariable(identifier string, value Value) *Variable {
	// NOTE: semantic analysis already checked possible invalid redeclaration
	variable := NewVariableWithValue(value)
	interpreter.setVariable(identifier, variable)
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
		target := getterSetter.get()

		// The value may be a NilValue or nil.
		// The latter case exists when the force-move assignment is the initialization of a field
		// in an initializer, in which case there is no prior value for the field.

		if _, ok := target.(NilValue); !ok && target != nil {
			getLocationRange := locationRangeGetter(interpreter.Location, position)
			panic(ForceAssignmentToNonNilResourceError{
				LocationRange: getLocationRange(),
			})
		}
	}

	// Finally, evaluate the value, and assign it using the setter function

	value := interpreter.evalExpression(valueExpression)

	valueCopy := interpreter.copyAndConvert(value, valueType, targetType, getLocationRange)

	getterSetter.set(valueCopy)
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

	nestedVariables := NewStringVariableOrderedMap()

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
			nestedVariables.Set(memberIdentifier, nestedVariable)
		}
	})()

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationTypes[declaration]

	var initializerFunction FunctionValue
	if declaration.CompositeKind == common.CompositeKindEvent {
		initializerFunction = NewHostFunctionValue(
			func(invocation Invocation) Value {
				for i, argument := range invocation.Arguments {
					parameter := compositeType.ConstructorParameters[i]
					invocation.Self.Fields.Set(parameter.Identifier, argument)
				}
				return nil
			},
		)
	} else {
		compositeInitializerFunction := interpreter.compositeInitializerFunction(declaration, lexicalScope)
		if compositeInitializerFunction != nil {
			initializerFunction = *compositeInitializerFunction
		}
	}

	var destructorFunction FunctionValue
	compositeDestructorFunction := interpreter.compositeDestructorFunction(declaration, lexicalScope)
	if compositeDestructorFunction != nil {
		destructorFunction = *compositeDestructorFunction
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

	constructor := NewHostFunctionValue(
		func(invocation Invocation) Value {

			// Load injected fields
			var injectedFields *StringValueOrderedMap
			if interpreter.injectedCompositeFieldsHandler != nil {
				injectedFields = interpreter.injectedCompositeFieldsHandler(
					interpreter,
					location,
					qualifiedIdentifier,
					declaration.CompositeKind,
				)
			}

			fields := NewStringValueOrderedMap()

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

				fields.Set(sema.ResourceUUIDFieldName, UInt64Value(uuid))
			}

			value := &CompositeValue{
				Location:            location,
				QualifiedIdentifier: qualifiedIdentifier,
				Kind:                declaration.CompositeKind,
				Fields:              fields,
				InjectedFields:      injectedFields,
				Functions:           functions,
				Destructor:          destructorFunction,
				// NOTE: new value has no owner
				Owner:    nil,
				modified: true,
			}

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

				_ = initializerFunction.Invoke(invocation)
			}

			return value
		},
	)

	// Contract declarations declare a value / instance (singleton),
	// for all other composite kinds, the constructor is declared

	if declaration.CompositeKind == common.CompositeKindContract {
		variable.getter = func() Value {
			positioned := ast.NewRangeFromPositioned(declaration.Identifier)
			contract := interpreter.contractValueHandler(
				interpreter,
				compositeType,
				constructor,
				positioned,
			)
			contract.NestedVariables = nestedVariables
			return contract
		}
	} else {
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

	constructorNestedVariables := NewStringVariableOrderedMap()

	for i, enumCase := range enumCases {

		rawValue := interpreter.convert(
			NewIntValueFromInt64(int64(i)),
			intType,
			compositeType.EnumRawType,
		)

		caseValueFields := NewStringValueOrderedMap()
		caseValueFields.Set(sema.EnumRawValueFieldName, rawValue)

		caseValue := &CompositeValue{
			Location:            location,
			QualifiedIdentifier: qualifiedIdentifier,
			Kind:                declaration.CompositeKind,
			Fields:              caseValueFields,
			// NOTE: new value has no owner
			Owner:    nil,
			modified: true,
		}
		caseValues[i] = caseValue

		constructorNestedVariables.Set(
			enumCase.Identifier.Identifier,
			NewVariableWithValue(caseValue),
		)
	}

	value := EnumConstructorFunction(caseValues, constructorNestedVariables)
	variable.SetValue(value)

	return lexicalScope, variable
}

func EnumConstructorFunction(caseValues []*CompositeValue, nestedVariables *StringVariableOrderedMap) HostFunctionValue {

	// Prepare a lookup table based on the big-endian byte representation

	lookupTable := make(map[string]*CompositeValue)

	for _, caseValue := range caseValues {
		rawValue, ok := caseValue.Fields.Get(sema.EnumRawValueFieldName)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		rawValueBigEndianBytes := rawValue.(IntegerValue).ToBigEndianBytes()
		lookupTable[string(rawValueBigEndianBytes)] = caseValue
	}

	// Prepare the constructor function which performs a lookup in the lookup table

	constructor := NewHostFunctionValue(
		func(invocation Invocation) Value {

			rawValueArgumentBigEndianBytes := invocation.Arguments[0].(IntegerValue).ToBigEndianBytes()

			caseValue, ok := lookupTable[string(rawValueArgumentBigEndianBytes)]
			if !ok {
				return NilValue{}
			}

			return NewSomeValueOwningNonCopying(caseValue)
		},
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
	functionType := interpreter.Program.Elaboration.ConstructorFunctionTypes[initializer].FunctionType

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

	return &InterpretedFunctionValue{
		Interpreter:      interpreter,
		ParameterList:    parameterList,
		Type:             functionType,
		Activation:       lexicalScope,
		BeforeStatements: beforeStatements,
		PreConditions:    preConditions,
		Statements:       statements,
		PostConditions:   rewrittenPostConditions,
	}
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

	return &InterpretedFunctionValue{
		Interpreter:      interpreter,
		Type:             emptyFunctionType,
		Activation:       lexicalScope,
		BeforeStatements: beforeStatements,
		PreConditions:    preConditions,
		Statements:       statements,
		PostConditions:   rewrittenPostConditions,
	}
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
) InterpretedFunctionValue {

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

	return InterpretedFunctionValue{
		Interpreter:      interpreter,
		ParameterList:    parameterList,
		Type:             functionType,
		Activation:       lexicalScope,
		BeforeStatements: beforeStatements,
		PreConditions:    preConditions,
		Statements:       statements,
		PostConditions:   postConditions,
	}
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

	dynamicTypeResults := DynamicTypeResults{}

	valueDynamicType := value.DynamicType(interpreter, dynamicTypeResults)
	if IsSubType(valueDynamicType, targetType) {
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

func (interpreter *Interpreter) copyAndConvert(
	value Value,
	valueType, targetType sema.Type,
	getLocationRange func() LocationRange,
) Value {

	result := interpreter.convertAndBox(value.Copy(), valueType, targetType)

	if !interpreter.checkValueTransferTargetType(result, targetType) {
		panic(ValueTransferTypeError{
			TargetType:    targetType,
			LocationRange: getLocationRange(),
		})
	}

	return result
}

// convertAndBox converts a value to a target type, and boxes in optionals and any value, if necessary
func (interpreter *Interpreter) convertAndBox(value Value, valueType, targetType sema.Type) Value {
	value = interpreter.convert(value, valueType, targetType)
	return interpreter.boxOptional(value, valueType, targetType)
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
			return ConvertInt(value)
		}

	case sema.UIntType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt(value)
		}

	// Int*
	case sema.Int8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt8(value)
		}

	case sema.Int16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt16(value)
		}

	case sema.Int32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt32(value)
		}

	case sema.Int64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt64(value)
		}

	case sema.Int128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt128(value)
		}

	case sema.Int256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt256(value)
		}

	// UInt*
	case sema.UInt8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt8(value)
		}

	case sema.UInt16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt16(value)
		}

	case sema.UInt32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt32(value)
		}

	case sema.UInt64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt64(value)
		}

	case sema.UInt128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt128(value)
		}

	case sema.UInt256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt256(value)
		}

	// Word*
	case sema.Word8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord8(value)
		}

	case sema.Word16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord16(value)
		}

	case sema.Word32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord32(value)
		}

	case sema.Word64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord64(value)
		}

	// Fix*

	case sema.Fix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertFix64(value)
		}

	case sema.UFix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUFix64(value)
		}
	}

	switch unwrappedTargetType.(type) {
	case *sema.AddressType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertAddress(value)
		}
	}

	return value
}

// boxOptional boxes a value in optionals, if necessary
func (interpreter *Interpreter) boxOptional(value Value, valueType, targetType sema.Type) Value {
	inner := value
	for {
		optionalType, ok := targetType.(*sema.OptionalType)
		if !ok {
			break
		}

		switch typedInner := inner.(type) {
		case *SomeValue:
			inner = typedInner.Value

		case NilValue:
			// NOTE: nested nil will be unboxed!
			return inner

		default:
			value = NewSomeValueOwningNonCopying(value)
			valueType = &sema.OptionalType{
				Type: valueType,
			}
		}

		targetType = optionalType.Type
	}
	return value
}

func (interpreter *Interpreter) unbox(value Value) Value {
	for {
		some, ok := value.(*SomeValue)
		if !ok {
			return value
		}

		value = some.Value
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
		return NewHostFunctionValue(func(invocation Invocation) Value {
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
				// NOTE: It is important to wrap the invocation in a trampoline,
				//  so the inner function isn't invoked here

				body = func() controlReturn {

					// NOTE: It is important to actually return the value returned
					//   from the inner function, otherwise it is lost

					returnValue := inner.Invoke(invocation)
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
		})
	}
}

func (interpreter *Interpreter) ensureLoaded(
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
			subInterpreter.Globals[global.Name] = variable
		}

		subInterpreter.typeCodes.
			Merge(virtualImport.TypeCodes)

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
		WithPredeclaredValues(interpreter.PredeclaredValues),
		WithOnEventEmittedHandler(interpreter.onEventEmitted),
		WithOnStatementHandler(interpreter.onStatement),
		WithOnLoopIterationHandler(interpreter.onLoopIteration),
		WithOnFunctionInvocationHandler(interpreter.onFunctionInvocation),
		WithStorageExistenceHandler(interpreter.storageExistenceHandler),
		WithStorageReadHandler(interpreter.storageReadHandler),
		WithStorageWriteHandler(interpreter.storageWriteHandler),
		WithInjectedCompositeFieldsHandler(interpreter.injectedCompositeFieldsHandler),
		WithContractValueHandler(interpreter.contractValueHandler),
		WithImportLocationHandler(interpreter.importLocationHandler),
		WithUUIDHandler(interpreter.uuidHandler),
		WithAllInterpreters(interpreter.allInterpreters),
		withTypeCodes(interpreter.typeCodes),
		WithAccountHandlerFunc(interpreter.accountHandler),
		WithPublicKeyValidationHandler(interpreter.PublicKeyValidationHandler),
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

func (interpreter *Interpreter) storedValueExists(storageAddress common.Address, key string) bool {
	return interpreter.storageExistenceHandler(interpreter, storageAddress, key)
}

func (interpreter *Interpreter) ReadStored(storageAddress common.Address, key string, deferred bool) OptionalValue {
	return interpreter.storageReadHandler(interpreter, storageAddress, key, deferred)
}

func (interpreter *Interpreter) writeStored(storageAddress common.Address, key string, value OptionalValue) {
	value.SetOwner(&storageAddress)

	interpreter.storageWriteHandler(interpreter, storageAddress, key, value)
}

type valueConverterDeclaration struct {
	name    string
	convert func(Value) Value
	min     Value
	max     Value
}

// It would be nice if return types in Go's function types would be covariant
//
var converterDeclarations = []valueConverterDeclaration{
	{
		name: sema.IntTypeName,
		convert: func(value Value) Value {
			return ConvertInt(value)
		},
	},
	{
		name: sema.UIntTypeName,
		convert: func(value Value) Value {
			return ConvertUInt(value)
		},
		min: NewUIntValueFromBigInt(sema.UIntTypeMin),
	},
	{
		name: sema.Int8TypeName,
		convert: func(value Value) Value {
			return ConvertInt8(value)
		},
		min: Int8Value(math.MinInt8),
		max: Int8Value(math.MaxInt8),
	},
	{
		name: sema.Int16TypeName,
		convert: func(value Value) Value {
			return ConvertInt16(value)
		},
		min: Int16Value(math.MinInt16),
		max: Int16Value(math.MaxInt16),
	},
	{
		name: sema.Int32TypeName,
		convert: func(value Value) Value {
			return ConvertInt32(value)
		},
		min: Int32Value(math.MinInt32),
		max: Int32Value(math.MaxInt32),
	},
	{
		name: sema.Int64TypeName,
		convert: func(value Value) Value {
			return ConvertInt64(value)
		},
		min: Int64Value(math.MinInt64),
		max: Int64Value(math.MaxInt64),
	},
	{
		name: sema.Int128TypeName,
		convert: func(value Value) Value {
			return ConvertInt128(value)
		},
		min: NewInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
		max: NewInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
	},
	{
		name: sema.Int256TypeName,
		convert: func(value Value) Value {
			return ConvertInt256(value)
		},
		min: NewInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
		max: NewInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
	},
	{
		name: sema.UInt8TypeName,
		convert: func(value Value) Value {
			return ConvertUInt8(value)
		},
		min: UInt8Value(0),
		max: UInt8Value(math.MaxUint8),
	},
	{
		name: sema.UInt16TypeName,
		convert: func(value Value) Value {
			return ConvertUInt16(value)
		},
		min: UInt16Value(0),
		max: UInt16Value(math.MaxUint16),
	},
	{
		name: sema.UInt32TypeName,
		convert: func(value Value) Value {
			return ConvertUInt32(value)
		},
		min: UInt32Value(0),
		max: UInt32Value(math.MaxUint32),
	},
	{
		name: sema.UInt64TypeName,
		convert: func(value Value) Value {
			return ConvertUInt64(value)
		},
		min: UInt64Value(0),
		max: UInt64Value(math.MaxUint64),
	},
	{
		name: sema.UInt128TypeName,
		convert: func(value Value) Value {
			return ConvertUInt128(value)
		},
		min: NewUInt128ValueFromUint64(0),
		max: NewUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
	},
	{
		name: sema.UInt256TypeName,
		convert: func(value Value) Value {
			return ConvertUInt256(value)
		},
		min: NewUInt256ValueFromUint64(0),
		max: NewUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
	},
	{
		name: sema.Word8TypeName,
		convert: func(value Value) Value {
			return ConvertWord8(value)
		},
		min: Word8Value(0),
		max: Word8Value(math.MaxUint8),
	},
	{
		name: sema.Word16TypeName,
		convert: func(value Value) Value {
			return ConvertWord16(value)
		},
		min: Word16Value(0),
		max: Word16Value(math.MaxUint16),
	},
	{
		name: sema.Word32TypeName,
		convert: func(value Value) Value {
			return ConvertWord32(value)
		},
		min: Word32Value(0),
		max: Word32Value(math.MaxUint32),
	},
	{
		name: sema.Word64TypeName,
		convert: func(value Value) Value {
			return ConvertWord64(value)
		},
		min: Word64Value(0),
		max: Word64Value(math.MaxUint64),
	},
	{
		name: sema.Fix64TypeName,
		convert: func(value Value) Value {
			return ConvertFix64(value)
		},
		min: Fix64Value(math.MinInt64),
		max: Fix64Value(math.MaxInt64),
	},
	{
		name: sema.UFix64TypeName,
		convert: func(value Value) Value {
			return ConvertUFix64(value)
		},
		min: UFix64Value(0),
		max: UFix64Value(math.MaxUint64),
	},
	{
		name: "Address",
		convert: func(value Value) Value {
			return ConvertAddress(value)
		},
	},
}

func init() {

	converterNames := make(map[string]struct{}, len(converterDeclarations))

	for _, converterDeclaration := range converterDeclarations {
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
}

func (interpreter *Interpreter) defineBaseFunctions() {
	interpreter.defineConverterFunctions()
	interpreter.defineTypeFunction()
}

func (interpreter *Interpreter) defineConverterFunctions() {
	for _, declaration := range converterDeclarations {
		// NOTE: declare in loop, as captured in closure below
		convert := declaration.convert
		converterFunctionValue := NewHostFunctionValue(
			func(invocation Invocation) Value {
				return convert(invocation.Arguments[0])
			},
		)

		addMember := func(name string, value Value) {
			if converterFunctionValue.NestedVariables == nil {
				converterFunctionValue.NestedVariables = NewStringVariableOrderedMap()
			}
			converterFunctionValue.NestedVariables.Set(name, NewVariableWithValue(value))
		}

		if declaration.min != nil {
			addMember(sema.NumberTypeMinFieldName, declaration.min)
		}

		if declaration.max != nil {
			addMember(sema.NumberTypeMaxFieldName, declaration.max)
		}

		err := interpreter.ImportValue(declaration.name, converterFunctionValue)
		if err != nil {
			panic(errors.NewUnreachableError())
		}
	}
}

func (interpreter *Interpreter) defineTypeFunction() {
	err := interpreter.ImportValue(
		"Type",
		NewHostFunctionValue(
			func(invocation Invocation) Value {

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair == nil {
					panic(errors.NewUnreachableError())
				}

				ty := typeParameterPair.Value

				return TypeValue{
					Type: ConvertSemaToStaticType(ty),
				}
			},
		),
	)
	if err != nil {
		panic(errors.NewUnreachableError())
	}
}

// TODO:
// - FunctionType
//
// - Character
// - Block

func IsSubType(subType DynamicType, superType sema.Type) bool {
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

	case StringDynamicType:
		switch superType {
		case sema.AnyStructType, sema.StringType, sema.CharacterType:
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
		return superType == sema.AnyStructType

	case CompositeDynamicType:
		return sema.IsSubType(typedSubType.StaticType, superType)

	case ArrayDynamicType:
		var superTypeElementType sema.Type

		switch typedSuperType := superType.(type) {
		case *sema.VariableSizedType:
			superTypeElementType = typedSuperType.Type

		case *sema.ConstantSizedType:
			superTypeElementType = typedSuperType.Type

		default:
			switch superType {
			case sema.AnyStructType, sema.AnyResourceType:
				return true
			default:
				return false
			}
		}

		for _, elementType := range typedSubType.ElementTypes {
			if !IsSubType(elementType, superTypeElementType) {
				return false
			}
		}

		return true

	case DictionaryDynamicType:

		if typedSuperType, ok := superType.(*sema.DictionaryType); ok {
			for _, entryTypes := range typedSubType.EntryTypes {
				if !IsSubType(entryTypes.KeyType, typedSuperType.KeyType) ||
					!IsSubType(entryTypes.ValueType, typedSuperType.ValueType) {

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
			return IsSubType(typedSubType.InnerType, typedSuperType.Type)
		}

		switch superType {
		case sema.AnyStructType, sema.AnyResourceType:
			return true
		}

	case ReferenceDynamicType:
		if typedSuperType, ok := superType.(*sema.ReferenceType); ok {
			if typedSubType.Authorized() {
				return IsSubType(typedSubType.InnerType(), typedSuperType.Type)
			} else {
				// NOTE: Allowing all casts for casting unauthorized references is intentional:
				// all invalid cases have already been rejected statically
				return true
			}
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

// StorageKey returns the storage identifier with the proper prefix
// for the given path.
//
// \x1F = Information Separator One
//
func StorageKey(path PathValue) string {
	return fmt.Sprintf("%s\x1F%s", path.Domain.Identifier(), path.Identifier)
}

func (interpreter *Interpreter) authAccountSaveFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Value {

		value := invocation.Arguments[0]
		path := invocation.Arguments[1].(PathValue)

		address := addressValue.ToAddress()
		key := StorageKey(path)

		// Prevent an overwrite

		if interpreter.storedValueExists(address, key) {
			panic(
				OverwriteError{
					Address:       addressValue,
					Path:          path,
					LocationRange: invocation.GetLocationRange(),
				},
			)
		}

		// Write new value

		interpreter.writeStored(
			address,
			key,
			NewSomeValueOwningNonCopying(value),
		)

		return VoidValue{}
	})
}

func (interpreter *Interpreter) authAccountLoadFunction(addressValue AddressValue) HostFunctionValue {
	return interpreter.authAccountReadFunction(addressValue, true)
}

func (interpreter *Interpreter) authAccountCopyFunction(addressValue AddressValue) HostFunctionValue {
	return interpreter.authAccountReadFunction(addressValue, false)
}

func (interpreter *Interpreter) authAccountReadFunction(addressValue AddressValue, clear bool) HostFunctionValue {

	return NewHostFunctionValue(func(invocation Invocation) Value {

		address := addressValue.ToAddress()

		path := invocation.Arguments[0].(PathValue)
		key := StorageKey(path)

		value := interpreter.ReadStored(address, key, false)

		switch value := value.(type) {
		case NilValue:
			return value

		case *SomeValue:

			// If there is value stored for the given path,
			// check that it satisfies the type given as the type argument.

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := typeParameterPair.Value

			dynamicTypeResults := DynamicTypeResults{}

			dynamicType := value.Value.DynamicType(interpreter, dynamicTypeResults)
			if !IsSubType(dynamicType, ty) {
				return NilValue{}
			}

			if clear {
				// Remove the value from storage,
				// but only if the type check succeeded.

				interpreter.writeStored(address, key, NilValue{})
			}

			return value

		default:
			panic(errors.NewUnreachableError())
		}
	})
}

func (interpreter *Interpreter) authAccountBorrowFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Value {

		address := addressValue.ToAddress()

		path := invocation.Arguments[0].(PathValue)
		key := StorageKey(path)

		typeParameterPair := invocation.TypeParameterTypes.Oldest()
		if typeParameterPair == nil {
			panic(errors.NewUnreachableError())
		}

		ty := typeParameterPair.Value

		referenceType := ty.(*sema.ReferenceType)

		reference := &StorageReferenceValue{
			Authorized:           referenceType.Authorized,
			TargetStorageAddress: address,
			TargetKey:            key,
			BorrowedType:         referenceType.Type,
		}

		// Attempt to dereference,
		// which reads the stored value
		// and performs a dynamic type check

		if reference.ReferencedValue(interpreter) == nil {
			return NilValue{}
		}

		return NewSomeValueOwningNonCopying(reference)
	})
}

func (interpreter *Interpreter) authAccountLinkFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Value {

		address := addressValue.ToAddress()

		typeParameterPair := invocation.TypeParameterTypes.Oldest()
		if typeParameterPair == nil {
			panic(errors.NewUnreachableError())
		}

		borrowType := typeParameterPair.Value.(*sema.ReferenceType)

		newCapabilityPath := invocation.Arguments[0].(PathValue)
		targetPath := invocation.Arguments[1].(PathValue)

		newCapabilityKey := StorageKey(newCapabilityPath)

		if interpreter.storedValueExists(address, newCapabilityKey) {
			return NilValue{}
		}

		// Write new value

		borrowStaticType := ConvertSemaToStaticType

		storedValue := NewSomeValueOwningNonCopying(
			LinkValue{
				TargetPath: targetPath,
				Type:       borrowStaticType(borrowType),
			},
		)

		interpreter.writeStored(
			address,
			newCapabilityKey,
			storedValue,
		)

		return NewSomeValueOwningNonCopying(
			CapabilityValue{
				Address:    addressValue,
				Path:       targetPath,
				BorrowType: borrowStaticType(borrowType),
			},
		)
	})
}

func (interpreter *Interpreter) accountGetLinkTargetFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Value {

		address := addressValue.ToAddress()

		capabilityPath := invocation.Arguments[0].(PathValue)

		capabilityKey := StorageKey(capabilityPath)

		value := interpreter.ReadStored(address, capabilityKey, false)

		switch value := value.(type) {
		case NilValue:
			return value

		case *SomeValue:

			link, ok := value.Value.(LinkValue)
			if !ok {
				return NilValue{}
			}

			return NewSomeValueOwningNonCopying(link.TargetPath)

		default:
			panic(errors.NewUnreachableError())
		}
	})
}

func (interpreter *Interpreter) authAccountUnlinkFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Value {

		address := addressValue.ToAddress()

		capabilityPath := invocation.Arguments[0].(PathValue)
		capabilityKey := StorageKey(capabilityPath)

		// Write new value

		interpreter.writeStored(
			address,
			capabilityKey,
			NilValue{},
		)

		return VoidValue{}
	})
}

func (interpreter *Interpreter) capabilityBorrowFunction(
	addressValue AddressValue,
	pathValue PathValue,
	borrowType *sema.ReferenceType,
) HostFunctionValue {

	return NewHostFunctionValue(
		func(invocation Invocation) Value {

			if borrowType == nil {
				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair != nil {
					ty := typeParameterPair.Value
					borrowType = ty.(*sema.ReferenceType)
				}
			}

			if borrowType == nil {
				panic(errors.NewUnreachableError())
			}

			address := addressValue.ToAddress()

			targetStorageKey, authorized, err :=
				interpreter.GetCapabilityFinalTargetStorageKey(
					address,
					pathValue,
					borrowType,
					invocation.GetLocationRange,
				)
			if err != nil {
				panic(err)
			}

			if targetStorageKey == "" {
				return NilValue{}
			}

			reference := &StorageReferenceValue{
				Authorized:           authorized,
				TargetStorageAddress: address,
				TargetKey:            targetStorageKey,
				BorrowedType:         borrowType.Type,
			}

			// Attempt to dereference,
			// which reads the stored value
			// and performs a dynamic type check

			if reference.ReferencedValue(interpreter) == nil {
				return NilValue{}
			}

			return NewSomeValueOwningNonCopying(reference)
		},
	)
}

func (interpreter *Interpreter) capabilityCheckFunction(
	addressValue AddressValue,
	pathValue PathValue,
	borrowType *sema.ReferenceType,
) HostFunctionValue {

	return NewHostFunctionValue(
		func(invocation Invocation) Value {

			if borrowType == nil {

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair != nil {
					ty := typeParameterPair.Value
					borrowType = ty.(*sema.ReferenceType)
				}
			}

			if borrowType == nil {
				panic(errors.NewUnreachableError())
			}

			address := addressValue.ToAddress()

			targetStorageKey, authorized, err :=
				interpreter.GetCapabilityFinalTargetStorageKey(
					address,
					pathValue,
					borrowType,
					invocation.GetLocationRange,
				)
			if err != nil {
				panic(err)
			}

			if targetStorageKey == "" {
				return BoolValue(false)
			}

			reference := &StorageReferenceValue{
				Authorized:           authorized,
				TargetStorageAddress: address,
				TargetKey:            targetStorageKey,
				BorrowedType:         borrowType.Type,
			}

			// Attempt to dereference,
			// which reads the stored value
			// and performs a dynamic type check

			if reference.ReferencedValue(interpreter) == nil {
				return BoolValue(false)
			}

			return BoolValue(true)
		},
	)
}

func (interpreter *Interpreter) GetCapabilityFinalTargetStorageKey(
	address common.Address,
	path PathValue,
	wantedBorrowType *sema.ReferenceType,
	getLocationRange func() LocationRange,
) (
	finalStorageKey string,
	authorized bool,
	err error,
) {
	key := StorageKey(path)

	wantedReferenceType := wantedBorrowType

	seenKeys := map[string]struct{}{}
	paths := []PathValue{path}

	for {
		// Detect cyclic links

		if _, ok := seenKeys[key]; ok {
			return "", false, CyclicLinkError{
				Address:       address,
				Paths:         paths,
				LocationRange: getLocationRange(),
			}
		} else {
			seenKeys[key] = struct{}{}
		}

		value := interpreter.ReadStored(address, key, false)

		switch value := value.(type) {
		case NilValue:
			return "", false, nil

		case *SomeValue:

			if link, ok := value.Value.(LinkValue); ok {

				allowedType := interpreter.ConvertStaticToSemaType(link.Type)

				if !sema.IsSubType(allowedType, wantedBorrowType) {
					return "", false, nil
				}

				targetPath := link.TargetPath
				paths = append(paths, targetPath)
				key = StorageKey(targetPath)

			} else {
				return key, wantedReferenceType.Authorized, nil
			}

		default:
			panic(errors.NewUnreachableError())
		}
	}
}

func (interpreter *Interpreter) ConvertStaticToSemaType(staticType StaticType) sema.Type {
	return ConvertStaticToSemaType(
		staticType,
		func(location common.Location, qualifiedIdentifier string) *sema.InterfaceType {
			return interpreter.getInterfaceType(location, qualifiedIdentifier)
		},
		func(location common.Location, qualifiedIdentifier string) *sema.CompositeType {
			return interpreter.getCompositeType(location, qualifiedIdentifier)
		},
	)
}

func (interpreter *Interpreter) getElaboration(location common.Location) *sema.Elaboration {

	// Ensure the program for this location is loaded,
	// so its checker is available

	inter := interpreter.ensureLoaded(
		location,
		func() Import {
			return interpreter.importLocationHandler(interpreter, location)
		},
	)

	locationID := location.ID()

	subInterpreter := inter.allInterpreters[locationID]
	if subInterpreter == nil || subInterpreter.Program == nil {
		return nil
	}

	return subInterpreter.Program.Elaboration
}

func (interpreter *Interpreter) getCompositeType(location common.Location, qualifiedIdentifier string) *sema.CompositeType {
	if location == nil {
		ty := sema.NativeCompositeTypes[qualifiedIdentifier]
		if ty == nil {
			panic(TypeLoadingError{
				TypeID: common.TypeID(qualifiedIdentifier),
			})
		}

		return ty
	}

	typeID := location.TypeID(qualifiedIdentifier)

	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		panic(TypeLoadingError{
			TypeID: typeID,
		})
	}

	ty := elaboration.CompositeTypes[typeID]
	if ty == nil {
		panic(TypeLoadingError{
			TypeID: typeID,
		})
	}

	return ty
}

func (interpreter *Interpreter) getInterfaceType(location common.Location, qualifiedIdentifier string) *sema.InterfaceType {
	typeID := location.TypeID(qualifiedIdentifier)

	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		panic(TypeLoadingError{
			TypeID: typeID,
		})
	}

	ty := elaboration.InterfaceTypes[typeID]
	if ty == nil {
		panic(TypeLoadingError{
			TypeID: typeID,
		})
	}
	return ty
}

func (interpreter *Interpreter) reportLoopIteration(pos ast.HasPosition) {
	if interpreter.onLoopIteration == nil {
		return
	}

	line := pos.StartPosition().Line
	interpreter.onLoopIteration(interpreter, line)
}

func (interpreter *Interpreter) reportFunctionInvocation(pos ast.HasPosition) {
	if interpreter.onFunctionInvocation == nil {
		return
	}

	line := pos.StartPosition().Line
	interpreter.onFunctionInvocation(interpreter, line)
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

func (interpreter *Interpreter) isInstanceFunction(self Value) HostFunctionValue {
	return NewHostFunctionValue(
		func(invocation Invocation) Value {
			firstArgument := invocation.Arguments[0]
			typeValue := firstArgument.(TypeValue)

			staticType := typeValue.Type

			// Values are never instances of unknown types
			if staticType == nil {
				return BoolValue(false)
			}

			semaType := interpreter.ConvertStaticToSemaType(staticType)
			// NOTE: not invocation.Self, as that is only set for composite values
			dynamicTypeResults := DynamicTypeResults{}
			dynamicType := self.DynamicType(interpreter, dynamicTypeResults)
			result := IsSubType(dynamicType, semaType)
			return BoolValue(result)
		},
	)
}

func (interpreter *Interpreter) getTypeFunction(self Value) HostFunctionValue {
	return NewHostFunctionValue(
		func(invocation Invocation) Value {
			return TypeValue{
				Type: self.StaticType(),
			}
		},
	)
}

func (interpreter *Interpreter) setMember(self Value, getLocationRange func() LocationRange, identifier string, value Value) {
	self.(MemberAccessibleValue).SetMember(interpreter, getLocationRange, identifier, value)
}
