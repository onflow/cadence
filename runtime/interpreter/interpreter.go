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
	goErrors "errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/runtime/activations"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

//

var emptyImpureFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityImpure,
	nil,
	sema.VoidTypeAnnotation,
)

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
type OnEventEmittedFunc func(
	inter *Interpreter,
	locationRange LocationRange,
	event *CompositeValue,
	eventType *sema.CompositeType,
) error

// OnStatementFunc is a function that is triggered when a statement is about to be executed.
type OnStatementFunc func(
	inter *Interpreter,
	statement ast.Statement,
)

// OnLoopIterationFunc is a function that is triggered when a loop iteration is about to be executed.
type OnLoopIterationFunc func(
	inter *Interpreter,
	line int,
)

// OnFunctionInvocationFunc is a function that is triggered when a function is about to be invoked.
type OnFunctionInvocationFunc func(inter *Interpreter)

// OnInvokedFunctionReturnFunc is a function that is triggered when an invoked function returned.
type OnInvokedFunctionReturnFunc func(inter *Interpreter)

// OnRecordTraceFunc is a function that records a trace.
type OnRecordTraceFunc func(
	inter *Interpreter,
	operationName string,
	duration time.Duration,
	attrs []attribute.KeyValue,
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

// OnAccountLinkedFunc is a function that is triggered when an account is linked by the program.
type OnAccountLinkedFunc func(
	inter *Interpreter,
	locationRange LocationRange,
	address AddressValue,
	path PathValue,
) error

// InjectedCompositeFieldsHandlerFunc is a function that handles storage reads.
type InjectedCompositeFieldsHandlerFunc func(
	inter *Interpreter,
	location common.Location,
	qualifiedIdentifier string,
	compositeKind common.CompositeKind,
) map[string]Value

// ContractValueHandlerFunc is a function that handles contract values.
type ContractValueHandlerFunc func(
	inter *Interpreter,
	compositeType *sema.CompositeType,
	constructorGenerator func(common.Address) *HostFunctionValue,
	invocationRange ast.Range,
) ContractValue

// ImportLocationHandlerFunc is a function that handles imports of locations.
type ImportLocationHandlerFunc func(
	inter *Interpreter,
	location common.Location,
) Import

// AuthAccountHandlerFunc is a function that handles retrieving an auth account at a given address.
// The account returned must be of type `AuthAccount`.
type AuthAccountHandlerFunc func(
	address AddressValue,
) Value

// PublicAccountHandlerFunc is a function that handles retrieving a public account at a given address.
// The account returned must be of type `PublicAccount`.
type PublicAccountHandlerFunc func(
	address AddressValue,
) Value

// UUIDHandlerFunc is a function that handles the generation of UUIDs.
type UUIDHandlerFunc func() (uint64, error)

// CompositeTypeCode contains the "prepared" / "callable" "code"
// for the functions and the destructor of a composite
// (contract, struct, resource, event).
//
// As there is no support for inheritance of concrete types,
// these are the "leaf" nodes in the call chain, and are functions.
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
type WrapperCode struct {
	InitializerFunctionWrapper FunctionWrapper
	DestructorFunctionWrapper  FunctionWrapper
	FunctionWrappers           map[string]FunctionWrapper
	Functions                  map[string]FunctionValue
}

// TypeCodes is the value which stores the "prepared" / "callable" "code"
// of all composite types, interface types, and type requirements.
type TypeCodes struct {
	CompositeCodes       map[sema.TypeID]CompositeTypeCode
	InterfaceCodes       map[sema.TypeID]WrapperCode
	TypeRequirementCodes map[sema.TypeID]WrapperCode
}

func (c TypeCodes) Merge(codes TypeCodes) {

	// Iterating over the maps in a non-deterministic way is OK,
	// we only copy the values over.

	for typeID, code := range codes.CompositeCodes { //nolint:maprange
		c.CompositeCodes[typeID] = code
	}

	for typeID, code := range codes.InterfaceCodes { //nolint:maprange
		c.InterfaceCodes[typeID] = code
	}

	for typeID, code := range codes.TypeRequirementCodes { //nolint:maprange
		c.TypeRequirementCodes[typeID] = code
	}
}

type Storage interface {
	atree.SlabStorage
	GetStorageMap(address common.Address, domain string, createIfNotExists bool) *StorageMap
	CheckHealth() error
}

type ReferencedResourceKindedValues map[atree.StorageID]map[ReferenceTrackedResourceKindedValue]struct{}

type Interpreter struct {
	Location     common.Location
	statement    ast.Statement
	Program      *Program
	SharedState  *SharedState
	Globals      GlobalVariables
	activations  *VariableActivations
	Transactions []*HostFunctionValue
	interpreted  bool
}

var _ common.MemoryGauge = &Interpreter{}
var _ ast.DeclarationVisitor[StatementResult] = &Interpreter{}
var _ ast.StatementVisitor[StatementResult] = &Interpreter{}
var _ ast.ExpressionVisitor[Value] = &Interpreter{}

// BaseActivation is the activation which contains all base declarations.
// It is reused across all interpreters.
var BaseActivation = func() *VariableActivation {
	// No need to meter since this is only created once
	activation := activations.NewActivation[*Variable](nil, nil)

	defineBaseFunctions(activation)
	return activation
}()

func NewInterpreter(
	program *Program,
	location common.Location,
	config *Config,
) (*Interpreter, error) {
	return NewInterpreterWithSharedState(
		program,
		location,
		NewSharedState(config),
	)
}

func NewInterpreterWithSharedState(
	program *Program,
	location common.Location,
	sharedState *SharedState,
) (*Interpreter, error) {

	interpreter := &Interpreter{
		Program:     program,
		Location:    location,
		SharedState: sharedState,
	}

	// Register self
	if location != nil {
		sharedState.allInterpreters[location] = interpreter
	}

	interpreter.activations = activations.NewActivations[*Variable](interpreter)

	baseActivation := sharedState.Config.BaseActivation
	if baseActivation == nil {
		baseActivation = BaseActivation
	}

	interpreter.activations.PushNewWithParent(baseActivation)

	return interpreter, nil
}

func (interpreter *Interpreter) FindVariable(name string) *Variable {
	return interpreter.activations.Find(name)
}

func (interpreter *Interpreter) findOrDeclareVariable(name string) *Variable {
	variable := interpreter.FindVariable(name)
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
		interpreter.VisitProgram(interpreter.Program.Program)
	}

	interpreter.interpreted = true

	return nil
}

// visitGlobalDeclaration firsts interprets the global declaration,
// then finds the declaration and adds it to the globals
func (interpreter *Interpreter) visitGlobalDeclaration(declaration ast.Declaration) {
	ast.AcceptDeclaration[StatementResult](declaration, interpreter)
	interpreter.declareGlobal(declaration)
}

func (interpreter *Interpreter) declareGlobal(declaration ast.Declaration) {
	identifier := declaration.DeclarationIdentifier()
	if identifier == nil {
		return
	}
	name := identifier.Identifier
	// NOTE: semantic analysis already checked possible invalid redeclaration
	interpreter.Globals.Set(name, interpreter.FindVariable(name))
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
	variable := interpreter.Globals.Get(functionName)
	if variable == nil {
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

	functionVariable, ok := interpreter.Program.Elaboration.GetGlobalValue(functionName)
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

	return interpreter.InvokeExternally(functionValue, functionType, arguments)
}

func (interpreter *Interpreter) InvokeExternally(
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

	locationRange := EmptyLocationRange

	var preparedArguments []Value
	if argumentCount > 0 {
		preparedArguments = make([]Value, argumentCount)
		for i, argument := range arguments {
			parameterType := parameters[i].TypeAnnotation.Type

			// converts the argument into the parameter type declared by the function
			preparedArguments[i] = interpreter.ConvertAndBox(locationRange, argument, nil, parameterType)
		}
	}

	var self *MemberAccessibleValue
	var base *EphemeralReferenceValue
	if boundFunc, ok := functionValue.(BoundFunctionValue); ok {
		self = boundFunc.Self
		base = boundFunc.Base
	}

	// NOTE: can't fill argument types, as they are unknown
	invocation := NewInvocation(
		interpreter,
		self,
		base,
		preparedArguments,
		nil,
		nil,
		locationRange,
	)

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

	if index >= len(interpreter.Transactions) {
		return TransactionNotDeclaredError{Index: index}
	}

	functionValue := interpreter.Transactions[index]

	transactionType := interpreter.Program.Elaboration.TransactionTypes[index]
	functionType := transactionType.EntryPointFunctionType()

	_, err = interpreter.InvokeExternally(functionValue, functionType, arguments)
	return err
}

func (interpreter *Interpreter) RecoverErrors(onError func(error)) {
	if r := recover(); r != nil {
		var err error

		// Recover all errors, because interpreter can be directly invoked by FVM.
		switch r := r.(type) {
		case Error,
			errors.ExternalError,
			errors.InternalError,
			errors.UserError:
			err = r.(error)
		case error:
			err = errors.NewUnexpectedErrorFromCause(r)
		default:
			err = errors.NewUnexpectedError("%s", r)
		}

		// if the error is not yet an interpreter error, wrap it
		if _, ok := err.(Error); !ok {

			// wrap the error with position information if needed

			_, ok := err.(ast.HasPosition)
			if !ok && interpreter.statement != nil {
				r := ast.NewUnmeteredRangeFromPositioned(interpreter.statement)

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

		interpreterErr := err.(Error)
		interpreterErr.StackTrace = interpreter.CallStack()

		onError(interpreterErr)
	}
}

func (interpreter *Interpreter) CallStack() []Invocation {
	return interpreter.SharedState.callStack.Invocations[:]
}

func (interpreter *Interpreter) VisitProgram(program *ast.Program) {

	for _, declaration := range program.ImportDeclarations() {
		interpreter.visitGlobalDeclaration(declaration)
	}

	for _, declaration := range program.InterfaceDeclarations() {
		interpreter.visitGlobalDeclaration(declaration)
	}

	for _, declaration := range program.CompositeDeclarations() {
		interpreter.visitGlobalDeclaration(declaration)
	}

	for _, declaration := range program.AttachmentDeclarations() {
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

	var variableDeclarationVariables []*Variable

	variableDeclarationCount := len(program.VariableDeclarations())
	if variableDeclarationCount > 0 {
		variableDeclarationVariables = make([]*Variable, 0, variableDeclarationCount)

		for _, declaration := range program.VariableDeclarations() {

			// Rebind declaration, so the closure captures to current iteration's value,
			// i.e. the next iteration doesn't override `declaration`

			declaration := declaration

			identifier := declaration.Identifier.Identifier

			var variable *Variable

			variable = NewVariableWithGetter(interpreter, func() Value {
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
	}

	// Second, force the evaluation of all variable declarations,
	// in the order they were declared.

	for _, variable := range variableDeclarationVariables {
		_ = variable.GetValue()
	}
}

func (interpreter *Interpreter) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) StatementResult {
	return interpreter.VisitFunctionDeclaration(declaration.FunctionDeclaration)
}

func (interpreter *Interpreter) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) StatementResult {

	identifier := declaration.Identifier.Identifier

	functionType := interpreter.Program.Elaboration.FunctionDeclarationFunctionType(declaration)

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
			interpreter.Program.Elaboration.PostConditionsRewrite(declaration.FunctionBlock.PostConditions)

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

func (interpreter *Interpreter) visitBlock(block *ast.Block) StatementResult {
	// block scope: each block gets an activation record
	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	return interpreter.visitStatements(block.Statements)
}

func (interpreter *Interpreter) visitFunctionBody(
	beforeStatements []ast.Statement,
	preConditions ast.Conditions,
	body func() StatementResult,
	postConditions ast.Conditions,
	returnType sema.Type,
) Value {

	// block scope: each function block gets an activation record
	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	result := interpreter.visitStatements(beforeStatements)
	if result, ok := result.(ReturnResult); ok {
		return result.Value
	}

	interpreter.visitConditions(preConditions)

	var returnValue Value

	if body != nil {
		result = body()
		if result, ok := result.(ReturnResult); ok {
			returnValue = result.Value
		} else {
			returnValue = Void
		}
	} else {
		returnValue = Void
	}

	// If there is a return type, declare the constant `result`.

	if returnType != sema.VoidType {
		resultValue := interpreter.resultValue(returnValue, returnType)
		interpreter.declareVariable(
			sema.ResultIdentifier,
			resultValue,
		)
	}

	interpreter.visitConditions(postConditions)

	return returnValue
}

// resultValue returns the value for the `result` constant.
// If the return type is not a resource:
//   - The constant has the same type as the return type.
//   - `result` value is the same as the return value.
//
// If the return type is a resource:
//   - The constant has the same type as a reference to the return type.
//   - `result` value is a reference to the return value.
func (interpreter *Interpreter) resultValue(returnValue Value, returnType sema.Type) Value {
	if !returnType.IsResourceType() {
		return returnValue
	}

	if optionalType, ok := returnType.(*sema.OptionalType); ok {
		switch returnValue := returnValue.(type) {
		// If this value is an optional value (T?), then transform it into an optional reference (&T)?.
		case *SomeValue:
			innerValue := NewEphemeralReferenceValue(
				interpreter,
				false,
				returnValue.value,
				optionalType.Type,
			)

			interpreter.maybeTrackReferencedResourceKindedValue(returnValue.value)
			return NewSomeValueNonCopying(interpreter, innerValue)
		case NilValue:
			return NilValue{}
		}
	}

	interpreter.maybeTrackReferencedResourceKindedValue(returnValue)
	return NewEphemeralReferenceValue(interpreter, false, returnValue, returnType)
}

func (interpreter *Interpreter) visitConditions(conditions []*ast.Condition) {
	for _, condition := range conditions {
		interpreter.visitCondition(condition)
	}
}

func (interpreter *Interpreter) visitCondition(condition *ast.Condition) {

	// Evaluate the condition as a statement, so we get position information in case of an error

	statement := ast.NewExpressionStatement(interpreter, condition.Test)

	result, ok := interpreter.evalStatement(statement).(ExpressionResult)

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
		LocationRange: LocationRange{
			Location:    interpreter.Location,
			HasPosition: condition.Test,
		},
	})
}

// declareVariable declares a variable in the latest scope
func (interpreter *Interpreter) declareVariable(identifier string, value Value) *Variable {
	// NOTE: semantic analysis already checked possible invalid redeclaration
	variable := NewVariableWithValue(interpreter, value)
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

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: position,
	}

	// If the assignment is a forced move,
	// ensure that the target is nil,
	// otherwise panic

	if transferOperation == ast.TransferOperationMoveForced {

		// If the force-move assignment is used for the initialization of a field,
		// then there is no prior value for the field, so allow missing

		const allowMissing = true

		target := getterSetter.get(allowMissing)

		if _, ok := target.(NilValue); !ok && target != nil {
			panic(ForceAssignmentToNonNilResourceError{
				LocationRange: locationRange,
			})
		}
	}

	// Finally, evaluate the value, and assign it using the setter function

	value := interpreter.evalExpression(valueExpression)

	transferredValue := interpreter.transferAndConvert(value, valueType, targetType, locationRange)

	getterSetter.set(transferredValue)
}

// NOTE: only called for top-level composite declarations
func (interpreter *Interpreter) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) StatementResult {

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	_, _ = interpreter.declareCompositeValue(declaration, lexicalScope)

	return nil
}

func (interpreter *Interpreter) VisitAttachmentDeclaration(declaration *ast.AttachmentDeclaration) StatementResult {
	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()
	_, _ = interpreter.declareAttachmentValue(declaration, lexicalScope)
	return nil
}

func (interpreter *Interpreter) declareAttachmentValue(
	declaration *ast.AttachmentDeclaration,
	lexicalScope *VariableActivation,
) (
	scope *VariableActivation,
	variable *Variable,
) {
	return interpreter.declareCompositeValue(declaration, lexicalScope)
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
func (interpreter *Interpreter) declareCompositeValue(
	declaration ast.CompositeLikeDeclaration,
	lexicalScope *VariableActivation,
) (
	scope *VariableActivation,
	variable *Variable,
) {
	if declaration.Kind() == common.CompositeKindEnum {
		return interpreter.declareEnumConstructor(declaration.(*ast.CompositeDeclaration), lexicalScope)
	} else {
		return interpreter.declareNonEnumCompositeValue(declaration, lexicalScope)
	}
}

func (interpreter *Interpreter) declareNonEnumCompositeValue(
	declaration ast.CompositeLikeDeclaration,
	lexicalScope *VariableActivation,
) (
	scope *VariableActivation,
	variable *Variable,
) {
	identifier := declaration.DeclarationIdentifier().Identifier
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

		members := declaration.DeclarationMembers()

		for _, nestedInterfaceDeclaration := range members.Interfaces() {
			predeclare(nestedInterfaceDeclaration.Identifier)
		}

		for _, nestedCompositeDeclaration := range members.Composites() {
			predeclare(nestedCompositeDeclaration.Identifier)
		}

		for _, nestedAttachmentDeclaration := range members.Attachments() {
			predeclare(nestedAttachmentDeclaration.Identifier)
		}

		for _, nestedInterfaceDeclaration := range members.Interfaces() {
			interpreter.declareInterface(nestedInterfaceDeclaration, lexicalScope)
		}

		for _, nestedCompositeDeclaration := range members.Composites() {

			// Pass the lexical scope, which has the containing composite's value declared,
			// to the nested declarations so they can refer to it, and update the lexical scope
			// so the container's functions can refer to the nested composite's value

			var nestedVariable *Variable
			lexicalScope, nestedVariable =
				interpreter.declareCompositeValue(
					nestedCompositeDeclaration,
					lexicalScope,
				)

			memberIdentifier := nestedCompositeDeclaration.Identifier.Identifier
			nestedVariables[memberIdentifier] = nestedVariable
		}

		for _, nestedAttachmentDeclaration := range members.Attachments() {

			// Pass the lexical scope, which has the containing composite's value declared,
			// to the nested declarations so they can refer to it, and update the lexical scope
			// so the container's functions can refer to the nested composite's value

			var nestedVariable *Variable
			lexicalScope, nestedVariable =
				interpreter.declareAttachmentValue(
					nestedAttachmentDeclaration,
					lexicalScope,
				)

			memberIdentifier := nestedAttachmentDeclaration.Identifier.Identifier
			nestedVariables[memberIdentifier] = nestedVariable
		}
	})()

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationType(declaration)

	constructorType := &sema.FunctionType{
		IsConstructor: true,
		Purity:        compositeType.ConstructorPurity,
		Parameters:    compositeType.ConstructorParameters,
		ReturnTypeAnnotation: sema.TypeAnnotation{
			Type: compositeType,
		},
	}

	var initializerFunction FunctionValue
	if declaration.Kind() == common.CompositeKindEvent {
		initializerFunction = NewHostFunctionValue(
			interpreter,
			constructorType,
			func(invocation Invocation) Value {
				inter := invocation.Interpreter
				locationRange := invocation.LocationRange
				self := *invocation.Self

				for i, argument := range invocation.Arguments {
					parameter := compositeType.ConstructorParameters[i]
					self.SetMember(
						inter,
						locationRange,
						parameter.Identifier,
						argument,
					)
				}
				return nil
			},
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

		// Apply default functions, if conforming type does not provide the function

		// Iterating over the map in a non-deterministic way is OK,
		// we only apply the function wrapper to each function,
		// the order does not matter.

		for name, function := range code.Functions { //nolint:maprange
			if functions[name] != nil {
				continue
			}
			if functions == nil {
				functions = map[string]FunctionValue{}
			}
			functions[name] = function
		}

		// Wrap functions

		// Iterating over the map in a non-deterministic way is OK,
		// we only apply the function wrapper to each function,
		// the order does not matter.

		for name, functionWrapper := range code.FunctionWrappers { //nolint:maprange
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

		wrapFunctions(interpreter.SharedState.typeCodes.InterfaceCodes[conformance.ID()])
	}

	typeRequirements := compositeType.TypeRequirements()

	for i := len(typeRequirements) - 1; i >= 0; i-- {
		typeRequirement := typeRequirements[i]
		wrapFunctions(interpreter.SharedState.typeCodes.TypeRequirementCodes[typeRequirement.ID()])
	}

	interpreter.SharedState.typeCodes.CompositeCodes[compositeType.ID()] = CompositeTypeCode{
		DestructorFunction: destructorFunction,
		CompositeFunctions: functions,
	}

	location := interpreter.Location

	qualifiedIdentifier := compositeType.QualifiedIdentifier()

	config := interpreter.SharedState.Config

	constructorGenerator := func(address common.Address) *HostFunctionValue {
		return NewHostFunctionValue(
			interpreter,
			constructorType,
			func(invocation Invocation) Value {

				interpreter := invocation.Interpreter

				// Check that the resource is constructed
				// in the same location as it was declared

				locationRange := invocation.LocationRange

				if compositeType.Kind == common.CompositeKindResource &&
					interpreter.Location != compositeType.Location {

					panic(ResourceConstructionError{
						CompositeType: compositeType,
						LocationRange: locationRange,
					})
				}

				// Load injected fields
				var injectedFields map[string]Value
				injectedCompositeFieldsHandler :=
					config.InjectedCompositeFieldsHandler
				if injectedCompositeFieldsHandler != nil {
					injectedFields = injectedCompositeFieldsHandler(
						interpreter,
						location,
						qualifiedIdentifier,
						declaration.Kind(),
					)
				}

				var fields []CompositeField

				if declaration.Kind() == common.CompositeKindResource {

					uuidHandler := config.UUIDHandler
					if uuidHandler == nil {
						panic(UUIDUnavailableError{
							LocationRange: locationRange,
						})
					}

					uuid, err := uuidHandler()
					if err != nil {
						panic(err)
					}

					fields = append(
						fields,
						NewCompositeField(
							interpreter,
							sema.ResourceUUIDFieldName,
							NewUInt64Value(
								interpreter,
								func() uint64 {
									return uuid
								},
							),
						),
					)
				}

				value := NewCompositeValue(
					interpreter,
					locationRange,
					location,
					qualifiedIdentifier,
					declaration.Kind(),
					fields,
					address,
				)

				value.InjectedFields = injectedFields
				value.Functions = functions
				value.Destructor = destructorFunction

				var self MemberAccessibleValue = value
				if declaration.Kind() == common.CompositeKindAttachment {
					self = NewEphemeralReferenceValue(interpreter, false, value, interpreter.MustSemaTypeOfValue(value))
					// set the base to the implicitly provided value, and remove this implicit argument from the list
					implicitArgumentPos := len(invocation.Arguments) - 1
					invocation.Base = invocation.Arguments[implicitArgumentPos].(*EphemeralReferenceValue)
					invocation.Arguments[implicitArgumentPos] = nil
					invocation.Arguments = invocation.Arguments[:implicitArgumentPos]
					invocation.ArgumentTypes[implicitArgumentPos] = nil
					invocation.ArgumentTypes = invocation.ArgumentTypes[:implicitArgumentPos]
				}
				invocation.Self = &self

				if declaration.Kind() == common.CompositeKindContract {
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
		)
	}

	// Contract declarations declare a value / instance (singleton),
	// for all other composite kinds, the constructor is declared

	if declaration.Kind() == common.CompositeKindContract {
		variable.getter = func() Value {
			positioned := ast.NewRangeFromPositioned(interpreter, declaration.DeclarationIdentifier())

			contractValue := config.ContractValueHandler(
				interpreter,
				compositeType,
				constructorGenerator,
				positioned,
			)

			contractValue.SetNestedVariables(nestedVariables)
			return contractValue
		}
	} else {
		constructor := constructorGenerator(common.ZeroAddress)
		constructor.NestedVariables = nestedVariables
		variable.SetValue(constructor)
	}

	return lexicalScope, variable
}

type EnumCase struct {
	RawValue IntegerValue
	Value    MemberAccessibleValue
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

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationType(declaration)
	qualifiedIdentifier := compositeType.QualifiedIdentifier()

	location := interpreter.Location

	intType := sema.IntType

	enumCases := declaration.Members.EnumCases()
	caseValues := make([]EnumCase, len(enumCases))

	constructorNestedVariables := map[string]*Variable{}

	for i, enumCase := range enumCases {

		// TODO: replace, avoid conversion
		rawValue := interpreter.convert(
			NewIntValueFromInt64(interpreter, int64(i)),
			intType,
			compositeType.EnumRawType,
			LocationRange{
				Location:    location,
				HasPosition: enumCase,
			},
		).(IntegerValue)

		caseValueFields := []CompositeField{
			{
				Name:  sema.EnumRawValueFieldName,
				Value: rawValue,
			},
		}

		locationRange := LocationRange{
			Location:    location,
			HasPosition: enumCase,
		}

		caseValue := NewCompositeValue(
			interpreter,
			locationRange,
			location,
			qualifiedIdentifier,
			declaration.CompositeKind,
			caseValueFields,
			common.ZeroAddress,
		)
		caseValues[i] = EnumCase{
			Value:    caseValue,
			RawValue: rawValue,
		}

		constructorNestedVariables[enumCase.Identifier.Identifier] =
			NewVariableWithValue(interpreter, caseValue)
	}

	locationRange := LocationRange{
		Location:    location,
		HasPosition: declaration,
	}

	value := EnumConstructorFunction(
		interpreter,
		locationRange,
		compositeType,
		caseValues,
		constructorNestedVariables,
	)
	variable.SetValue(value)

	return lexicalScope, variable
}

func EnumConstructorFunction(
	gauge common.MemoryGauge,
	locationRange LocationRange,
	enumType *sema.CompositeType,
	cases []EnumCase,
	nestedVariables map[string]*Variable,
) *HostFunctionValue {

	// Prepare a lookup table based on the big-endian byte representation

	lookupTable := make(map[string]Value, len(cases))

	for _, c := range cases {
		rawValueBigEndianBytes := c.RawValue.ToBigEndianBytes()
		lookupTable[string(rawValueBigEndianBytes)] = c.Value
	}

	// Prepare the constructor function which performs a lookup in the lookup table

	constructor := NewHostFunctionValue(
		gauge,
		sema.EnumConstructorType(enumType),
		func(invocation Invocation) Value {
			rawValue, ok := invocation.Arguments[0].(IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			rawValueArgumentBigEndianBytes := rawValue.ToBigEndianBytes()

			caseValue, ok := lookupTable[string(rawValueArgumentBigEndianBytes)]
			if !ok {
				return Nil
			}

			return NewSomeValueNonCopying(invocation.Interpreter, caseValue)
		},
	)

	constructor.NestedVariables = nestedVariables

	return constructor
}

func (interpreter *Interpreter) compositeInitializerFunction(
	compositeDeclaration ast.CompositeLikeDeclaration,
	lexicalScope *VariableActivation,
) *InterpretedFunctionValue {

	// TODO: support multiple overloaded initializers

	initializers := compositeDeclaration.DeclarationMembers().Initializers()
	var initializer *ast.SpecialFunctionDeclaration
	if len(initializers) == 0 {
		return nil
	}

	initializer = initializers[0]
	functionType := interpreter.Program.Elaboration.ConstructorFunctionType(initializer)

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
			interpreter.Program.Elaboration.PostConditionsRewrite(postConditions)

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
	compositeDeclaration ast.CompositeLikeDeclaration,
	lexicalScope *VariableActivation,
) *InterpretedFunctionValue {

	destructor := compositeDeclaration.DeclarationMembers().Destructor()
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
			interpreter.Program.Elaboration.PostConditionsRewrite(postConditions)

		beforeStatements = postConditionsRewrite.BeforeStatements
		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
	}

	return NewInterpretedFunctionValue(
		interpreter,
		nil,
		emptyImpureFunctionType,
		lexicalScope,
		beforeStatements,
		preConditions,
		statements,
		rewrittenPostConditions,
	)
}

func (interpreter *Interpreter) defaultFunctions(
	members *ast.Members,
	lexicalScope *VariableActivation,
) map[string]FunctionValue {

	functionDeclarations := members.Functions()
	functionCount := len(functionDeclarations)

	if functionCount == 0 {
		return nil
	}

	functions := make(map[string]FunctionValue, functionCount)

	for _, functionDeclaration := range functionDeclarations {
		name := functionDeclaration.Identifier.Identifier
		if !functionDeclaration.FunctionBlock.HasStatements() {
			continue
		}

		functions[name] = interpreter.compositeFunction(
			functionDeclaration,
			lexicalScope,
		)
	}

	return functions
}

func (interpreter *Interpreter) compositeFunctions(
	compositeDeclaration ast.CompositeLikeDeclaration,
	lexicalScope *VariableActivation,
) map[string]FunctionValue {

	functions := map[string]FunctionValue{}

	for _, functionDeclaration := range compositeDeclaration.DeclarationMembers().Functions() {
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

		functionType := interpreter.Program.Elaboration.FunctionDeclarationFunctionType(functionDeclaration)

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

	functionType := interpreter.Program.Elaboration.FunctionDeclarationFunctionType(functionDeclaration)

	var preConditions ast.Conditions

	if functionDeclaration.FunctionBlock.PreConditions != nil {
		preConditions = *functionDeclaration.FunctionBlock.PreConditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

	if functionDeclaration.FunctionBlock.PostConditions != nil {

		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite(functionDeclaration.FunctionBlock.PostConditions)

		beforeStatements = postConditionsRewrite.BeforeStatements
		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
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
		rewrittenPostConditions,
	)
}

func (interpreter *Interpreter) VisitFieldDeclaration(_ *ast.FieldDeclaration) StatementResult {
	// fields aren't interpreted
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) StatementResult {
	// enum cases aren't interpreted
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) ValueIsSubtypeOfSemaType(value Value, targetType sema.Type) bool {
	return interpreter.IsSubTypeOfSemaType(value.StaticType(interpreter), targetType)
}

func (interpreter *Interpreter) transferAndConvert(
	value Value,
	valueType, targetType sema.Type,
	locationRange LocationRange,
) Value {

	transferredValue := value.Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		false,
		nil,
	)

	result := interpreter.ConvertAndBox(
		locationRange,
		transferredValue,
		valueType,
		targetType,
	)

	// Defensively check the value's type matches the target type
	resultStaticType := result.StaticType(interpreter)

	if targetType != nil &&
		!interpreter.IsSubTypeOfSemaType(resultStaticType, targetType) {

		resultSemaType := interpreter.MustConvertStaticToSemaType(resultStaticType)

		panic(ValueTransferTypeError{
			ExpectedType:  targetType,
			ActualType:    resultSemaType,
			LocationRange: locationRange,
		})
	}

	return result
}

// ConvertAndBox converts a value to a target type, and boxes in optionals and any value, if necessary
func (interpreter *Interpreter) ConvertAndBox(
	locationRange LocationRange,
	value Value,
	valueType, targetType sema.Type,
) Value {
	value = interpreter.convert(value, valueType, targetType, locationRange)
	return interpreter.BoxOptional(locationRange, value, targetType)
}

func (interpreter *Interpreter) convert(value Value, valueType, targetType sema.Type, locationRange LocationRange) Value {
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
			return ConvertInt(interpreter, value, locationRange)
		}

	case sema.UIntType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt(interpreter, value, locationRange)
		}

	// Int*
	case sema.Int8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt8(interpreter, value, locationRange)
		}

	case sema.Int16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt16(interpreter, value, locationRange)
		}

	case sema.Int32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt32(interpreter, value, locationRange)
		}

	case sema.Int64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt64(interpreter, value, locationRange)
		}

	case sema.Int128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt128(interpreter, value, locationRange)
		}

	case sema.Int256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt256(interpreter, value, locationRange)
		}

	// UInt*
	case sema.UInt8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt8(interpreter, value, locationRange)
		}

	case sema.UInt16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt16(interpreter, value, locationRange)
		}

	case sema.UInt32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt32(interpreter, value, locationRange)
		}

	case sema.UInt64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt64(interpreter, value, locationRange)
		}

	case sema.UInt128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt128(interpreter, value, locationRange)
		}

	case sema.UInt256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt256(interpreter, value, locationRange)
		}

	// Word*
	case sema.Word8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord8(interpreter, value, locationRange)
		}

	case sema.Word16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord16(interpreter, value, locationRange)
		}

	case sema.Word32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord32(interpreter, value, locationRange)
		}

	case sema.Word64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord64(interpreter, value, locationRange)
		}

	// Fix*

	case sema.Fix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertFix64(interpreter, value, locationRange)
		}

	case sema.UFix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUFix64(interpreter, value, locationRange)
		}
	}

	switch unwrappedTargetType := unwrappedTargetType.(type) {
	case *sema.AddressType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertAddress(interpreter, value, locationRange)
		}

	case *sema.ReferenceType:
		if !valueType.Equal(unwrappedTargetType) {
			switch ref := value.(type) {
			case *EphemeralReferenceValue:
				return NewEphemeralReferenceValue(
					interpreter,
					unwrappedTargetType.Authorized,
					ref.Value,
					unwrappedTargetType.Type,
				)

			case *StorageReferenceValue:
				return NewStorageReferenceValue(
					interpreter,
					unwrappedTargetType.Authorized,
					ref.TargetStorageAddress,
					ref.TargetPath,
					unwrappedTargetType.Type,
				)

			case *AccountReferenceValue:
				return NewAccountReferenceValue(
					interpreter,
					ref.Address,
					ref.Path,
					unwrappedTargetType.Type,
				)

			default:
				panic(errors.NewUnexpectedError("unsupported reference value: %T", ref))
			}
		}
	}

	return value
}

// BoxOptional boxes a value in optionals, if necessary
func (interpreter *Interpreter) BoxOptional(
	locationRange LocationRange,
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
			inner = typedInner.InnerValue(interpreter, locationRange)

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

func (interpreter *Interpreter) Unbox(locationRange LocationRange, value Value) Value {
	for {
		some, ok := value.(*SomeValue)
		if !ok {
			return value
		}

		value = some.InnerValue(interpreter, locationRange)
	}
}

// NOTE: only called for top-level interface declarations
func (interpreter *Interpreter) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) StatementResult {

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

	interfaceType := interpreter.Program.Elaboration.InterfaceDeclarationType(declaration)
	typeID := interfaceType.ID()

	initializerFunctionWrapper := interpreter.initializerFunctionWrapper(declaration.Members, lexicalScope)
	destructorFunctionWrapper := interpreter.destructorFunctionWrapper(declaration.Members, lexicalScope)
	functionWrappers := interpreter.functionWrappers(declaration.Members, lexicalScope)
	defaultFunctions := interpreter.defaultFunctions(declaration.Members, lexicalScope)

	interpreter.SharedState.typeCodes.InterfaceCodes[typeID] = WrapperCode{
		InitializerFunctionWrapper: initializerFunctionWrapper,
		DestructorFunctionWrapper:  destructorFunctionWrapper,
		FunctionWrappers:           functionWrappers,
		Functions:                  defaultFunctions,
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

	compositeType := interpreter.Program.Elaboration.CompositeDeclarationType(declaration)
	typeID := compositeType.ID()

	initializerFunctionWrapper := interpreter.initializerFunctionWrapper(declaration.Members, lexicalScope)
	destructorFunctionWrapper := interpreter.destructorFunctionWrapper(declaration.Members, lexicalScope)
	functionWrappers := interpreter.functionWrappers(declaration.Members, lexicalScope)
	defaultFunctions := interpreter.defaultFunctions(declaration.Members, lexicalScope)

	interpreter.SharedState.typeCodes.TypeRequirementCodes[typeID] = WrapperCode{
		InitializerFunctionWrapper: initializerFunctionWrapper,
		DestructorFunctionWrapper:  destructorFunctionWrapper,
		FunctionWrappers:           functionWrappers,
		Functions:                  defaultFunctions,
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
			interpreter.Program.Elaboration.PostConditionsRewrite(declaration.FunctionBlock.PostConditions)

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
					interpreter.declareVariable(sema.SelfIdentifier, *invocation.Self)
				}
				if invocation.Base != nil {
					interpreter.declareVariable(sema.BaseIdentifier, invocation.Base)
				}

				// NOTE: The `inner` function might be nil.
				//   This is the case if the conforming type did not declare a function.

				var body func() StatementResult
				if inner != nil {
					// NOTE: It is important to wrap the invocation in a function,
					//  so the inner function isn't invoked here

					body = func() StatementResult {

						// Pre- and post-condition wrappers "re-declare" the same
						// parameters as are used in the actual body of the function,
						// see the use of bindParameterArguments at the start of this function wrapper.
						//
						// When these parameters are given resource-kinded arguments,
						// this can trick the resource analysis into believing that these
						// resources exist in multiple variables at once
						// (one for each condition wrapper + the function itself).
						//
						// This is not the case, however, as execution of the pre- and post-conditions
						// occurs strictly before and after execution of the body respectively.
						//
						// To prevent the analysis from reporting a false positive here,
						// when we enter the body of the wrapped function,
						// we invalidate any resources that were assigned to parameters by the precondition block,
						// and then restore them after execution of the wrapped function,
						// for use by the post-condition block.

						type argumentVariable struct {
							variable *Variable
							value    ResourceKindedValue
						}

						var argumentVariables []argumentVariable
						for _, argument := range invocation.Arguments {
							resourceKindedValue := interpreter.resourceForValidation(argument)
							if resourceKindedValue == nil {
								continue
							}

							argumentVariables = append(
								argumentVariables,
								argumentVariable{
									variable: interpreter.SharedState.resourceVariables[resourceKindedValue],
									value:    resourceKindedValue,
								},
							)

							interpreter.invalidateResource(resourceKindedValue)
						}

						// NOTE: It is important to actually return the value returned
						//   from the inner function, otherwise it is lost

						returnValue := inner.invoke(invocation)

						// Restore the resources which were temporarily invalidated
						// before execution of the inner function

						for _, argumentVariable := range argumentVariables {
							value := argumentVariable.value
							interpreter.invalidateResource(value)
							interpreter.SharedState.resourceVariables[value] = argumentVariable.variable
						}
						return ReturnResult{Value: returnValue}
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
			return interpreter.SharedState.Config.ImportLocationHandler(interpreter, location)
		},
	)
}

func (interpreter *Interpreter) ensureLoadedWithLocationHandler(
	location common.Location,
	loadLocation func() Import,
) *Interpreter {

	// If a sub-interpreter already exists, return it

	subInterpreter := interpreter.SharedState.allInterpreters[location]
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
			variable := NewVariableWithValue(interpreter, global.Value)
			subInterpreter.setVariable(global.Name, variable)
			subInterpreter.Globals.Set(global.Name, variable)
		}

		subInterpreter.SharedState.typeCodes.
			Merge(virtualImport.TypeCodes)

		// Virtual import does not register interpreter itself,
		// unlike InterpreterImport
		interpreter.SharedState.allInterpreters[location] = subInterpreter

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
) (
	*Interpreter,
	error,
) {
	return NewInterpreterWithSharedState(
		program,
		location,
		interpreter.SharedState,
	)
}

func (interpreter *Interpreter) storedValueExists(
	storageAddress common.Address,
	domain string,
	identifier string,
) bool {
	config := interpreter.SharedState.Config
	accountStorage := config.Storage.GetStorageMap(storageAddress, domain, false)
	if accountStorage == nil {
		return false
	}
	return accountStorage.ValueExists(identifier)
}

func (interpreter *Interpreter) ReadStored(
	storageAddress common.Address,
	domain string,
	identifier string,
) Value {
	config := interpreter.SharedState.Config
	accountStorage := config.Storage.GetStorageMap(storageAddress, domain, false)
	if accountStorage == nil {
		return nil
	}
	return accountStorage.ReadValue(interpreter, identifier)
}

func (interpreter *Interpreter) WriteStored(
	storageAddress common.Address,
	domain string,
	identifier string,
	value Value,
) {
	config := interpreter.SharedState.Config
	accountStorage := config.Storage.GetStorageMap(storageAddress, domain, true)
	accountStorage.WriteValue(interpreter, identifier, value)
	interpreter.recordStorageMutation()
}

type fromStringFunctionValue struct {
	receiverType sema.Type
	hostFunction *HostFunctionValue
}

// a function that attempts to create a Cadence value from a string, e.g. parsing a number from a string
type stringValueParser func(*Interpreter, string) OptionalValue

func newFromStringFunction(ty sema.Type, parser stringValueParser) fromStringFunctionValue {
	functionType := sema.FromStringFunctionType(ty)

	hostFunctionImpl := NewUnmeteredHostFunctionValue(
		functionType,
		func(invocation Invocation) Value {
			argument, ok := invocation.Arguments[0].(*StringValue)
			if !ok {
				// expect typechecker to catch a mismatch here
				panic(errors.NewUnreachableError())
			}
			inter := invocation.Interpreter
			return parser(inter, argument.Str)
		},
	)
	return fromStringFunctionValue{
		receiverType: ty,
		hostFunction: hostFunctionImpl,
	}
}

// default implementation for parsing a given unsigned numeric type from a string.
// the size provided by sizeInBytes is passed to strconv.ParseUint, ensuring that the parsed value fits in the target type.
// input strings must not begin with a '+' or '-'.
func unsignedIntValueParser[ValueType Value, IntType any](
	bitSize int,
	toValue func(common.MemoryGauge, func() IntType) ValueType,
	fromUInt64 func(uint64) IntType,
) stringValueParser {
	return func(interpreter *Interpreter, input string) OptionalValue {
		val, err := strconv.ParseUint(input, 10, bitSize)
		if err != nil {
			return NilOptionalValue
		}

		converted := toValue(interpreter, func() IntType {
			return fromUInt64(val)
		})
		return NewSomeValueNonCopying(interpreter, converted)
	}
}

// default implementation for parsing a given signed numeric type from a string.
// the size provided by sizeInBytes is passed to strconv.ParseUint, ensuring that the parsed value fits in the target type.
// input strings may begin with a '+' or '-'.
func signedIntValueParser[ValueType Value, IntType any](
	bitSize int,
	toValue func(common.MemoryGauge, func() IntType) ValueType,
	fromInt64 func(int64) IntType,
) stringValueParser {

	return func(interpreter *Interpreter, input string) OptionalValue {
		val, err := strconv.ParseInt(input, 10, bitSize)
		if err != nil {
			return NilOptionalValue
		}

		converted := toValue(interpreter, func() IntType {
			return fromInt64(val)
		})
		return NewSomeValueNonCopying(interpreter, converted)
	}
}

// No need to use metered constructors for values represented by big.Ints,
// since estimation is more granular than fixed-size types.
func bigIntValueParser(convert func(*big.Int) (Value, bool)) stringValueParser {
	return func(interpreter *Interpreter, input string) OptionalValue {
		literalKind := common.IntegerLiteralKindDecimal
		estimatedSize := common.OverEstimateBigIntFromString(input, literalKind)
		common.UseMemory(interpreter, common.NewBigIntMemoryUsage(estimatedSize))

		val, ok := new(big.Int).SetString(input, literalKind.Base())
		if !ok {
			return NilOptionalValue
		}

		converted, ok := convert(val)

		if !ok {
			return NilOptionalValue
		}
		return NewSomeValueNonCopying(interpreter, converted)
	}
}

// check if val is in the inclusive interval [low, high]
func inRange(val *big.Int, low *big.Int, high *big.Int) bool {
	return -1 < val.Cmp(low) && val.Cmp(high) < 1
}

func identity[T any](t T) T { return t }

var fromStringFunctionValues = func() map[string]fromStringFunctionValue {
	u64_8 := func(n uint64) uint8 { return uint8(n) }
	u64_16 := func(n uint64) uint16 { return uint16(n) }
	u64_32 := func(n uint64) uint32 { return uint32(n) }
	u64_64 := identity[uint64]

	declarations := []fromStringFunctionValue{
		// signed int values from 8 bit -> infinity
		newFromStringFunction(sema.Int8Type, signedIntValueParser(8, NewInt8Value, func(n int64) int8 {
			return int8(n)
		})),
		newFromStringFunction(sema.Int16Type, signedIntValueParser(16, NewInt16Value, func(n int64) int16 {
			return int16(n)
		})),
		newFromStringFunction(sema.Int32Type, signedIntValueParser(32, NewInt32Value, func(n int64) int32 {
			return int32(n)
		})),
		newFromStringFunction(sema.Int64Type, signedIntValueParser(64, NewInt64Value, identity[int64])),
		newFromStringFunction(sema.Int128Type, bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
			if ok = inRange(b, sema.Int128TypeMinIntBig, sema.Int128TypeMaxIntBig); ok {
				v = NewUnmeteredInt128ValueFromBigInt(b)
			}
			return
		})),
		newFromStringFunction(sema.Int256Type, bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
			if ok = inRange(b, sema.Int256TypeMinIntBig, sema.Int256TypeMaxIntBig); ok {
				v = NewUnmeteredInt256ValueFromBigInt(b)
			}
			return
		})),
		newFromStringFunction(sema.IntType, bigIntValueParser(func(b *big.Int) (Value, bool) {
			return NewUnmeteredIntValueFromBigInt(b), true
		})),

		// unsigned int values from 8 bit -> infinity
		newFromStringFunction(sema.UInt8Type, unsignedIntValueParser(8, NewUInt8Value, u64_8)),
		newFromStringFunction(sema.UInt16Type, unsignedIntValueParser(16, NewUInt16Value, u64_16)),
		newFromStringFunction(sema.UInt32Type, unsignedIntValueParser(32, NewUInt32Value, u64_32)),
		newFromStringFunction(sema.UInt64Type, unsignedIntValueParser(64, NewUInt64Value, u64_64)),
		newFromStringFunction(sema.UInt128Type, bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
			if ok = inRange(b, sema.UInt128TypeMinIntBig, sema.UInt128TypeMaxIntBig); ok {
				v = NewUnmeteredUInt128ValueFromBigInt(b)
			}
			return
		})),
		newFromStringFunction(sema.UInt256Type, bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
			if ok = inRange(b, sema.UInt256TypeMinIntBig, sema.UInt256TypeMaxIntBig); ok {
				v = NewUnmeteredUInt256ValueFromBigInt(b)
			}
			return
		})),
		newFromStringFunction(sema.UIntType, bigIntValueParser(func(b *big.Int) (Value, bool) {
			return NewUnmeteredUIntValueFromBigInt(b), true
		})),

		// machine-sized word types
		newFromStringFunction(sema.Word8Type, unsignedIntValueParser(8, NewWord8Value, u64_8)),
		newFromStringFunction(sema.Word16Type, unsignedIntValueParser(16, NewWord16Value, u64_16)),
		newFromStringFunction(sema.Word32Type, unsignedIntValueParser(32, NewWord32Value, u64_32)),
		newFromStringFunction(sema.Word64Type, unsignedIntValueParser(64, NewWord64Value, u64_64)),

		// fixed-points
		newFromStringFunction(sema.Fix64Type, func(inter *Interpreter, input string) OptionalValue {
			n, err := fixedpoint.ParseFix64(input)
			if err != nil {
				return NilOptionalValue
			}

			val := NewFix64Value(inter, n.Int64)
			return NewSomeValueNonCopying(inter, val)

		}),
		newFromStringFunction(sema.UFix64Type, func(inter *Interpreter, input string) OptionalValue {
			n, err := fixedpoint.ParseUFix64(input)
			if err != nil {
				return NilOptionalValue
			}
			val := NewUFix64Value(inter, n.Uint64)
			return NewSomeValueNonCopying(inter, val)
		}),
	}

	values := make(map[string]fromStringFunctionValue, len(declarations))
	for _, decl := range declarations {
		// index declaration by type name
		values[decl.receiverType.String()] = decl
	}

	return values
}()

type ValueConverterDeclaration struct {
	min             Value
	max             Value
	convert         func(*Interpreter, Value, LocationRange) Value
	functionType    *sema.FunctionType
	nestedVariables []struct {
		Name  string
		Value Value
	}
	name string
}

// It would be nice if return types in Go's function types would be covariant
var ConverterDeclarations = []ValueConverterDeclaration{
	{
		name:         sema.IntTypeName,
		functionType: sema.NumberConversionFunctionType(sema.IntType),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertInt(interpreter, value, locationRange)
		},
	},
	{
		name:         sema.UIntTypeName,
		functionType: sema.NumberConversionFunctionType(sema.UIntType),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertUInt(interpreter, value, locationRange)
		},
		min: NewUnmeteredUIntValueFromBigInt(sema.UIntTypeMin),
	},
	{
		name:         sema.Int8TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int8Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertInt8(interpreter, value, locationRange)
		},
		min: NewUnmeteredInt8Value(math.MinInt8),
		max: NewUnmeteredInt8Value(math.MaxInt8),
	},
	{
		name:         sema.Int16TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int16Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertInt16(interpreter, value, locationRange)
		},
		min: NewUnmeteredInt16Value(math.MinInt16),
		max: NewUnmeteredInt16Value(math.MaxInt16),
	},
	{
		name:         sema.Int32TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int32Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertInt32(interpreter, value, locationRange)
		},
		min: NewUnmeteredInt32Value(math.MinInt32),
		max: NewUnmeteredInt32Value(math.MaxInt32),
	},
	{
		name:         sema.Int64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int64Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertInt64(interpreter, value, locationRange)
		},
		min: NewUnmeteredInt64Value(math.MinInt64),
		max: NewUnmeteredInt64Value(math.MaxInt64),
	},
	{
		name:         sema.Int128TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int128Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertInt128(interpreter, value, locationRange)
		},
		min: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
		max: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
	},
	{
		name:         sema.Int256TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Int256Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertInt256(interpreter, value, locationRange)
		},
		min: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
		max: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
	},
	{
		name:         sema.UInt8TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt8Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertUInt8(interpreter, value, locationRange)
		},
		min: NewUnmeteredUInt8Value(0),
		max: NewUnmeteredUInt8Value(math.MaxUint8),
	},
	{
		name:         sema.UInt16TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt16Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertUInt16(interpreter, value, locationRange)
		},
		min: NewUnmeteredUInt16Value(0),
		max: NewUnmeteredUInt16Value(math.MaxUint16),
	},
	{
		name:         sema.UInt32TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt32Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertUInt32(interpreter, value, locationRange)
		},
		min: NewUnmeteredUInt32Value(0),
		max: NewUnmeteredUInt32Value(math.MaxUint32),
	},
	{
		name:         sema.UInt64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt64Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertUInt64(interpreter, value, locationRange)
		},
		min: NewUnmeteredUInt64Value(0),
		max: NewUnmeteredUInt64Value(math.MaxUint64),
	},
	{
		name:         sema.UInt128TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt128Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertUInt128(interpreter, value, locationRange)
		},
		min: NewUnmeteredUInt128ValueFromUint64(0),
		max: NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
	},
	{
		name:         sema.UInt256TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UInt256Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertUInt256(interpreter, value, locationRange)
		},
		min: NewUnmeteredUInt256ValueFromUint64(0),
		max: NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
	},
	{
		name:         sema.Word8TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Word8Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertWord8(interpreter, value, locationRange)
		},
		min: NewUnmeteredWord8Value(0),
		max: NewUnmeteredWord8Value(math.MaxUint8),
	},
	{
		name:         sema.Word16TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Word16Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertWord16(interpreter, value, locationRange)
		},
		min: NewUnmeteredWord16Value(0),
		max: NewUnmeteredWord16Value(math.MaxUint16),
	},
	{
		name:         sema.Word32TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Word32Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertWord32(interpreter, value, locationRange)
		},
		min: NewUnmeteredWord32Value(0),
		max: NewUnmeteredWord32Value(math.MaxUint32),
	},
	{
		name:         sema.Word64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Word64Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertWord64(interpreter, value, locationRange)
		},
		min: NewUnmeteredWord64Value(0),
		max: NewUnmeteredWord64Value(math.MaxUint64),
	},
	{
		name:         sema.Fix64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.Fix64Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertFix64(interpreter, value, locationRange)
		},
		min: NewUnmeteredFix64Value(math.MinInt64),
		max: NewUnmeteredFix64Value(math.MaxInt64),
	},
	{
		name:         sema.UFix64TypeName,
		functionType: sema.NumberConversionFunctionType(sema.UFix64Type),
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertUFix64(interpreter, value, locationRange)
		},
		min: NewUnmeteredUFix64Value(0),
		max: NewUnmeteredUFix64Value(math.MaxUint64),
	},
	{
		name:         sema.AddressTypeName,
		functionType: sema.AddressConversionFunctionType,
		convert: func(interpreter *Interpreter, value Value, locationRange LocationRange) Value {
			return ConvertAddress(interpreter, value, locationRange)
		},
		nestedVariables: []struct {
			Name  string
			Value Value
		}{{
			Name: sema.AddressTypeFromBytesFunctionName,
			Value: NewUnmeteredHostFunctionValue(
				sema.AddressConversionFunctionType,
				AddressFromBytes,
			),
		}},
	},
	{
		name:         sema.PublicPathType.Name,
		functionType: sema.PublicPathConversionFunctionType,
		convert: func(interpreter *Interpreter, value Value, _ LocationRange) Value {
			return ConvertPublicPath(interpreter, value)
		},
	},
	{
		name:         sema.PrivatePathType.Name,
		functionType: sema.PrivatePathConversionFunctionType,
		convert: func(interpreter *Interpreter, value Value, _ LocationRange) Value {
			return ConvertPrivatePath(interpreter, value)
		},
	},
	{
		name:         sema.StoragePathType.Name,
		functionType: sema.StoragePathConversionFunctionType,
		convert: func(interpreter *Interpreter, value Value, _ LocationRange) Value {
			return ConvertStoragePath(interpreter, value)
		},
	},
}

func lookupInterface(interpreter *Interpreter, typeID string) (*sema.InterfaceType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(interpreter, typeID)
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
	location, qualifiedIdentifier, err := common.DecodeTypeID(interpreter, typeID)
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

		// todo use TypeID's here?
		typeName := numberType.String()

		if _, ok := converterNames[typeName]; !ok {
			panic(fmt.Sprintf("missing converter for number type: %s", numberType))
		}

		if _, ok := fromStringFunctionValues[typeName]; !ok {
			panic(fmt.Sprintf("missing fromString implementation for number type: %s", numberType))
		}
	}

	// We assign this here because it depends on the interpreter, so this breaks the initialization cycle
	defineBaseValue(
		BaseActivation,
		"DictionaryType",
		NewUnmeteredHostFunctionValue(
			sema.DictionaryTypeFunctionType,
			dictionaryTypeFunction,
		))

	defineBaseValue(
		BaseActivation,
		"CompositeType",
		NewUnmeteredHostFunctionValue(
			sema.CompositeTypeFunctionType,
			compositeTypeFunction,
		),
	)

	defineBaseValue(
		BaseActivation,
		"InterfaceType",
		NewUnmeteredHostFunctionValue(
			sema.InterfaceTypeFunctionType,
			interfaceTypeFunction,
		),
	)

	defineBaseValue(
		BaseActivation,
		"FunctionType",
		NewUnmeteredHostFunctionValue(
			sema.FunctionTypeFunctionType,
			functionTypeFunction,
		),
	)

	defineBaseValue(
		BaseActivation,
		"RestrictedType",
		NewUnmeteredHostFunctionValue(
			sema.RestrictedTypeFunctionType,
			restrictedTypeFunction,
		),
	)
}

func dictionaryTypeFunction(invocation Invocation) Value {
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
		return Nil
	}

	return NewSomeValueNonCopying(
		invocation.Interpreter,
		NewTypeValue(
			invocation.Interpreter,
			NewDictionaryStaticType(
				invocation.Interpreter,
				keyType,
				valueType,
			),
		),
	)
}

func compositeTypeFunction(invocation Invocation) Value {
	typeIDValue, ok := invocation.Arguments[0].(*StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	typeID := typeIDValue.Str

	composite, err := lookupComposite(invocation.Interpreter, typeID)
	if err != nil {
		return Nil
	}

	return NewSomeValueNonCopying(
		invocation.Interpreter,
		NewTypeValue(
			invocation.Interpreter,
			ConvertSemaToStaticType(invocation.Interpreter, composite),
		),
	)
}

func interfaceTypeFunction(invocation Invocation) Value {
	typeIDValue, ok := invocation.Arguments[0].(*StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	typeID := typeIDValue.Str

	interfaceType, err := lookupInterface(invocation.Interpreter, typeID)
	if err != nil {
		return Nil
	}

	return NewSomeValueNonCopying(
		invocation.Interpreter,
		NewTypeValue(
			invocation.Interpreter,
			ConvertSemaToStaticType(invocation.Interpreter, interfaceType),
		),
	)
}

func functionTypeFunction(invocation Invocation) Value {
	interpreter := invocation.Interpreter

	parameters, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	typeValue, ok := invocation.Arguments[1].(TypeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	returnType := interpreter.MustConvertStaticToSemaType(typeValue.Type)

	var parameterTypes []sema.Parameter
	parameterCount := parameters.Count()
	if parameterCount > 0 {
		parameterTypes = make([]sema.Parameter, 0, parameterCount)
		parameters.Iterate(interpreter, func(param Value) bool {
			semaType := interpreter.MustConvertStaticToSemaType(param.(TypeValue).Type)
			parameterTypes = append(
				parameterTypes,
				sema.Parameter{
					TypeAnnotation: sema.NewTypeAnnotation(semaType),
				},
			)

			// Continue iteration
			return true
		})
	}
	functionStaticType := NewFunctionStaticType(
		interpreter,
		sema.NewSimpleFunctionType(
			sema.FunctionPurityImpure,
			parameterTypes,
			sema.NewTypeAnnotation(returnType),
		),
	)
	return NewUnmeteredTypeValue(functionStaticType)
}

func restrictedTypeFunction(invocation Invocation) Value {
	restrictionIDs, ok := invocation.Arguments[1].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var staticRestrictions []InterfaceStaticType
	var semaRestrictions []*sema.InterfaceType

	count := restrictionIDs.Count()
	if count > 0 {
		staticRestrictions = make([]InterfaceStaticType, 0, count)
		semaRestrictions = make([]*sema.InterfaceType, 0, count)

		var invalidRestrictionID bool
		restrictionIDs.Iterate(invocation.Interpreter, func(typeID Value) bool {
			typeIDValue, ok := typeID.(*StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			restrictionInterface, err := lookupInterface(invocation.Interpreter, typeIDValue.Str)
			if err != nil {
				invalidRestrictionID = true
				return true
			}

			staticRestrictions = append(
				staticRestrictions,
				ConvertSemaToStaticType(invocation.Interpreter, restrictionInterface).(InterfaceStaticType),
			)
			semaRestrictions = append(semaRestrictions, restrictionInterface)

			// Continue iteration
			return true
		})

		// If there are any invalid restrictions,
		// then return nil
		if invalidRestrictionID {
			return Nil
		}
	}

	var semaType sema.Type
	var err error

	switch typeID := invocation.Arguments[0].(type) {
	case NilValue:
		semaType = nil
	case *SomeValue:
		innerValue := typeID.InnerValue(invocation.Interpreter, invocation.LocationRange)
		semaType, err = lookupComposite(invocation.Interpreter, innerValue.(*StringValue).Str)
		if err != nil {
			return Nil
		}
	default:
		panic(errors.NewUnreachableError())
	}

	var invalidRestrictedType bool
	ty := sema.CheckRestrictedType(
		invocation.Interpreter,
		semaType,
		semaRestrictions,
		func(_ func(*ast.RestrictedType) error) {
			invalidRestrictedType = true
		},
	)

	// If the restricted type would have failed to type-check statically,
	// then return nil
	if invalidRestrictedType {
		return Nil
	}

	return NewSomeValueNonCopying(
		invocation.Interpreter,
		NewTypeValue(
			invocation.Interpreter,
			NewRestrictedStaticType(
				invocation.Interpreter,
				ConvertSemaToStaticType(invocation.Interpreter, ty),
				staticRestrictions,
			),
		),
	)
}

func defineBaseFunctions(activation *VariableActivation) {
	defineConverterFunctions(activation)
	defineTypeFunction(activation)
	defineRuntimeTypeConstructorFunctions(activation)
	defineStringFunction(activation)
}

type converterFunction struct {
	converter *HostFunctionValue
	name      string
}

// Converter functions are stateless functions. Hence they can be re-used across interpreters.
var converterFunctionValues = func() []converterFunction {

	converterFuncValues := make([]converterFunction, len(ConverterDeclarations))

	for index, declaration := range ConverterDeclarations {
		// NOTE: declare in loop, as captured in closure below
		convert := declaration.convert
		converterFunctionValue := NewUnmeteredHostFunctionValue(
			declaration.functionType,
			func(invocation Invocation) Value {
				return convert(invocation.Interpreter, invocation.Arguments[0], invocation.LocationRange)
			},
		)

		addMember := func(name string, value Value) {
			if converterFunctionValue.NestedVariables == nil {
				converterFunctionValue.NestedVariables = map[string]*Variable{}
			}
			// these variables are not needed to be metered as they are only ever declared once,
			// and can be considered base interpreter overhead
			converterFunctionValue.NestedVariables[name] = NewVariableWithValue(nil, value)
		}

		if declaration.min != nil {
			addMember(sema.NumberTypeMinFieldName, declaration.min)
		}

		if declaration.max != nil {
			addMember(sema.NumberTypeMaxFieldName, declaration.max)
		}

		fromStringVal := fromStringFunctionValues[declaration.name]

		addMember(sema.FromStringFunctionName, fromStringVal.hostFunction)

		if declaration.nestedVariables != nil {
			for _, variable := range declaration.nestedVariables {
				addMember(variable.Name, variable.Value)
			}
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
	converter *HostFunctionValue
	name      string
}

// Constructor functions are stateless functions. Hence they can be re-used across interpreters.
var runtimeTypeConstructors = []runtimeTypeConstructor{
	{
		name: "OptionalType",
		converter: NewUnmeteredHostFunctionValue(
			sema.OptionalTypeFunctionType,
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return NewTypeValue(
					invocation.Interpreter,
					NewOptionalStaticType(
						invocation.Interpreter,
						typeValue.Type,
					),
				)
			},
		),
	},
	{
		name: "VariableSizedArrayType",
		converter: NewUnmeteredHostFunctionValue(
			sema.VariableSizedArrayTypeFunctionType,
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return NewTypeValue(
					invocation.Interpreter,
					//nolint:gosimple
					NewVariableSizedStaticType(
						invocation.Interpreter,
						typeValue.Type,
					),
				)
			},
		),
	},
	{
		name: "ConstantSizedArrayType",
		converter: NewUnmeteredHostFunctionValue(
			sema.ConstantSizedArrayTypeFunctionType,
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				sizeValue, ok := invocation.Arguments[1].(IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return NewTypeValue(
					invocation.Interpreter,
					NewConstantSizedStaticType(
						invocation.Interpreter,
						typeValue.Type,
						int64(sizeValue.ToInt(invocation.LocationRange)),
					),
				)
			},
		),
	},
	{
		name: "ReferenceType",
		converter: NewUnmeteredHostFunctionValue(
			sema.ReferenceTypeFunctionType,
			func(invocation Invocation) Value {
				authorizedValue, ok := invocation.Arguments[0].(BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				typeValue, ok := invocation.Arguments[1].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return NewTypeValue(
					invocation.Interpreter,
					NewReferenceStaticType(
						invocation.Interpreter,
						bool(authorizedValue),
						typeValue.Type,
						nil,
					),
				)
			},
		),
	},
	{
		name: "CapabilityType",
		converter: NewUnmeteredHostFunctionValue(
			sema.CapabilityTypeFunctionType,
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				ty := typeValue.Type
				// Capabilities must hold references
				_, ok = ty.(ReferenceStaticType)
				if !ok {
					return Nil
				}

				return NewSomeValueNonCopying(
					invocation.Interpreter,
					NewTypeValue(
						invocation.Interpreter,
						NewCapabilityStaticType(
							invocation.Interpreter,
							ty,
						),
					),
				)
			},
		),
	},
}

func defineRuntimeTypeConstructorFunctions(activation *VariableActivation) {
	for _, constructorFunc := range runtimeTypeConstructors {
		defineBaseValue(activation, constructorFunc.name, constructorFunc.converter)
	}
}

// typeFunction is the `Type` function. It is stateless, hence it can be re-used across interpreters.
var typeFunction = NewUnmeteredHostFunctionValue(
	sema.MetaTypeFunctionType,
	func(invocation Invocation) Value {
		typeParameterPair := invocation.TypeParameterTypes.Oldest()
		if typeParameterPair == nil {
			panic(errors.NewUnreachableError())
		}

		ty := typeParameterPair.Value

		staticType := ConvertSemaToStaticType(invocation.Interpreter, ty)
		return NewTypeValue(invocation.Interpreter, staticType)
	},
)

func defineTypeFunction(activation *VariableActivation) {
	defineBaseValue(activation, sema.MetaTypeName, typeFunction)
}

func defineBaseValue(activation *VariableActivation, name string, value Value) {
	if activation.Find(name) != nil {
		panic(errors.NewUnreachableError())
	}
	// these variables are not needed to be metered as they are only ever declared once,
	// and can be considered base interpreter overhead
	activation.Set(name, NewVariableWithValue(nil, value))
}

func defineStringFunction(activation *VariableActivation) {
	defineBaseValue(activation, sema.StringType.String(), stringFunction)
}

func (interpreter *Interpreter) IsSubType(subType StaticType, superType StaticType) bool {
	if superType == PrimitiveStaticTypeAny {
		return true
	}

	// This is an optimization: If the static types are equal, then no need to check further.
	// i.e: Saves the conversion time.
	if subType.Equal(superType) {
		return true
	}

	semaType := interpreter.MustConvertStaticToSemaType(superType)

	return interpreter.IsSubTypeOfSemaType(subType, semaType)
}

func (interpreter *Interpreter) IsSubTypeOfSemaType(subType StaticType, superType sema.Type) bool {
	if superType == sema.AnyType {
		return true
	}

	switch subType := subType.(type) {
	case OptionalStaticType:
		if superType, ok := superType.(*sema.OptionalType); ok {
			return interpreter.IsSubTypeOfSemaType(subType.Type, superType.Type)
		}

		switch superType {
		case sema.AnyStructType, sema.AnyResourceType:
			return interpreter.IsSubTypeOfSemaType(subType.Type, superType)
		}

	case ReferenceStaticType:
		if superType, ok := superType.(*sema.ReferenceType); ok {

			// First, check that the static type of the referenced value
			// is a subtype of the super type

			if subType.ReferencedType == nil ||
				!interpreter.IsSubTypeOfSemaType(subType.ReferencedType, superType.Type) {

				return false
			}

			// If the reference value is authorized it may be downcasted

			authorized := subType.Authorized

			if authorized {
				return true
			}

			// If the reference value is not authorized,
			// it may not be down-casted

			borrowType := interpreter.MustConvertStaticToSemaType(subType.BorrowedType)

			return sema.IsSubType(
				&sema.ReferenceType{
					Authorized: authorized,
					Type:       borrowType,
				},
				superType,
			)
		}

		return superType == sema.AnyStructType
	}

	semaType := interpreter.MustConvertStaticToSemaType(subType)

	return sema.IsSubType(semaType, superType)
}

func (interpreter *Interpreter) domainPaths(address common.Address, domain common.PathDomain) []Value {
	config := interpreter.SharedState.Config
	storageMap := config.Storage.GetStorageMap(address, domain.Identifier(), false)
	if storageMap == nil {
		return []Value{}
	}
	iterator := storageMap.Iterator(interpreter)
	var values []Value

	count := storageMap.Count()
	if count > 0 {
		values = make([]Value, 0, count)
		for key := iterator.NextKey(); key != ""; key = iterator.NextKey() {
			values = append(values, NewPathValue(interpreter, domain, key))
		}
	}
	return values
}

func (interpreter *Interpreter) accountPaths(addressValue AddressValue, locationRange LocationRange, domain common.PathDomain, pathType StaticType) *ArrayValue {
	address := addressValue.ToAddress()
	values := interpreter.domainPaths(address, domain)
	return NewArrayValue(
		interpreter,
		locationRange,
		NewVariableSizedStaticType(interpreter, pathType),
		common.ZeroAddress,
		values...,
	)
}

func (interpreter *Interpreter) publicAccountPaths(addressValue AddressValue, locationRange LocationRange) *ArrayValue {
	return interpreter.accountPaths(addressValue, locationRange, common.PathDomainPublic, PrimitiveStaticTypePublicPath)
}

func (interpreter *Interpreter) privateAccountPaths(addressValue AddressValue, locationRange LocationRange) *ArrayValue {
	return interpreter.accountPaths(addressValue, locationRange, common.PathDomainPrivate, PrimitiveStaticTypePrivatePath)
}

func (interpreter *Interpreter) storageAccountPaths(addressValue AddressValue, locationRange LocationRange) *ArrayValue {
	return interpreter.accountPaths(addressValue, locationRange, common.PathDomainStorage, PrimitiveStaticTypeStoragePath)
}

func (interpreter *Interpreter) recordStorageMutation() {
	if interpreter.SharedState.inStorageIteration {
		interpreter.SharedState.storageMutatedDuringIteration = true
	}
}

func (interpreter *Interpreter) newStorageIterationFunction(
	functionType *sema.FunctionType,
	addressValue AddressValue,
	domain common.PathDomain,
	pathType sema.Type,
) *HostFunctionValue {

	address := addressValue.ToAddress()
	config := interpreter.SharedState.Config

	return NewHostFunctionValue(
		interpreter,
		functionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			fn, ok := invocation.Arguments[0].(FunctionValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			locationRange := invocation.LocationRange
			inter := invocation.Interpreter
			storageMap := config.Storage.GetStorageMap(address, domain.Identifier(), false)
			if storageMap == nil {
				// if nothing is stored, no iteration is required
				return Void
			}
			storageIterator := storageMap.Iterator(interpreter)

			invocationArgumentTypes := []sema.Type{pathType, sema.MetaType}

			inIteration := inter.SharedState.inStorageIteration
			inter.SharedState.inStorageIteration = true
			defer func() {
				inter.SharedState.inStorageIteration = inIteration
			}()

			for key, value := storageIterator.Next(); key != "" && value != nil; key, value = storageIterator.Next() {
				staticType := value.StaticType(inter)

				// Perform a forced type loading to see if the underlying type is not broken.
				// If broken, skip this value from the iteration.
				typeError := inter.checkTypeLoading(staticType)
				if typeError != nil {
					continue
				}

				pathValue := NewPathValue(inter, domain, key)
				runtimeType := NewTypeValue(inter, staticType)

				subInvocation := NewInvocation(
					inter,
					nil,
					nil,
					[]Value{pathValue, runtimeType},
					invocationArgumentTypes,
					nil,
					locationRange,
				)

				shouldContinue, ok := fn.invoke(subInvocation).(BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if !shouldContinue {
					break
				}

				// it is not safe to check this at the beginning of the loop (i.e. on the next invocation of the callback)
				// because if the mutation performed in the callback reorganized storage such that the iteration pointer is now
				// at the end, we will not invoke the callback again but will still silently skip elements of storage. In order
				// to be safe, we perform this check here to effectively enforce that users return `false` from their callback
				// in all cases where storage is mutated
				if inter.SharedState.storageMutatedDuringIteration {
					panic(StorageMutatedDuringIterationError{
						LocationRange: locationRange,
					})
				}

			}

			return Void
		},
	)
}

func (interpreter *Interpreter) checkTypeLoading(staticType StaticType) (typeError error) {
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case errors.UserError, errors.ExternalError:
				typeError = r.(error)
			default:
				panic(r)
			}
		}
	}()

	// Here it is only interested in whether the type can be properly loaded.
	_, typeError = interpreter.ConvertStaticToSemaType(staticType)

	return
}

func (interpreter *Interpreter) authAccountSaveFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		sema.AuthAccountTypeSaveFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			value := invocation.Arguments[0]

			path, ok := invocation.Arguments[1].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := path.Domain.Identifier()
			identifier := path.Identifier

			// Prevent an overwrite

			locationRange := invocation.LocationRange

			if interpreter.storedValueExists(
				address,
				domain,
				identifier,
			) {
				panic(
					OverwriteError{
						Address:       addressValue,
						Path:          path,
						LocationRange: locationRange,
					},
				)
			}

			value = value.Transfer(
				interpreter,
				locationRange,
				atree.Address(address),
				true,
				nil,
			)

			// Write new value

			interpreter.WriteStored(address, domain, identifier, value)

			return Void
		},
	)
}

func (interpreter *Interpreter) authAccountTypeFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		sema.AuthAccountTypeTypeFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			path, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := path.Domain.Identifier()
			identifier := path.Identifier

			value := interpreter.ReadStored(address, domain, identifier)

			if value == nil {
				return Nil
			}

			return NewSomeValueNonCopying(
				interpreter,
				NewTypeValue(
					interpreter,
					value.StaticType(interpreter),
				),
			)
		},
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
		// same as sema.AuthAccountTypeCopyFunctionType
		sema.AuthAccountTypeLoadFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			path, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := path.Domain.Identifier()
			identifier := path.Identifier

			value := interpreter.ReadStored(address, domain, identifier)

			if value == nil {
				return Nil
			}

			// If there is value stored for the given path,
			// check that it satisfies the type given as the type argument.

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := typeParameterPair.Value

			valueStaticType := value.StaticType(interpreter)

			if !interpreter.IsSubTypeOfSemaType(valueStaticType, ty) {
				valueSemaType := interpreter.MustConvertStaticToSemaType(valueStaticType)

				panic(ForceCastTypeMismatchError{
					ExpectedType:  ty,
					ActualType:    valueSemaType,
					LocationRange: invocation.LocationRange,
				})
			}

			locationRange := invocation.LocationRange

			// We could also pass remove=true and the storable stored in storage,
			// but passing remove=false here and writing nil below has the same effect
			// TODO: potentially refactor and get storable in storage, pass it and remove=true
			transferredValue := value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
			)

			// Remove the value from storage,
			// but only if the type check succeeded.
			if clear {
				interpreter.WriteStored(address, domain, identifier, nil)
			}

			return NewSomeValueNonCopying(invocation.Interpreter, transferredValue)
		},
	)
}

func (interpreter *Interpreter) authAccountBorrowFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		sema.AuthAccountTypeBorrowFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

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
				interpreter,
				referenceType.Authorized,
				address,
				path,
				referenceType.Type,
			)

			// Attempt to dereference,
			// which reads the stored value
			// and performs a dynamic type check

			value, err := reference.dereference(interpreter, invocation.LocationRange)
			if err != nil {
				panic(err)
			}
			if value == nil {
				return Nil
			}

			return NewSomeValueNonCopying(interpreter, reference)
		},
	)
}

func (interpreter *Interpreter) authAccountLinkFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		sema.AuthAccountTypeLinkFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

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
				return Nil
			}

			// Write new value

			borrowStaticType := ConvertSemaToStaticType(interpreter, borrowType)

			// Note that this will be metered twice if Atree validation is enabled.
			pathLink := NewPathLinkValue(interpreter, targetPath, borrowStaticType)

			interpreter.WriteStored(
				address,
				newCapabilityDomain,
				newCapabilityIdentifier,
				pathLink,
			)

			return NewSomeValueNonCopying(
				interpreter,
				NewStorageCapabilityValue(
					interpreter,
					addressValue,
					newCapabilityPath,
					borrowStaticType,
				),
			)

		},
	)
}

var authAccountReferenceStaticType = ReferenceStaticType{
	BorrowedType:   PrimitiveStaticTypeAuthAccount,
	ReferencedType: PrimitiveStaticTypeAuthAccount,
}

// Linking
//
// When linking to a path with the `AuthAccount.link function`,
// an interpreter.PathLink (formerly Link) is stored in storage.
//
// When linking to an account with the new AuthAccount.linkAccount function,
// an interpreter.AccountLink is stored in the account.
//
// In both cases, when acquiring a capability, e.g. using getCapability,
// a StorageCapabilityValue is returned.
// This is because in both cases, we are looking up a path in an account.
// Depending on what is stored in the path, PathLink or AccountLink,
// we return a respective reference value, a StorageReferenceValue for PathLink
// (after following the links to the final target),
// or an AccountReferenceValue for an AccountLink.
//
// Again, in both cases for StorageReferenceValue and AccountReferenceValue,
// for each use, e.g. member access,
// we dereference/check that the link still exists after the capability was borrowed.

func (interpreter *Interpreter) authAccountLinkAccountFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		sema.AuthAccountTypeLinkAccountFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			if !interpreter.SharedState.Config.AccountLinkingAllowed {
				panic(AccountLinkingForbiddenError{
					LocationRange: invocation.LocationRange,
				})
			}

			newCapabilityPath, ok := invocation.Arguments[0].(PathValue)
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
				return Nil
			}

			accountLinkValue := NewAccountLinkValue(interpreter)

			interpreter.WriteStored(
				address,
				newCapabilityDomain,
				newCapabilityIdentifier,
				accountLinkValue,
			)

			onAccountLinked := interpreter.SharedState.Config.OnAccountLinked
			if onAccountLinked != nil {
				err := onAccountLinked(
					interpreter,
					invocation.LocationRange,
					addressValue,
					newCapabilityPath,
				)
				if err != nil {
					panic(err)
				}
			}

			return NewSomeValueNonCopying(
				interpreter,
				NewStorageCapabilityValue(
					interpreter,
					addressValue,
					newCapabilityPath,
					authAccountReferenceStaticType,
				),
			)

		},
	)
}

func (interpreter *Interpreter) accountGetLinkTargetFunction(
	functionType *sema.FunctionType,
	addressValue AddressValue,
) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		functionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			capabilityPath, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := capabilityPath.Domain.Identifier()
			identifier := capabilityPath.Identifier

			value := interpreter.ReadStored(address, domain, identifier)

			if value == nil {
				return Nil
			}

			link, ok := value.(PathLinkValue)
			if !ok {
				return Nil
			}

			return NewSomeValueNonCopying(
				interpreter,
				link.TargetPath,
			)
		},
	)
}

func (interpreter *Interpreter) authAccountUnlinkFunction(addressValue AddressValue) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		sema.AuthAccountTypeUnlinkFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			capabilityPath, ok := invocation.Arguments[0].(PathValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			domain := capabilityPath.Domain.Identifier()
			identifier := capabilityPath.Identifier

			// Write new value

			interpreter.WriteStored(address, domain, identifier, nil)

			return Void
		},
	)
}

func (interpreter *Interpreter) storageCapabilityBorrowFunction(
	addressValue AddressValue,
	pathValue PathValue,
	borrowType *sema.ReferenceType,
) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		sema.CapabilityTypeBorrowFunctionType(borrowType),
		func(invocation Invocation) Value {

			interpreter := invocation.Interpreter

			// NOTE: if a type argument is provided for the function,
			// use it *instead* of the type of the value (if any)

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair != nil {
				ty := typeParameterPair.Value
				var ok bool
				borrowType, ok = ty.(*sema.ReferenceType)
				if !ok {
					panic(errors.NewUnreachableError())
				}
			}

			if borrowType == nil {
				panic(errors.NewUnreachableError())
			}

			target, authorized, err :=
				interpreter.GetStorageCapabilityFinalTarget(
					address,
					pathValue,
					borrowType,
					invocation.LocationRange,
				)
			if err != nil {
				panic(err)
			}

			if target == nil {
				return Nil
			}

			switch target := target.(type) {
			case AccountCapabilityTarget:
				return NewSomeValueNonCopying(
					interpreter,
					NewAccountReferenceValue(
						interpreter,
						address,
						pathValue,
						borrowType.Type,
					),
				)

			case PathCapabilityTarget:
				targetPath := PathValue(target)

				reference := NewStorageReferenceValue(
					interpreter,
					authorized,
					address,
					targetPath,
					borrowType.Type,
				)

				// Attempt to dereference,
				// which reads the stored value
				// and performs a dynamic type check

				value, err := reference.dereference(interpreter, invocation.LocationRange)
				if err != nil {
					panic(err)
				}
				if value == nil {
					return Nil
				}

				return NewSomeValueNonCopying(interpreter, reference)

			default:
				panic(errors.NewUnreachableError())
			}
		},
	)
}

func (interpreter *Interpreter) storageCapabilityCheckFunction(
	addressValue AddressValue,
	pathValue PathValue,
	borrowType *sema.ReferenceType,
) *HostFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewHostFunctionValue(
		interpreter,
		sema.CapabilityTypeCheckFunctionType(borrowType),
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			// NOTE: if a type argument is provided for the function,
			// use it *instead* of the type of the value (if any)

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair != nil {
				ty := typeParameterPair.Value
				var ok bool
				borrowType, ok = ty.(*sema.ReferenceType)
				if !ok {
					panic(errors.NewUnreachableError())
				}
			}

			if borrowType == nil {
				panic(errors.NewUnreachableError())
			}

			target, authorized, err :=
				interpreter.GetStorageCapabilityFinalTarget(
					address,
					pathValue,
					borrowType,
					invocation.LocationRange,
				)
			if err != nil {
				panic(err)
			}

			if target == nil {
				return FalseValue
			}

			switch target := target.(type) {
			case AccountCapabilityTarget:
				return TrueValue

			case PathCapabilityTarget:
				targetPath := PathValue(target)

				reference := NewStorageReferenceValue(
					interpreter,
					authorized,
					address,
					targetPath,
					borrowType.Type,
				)

				// Attempt to dereference,
				// which reads the stored value
				// and performs a dynamic type check

				return AsBoolValue(
					reference.ReferencedValue(interpreter, invocation.LocationRange, false) != nil,
				)

			default:
				panic(errors.NewUnreachableError())
			}
		},
	)
}

func (interpreter *Interpreter) GetStorageCapabilityFinalTarget(
	address common.Address,
	path PathValue,
	wantedBorrowType *sema.ReferenceType,
	locationRange LocationRange,
) (
	target CapabilityTarget,
	authorized bool,
	err error,
) {
	wantedReferenceType := wantedBorrowType

	seenPaths := map[PathValue]struct{}{}
	paths := []PathValue{path}

	for {
		// Detect cyclic links

		if _, ok := seenPaths[path]; ok {
			return nil, false, CyclicLinkError{
				Address:       address,
				Paths:         paths,
				LocationRange: locationRange,
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
			return nil, false, nil
		}

		switch value := value.(type) {
		case PathLinkValue:
			allowedType := interpreter.MustConvertStaticToSemaType(value.Type)

			if !sema.IsSubType(allowedType, wantedBorrowType) {
				return nil, false, nil
			}

			targetPath := value.TargetPath
			paths = append(paths, targetPath)
			path = targetPath

		case AccountLinkValue:
			if !interpreter.IsSubTypeOfSemaType(
				authAccountReferenceStaticType,
				wantedBorrowType,
			) {
				return nil, false, nil
			}

			return AccountCapabilityTarget(address),
				false,
				nil

		default:
			return PathCapabilityTarget(path),
				wantedReferenceType.Authorized,
				nil
		}
	}
}

func (interpreter *Interpreter) ConvertStaticToSemaType(staticType StaticType) (sema.Type, error) {
	config := interpreter.SharedState.Config
	return ConvertStaticToSemaType(
		config.MemoryGauge,
		staticType,
		func(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error) {
			return interpreter.getInterfaceType(location, qualifiedIdentifier)
		},
		func(location common.Location, qualifiedIdentifier string, typeID common.TypeID) (*sema.CompositeType, error) {
			return interpreter.GetCompositeType(location, qualifiedIdentifier, typeID)
		},
	)
}

func (interpreter *Interpreter) MustSemaTypeOfValue(value Value) sema.Type {
	return interpreter.MustConvertStaticToSemaType(value.StaticType(interpreter))
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

	subInterpreter := inter.SharedState.allInterpreters[location]
	if subInterpreter == nil || subInterpreter.Program == nil {
		return nil
	}

	return subInterpreter.Program.Elaboration
}

// GetContractComposite gets the composite value of the contract at the address location.
func (interpreter *Interpreter) GetContractComposite(contractLocation common.AddressLocation) (*CompositeValue, error) {
	contractGlobal := interpreter.Globals.Get(contractLocation.Name)
	if contractGlobal == nil {
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

	ty := elaboration.CompositeType(typeID)
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
		return nil, TypeLoadingError{
			TypeID: common.TypeID(qualifiedIdentifier),
		}
	}

	return ty, nil
}

func (interpreter *Interpreter) getInterfaceType(location common.Location, qualifiedIdentifier string) (*sema.InterfaceType, error) {
	if location == nil {
		return nil, InterfaceMissingLocationError{QualifiedIdentifier: qualifiedIdentifier}
	}

	typeID := location.TypeID(interpreter, qualifiedIdentifier)

	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	ty := elaboration.InterfaceType(typeID)
	if ty == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	return ty, nil
}

func (interpreter *Interpreter) reportLoopIteration(pos ast.HasPosition) {
	config := interpreter.SharedState.Config

	onMeterComputation := config.OnMeterComputation
	if onMeterComputation != nil {
		onMeterComputation(common.ComputationKindLoop, 1)
	}

	onLoopIteration := config.OnLoopIteration
	if onLoopIteration != nil {
		line := pos.StartPosition().Line
		onLoopIteration(interpreter, line)
	}
}

func (interpreter *Interpreter) reportFunctionInvocation() {
	config := interpreter.SharedState.Config

	onMeterComputation := config.OnMeterComputation
	if onMeterComputation != nil {
		onMeterComputation(common.ComputationKindFunctionInvocation, 1)
	}

	onFunctionInvocation := config.OnFunctionInvocation
	if onFunctionInvocation != nil {
		onFunctionInvocation(interpreter)
	}
}

func (interpreter *Interpreter) reportInvokedFunctionReturn() {
	config := interpreter.SharedState.Config

	onInvokedFunctionReturn := config.OnInvokedFunctionReturn
	if onInvokedFunctionReturn == nil {
		return
	}

	onInvokedFunctionReturn(interpreter)
}

func (interpreter *Interpreter) ReportComputation(compKind common.ComputationKind, intensity uint) {
	config := interpreter.SharedState.Config

	onMeterComputation := config.OnMeterComputation
	if onMeterComputation != nil {
		onMeterComputation(compKind, intensity)
	}
}

// getMember gets the member value by the given identifier from the given Value depending on its type.
// May return nil if the member does not exist.
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

	// NOTE: do not panic if the member is nil. This is a valid state.
	// For example, when a composite field is initialized with a force-assignment, the field's value is read.

	return result
}

func (interpreter *Interpreter) isInstanceFunction(self Value) *HostFunctionValue {
	return NewHostFunctionValue(
		interpreter,
		sema.IsInstanceFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			firstArgument := invocation.Arguments[0]
			typeValue, ok := firstArgument.(TypeValue)

			if !ok {
				panic(errors.NewUnreachableError())
			}

			staticType := typeValue.Type

			// Values are never instances of unknown types
			if staticType == nil {
				return FalseValue
			}

			// NOTE: not invocation.Self, as that is only set for composite values
			selfType := self.StaticType(interpreter)
			return AsBoolValue(
				interpreter.IsSubType(selfType, staticType),
			)
		},
	)
}

func (interpreter *Interpreter) getTypeFunction(self Value) *HostFunctionValue {
	return NewHostFunctionValue(
		interpreter,
		sema.GetTypeFunctionType,
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter
			staticType := self.StaticType(interpreter)
			return NewTypeValue(interpreter, staticType)
		},
	)
}

func (interpreter *Interpreter) setMember(self Value, locationRange LocationRange, identifier string, value Value) bool {
	return self.(MemberAccessibleValue).SetMember(interpreter, locationRange, identifier, value)
}

func (interpreter *Interpreter) ExpectType(
	value Value,
	expectedType sema.Type,
	locationRange LocationRange,
) {
	valueStaticType := value.StaticType(interpreter)

	if !interpreter.IsSubTypeOfSemaType(valueStaticType, expectedType) {
		valueSemaType := interpreter.MustConvertStaticToSemaType(valueStaticType)

		panic(TypeMismatchError{
			ExpectedType:  expectedType,
			ActualType:    valueSemaType,
			LocationRange: locationRange,
		})
	}
}

func (interpreter *Interpreter) checkContainerMutation(
	elementType StaticType,
	element Value,
	locationRange LocationRange,
) {
	if !interpreter.IsSubType(element.StaticType(interpreter), elementType) {
		panic(ContainerMutationError{
			ExpectedType:  interpreter.MustConvertStaticToSemaType(elementType),
			ActualType:    interpreter.MustSemaTypeOfValue(element),
			LocationRange: locationRange,
		})
	}
}

func (interpreter *Interpreter) checkReferencedResourceNotDestroyed(value Value, locationRange LocationRange) {
	resourceKindedValue, ok := value.(ResourceKindedValue)
	if !ok || !resourceKindedValue.IsDestroyed() {
		return
	}

	panic(DestroyedResourceError{
		LocationRange: locationRange,
	})
}

func (interpreter *Interpreter) checkReferencedResourceNotMovedOrDestroyed(
	referencedValue Value,
	locationRange LocationRange,
) {
	resourceKindedValue, ok := referencedValue.(ReferenceTrackedResourceKindedValue)
	if !ok {
		return
	}

	if resourceKindedValue.IsDestroyed() {
		panic(DestroyedResourceError{
			LocationRange: locationRange,
		})
	}

	if resourceKindedValue.IsStaleResource(interpreter) {
		panic(InvalidatedResourceReferenceError{
			LocationRange: locationRange,
		})
	}
}

func (interpreter *Interpreter) RemoveReferencedSlab(storable atree.Storable) {
	storageIDStorable, ok := storable.(atree.StorageIDStorable)
	if !ok {
		return
	}

	config := interpreter.SharedState.Config

	storageID := atree.StorageID(storageIDStorable)
	err := config.Storage.Remove(storageID)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}

func (interpreter *Interpreter) maybeValidateAtreeValue(v atree.Value) {
	config := interpreter.SharedState.Config

	if config.AtreeValueValidationEnabled {
		interpreter.ValidateAtreeValue(v)
	}

	if config.AtreeStorageValidationEnabled {
		err := config.Storage.CheckHealth()
		if err != nil {
			panic(errors.NewExternalError(err))
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

	defaultHIP := newHashInputProvider(interpreter, EmptyLocationRange)

	hip := func(value atree.Value, buffer []byte) ([]byte, error) {
		if _, ok := value.(StringAtreeValue); ok {
			return StringAtreeHashInput(value, buffer)
		}

		return defaultHIP(value, buffer)
	}

	config := interpreter.SharedState.Config
	storage := config.Storage

	compare := func(storable, otherStorable atree.Storable) bool {
		value, err := storable.StoredValue(storage)
		if err != nil {
			panic(err)
		}

		if _, ok := value.(StringAtreeValue); ok {
			equal, err := StringAtreeComparator(
				storage,
				value,
				otherStorable,
			)
			if err != nil {
				panic(err)
			}

			return equal
		}

		if equatableValue, ok := value.(EquatableValue); ok {
			otherValue := StoredValue(interpreter, otherStorable, storage)
			return equatableValue.Equal(interpreter, EmptyLocationRange, otherValue)
		}

		// Not all values are comparable, assume valid for now
		return true
	}

	switch value := value.(type) {
	case *atree.Array:
		err := atree.ValidArray(value, value.Type(), tic, hip)
		if err != nil {
			panic(errors.NewExternalError(err))
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
				panic(errors.NewExternalError(err))
			}
		}

	case *atree.OrderedMap:
		err := atree.ValidMap(value, value.Type(), tic, hip)
		if err != nil {
			panic(errors.NewExternalError(err))
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
				panic(errors.NewExternalError(err))
			}
		}
	}
}

func (interpreter *Interpreter) maybeTrackReferencedResourceKindedValue(value Value) {
	if value, ok := value.(ReferenceTrackedResourceKindedValue); ok {
		interpreter.trackReferencedResourceKindedValue(value.StorageID(), value)
	}
}

func (interpreter *Interpreter) trackReferencedResourceKindedValue(
	id atree.StorageID,
	value ReferenceTrackedResourceKindedValue,
) {
	values := interpreter.SharedState.referencedResourceKindedValues[id]
	if values == nil {
		values = map[ReferenceTrackedResourceKindedValue]struct{}{}
		interpreter.SharedState.referencedResourceKindedValues[id] = values
	}
	values[value] = struct{}{}
}

func (interpreter *Interpreter) updateReferencedResource(
	currentStorageID atree.StorageID,
	newStorageID atree.StorageID,
	updateFunc func(value ReferenceTrackedResourceKindedValue),
) {
	values := interpreter.SharedState.referencedResourceKindedValues[currentStorageID]
	if values == nil {
		return
	}
	for value := range values { //nolint:maprange
		updateFunc(value)
	}

	// If the move is to a new location, then the resources are already cleared via the update function above.
	// So no need to track those stale resources anymore.
	if newStorageID != currentStorageID {
		interpreter.SharedState.referencedResourceKindedValues[newStorageID] = values
		interpreter.SharedState.referencedResourceKindedValues[currentStorageID] = nil
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

	config := interpreter.SharedState.Config

	if !config.InvalidatedResourceValidationEnabled ||
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
	if _, exists := interpreter.SharedState.resourceVariables[resourceKindedValue]; exists {
		panic(InvalidatedResourceError{
			LocationRange: LocationRange{
				Location:    interpreter.Location,
				HasPosition: hasPosition,
			},
		})
	}

	interpreter.SharedState.resourceVariables[resourceKindedValue] = variable
}

// checkInvalidatedResourceUse checks whether a resource variable is used after invalidation.
func (interpreter *Interpreter) checkInvalidatedResourceUse(
	value Value,
	variable *Variable,
	identifier string,
	hasPosition ast.HasPosition,
) {
	config := interpreter.SharedState.Config

	if !config.InvalidatedResourceValidationEnabled ||
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
	if existingVar, exists := interpreter.SharedState.resourceVariables[resourceKindedValue]; !exists || existingVar != variable {
		panic(InvalidatedResourceError{
			LocationRange: LocationRange{
				Location:    interpreter.Location,
				HasPosition: hasPosition,
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
	config := interpreter.SharedState.Config

	if !config.InvalidatedResourceValidationEnabled {
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
	delete(interpreter.SharedState.resourceVariables, resourceKindedValue)
}

// MeterMemory delegates the memory usage to the interpreter's memory gauge, if any.
func (interpreter *Interpreter) MeterMemory(usage common.MemoryUsage) error {
	config := interpreter.SharedState.Config
	common.UseMemory(config.MemoryGauge, usage)
	return nil
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

func (interpreter *Interpreter) Storage() Storage {
	return interpreter.SharedState.Config.Storage
}

// ConfigureAccountLinkingAllowed configures if execution is allowed to use account linking,
// depending on the occurrence of the pragma declaration #allowAccountLinking.
//
// The pragma declaration must appear as a top-level declaration (i.e. not nested in the program),
// and must appear before all other declarations (i.e. at the top of the program).
//
// This requirement is also checked statically.
//
// This is a temporary feature, which is planned to get replaced by capability controllers,
// and a new Account type with entitlements.
func (interpreter *Interpreter) ConfigureAccountLinkingAllowed() {
	config := interpreter.SharedState.Config

	config.AccountLinkingAllowed = false

	declarations := interpreter.Program.Program.Declarations()
	if len(declarations) < 1 {
		return
	}

	pragmaDeclaration, isPragma := declarations[0].(*ast.PragmaDeclaration)
	if !isPragma || !sema.IsAllowAccountLinkingPragma(pragmaDeclaration) {
		return
	}

	config.AccountLinkingAllowed = true
}
