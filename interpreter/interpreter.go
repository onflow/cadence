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

package interpreter

import (
	"encoding/binary"
	goErrors "errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"

	"golang.org/x/xerrors"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

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

// CapabilityBorrowHandlerFunc is a function that is used to borrow ID capabilities.
type CapabilityBorrowHandlerFunc func(
	context BorrowCapabilityControllerContext,
	locationRange LocationRange,
	address AddressValue,
	capabilityID UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) ReferenceValue

// CapabilityCheckHandlerFunc is a function that is used to check ID capabilities.
type CapabilityCheckHandlerFunc func(
	context CheckCapabilityControllerContext,
	locationRange LocationRange,
	address AddressValue,
	capabilityID UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) BoolValue

// InjectedCompositeFieldsHandlerFunc is a function that handles storage reads.
type InjectedCompositeFieldsHandlerFunc func(
	context AccountCreationContext,
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

// AccountHandlerFunc is a function that handles retrieving an auth account at a given address.
// The account returned must be of type `Account`.
type AccountHandlerFunc func(
	context AccountCreationContext,
	address AddressValue,
) Value

// ValidateAccountCapabilitiesGetHandlerFunc is a function that is used to handle when a capability of an account is got.
type ValidateAccountCapabilitiesGetHandlerFunc func(
	context AccountCapabilityGetValidationContext,
	locationRange LocationRange,
	address AddressValue,
	path PathValue,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) (bool, error)

// ValidateAccountCapabilitiesPublishHandlerFunc is a function that is used to handle when a capability of an account is got.
type ValidateAccountCapabilitiesPublishHandlerFunc func(
	context AccountCapabilityPublishValidationContext,
	locationRange LocationRange,
	address AddressValue,
	path PathValue,
	capabilityBorrowType *ReferenceStaticType,
) (bool, error)

// UUIDHandlerFunc is a function that handles the generation of UUIDs.
type UUIDHandlerFunc func() (uint64, error)

// CompositeTypeHandlerFunc is a function that loads composite types.
type CompositeTypeHandlerFunc func(location common.Location, typeID TypeID) *sema.CompositeType

// InterfaceTypeHandlerFunc is a function that loads interface types.
type InterfaceTypeHandlerFunc func(location common.Location, typeID TypeID) *sema.InterfaceType

// CompositeValueFunctionsHandlerFunc is a function that loads composite value functions.
type CompositeValueFunctionsHandlerFunc func(
	inter *Interpreter,
	locationRange LocationRange,
	compositeValue *CompositeValue,
) *FunctionOrderedMap

// CompositeTypeCode contains the "prepared" / "callable" "code"
// for the functions and the destructor of a composite
// (contract, struct, resource, event).
//
// As there is no support for inheritance of concrete types,
// these are the "leaf" nodes in the call chain, and are functions.
type CompositeTypeCode struct {
	CompositeFunctions *FunctionOrderedMap
}

type FunctionWrapper = func(inner FunctionValue) FunctionValue

// WrapperCode contains the "prepared" / "callable" "code"
// for inherited types.
//
// These are "branch" nodes in the call chain, and are function wrappers,
// i.e. they wrap the functions / function wrappers that inherit them.
type WrapperCode struct {
	InitializerFunctionWrapper     FunctionWrapper
	FunctionWrappers               map[string]FunctionWrapper
	Functions                      *FunctionOrderedMap
	DefaultDestroyEventConstructor FunctionValue
}

// TypeCodes is the value which stores the "prepared" / "callable" "code"
// of all composite types and interface types.
type TypeCodes struct {
	CompositeCodes map[sema.TypeID]CompositeTypeCode
	InterfaceCodes map[sema.TypeID]WrapperCode
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
}

type Storage interface {
	atree.SlabStorage
	GetDomainStorageMap(
		storageMutationTracker StorageMutationTracker,
		address common.Address,
		domain common.StorageDomain,
		createIfNotExists bool,
	) *DomainStorageMap
	CheckHealth() error
}

type ReferencedResourceKindedValues map[atree.ValueID]map[*EphemeralReferenceValue]struct{}

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
var BaseActivation *VariableActivation

func init() {
	// No need to meter since this is only created once
	BaseActivation = activations.NewActivation[Variable](nil, nil)
	defineBaseFunctions(BaseActivation)
}

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

	// Initialize activations

	interpreter.activations = activations.NewActivations[Variable](interpreter)

	var baseActivation *VariableActivation
	baseActivationHandler := sharedState.Config.BaseActivationHandler
	if baseActivationHandler != nil {
		baseActivation = baseActivationHandler(location)
	}
	if baseActivation == nil {
		baseActivation = BaseActivation
	}

	interpreter.activations.PushNewWithParent(baseActivation)

	return interpreter, nil
}

func (interpreter *Interpreter) FindVariable(name string) Variable {
	return interpreter.activations.Find(name)
}

func (interpreter *Interpreter) GetValueOfVariable(name string) Value {
	variable := interpreter.activations.Find(name)
	return variable.GetValue(interpreter)
}

func (interpreter *Interpreter) findOrDeclareVariable(name string) Variable {
	variable := interpreter.FindVariable(name)
	if variable == nil {
		variable = interpreter.declareVariable(name, nil)
	}
	return variable
}

func (interpreter *Interpreter) setVariable(name string, variable Variable) {
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

	variableValue := variable.GetValue(interpreter)

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

	return InvokeExternally(interpreter, functionValue, functionType, arguments)
}

func InvokeExternally(
	context InvocationContext,
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

		if argumentCount < functionType.Arity.MinCount(parameterCount) {
			return nil, ArgumentCountError{
				ParameterCount: parameterCount,
				ArgumentCount:  argumentCount,
			}
		}

		maxCount := functionType.Arity.MaxCount(parameterCount)
		if maxCount != nil && argumentCount > *maxCount {
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
			preparedArguments[i] = ConvertAndBox(context, locationRange, argument, nil, parameterType)
		}
	}

	var self *Value
	var base *EphemeralReferenceValue
	if boundFunc, ok := functionValue.(BoundFunctionValue); ok {
		self = boundFunc.SelfReference.ReferencedValue(
			context,
			EmptyLocationRange,
			true,
		)
		base = boundFunc.Base
	}

	// NOTE: can't fill argument types, as they are unknown
	invocation := NewInvocation(
		context,
		self,
		base,
		preparedArguments,
		nil,
		nil,
		locationRange,
	)

	return functionValue.Invoke(invocation), nil
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
func InvokeFunction(errorHandler ErrorHandler, function FunctionValue, invocation Invocation) (value Value, err error) {

	// recover internal panics and return them as an error
	defer errorHandler.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	value = function.Invoke(invocation)
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

	_, err = InvokeExternally(interpreter, functionValue, functionType, arguments)
	return err
}

func (interpreter *Interpreter) RecoverErrors(onError func(error)) {
	if r := recover(); r != nil {
		// Recover all errors, because interpreter can be directly invoked by FVM.
		err := asCadenceError(r)

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

func asCadenceError(r any) error {
	err, isError := r.(error)
	if !isError {
		return errors.NewUnexpectedError("%s", r)
	}

	rootError := err

	for {
		switch typedError := err.(type) {
		case Error,
			errors.ExternalError,
			errors.InternalError,
			errors.UserError:
			return typedError
		case xerrors.Wrapper:
			err = typedError.Unwrap()
		case error:
			return errors.NewUnexpectedErrorFromCause(rootError)
		default:
			return errors.NewUnexpectedErrorFromCause(rootError)
		}
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

	var variableDeclarationVariables []Variable

	variableDeclarationCount := len(program.VariableDeclarations())
	if variableDeclarationCount > 0 {
		variableDeclarationVariables = make([]Variable, 0, variableDeclarationCount)

		for _, declaration := range program.VariableDeclarations() {

			// Rebind declaration, so the closure captures to current iteration's value,
			// i.e. the next iteration doesn't override `declaration`

			declaration := declaration

			identifier := declaration.Identifier.Identifier

			var variable Variable

			variable = NewVariableWithGetter(interpreter, func() Value {
				result := interpreter.visitVariableDeclaration(declaration, false)

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
		_ = variable.GetValue(interpreter)
	}
}

func (interpreter *Interpreter) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) StatementResult {
	return interpreter.VisitFunctionDeclaration(declaration.FunctionDeclaration, false)
}

func (interpreter *Interpreter) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration, isStatement bool) StatementResult {

	identifier := declaration.Identifier.Identifier

	functionType := interpreter.Program.Elaboration.FunctionDeclarationFunctionType(declaration)

	// NOTE: find *or* declare, as the function might have not been pre-declared (e.g. in the REPL)
	variable := interpreter.findOrDeclareVariable(identifier)

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	if isStatement {

		// This function declaration is an inner function.
		//
		// Variables which are declared after this function declaration
		// should not be visible or even overwrite the variables captured by the closure
		/// (e.g. through shadowing).
		//
		// For example:
		//
		//     fun foo(a: Int): Int {
		//         fun bar(): Int {
		//             return a
		//             //     ^ should refer to the `a` parameter of `foo`,
		//             //     not to the `a` variable declared after `bar`
		//         }
		//         let a = 2
		//         return bar()
		//     }
		//
		// As variable declarations mutate the current activation in place, capture a clone of the current activation,
		// so that the mutations are not performed on the captured activation.

		lexicalScope = lexicalScope.Clone()
	}

	// make the function itself available inside the function
	lexicalScope.Set(identifier, variable)

	value := interpreter.functionDeclarationValue(
		declaration,
		functionType,
		lexicalScope,
	)

	variable.SetValue(
		interpreter,
		LocationRange{
			Location:    interpreter.Location,
			HasPosition: declaration,
		},
		value,
	)

	return nil
}

func (interpreter *Interpreter) functionDeclarationValue(
	declaration *ast.FunctionDeclaration,
	functionType *sema.FunctionType,
	lexicalScope *VariableActivation,
) *InterpretedFunctionValue {

	var preConditions []ast.Condition
	if declaration.FunctionBlock.PreConditions != nil {
		preConditions = declaration.FunctionBlock.PreConditions.Conditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions []ast.Condition

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
	preConditions []ast.Condition,
	body func() StatementResult,
	postConditions []ast.Condition,
	returnType sema.Type,
	declarationLocationRange LocationRange,
) Value {

	// block scope: each function block gets an activation record
	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	result := interpreter.visitStatements(beforeStatements)
	if result, ok := result.(ReturnResult); ok {
		return result.Value
	}

	interpreter.visitConditions(preConditions, ast.ConditionKindPre)

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
		resultValue := interpreter.resultValue(returnValue, returnType, declarationLocationRange)
		interpreter.declareVariable(
			sema.ResultIdentifier,
			resultValue,
		)
	}

	interpreter.visitConditions(postConditions, ast.ConditionKindPost)

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
func (interpreter *Interpreter) resultValue(returnValue Value, returnType sema.Type, declarationLocationRange LocationRange) Value {
	if !returnType.IsResourceType() {
		return returnValue
	}

	resultAuth := func(ty sema.Type) Authorization {
		auth := UnauthorizedAccess
		// reference is authorized to the entire resource, since it is only accessible in a function where a resource value is owned
		if entitlementSupportingType, ok := ty.(sema.EntitlementSupportingType); ok {
			access := entitlementSupportingType.SupportedEntitlements().Access()
			auth = ConvertSemaAccessToStaticAuthorization(interpreter, access)
		}
		return auth
	}

	if optionalType, ok := returnType.(*sema.OptionalType); ok {
		switch returnValue := returnValue.(type) {
		// If this value is an optional value (T?), then transform it into an optional reference (&T)?.
		case *SomeValue:

			innerValue := NewEphemeralReferenceValue(
				interpreter,
				resultAuth(returnType),
				returnValue.value,
				optionalType.Type,
				declarationLocationRange,
			)

			return NewSomeValueNonCopying(interpreter, innerValue)
		case NilValue:
			return NilValue{}
		}
	}

	return NewEphemeralReferenceValue(
		interpreter,
		resultAuth(returnType),
		returnValue,
		returnType,
		declarationLocationRange,
	)
}

func (interpreter *Interpreter) visitConditions(conditions []ast.Condition, kind ast.ConditionKind) {
	for _, condition := range conditions {
		interpreter.visitCondition(condition, kind)
	}
}

func (interpreter *Interpreter) visitCondition(condition ast.Condition, kind ast.ConditionKind) {

	switch condition := condition.(type) {
	case *ast.TestCondition:
		// Evaluate the condition as a statement, so we get position information in case of an error
		statement := ast.NewExpressionStatement(interpreter, condition.Test)

		result, ok := interpreter.evalStatement(statement).(ExpressionResult)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		value, ok := result.Value.(BoolValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		if value {
			return
		}

		messageExpression := condition.Message
		var message string
		if messageExpression != nil {
			messageValue := interpreter.evalExpression(messageExpression)
			message = messageValue.(*StringValue).Str
		}

		panic(ConditionError{
			ConditionKind: kind,
			Message:       message,
			LocationRange: LocationRange{
				Location:    interpreter.Location,
				HasPosition: statement,
			},
		})

	case *ast.EmitCondition:
		interpreter.evalStatement((*ast.EmitStatement)(condition))

	default:
		panic(errors.NewUnreachableError())
	}

}

// declareVariable declares a variable in the latest scope
func (interpreter *Interpreter) declareVariable(identifier string, value Value) Variable {
	// NOTE: semantic analysis already checked possible invalid redeclaration

	variable := NewVariableWithValue(interpreter, value)
	interpreter.setVariable(identifier, variable)

	// TODO: add proper location info
	interpreter.startResourceTracking(value, variable, identifier, nil)

	return variable
}

// declareSelfVariable declares a special "self" variable in the latest scope
func (interpreter *Interpreter) declareSelfVariable(value Value, locationRange LocationRange) Variable {
	identifier := sema.SelfIdentifier

	// If the self variable is already a reference (e.g: in attachments),
	// then declare it as a normal variable.
	// No need to explicitly create a new reference for tracking.

	switch value := value.(type) {
	case ReferenceValue:
		return interpreter.declareVariable(identifier, value)
	case *SimpleCompositeValue:
		if value.isTransaction {
			return interpreter.declareVariable(identifier, value)
		}
	}

	// NOTE: semantic analysis already checked possible invalid redeclaration
	variable := NewSelfVariableWithValue(interpreter, value, locationRange)
	interpreter.setVariable(identifier, variable)

	interpreter.startResourceTracking(value, variable, identifier, locationRange)

	return variable
}

func (interpreter *Interpreter) visitAssignment(
	_ ast.TransferOperation,
	targetGetterSetter getterSetter, targetType sema.Type,
	valueExpression ast.Expression, valueType sema.Type,
	position ast.HasPosition,
) {
	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: position,
	}

	// Evaluate the value, and assign it using the setter function

	// Here it is too early to check whether the existing value is a
	// valid non-nil resource (i.e: causing a resource loss), because
	// evaluating the `valueExpression` could change things, and
	// a `nil`/invalid resource at this point could be valid after
	// the evaluation of `valueExpression`.
	// Therefore, delay the checking of resource loss as much as possible,
	// and check it at the 'setter', at the point where the value is assigned.

	value := interpreter.evalExpression(valueExpression)

	transferredValue := transferAndConvert(interpreter, value, valueType, targetType, locationRange)

	targetGetterSetter.set(transferredValue)
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
	variable Variable,
) {
	return interpreter.declareCompositeValue(declaration, lexicalScope)
}

// evaluateDefaultDestroyEvent evaluates all the implicit default arguments to the default destroy event.
//
// the handling of default arguments makes a number of assumptions to simplify the implementation;
// namely that a) all default arguments are lazily evaluated at the site of the invocation,
// b) that either all the parameters or none of the parameters of a function have default arguments,
// and c) functions cannot currently be explicitly invoked if they have default arguments
//
// if we plan to generalize this further, we will need to relax those assumptions
func (interpreter *Interpreter) evaluateDefaultDestroyEvent(
	containingResourceComposite *CompositeValue,
	eventDecl *ast.CompositeDeclaration,
	declarationActivation *VariableActivation,
) (arguments []Value) {

	declarationInterpreter := interpreter
	parameters := eventDecl.DeclarationMembers().Initializers()[0].FunctionDeclaration.ParameterList.Parameters

	declarationInterpreter.activations.PushNewWithParent(declarationActivation)
	defer declarationInterpreter.activations.Pop()

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: eventDecl,
	}

	var self MemberAccessibleValue = containingResourceComposite
	if containingResourceComposite.Kind == common.CompositeKindAttachment {
		var base *EphemeralReferenceValue
		// in evaluation of destroy events, base and self are fully entitled, as the value must be owned
		entitlementSupportingType, ok := MustSemaTypeOfValue(containingResourceComposite, interpreter).(sema.EntitlementSupportingType)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		supportedEntitlements := entitlementSupportingType.SupportedEntitlements()
		access := supportedEntitlements.Access()
		base, self = attachmentBaseAndSelfValues(
			declarationInterpreter,
			access,
			containingResourceComposite,
			locationRange,
		)
		declarationInterpreter.declareVariable(sema.BaseIdentifier, base)
	}
	declarationInterpreter.declareSelfVariable(self, locationRange)

	for _, parameter := range parameters {
		// "lazily" evaluate the default argument expressions.
		// This is "lazy" with respect to the event's declaration:
		// if we declare a default event `ResourceDestroyed(foo: Int = self.x)`,
		// `self.x` is evaluated in the context that exists when the event is destroyed,
		// not the context when it is declared. This function is only called after the destroy
		// triggers the event emission, so with respect to this function it's "eager".
		defaultArg := declarationInterpreter.evalExpression(parameter.DefaultArgument)
		arguments = append(arguments, defaultArg)
	}

	return
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
	variable Variable,
) {
	if declaration.Kind() == common.CompositeKindEnum {
		return interpreter.declareEnumConstructor(declaration.(*ast.CompositeDeclaration), lexicalScope)
	} else {
		return interpreter.declareNonEnumCompositeValue(declaration, lexicalScope)
	}
}

func (declarationInterpreter *Interpreter) declareNonEnumCompositeValue(
	declaration ast.CompositeLikeDeclaration,
	lexicalScope *VariableActivation,
) (
	scope *VariableActivation,
	variable Variable,
) {
	identifier := declaration.DeclarationIdentifier().Identifier
	// NOTE: find *or* declare, as the function might have not been pre-declared (e.g. in the REPL)
	variable = declarationInterpreter.findOrDeclareVariable(identifier)

	// Make the value available in the initializer
	lexicalScope.Set(identifier, variable)

	// Evaluate nested declarations in a new scope, so values
	// of nested declarations won't be visible after the containing declaration

	nestedVariables := map[string]Variable{}

	var destroyEventConstructor FunctionValue

	(func() {
		declarationInterpreter.activations.PushNewWithCurrent()
		defer declarationInterpreter.activations.Pop()

		// Pre-declare empty variables for all interfaces, composites, and function declarations
		predeclare := func(identifier ast.Identifier) {
			name := identifier.Identifier
			lexicalScope.Set(
				name,
				declarationInterpreter.declareVariable(name, nil),
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
			declarationInterpreter.declareInterface(nestedInterfaceDeclaration, lexicalScope)
		}

		for _, nestedCompositeDeclaration := range members.Composites() {

			// Pass the lexical scope, which has the containing composite's value declared,
			// to the nested declarations so they can refer to it, and update the lexical scope
			// so the container's functions can refer to the nested composite's value

			var nestedVariable Variable
			lexicalScope, nestedVariable =
				declarationInterpreter.declareCompositeValue(
					nestedCompositeDeclaration,
					lexicalScope,
				)

			memberIdentifier := nestedCompositeDeclaration.Identifier.Identifier
			nestedVariables[memberIdentifier] = nestedVariable

			// statically we know there is at most one of these
			if nestedCompositeDeclaration.IsResourceDestructionDefaultEvent() {
				destroyEventConstructor = nestedVariable.GetValue(declarationInterpreter).(FunctionValue)
			}
		}

		for _, nestedAttachmentDeclaration := range members.Attachments() {

			// Pass the lexical scope, which has the containing composite's value declared,
			// to the nested declarations so they can refer to it, and update the lexical scope
			// so the container's functions can refer to the nested composite's value

			var nestedVariable Variable
			lexicalScope, nestedVariable =
				declarationInterpreter.declareAttachmentValue(
					nestedAttachmentDeclaration,
					lexicalScope,
				)

			memberIdentifier := nestedAttachmentDeclaration.Identifier.Identifier
			nestedVariables[memberIdentifier] = nestedVariable
		}
	})()

	compositeType := declarationInterpreter.Program.Elaboration.CompositeDeclarationType(declaration)

	initializerType := compositeType.InitializerFunctionType()

	declarationActivation := declarationInterpreter.activations.CurrentOrNew()

	var initializerFunction FunctionValue
	if declaration.Kind() == common.CompositeKindEvent {
		// Initializer could ideally be a bound function.
		// However, since it is created and being called here itself, and
		// because it is never passed around, it is OK to just create as static function
		// without  the bound-function wrapper.
		initializerFunction = NewStaticHostFunctionValue(
			declarationInterpreter,
			initializerType,
			func(invocation Invocation) Value {
				invocationInterpreter := invocation.InvocationContext
				locationRange := invocation.LocationRange
				self := *invocation.Self

				compositeSelf, ok := self.(*CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if len(compositeType.ConstructorParameters) < 1 {
					return nil
				}

				// event interfaces do not exist
				compositeDecl := declaration.(*ast.CompositeDeclaration)
				if compositeDecl.IsResourceDestructionDefaultEvent() {
					// we implicitly pass the containing composite value as an argument to this invocation
					containerComposite := invocation.Arguments[0].(*CompositeValue)
					invocation.Arguments = declarationInterpreter.evaluateDefaultDestroyEvent(
						containerComposite,
						compositeDecl,
						// to properly lexically scope the evaluation of default arguments, we capture the
						// activations existing at the time when the event was defined and use them here
						declarationActivation,
					)
				}

				for i, argument := range invocation.Arguments {
					parameter := compositeType.ConstructorParameters[i]
					compositeSelf.SetMember(
						invocationInterpreter,
						locationRange,
						parameter.Identifier,
						argument,
					)
				}
				return nil
			},
		)
	} else {
		compositeInitializerFunction := declarationInterpreter.compositeInitializerFunction(declaration, lexicalScope)
		if compositeInitializerFunction != nil {
			initializerFunction = compositeInitializerFunction
		}
	}

	functions := declarationInterpreter.compositeFunctions(declaration, lexicalScope)

	if destroyEventConstructor != nil {
		functions.Set(resourceDefaultDestroyEventName(compositeType), destroyEventConstructor)
	}

	applyDefaultFunctions := func(_ *sema.InterfaceType, code WrapperCode) {

		// Apply default functions, if conforming type does not provide the function

		// Iterating over the map in a non-deterministic way is OK,
		// we only apply the function wrapper to each function,
		// the order does not matter.

		if code.Functions != nil {
			code.Functions.Foreach(func(name string, function FunctionValue) {
				if functions == nil {
					functions = orderedmap.New[FunctionOrderedMap](code.Functions.Len())
				}
				if functions.Contains(name) {
					return
				}
				functions.Set(name, function)
			})
		}
	}

	config := declarationInterpreter.SharedState.Config

	wrapFunctions := func(ty *sema.InterfaceType, code WrapperCode) {

		// Wrap initializer

		initializerFunctionWrapper :=
			code.InitializerFunctionWrapper

		if initializerFunctionWrapper != nil {
			initializerFunction = initializerFunctionWrapper(initializerFunction)
		}

		// Wrap functions

		// Iterating over the map in a non-deterministic way is OK,
		// we only apply the function wrapper to each function,
		// the order does not matter.

		for name, functionWrapper := range code.FunctionWrappers { //nolint:maprange
			// If there's a default implementation, then skip explicitly/separately
			// running the conditions of that functions.
			// Because the conditions also get executed when the default implementation is executed.
			// This works because:
			// 	- `code.Functions` only contains default implementations.
			//	- There is always only one default implementation (cannot override by other interfaces).
			if code.Functions.Contains(name) {
				continue
			}

			fn, ok := functions.Get(name)
			// If there is a wrapper, there MUST be a body.
			if !ok {
				panic(errors.NewUnreachableError())
			}
			functions.Set(name, functionWrapper(fn))
		}

		if code.DefaultDestroyEventConstructor != nil {
			functions.Set(resourceDefaultDestroyEventName(ty), code.DefaultDestroyEventConstructor)
		}
	}

	conformances := compositeType.EffectiveInterfaceConformances()
	interfaceCodes := declarationInterpreter.SharedState.typeCodes.InterfaceCodes

	// First apply the default functions, and then wrap with conditions.
	// These needs to be done in separate phases.
	// Otherwise, if the condition and the default implementation are coming from two different inherited interfaces,
	// then the condition would wrap an empty implementation, because the default impl is not resolved by the time.

	for i := len(conformances) - 1; i >= 0; i-- {
		conformance := conformances[i].InterfaceType
		applyDefaultFunctions(conformance, interfaceCodes[conformance.ID()])
	}

	for i := len(conformances) - 1; i >= 0; i-- {
		conformance := conformances[i].InterfaceType
		wrapFunctions(conformance, interfaceCodes[conformance.ID()])
	}

	declarationInterpreter.SharedState.typeCodes.CompositeCodes[compositeType.ID()] = CompositeTypeCode{
		CompositeFunctions: functions,
	}

	location := declarationInterpreter.Location

	qualifiedIdentifier := compositeType.QualifiedIdentifier()

	constructorType := compositeType.ConstructorFunctionType()

	constructorGenerator := func(address common.Address) *HostFunctionValue {
		// Constructor is a static function.
		return NewStaticHostFunctionValue(
			declarationInterpreter,
			constructorType,
			func(invocation Invocation) Value {

				invocationContext := invocation.InvocationContext

				// Check that the resource is constructed
				// in the same location as it was declared

				locationRange := invocation.LocationRange

				if compositeType.Kind == common.CompositeKindResource &&
					invocationContext.GetLocation() != compositeType.Location {

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
						invocationContext,
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
							invocationContext,
							sema.ResourceUUIDFieldName,
							NewUInt64Value(
								invocationContext,
								func() uint64 {
									return uuid
								},
							),
						),
					)
				}

				value := NewCompositeValue(
					invocationContext,
					locationRange,
					location,
					qualifiedIdentifier,
					declaration.Kind(),
					fields,
					address,
				)

				value.injectedFields = injectedFields
				value.Functions = functions

				var self Value = value
				if declaration.Kind() == common.CompositeKindAttachment {

					attachmentType := MustSemaTypeOfValue(value, invocationContext).(*sema.CompositeType)
					// Self's type in the constructor is fully entitled, since
					// the constructor can only be called when in possession of the base resource

					access := attachmentType.SupportedEntitlements().Access()
					auth := ConvertSemaAccessToStaticAuthorization(invocationContext, access)

					self = NewEphemeralReferenceValue(invocationContext, auth, value, attachmentType, locationRange)

					// set the base to the implicitly provided value, and remove this implicit argument from the list
					implicitArgumentPos := len(invocation.Arguments) - 1
					invocation.Base = invocation.Arguments[implicitArgumentPos].(*EphemeralReferenceValue)

					var ok bool
					value.base, ok = invocation.Base.Value.(*CompositeValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					invocation.Arguments[implicitArgumentPos] = nil
					invocation.Arguments = invocation.Arguments[:implicitArgumentPos]
					invocation.ArgumentTypes[implicitArgumentPos] = nil
					invocation.ArgumentTypes = invocation.ArgumentTypes[:implicitArgumentPos]
				}
				invocation.Self = &self

				if declaration.Kind() == common.CompositeKindContract {
					// NOTE: set the variable value immediately, as the contract value
					// needs to be available for nested declarations

					variable.InitializeWithValue(value)

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
	}

	// Contract declarations declare a value / instance (singleton),
	// for all other composite kinds, the constructor is declared

	if declaration.Kind() == common.CompositeKindContract {
		variable.InitializeWithGetter(func() Value {
			positioned := ast.NewRangeFromPositioned(declarationInterpreter, declaration.DeclarationIdentifier())

			contractValue := config.ContractValueHandler(
				declarationInterpreter,
				compositeType,
				constructorGenerator,
				positioned,
			)

			contractValue.SetNestedVariables(nestedVariables)
			return contractValue
		})
	} else {
		constructor := constructorGenerator(common.ZeroAddress)
		constructor.NestedVariables = nestedVariables
		variable.SetValue(
			declarationInterpreter,
			LocationRange{
				Location:    location,
				HasPosition: declaration,
			},
			constructor,
		)
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
	variable Variable,
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

	constructorNestedVariables := map[string]Variable{}

	for i, enumCase := range enumCases {

		// TODO: replace, avoid conversion
		rawValue := convert(
			interpreter,
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

	value := EnumConstructorFunction(interpreter, compositeType, caseValues, constructorNestedVariables)
	variable.SetValue(
		interpreter,
		locationRange,
		value,
	)

	return lexicalScope, variable
}

func EnumConstructorFunction(
	gauge common.MemoryGauge,
	enumType *sema.CompositeType,
	cases []EnumCase,
	nestedVariables map[string]Variable,
) *HostFunctionValue {

	// Prepare a lookup table based on the big-endian byte representation

	lookupTable := make(map[string]Value, len(cases))

	for _, c := range cases {
		rawValueBigEndianBytes := c.RawValue.ToBigEndianBytes()
		lookupTable[string(rawValueBigEndianBytes)] = c.Value
	}

	// Prepare the constructor function which performs a lookup in the lookup table

	// Constructor is a static function.
	constructor := NewStaticHostFunctionValue(
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

			return NewSomeValueNonCopying(invocation.InvocationContext, caseValue)
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

	var preConditions []ast.Condition
	if initializer.FunctionDeclaration.FunctionBlock.PreConditions != nil {
		preConditions = initializer.FunctionDeclaration.FunctionBlock.PreConditions.Conditions
	}

	statements := initializer.FunctionDeclaration.FunctionBlock.Block.Statements

	var beforeStatements []ast.Statement
	var rewrittenPostConditions []ast.Condition

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

func (interpreter *Interpreter) defaultFunctions(
	members *ast.Members,
	lexicalScope *VariableActivation,
) *FunctionOrderedMap {

	functionDeclarations := members.Functions()
	functionCount := len(functionDeclarations)

	if functionCount == 0 {
		return nil
	}

	functions := orderedmap.New[FunctionOrderedMap](functionCount)

	for _, functionDeclaration := range functionDeclarations {
		name := functionDeclaration.Identifier.Identifier
		if !functionDeclaration.FunctionBlock.HasStatements() {
			continue
		}

		functions.Set(
			name,
			interpreter.compositeFunction(
				functionDeclaration,
				lexicalScope,
			),
		)
	}

	return functions
}

func (interpreter *Interpreter) compositeFunctions(
	compositeDeclaration ast.CompositeLikeDeclaration,
	lexicalScope *VariableActivation,
) *FunctionOrderedMap {

	functions := orderedmap.New[FunctionOrderedMap](len(compositeDeclaration.DeclarationMembers().Functions()))

	for _, functionDeclaration := range compositeDeclaration.DeclarationMembers().Functions() {
		name := functionDeclaration.Identifier.Identifier
		functions.Set(
			name,
			interpreter.compositeFunction(
				functionDeclaration,
				lexicalScope,
			),
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
			functionType,
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

	var preConditions []ast.Condition

	if functionDeclaration.FunctionBlock.PreConditions != nil {
		preConditions = functionDeclaration.FunctionBlock.PreConditions.Conditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions []ast.Condition

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
	return IsSubTypeOfSemaType(interpreter, value.StaticType(interpreter), targetType)
}

func transferAndConvert(
	context ValueConversionContext,
	value Value,
	valueType, targetType sema.Type,
	locationRange LocationRange,
) Value {

	transferredValue := value.Transfer(
		context,
		locationRange,
		atree.Address{},
		false,
		nil,
		nil,
		true, // value is standalone.
	)

	result := ConvertAndBox(
		context,
		locationRange,
		transferredValue,
		valueType,
		targetType,
	)

	// Defensively check the value's type matches the target type
	resultStaticType := result.StaticType(context)

	if targetType != nil &&
		!IsSubTypeOfSemaType(context, resultStaticType, targetType) {

		resultSemaType := MustConvertStaticToSemaType(resultStaticType, context)

		panic(ValueTransferTypeError{
			ExpectedType:  targetType,
			ActualType:    resultSemaType,
			LocationRange: locationRange,
		})
	}

	return result
}

// ConvertAndBox converts a value to a target type, and boxes in optionals and any value, if necessary
func ConvertAndBox(
	context ValueCreationContext,
	locationRange LocationRange,
	value Value,
	valueType, targetType sema.Type,
) Value {
	value = convert(context, value, valueType, targetType, locationRange)
	return BoxOptional(context, value, targetType)
}

// Produces the `valueStaticType` argument into a new static type that conforms
// to the specification of the `targetSemaType`. At the moment, this means that the
// authorization of any reference types in `valueStaticType` are changed to match the
// authorization of any equivalently-positioned reference types in `targetSemaType`.
func convertStaticType(
	gauge common.MemoryGauge,
	valueStaticType StaticType,
	targetSemaType sema.Type,
) StaticType {
	switch valueStaticType := valueStaticType.(type) {
	case *ReferenceStaticType:
		if targetReferenceType, isReferenceType := targetSemaType.(*sema.ReferenceType); isReferenceType {
			return NewReferenceStaticType(
				gauge,
				ConvertSemaAccessToStaticAuthorization(gauge, targetReferenceType.Authorization),
				valueStaticType.ReferencedType,
			)
		}

	case *OptionalStaticType:
		if targetOptionalType, isOptionalType := targetSemaType.(*sema.OptionalType); isOptionalType {
			return NewOptionalStaticType(
				gauge,
				convertStaticType(
					gauge,
					valueStaticType.Type,
					targetOptionalType.Type,
				),
			)
		}

	case *DictionaryStaticType:
		if targetDictionaryType, isDictionaryType := targetSemaType.(*sema.DictionaryType); isDictionaryType {
			return NewDictionaryStaticType(
				gauge,
				convertStaticType(
					gauge,
					valueStaticType.KeyType,
					targetDictionaryType.KeyType,
				),
				convertStaticType(
					gauge,
					valueStaticType.ValueType,
					targetDictionaryType.ValueType,
				),
			)
		}

	case *VariableSizedStaticType:
		if targetArrayType, isArrayType := targetSemaType.(*sema.VariableSizedType); isArrayType {
			return NewVariableSizedStaticType(
				gauge,
				convertStaticType(
					gauge,
					valueStaticType.Type,
					targetArrayType.Type,
				),
			)
		}

	case *ConstantSizedStaticType:
		if targetArrayType, isArrayType := targetSemaType.(*sema.ConstantSizedType); isArrayType {
			return NewConstantSizedStaticType(
				gauge,
				convertStaticType(
					gauge,
					valueStaticType.Type,
					targetArrayType.Type,
				),
				valueStaticType.Size,
			)
		}

	case *CapabilityStaticType:
		if targetCapabilityType, isCapabilityType := targetSemaType.(*sema.CapabilityType); isCapabilityType {
			return NewCapabilityStaticType(
				gauge,
				convertStaticType(
					gauge,
					valueStaticType.BorrowType,
					targetCapabilityType.BorrowType,
				),
			)
		}
	}
	return valueStaticType
}

func convert(
	context ValueCreationContext,
	value Value,
	valueType,
	targetType sema.Type,
	locationRange LocationRange,
) Value {
	if valueType == nil {
		return value
	}

	unwrappedTargetType := sema.UnwrapOptionalType(targetType)

	// if the value is optional, convert the inner value to the unwrapped target type
	if optionalValueType, valueIsOptional := valueType.(*sema.OptionalType); valueIsOptional {
		switch value := value.(type) {
		case NilValue:
			return value

		case *SomeValue:
			if !optionalValueType.Type.Equal(unwrappedTargetType) {
				innerValue := convert(
					context,
					value.value,
					optionalValueType.Type,
					unwrappedTargetType,
					locationRange,
				)
				return NewSomeValueNonCopying(context, innerValue)
			}
			return value
		}
	}

	switch unwrappedTargetType {
	case sema.IntType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt(context, value, locationRange)
		}

	case sema.UIntType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt(context, value, locationRange)
		}

	// Int*
	case sema.Int8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt8(context, value, locationRange)
		}

	case sema.Int16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt16(context, value, locationRange)
		}

	case sema.Int32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt32(context, value, locationRange)
		}

	case sema.Int64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt64(context, value, locationRange)
		}

	case sema.Int128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt128(context, value, locationRange)
		}

	case sema.Int256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt256(context, value, locationRange)
		}

	// UInt*
	case sema.UInt8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt8(context, value, locationRange)
		}

	case sema.UInt16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt16(context, value, locationRange)
		}

	case sema.UInt32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt32(context, value, locationRange)
		}

	case sema.UInt64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt64(context, value, locationRange)
		}

	case sema.UInt128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt128(context, value, locationRange)
		}

	case sema.UInt256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt256(context, value, locationRange)
		}

	// Word*
	case sema.Word8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord8(context, value, locationRange)
		}

	case sema.Word16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord16(context, value, locationRange)
		}

	case sema.Word32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord32(context, value, locationRange)
		}

	case sema.Word64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord64(context, value, locationRange)
		}

	case sema.Word128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord128(context, value, locationRange)
		}

	case sema.Word256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord256(context, value, locationRange)
		}

	// Fix*

	case sema.Fix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertFix64(context, value, locationRange)
		}

	case sema.UFix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUFix64(context, value, locationRange)
		}
	}

	switch unwrappedTargetType := unwrappedTargetType.(type) {
	case *sema.AddressType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertAddress(context, value, locationRange)
		}

	case sema.ArrayType:
		if arrayValue, isArray := value.(*ArrayValue); isArray && !valueType.Equal(unwrappedTargetType) {

			oldArrayStaticType := arrayValue.StaticType(context)
			arrayStaticType := convertStaticType(context, oldArrayStaticType, unwrappedTargetType).(ArrayStaticType)

			if oldArrayStaticType.Equal(arrayStaticType) {
				return value
			}

			targetElementType := MustConvertStaticToSemaType(arrayStaticType.ElementType(), context)

			array := arrayValue.array

			iterator, err := array.ReadOnlyIterator()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			return NewArrayValueWithIterator(
				context,
				arrayStaticType,
				arrayValue.GetOwner(),
				array.Count(),
				func() Value {
					element, err := iterator.Next()
					if err != nil {
						panic(errors.NewExternalError(err))
					}
					if element == nil {
						return nil
					}

					value := MustConvertStoredValue(context, element)
					valueType := MustConvertStaticToSemaType(value.StaticType(context), context)
					return convert(context, value, valueType, targetElementType, locationRange)
				},
			)
		}

	case *sema.DictionaryType:
		if dictValue, isDict := value.(*DictionaryValue); isDict && !valueType.Equal(unwrappedTargetType) {

			oldDictStaticType := dictValue.StaticType(context)
			dictStaticType := convertStaticType(context, oldDictStaticType, unwrappedTargetType).(*DictionaryStaticType)

			if oldDictStaticType.Equal(dictStaticType) {
				return value
			}

			targetKeyType := MustConvertStaticToSemaType(dictStaticType.KeyType, context)
			targetValueType := MustConvertStaticToSemaType(dictStaticType.ValueType, context)

			dictionary := dictValue.dictionary

			iterator, err := dictionary.ReadOnlyIterator()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			return newDictionaryValueWithIterator(
				context,
				locationRange,
				dictStaticType,
				dictionary.Count(),
				dictionary.Seed(),
				common.Address(dictionary.Address()),
				func() (Value, Value) {
					k, v, err := iterator.Next()

					if err != nil {
						panic(errors.NewExternalError(err))
					}
					if k == nil || v == nil {
						return nil, nil
					}

					key := MustConvertStoredValue(context, k)
					value := MustConvertStoredValue(context, v)

					keyType := MustConvertStaticToSemaType(key.StaticType(context), context)
					valueType := MustConvertStaticToSemaType(value.StaticType(context), context)

					convertedKey := convert(context, key, keyType, targetKeyType, locationRange)
					convertedValue := convert(context, value, valueType, targetValueType, locationRange)

					return convertedKey, convertedValue
				},
			)
		}

	case *sema.CapabilityType:
		if !valueType.Equal(unwrappedTargetType) && unwrappedTargetType.BorrowType != nil {
			targetBorrowType := unwrappedTargetType.BorrowType.(*sema.ReferenceType)

			switch capability := value.(type) {
			case *IDCapabilityValue:
				valueBorrowType := capability.BorrowType.(*ReferenceStaticType)
				borrowType := convertStaticType(context, valueBorrowType, targetBorrowType)
				if capability.isInvalid() {
					return NewInvalidCapabilityValue(context, capability.address, borrowType)
				}
				return NewCapabilityValue(
					context,
					capability.ID,
					capability.address,
					borrowType,
				)
			default:
				// unsupported capability value
				panic(errors.NewUnreachableError())
			}
		}

	case *sema.ReferenceType:
		targetAuthorization := ConvertSemaAccessToStaticAuthorization(context, unwrappedTargetType.Authorization)
		switch ref := value.(type) {
		case *EphemeralReferenceValue:
			if shouldConvertReference(ref, valueType, unwrappedTargetType, targetAuthorization) {
				checkMappedEntitlements(unwrappedTargetType, locationRange)
				return NewEphemeralReferenceValue(
					context,
					targetAuthorization,
					ref.Value,
					unwrappedTargetType.Type,
					locationRange,
				)
			}

		case *StorageReferenceValue:
			if shouldConvertReference(ref, valueType, unwrappedTargetType, targetAuthorization) {
				checkMappedEntitlements(unwrappedTargetType, locationRange)
				return NewStorageReferenceValue(
					context,
					targetAuthorization,
					ref.TargetStorageAddress,
					ref.TargetPath,
					unwrappedTargetType.Type,
				)
			}

		default:
			panic(errors.NewUnexpectedError("unsupported reference value: %T", ref))
		}
	}

	return value
}

func shouldConvertReference(
	ref ReferenceValue,
	valueType sema.Type,
	unwrappedTargetType *sema.ReferenceType,
	targetAuthorization Authorization,
) bool {
	if !valueType.Equal(unwrappedTargetType) {
		return true
	}

	return !ref.BorrowType().Equal(unwrappedTargetType.Type) ||
		!ref.GetAuthorization().Equal(targetAuthorization)
}

func checkMappedEntitlements(unwrappedTargetType *sema.ReferenceType, locationRange LocationRange) {
	// check defensively that we never create a runtime mapped entitlement value
	if _, isMappedAuth := unwrappedTargetType.Authorization.(*sema.EntitlementMapAccess); isMappedAuth {
		panic(UnexpectedMappedEntitlementError{
			Type:          unwrappedTargetType,
			LocationRange: locationRange,
		})
	}
}

// BoxOptional boxes a value in optionals, if necessary
func BoxOptional(gauge common.MemoryGauge, value Value, targetType sema.Type) Value {

	inner := value

	for {
		optionalType, ok := targetType.(*sema.OptionalType)
		if !ok {
			break
		}

		switch typedInner := inner.(type) {
		case *SomeValue:
			inner = typedInner.InnerValue()

		case NilValue:
			// NOTE: nested nil will be unboxed!
			return inner

		default:
			value = NewSomeValueNonCopying(gauge, value)
		}

		targetType = optionalType.Type
	}
	return value
}

func Unbox(value Value) Value {
	for {
		some, ok := value.(*SomeValue)
		if !ok {
			return value
		}

		value = some.InnerValue()
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
			if nestedCompositeDeclaration.Kind() == common.CompositeKindEvent {
				interpreter.declareNonEnumCompositeValue(nestedCompositeDeclaration, lexicalScope)
			} else {
				// this is statically prevented in the checker
				panic(errors.NewUnreachableError())
			}
		}
	})()

	interfaceType := interpreter.Program.Elaboration.InterfaceDeclarationType(declaration)
	typeID := interfaceType.ID()

	initializerFunctionWrapper := interpreter.initializerFunctionWrapper(
		declaration.Members,
		interfaceType.InitializerParameters,
		lexicalScope,
	)

	var defaultDestroyEventConstructor FunctionValue
	if defautlDestroyEvent := interpreter.Program.Elaboration.DefaultDestroyDeclaration(declaration); defautlDestroyEvent != nil {
		var nestedVariable Variable
		lexicalScope, nestedVariable = interpreter.declareCompositeValue(
			defautlDestroyEvent,
			lexicalScope,
		)
		defaultDestroyEventConstructor = nestedVariable.GetValue(interpreter).(FunctionValue)
	}

	functionWrappers := interpreter.functionWrappers(declaration.Members, lexicalScope)
	defaultFunctions := interpreter.defaultFunctions(declaration.Members, lexicalScope)

	interpreter.SharedState.typeCodes.InterfaceCodes[typeID] = WrapperCode{
		InitializerFunctionWrapper:     initializerFunctionWrapper,
		FunctionWrappers:               functionWrappers,
		Functions:                      defaultFunctions,
		DefaultDestroyEventConstructor: defaultDestroyEventConstructor,
	}
}

func (interpreter *Interpreter) initializerFunctionWrapper(
	members *ast.Members,
	parameters []sema.Parameter,
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
		&sema.FunctionType{
			Parameters:           parameters,
			ReturnTypeAnnotation: sema.VoidTypeAnnotation,
		},
		lexicalScope,
	)
}

func (interpreter *Interpreter) functionConditionsWrapper(
	declaration *ast.FunctionDeclaration,
	functionType *sema.FunctionType,
	lexicalScope *VariableActivation,
) FunctionWrapper {

	if declaration.FunctionBlock == nil ||
		declaration.FunctionBlock.HasStatements() {
		// If there's a default implementation (i.e: has statements),
		// then skip explicitly/separately running the conditions of that functions.
		// Because the conditions also get executed when the default implementation is executed.
		return nil
	}

	var preConditions []ast.Condition
	if declaration.FunctionBlock.PreConditions != nil {
		preConditions = declaration.FunctionBlock.PreConditions.Conditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions []ast.Condition

	if declaration.FunctionBlock.PostConditions != nil {

		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite(declaration.FunctionBlock.PostConditions)

		beforeStatements = postConditionsRewrite.BeforeStatements
		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
	}

	return func(inner FunctionValue) FunctionValue {

		// NOTE: The `inner` function cannot be nil.
		// An executing function always have a body.
		if inner == nil {
			panic(errors.NewUnreachableError())
		}

		// Condition wrapper is a static function.
		return NewStaticHostFunctionValue(
			interpreter,
			functionType,
			func(invocation Invocation) Value {
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
					interpreter.declareSelfVariable(*invocation.Self, invocation.LocationRange)
				}
				if invocation.Base != nil {
					interpreter.declareVariable(sema.BaseIdentifier, invocation.Base)
				}

				// NOTE: It is important to wrap the invocation in a function,
				//  so the inner function isn't invoked here

				body := func() StatementResult {

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
						variable Variable
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

					returnValue := inner.Invoke(invocation)

					// Restore the resources which were temporarily invalidated
					// before execution of the inner function

					for _, argumentVariable := range argumentVariables {
						value := argumentVariable.value
						interpreter.invalidateResource(value)
						interpreter.SharedState.resourceVariables[value] = argumentVariable.variable
					}
					return ReturnResult{Value: returnValue}
				}

				declarationLocationRange := LocationRange{
					Location:    interpreter.Location,
					HasPosition: declaration,
				}

				return interpreter.visitFunctionBody(
					beforeStatements,
					preConditions,
					body,
					rewrittenPostConditions,
					functionType.ReturnTypeAnnotation.Type,
					declarationLocationRange,
				)
			},
		)
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

func StoredValueExists(
	context StorageContext,
	storageAddress common.Address,
	domain common.StorageDomain,
	identifier StorageMapKey,
) bool {
	accountStorage := context.Storage().GetDomainStorageMap(context, storageAddress, domain, false)
	if accountStorage == nil {
		return false
	}
	return accountStorage.ValueExists(identifier)
}

func (interpreter *Interpreter) ReadStored(
	storageAddress common.Address,
	domain common.StorageDomain,
	identifier StorageMapKey,
) Value {
	accountStorage := interpreter.Storage().GetDomainStorageMap(interpreter, storageAddress, domain, false)
	if accountStorage == nil {
		return nil
	}
	return accountStorage.ReadValue(interpreter, identifier)
}

func (interpreter *Interpreter) WriteStored(
	storageAddress common.Address,
	domain common.StorageDomain,
	key StorageMapKey,
	value Value,
) (existed bool) {
	accountStorage := interpreter.Storage().GetDomainStorageMap(interpreter, storageAddress, domain, true)
	return accountStorage.WriteValue(interpreter, key, value)
}

type fromStringFunctionValue struct {
	receiverType sema.Type
	hostFunction *HostFunctionValue
}

// a function that attempts to create a Cadence value from a string, e.g. parsing a number from a string
type stringValueParser func(common.MemoryGauge, string) OptionalValue

func newFromStringFunction(ty sema.Type, parser stringValueParser) fromStringFunctionValue {
	functionType := sema.FromStringFunctionType(ty)

	hostFunctionImpl := NewUnmeteredStaticHostFunctionValue(
		functionType,
		func(invocation Invocation) Value {
			argument, ok := invocation.Arguments[0].(*StringValue)
			if !ok {
				// expect typechecker to catch a mismatch here
				panic(errors.NewUnreachableError())
			}
			inter := invocation.InvocationContext
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
	return func(memoryGauge common.MemoryGauge, input string) OptionalValue {
		val, err := strconv.ParseUint(input, 10, bitSize)
		if err != nil {
			return NilOptionalValue
		}

		converted := toValue(memoryGauge, func() IntType {
			return fromUInt64(val)
		})
		return NewSomeValueNonCopying(memoryGauge, converted)
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

	return func(memoryGauge common.MemoryGauge, input string) OptionalValue {
		val, err := strconv.ParseInt(input, 10, bitSize)
		if err != nil {
			return NilOptionalValue
		}

		converted := toValue(memoryGauge, func() IntType {
			return fromInt64(val)
		})
		return NewSomeValueNonCopying(memoryGauge, converted)
	}
}

// No need to use metered constructors for values represented by big.Ints,
// since estimation is more granular than fixed-size types.
func bigIntValueParser(convert func(*big.Int) (Value, bool)) stringValueParser {
	return func(memoryGauge common.MemoryGauge, input string) OptionalValue {
		literalKind := common.IntegerLiteralKindDecimal
		estimatedSize := common.OverEstimateBigIntFromString(input, literalKind)
		common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(estimatedSize))

		val, ok := new(big.Int).SetString(input, literalKind.Base())
		if !ok {
			return NilOptionalValue
		}

		converted, ok := convert(val)

		if !ok {
			return NilOptionalValue
		}
		return NewSomeValueNonCopying(memoryGauge, converted)
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
		newFromStringFunction(sema.Word128Type, bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
			if ok = inRange(b, sema.Word128TypeMinIntBig, sema.Word128TypeMaxIntBig); ok {
				v = NewUnmeteredWord128ValueFromBigInt(b)
			}
			return
		})),
		newFromStringFunction(sema.Word256Type, bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
			if ok = inRange(b, sema.Word256TypeMinIntBig, sema.Word256TypeMaxIntBig); ok {
				v = NewUnmeteredWord256ValueFromBigInt(b)
			}
			return
		})),

		// fixed-points
		newFromStringFunction(sema.Fix64Type, func(memoryGauge common.MemoryGauge, input string) OptionalValue {
			n, err := fixedpoint.ParseFix64(input)
			if err != nil {
				return NilOptionalValue
			}

			val := NewFix64Value(memoryGauge, n.Int64)
			return NewSomeValueNonCopying(memoryGauge, val)

		}),
		newFromStringFunction(sema.UFix64Type, func(memoryGauge common.MemoryGauge, input string) OptionalValue {
			n, err := fixedpoint.ParseUFix64(input)
			if err != nil {
				return NilOptionalValue
			}
			val := NewUFix64Value(memoryGauge, n.Uint64)
			return NewSomeValueNonCopying(memoryGauge, val)
		}),
	}

	values := make(map[string]fromStringFunctionValue, len(declarations))
	for _, decl := range declarations {
		// index declaration by type name
		values[decl.receiverType.String()] = decl
	}

	return values
}()

type fromBigEndianBytesFunctionValue struct {
	receiverType sema.Type
	hostFunction *HostFunctionValue
}

func padWithZeroes(b []byte, expectedLen int) []byte {
	l := len(b)
	if l > expectedLen {
		panic(errors.NewUnreachableError())
	} else if l == expectedLen {
		return b
	}

	var res []byte
	// use existing allocated slice if possible.
	if cap(b) >= expectedLen {
		res = b[:expectedLen]
	} else {
		res = make([]byte, expectedLen)
	}

	copy(res[expectedLen-l:], b)

	// explicitly set to 0 for the first expectedLen - l bytes.
	if cap(b) >= expectedLen {
		for i := 0; i < expectedLen-l; i++ {
			res[i] = 0
		}
	}
	return res
}

// a function that attempts to create a Number from a big-endian bytes.
type bigEndianBytesConverter func(common.MemoryGauge, []byte) Value

func newFromBigEndianBytesFunction(
	ty sema.Type,
	byteLength int,
	converter bigEndianBytesConverter,
) fromBigEndianBytesFunctionValue {
	functionType := sema.FromBigEndianBytesFunctionType(ty)

	// Converter functions are static functions.
	hostFunctionImpl := NewUnmeteredStaticHostFunctionValue(
		functionType,
		func(invocation Invocation) Value {
			argument, ok := invocation.Arguments[0].(*ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			context := invocation.InvocationContext
			bytes, err := ByteArrayValueToByteSlice(context, argument, invocation.LocationRange)
			if err != nil {
				return Nil
			}

			// overflow
			if byteLength != 0 && len(bytes) > byteLength {
				return Nil
			}

			return NewSomeValueNonCopying(context, converter(context, bytes))
		},
	)
	return fromBigEndianBytesFunctionValue{
		receiverType: ty,
		hostFunction: hostFunctionImpl,
	}
}

var fromBigEndianBytesFunctionValues = func() map[string]fromBigEndianBytesFunctionValue {
	declarations := []fromBigEndianBytesFunctionValue{
		// signed int values
		newFromBigEndianBytesFunction(sema.Int8Type, 1, func(gauge common.MemoryGauge, b []byte) Value {
			return NewInt8Value(gauge, func() int8 {
				bytes := padWithZeroes(b, 1)
				return int8(bytes[0])
			})
		}),
		newFromBigEndianBytesFunction(sema.Int16Type, 2, func(gauge common.MemoryGauge, b []byte) Value {
			return NewInt16Value(gauge, func() int16 {
				bytes := padWithZeroes(b, 2)
				val := binary.BigEndian.Uint16(bytes)
				return int16(val)
			})
		}),
		newFromBigEndianBytesFunction(sema.Int32Type, 4, func(gauge common.MemoryGauge, b []byte) Value {
			return NewInt32Value(gauge, func() int32 {
				bytes := padWithZeroes(b, 4)
				val := binary.BigEndian.Uint32(bytes)
				return int32(val)
			})
		}),
		newFromBigEndianBytesFunction(sema.Int64Type, 8, func(gauge common.MemoryGauge, b []byte) Value {
			return NewInt64Value(gauge, func() int64 {
				bytes := padWithZeroes(b, 8)
				val := binary.BigEndian.Uint64(bytes)
				return int64(val)
			})
		}),
		newFromBigEndianBytesFunction(sema.Int128Type, 16, func(gauge common.MemoryGauge, b []byte) Value {
			return NewInt128ValueFromBigInt(gauge, func() *big.Int {
				bi := values.BigEndianBytesToSignedBigInt(b)
				return bi
			})
		}),
		newFromBigEndianBytesFunction(sema.Int256Type, 32, func(gauge common.MemoryGauge, b []byte) Value {
			return NewInt256ValueFromBigInt(gauge, func() *big.Int {
				bi := values.BigEndianBytesToSignedBigInt(b)
				return bi
			})
		}),
		newFromBigEndianBytesFunction(sema.IntType, 0, func(gauge common.MemoryGauge, b []byte) Value {
			bi := values.BigEndianBytesToSignedBigInt(b)
			memoryUsage := common.NewBigIntMemoryUsage(
				common.BigIntByteLength(bi),
			)
			return NewIntValueFromBigInt(gauge, memoryUsage, func() *big.Int { return bi })
		}),

		// unsigned int values
		newFromBigEndianBytesFunction(sema.UInt8Type, 1, func(gauge common.MemoryGauge, b []byte) Value {
			return NewUInt8Value(gauge, func() uint8 { return b[0] })
		}),
		newFromBigEndianBytesFunction(sema.UInt16Type, 2, func(gauge common.MemoryGauge, b []byte) Value {
			return NewUInt16Value(gauge, func() uint16 {
				bytes := padWithZeroes(b, 2)
				val := binary.BigEndian.Uint16(bytes)
				return val
			})
		}),
		newFromBigEndianBytesFunction(sema.UInt32Type, 4, func(gauge common.MemoryGauge, b []byte) Value {
			return NewUInt32Value(gauge, func() uint32 {
				bytes := padWithZeroes(b, 4)
				val := binary.BigEndian.Uint32(bytes)
				return val
			})
		}),
		newFromBigEndianBytesFunction(sema.UInt64Type, 8, func(gauge common.MemoryGauge, b []byte) Value {
			return NewUInt64Value(gauge, func() uint64 {
				bytes := padWithZeroes(b, 8)
				val := binary.BigEndian.Uint64(bytes)
				return val
			})
		}),
		newFromBigEndianBytesFunction(sema.UInt128Type, 16, func(gauge common.MemoryGauge, b []byte) Value {
			return NewUInt128ValueFromBigInt(gauge, func() *big.Int {
				return values.BigEndianBytesToUnsignedBigInt(b)
			})
		}),
		newFromBigEndianBytesFunction(sema.UInt256Type, 32, func(gauge common.MemoryGauge, b []byte) Value {
			return NewUInt256ValueFromBigInt(gauge, func() *big.Int {
				return values.BigEndianBytesToUnsignedBigInt(b)
			})
		}),
		newFromBigEndianBytesFunction(sema.UIntType, 0, func(gauge common.MemoryGauge, b []byte) Value {
			bi := values.BigEndianBytesToUnsignedBigInt(b)
			memoryUsage := common.NewBigIntMemoryUsage(
				common.BigIntByteLength(bi),
			)
			return NewUIntValueFromBigInt(gauge, memoryUsage, func() *big.Int { return bi })
		}),

		// machine-sized word types
		newFromBigEndianBytesFunction(sema.Word8Type, 1, func(gauge common.MemoryGauge, b []byte) Value {
			return NewWord8Value(gauge, func() uint8 { return b[0] })
		}),
		newFromBigEndianBytesFunction(sema.Word16Type, 2, func(gauge common.MemoryGauge, b []byte) Value {
			return NewWord16Value(gauge, func() uint16 {
				bytes := padWithZeroes(b, 2)
				val := binary.BigEndian.Uint16(bytes)
				return val
			})
		}),
		newFromBigEndianBytesFunction(sema.Word32Type, 4, func(gauge common.MemoryGauge, b []byte) Value {
			return NewWord32Value(gauge, func() uint32 {
				bytes := padWithZeroes(b, 4)
				val := binary.BigEndian.Uint32(bytes)
				return val
			})
		}),
		newFromBigEndianBytesFunction(sema.Word64Type, 8, func(gauge common.MemoryGauge, b []byte) Value {
			return NewWord64Value(gauge, func() uint64 {
				bytes := padWithZeroes(b, 8)
				val := binary.BigEndian.Uint64(bytes)
				return val
			})
		}),
		newFromBigEndianBytesFunction(sema.Word128Type, 16, func(gauge common.MemoryGauge, b []byte) Value {
			return NewWord128ValueFromBigInt(gauge, func() *big.Int {
				return values.BigEndianBytesToUnsignedBigInt(b)
			})
		}),
		newFromBigEndianBytesFunction(sema.Word256Type, 32, func(gauge common.MemoryGauge, b []byte) Value {
			return NewWord256ValueFromBigInt(gauge, func() *big.Int {
				return values.BigEndianBytesToUnsignedBigInt(b)
			})
		}),

		// fixed-points
		newFromBigEndianBytesFunction(sema.Fix64Type, 8, func(gauge common.MemoryGauge, b []byte) Value {
			return NewFix64Value(gauge, func() int64 {
				bytes := padWithZeroes(b, 8)
				val := binary.BigEndian.Uint64(bytes)
				return int64(val)
			})
		}),
		newFromBigEndianBytesFunction(sema.UFix64Type, 8, func(gauge common.MemoryGauge, b []byte) Value {
			return NewUFix64Value(gauge, func() uint64 {
				bytes := padWithZeroes(b, 8)
				val := binary.BigEndian.Uint64(bytes)
				return val
			})
		}),
	}

	values := make(map[string]fromBigEndianBytesFunctionValue, len(declarations))
	for _, decl := range declarations {
		// index declaration by type name
		values[decl.receiverType.String()] = decl
	}

	return values
}()

type ValueConverterDeclaration struct {
	min             Value
	max             Value
	Convert         func(common.MemoryGauge, Value, LocationRange) Value
	FunctionType    *sema.FunctionType
	nestedVariables []struct {
		Name  string
		Value Value
	}
	Name string
}

// It would be nice if return types in Go's function types would be covariant
var ConverterDeclarations = []ValueConverterDeclaration{
	{
		Name:         sema.IntTypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.IntType),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertInt(gauge, value, locationRange)
		},
	},
	{
		Name:         sema.UIntTypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.UIntType),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertUInt(gauge, value, locationRange)
		},
		min: NewUnmeteredUIntValueFromBigInt(sema.UIntTypeMin),
	},
	{
		Name:         sema.Int8TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Int8Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertInt8(gauge, value, locationRange)
		},
		min: NewUnmeteredInt8Value(math.MinInt8),
		max: NewUnmeteredInt8Value(math.MaxInt8),
	},
	{
		Name:         sema.Int16TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Int16Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertInt16(gauge, value, locationRange)
		},
		min: NewUnmeteredInt16Value(math.MinInt16),
		max: NewUnmeteredInt16Value(math.MaxInt16),
	},
	{
		Name:         sema.Int32TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Int32Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertInt32(gauge, value, locationRange)
		},
		min: NewUnmeteredInt32Value(math.MinInt32),
		max: NewUnmeteredInt32Value(math.MaxInt32),
	},
	{
		Name:         sema.Int64TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Int64Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertInt64(gauge, value, locationRange)
		},
		min: NewUnmeteredInt64Value(math.MinInt64),
		max: NewUnmeteredInt64Value(math.MaxInt64),
	},
	{
		Name:         sema.Int128TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Int128Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertInt128(gauge, value, locationRange)
		},
		min: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
		max: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
	},
	{
		Name:         sema.Int256TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Int256Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertInt256(gauge, value, locationRange)
		},
		min: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
		max: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
	},
	{
		Name:         sema.UInt8TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.UInt8Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertUInt8(gauge, value, locationRange)
		},
		min: NewUnmeteredUInt8Value(0),
		max: NewUnmeteredUInt8Value(math.MaxUint8),
	},
	{
		Name:         sema.UInt16TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.UInt16Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertUInt16(gauge, value, locationRange)
		},
		min: NewUnmeteredUInt16Value(0),
		max: NewUnmeteredUInt16Value(math.MaxUint16),
	},
	{
		Name:         sema.UInt32TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.UInt32Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertUInt32(gauge, value, locationRange)
		},
		min: NewUnmeteredUInt32Value(0),
		max: NewUnmeteredUInt32Value(math.MaxUint32),
	},
	{
		Name:         sema.UInt64TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.UInt64Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertUInt64(gauge, value, locationRange)
		},
		min: NewUnmeteredUInt64Value(0),
		max: NewUnmeteredUInt64Value(math.MaxUint64),
	},
	{
		Name:         sema.UInt128TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.UInt128Type),
		Convert:      ConvertUInt128,
		min:          NewUnmeteredUInt128ValueFromUint64(0),
		max:          NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
	},
	{
		Name:         sema.UInt256TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.UInt256Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertUInt256(gauge, value, locationRange)
		},
		min: NewUnmeteredUInt256ValueFromUint64(0),
		max: NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
	},
	{
		Name:         sema.Word8TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Word8Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertWord8(gauge, value, locationRange)
		},
		min: NewUnmeteredWord8Value(0),
		max: NewUnmeteredWord8Value(math.MaxUint8),
	},
	{
		Name:         sema.Word16TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Word16Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertWord16(gauge, value, locationRange)
		},
		min: NewUnmeteredWord16Value(0),
		max: NewUnmeteredWord16Value(math.MaxUint16),
	},
	{
		Name:         sema.Word32TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Word32Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertWord32(gauge, value, locationRange)
		},
		min: NewUnmeteredWord32Value(0),
		max: NewUnmeteredWord32Value(math.MaxUint32),
	},
	{
		Name:         sema.Word64TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Word64Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertWord64(gauge, value, locationRange)
		},
		min: NewUnmeteredWord64Value(0),
		max: NewUnmeteredWord64Value(math.MaxUint64),
	},
	{
		Name:         sema.Word128TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Word128Type),
		Convert:      ConvertWord128,
		min:          NewUnmeteredWord128ValueFromUint64(0),
		max:          NewUnmeteredWord128ValueFromBigInt(sema.Word128TypeMaxIntBig),
	},
	{
		Name:         sema.Word256TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Word256Type),
		Convert:      ConvertWord256,
		min:          NewUnmeteredWord256ValueFromUint64(0),
		max:          NewUnmeteredWord256ValueFromBigInt(sema.Word256TypeMaxIntBig),
	},
	{
		Name:         sema.Fix64TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.Fix64Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertFix64(gauge, value, locationRange)
		},
		min: NewUnmeteredFix64Value(math.MinInt64),
		max: NewUnmeteredFix64Value(math.MaxInt64),
	},
	{
		Name:         sema.UFix64TypeName,
		FunctionType: sema.NumberConversionFunctionType(sema.UFix64Type),
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertUFix64(gauge, value, locationRange)
		},
		min: NewUnmeteredUFix64Value(0),
		max: NewUnmeteredUFix64Value(math.MaxUint64),
	},
	{
		Name:         sema.AddressTypeName,
		FunctionType: sema.AddressConversionFunctionType,
		Convert: func(gauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
			return ConvertAddress(gauge, value, locationRange)
		},
		nestedVariables: []struct {
			Name  string
			Value Value
		}{
			// Converter functions are static functions.
			{
				Name: sema.AddressTypeFromBytesFunctionName,
				Value: NewUnmeteredStaticHostFunctionValue(
					sema.AddressTypeFromBytesFunctionType,
					AddressFromBytes,
				),
			},
			{
				Name: sema.AddressTypeFromStringFunctionName,
				Value: NewUnmeteredStaticHostFunctionValue(
					sema.AddressTypeFromStringFunctionType,
					AddressFromString,
				),
			},
		},
	},
	{
		Name:         sema.PublicPathType.Name,
		FunctionType: sema.PublicPathConversionFunctionType,
		Convert: func(gauge common.MemoryGauge, value Value, _ LocationRange) Value {
			return newPathFromStringValue(gauge, common.PathDomainPublic, value)
		},
	},
	{
		Name:         sema.PrivatePathType.Name,
		FunctionType: sema.PrivatePathConversionFunctionType,
		Convert: func(gauge common.MemoryGauge, value Value, _ LocationRange) Value {
			return newPathFromStringValue(gauge, common.PathDomainPrivate, value)
		},
	},
	{
		Name:         sema.StoragePathType.Name,
		FunctionType: sema.StoragePathConversionFunctionType,
		Convert: func(gauge common.MemoryGauge, value Value, _ LocationRange) Value {
			return newPathFromStringValue(gauge, common.PathDomainStorage, value)
		},
	},
}

func lookupInterface(typeConverter TypeConverter, typeID string) (*sema.InterfaceType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(typeConverter, typeID)
	// if the typeID is invalid, return nil
	if err != nil {
		return nil, err
	}

	typ, err := typeConverter.GetInterfaceType(location, qualifiedIdentifier, TypeID(typeID))
	if err != nil {
		return nil, err
	}

	return typ, nil
}

func lookupComposite(typeConverter TypeConverter, typeID string) (*sema.CompositeType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(typeConverter, typeID)
	// if the typeID is invalid, return nil
	if err != nil {
		return nil, err
	}

	typ, err := typeConverter.GetCompositeType(location, qualifiedIdentifier, TypeID(typeID))
	if err != nil {
		return nil, err
	}

	return typ, nil
}

func lookupEntitlement(typeConverter TypeConverter, typeID string) (*sema.EntitlementType, error) {
	_, _, err := common.DecodeTypeID(typeConverter, typeID)
	// if the typeID is invalid, return nil
	if err != nil {
		return nil, err
	}

	typ, err := typeConverter.GetEntitlementType(common.TypeID(typeID))
	if err != nil {
		return nil, err
	}

	return typ, nil
}

func init() {

	converterNames := make(map[string]struct{}, len(ConverterDeclarations))

	for _, converterDeclaration := range ConverterDeclarations {
		converterNames[converterDeclaration.Name] = struct{}{}
	}

	for _, numberType := range sema.AllNumberTypes {

		// Only leaf number types require a converter,
		// "hierarchy" number types don't need one

		switch numberType {
		case sema.NumberType, sema.SignedNumberType,
			sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType,
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

		if _, ok := fromBigEndianBytesFunctionValues[typeName]; !ok {
			panic(fmt.Sprintf("missing fromBigEndianBytes implementation for number type: %s", numberType))
		}
	}

	// We assign this here because it depends on the interpreter, so this breaks the initialization cycle

	// All of the following methods are static functions.
	defineBaseValue(
		BaseActivation,
		sema.DictionaryTypeFunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.DictionaryTypeFunctionType,
			dictionaryTypeFunction,
		))

	defineBaseValue(
		BaseActivation,
		sema.CompositeTypeFunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.CompositeTypeFunctionType,
			compositeTypeFunction,
		),
	)

	defineBaseValue(
		BaseActivation,
		sema.ReferenceTypeFunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.ReferenceTypeFunctionType,
			referenceTypeFunction,
		),
	)

	defineBaseValue(
		BaseActivation,
		sema.FunctionTypeFunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.FunctionTypeFunctionType,
			functionTypeFunction,
		),
	)

	defineBaseValue(
		BaseActivation,
		sema.IntersectionTypeFunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.IntersectionTypeFunctionType,
			intersectionTypeFunction,
		),
	)
}

func dictionaryTypeFunction(invocation Invocation) Value {
	inter := invocation.InvocationContext

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
		!sema.IsSubType(
			MustConvertStaticToSemaType(keyType, inter),
			sema.HashableStructType,
		) {
		return Nil
	}

	return NewSomeValueNonCopying(
		inter,
		NewTypeValue(
			inter,
			NewDictionaryStaticType(
				inter,
				keyType,
				valueType,
			),
		),
	)
}

func referenceTypeFunction(invocation Invocation) Value {
	entitlementValues, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	typeValue, ok := invocation.Arguments[1].(TypeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	invocationContext := invocation.InvocationContext
	locationRange := invocation.LocationRange

	return ConstructReferenceStaticType(
		invocationContext,
		entitlementValues,
		locationRange,
		typeValue,
	)
}

func ConstructReferenceStaticType(
	invocationContext InvocationContext,
	entitlementValues *ArrayValue,
	locationRange LocationRange,
	typeValue TypeValue,
) Value {
	authorization := UnauthorizedAccess
	errInIteration := false
	entitlementsCount := entitlementValues.Count()

	if entitlementsCount > 0 {
		authorization = NewEntitlementSetAuthorization(
			invocationContext,
			func() []common.TypeID {
				entitlements := make([]common.TypeID, 0, entitlementsCount)
				entitlementValues.Iterate(
					invocationContext,
					func(element Value) (resume bool) {
						entitlementString, isString := element.(*StringValue)
						if !isString {
							errInIteration = true
							return false
						}

						_, err := lookupEntitlement(invocationContext, entitlementString.Str)
						if err != nil {
							errInIteration = true
							return false
						}
						entitlements = append(entitlements, common.TypeID(entitlementString.Str))

						return true
					},
					false,
					locationRange,
				)
				return entitlements
			},
			entitlementsCount,
			sema.Conjunction,
		)
	}

	if errInIteration {
		return Nil
	}

	return NewSomeValueNonCopying(
		invocationContext,
		NewTypeValue(
			invocationContext,
			NewReferenceStaticType(
				invocationContext,
				authorization,
				typeValue.Type,
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

	composite, err := lookupComposite(invocation.InvocationContext, typeID)
	if err != nil {
		return Nil
	}

	return NewSomeValueNonCopying(
		invocation.InvocationContext,
		NewTypeValue(
			invocation.InvocationContext,
			ConvertSemaToStaticType(invocation.InvocationContext, composite),
		),
	)
}

func functionTypeFunction(invocation Invocation) Value {
	interpreter := invocation.InvocationContext

	parameters, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	typeValue, ok := invocation.Arguments[1].(TypeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	returnType := MustConvertStaticToSemaType(typeValue.Type, interpreter)

	var parameterTypes []sema.Parameter
	parameterCount := parameters.Count()
	if parameterCount > 0 {
		parameterTypes = make([]sema.Parameter, 0, parameterCount)
		parameters.Iterate(
			interpreter,
			func(param Value) bool {
				semaType := MustConvertStaticToSemaType(param.(TypeValue).Type, interpreter)
				parameterTypes = append(
					parameterTypes,
					sema.Parameter{
						TypeAnnotation: sema.NewTypeAnnotation(semaType),
					},
				)

				// Continue iteration
				return true
			},
			false,
			invocation.LocationRange,
		)
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

func intersectionTypeFunction(invocation Invocation) Value {
	intersectionIDs, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var staticIntersections []*InterfaceStaticType
	var semaIntersections []*sema.InterfaceType

	count := intersectionIDs.Count()
	if count > 0 {
		staticIntersections = make([]*InterfaceStaticType, 0, count)
		semaIntersections = make([]*sema.InterfaceType, 0, count)

		var invalidIntersectionID bool
		intersectionIDs.Iterate(
			invocation.InvocationContext,
			func(typeID Value) bool {
				typeIDValue, ok := typeID.(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				intersectedInterface, err := lookupInterface(invocation.InvocationContext, typeIDValue.Str)
				if err != nil {
					invalidIntersectionID = true
					return true
				}

				staticIntersections = append(
					staticIntersections,
					ConvertSemaToStaticType(invocation.InvocationContext, intersectedInterface).(*InterfaceStaticType),
				)
				semaIntersections = append(semaIntersections, intersectedInterface)

				// Continue iteration
				return true
			},
			false,
			invocation.LocationRange,
		)

		// If there are any invalid interfaces,
		// then return nil
		if invalidIntersectionID {
			return Nil
		}
	}

	var invalidIntersectionType bool
	sema.CheckIntersectionType(
		invocation.InvocationContext,
		semaIntersections,
		func(_ func(*ast.IntersectionType) error) {
			invalidIntersectionType = true
		},
	)

	// If the intersection type would have failed to type-check statically,
	// then return nil
	if invalidIntersectionType {
		return Nil
	}

	return NewSomeValueNonCopying(
		invocation.InvocationContext,
		NewTypeValue(
			invocation.InvocationContext,
			NewIntersectionStaticType(
				invocation.InvocationContext,
				staticIntersections,
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
		convert := declaration.Convert
		converterFunctionValue := NewUnmeteredStaticHostFunctionValue(
			declaration.FunctionType,
			func(invocation Invocation) Value {
				return convert(invocation.InvocationContext, invocation.Arguments[0], invocation.LocationRange)
			},
		)

		addMember := func(name string, value Value) {
			if converterFunctionValue.NestedVariables == nil {
				converterFunctionValue.NestedVariables = map[string]Variable{}
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

		fromStringVal := fromStringFunctionValues[declaration.Name]

		addMember(sema.FromStringFunctionName, fromStringVal.hostFunction)

		fromBigEndianBytesVal := fromBigEndianBytesFunctionValues[declaration.Name]

		addMember(sema.FromBigEndianBytesFunctionName, fromBigEndianBytesVal.hostFunction)

		if declaration.nestedVariables != nil {
			for _, variable := range declaration.nestedVariables {
				addMember(variable.Name, variable.Value)
			}
		}

		converterFuncValues[index] = converterFunction{
			name:      declaration.Name,
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
// They are also static functions.
var runtimeTypeConstructors = []runtimeTypeConstructor{
	{
		name: sema.OptionalTypeFunctionName,
		converter: NewUnmeteredStaticHostFunctionValue(
			sema.OptionalTypeFunctionType,
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return NewTypeValue(
					invocation.InvocationContext,
					NewOptionalStaticType(
						invocation.InvocationContext,
						typeValue.Type,
					),
				)
			},
		),
	},
	{
		name: sema.VariableSizedArrayTypeFunctionName,
		converter: NewUnmeteredStaticHostFunctionValue(
			sema.VariableSizedArrayTypeFunctionType,
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return NewTypeValue(
					invocation.InvocationContext,
					//nolint:gosimple
					NewVariableSizedStaticType(
						invocation.InvocationContext,
						typeValue.Type,
					),
				)
			},
		),
	},
	{
		name: sema.ConstantSizedArrayTypeFunctionName,
		converter: NewUnmeteredStaticHostFunctionValue(
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
					invocation.InvocationContext,
					NewConstantSizedStaticType(
						invocation.InvocationContext,
						typeValue.Type,
						int64(sizeValue.ToInt(invocation.LocationRange)),
					),
				)
			},
		),
	},
	{
		name: sema.CapabilityTypeFunctionName,
		converter: NewUnmeteredStaticHostFunctionValue(
			sema.CapabilityTypeFunctionType,
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				ty := typeValue.Type
				// Capabilities must hold references
				_, ok = ty.(*ReferenceStaticType)
				if !ok {
					return Nil
				}

				return NewSomeValueNonCopying(
					invocation.InvocationContext,
					NewTypeValue(
						invocation.InvocationContext,
						NewCapabilityStaticType(
							invocation.InvocationContext,
							ty,
						),
					),
				)
			},
		),
	},
	{
		name: sema.InclusiveRangeTypeFunctionName,
		converter: NewUnmeteredStaticHostFunctionValue(
			sema.InclusiveRangeTypeFunctionType,
			func(invocation Invocation) Value {
				typeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				inter := invocation.InvocationContext

				ty := typeValue.Type
				// InclusiveRanges must hold integers
				elemSemaTy := MustConvertStaticToSemaType(ty, inter)
				if !sema.IsSameTypeKind(elemSemaTy, sema.IntegerType) {
					return Nil
				}

				return NewSomeValueNonCopying(
					inter,
					NewTypeValue(
						inter,
						NewInclusiveRangeStaticType(
							inter,
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
// It's also a static function.
var typeFunction = NewUnmeteredStaticHostFunctionValue(
	sema.MetaTypeFunctionType,
	func(invocation Invocation) Value {
		typeParameterPair := invocation.TypeParameterTypes.Oldest()
		if typeParameterPair == nil {
			panic(errors.NewUnreachableError())
		}

		ty := typeParameterPair.Value

		staticType := ConvertSemaToStaticType(invocation.InvocationContext, ty)
		return NewTypeValue(invocation.InvocationContext, staticType)
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

func IsSubType(typeConverter TypeConverter, subType StaticType, superType StaticType) bool {
	if superType == PrimitiveStaticTypeAny {
		return true
	}

	// This is an optimization: If the static types are equal, then no need to check further.
	// i.e: Saves the conversion time.
	if subType.Equal(superType) {
		return true
	}

	semaType := MustConvertStaticToSemaType(superType, typeConverter)

	return IsSubTypeOfSemaType(typeConverter, subType, semaType)
}

func IsSubTypeOfSemaType(typeConverter TypeConverter, staticSubType StaticType, superType sema.Type) bool {
	if superType == sema.AnyType {
		return true
	}

	// Optimization: Implement subtyping for common cases directly,
	// without converting the subtype to a sema type.

	switch staticSubType := staticSubType.(type) {
	case *OptionalStaticType:
		if superType, ok := superType.(*sema.OptionalType); ok {
			return IsSubTypeOfSemaType(typeConverter, staticSubType.Type, superType.Type)
		}

		switch superType {
		case sema.AnyStructType, sema.AnyResourceType:
			return IsSubTypeOfSemaType(typeConverter, staticSubType.Type, superType)
		}

		return superType == sema.AnyStructType
	}

	semaSubType := MustConvertStaticToSemaType(staticSubType, typeConverter)

	return sema.IsSubType(semaSubType, superType)
}

func domainPaths(context StorageContext, address common.Address, domain common.PathDomain) []Value {
	storageMap := context.Storage().GetDomainStorageMap(context, address, domain.StorageDomain(), false)
	if storageMap == nil {
		return []Value{}
	}
	iterator := storageMap.Iterator(context)
	var paths []Value

	count := storageMap.Count()
	if count > 0 {
		paths = make([]Value, 0, count)
		for key := iterator.NextKey(); key != nil; key = iterator.NextKey() {
			// TODO: unfortunately, the iterator only returns an atree.Value, not a StorageMapKey
			identifier := string(key.(StringAtreeValue))
			path := NewPathValue(context, domain, identifier)
			paths = append(paths, path)
		}
	}
	return paths
}

func accountPaths(
	context ArrayCreationContext,
	addressValue AddressValue,
	locationRange LocationRange,
	domain common.PathDomain,
	pathType StaticType,
) *ArrayValue {
	address := addressValue.ToAddress()
	values := domainPaths(context, address, domain)
	return NewArrayValue(
		context,
		locationRange,
		NewVariableSizedStaticType(context, pathType),
		common.ZeroAddress,
		values...,
	)
}

func publicAccountPaths(
	context ArrayCreationContext,
	addressValue AddressValue,
	locationRange LocationRange,
) *ArrayValue {
	return accountPaths(
		context,
		addressValue,
		locationRange,
		common.PathDomainPublic,
		PrimitiveStaticTypePublicPath,
	)
}

func storageAccountPaths(
	context ArrayCreationContext,
	addressValue AddressValue,
	locationRange LocationRange,
) *ArrayValue {
	return accountPaths(
		context,
		addressValue,
		locationRange,
		common.PathDomainStorage,
		PrimitiveStaticTypeStoragePath,
	)
}

func (interpreter *Interpreter) RecordStorageMutation() {
	if interpreter.SharedState.inStorageIteration {
		interpreter.SharedState.storageMutatedDuringIteration = true
	}
}

func newStorageIterationFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	functionType *sema.FunctionType,
	addressValue AddressValue,
	domain common.PathDomain,
	pathType sema.Type,
) BoundFunctionValue {

	address := addressValue.ToAddress()

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		functionType,
		func(_ *SimpleCompositeValue, invocation Invocation) Value {
			invocationContext := invocation.InvocationContext
			locationRange := invocation.LocationRange
			arguments := invocation.Arguments

			return AccountStorageIterate(
				invocationContext,
				arguments,
				address,
				domain,
				pathType,
				locationRange,
			)
		},
	)
}

func AccountStorageIterate(
	invocationContext InvocationContext,
	arguments []Value,
	address common.Address,
	domain common.PathDomain,
	pathType sema.Type,
	locationRange LocationRange,
) Value {
	fn, ok := arguments[0].(FunctionValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	storage := invocationContext.Storage()
	storageMap := storage.GetDomainStorageMap(invocationContext, address, domain.StorageDomain(), false)
	if storageMap == nil {
		// if nothing is stored, no iteration is required
		return Void
	}
	storageIterator := storageMap.Iterator(invocationContext)

	inIteration := invocationContext.InStorageIteration()
	invocationContext.SetInStorageIteration(true)
	defer func() {
		invocationContext.SetInStorageIteration(inIteration)
	}()

	for key, value := storageIterator.Next(); key != nil && value != nil; key, value = storageIterator.Next() {

		staticType := value.StaticType(invocationContext)

		// Perform a forced value de-referencing to see if the associated type is not broken.
		// If broken, skip this value from the iteration.
		valueError := checkValue(
			invocationContext,
			value,
			staticType,
			locationRange,
		)

		if valueError != nil {
			continue
		}

		// TODO: unfortunately, the iterator only returns an atree.Value, not a StorageMapKey
		identifier := string(key.(StringAtreeValue))
		pathValue := NewPathValue(invocationContext, domain, identifier)
		runtimeType := NewTypeValue(invocationContext, staticType)

		arguments := []Value{pathValue, runtimeType}
		invocationArgumentTypes := []sema.Type{pathType, sema.MetaType}

		result := invocationContext.InvokeFunction(
			fn,
			arguments,
			invocationArgumentTypes,
			locationRange,
		)

		shouldContinue, ok := result.(BoolValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		if !shouldContinue {
			break
		}

		// It is not safe to check this at the beginning of the loop
		// (i.e. on the next invocation of the callback),
		// because if the mutation performed in the callback reorganized storage
		// such that the iteration pointer is now at the end,
		// we will not invoke the callback again but will still silently skip elements of storage.
		//
		// In order to be safe, we perform this check here to effectively enforce
		// that users return `false` from their callback in all cases where storage is mutated.
		if invocationContext.StorageMutatedDuringIteration() {
			panic(StorageMutatedDuringIterationError{
				LocationRange: locationRange,
			})
		}

	}

	return Void
}

func (interpreter *Interpreter) InvokeFunction(
	fn FunctionValue,
	arguments []Value,
	invocationArgumentTypes []sema.Type,
	locationRange LocationRange,
) Value {
	fnType := fn.FunctionType()
	parameterTypes := fnType.ParameterTypes()
	returnType := fnType.ReturnTypeAnnotation.Type

	result := invokeFunctionValue(
		interpreter,
		fn,
		arguments,
		nil,
		invocationArgumentTypes,
		parameterTypes,
		returnType,
		nil,
		locationRange,
	)
	return result
}

func checkValue(
	context StoredValueCheckContext,
	value Value,
	staticType StaticType,
	locationRange LocationRange,
) (valueError error) {

	defer func() {
		if r := recover(); r != nil {
			rootError := r
			for {
				switch err := r.(type) {
				case errors.UserError, errors.ExternalError:
					valueError = err.(error)
					return
				case xerrors.Wrapper:
					r = err.Unwrap()
				default:
					panic(rootError)
				}
			}
		}
	}()

	// Here, the value at the path could be either:
	//	1) The actual stored value (storage path)
	//	2) A capability to the value at the storage (private/public paths)

	if capability, ok := value.(*IDCapabilityValue); ok {
		// If, the value is a capability, try to load the value at the capability target.
		// However, borrow type is not statically known.
		// So take the borrow type from the value itself

		// Capability values always have a `CapabilityStaticType` static type.
		borrowType := staticType.(*CapabilityStaticType).BorrowType

		var borrowSemaType sema.Type
		borrowSemaType, valueError = ConvertStaticToSemaType(context, borrowType)
		if valueError != nil {
			return valueError
		}

		referenceType, ok := borrowSemaType.(*sema.ReferenceType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		capabilityCheckHandler := context.GetCapabilityCheckHandler()

		_ = capabilityCheckHandler(
			context,
			locationRange,
			capability.address,
			capability.ID,
			referenceType,
			referenceType,
		)

	} else {
		// For all other values, trying to load the type is sufficient.
		// Here it is only interested in whether the type can be properly loaded.
		_, valueError = ConvertStaticToSemaType(context, staticType)
	}

	return
}

func authAccountStorageSaveFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		sema.Account_StorageTypeSaveFunctionType,
		func(_ *SimpleCompositeValue, invocation Invocation) Value {
			interpreter := invocation.InvocationContext
			arguments := invocation.Arguments
			locationRange := invocation.LocationRange

			return AccountStorageSave(
				interpreter,
				arguments,
				addressValue,
				locationRange,
			)
		},
	)
}

func AccountStorageSave(
	context InvocationContext,
	arguments []Value,
	addressValue AddressValue,
	locationRange LocationRange,
) Value {
	value := arguments[0]

	path, ok := arguments[1].(PathValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	domain := path.Domain.StorageDomain()
	identifier := path.Identifier

	// Prevent an overwrite

	storageMapKey := StringStorageMapKey(identifier)

	address := addressValue.ToAddress()

	if StoredValueExists(context, address, domain, storageMapKey) {
		panic(
			OverwriteError{
				Address:       addressValue,
				Path:          path,
				LocationRange: locationRange,
			},
		)
	}

	value = value.Transfer(
		context,
		locationRange,
		atree.Address(address),
		true,
		nil,
		nil,
		true, // value is standalone because it is from invocation.Arguments[0].
	)

	// Write new value

	context.WriteStored(
		address,
		domain,
		storageMapKey,
		value,
	)

	return Void
}

func authAccountStorageTypeFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		sema.Account_StorageTypeTypeFunctionType,
		func(_ *SimpleCompositeValue, invocation Invocation) Value {
			interpreter := invocation.InvocationContext
			arguments := invocation.Arguments

			return AccountStorageType(
				interpreter,
				arguments,
				address,
			)
		},
	)
}

func AccountStorageType(
	interpreter InvocationContext,
	arguments []Value,
	address common.Address,
) Value {
	path, ok := arguments[0].(PathValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	domain := path.Domain.StorageDomain()
	identifier := path.Identifier

	storageMapKey := StringStorageMapKey(identifier)

	value := interpreter.ReadStored(address, domain, storageMapKey)

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
}

func authAccountStorageLoadFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {
	const clear = true
	return authAccountReadFunction(
		context,
		storageValue,
		addressValue,
		sema.Account_StorageTypeLoadFunctionType,
		clear,
	)
}

func authAccountStorageCopyFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {
	const clear = false
	return authAccountReadFunction(
		context,
		storageValue,
		addressValue,
		sema.Account_StorageTypeCopyFunctionType,
		clear,
	)
}

func authAccountReadFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
	functionType *sema.FunctionType,
	clear bool,
) BoundFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		functionType,
		func(_ *SimpleCompositeValue, invocation Invocation) Value {
			invocationContext := invocation.InvocationContext
			arguments := invocation.Arguments
			locationRange := invocation.LocationRange

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			typeParameter := typeParameterPair.Value

			return AccountStorageRead(
				invocationContext,
				arguments,
				typeParameter,
				address,
				clear,
				locationRange,
			)
		},
	)
}

func AccountStorageRead(
	invocationContext InvocationContext,
	arguments []Value,
	typeParameter sema.Type,
	address common.Address,
	clear bool,
	locationRange LocationRange,
) Value {
	path, ok := arguments[0].(PathValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	domain := path.Domain.StorageDomain()
	identifier := path.Identifier

	storageMapKey := StringStorageMapKey(identifier)

	value := invocationContext.ReadStored(address, domain, storageMapKey)

	if value == nil {
		return Nil
	}

	// If there is value stored for the given path,
	// check that it satisfies the type given as the type argument.

	valueStaticType := value.StaticType(invocationContext)

	if !IsSubTypeOfSemaType(invocationContext, valueStaticType, typeParameter) {
		valueSemaType := MustConvertStaticToSemaType(valueStaticType, invocationContext)

		panic(ForceCastTypeMismatchError{
			ExpectedType:  typeParameter,
			ActualType:    valueSemaType,
			LocationRange: locationRange,
		})
	}

	// We could also pass remove=true and the storable stored in storage,
	// but passing remove=false here and writing nil below has the same effect
	// TODO: potentially refactor and get storable in storage, pass it and remove=true
	transferredValue := value.Transfer(
		invocationContext,
		locationRange,
		atree.Address{},
		false,
		nil,
		nil,
		false, // value is an element in storage map because it is from "ReadStored".
	)

	// Remove the value from storage,
	// but only if the type check succeeded.
	if clear {
		invocationContext.WriteStored(
			address,
			domain,
			storageMapKey,
			nil,
		)
	}

	return NewSomeValueNonCopying(invocationContext, transferredValue)
}

func authAccountStorageBorrowFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		sema.Account_StorageTypeBorrowFunctionType,
		func(_ *SimpleCompositeValue, invocation Invocation) Value {
			invocationContext := invocation.InvocationContext
			arguments := invocation.Arguments
			typeParameterPair := invocation.TypeParameterTypes.Oldest().Value
			locationRange := invocation.LocationRange

			return AccountStorageBorrow(
				invocationContext,
				arguments,
				typeParameterPair,
				address,
				locationRange,
			)
		},
	)
}

func AccountStorageBorrow(
	invocationContext InvocationContext,
	arguments []Value,
	typeParameter sema.Type,
	address common.Address,
	locationRange LocationRange,
) Value {
	path, ok := arguments[0].(PathValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	referenceType, ok := typeParameter.(*sema.ReferenceType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	reference := NewStorageReferenceValue(
		invocationContext,
		ConvertSemaAccessToStaticAuthorization(invocationContext, referenceType.Authorization),
		address,
		path,
		referenceType.Type,
	)

	// Attempt to dereference,
	// which reads the stored value
	// and performs a dynamic type check

	value, err := reference.dereference(invocationContext, locationRange)
	if err != nil {
		panic(err)
	}
	if value == nil {
		return Nil
	}

	return NewSomeValueNonCopying(invocationContext, reference)
}

func authAccountStorageCheckFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {

	// Converted addresses can be cached and don't have to be recomputed on each function invocation
	address := addressValue.ToAddress()

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		sema.Account_StorageTypeCheckFunctionType,
		func(_ *SimpleCompositeValue, invocation Invocation) Value {
			invocationContext := invocation.InvocationContext
			arguments := invocation.Arguments

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}
			typeParameter := typeParameterPair.Value

			return AccountStorageCheck(
				invocationContext,
				address,
				arguments,
				typeParameter,
			)
		},
	)
}

func AccountStorageCheck(
	invocationContext InvocationContext,
	address common.Address,
	arguments []Value,
	typeParameter sema.Type,
) Value {
	path, ok := arguments[0].(PathValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	domain := path.Domain.StorageDomain()
	identifier := path.Identifier

	storageMapKey := StringStorageMapKey(identifier)

	value := invocationContext.ReadStored(address, domain, storageMapKey)

	if value == nil {
		return FalseValue
	}

	// If there is value stored for the given path,
	// check that it satisfies the type given as the type argument.

	valueStaticType := value.StaticType(invocationContext)

	return BoolValue(IsSubTypeOfSemaType(invocationContext, valueStaticType, typeParameter))
}

func (interpreter *Interpreter) GetEntitlementType(typeID common.TypeID) (*sema.EntitlementType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(interpreter, string(typeID))
	if err != nil {
		return nil, err
	}

	if location == nil {
		ty := sema.BuiltinEntitlements[qualifiedIdentifier]
		if ty == nil {
			return nil, TypeLoadingError{
				TypeID: typeID,
			}
		}

		return ty, nil
	}

	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	ty := elaboration.EntitlementType(typeID)
	if ty == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	return ty, nil
}

func (interpreter *Interpreter) GetEntitlementMapType(typeID common.TypeID) (*sema.EntitlementMapType, error) {
	location, qualifiedIdentifier, err := common.DecodeTypeID(interpreter, string(typeID))
	if err != nil {
		return nil, err
	}

	if location == nil {
		ty := sema.BuiltinEntitlementMappings[qualifiedIdentifier]
		if ty == nil {
			return nil, TypeLoadingError{
				TypeID: typeID,
			}
		}

		return ty, nil
	}

	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	ty := elaboration.EntitlementMapType(typeID)
	if ty == nil {
		return nil, TypeLoadingError{
			TypeID: typeID,
		}
	}

	return ty, nil
}

func MustConvertStaticAuthorizationToSemaAccess(
	handler StaticAuthorizationConversionHandler,
	auth Authorization,
) sema.Access {
	access, err := ConvertStaticAuthorizationToSemaAccess(auth, handler)
	if err != nil {
		panic(err)
	}
	return access
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

func (interpreter *Interpreter) AllElaborations() (elaborations map[common.Location]*sema.Elaboration) {

	elaborations = map[common.Location]*sema.Elaboration{}

	allInterpreters := interpreter.SharedState.allInterpreters

	locations := make([]common.Location, 0, len(allInterpreters))

	for location := range allInterpreters { //nolint:maprange
		locations = append(locations, location)
	}

	sort.Slice(locations, func(i, j int) bool {
		a := locations[i]
		b := locations[j]
		return a.ID() < b.ID()
	})

	for _, location := range locations {
		elaboration := interpreter.getElaboration(location)
		if elaboration == nil {
			panic(errors.NewUnexpectedError("missing elaboration for location %s", location))
		}
		elaborations[location] = elaboration
	}

	return
}

func (interpreter *Interpreter) GetContractValue(contractLocation common.AddressLocation) (*CompositeValue, error) {
	inter := interpreter.EnsureLoaded(contractLocation)
	return inter.GetContractComposite(contractLocation)
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
	contractValue, ok := contractGlobal.GetValue(interpreter).(*CompositeValue)
	if !ok {
		return nil, NotDeclaredError{
			ExpectedKind: common.DeclarationKindContract,
			Name:         contractLocation.Name,
		}
	}

	return contractValue, nil
}

func GetNativeCompositeValueComputedFields(qualifiedIdentifier string) map[string]ComputedField {
	switch qualifiedIdentifier {
	case sema.PublicKeyType.Identifier:
		return map[string]ComputedField{
			sema.PublicKeyTypePublicKeyFieldName: func(
				context ValueTransferContext,
				locationRange LocationRange,
				v *CompositeValue,
			) Value {
				publicKeyValue := v.GetField(context, sema.PublicKeyTypePublicKeyFieldName)
				return publicKeyValue.Transfer(
					context,
					locationRange,
					atree.Address{},
					false,
					nil,
					nil,
					false,
				)
			},
		}
	}

	return nil
}

func GetCompositeValueComputedFields(v *CompositeValue) map[string]ComputedField {

	var computedFields map[string]ComputedField
	if v.Location == nil {
		computedFields = GetNativeCompositeValueComputedFields(v.QualifiedIdentifier)
		if computedFields != nil {
			return computedFields
		}
	}

	// TODO: add handler to config

	return nil
}

func GetCompositeValueInjectedFields(context MemberAccessibleContext, v *CompositeValue) map[string]Value {
	injectedCompositeFieldsHandler := context.InjectedCompositeFieldsHandler()
	if injectedCompositeFieldsHandler == nil {
		return nil
	}

	return injectedCompositeFieldsHandler(
		context,
		v.Location,
		v.QualifiedIdentifier,
		v.Kind,
	)
}

func (interpreter *Interpreter) GetCompositeValueFunctions(
	v *CompositeValue,
	locationRange LocationRange,
) *FunctionOrderedMap {

	var functions *FunctionOrderedMap

	typeID := v.TypeID()

	sharedState := interpreter.SharedState

	compositeValueFunctionsHandler := sharedState.Config.CompositeValueFunctionsHandler
	if compositeValueFunctionsHandler != nil {
		functions = compositeValueFunctionsHandler(interpreter, locationRange, v)
		if functions != nil {
			return functions
		}
	}

	compositeCodes := sharedState.typeCodes.CompositeCodes
	return compositeCodes[typeID].CompositeFunctions
}

func (interpreter *Interpreter) GetCompositeType(
	location common.Location,
	qualifiedIdentifier string,
	typeID TypeID,
) (*sema.CompositeType, error) {
	var compositeType *sema.CompositeType
	if location == nil {
		compositeType = sema.NativeCompositeTypes[qualifiedIdentifier]
		if compositeType != nil {
			return compositeType, nil
		}
	}

	config := interpreter.SharedState.Config
	compositeTypeHandler := config.CompositeTypeHandler
	if compositeTypeHandler != nil {
		compositeType = compositeTypeHandler(location, typeID)
		if compositeType != nil {
			return compositeType, nil
		}
	}

	if location != nil {
		compositeType = interpreter.getUserCompositeType(location, typeID)
		if compositeType != nil {
			return compositeType, nil
		}
	}

	return nil, TypeLoadingError{
		TypeID: typeID,
	}
}

func (interpreter *Interpreter) getUserCompositeType(location common.Location, typeID TypeID) *sema.CompositeType {
	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		return nil
	}

	return elaboration.CompositeType(typeID)
}

func (interpreter *Interpreter) GetInterfaceType(
	location common.Location,
	qualifiedIdentifier string,
	typeID TypeID,
) (*sema.InterfaceType, error) {
	if location == nil {
		var interfaceType = sema.NativeInterfaceTypes[qualifiedIdentifier]
		if interfaceType != nil {
			return interfaceType, nil
		}
		return nil, InterfaceMissingLocationError{
			QualifiedIdentifier: qualifiedIdentifier,
		}
	}

	config := interpreter.SharedState.Config
	interfaceTypeHandler := config.InterfaceTypeHandler
	if interfaceTypeHandler != nil {
		interfaceType := interfaceTypeHandler(location, typeID)
		if interfaceType != nil {
			return interfaceType, nil
		}
	}

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

func getAccessOfMember(context ValueStaticTypeContext, self Value, identifier string) sema.Access {
	typ, err := ConvertStaticToSemaType(context, self.StaticType(context))
	// some values (like transactions) do not have types that can be looked up this way. These types
	// do not support entitled members, so their access is always unauthorized
	if err != nil {
		return sema.UnauthorizedAccess
	}
	member, hasMember := typ.GetMembers()[identifier]
	// certain values (like functions) have builtin members that are not present on the type
	// in such cases the access is always unauthorized
	if !hasMember {
		return sema.UnauthorizedAccess
	}
	return member.Resolve(context, identifier, ast.EmptyRange, func(err error) {}).Access
}

// getMember gets the member value by the given identifier from the given Value depending on its type.
// May return nil if the member does not exist.
func getMember(context MemberAccessibleContext, self Value, locationRange LocationRange, identifier string) Value {
	var result Value
	// When the accessed value has a type that supports the declaration of members
	// or is a built-in type that has members (`MemberAccessibleValue`),
	// then try to get the member for the given identifier.
	// For example, the built-in type `String` has a member "length",
	// and composite declarations may contain member declarations
	if memberAccessibleValue, ok := self.(MemberAccessibleValue); ok {
		result = memberAccessibleValue.GetMember(context, locationRange, identifier)
	}
	if result == nil {
		result = getBuiltinFunctionMember(context, self, identifier)
	}

	// NOTE: do not panic if the member is nil. This is a valid state.
	// For example, when a composite field is initialized with a force-assignment, the field's value is read.

	return result
}

func getBuiltinFunctionMember(context MemberAccessibleContext, self Value, identifier string) FunctionValue {
	switch identifier {
	case sema.IsInstanceFunctionName:
		return isInstanceFunction(context, self)
	case sema.GetTypeFunctionName:
		return getTypeFunction(context, self)
	default:
		return nil
	}
}

func isInstanceFunction(context FunctionCreationContext, self Value) FunctionValue {
	return NewBoundHostFunctionValue(
		context,
		self,
		sema.IsInstanceFunctionType,
		func(self Value, invocation Invocation) Value {
			invocationContext := invocation.InvocationContext

			firstArgument := invocation.Arguments[0]
			typeValue, ok := firstArgument.(TypeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return IsInstance(invocationContext, self, typeValue)
		},
	)
}

func IsInstance(invocationContext InvocationContext, self Value, typeValue TypeValue) Value {
	staticType := typeValue.Type

	// Values are never instances of unknown types
	if staticType == nil {
		return FalseValue
	}

	// NOTE: not invocation.Self, as that is only set for composite values
	selfType := self.StaticType(invocationContext)
	return BoolValue(
		IsSubType(invocationContext, selfType, staticType),
	)
}

func getTypeFunction(context FunctionCreationContext, self Value) FunctionValue {
	return NewBoundHostFunctionValue(
		context,
		self,
		sema.GetTypeFunctionType,
		func(self Value, invocation Invocation) Value {
			invocationContext := invocation.InvocationContext
			return ValueGetType(invocationContext, self)
		},
	)
}

func ValueGetType(context InvocationContext, self Value) Value {
	staticType := self.StaticType(context)
	return NewTypeValue(context, staticType)
}

func setMember(
	context ValueTransferContext,
	self Value,
	locationRange LocationRange,
	identifier string,
	value Value,
) bool {
	return self.(MemberAccessibleValue).SetMember(context, locationRange, identifier, value)
}

func ExpectType(
	context ValueStaticTypeContext,
	value Value,
	expectedType sema.Type,
	locationRange LocationRange,
) {
	valueStaticType := value.StaticType(context)

	if !IsSubTypeOfSemaType(context, valueStaticType, expectedType) {
		valueSemaType := MustConvertStaticToSemaType(valueStaticType, context)

		panic(TypeMismatchError{
			ExpectedType:  expectedType,
			ActualType:    valueSemaType,
			LocationRange: locationRange,
		})
	}
}

func checkContainerMutation(
	context ValueStaticTypeContext,
	elementType StaticType,
	element Value,
	locationRange LocationRange,
) {
	actualElementType := element.StaticType(context)

	if !IsSubType(context, actualElementType, elementType) {
		panic(ContainerMutationError{
			ExpectedType:  MustConvertStaticToSemaType(elementType, context),
			ActualType:    MustSemaTypeOfValue(element, context),
			LocationRange: locationRange,
		})
	}
}

func RemoveReferencedSlab(context StorageContext, storable atree.Storable) {
	slabIDStorable, ok := storable.(atree.SlabIDStorable)
	if !ok {
		return
	}

	slabID := atree.SlabID(slabIDStorable)
	err := context.Storage().Remove(slabID)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}

func (interpreter *Interpreter) MaybeValidateAtreeValue(v atree.Value) {
	config := interpreter.SharedState.Config

	if config.AtreeValueValidationEnabled {
		interpreter.ValidateAtreeValue(v)
	}
}

func (interpreter *Interpreter) MaybeValidateAtreeStorage() {
	config := interpreter.SharedState.Config

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
		case *ConstantSizedStaticType:
			return info.Equal(other.(StaticType))
		case *VariableSizedStaticType:
			return info.Equal(other.(StaticType))
		case *DictionaryStaticType:
			return info.Equal(other.(StaticType))
		case CompositeTypeInfo:
			return info.Equal(other)
		case EmptyTypeInfo:
			_, ok := other.(EmptyTypeInfo)
			return ok
		}
		panic(errors.NewUnreachableError())
	}

	defaultHIP := newHashInputProvider(interpreter, EmptyLocationRange)

	hip := func(value atree.Value, buffer []byte) ([]byte, error) {
		switch value := value.(type) {
		case StringAtreeValue:
			return StringAtreeValueHashInput(value, buffer)
		case Uint64AtreeValue:
			return Uint64AtreeValueHashInput(value, buffer)
		default:
			return defaultHIP(value, buffer)
		}
	}

	config := interpreter.SharedState.Config
	storage := config.Storage

	compare := func(storable, otherStorable atree.Storable) bool {
		value, err := storable.StoredValue(storage)
		if err != nil {
			panic(err)
		}

		switch value := value.(type) {
		case StringAtreeValue:
			equal, err := StringAtreeValueComparator(
				storage,
				value,
				otherStorable,
			)
			if err != nil {
				panic(err)
			}

			return equal

		case Uint64AtreeValue:
			equal, err := Uint64AtreeValueComparator(
				storage,
				value,
				otherStorable,
			)
			if err != nil {
				panic(err)
			}

			return equal

		case EquatableValue:
			otherValue := StoredValue(interpreter, otherStorable, storage)
			return value.Equal(interpreter, EmptyLocationRange, otherValue)

		default:
			// Not all values are comparable, assume valid for now
			return true
		}
	}

	atreeInliningEnabled := true

	switch value := value.(type) {
	case *atree.Array:
		err := atree.VerifyArray(value, value.Address(), value.Type(), tic, hip, atreeInliningEnabled)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		err = atree.VerifyArraySerialization(
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
		err := atree.VerifyMap(value, value.Address(), value.Type(), tic, hip, atreeInliningEnabled)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		err = atree.VerifyMapSerialization(
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

func (interpreter *Interpreter) MaybeTrackReferencedResourceKindedValue(referenceValue *EphemeralReferenceValue) {
	if value, ok := referenceValue.Value.(ReferenceTrackedResourceKindedValue); ok {
		interpreter.trackReferencedResourceKindedValue(value.ValueID(), referenceValue)
	}
}

func (interpreter *Interpreter) trackReferencedResourceKindedValue(
	id atree.ValueID,
	value *EphemeralReferenceValue,
) {
	values := interpreter.SharedState.referencedResourceKindedValues[id]
	if values == nil {
		values = map[*EphemeralReferenceValue]struct{}{}
		interpreter.SharedState.referencedResourceKindedValues[id] = values
	}
	values[value] = struct{}{}
}

// TODO: Remove the `destroyed` flag
func InvalidateReferencedResources(
	context ContainerMutationContext,
	value Value,
	locationRange LocationRange,
) {
	// skip non-resource typed values
	if !value.IsResourceKinded(context) {
		return
	}

	var valueID atree.ValueID

	switch value := value.(type) {
	case *CompositeValue:
		value.ForEachReadOnlyLoadedField(
			context,
			func(_ string, fieldValue Value) (resume bool) {
				InvalidateReferencedResources(context, fieldValue, locationRange)
				// continue iteration
				return true
			},
			locationRange,
		)
		valueID = value.ValueID()

	case *DictionaryValue:
		value.IterateReadOnlyLoaded(
			context,
			locationRange,
			func(_, value Value) (resume bool) {
				InvalidateReferencedResources(context, value, locationRange)
				return true
			},
		)
		valueID = value.ValueID()

	case *ArrayValue:
		value.IterateReadOnlyLoaded(
			context,
			func(element Value) (resume bool) {
				InvalidateReferencedResources(context, element, locationRange)
				return true
			},
			locationRange,
		)
		valueID = value.ValueID()

	case *SomeValue:
		InvalidateReferencedResources(context, value.value, locationRange)
		return

	default:
		// skip non-container typed values.
		return
	}

	values := context.ReferencedResourceKindedValues(valueID)
	if values == nil {
		return
	}

	for value := range values { //nolint:maprange
		value.Value = nil
	}

	// The old resource instances are already cleared/invalidated above.
	// So no need to track those stale resources anymore. We will not need to update/clear them again.
	// Therefore, remove them from the mapping.
	// This is only to allow GC. No impact to the behavior.
	context.ClearReferencedResourceKindedValues(valueID)
}

func (interpreter *Interpreter) ClearReferencedResourceKindedValues(valueID atree.ValueID) {
	delete(interpreter.SharedState.referencedResourceKindedValues, valueID)
}

func (interpreter *Interpreter) ReferencedResourceKindedValues(valueID atree.ValueID) map[*EphemeralReferenceValue]struct{} {
	return interpreter.SharedState.referencedResourceKindedValues[valueID]
}

// startResourceTracking starts tracking the life-span of a resource.
// A resource can only be associated with one variable at most, at a given time.
func (interpreter *Interpreter) startResourceTracking(
	value Value,
	variable Variable,
	identifier string,
	hasPosition ast.HasPosition,
) {

	if identifier == sema.SelfIdentifier {
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
	variable Variable,
	identifier string,
	hasPosition ast.HasPosition,
) {

	if identifier == sema.SelfIdentifier {
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
	if interpreter != nil {
		config := interpreter.SharedState.Config
		common.UseMemory(config.MemoryGauge, usage)
	}
	return nil
}

func (interpreter *Interpreter) DecodeStorable(
	decoder *cbor.StreamDecoder,
	slabID atree.SlabID,
	inlinedExtraData []atree.ExtraData,
) (
	atree.Storable,
	error,
) {
	return DecodeStorable(decoder, slabID, inlinedExtraData, interpreter)
}

func (interpreter *Interpreter) DecodeTypeInfo(decoder *cbor.StreamDecoder) (atree.TypeInfo, error) {
	return DecodeTypeInfo(decoder, interpreter)
}

func (interpreter *Interpreter) Storage() Storage {
	return interpreter.SharedState.Config.Storage
}

func capabilityBorrowFunction(
	context FunctionCreationContext,
	capabilityValue CapabilityValue,
	addressValue AddressValue,
	capabilityID UInt64Value,
	capabilityBorrowType *sema.ReferenceType,
) FunctionValue {

	return NewBoundHostFunctionValue(
		context,
		capabilityValue,
		sema.CapabilityTypeBorrowFunctionType(capabilityBorrowType),
		func(_ CapabilityValue, invocation Invocation) Value {
			invocationContext := invocation.InvocationContext
			locationRange := invocation.LocationRange
			typeParameterPair := invocation.TypeParameterTypes.Oldest()

			var typeParameter sema.Type
			if typeParameterPair != nil {
				typeParameter = typeParameterPair.Value
			}

			return CapabilityBorrow(
				invocationContext,
				typeParameter,
				addressValue,
				capabilityID,
				capabilityBorrowType,
				locationRange,
			)
		},
	)
}

func CapabilityBorrow(
	invocationContext InvocationContext,
	typeParameter sema.Type,
	addressValue AddressValue,
	capabilityID UInt64Value,
	capabilityBorrowType *sema.ReferenceType,
	locationRange LocationRange,
) Value {
	if capabilityID == InvalidCapabilityID {
		return Nil
	}

	var wantedBorrowType *sema.ReferenceType
	if typeParameter != nil {
		var ok bool
		wantedBorrowType, ok = typeParameter.(*sema.ReferenceType)
		if !ok {
			panic(errors.NewUnreachableError())
		}
	}

	borrowHandler := invocationContext.CapabilityBorrowHandler()

	referenceValue := borrowHandler(
		invocationContext,
		locationRange,
		addressValue,
		capabilityID,
		wantedBorrowType,
		capabilityBorrowType,
	)
	if referenceValue == nil {
		return Nil
	}
	return NewSomeValueNonCopying(invocationContext, referenceValue)
}

func capabilityCheckFunction(
	context FunctionCreationContext,
	capabilityValue CapabilityValue,
	addressValue AddressValue,
	capabilityID UInt64Value,
	capabilityBorrowType *sema.ReferenceType,
) FunctionValue {

	return NewBoundHostFunctionValue(
		context,
		capabilityValue,
		sema.CapabilityTypeCheckFunctionType(capabilityBorrowType),
		func(_ CapabilityValue, invocation Invocation) Value {

			if capabilityID == InvalidCapabilityID {
				return FalseValue
			}

			invocationContext := invocation.InvocationContext
			locationRange := invocation.LocationRange

			// NOTE: if a type argument is provided for the function,
			// use it *instead* of the type of the value (if any)

			var wantedBorrowType *sema.ReferenceType
			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair != nil {
				ty := typeParameterPair.Value
				var ok bool
				wantedBorrowType, ok = ty.(*sema.ReferenceType)
				if !ok {
					panic(errors.NewUnreachableError())
				}
			}

			capabilityCheckHandler := invocationContext.GetCapabilityCheckHandler()

			return capabilityCheckHandler(
				invocationContext,
				locationRange,
				addressValue,
				capabilityID,
				wantedBorrowType,
				capabilityBorrowType,
			)
		},
	)
}

func (interpreter *Interpreter) ValidateMutation(valueID atree.ValueID, locationRange LocationRange) {
	_, present := interpreter.SharedState.containerValueIteration[valueID]
	if !present {
		return
	}
	panic(ContainerMutatedDuringIterationError{
		LocationRange: locationRange,
	})
}

func (interpreter *Interpreter) WithMutationPrevention(valueID atree.ValueID, f func()) {
	if interpreter == nil {
		f()
		return
	}

	oldIteration, present := interpreter.SharedState.containerValueIteration[valueID]
	interpreter.SharedState.containerValueIteration[valueID] = struct{}{}

	f()

	if !present {
		delete(interpreter.SharedState.containerValueIteration, valueID)
	} else {
		interpreter.SharedState.containerValueIteration[valueID] = oldIteration
	}
}

func (interpreter *Interpreter) EnforceNotResourceDestruction(
	valueID atree.ValueID,
	locationRange LocationRange,
) {
	_, exists := interpreter.SharedState.destroyedResources[valueID]
	if exists {
		panic(DestroyedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (interpreter *Interpreter) WithResourceDestruction(
	valueID atree.ValueID,
	locationRange LocationRange,
	f func(),
) {
	interpreter.EnforceNotResourceDestruction(valueID, locationRange)

	interpreter.SharedState.destroyedResources[valueID] = struct{}{}

	f()
}

func checkResourceLoss(context ValueStaticTypeContext, value Value, locationRange LocationRange) {
	if !value.IsResourceKinded(context) {
		return
	}

	var resourceKindedValue ResourceKindedValue

	switch existingValue := value.(type) {
	case *CompositeValue:
		// A dedicated error is thrown when setting duplicate attachments.
		// So don't throw an error here.
		if existingValue.Kind == common.CompositeKindAttachment {
			return
		}
		resourceKindedValue = existingValue
	case ResourceKindedValue:
		resourceKindedValue = existingValue
	default:
		panic(errors.NewUnreachableError())
	}

	if !resourceKindedValue.isInvalidatedResource(context) {
		panic(ResourceLossError{
			LocationRange: locationRange,
		})
	}
}

func (interpreter *Interpreter) OnResourceOwnerChange(resource *CompositeValue, oldOwner common.Address, newOwner common.Address) {
	onResourceOwnerChange := interpreter.SharedState.Config.OnResourceOwnerChange
	if onResourceOwnerChange == nil {
		return
	}

	onResourceOwnerChange(interpreter, resource, oldOwner, newOwner)
}

func (interpreter *Interpreter) TracingEnabled() bool {
	return interpreter.SharedState.Config.TracingEnabled
}

func (interpreter *Interpreter) CheckInvalidatedResourceOrResourceReference(value Value, locationRange LocationRange) {
	checkInvalidatedResourceOrResourceReference(value, locationRange, interpreter)
}

func (interpreter *Interpreter) IsTypeInfoRecovered(location common.Location) bool {
	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		return false
	}

	return elaboration.IsRecovered
}

func (interpreter *Interpreter) AccountHandler() AccountHandlerFunc {
	return interpreter.SharedState.Config.AccountHandler
}

func (interpreter *Interpreter) InjectedCompositeFieldsHandler() InjectedCompositeFieldsHandlerFunc {
	return interpreter.SharedState.Config.InjectedCompositeFieldsHandler
}

func MaybeSetMutationDuringCapConIteration(context CapabilityControllerIterationContext, addressPath AddressPath) {
	iterations := context.GetCapabilityControllerIterations()
	if iterations[addressPath] > 0 {
		context.SetMutationDuringCapabilityControllerIteration()
	}
}

func (interpreter *Interpreter) GetMemberAccessContextForLocation(location common.Location) MemberAccessibleContext {
	return interpreter.ensureLoaded(location)
}

func (interpreter *Interpreter) GetResourceDestructionContextForLocation(location common.Location) ResourceDestructionContext {
	return interpreter.ensureLoaded(location)
}

func (interpreter *Interpreter) ensureLoaded(location common.Location) *Interpreter {
	if location == nil || interpreter.Location == location {
		return interpreter
	}

	return interpreter.EnsureLoaded(location)
}

func (interpreter *Interpreter) GetLocation() common.Location {
	return interpreter.Location
}

func (interpreter *Interpreter) SetAttachmentIteration(base *CompositeValue, state bool) (oldState bool) {
	oldSharedState := interpreter.SharedState.inAttachmentIteration(base)
	interpreter.SharedState.setAttachmentIteration(base, state)
	return oldSharedState
}

func (interpreter *Interpreter) GetCapabilityCheckHandler() CapabilityCheckHandlerFunc {
	return interpreter.SharedState.Config.CapabilityCheckHandler
}

func (interpreter *Interpreter) GetCapabilityControllerIterations() map[AddressPath]int {
	return interpreter.SharedState.CapabilityControllerIterations
}

func (interpreter *Interpreter) SetMutationDuringCapabilityControllerIteration() {
	interpreter.SharedState.MutationDuringCapabilityControllerIteration = true
}

func (interpreter *Interpreter) MutationDuringCapabilityControllerIteration() bool {
	return interpreter.SharedState.MutationDuringCapabilityControllerIteration
}

func (interpreter *Interpreter) ValidateAccountCapabilitiesGetHandler() ValidateAccountCapabilitiesGetHandlerFunc {
	return interpreter.SharedState.Config.ValidateAccountCapabilitiesGetHandler
}

func (interpreter *Interpreter) ValidateAccountCapabilitiesPublishHandler() ValidateAccountCapabilitiesPublishHandlerFunc {
	return interpreter.SharedState.Config.ValidateAccountCapabilitiesPublishHandler
}

func (interpreter *Interpreter) CapabilityBorrowHandler() CapabilityBorrowHandlerFunc {
	return interpreter.SharedState.Config.CapabilityBorrowHandler
}

func (interpreter *Interpreter) InStorageIteration() bool {
	return interpreter.SharedState.inStorageIteration
}

func (interpreter *Interpreter) SetInStorageIteration(inStorageIteration bool) {
	interpreter.SharedState.inStorageIteration = inStorageIteration
}

func (interpreter *Interpreter) StorageMutatedDuringIteration() bool {
	return interpreter.SharedState.storageMutatedDuringIteration
}

func (interpreter *Interpreter) GetMethod(value MemberAccessibleValue, name string, locationRange LocationRange) FunctionValue {
	return value.GetMethod(interpreter, locationRange, name)
}
