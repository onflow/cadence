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
	goRuntime "runtime"
	"sort"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"

	. "github.com/onflow/cadence/runtime/trampoline"
)

type controlReturn interface {
	isControlReturn()
}

type controlBreak struct{}

func (controlBreak) isControlReturn() {}

type controlContinue struct{}

func (controlContinue) isControlReturn() {}

type functionReturn struct {
	Value
}

func (functionReturn) isControlReturn() {}

type ExpressionStatementResult struct {
	Value
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
	statement *Statement,
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

// StorageKeyHandlerFunc is a function that handles storage indexing types.
//
type StorageKeyHandlerFunc func(
	inter *Interpreter,
	storageAddress common.Address,
	indexingType sema.Type,
) string

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

// UUIDHandlerFunc is a function that handles the generation of UUIDs.
type UUIDHandlerFunc func() (uint64, error)

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
	activations                    *activations.Activations
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
	storageKeyHandler              StorageKeyHandlerFunc
	injectedCompositeFieldsHandler InjectedCompositeFieldsHandlerFunc
	contractValueHandler           ContractValueHandlerFunc
	importLocationHandler          ImportLocationHandlerFunc
	uuidHandler                    UUIDHandlerFunc
	interpreted                    bool
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

// WithStorageKeyHandler returns an interpreter option which sets the given function
// as the function that is used when a stored value is written.
//
func WithStorageKeyHandler(handler StorageKeyHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetStorageKeyHandler(handler)
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

// WithUUIDHandler returns an interpreter option which sets the given function
// as the function that is used to generate UUIDs.
//
func WithUUIDHandler(handler UUIDHandlerFunc) Option {
	return func(interpreter *Interpreter) error {
		interpreter.SetUUIDHandler(handler)
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
		activations:                &activations.Activations{},
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

// SetStorageKeyHandler sets the function that is used when a storage is indexed.
//
func (interpreter *Interpreter) SetStorageKeyHandler(function StorageKeyHandlerFunc) {
	interpreter.storageKeyHandler = function
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

// SetUUIDHandler sets the function that is used to handle the generation of UUIDs.
//
func (interpreter *Interpreter) SetUUIDHandler(function UUIDHandlerFunc) {
	interpreter.uuidHandler = function
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

// locationRange returns a new location range for the given positioned element.
//
func (interpreter *Interpreter) locationRange(hasPosition ast.HasPosition) LocationRange {
	return LocationRange{
		Location: interpreter.Location,
		Range:    ast.NewRangeFromPositioned(hasPosition),
	}
}

func (interpreter *Interpreter) findVariable(name string) *Variable {
	result := interpreter.activations.Find(name)
	if result == nil {
		return nil
	}
	return result.(*Variable)
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
	defer recoverErrors(func(internalErr error) {
		err = internalErr
	})

	interpreter.runAllStatements(interpreter.interpret())

	interpreter.interpreted = true

	return nil
}

type Statement struct {
	Interpreter *Interpreter
	Trampoline  Trampoline
	Statement   ast.Statement
}

// runUntilNextStatement executes the trampline until the next statement.
// It either returns a result or a statement.
// The difference between "runUntilNextStatement" and "Run" is that:
// "Run" executes the Trampoline chain all the way until there is no more trampoline and returns the result,
// whereas "runUntilNextStatement" executes the Trampoline chain, stops as soon as it meets a statement trampoline,
// and returns the statement, which can be later resumed by calling "runUntilNextStatement" again.
// Useful for implementing breakpoint debugging.
func (interpreter *Interpreter) runUntilNextStatement(t Trampoline) (interface{}, *Statement) {
	for {
		statement := getStatement(t)

		if statement != nil {
			return nil, &Statement{
				// NOTE: resumption using outer trampoline,
				// not just inner statement trampoline
				Trampoline:  t,
				Interpreter: statement.Interpreter,
				Statement:   statement.Statement,
			}
		}

		result := t.Resume()

		if continuation, ok := result.(func() Trampoline); ok {

			t = continuation()
			continue
		}

		return result, nil
	}
}

// runAllStatements runs all the statement until there is no more trampoline and returns the result.
// When there is a statement, it calls the onStatement callback, and then continues the execution.
func (interpreter *Interpreter) runAllStatements(t Trampoline) interface{} {

	var statement *Statement

	// Wrap errors if needed

	defer recoverErrors(func(internalErr error) {

		// if the error is already an execution error, use it as is
		if _, ok := internalErr.(Error); ok {
			panic(internalErr)
		}

		// wrap the error with position information if needed

		_, ok := internalErr.(ast.HasPosition)
		if !ok {
			internalErr = PositionedError{
				Err:   internalErr,
				Range: ast.NewRangeFromPositioned(statement.Statement),
			}
		}

		panic(Error{
			Err:      internalErr,
			Location: statement.Interpreter.Location,
		})
	})

	for {
		var result interface{}
		result, statement = interpreter.runUntilNextStatement(t)
		if statement == nil {
			return result
		}

		if interpreter.onStatement != nil {
			interpreter.onStatement(statement)
		}

		result = statement.Trampoline.Resume()
		if continuation, ok := result.(func() Trampoline); ok {
			t = continuation()
			continue
		}

		return result
	}
}

// getStatement goes through the Trampoline chain and find the first StatementTrampoline
func getStatement(t Trampoline) *StatementTrampoline {
	switch t := t.(type) {
	case FlatMap:
		// Recurse into the nested trampoline
		return getStatement(t.Subroutine)
	case StatementTrampoline:
		return &t
	default:
		return nil
	}
}

// interpret returns a Trampoline that is done when all top-level declarations
// have been declared and evaluated.
func (interpreter *Interpreter) interpret() Trampoline {
	return interpreter.Program.Program.Accept(interpreter).(Trampoline)
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

func (interpreter *Interpreter) visitGlobalDeclarations(declarations []ast.Declaration) Trampoline {
	count := len(declarations)

	// no declarations? stop
	if count == 0 {
		// NOTE: no result, so it does *not* act like a return-statement
		return Done{}
	}

	// interpret the first declaration, then the remaining ones
	return interpreter.visitGlobalDeclaration(declarations[0]).
		FlatMap(func(_ interface{}) Trampoline {
			return interpreter.visitGlobalDeclarations(declarations[1:])
		})
}

// visitGlobalDeclaration firsts interprets the global declaration,
// then finds the declaration and adds it to the globals
func (interpreter *Interpreter) visitGlobalDeclaration(declaration ast.Declaration) Trampoline {
	return declaration.Accept(interpreter).(Trampoline).
		Then(func(_ interface{}) {
			interpreter.declareGlobal(declaration)
		})
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

// prepareInvokeVariable looks up the function by the given name from global
// variables, checks the function type, and returns a trampoline which executes
// the function with the given arguments
func (interpreter *Interpreter) prepareInvokeVariable(
	functionName string,
	arguments []Value,
) (trampoline Trampoline, err error) {

	// function must be defined as a global variable
	variable, ok := interpreter.Globals[functionName]
	if !ok {
		return nil, NotDeclaredError{
			ExpectedKind: common.DeclarationKindFunction,
			Name:         functionName,
		}
	}

	variableValue := variable.Value

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
) (trampoline Trampoline, err error) {
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
) (trampoline Trampoline, err error) {

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
	trampoline = functionValue.Invoke(Invocation{
		Arguments:   preparedArguments,
		Interpreter: interpreter,
	})

	return trampoline, nil
}

// Invoke invokes a global function with the given arguments
func (interpreter *Interpreter) Invoke(functionName string, arguments ...Value) (value Value, err error) {
	// recover internal panics and return them as an error
	defer recoverErrors(func(internalErr error) {
		err = internalErr
	})

	trampoline, err := interpreter.prepareInvokeVariable(functionName, arguments)
	if err != nil {
		return nil, err
	}
	result := interpreter.runAllStatements(trampoline)
	if result == nil {
		return nil, nil
	}
	return result.(Value), nil
}

func (interpreter *Interpreter) InvokeTransaction(index int, arguments ...Value) (err error) {
	// recover internal panics and return them as an error
	defer recoverErrors(func(internalErr error) {
		err = internalErr
	})

	trampoline, err := interpreter.prepareInvokeTransaction(index, arguments)
	if err != nil {
		return err
	}

	_ = interpreter.runAllStatements(trampoline)

	return nil
}

func recoverErrors(onError func(error)) {
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

		onError(err)
	}
}

func (interpreter *Interpreter) VisitProgram(program *ast.Program) ast.Repr {
	interpreter.prepareInterpretation()

	return interpreter.visitGlobalDeclarations(program.Declarations())
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

	variable.Value = interpreter.functionDeclarationValue(
		declaration,
		functionType,
		lexicalScope,
	)

	// NOTE: no result, so it does *not* act like a return-statement
	return Done{}
}

func (interpreter *Interpreter) functionDeclarationValue(
	declaration *ast.FunctionDeclaration,
	functionType *sema.FunctionType,
	lexicalScope *activations.Activation,
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

	return interpreter.visitStatements(block.Statements).
		Then(func(_ interface{}) {
			interpreter.activations.Pop()
		})
}

func (interpreter *Interpreter) visitStatements(statements []ast.Statement) Trampoline {
	count := len(statements)

	// no statements? stop
	if count == 0 {
		// NOTE: no result, so it does *not* act like a return-statement
		return Done{}
	}

	statement := statements[0]

	// interpret the first statement, then the remaining ones
	return StatementTrampoline{
		F: func() Trampoline {
			return statement.Accept(interpreter).(Trampoline)
		},
		Interpreter: interpreter,
		Statement:   statement,
	}.FlatMap(func(returnValue interface{}) Trampoline {
		if _, isReturn := returnValue.(controlReturn); isReturn {
			return Done{Result: returnValue}
		}
		return interpreter.visitStatements(statements[1:])
	})
}

func (interpreter *Interpreter) VisitFunctionBlock(_ *ast.FunctionBlock) ast.Repr {
	// NOTE: see visitBlock
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) visitFunctionBody(
	beforeStatements []ast.Statement,
	preConditions ast.Conditions,
	body Trampoline,
	postConditions ast.Conditions,
	returnType sema.Type,
) Trampoline {

	// block scope: each function block gets an activation record
	interpreter.activations.PushNewWithCurrent()

	return interpreter.visitStatements(beforeStatements).
		FlatMap(func(_ interface{}) Trampoline {
			return interpreter.visitConditions(preConditions)
		}).
		FlatMap(func(_ interface{}) Trampoline {
			return body
		}).
		FlatMap(func(blockResult interface{}) Trampoline {
			var resultValue Value
			if _, ok := blockResult.(functionReturn); ok {
				resultValue = blockResult.(functionReturn).Value
			} else {
				resultValue = VoidValue{}
			}

			// If there is a return type, declare the constant `result`
			// which has the return value

			if returnType != sema.VoidType {
				interpreter.declareVariable(sema.ResultIdentifier, resultValue)
			}

			return interpreter.visitConditions(postConditions).
				Map(func(_ interface{}) interface{} {
					return resultValue
				})
		}).
		Then(func(_ interface{}) {
			interpreter.activations.Pop()
		})
}

func (interpreter *Interpreter) visitConditions(conditions []*ast.Condition) Trampoline {
	count := len(conditions)

	// no conditions? stop
	if count == 0 {
		return Done{}
	}

	// interpret the first condition, then the remaining ones.
	// treat the condition as a statement, so we get position information in case of an error
	condition := conditions[0]
	return StatementTrampoline{
		F: func() Trampoline {
			return condition.Accept(interpreter).(Trampoline)
		},
		Interpreter: interpreter,
		Statement: &ast.ExpressionStatement{
			Expression: condition.Test,
		},
	}.FlatMap(func(value interface{}) Trampoline {
		result := value.(BoolValue)

		if !result {

			var messageTrampoline Trampoline

			if condition.Message == nil {
				messageTrampoline = Done{Result: NewStringValue("")}
			} else {
				messageTrampoline = condition.Message.Accept(interpreter).(Trampoline)
			}

			return messageTrampoline.
				Then(func(result interface{}) {
					message := result.(*StringValue).Str

					panic(ConditionError{
						ConditionKind: condition.Kind,
						Message:       message,
						LocationRange: interpreter.locationRange(condition.Test),
					})
				})
		}

		return interpreter.visitConditions(conditions[1:])
	})
}

func (interpreter *Interpreter) VisitCondition(condition *ast.Condition) ast.Repr {
	return condition.Test.Accept(interpreter)
}

func (interpreter *Interpreter) VisitReturnStatement(statement *ast.ReturnStatement) ast.Repr {
	// NOTE: returning result

	if statement.Expression == nil {
		return Done{Result: functionReturn{VoidValue{}}}
	}

	return statement.Expression.Accept(interpreter).(Trampoline).
		Map(func(result interface{}) interface{} {
			value := result.(Value)

			valueType := interpreter.Program.Elaboration.ReturnStatementValueTypes[statement]
			returnType := interpreter.Program.Elaboration.ReturnStatementReturnTypes[statement]

			// NOTE: copy on return
			value = interpreter.copyAndConvert(value, valueType, returnType)

			return functionReturn{value}
		})
}

func (interpreter *Interpreter) VisitBreakStatement(_ *ast.BreakStatement) ast.Repr {
	return Done{Result: controlBreak{}}
}

func (interpreter *Interpreter) VisitContinueStatement(_ *ast.ContinueStatement) ast.Repr {
	return Done{Result: controlContinue{}}
}

func (interpreter *Interpreter) VisitIfStatement(statement *ast.IfStatement) ast.Repr {
	switch test := statement.Test.(type) {
	case ast.Expression:
		return interpreter.visitIfStatementWithTestExpression(test, statement.Then, statement.Else)
	case *ast.VariableDeclaration:
		return interpreter.visitIfStatementWithVariableDeclaration(test, statement.Then, statement.Else)
	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) visitIfStatementWithTestExpression(
	test ast.Expression,
	thenBlock, elseBlock *ast.Block,
) Trampoline {

	return test.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			value := result.(BoolValue)
			if value {
				return thenBlock.Accept(interpreter).(Trampoline)
			} else if elseBlock != nil {
				return elseBlock.Accept(interpreter).(Trampoline)
			}

			// NOTE: no result, so it does *not* act like a return-statement
			return Done{}
		})
}

func (interpreter *Interpreter) visitIfStatementWithVariableDeclaration(
	declaration *ast.VariableDeclaration,
	thenBlock, elseBlock *ast.Block,
) Trampoline {

	return declaration.Value.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {

			if someValue, ok := result.(*SomeValue); ok {

				targetType := interpreter.Program.Elaboration.VariableDeclarationTargetTypes[declaration]
				valueType := interpreter.Program.Elaboration.VariableDeclarationValueTypes[declaration]
				unwrappedValueCopy := interpreter.copyAndConvert(someValue.Value, valueType, targetType)

				interpreter.activations.PushNewWithCurrent()
				interpreter.declareVariable(
					declaration.Identifier.Identifier,
					unwrappedValueCopy,
				)

				return thenBlock.Accept(interpreter).(Trampoline).
					Then(func(_ interface{}) {
						interpreter.activations.Pop()
					})
			} else if elseBlock != nil {
				return elseBlock.Accept(interpreter).(Trampoline)
			}

			// NOTE: ignore result, so it does *not* act like a return-statement
			return Done{}
		})
}

func (interpreter *Interpreter) VisitSwitchStatement(switchStatement *ast.SwitchStatement) ast.Repr {

	var visitCase func(i int, testValue EquatableValue) Trampoline
	visitCase = func(i int, testValue EquatableValue) Trampoline {

		// If no cases are left to evaluate, return (base case)

		if i >= len(switchStatement.Cases) {
			// NOTE: no result, so it does *not* act like a return-statement
			return Done{}
		}

		switchCase := switchStatement.Cases[i]

		runStatements := func() Trampoline {
			// NOTE: the new block ensures that a new block is introduced

			block := &ast.Block{
				Statements: switchCase.Statements,
			}

			return block.Accept(interpreter).(Trampoline).
				FlatMap(func(value interface{}) Trampoline {

					if _, ok := value.(controlBreak); ok {
						return Done{}
					}

					return Done{Result: value}
				})
		}

		// If the case has no expression it is the default case.
		// Evaluate it, i.e. all statements

		if switchCase.Expression == nil {
			return runStatements()
		}

		// The case has an expression.
		// Evaluate it and compare it to the test value

		return switchCase.Expression.Accept(interpreter).(Trampoline).
			FlatMap(func(result interface{}) Trampoline {
				caseValue := result.(EquatableValue)

				// If the test value and case values are equal,
				// evaluate the case's statements

				if testValue.Equal(interpreter, caseValue) {
					return runStatements()
				}

				// If the test value and the case values are unequal,
				// try the next case (recurse)

				return visitCase(i+1, testValue)
			})
	}

	return switchStatement.Expression.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			testValue := result.(EquatableValue)
			return visitCase(0, testValue)
		})
}

func (interpreter *Interpreter) VisitWhileStatement(statement *ast.WhileStatement) ast.Repr {

	return statement.Test.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			value := result.(BoolValue)
			if !value {
				return Done{}
			}

			interpreter.reportLoopIteration(statement)

			return statement.Block.Accept(interpreter).(Trampoline).
				FlatMap(func(value interface{}) Trampoline {

					switch value.(type) {
					case controlBreak:
						return Done{}

					case controlContinue:
						// NO-OP

					case functionReturn:
						return Done{Result: value}
					}

					// recurse
					return statement.Accept(interpreter).(Trampoline)
				})
		})
}

func (interpreter *Interpreter) VisitForStatement(statement *ast.ForStatement) ast.Repr {
	interpreter.activations.PushNewWithCurrent()

	variable := interpreter.declareVariable(
		statement.Identifier.Identifier,
		nil,
	)

	var loop func(i, count int, values []Value) Trampoline
	loop = func(i, count int, values []Value) Trampoline {

		if i == count {
			return Done{}
		}

		interpreter.reportLoopIteration(statement)

		variable.Value = values[i]

		return statement.Block.Accept(interpreter).(Trampoline).
			FlatMap(func(value interface{}) Trampoline {

				switch value.(type) {
				case controlBreak:
					return Done{}

				case controlContinue:
					// NO-OP

				case functionReturn:
					return Done{Result: value}
				}

				// recurse
				if i == count {
					return Done{}
				}
				return loop(i+1, count, values)
			})
	}

	return statement.Value.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {

			values := result.(*ArrayValue).Values[:]
			count := len(values)

			return loop(0, count, values)
		}).
		Then(func(_ interface{}) {
			interpreter.activations.Pop()
		})
}

func (interpreter *Interpreter) visitPotentialStorageRemoval(expression ast.Expression) Trampoline {
	movingStorageIndexExpression := interpreter.movingStorageIndexExpression(expression)
	if movingStorageIndexExpression == nil {
		return expression.Accept(interpreter).(Trampoline)
	}

	return interpreter.indexExpressionGetterSetter(movingStorageIndexExpression).
		Map(func(result interface{}) interface{} {
			getterSetter := result.(getterSetter)
			value := getterSetter.get()
			getterSetter.set(NilValue{})
			return value
		})
}

// VisitVariableDeclaration first visits the declaration's value,
// then declares the variable with the name bound to the value
func (interpreter *Interpreter) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Repr {

	targetType := interpreter.Program.Elaboration.VariableDeclarationTargetTypes[declaration]
	valueType := interpreter.Program.Elaboration.VariableDeclarationValueTypes[declaration]
	secondValueType := interpreter.Program.Elaboration.VariableDeclarationSecondValueTypes[declaration]

	return interpreter.visitPotentialStorageRemoval(declaration.Value).
		FlatMap(func(result interface{}) Trampoline {

			valueCopy := interpreter.copyAndConvert(result.(Value), valueType, targetType)

			interpreter.declareVariable(
				declaration.Identifier.Identifier,
				valueCopy,
			)

			if declaration.SecondValue == nil {
				// NOTE: ignore result, so it does *not* act like a return-statement
				return Done{}
			}

			return interpreter.visitAssignment(
				declaration.Transfer.Operation,
				declaration.Value,
				valueType,
				declaration.SecondValue,
				secondValueType,
				declaration,
			)
		})
}

func (interpreter *Interpreter) movingStorageIndexExpression(expression ast.Expression) *ast.IndexExpression {
	indexExpression, ok := expression.(*ast.IndexExpression)
	if !ok || !interpreter.Program.Elaboration.IsResourceMovingStorageIndexExpression[indexExpression] {
		return nil
	}

	return indexExpression
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
	variable := NewVariable(value)
	interpreter.setVariable(identifier, variable)
	return variable
}

func (interpreter *Interpreter) VisitAssignmentStatement(assignment *ast.AssignmentStatement) ast.Repr {
	targetType := interpreter.Program.Elaboration.AssignmentStatementTargetTypes[assignment]
	valueType := interpreter.Program.Elaboration.AssignmentStatementValueTypes[assignment]

	target := assignment.Target
	value := assignment.Value

	return interpreter.visitAssignment(
		assignment.Transfer.Operation,
		target, targetType,
		value, valueType,
		assignment,
	)
}

func (interpreter *Interpreter) visitAssignment(
	transferOperation ast.TransferOperation,
	target ast.Expression, targetType sema.Type,
	value ast.Expression, valueType sema.Type,
	position ast.HasPosition,
) Trampoline {

	// First evaluate the target, which results in a getter/setter function pair
	return interpreter.assignmentGetterSetter(target).
		FlatMap(func(result interface{}) Trampoline {
			getterSetter := result.(getterSetter)

			// If the assignment is a forced move,
			// ensure that the target is nil,
			// otherwise panic

			if transferOperation == ast.TransferOperationMoveForced {
				target := getterSetter.get()
				if _, ok := target.(NilValue); !ok {
					locationRange := interpreter.locationRange(position)

					panic(ForceAssignmentToNonNilResourceError{
						LocationRange: locationRange,
					})
				}
			}

			// Finally, evaluate the value, and assign it using the setter function
			return value.Accept(interpreter).(Trampoline).
				FlatMap(func(result interface{}) Trampoline {

					valueCopy := interpreter.copyAndConvert(result.(Value), valueType, targetType)
					getterSetter.set(valueCopy)

					// NOTE: no result, so it does *not* act like a return-statement
					return Done{}
				})
		})
}

func (interpreter *Interpreter) VisitSwapStatement(swap *ast.SwapStatement) ast.Repr {

	leftType := interpreter.Program.Elaboration.SwapStatementLeftTypes[swap]
	rightType := interpreter.Program.Elaboration.SwapStatementRightTypes[swap]

	// Evaluate the left expression
	return interpreter.assignmentGetterSetter(swap.Left).
		FlatMap(func(result interface{}) Trampoline {
			leftGetterSetter := result.(getterSetter)
			leftValue := leftGetterSetter.get()
			if interpreter.movingStorageIndexExpression(swap.Left) != nil {
				leftGetterSetter.set(NilValue{})
			}

			// Evaluate the right expression
			return interpreter.assignmentGetterSetter(swap.Right).
				Then(func(result interface{}) {
					rightGetterSetter := result.(getterSetter)
					rightValue := rightGetterSetter.get()
					if interpreter.movingStorageIndexExpression(swap.Right) != nil {
						rightGetterSetter.set(NilValue{})
					}

					// Add right value to left target
					// and left value to right target

					rightValueCopy := interpreter.copyAndConvert(rightValue.(Value), rightType, leftType)
					leftValueCopy := interpreter.copyAndConvert(leftValue.(Value), leftType, rightType)

					leftGetterSetter.set(rightValueCopy)
					rightGetterSetter.set(leftValueCopy)
				})
		})
}

// assignmentGetterSetter returns a getter/setter function pair
// for the target expression, wrapped in a trampoline
//
func (interpreter *Interpreter) assignmentGetterSetter(target ast.Expression) Trampoline {
	switch target := target.(type) {
	case *ast.IdentifierExpression:
		return interpreter.identifierExpressionGetterSetter(target)

	case *ast.IndexExpression:
		return interpreter.indexExpressionGetterSetter(target)

	case *ast.MemberExpression:
		return interpreter.memberExpressionGetterSetter(target)
	}

	panic(errors.NewUnreachableError())
}

// identifierExpressionGetterSetter returns a getter/setter function pair
// for the target identifier expression, wrapped in a trampoline
//
func (interpreter *Interpreter) identifierExpressionGetterSetter(identifierExpression *ast.IdentifierExpression) Trampoline {
	variable := interpreter.findVariable(identifierExpression.Identifier.Identifier)
	return Done{
		Result: getterSetter{
			get: func() Value {
				return variable.Value
			},
			set: func(value Value) {
				variable.Value = value
			},
		},
	}
}

// indexExpressionGetterSetter returns a getter/setter function pair
// for the target index expression, wrapped in a trampoline
//
func (interpreter *Interpreter) indexExpressionGetterSetter(indexExpression *ast.IndexExpression) Trampoline {
	return indexExpression.TargetExpression.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			typedResult := result.(ValueIndexableValue)
			return indexExpression.IndexingExpression.Accept(interpreter).(Trampoline).
				FlatMap(func(result interface{}) Trampoline {
					indexingValue := result.(Value)
					locationRange := interpreter.locationRange(indexExpression)
					return Done{
						Result: getterSetter{
							get: func() Value {
								return typedResult.Get(interpreter, locationRange, indexingValue)
							},
							set: func(value Value) {
								typedResult.Set(interpreter, locationRange, indexingValue, value)
							},
						},
					}
				})
		})
}

// memberExpressionGetterSetter returns a getter/setter function pair
// for the target member expression, wrapped in a trampoline
//
func (interpreter *Interpreter) memberExpressionGetterSetter(memberExpression *ast.MemberExpression) Trampoline {
	return memberExpression.Expression.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			target := result.(Value)
			locationRange := interpreter.locationRange(memberExpression)
			identifier := memberExpression.Identifier.Identifier
			return Done{
				Result: getterSetter{
					get: func() Value {
						return interpreter.getMember(target, locationRange, identifier)
					},
					set: func(value Value) {
						interpreter.setMember(target, locationRange, identifier, value)
					},
				},
			}
		})
}

func (interpreter *Interpreter) VisitIdentifierExpression(expression *ast.IdentifierExpression) ast.Repr {
	name := expression.Identifier.Identifier
	variable := interpreter.findVariable(name)
	return Done{Result: variable.Value}
}

// valueTuple

type valueTuple struct {
	left, right Value
}

// visitBinaryOperation interprets the left-hand side and the right-hand side and returns
// the result in a valueTuple
func (interpreter *Interpreter) visitBinaryOperation(expr *ast.BinaryExpression) Trampoline {
	// interpret the left-hand side
	return expr.Left.Accept(interpreter).(Trampoline).
		FlatMap(func(left interface{}) Trampoline {
			// after interpreting the left-hand side,
			// interpret the right-hand side
			return expr.Right.Accept(interpreter).(Trampoline).
				FlatMap(func(right interface{}) Trampoline {
					tuple := valueTuple{
						left.(Value),
						right.(Value),
					}
					return Done{Result: tuple}
				})
		})
}

func (interpreter *Interpreter) visitNumberBinaryOperation(
	expression *ast.BinaryExpression,
	f func(left, right NumberValue) Value,
) ast.Repr {
	return interpreter.visitBinaryOperation(expression).
		Map(func(result interface{}) interface{} {
			tuple := result.(valueTuple)
			left := tuple.left.(NumberValue)
			right := tuple.right.(NumberValue)
			return f(left, right)
		})
}

func (interpreter *Interpreter) VisitBinaryExpression(expression *ast.BinaryExpression) ast.Repr {
	switch expression.Operation {
	case ast.OperationPlus:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.Plus(right)
			},
		)

	case ast.OperationMinus:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.Minus(right)
			},
		)

	case ast.OperationMod:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.Mod(right)
			},
		)

	case ast.OperationMul:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.Mul(right)
			},
		)

	case ast.OperationDiv:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.Div(right)
			},
		)

	case ast.OperationBitwiseOr:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				leftInteger := left.(IntegerValue)
				rightInteger := right.(IntegerValue)
				return leftInteger.BitwiseOr(rightInteger)
			},
		)

	case ast.OperationBitwiseXor:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				leftInteger := left.(IntegerValue)
				rightInteger := right.(IntegerValue)
				return leftInteger.BitwiseXor(rightInteger)
			},
		)

	case ast.OperationBitwiseAnd:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				leftInteger := left.(IntegerValue)
				rightInteger := right.(IntegerValue)
				return leftInteger.BitwiseAnd(rightInteger)
			},
		)

	case ast.OperationBitwiseLeftShift:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				leftInteger := left.(IntegerValue)
				rightInteger := right.(IntegerValue)
				return leftInteger.BitwiseLeftShift(rightInteger)
			},
		)

	case ast.OperationBitwiseRightShift:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				leftInteger := left.(IntegerValue)
				rightInteger := right.(IntegerValue)
				return leftInteger.BitwiseRightShift(rightInteger)
			},
		)

	case ast.OperationLess:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.Less(right)
			},
		)

	case ast.OperationLessEqual:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.LessEqual(right)
			},
		)

	case ast.OperationGreater:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.Greater(right)
			},
		)

	case ast.OperationGreaterEqual:
		return interpreter.visitNumberBinaryOperation(
			expression,
			func(left, right NumberValue) Value {
				return left.GreaterEqual(right)
			},
		)

	case ast.OperationEqual:
		return interpreter.visitBinaryOperation(expression).
			Map(func(result interface{}) interface{} {
				tuple := result.(valueTuple)
				return interpreter.testEqual(tuple.left, tuple.right)
			})

	case ast.OperationNotEqual:
		return interpreter.visitBinaryOperation(expression).
			Map(func(result interface{}) interface{} {
				tuple := result.(valueTuple)
				return !interpreter.testEqual(tuple.left, tuple.right)
			})

	case ast.OperationOr:
		// interpret the left-hand side
		return expression.Left.Accept(interpreter).(Trampoline).
			FlatMap(func(left interface{}) Trampoline {
				// only interpret right-hand side if left-hand side is false
				leftBool := left.(BoolValue)
				if leftBool {
					return Done{Result: leftBool}
				}

				// after interpreting the left-hand side,
				// interpret the right-hand side
				return expression.Right.Accept(interpreter).(Trampoline).
					FlatMap(func(right interface{}) Trampoline {
						return Done{Result: right.(BoolValue)}
					})
			})

	case ast.OperationAnd:
		// interpret the left-hand side
		return expression.Left.Accept(interpreter).(Trampoline).
			FlatMap(func(left interface{}) Trampoline {
				// only interpret right-hand side if left-hand side is true
				leftBool := left.(BoolValue)
				if !leftBool {
					return Done{Result: leftBool}
				}

				// after interpreting the left-hand side,
				// interpret the right-hand side
				return expression.Right.Accept(interpreter).(Trampoline).
					FlatMap(func(right interface{}) Trampoline {
						return Done{Result: right.(BoolValue)}
					})
			})

	case ast.OperationNilCoalesce:
		// interpret the left-hand side
		return expression.Left.Accept(interpreter).(Trampoline).
			FlatMap(func(left interface{}) Trampoline {
				// only evaluate right-hand side if left-hand side is nil
				if _, ok := left.(NilValue); ok {
					return expression.Right.Accept(interpreter).(Trampoline).
						Map(func(result interface{}) interface{} {
							value := result.(Value)

							rightType := interpreter.Program.Elaboration.BinaryExpressionRightTypes[expression]
							resultType := interpreter.Program.Elaboration.BinaryExpressionResultTypes[expression]

							// NOTE: important to convert both any and optional
							return interpreter.convertAndBox(value, rightType, resultType)
						})
				}

				value := left.(*SomeValue).Value
				return Done{Result: value}
			})
	}

	panic(&unsupportedOperation{
		kind:      common.OperationKindBinary,
		operation: expression.Operation,
		Range:     ast.NewRangeFromPositioned(expression),
	})
}

func (interpreter *Interpreter) testEqual(left, right Value) BoolValue {
	left = interpreter.unbox(left)
	right = interpreter.unbox(right)

	// TODO: add support for arrays and dictionaries

	switch left := left.(type) {
	case NilValue:
		_, ok := right.(NilValue)
		return BoolValue(ok)

	case EquatableValue:
		// NOTE: might be NilValue
		right, ok := right.(EquatableValue)
		if !ok {
			return false
		}
		return left.Equal(interpreter, right)

	case *ArrayValue,
		*DictionaryValue:
		// TODO:
		return false

	default:
		return false
	}
}

func (interpreter *Interpreter) VisitUnaryExpression(expression *ast.UnaryExpression) ast.Repr {
	return expression.Expression.Accept(interpreter).(Trampoline).
		Map(func(result interface{}) interface{} {
			value := result.(Value)

			switch expression.Operation {
			case ast.OperationNegate:
				boolValue := value.(BoolValue)
				return boolValue.Negate()

			case ast.OperationMinus:
				integerValue := value.(NumberValue)
				return integerValue.Negate()

			case ast.OperationMove:
				return value
			}

			panic(&unsupportedOperation{
				kind:      common.OperationKindUnary,
				operation: expression.Operation,
				Range:     ast.NewRangeFromPositioned(expression),
			})
		})
}

func (interpreter *Interpreter) VisitExpressionStatement(statement *ast.ExpressionStatement) ast.Repr {
	return statement.Expression.Accept(interpreter).(Trampoline).
		Map(func(result interface{}) interface{} {
			var value Value
			var ok bool
			value, ok = result.(Value)
			if !ok {
				value = nil
			}
			return ExpressionStatementResult{value}
		})
}

func (interpreter *Interpreter) VisitBoolExpression(expression *ast.BoolExpression) ast.Repr {
	value := BoolValue(expression.Value)

	return Done{Result: value}
}

func (interpreter *Interpreter) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	value := NilValue{}
	return Done{Result: value}
}

func (interpreter *Interpreter) VisitIntegerExpression(expression *ast.IntegerExpression) ast.Repr {
	value := IntValue{expression.Value}

	return Done{Result: value}
}

func (interpreter *Interpreter) VisitFixedPointExpression(expression *ast.FixedPointExpression) ast.Repr {
	// TODO: adjust once/if we support more fixed point types

	value := fixedpoint.ConvertToFixedPointBigInt(
		expression.Negative,
		expression.UnsignedInteger,
		expression.Fractional,
		expression.Scale,
		sema.Fix64Scale,
	)

	var result Value

	if expression.Negative {
		result = Fix64Value(value.Int64())
	} else {
		result = UFix64Value(value.Uint64())
	}

	return Done{Result: result}
}

func (interpreter *Interpreter) VisitStringExpression(expression *ast.StringExpression) ast.Repr {
	value := NewStringValue(expression.Value)

	return Done{Result: value}
}

func (interpreter *Interpreter) VisitArrayExpression(expression *ast.ArrayExpression) ast.Repr {
	return interpreter.visitExpressionsNonCopying(expression.Values).
		FlatMap(func(result interface{}) Trampoline {
			values := result.(*ArrayValue)

			argumentTypes := interpreter.Program.Elaboration.ArrayExpressionArgumentTypes[expression]
			elementType := interpreter.Program.Elaboration.ArrayExpressionElementType[expression]

			copies := make([]Value, len(values.Values))
			for i, argument := range values.Values {
				argumentType := argumentTypes[i]
				copies[i] = interpreter.copyAndConvert(argument, argumentType, elementType)
			}

			return Done{Result: NewArrayValueUnownedNonCopying(copies...)}
		})
}

func (interpreter *Interpreter) VisitDictionaryExpression(expression *ast.DictionaryExpression) ast.Repr {
	return interpreter.visitEntries(expression.Entries).
		FlatMap(func(result interface{}) Trampoline {

			entryTypes := interpreter.Program.Elaboration.DictionaryExpressionEntryTypes[expression]
			dictionaryType := interpreter.Program.Elaboration.DictionaryExpressionType[expression]

			newDictionary := NewDictionaryValueUnownedNonCopying()
			for i, dictionaryEntryValues := range result.([]DictionaryEntryValues) {
				entryType := entryTypes[i]

				key := interpreter.copyAndConvert(
					dictionaryEntryValues.Key,
					entryType.KeyType,
					dictionaryType.KeyType,
				)

				value := interpreter.copyAndConvert(
					dictionaryEntryValues.Value,
					entryType.ValueType,
					dictionaryType.ValueType,
				)

				// TODO: panic for duplicate keys?

				// NOTE: important to convert in optional, as assignment to dictionary
				// is always considered as an optional

				locationRange := interpreter.locationRange(expression)
				_ = newDictionary.Insert(interpreter, locationRange, key, value)
			}

			return Done{Result: newDictionary}
		})
}

func (interpreter *Interpreter) VisitMemberExpression(expression *ast.MemberExpression) ast.Repr {
	return expression.Expression.Accept(interpreter).(Trampoline).
		Map(func(result interface{}) interface{} {
			if expression.Optional {
				switch typedResult := result.(type) {
				case NilValue:
					return typedResult

				case *SomeValue:
					result = typedResult.Value

				default:
					panic(errors.NewUnreachableError())
				}
			}

			value := result.(Value)
			locationRange := interpreter.locationRange(expression)
			resultValue := interpreter.getMember(value, locationRange, expression.Identifier.Identifier)

			// If the member access is optional chaining, only wrap the result value
			// in an optional, if it is not already an optional value

			if expression.Optional {
				if _, ok := resultValue.(OptionalValue); !ok {
					return NewSomeValueOwningNonCopying(resultValue)
				}
			}
			return resultValue
		})
}

func (interpreter *Interpreter) VisitIndexExpression(expression *ast.IndexExpression) ast.Repr {
	return expression.TargetExpression.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			typedResult := result.(ValueIndexableValue)
			return expression.IndexingExpression.Accept(interpreter).(Trampoline).
				FlatMap(func(result interface{}) Trampoline {
					indexingValue := result.(Value)
					locationRange := interpreter.locationRange(expression)
					value := typedResult.Get(interpreter, locationRange, indexingValue)
					return Done{Result: value}
				})
		})
}

func (interpreter *Interpreter) VisitConditionalExpression(expression *ast.ConditionalExpression) ast.Repr {
	return expression.Test.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			value := result.(BoolValue)

			if value {
				return expression.Then.Accept(interpreter).(Trampoline)
			}
			return expression.Else.Accept(interpreter).(Trampoline)
		})
}

func (interpreter *Interpreter) VisitInvocationExpression(invocationExpression *ast.InvocationExpression) ast.Repr {
	// interpret the invoked expression
	return invocationExpression.InvokedExpression.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {

			// Handle optional chaining on member expression, if any:
			// - If the member expression is nil, finish execution
			// - If the member expression is some value, the wrapped value
			//   is the function value that should be invoked

			isOptionalChaining := false

			if invokedMemberExpression, ok :=
				invocationExpression.InvokedExpression.(*ast.MemberExpression); ok && invokedMemberExpression.Optional {

				isOptionalChaining = true

				switch typedResult := result.(type) {
				case NilValue:
					return Done{Result: typedResult}

				case *SomeValue:
					result = typedResult.Value

				default:
					panic(errors.NewUnreachableError())
				}
			}

			function := result.(FunctionValue)

			// NOTE: evaluate all argument expressions in call-site scope, not in function body
			argumentExpressions := make([]ast.Expression, len(invocationExpression.Arguments))
			for i, argument := range invocationExpression.Arguments {
				argumentExpressions[i] = argument.Expression
			}

			return interpreter.visitExpressionsNonCopying(argumentExpressions).
				FlatMap(func(result interface{}) Trampoline {
					arguments := result.(*ArrayValue).Values

					typeParameterTypes :=
						interpreter.Program.Elaboration.InvocationExpressionTypeArguments[invocationExpression]
					argumentTypes :=
						interpreter.Program.Elaboration.InvocationExpressionArgumentTypes[invocationExpression]
					parameterTypes :=
						interpreter.Program.Elaboration.InvocationExpressionParameterTypes[invocationExpression]

					invocation := interpreter.functionValueInvocationTrampoline(
						function,
						arguments,
						argumentTypes,
						parameterTypes,
						typeParameterTypes,
						ast.NewRangeFromPositioned(invocationExpression),
					)

					interpreter.reportFunctionInvocation(invocationExpression)

					// If this is invocation is optional chaining, wrap the result
					// as an optional, as the result is expected to be an optional

					if !isOptionalChaining {
						return invocation
					}

					return invocation.Map(func(result interface{}) interface{} {
						return NewSomeValueOwningNonCopying(result.(Value))
					})
				})
		})
}

func (interpreter *Interpreter) InvokeFunctionValue(
	function FunctionValue,
	arguments []Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	invocationRange ast.Range,
) (value Value, err error) {
	// recover internal panics and return them as an error
	defer recoverErrors(func(internalErr error) {
		err = internalErr
	})

	trampoline := interpreter.functionValueInvocationTrampoline(
		function,
		arguments,
		argumentTypes,
		parameterTypes,
		nil,
		invocationRange,
	)

	result := interpreter.runAllStatements(trampoline)
	if result == nil {
		return nil, nil
	}
	return result.(Value), nil
}

func (interpreter *Interpreter) functionValueInvocationTrampoline(
	function FunctionValue,
	arguments []Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	typeParameterTypes *sema.TypeParameterTypeOrderedMap,
	invocationRange ast.Range,
) Trampoline {

	parameterTypeCount := len(parameterTypes)
	argumentCopies := make([]Value, len(arguments))

	for i, argument := range arguments {
		argumentType := argumentTypes[i]
		if i < parameterTypeCount {
			parameterType := parameterTypes[i]
			argumentCopies[i] = interpreter.copyAndConvert(argument, argumentType, parameterType)
		} else {
			argumentCopies[i] = argument.Copy()
		}
	}

	// TODO: optimize: only potentially used by host-functions

	locationRange := LocationRange{
		Location: interpreter.Location,
		Range:    invocationRange,
	}

	return function.Invoke(
		Invocation{
			Arguments:          argumentCopies,
			ArgumentTypes:      argumentTypes,
			TypeParameterTypes: typeParameterTypes,
			LocationRange:      locationRange,
			Interpreter:        interpreter,
		},
	)
}

func (interpreter *Interpreter) invokeInterpretedFunction(
	function InterpretedFunctionValue,
	invocation Invocation,
) Trampoline {

	// Start a new activation record.
	// Lexical scope: use the function declaration's activation record,
	// not the current one (which would be dynamic scope)
	interpreter.activations.PushNewWithParent(function.Activation)

	// Make `self` available, if any
	if invocation.Self != nil {
		interpreter.declareVariable(sema.SelfIdentifier, invocation.Self)
	}

	return interpreter.invokeInterpretedFunctionActivated(function, invocation.Arguments)
}

// NOTE: assumes the function's activation (or an extension of it) is pushed!
//
func (interpreter *Interpreter) invokeInterpretedFunctionActivated(
	function InterpretedFunctionValue,
	arguments []Value,
) Trampoline {

	if function.ParameterList != nil {
		interpreter.bindParameterArguments(function.ParameterList, arguments)
	}

	functionBlockTrampoline := interpreter.visitFunctionBody(
		function.BeforeStatements,
		function.PreConditions,
		interpreter.visitStatements(function.Statements),
		function.PostConditions,
		function.Type.ReturnTypeAnnotation.Type,
	)

	return functionBlockTrampoline.
		Then(func(_ interface{}) {
			interpreter.activations.Pop()
		})
}

// bindParameterArguments binds the argument values to the given parameters
//
func (interpreter *Interpreter) bindParameterArguments(
	parameterList *ast.ParameterList,
	arguments []Value,
) {
	for parameterIndex, parameter := range parameterList.Parameters {
		argument := arguments[parameterIndex]
		interpreter.declareVariable(parameter.Identifier.Identifier, argument)
	}
}

func (interpreter *Interpreter) visitExpressionsNonCopying(expressions []ast.Expression) Trampoline {
	var trampoline Trampoline = Done{Result: NewArrayValueUnownedNonCopying()}

	for _, expression := range expressions {
		// NOTE: important: rebind expression, because it is captured in the closure below
		expression := expression

		// append the evaluation of this expression
		trampoline = trampoline.FlatMap(func(result interface{}) Trampoline {
			array := result.(*ArrayValue)

			// evaluate the expression
			return expression.Accept(interpreter).(Trampoline).
				FlatMap(func(result interface{}) Trampoline {
					value := result.(Value)

					newValues := append(array.Values, value)
					return Done{Result: NewArrayValueUnownedNonCopying(newValues...)}
				})
		})
	}

	return trampoline
}

func (interpreter *Interpreter) visitEntries(entries []ast.DictionaryEntry) Trampoline {
	var trampoline Trampoline = Done{Result: []DictionaryEntryValues{}}

	for _, entry := range entries {
		// NOTE: important: rebind entry, because it is captured in the closure below
		func(entry ast.DictionaryEntry) {
			// append the evaluation of this entry
			trampoline = trampoline.FlatMap(func(result interface{}) Trampoline {
				resultEntries := result.([]DictionaryEntryValues)

				// evaluate the key expression
				return entry.Key.Accept(interpreter).(Trampoline).
					FlatMap(func(result interface{}) Trampoline {
						key := result.(Value)

						// evaluate the value expression
						return entry.Value.Accept(interpreter).(Trampoline).
							FlatMap(func(result interface{}) Trampoline {
								value := result.(Value)

								newResultEntries := append(
									resultEntries,
									DictionaryEntryValues{
										Key:   key,
										Value: value,
									},
								)
								return Done{Result: newResultEntries}
							})
					})
			})
		}(entry)
	}

	return trampoline
}

func (interpreter *Interpreter) VisitFunctionExpression(expression *ast.FunctionExpression) ast.Repr {

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	functionType := interpreter.Program.Elaboration.FunctionExpressionFunctionType[expression]

	var preConditions ast.Conditions
	if expression.FunctionBlock.PreConditions != nil {
		preConditions = *expression.FunctionBlock.PreConditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

	if expression.FunctionBlock.PostConditions != nil {
		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite[expression.FunctionBlock.PostConditions]

		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
		beforeStatements = postConditionsRewrite.BeforeStatements
	}

	statements := expression.FunctionBlock.Block.Statements

	function := InterpretedFunctionValue{
		Interpreter:      interpreter,
		ParameterList:    expression.ParameterList,
		Type:             functionType,
		Activation:       lexicalScope,
		BeforeStatements: beforeStatements,
		PreConditions:    preConditions,
		Statements:       statements,
		PostConditions:   rewrittenPostConditions,
	}

	return Done{Result: function}
}

// NOTE: only called for top-level composite declarations
func (interpreter *Interpreter) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) ast.Repr {

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	_, _ = interpreter.declareCompositeValue(declaration, lexicalScope)

	// NOTE: no result, so it does *not* act like a return-statement
	return Done{}
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
	lexicalScope *activations.Activation,
) (
	scope *activations.Activation,
	value Value,
) {
	if declaration.CompositeKind == common.CompositeKindEnum {
		return interpreter.declareEnumConstructor(declaration, lexicalScope)
	} else {
		return interpreter.declareNonEnumCompositeValue(declaration, lexicalScope)
	}
}

func (interpreter *Interpreter) declareNonEnumCompositeValue(
	declaration *ast.CompositeDeclaration,
	lexicalScope *activations.Activation,
) (
	scope *activations.Activation,
	value Value,
) {
	identifier := declaration.Identifier.Identifier
	// NOTE: find *or* declare, as the function might have not been pre-declared (e.g. in the REPL)
	variable := interpreter.findOrDeclareVariable(identifier)

	// Make the value available in the initializer
	lexicalScope.Set(identifier, variable)

	// Evaluate nested declarations in a new scope, so values
	// of nested declarations won't be visible after the containing declaration

	members := NewStringValueOrderedMap()

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

			var nestedValue Value
			lexicalScope, nestedValue =
				interpreter.declareCompositeValue(nestedCompositeDeclaration, lexicalScope)

			memberIdentifier := nestedCompositeDeclaration.Identifier.Identifier
			members.Set(memberIdentifier, nestedValue)
		}
	})()

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationTypes[declaration]

	var initializerFunction FunctionValue
	if declaration.CompositeKind == common.CompositeKindEvent {
		initializerFunction = NewHostFunctionValue(
			func(invocation Invocation) Trampoline {
				for i, argument := range invocation.Arguments {
					parameter := compositeType.ConstructorParameters[i]
					invocation.Self.Fields.Set(parameter.Identifier, argument)
				}
				return Done{}
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
		func(invocation Invocation) Trampoline {

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
						LocationRange: invocation.LocationRange,
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

				variable.Value = value

				// Also, immediately set the nested values,
				// as the initializer of the contract may use nested declarations

				value.NestedValues = members
			}

			var initializationTrampoline Trampoline = Done{}

			if initializerFunction != nil {
				// NOTE: arguments are already properly boxed by invocation expression

				initializationTrampoline = initializerFunction.Invoke(invocation)
			}

			return initializationTrampoline.
				Map(func(_ interface{}) interface{} {
					return value
				})
		},
	)

	// Contract declarations declare a value / instance (singleton),
	// for all other composite kinds, the constructor is declared

	if declaration.CompositeKind == common.CompositeKindContract {
		positioned := ast.NewRangeFromPositioned(declaration.Identifier)
		contract := interpreter.contractValueHandler(
			interpreter,
			compositeType,
			constructor,
			positioned,
		)
		contract.NestedValues = members
		value = contract
		// NOTE: variable value is also set in the constructor function: it needs to be available
		// for nested declarations, which might be invoked when the constructor is invoked
		variable.Value = value
	} else {
		constructor.Members = members
		value = constructor
		variable.Value = value
	}

	return lexicalScope, value
}

func (interpreter *Interpreter) declareEnumConstructor(
	declaration *ast.CompositeDeclaration,
	lexicalScope *activations.Activation,
) (
	scope *activations.Activation,
	value Value,
) {
	identifier := declaration.Identifier.Identifier
	// NOTE: find *or* declare, as the function might have not been pre-declared (e.g. in the REPL)
	variable := interpreter.findOrDeclareVariable(identifier)

	lexicalScope.Set(identifier, variable)

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationTypes[declaration]
	qualifiedIdentifier := compositeType.QualifiedIdentifier()

	location := interpreter.Location

	intType := &sema.IntType{}

	enumCases := declaration.Members.EnumCases()
	caseCount := len(enumCases)
	caseValues := make([]*CompositeValue, caseCount)

	constructorMembers := NewStringValueOrderedMap()

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
		constructorMembers.Set(enumCase.Identifier.Identifier, caseValue)
	}

	constructor := NewHostFunctionValue(
		func(invocation Invocation) Trampoline {

			rawValueArgument := invocation.Arguments[0].(IntegerValue).ToInt()

			var result Value = NilValue{}

			if rawValueArgument >= 0 && rawValueArgument < caseCount {
				caseValue := caseValues[rawValueArgument]
				result = NewSomeValueOwningNonCopying(caseValue)
			}

			return Done{Result: result}
		},
	)

	constructor.Members = constructorMembers

	value = constructor
	variable.Value = value

	return lexicalScope, value
}

func (interpreter *Interpreter) compositeInitializerFunction(
	compositeDeclaration *ast.CompositeDeclaration,
	lexicalScope *activations.Activation,
) *InterpretedFunctionValue {

	// TODO: support multiple overloaded initializers

	initializers := compositeDeclaration.Members.Initializers()
	var initializer *ast.SpecialFunctionDeclaration
	if len(initializers) == 0 {
		return nil
	}

	initializer = initializers[0]
	functionType := interpreter.Program.Elaboration.SpecialFunctionTypes[initializer].FunctionType

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
	lexicalScope *activations.Activation,
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
	lexicalScope *activations.Activation,
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
	lexicalScope *activations.Activation,
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
	lexicalScope *activations.Activation,
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

func (interpreter *Interpreter) copyAndConvert(value Value, valueType, targetType sema.Type) Value {
	return interpreter.convertAndBox(value.Copy(), valueType, targetType)
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

	if valueType.Equal(unwrappedTargetType) {
		return value
	}

	switch unwrappedTargetType.(type) {
	case *sema.IntType:
		return ConvertInt(value, interpreter)

	case *sema.UIntType:
		return ConvertUInt(value, interpreter)

	case *sema.AddressType:
		return ConvertAddress(value, interpreter)

	// Int*
	case *sema.Int8Type:
		return ConvertInt8(value, interpreter)

	case *sema.Int16Type:
		return ConvertInt16(value, interpreter)

	case *sema.Int32Type:
		return ConvertInt32(value, interpreter)

	case *sema.Int64Type:
		return ConvertInt64(value, interpreter)

	case *sema.Int128Type:
		return ConvertInt128(value, interpreter)

	case *sema.Int256Type:
		return ConvertInt256(value, interpreter)

	// UInt*
	case *sema.UInt8Type:
		return ConvertUInt8(value, interpreter)

	case *sema.UInt16Type:
		return ConvertUInt16(value, interpreter)

	case *sema.UInt32Type:
		return ConvertUInt32(value, interpreter)

	case *sema.UInt64Type:
		return ConvertUInt64(value, interpreter)

	case *sema.UInt128Type:
		return ConvertUInt128(value, interpreter)

	case *sema.UInt256Type:
		return ConvertUInt256(value, interpreter)

	// Word*
	case *sema.Word8Type:
		return ConvertWord8(value, interpreter)

	case *sema.Word16Type:
		return ConvertWord16(value, interpreter)

	case *sema.Word32Type:
		return ConvertWord32(value, interpreter)

	case *sema.Word64Type:
		return ConvertWord64(value, interpreter)

	// Fix*

	case *sema.Fix64Type:
		return ConvertFix64(value, interpreter)

	case *sema.UFix64Type:
		return ConvertUFix64(value, interpreter)

	default:
		return value
	}
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

	// NOTE: no result, so it does *not* act like a return-statement
	return Done{}
}

func (interpreter *Interpreter) declareInterface(
	declaration *ast.InterfaceDeclaration,
	lexicalScope *activations.Activation,
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
	lexicalScope *activations.Activation,
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
	lexicalScope *activations.Activation,
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
	lexicalScope *activations.Activation,
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
	lexicalScope *activations.Activation,
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
		return NewHostFunctionValue(func(invocation Invocation) Trampoline {
			// Start a new activation record.
			// Lexical scope: use the function declaration's activation record,
			// not the current one (which would be dynamic scope)
			interpreter.activations.PushNewWithParent(lexicalScope)

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

			var body Trampoline = Done{}
			if inner != nil {
				// NOTE: It is important to wrap the invocation in a trampoline,
				//  so the inner function isn't invoked here

				body = More(func() Trampoline {

					// NOTE: It is important to actually return the value returned
					//   from the inner function, otherwise it is lost

					return inner.Invoke(invocation).
						Map(func(returnValue interface{}) interface{} {
							return functionReturn{returnValue.(Value)}
						})
				})
			}

			functionBlockTrampoline := interpreter.visitFunctionBody(
				beforeStatements,
				preConditions,
				body,
				rewrittenPostConditions,
				returnType,
			)

			return functionBlockTrampoline.
				Then(func(_ interface{}) {
					interpreter.activations.Pop()
				})
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
			variable := NewVariable(global.Value)
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
		WithStorageKeyHandler(interpreter.storageKeyHandler),
		WithInjectedCompositeFieldsHandler(interpreter.injectedCompositeFieldsHandler),
		WithContractValueHandler(interpreter.contractValueHandler),
		WithImportLocationHandler(interpreter.importLocationHandler),
		WithUUIDHandler(interpreter.uuidHandler),
		WithAllInterpreters(interpreter.allInterpreters),
		withTypeCodes(interpreter.typeCodes),
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

func (interpreter *Interpreter) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) ast.Repr {
	return Done{}
}

func (interpreter *Interpreter) VisitImportDeclaration(declaration *ast.ImportDeclaration) ast.Repr {

	resolvedLocations := interpreter.Program.Elaboration.ImportDeclarationsResolvedLocations[declaration]

	for _, resolvedLocation := range resolvedLocations {
		interpreter.importResolvedLocation(resolvedLocation)
	}

	return Done{}
}

func (interpreter *Interpreter) importResolvedLocation(resolvedLocation sema.ResolvedLocation) {

	subInterpreter := interpreter.ensureLoaded(
		resolvedLocation.Location,
		func() Import {
			return interpreter.importLocationHandler(interpreter, resolvedLocation.Location)
		},
	)

	// determine which identifiers are imported /
	// which variables need to be declared

	var variables map[string]*Variable
	identifierLength := len(resolvedLocation.Identifiers)
	if identifierLength > 0 {
		variables = make(map[string]*Variable, identifierLength)
		for _, identifier := range resolvedLocation.Identifiers {
			variables[identifier.Identifier] =
				subInterpreter.Globals[identifier.Identifier]
		}
	} else {
		variables = subInterpreter.Globals
	}

	// Gather all variable names and sort them lexicographically

	var names []string

	for name := range variables { //nolint:maprangecheck
		names = append(names, name)
	}

	// Set variables for all imported values in lexicographic order

	sort.Strings(names)

	for _, name := range names {
		variable := variables[name]

		// don't import predeclared values
		if subInterpreter.Program != nil {
			if _, ok := subInterpreter.Program.Elaboration.EffectivePredeclaredValues[name]; ok {
				continue
			}
		}

		// don't import base values
		if _, ok := sema.BaseValues.Get(name); ok {
			continue
		}

		interpreter.setVariable(name, variable)
		interpreter.Globals[name] = variable
	}
}

func (interpreter *Interpreter) VisitTransactionDeclaration(declaration *ast.TransactionDeclaration) ast.Repr {
	interpreter.declareTransactionEntryPoint(declaration)

	// NOTE: no result, so it does *not* act like a return-statement
	return Done{}
}

func (interpreter *Interpreter) declareTransactionEntryPoint(declaration *ast.TransactionDeclaration) {
	transactionType := interpreter.Program.Elaboration.TransactionDeclarationTypes[declaration]

	lexicalScope := interpreter.activations.CurrentOrNew()

	var prepareFunction *ast.FunctionDeclaration
	var prepareFunctionType *sema.FunctionType
	if declaration.Prepare != nil {
		prepareFunction = declaration.Prepare.FunctionDeclaration
		prepareFunctionType = transactionType.PrepareFunctionType().InvocationFunctionType()
	}

	var executeFunction *ast.FunctionDeclaration
	var executeFunctionType *sema.FunctionType
	if declaration.Execute != nil {
		executeFunction = declaration.Execute.FunctionDeclaration
		executeFunctionType = transactionType.ExecuteFunctionType().InvocationFunctionType()
	}

	postConditionsRewrite :=
		interpreter.Program.Elaboration.PostConditionsRewrite[declaration.PostConditions]

	self := &CompositeValue{
		Location: interpreter.Location,
		Fields:   NewStringValueOrderedMap(),
		modified: true,
	}

	transactionFunction := NewHostFunctionValue(
		func(invocation Invocation) Trampoline {
			interpreter.activations.PushNewWithParent(lexicalScope)

			invocation.Self = self
			interpreter.declareVariable(sema.SelfIdentifier, self)

			if declaration.ParameterList != nil {
				// If the transaction has a parameter list of N parameters,
				// bind the first N arguments of the invocation to the transaction parameters,
				// then leave the remaining arguments for the prepare function

				transactionParameterCount := len(declaration.ParameterList.Parameters)

				transactionArguments := invocation.Arguments[:transactionParameterCount]
				prepareArguments := invocation.Arguments[transactionParameterCount:]

				interpreter.bindParameterArguments(declaration.ParameterList, transactionArguments)
				invocation.Arguments = prepareArguments
			}

			// NOTE: get current scope instead of using `lexicalScope`,
			// because current scope has `self` declared
			transactionScope := interpreter.activations.CurrentOrNew()

			var prepareTrampoline Trampoline = Done{}
			var executeTrampoline Trampoline = Done{}

			if prepareFunction != nil {
				prepare := interpreter.functionDeclarationValue(
					prepareFunction,
					prepareFunctionType,
					transactionScope,
				)

				prepareTrampoline = More(func() Trampoline {
					return prepare.Invoke(invocation)
				})
			}

			if executeFunction != nil {
				execute := interpreter.functionDeclarationValue(
					executeFunction,
					executeFunctionType,
					transactionScope,
				)

				executeTrampoline = More(func() Trampoline {
					invocationWithoutArguments := invocation
					invocationWithoutArguments.Arguments = nil
					return execute.Invoke(invocationWithoutArguments)
				})
			}

			var preConditions ast.Conditions
			if declaration.PreConditions != nil {
				preConditions = *declaration.PreConditions
			}

			return prepareTrampoline.
				FlatMap(func(_ interface{}) Trampoline {
					return interpreter.visitFunctionBody(
						postConditionsRewrite.BeforeStatements,
						preConditions,
						executeTrampoline,
						postConditionsRewrite.RewrittenPostConditions,
						sema.VoidType,
					)
				})
		})

	interpreter.Transactions = append(interpreter.Transactions, &transactionFunction)
}

func (interpreter *Interpreter) VisitEmitStatement(statement *ast.EmitStatement) ast.Repr {
	return statement.InvocationExpression.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			event := result.(*CompositeValue)

			eventType := interpreter.Program.Elaboration.EmitStatementEventTypes[statement]

			if interpreter.onEventEmitted == nil {
				panic(EventEmissionUnavailableError{
					LocationRange: interpreter.locationRange(statement),
				})
			}

			err := interpreter.onEventEmitted(interpreter, event, eventType)
			if err != nil {
				panic(err)
			}

			// NOTE: no result, so it does *not* act like a return-statement
			return Done{}
		})
}

func (interpreter *Interpreter) VisitCastingExpression(expression *ast.CastingExpression) ast.Repr {
	return expression.Expression.Accept(interpreter).(Trampoline).
		Map(func(result interface{}) interface{} {
			value := result.(Value)

			expectedType := interpreter.Program.Elaboration.CastingTargetTypes[expression]

			switch expression.Operation {
			case ast.OperationFailableCast, ast.OperationForceCast:
				dynamicType := value.DynamicType(interpreter)
				isSubType := IsSubType(dynamicType, expectedType)

				switch expression.Operation {
				case ast.OperationFailableCast:
					if !isSubType {
						return NilValue{}
					}

					return NewSomeValueOwningNonCopying(value)

				case ast.OperationForceCast:
					if !isSubType {
						panic(
							TypeMismatchError{
								ExpectedType:  expectedType,
								LocationRange: interpreter.locationRange(expression.Expression),
							},
						)
					}

					return value

				default:
					panic(errors.NewUnreachableError())
				}

			case ast.OperationCast:
				staticValueType := interpreter.Program.Elaboration.CastingStaticValueTypes[expression]
				return interpreter.convertAndBox(value, staticValueType, expectedType)

			default:
				panic(errors.NewUnreachableError())
			}
		})
}

func (interpreter *Interpreter) VisitCreateExpression(expression *ast.CreateExpression) ast.Repr {
	return expression.InvocationExpression.Accept(interpreter)
}

func (interpreter *Interpreter) VisitDestroyExpression(expression *ast.DestroyExpression) ast.Repr {
	return expression.Expression.Accept(interpreter).(Trampoline).
		FlatMap(func(result interface{}) Trampoline {
			value := result.(Value)

			// TODO: optimize: only potentially used by host-functions

			locationRange := interpreter.locationRange(expression)
			return value.(DestroyableValue).Destroy(interpreter, locationRange)
		})
}

func (interpreter *Interpreter) VisitReferenceExpression(referenceExpression *ast.ReferenceExpression) ast.Repr {

	authorized := referenceExpression.Type.(*ast.ReferenceType).Authorized

	return referenceExpression.Expression.Accept(interpreter).(Trampoline).
		Map(func(result interface{}) interface{} {
			return &EphemeralReferenceValue{
				Authorized: authorized,
				Value:      result.(Value),
			}
		})
}

func (interpreter *Interpreter) VisitForceExpression(expression *ast.ForceExpression) ast.Repr {
	return expression.Expression.Accept(interpreter).(Trampoline).
		Map(func(result interface{}) interface{} {
			switch result := result.(type) {
			case *SomeValue:
				return result.Value

			case NilValue:
				panic(
					ForceNilError{
						LocationRange: interpreter.locationRange(expression.Expression),
					},
				)

			default:
				return result
			}
		})
}

func (interpreter *Interpreter) VisitPathExpression(expression *ast.PathExpression) ast.Repr {
	domain := common.PathDomainFromIdentifier(expression.Domain.Identifier)

	return Done{
		Result: PathValue{
			Domain:     domain,
			Identifier: expression.Identifier.Identifier,
		},
	}
}

func (interpreter *Interpreter) storedValueExists(storageAddress common.Address, key string) bool {
	return interpreter.storageExistenceHandler(interpreter, storageAddress, key)
}

func (interpreter *Interpreter) readStored(storageAddress common.Address, key string, deferred bool) OptionalValue {
	return interpreter.storageReadHandler(interpreter, storageAddress, key, deferred)
}

func (interpreter *Interpreter) writeStored(storageAddress common.Address, key string, value OptionalValue) {
	value.SetOwner(&storageAddress)

	interpreter.storageWriteHandler(interpreter, storageAddress, key, value)
}

type ValueConverter func(Value, *Interpreter) Value

type valueConverterDeclaration struct {
	name  string
	value ValueConverter
}

var converterDeclarations = []valueConverterDeclaration{
	{"Int", ConvertInt},
	{"UInt", ConvertUInt},
	{"Int8", ConvertInt8},
	{"Int16", ConvertInt16},
	{"Int32", ConvertInt32},
	{"Int64", ConvertInt64},
	{"Int128", ConvertInt128},
	{"Int256", ConvertInt256},
	{"UInt8", ConvertUInt8},
	{"UInt16", ConvertUInt16},
	{"UInt32", ConvertUInt32},
	{"UInt64", ConvertUInt64},
	{"UInt128", ConvertUInt128},
	{"UInt256", ConvertUInt256},
	{"Word8", ConvertWord8},
	{"Word16", ConvertWord16},
	{"Word32", ConvertWord32},
	{"Word64", ConvertWord64},
	{"Fix64", ConvertFix64},
	{"UFix64", ConvertUFix64},
	{"Address", ConvertAddress},
}

func init() {

	converterNames := make(map[string]struct{}, len(converterDeclarations))

	for _, converterDeclaration := range converterDeclarations {
		converterNames[converterDeclaration.name] = struct{}{}
	}

	for _, numberType := range sema.AllNumberTypes {

		// Only leaf number types require a converter,
		// "hierarchy" number types don't need one

		switch numberType.(type) {
		case *sema.NumberType, *sema.SignedNumberType,
			*sema.IntegerType, *sema.SignedIntegerType,
			*sema.FixedPointType, *sema.SignedFixedPointType:
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
		err := interpreter.ImportValue(
			declaration.name,
			interpreter.newConverterFunction(declaration.value),
		)
		if err != nil {
			panic(errors.NewUnreachableError())
		}
	}
}

func (interpreter *Interpreter) defineTypeFunction() {
	err := interpreter.ImportValue(
		"Type",
		NewHostFunctionValue(
			func(invocation Invocation) Trampoline {

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair == nil {
					panic(errors.NewUnreachableError())
				}

				ty := typeParameterPair.Value

				result := TypeValue{
					Type: ConvertSemaToStaticType(ty),
				}

				return Done{Result: result}
			},
		),
	)
	if err != nil {
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) newConverterFunction(converter ValueConverter) FunctionValue {
	return NewHostFunctionValue(
		func(invocation Invocation) Trampoline {
			value := invocation.Arguments[0]
			return Done{Result: converter(value, interpreter)}
		},
	)
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

		default:
			return false
		}

	case NilDynamicType:
		if _, ok := superType.(*sema.OptionalType); ok {
			return true
		}

		switch superType {
		case sema.AnyStructType, sema.AnyResourceType:
			return true

		default:
			return false
		}

	case SomeDynamicType:
		if typedSuperType, ok := superType.(*sema.OptionalType); ok {
			return IsSubType(typedSubType.InnerType, typedSuperType.Type)
		}

		switch superType {
		case sema.AnyStructType, sema.AnyResourceType:
			return true

		default:
			return false
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
		default:
			return false
		}

	case PrivatePathDynamicType:
		switch superType {
		case sema.PrivatePathType, sema.CapabilityPathType, sema.PathType, sema.AnyStructType:
			return true
		default:
			return false
		}

	case StoragePathDynamicType:
		switch superType {
		case sema.StoragePathType, sema.PathType, sema.AnyStructType:
			return true
		default:
			return false
		}

	case PublicAccountDynamicType:
		switch superType {
		case sema.PublicAccountType, sema.AnyStructType:
			return true
		default:
			return false
		}

	case AuthAccountDynamicType:
		switch superType {
		case sema.AuthAccountType, sema.AnyStructType:
			return true
		default:
			return false
		}

	case DeployedContractDynamicType:
		switch superType {
		case sema.DeployedContractType, sema.AnyStructType:
			return true
		default:
			return false
		}

	case AuthAccountContractsDynamicType:
		switch superType {
		case sema.AuthAccountContractsType, sema.AnyStructType:
			return true
		default:
			return false
		}

	case BlockDynamicType:
		switch superType {
		case sema.BlockType, sema.AnyStructType:
			return true
		default:
			return false
		}
	}

	return false
}

// storageKey returns the storage identifier with the proper prefix
// for the given path.
//
// \x1F = Information Separator One
//
func storageKey(path PathValue) string {
	return fmt.Sprintf("%s\x1F%s", path.Domain.Identifier(), path.Identifier)
}

func (interpreter *Interpreter) authAccountSaveFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Trampoline {

		value := invocation.Arguments[0]
		path := invocation.Arguments[1].(PathValue)

		address := addressValue.ToAddress()
		key := storageKey(path)

		// Prevent an overwrite

		if interpreter.storedValueExists(address, key) {
			panic(
				OverwriteError{
					Address:       addressValue,
					Path:          path,
					LocationRange: invocation.LocationRange,
				},
			)
		}

		// Write new value

		interpreter.writeStored(
			address,
			key,
			NewSomeValueOwningNonCopying(value),
		)

		return Done{Result: VoidValue{}}
	})
}

func (interpreter *Interpreter) authAccountLoadFunction(addressValue AddressValue) HostFunctionValue {
	return interpreter.authAccountReadFunction(addressValue, true)
}

func (interpreter *Interpreter) authAccountCopyFunction(addressValue AddressValue) HostFunctionValue {
	return interpreter.authAccountReadFunction(addressValue, false)
}

func (interpreter *Interpreter) authAccountReadFunction(addressValue AddressValue, clear bool) HostFunctionValue {

	return NewHostFunctionValue(func(invocation Invocation) Trampoline {

		address := addressValue.ToAddress()

		path := invocation.Arguments[0].(PathValue)
		key := storageKey(path)

		value := interpreter.readStored(address, key, false)

		switch value := value.(type) {
		case NilValue:
			return Done{Result: value}

		case *SomeValue:

			// If there is value stored for the given path,
			// check that it satisfies the type given as the type argument.

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := typeParameterPair.Value

			dynamicType := value.Value.DynamicType(interpreter)
			if !IsSubType(dynamicType, ty) {
				return Done{Result: NilValue{}}
			}

			if clear {
				// Remove the value from storage,
				// but only if the type check succeeded.

				interpreter.writeStored(address, key, NilValue{})
			}

			return Done{Result: value}

		default:
			panic(errors.NewUnreachableError())
		}
	})
}

func (interpreter *Interpreter) authAccountBorrowFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Trampoline {

		address := addressValue.ToAddress()

		path := invocation.Arguments[0].(PathValue)
		key := storageKey(path)

		value := interpreter.readStored(address, key, false)

		switch value := value.(type) {
		case NilValue:
			return Done{Result: value}

		case *SomeValue:

			// If there is value stored for the given path,
			// check that it satisfies the type given as the type argument.

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := typeParameterPair.Value

			referenceType := ty.(*sema.ReferenceType)

			dynamicType := value.Value.DynamicType(interpreter)
			if !IsSubType(dynamicType, referenceType.Type) {
				return Done{Result: NilValue{}}
			}

			reference := &StorageReferenceValue{
				Authorized:           referenceType.Authorized,
				TargetStorageAddress: address,
				TargetKey:            key,
			}

			return Done{Result: NewSomeValueOwningNonCopying(reference)}

		default:
			panic(errors.NewUnreachableError())
		}
	})
}

func (interpreter *Interpreter) authAccountLinkFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Trampoline {

		address := addressValue.ToAddress()

		typeParameterPair := invocation.TypeParameterTypes.Oldest()
		if typeParameterPair == nil {
			panic(errors.NewUnreachableError())
		}

		borrowType := typeParameterPair.Value.(*sema.ReferenceType)

		newCapabilityPath := invocation.Arguments[0].(PathValue)
		targetPath := invocation.Arguments[1].(PathValue)

		newCapabilityKey := storageKey(newCapabilityPath)

		if interpreter.storedValueExists(address, newCapabilityKey) {
			return Done{Result: NilValue{}}
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

		returnValue := NewSomeValueOwningNonCopying(
			CapabilityValue{
				Address:    addressValue,
				Path:       targetPath,
				BorrowType: borrowStaticType(borrowType),
			},
		)

		return Done{Result: returnValue}
	})
}

func (interpreter *Interpreter) accountGetLinkTargetFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Trampoline {

		address := addressValue.ToAddress()

		capabilityPath := invocation.Arguments[0].(PathValue)

		capabilityKey := storageKey(capabilityPath)

		value := interpreter.readStored(address, capabilityKey, false)

		switch value := value.(type) {
		case NilValue:
			return Done{Result: value}

		case *SomeValue:

			link, ok := value.Value.(LinkValue)
			if !ok {
				return Done{Result: NilValue{}}
			}

			returnValue := NewSomeValueOwningNonCopying(link.TargetPath)

			return Done{Result: returnValue}

		default:
			panic(errors.NewUnreachableError())
		}
	})
}

func (interpreter *Interpreter) authAccountUnlinkFunction(addressValue AddressValue) HostFunctionValue {
	return NewHostFunctionValue(func(invocation Invocation) Trampoline {

		address := addressValue.ToAddress()

		capabilityPath := invocation.Arguments[0].(PathValue)
		capabilityKey := storageKey(capabilityPath)

		// Write new value

		interpreter.writeStored(
			address,
			capabilityKey,
			NilValue{},
		)

		return Done{Result: VoidValue{}}
	})
}

func (interpreter *Interpreter) capabilityBorrowFunction(
	addressValue AddressValue,
	pathValue PathValue,
	borrowType *sema.ReferenceType,
) HostFunctionValue {

	return NewHostFunctionValue(
		func(invocation Invocation) Trampoline {

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

			targetStorageKey, authorized :=
				interpreter.getCapabilityFinalTargetStorageKey(
					addressValue,
					pathValue,
					borrowType,
					invocation.LocationRange,
				)

			if targetStorageKey == "" {
				return Done{Result: NilValue{}}
			}

			address := addressValue.ToAddress()

			reference := &StorageReferenceValue{
				Authorized:           authorized,
				TargetStorageAddress: address,
				TargetKey:            targetStorageKey,
			}

			return Done{Result: NewSomeValueOwningNonCopying(reference)}
		},
	)
}

func (interpreter *Interpreter) capabilityCheckFunction(
	addressValue AddressValue,
	pathValue PathValue,
	borrowType *sema.ReferenceType,
) HostFunctionValue {

	return NewHostFunctionValue(
		func(invocation Invocation) Trampoline {

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

			targetStorageKey, _ :=
				interpreter.getCapabilityFinalTargetStorageKey(
					addressValue,
					pathValue,
					borrowType,
					invocation.LocationRange,
				)

			isValid := targetStorageKey != ""

			return Done{Result: BoolValue(isValid)}
		},
	)
}

func (interpreter *Interpreter) getCapabilityFinalTargetStorageKey(
	addressValue AddressValue,
	path PathValue,
	wantedBorrowType *sema.ReferenceType,
	locationRange LocationRange,
) (
	finalStorageKey string,
	authorized bool,
) {
	address := addressValue.ToAddress()

	key := storageKey(path)

	wantedReferenceType := wantedBorrowType

	seenKeys := map[string]struct{}{}
	paths := []PathValue{path}

	for {
		// Detect cyclic links

		if _, ok := seenKeys[key]; ok {
			panic(CyclicLinkError{
				Address:       addressValue,
				Paths:         paths,
				LocationRange: locationRange,
			})
		} else {
			seenKeys[key] = struct{}{}
		}

		value := interpreter.readStored(address, key, false)

		switch value := value.(type) {
		case NilValue:
			return "", false

		case *SomeValue:

			if link, ok := value.Value.(LinkValue); ok {

				allowedType := interpreter.ConvertStaticToSemaType(link.Type)

				if !sema.IsSubType(allowedType, wantedBorrowType) {
					return "", false
				}

				targetPath := link.TargetPath
				paths = append(paths, targetPath)
				key = storageKey(targetPath)

			} else {
				return key, wantedReferenceType.Authorized
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
func (interpreter *Interpreter) getMember(self Value, locationRange LocationRange, identifier string) Value {
	var result Value
	// When the accessed value has a type that supports the declaration of members
	// or is a built-in type that has members (`MemberAccessibleValue`),
	// then try to get the member for the given identifier.
	// For example, the built-in type `String` has a member "length",
	// and composite declarations may contain member declarations
	if memberAccessibleValue, ok := self.(MemberAccessibleValue); ok {
		result = memberAccessibleValue.GetMember(interpreter, locationRange, identifier)
	}
	if result == nil {
		switch identifier {
		case sema.IsInstanceFunctionName:
			return interpreter.isInstanceFunction(self)
		case sema.GetTypeFunctionName:
			return interpreter.getTypeFunction(self)
		}
	}
	if result == nil {
		panic(errors.NewUnreachableError())
	}
	return result
}

func (interpreter *Interpreter) isInstanceFunction(self Value) HostFunctionValue {
	return NewHostFunctionValue(
		func(invocation Invocation) Trampoline {
			firstArgument := invocation.Arguments[0]
			typeValue := firstArgument.(TypeValue)

			staticType := typeValue.Type

			// Values are never instances of unknown types
			if staticType == nil {
				return Done{Result: BoolValue(false)}
			}

			semaType := interpreter.ConvertStaticToSemaType(staticType)
			// NOTE: not invocation.Self, as that is only set for composite values
			dynamicType := self.DynamicType(interpreter)
			result := IsSubType(dynamicType, semaType)
			return Done{Result: BoolValue(result)}
		},
	)
}

func (interpreter *Interpreter) getTypeFunction(self Value) HostFunctionValue {
	return NewHostFunctionValue(
		func(invocation Invocation) Trampoline {
			result := TypeValue{
				Type: self.StaticType(),
			}
			return Done{Result: result}
		},
	)
}

func (interpreter *Interpreter) setMember(self Value, locationRange LocationRange, identifier string, value Value) {
	self.(MemberAccessibleValue).SetMember(interpreter, locationRange, identifier, value)
}
