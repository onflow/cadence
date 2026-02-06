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
	goErrors "errors"
	"fmt"
	"math"
	"math/big"
	"slices"
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
)

type getterFunc func(allowMissing bool) (value Value, insertedPlaceholder *PlaceholderValue)

type getterSetter struct {
	target Value
	// allowMissing may be true when the got value is nil.
	// For example, this is the case when a field is initialized
	// with the force-assignment operator (which checks the existing value)
	get getterFunc
	set func(Value)
}

// Visit-methods for statement which return a non-nil value
// are treated like they are returning a value.

// OnEventEmittedFunc is a function that is triggered when an event is emitted by the program.
type OnEventEmittedFunc func(
	context ValueExportContext,
	eventType *sema.CompositeType,
	eventFields []Value,
) error

// OnStatementFunc is a function that is triggered when a statement is about to be executed.
type OnStatementFunc func(
	inter *Interpreter,
	statement ast.Statement,
)

func CombineOnStatementFuncs(funcs ...OnStatementFunc) OnStatementFunc {
	funcs = slices.DeleteFunc(funcs, func(f OnStatementFunc) bool {
		return f == nil
	})

	switch len(funcs) {
	case 0:
		return nil
	case 1:
		return funcs[0]
	default:
		return func(inter *Interpreter, statement ast.Statement) {
			for _, f := range funcs {
				f(inter, statement)
			}
		}
	}
}

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

// CapabilityBorrowHandlerFunc is a function that is used to borrow ID capabilities.
type CapabilityBorrowHandlerFunc func(
	context BorrowCapabilityControllerContext,
	address AddressValue,
	capabilityID UInt64Value,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) ReferenceValue

// CapabilityCheckHandlerFunc is a function that is used to check ID capabilities.
type CapabilityCheckHandlerFunc func(
	context CheckCapabilityControllerContext,
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
	address AddressValue,
	path PathValue,
	wantedBorrowType *sema.ReferenceType,
	capabilityBorrowType *sema.ReferenceType,
) (bool, error)

// ValidateAccountCapabilitiesPublishHandlerFunc is a function that is used to handle when a capability of an account is got.
type ValidateAccountCapabilitiesPublishHandlerFunc func(
	context AccountCapabilityPublishValidationContext,
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
	Program      *Program
	SharedState  *SharedState
	Globals      GlobalVariables
	activations  *VariableActivations
	Transactions []*HostFunctionValue
	interpreted  bool
	Tracer

	statement  ast.Statement
	expression ast.Expression
}

var _ common.MemoryGauge = &Interpreter{}
var _ common.ComputationGauge = &Interpreter{}
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

	var tracer Tracer
	if TracingEnabled {
		onRecordTrace := sharedState.Config.OnRecordTrace
		if onRecordTrace == nil {
			tracer = NoOpTracer{}
		} else {
			tracer = CallbackTracer(onRecordTrace)
		}
	}

	interpreter := &Interpreter{
		Program:     program,
		Location:    location,
		SharedState: sharedState,
		Tracer:      tracer,
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
	return interpreter.invokeVariableWithValidation(functionName, arguments, true)
}

func (interpreter *Interpreter) invokeVariableWithValidation(
	functionName string,
	arguments []Value,
	validateConvertAndBox bool,
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

	return InvokeExternallyWithValidation(
		interpreter,
		functionValue,
		functionType,
		arguments,
		validateConvertAndBox,
	)
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
	return InvokeExternallyWithValidation(
		context,
		functionValue,
		functionType,
		arguments,
		true,
	)
}

func InvokeExternallyWithValidation(
	context InvocationContext,
	functionValue FunctionValue,
	functionType *sema.FunctionType,
	arguments []Value,
	validateConvertAndBox bool,
) (
	result Value,
	err error,
) {
	preparedArguments, err := PrepareExternalInvocationArgumentsWithValidation(
		context,
		functionType,
		arguments,
		validateConvertAndBox,
	)
	if err != nil {
		return nil, err
	}

	var self *Value
	var base *EphemeralReferenceValue
	if boundFunc, ok := functionValue.(BoundFunctionValue); ok {
		self = boundFunc.SelfReference.ReferencedValue(context, true)
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
		functionType.ReturnTypeAnnotation.Type,
		LocationRange{},
	)

	return functionValue.Invoke(invocation), nil
}

func PrepareExternalInvocationArgumentsWithValidation(
	context InvocationContext,
	functionType *sema.FunctionType,
	arguments []Value,
	validateConvertAndBox bool,
) ([]Value, error) {

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

	var convertAndBox = convertAndBox
	if validateConvertAndBox {
		convertAndBox = ConvertAndBoxWithValidation
	}

	var preparedArguments []Value
	if argumentCount > 0 {
		preparedArguments = make([]Value, argumentCount)
		for i, argument := range arguments {
			parameterType := parameters[i].TypeAnnotation.Type

			// converts the argument into the parameter type declared by the function
			preparedArguments[i] = convertAndBox(context, argument, nil, parameterType)
		}
	}

	return preparedArguments, nil
}

// Invoke invokes a global function with the given arguments
func (interpreter *Interpreter) Invoke(functionName string, arguments ...Value) (value Value, err error) {

	// recover internal panics and return them as an error
	defer interpreter.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	return interpreter.invokeVariable(functionName, arguments)
}

// Deprecated: InvokeUncheckedForTestingOnly invokes a global function with the given arguments,
// without validating them.
// NOTE: FOR TESTING PURPOSES ONLY! Use Invoke instead
func (interpreter *Interpreter) InvokeUncheckedForTestingOnly(functionName string, arguments ...Value) (value Value, err error) {

	// recover internal panics and return them as an error
	defer interpreter.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	return interpreter.invokeVariableWithValidation(functionName, arguments, false)
}

// InvokeFunction invokes a function value with the given invocation
func InvokeFunction(errorHandler ErrorHandler, function FunctionValue, invocation Invocation) (value Value, err error) {

	// recover internal panics and return them as an error
	defer errorHandler.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	common.UseComputation(
		invocation.InvocationContext,
		common.FunctionInvocationComputationUsage,
	)

	value = function.Invoke(invocation)
	return
}

func (interpreter *Interpreter) InvokeTransaction(arguments []Value, signers ...Value) (err error) {

	// recover internal panics and return them as an error
	defer interpreter.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	const transactionIndex = 0

	functionValue := interpreter.Transactions[transactionIndex]

	transactionType := interpreter.Program.Elaboration.TransactionTypes[transactionIndex]
	functionType := transactionType.EntryPointFunctionType()

	_, err = InvokeExternally(
		interpreter,
		functionValue,
		functionType,
		common.Concat(arguments, signers),
	)

	return
}

func (interpreter *Interpreter) RecoverErrors(onError func(error)) {
	if r := recover(); r != nil {
		// Recover all errors, because FVM can directly invoke interpreter.
		err := AsCadenceError(r)

		locationRange := interpreter.LocationRange()

		if locatedError, ok := err.(HasLocationRange); ok {
			locatedError.SetLocationRange(locationRange)
		}

		// if the error is not yet an interpreter error, wrap it
		if _, ok := err.(Error); !ok {

			// wrap the error with position information if needed

			_, ok := err.(ast.HasPosition)
			if !ok {
				errRange := ast.NewUnmeteredRangeFromPositioned(locationRange)

				err = PositionedError{
					Err:   err,
					Range: errRange,
				}
			}

			err = Error{
				Err:      err,
				Location: locationRange.Location,
			}
		}

		interpreterErr := err.(Error)
		interpreterErr.StackTrace = interpreter.CallStackLocations()

		onError(interpreterErr)
	}
}

func (interpreter *Interpreter) LocationRange() LocationRange {
	var hasPos ast.HasPosition
	if interpreter.expression != nil {
		hasPos = interpreter.expression
	} else if interpreter.statement != nil {
		hasPos = interpreter.statement
	}

	return LocationRange{
		Location:    interpreter.Location,
		HasPosition: hasPos,
	}
}

func AsCadenceError(r any) error {
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
	return interpreter.SharedState.callStack.Invocations
}

func (interpreter *Interpreter) CallStackLocations() []LocationRange {
	callstack := interpreter.CallStack()
	if len(callstack) == 0 {
		return nil
	}

	locationRanges := make([]LocationRange, 0, len(callstack))
	for _, invocation := range callstack {
		locationRanges = append(locationRanges, invocation.LocationRange)
	}
	return locationRanges
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
				common.UseComputation(interpreter, common.StatementComputationUsage)

				result := interpreter.visitVariableDeclaration(declaration, false)

				// Global variables are lazily loaded. Therefore, start resource tracking also
				// lazily when the resource is used for the first time.
				// This is needed to support forward referencing.
				interpreter.startResourceTracking(
					result,
					variable,
					identifier,
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

	variable.SetValue(interpreter, value)

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
		resultValue := interpreter.resultValue(returnValue, returnType)
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
func (interpreter *Interpreter) resultValue(returnValue Value, returnType sema.Type) Value {
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

		panic(&ConditionError{
			ConditionKind: kind,
			Message:       message,
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
	interpreter.startResourceTracking(value, variable, identifier)

	return variable
}

// declareSelfVariable declares a special "self" variable in the latest scope
func (interpreter *Interpreter) declareSelfVariable(value Value) Variable {
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
	variable := NewSelfVariableWithValue(interpreter, value)
	interpreter.setVariable(identifier, variable)

	interpreter.startResourceTracking(value, variable, identifier)

	return variable
}

func (interpreter *Interpreter) visitAssignment(
	_ ast.TransferOperation,
	targetGetterSetter getterSetter, targetType sema.Type,
	valueExpression ast.Expression, valueType sema.Type,
) {
	// Evaluate the value, and assign it using the setter function

	// Here it is too early to check whether the existing value is a
	// valid non-nil resource (i.e: causing a resource loss), because
	// evaluating the `valueExpression` could change things, and
	// a `nil`/invalid resource at this point could be valid after
	// the evaluation of `valueExpression`.
	// Therefore, delay the checking of resource loss as much as possible,
	// and check it at the 'setter', at the point where the value is assigned.

	value := interpreter.evalExpression(valueExpression)

	transferredValue := TransferAndConvert(interpreter, value, valueType, targetType)

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
		base, self = AttachmentBaseAndSelfValues(
			declarationInterpreter,
			access,
			containingResourceComposite,
		)
		declarationInterpreter.declareVariable(sema.BaseIdentifier, base)
	}
	declarationInterpreter.declareSelfVariable(self)

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
		return interpreter.declareEnumLookupFunction(declaration.(*ast.CompositeDeclaration), lexicalScope)
	} else {
		return interpreter.declareNonEnumCompositeValue(declaration, lexicalScope)
	}
}

func (interpreter *Interpreter) declareNonEnumCompositeValue(
	declaration ast.CompositeLikeDeclaration,
	lexicalScope *VariableActivation,
) (
	scope *VariableActivation,
	variable Variable,
) {
	declarationInterpreter := interpreter

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

				if compositeType.Kind == common.CompositeKindResource &&
					invocationContext.GetLocation() != compositeType.Location {

					panic(&ResourceConstructionError{
						CompositeType: compositeType,
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
						panic(&UUIDUnavailableError{})
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

					self = NewEphemeralReferenceValue(invocationContext, auth, value, attachmentType)

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
			contractValue := config.ContractValueHandler(
				declarationInterpreter,
				compositeType,
				constructorGenerator,
			)
			contractValue.SetNestedVariables(nestedVariables)
			return contractValue
		})
	} else {
		constructor := constructorGenerator(common.ZeroAddress)
		constructor.NestedVariables = nestedVariables
		variable.SetValue(declarationInterpreter, constructor)
	}

	return lexicalScope, variable
}

type EnumCase struct {
	RawValue IntegerValue
	Value    MemberAccessibleValue
}

func (interpreter *Interpreter) declareEnumLookupFunction(
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
		).(IntegerValue)

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
			common.ZeroAddress,
		)
		caseValues[i] = EnumCase{
			Value:    caseValue,
			RawValue: rawValue,
		}

		constructorNestedVariables[enumCase.Identifier.Identifier] =
			NewVariableWithValue(interpreter, caseValue)
	}

	enumLookupFunctionType := interpreter.Program.Elaboration.EnumLookupFunctionType(compositeType)
	value := EnumLookupFunction(
		interpreter,
		enumLookupFunctionType,
		caseValues,
		constructorNestedVariables,
	)
	variable.SetValue(interpreter, value)

	return lexicalScope, variable
}

func EnumLookupFunction(
	gauge common.MemoryGauge,
	functionType *sema.FunctionType,
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
		functionType,
		func(invocation Invocation) Value {
			inter := invocation.InvocationContext

			rawValue, ok := invocation.Arguments[0].(IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			rawValueArgumentBigEndianBytes := rawValue.ToBigEndianBytes()

			caseValue, ok := lookupTable[string(rawValueArgumentBigEndianBytes)]
			if !ok {
				return Nil
			}

			caseValue = caseValue.Transfer(
				inter,
				atree.Address{},
				false,
				nil,
				nil,
				true, // value is standalone.
			)

			return NewSomeValueNonCopying(inter, caseValue)
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

func TransferAndConvert(
	context ValueConversionContext,
	value Value,
	valueType, targetType sema.Type,
) Value {

	transferredValue := value.Transfer(
		context,
		atree.Address{},
		false,
		nil,
		nil,
		true, // value is standalone.
	)

	return ConvertAndBoxWithValidation(
		context,
		transferredValue,
		valueType,
		targetType,
	)
}

func ConvertAndBoxWithValidation(
	context ValueConversionContext,
	transferredValue Value,
	valueType sema.Type,
	targetType sema.Type,
) Value {
	// Defensively check the actual value's type matches the expected value type.
	if valueType != nil {
		valueStaticType := transferredValue.StaticType(context)
		if !IsSubTypeOfSemaType(context, valueStaticType, valueType) {
			resultSemaType := context.SemaTypeFromStaticType(valueStaticType)

			panic(&ValueTransferTypeError{
				ExpectedType: valueType,
				ActualType:   resultSemaType,
			})
		}
	}

	result := convertAndBox(
		context,
		transferredValue,
		valueType,
		targetType,
	)

	// Defensively check the value's type matches the target type
	if targetType != nil {
		resultStaticType := result.StaticType(context)
		if !IsSubTypeOfSemaType(context, resultStaticType, targetType) {

			resultSemaType := context.SemaTypeFromStaticType(resultStaticType)

			panic(&ValueTransferTypeError{
				ExpectedType: targetType,
				ActualType:   resultSemaType,
			})
		}
	}

	return result
}

func TransferIfNotResourceAndConvert(
	context ValueConversionContext,
	value Value,
	valueType, targetType sema.Type,
) Value {

	if !valueType.IsResourceType() {
		value = value.Transfer(
			context,
			atree.Address{},
			false,
			nil,
			nil,
			true, // value is standalone.
		)
	}

	return ConvertAndBoxWithValidation(
		context,
		value,
		valueType,
		targetType,
	)
}

// convertAndBox converts a value to a target type, and boxes in optionals and any value, if necessary
func convertAndBox(
	context ValueConversionContext,
	value Value,
	valueType sema.Type,
	targetType sema.Type,
) Value {
	value = convert(context, value, valueType, targetType)
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
				)
				return NewSomeValueNonCopying(context, innerValue)
			}
			return value
		}
	}

	switch unwrappedTargetType {
	case sema.IntType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt(context, value)
		}

	case sema.UIntType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt(context, value)
		}

	// Int*
	case sema.Int8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt8(context, value)
		}

	case sema.Int16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt16(context, value)
		}

	case sema.Int32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt32(context, value)
		}

	case sema.Int64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt64(context, value)
		}

	case sema.Int128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt128(context, value)
		}

	case sema.Int256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertInt256(context, value)
		}

	// UInt*
	case sema.UInt8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt8(context, value)
		}

	case sema.UInt16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt16(context, value)
		}

	case sema.UInt32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt32(context, value)
		}

	case sema.UInt64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt64(context, value)
		}

	case sema.UInt128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt128(context, value)
		}

	case sema.UInt256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUInt256(context, value)
		}

	// Word*
	case sema.Word8Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord8(context, value)
		}

	case sema.Word16Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord16(context, value)
		}

	case sema.Word32Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord32(context, value)
		}

	case sema.Word64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord64(context, value)
		}

	case sema.Word128Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord128(context, value)
		}

	case sema.Word256Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertWord256(context, value)
		}

	// Fix*

	case sema.Fix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertFix64(context, value)
		}

	case sema.UFix64Type:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertUFix64(context, value)
		}
	}

	switch unwrappedTargetType := unwrappedTargetType.(type) {
	case *sema.AddressType:
		if !valueType.Equal(unwrappedTargetType) {
			return ConvertAddress(context, value)
		}

	case sema.ArrayType:
		if arrayValue, isArray := value.(*ArrayValue); isArray && !valueType.Equal(unwrappedTargetType) {

			oldArrayStaticType := arrayValue.StaticType(context)
			arrayStaticType := convertStaticType(context, oldArrayStaticType, unwrappedTargetType).(ArrayStaticType)

			if oldArrayStaticType.Equal(arrayStaticType) {
				return value
			}

			targetElementType := context.SemaTypeFromStaticType(arrayStaticType.ElementType())

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
					valueType := context.SemaTypeFromStaticType(value.StaticType(context))
					return convert(context, value, valueType, targetElementType)
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

			targetKeyType := context.SemaTypeFromStaticType(dictStaticType.KeyType)
			targetValueType := context.SemaTypeFromStaticType(dictStaticType.ValueType)

			dictionary := dictValue.dictionary

			iterator, err := dictionary.ReadOnlyIterator()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			return newDictionaryValueWithIterator(
				context,
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

					keyType := context.SemaTypeFromStaticType(key.StaticType(context))
					valueType := context.SemaTypeFromStaticType(value.StaticType(context))

					convertedKey := convert(context, key, keyType, targetKeyType)
					convertedValue := convert(context, value, valueType, targetValueType)

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

		// Defensively check that we never create a reference to an optional type
		if _, ok := unwrappedTargetType.Type.(*sema.OptionalType); ok {
			panic(errors.NewUnexpectedError(
				"unsupported reference to optional target type: %s",
				unwrappedTargetType,
			))
		}

		targetAuthorization := ConvertSemaAccessToStaticAuthorization(context, unwrappedTargetType.Authorization)

		switch ref := value.(type) {
		case *EphemeralReferenceValue:
			if shouldConvertReference(ref, valueType, unwrappedTargetType, targetAuthorization) {
				checkMappedEntitlements(unwrappedTargetType)
				return NewEphemeralReferenceValue(
					context,
					targetAuthorization,
					ref.Value,
					unwrappedTargetType.Type,
				)
			}

		case *StorageReferenceValue:
			if shouldConvertReference(ref, valueType, unwrappedTargetType, targetAuthorization) {
				checkMappedEntitlements(unwrappedTargetType)
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

func checkMappedEntitlements(unwrappedTargetType *sema.ReferenceType) {
	// check defensively that we never create a runtime mapped entitlement value
	if _, isMappedAuth := unwrappedTargetType.Authorization.(*sema.EntitlementMapAccess); isMappedAuth {
		panic(&UnexpectedMappedEntitlementError{
			Type: unwrappedTargetType,
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
	if defaultDestroyEvent := interpreter.Program.Elaboration.DefaultDestroyDeclaration(declaration); defaultDestroyEvent != nil {
		var nestedVariable Variable
		lexicalScope, nestedVariable = interpreter.declareCompositeValue(
			defaultDestroyEvent,
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
					interpreter.declareSelfVariable(*invocation.Self)
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

				return interpreter.visitFunctionBody(
					beforeStatements,
					preConditions,
					body,
					rewrittenPostConditions,
					functionType.ReturnTypeAnnotation.Type,
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
	return accountStorage.ValueExists(context, identifier)
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

func (interpreter *Interpreter) RemoveStored(
	storageAddress common.Address,
	domain common.StorageDomain,
	key StorageMapKey,
) atree.Storable {
	accountStorage := interpreter.Storage().GetDomainStorageMap(interpreter, storageAddress, domain, false)
	if accountStorage == nil {
		return nil
	}

	return accountStorage.RemoveValueWithoutDeletion(interpreter, key)
}

type TypedStringValueParser struct {
	ReceiverType sema.Type
	Parser       StringValueParser
}

// StringValueParser is a function that attempts to create a Cadence value from a string,
// e.g. parsing a number from a string
type StringValueParser func(common.Gauge, string) OptionalValue

func newFromStringFunction(typedParser TypedStringValueParser) FunctionValue {
	functionType := sema.FromStringFunctionType(typedParser.ReceiverType)
	parser := typedParser.Parser

	return NewUnmeteredStaticHostFunctionValueFromNativeFunction(
		functionType,
		NativeFromStringFunction(parser),
	)
}

// default implementation for parsing a given unsigned numeric type from a string.
// the size provided by sizeInBytes is passed to strconv.ParseUint, ensuring that the parsed value fits in the target type.
// input strings must not begin with a '+' or '-'.
func unsignedIntValueParser[ValueType Value, IntType any](
	bitSize int,
	toValue func(common.MemoryGauge, func() IntType) ValueType,
	fromUInt64 func(uint64) IntType,
) StringValueParser {
	return func(gauge common.Gauge, input string) OptionalValue {
		common.UseComputation(
			gauge,
			common.ComputationUsage{
				Kind:      common.ComputationKindUintParse,
				Intensity: uint64(len(input)),
			},
		)

		val, err := strconv.ParseUint(input, 10, bitSize)
		if err != nil {
			return NilOptionalValue
		}

		converted := toValue(gauge, func() IntType {
			return fromUInt64(val)
		})
		return NewSomeValueNonCopying(gauge, converted)
	}
}

// default implementation for parsing a given signed numeric type from a string.
// the size provided by sizeInBytes is passed to strconv.ParseUint, ensuring that the parsed value fits in the target type.
// input strings may begin with a '+' or '-'.
func signedIntValueParser[ValueType Value, IntType any](
	bitSize int,
	toValue func(common.MemoryGauge, func() IntType) ValueType,
	fromInt64 func(int64) IntType,
) StringValueParser {

	return func(gauge common.Gauge, input string) OptionalValue {
		common.UseComputation(
			gauge,
			common.ComputationUsage{
				Kind:      common.ComputationKindIntParse,
				Intensity: uint64(len(input)),
			},
		)

		val, err := strconv.ParseInt(input, 10, bitSize)
		if err != nil {
			return NilOptionalValue
		}

		converted := toValue(gauge, func() IntType {
			return fromInt64(val)
		})
		return NewSomeValueNonCopying(gauge, converted)
	}
}

// No need to use metered constructors for values represented by big.Ints,
// since estimation is more granular than fixed-size types.
func bigIntValueParser(convert func(*big.Int) (Value, bool)) StringValueParser {
	return func(gauge common.Gauge, input string) OptionalValue {

		literalKind := common.IntegerLiteralKindDecimal
		estimatedSize := common.OverEstimateBigIntFromString(input, literalKind)
		common.UseMemory(gauge, common.NewBigIntMemoryUsage(estimatedSize))

		common.UseComputation(
			gauge,
			common.ComputationUsage{
				Kind:      common.ComputationKindBigIntParse,
				Intensity: uint64(len(input)),
			},
		)

		val, ok := new(big.Int).SetString(input, literalKind.Base())
		if !ok {
			return NilOptionalValue
		}

		converted, ok := convert(val)

		if !ok {
			return NilOptionalValue
		}
		return NewSomeValueNonCopying(gauge, converted)
	}
}

// check if val is in the inclusive interval [low, high]
func inRange(val *big.Int, low *big.Int, high *big.Int) bool {
	return -1 < val.Cmp(low) && val.Cmp(high) < 1
}

var StringValueParsers = func() map[string]TypedStringValueParser {
	parsers := map[string]TypedStringValueParser{}

	for _, parser := range []TypedStringValueParser{
		// Int*
		{
			ReceiverType: sema.Int8Type,
			Parser:       signedIntValueParser(8, NewInt8Value, func(n int64) int8 { return int8(n) }),
		},
		{
			ReceiverType: sema.Int16Type,
			Parser:       signedIntValueParser(16, NewInt16Value, func(n int64) int16 { return int16(n) }),
		},
		{
			ReceiverType: sema.Int32Type,
			Parser:       signedIntValueParser(32, NewInt32Value, func(n int64) int32 { return int32(n) }),
		},
		{
			ReceiverType: sema.Int64Type,
			Parser:       signedIntValueParser(64, NewInt64Value, func(n int64) int64 { return n }),
		},
		{
			ReceiverType: sema.Int128Type,
			Parser: bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
				if ok = inRange(b, sema.Int128TypeMinIntBig, sema.Int128TypeMaxIntBig); ok {
					v = NewUnmeteredInt128ValueFromBigInt(b)
				}
				return
			}),
		},
		{
			ReceiverType: sema.Int256Type,
			Parser: bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
				if ok = inRange(b, sema.Int256TypeMinIntBig, sema.Int256TypeMaxIntBig); ok {
					v = NewUnmeteredInt256ValueFromBigInt(b)
				}
				return
			}),
		},
		{
			ReceiverType: sema.IntType,
			Parser: bigIntValueParser(func(b *big.Int) (Value, bool) {
				return NewUnmeteredIntValueFromBigInt(b), true
			}),
		},

		// UInt*
		{
			ReceiverType: sema.UInt8Type,
			Parser:       unsignedIntValueParser(8, NewUInt8Value, func(n uint64) uint8 { return uint8(n) }),
		},
		{
			ReceiverType: sema.UInt16Type,
			Parser:       unsignedIntValueParser(16, NewUInt16Value, func(n uint64) uint16 { return uint16(n) }),
		},
		{
			ReceiverType: sema.UInt32Type,
			Parser:       unsignedIntValueParser(32, NewUInt32Value, func(n uint64) uint32 { return uint32(n) }),
		},
		{
			ReceiverType: sema.UInt64Type,
			Parser:       unsignedIntValueParser(64, NewUInt64Value, func(n uint64) uint64 { return n }),
		},
		{
			ReceiverType: sema.UInt128Type,
			Parser: bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
				if ok = inRange(b, sema.UInt128TypeMinIntBig, sema.UInt128TypeMaxIntBig); ok {
					v = NewUnmeteredUInt128ValueFromBigInt(b)
				}
				return
			}),
		},
		{
			ReceiverType: sema.UInt256Type,
			Parser: bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
				if ok = inRange(b, sema.UInt256TypeMinIntBig, sema.UInt256TypeMaxIntBig); ok {
					v = NewUnmeteredUInt256ValueFromBigInt(b)
				}
				return
			}),
		},
		{
			ReceiverType: sema.UIntType,
			Parser: bigIntValueParser(func(b *big.Int) (Value, bool) {
				return NewUnmeteredUIntValueFromBigInt(b), true
			}),
		},

		// Word*
		{
			ReceiverType: sema.Word8Type,
			Parser:       unsignedIntValueParser(8, NewWord8Value, func(n uint64) uint8 { return uint8(n) }),
		},
		{
			ReceiverType: sema.Word16Type,
			Parser:       unsignedIntValueParser(16, NewWord16Value, func(n uint64) uint16 { return uint16(n) }),
		},
		{
			ReceiverType: sema.Word32Type,
			Parser:       unsignedIntValueParser(32, NewWord32Value, func(n uint64) uint32 { return uint32(n) }),
		},
		{
			ReceiverType: sema.Word64Type,
			Parser:       unsignedIntValueParser(64, NewWord64Value, func(n uint64) uint64 { return n }),
		},
		{
			ReceiverType: sema.Word128Type,
			Parser: bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
				if ok = inRange(b, sema.Word128TypeMinIntBig, sema.Word128TypeMaxIntBig); ok {
					v = NewUnmeteredWord128ValueFromBigInt(b)
				}
				return
			}),
		},
		{
			ReceiverType: sema.Word256Type,
			Parser: bigIntValueParser(func(b *big.Int) (v Value, ok bool) {
				if ok = inRange(b, sema.Word256TypeMinIntBig, sema.Word256TypeMaxIntBig); ok {
					v = NewUnmeteredWord256ValueFromBigInt(b)
				}
				return
			}),
		},

		// Fix*
		{
			ReceiverType: sema.Fix64Type,
			Parser: func(gauge common.Gauge, input string) OptionalValue {

				common.UseComputation(
					gauge,
					common.ComputationUsage{
						Kind:      common.ComputationKindFixParse,
						Intensity: uint64(len(input)),
					},
				)

				n, err := fixedpoint.ParseFix64(input)
				if err != nil {
					return NilOptionalValue
				}

				val := NewFix64Value(gauge, n.Int64)
				return NewSomeValueNonCopying(gauge, val)

			},
		},
		{
			ReceiverType: sema.Fix128Type,
			Parser: func(gauge common.Gauge, input string) OptionalValue {

				common.UseComputation(
					gauge,
					common.ComputationUsage{
						Kind:      common.ComputationKindFixParse,
						Intensity: uint64(len(input)),
					},
				)

				n, err := fixedpoint.ParseFix128(input)
				if err != nil {
					return NilOptionalValue
				}

				// No need to check ranges, as `ParseFix128` already does that.
				val := NewFix128ValueFromBigInt(gauge, n)
				return NewSomeValueNonCopying(gauge, val)

			},
		},

		// UFix*
		{
			ReceiverType: sema.UFix64Type,
			Parser: func(gauge common.Gauge, input string) OptionalValue {

				common.UseComputation(
					gauge,
					common.ComputationUsage{
						Kind:      common.ComputationKindUfixParse,
						Intensity: uint64(len(input)),
					},
				)

				n, err := fixedpoint.ParseUFix64(input)
				if err != nil {
					return NilOptionalValue
				}

				val := NewUFix64Value(gauge, n.Uint64)
				return NewSomeValueNonCopying(gauge, val)
			},
		},
		{
			ReceiverType: sema.UFix128Type,
			Parser: func(gauge common.Gauge, input string) OptionalValue {

				common.UseComputation(
					gauge,
					common.ComputationUsage{
						Kind:      common.ComputationKindUfixParse,
						Intensity: uint64(len(input)),
					},
				)

				n, err := fixedpoint.ParseUFix128(input)
				if err != nil {
					return NilOptionalValue
				}

				// No need to check ranges, as `ParseUFix128` already does that.
				val := NewUFix128ValueFromBigInt(gauge, n)
				return NewSomeValueNonCopying(gauge, val)

			},
		},
	} {
		// index by type name
		typeName := parser.ReceiverType.String()
		if _, ok := parsers[typeName]; ok {
			panic(errors.NewUnexpectedError("duplicate string value parser for type %s", typeName))
		}
		parsers[typeName] = parser
	}

	return parsers
}()

type TypedBigEndianBytesConverter struct {
	ReceiverType sema.Type
	ByteLength   uint
	Converter    BigEndianBytesConverter
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

// BigEndianBytesConverter is a function that attempts to create a Number from big-endian bytes.
type BigEndianBytesConverter func(common.MemoryGauge, []byte) Value

func newFromBigEndianBytesFunction(typedConverter TypedBigEndianBytesConverter) FunctionValue {
	functionType := sema.FromBigEndianBytesFunctionType(typedConverter.ReceiverType)
	byteLength := typedConverter.ByteLength
	converter := typedConverter.Converter

	// Converter functions are static functions.
	return NewUnmeteredStaticHostFunctionValueFromNativeFunction(
		functionType,
		NativeFromBigEndianBytesFunction(byteLength, converter),
	)
}

var BigEndianBytesConverters = func() map[string]TypedBigEndianBytesConverter {
	converters := map[string]TypedBigEndianBytesConverter{}

	for _, converter := range []TypedBigEndianBytesConverter{
		// Int*
		{
			ReceiverType: sema.Int8Type,
			ByteLength:   sema.Int8TypeSize,
			Converter:    NewInt8ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Int16Type,
			ByteLength:   sema.Int16TypeSize,
			Converter:    NewInt16ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Int32Type,
			ByteLength:   sema.Int32TypeSize,
			Converter:    NewInt32ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Int64Type,
			ByteLength:   sema.Int64TypeSize,
			Converter:    NewInt64ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Int128Type,
			ByteLength:   sema.Int128TypeSize,
			Converter:    NewInt128ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Int256Type,
			ByteLength:   sema.Int256TypeSize,
			Converter:    NewInt256ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.IntType,
			Converter:    NewIntValueFromBigEndianBytes,
		},

		// UInt*
		{
			ReceiverType: sema.UInt8Type,
			ByteLength:   sema.UInt8TypeSize,
			Converter:    NewUInt8ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.UInt16Type,
			ByteLength:   sema.UInt16TypeSize,
			Converter:    NewUInt16ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.UInt32Type,
			ByteLength:   sema.UInt32TypeSize,
			Converter:    NewUInt32ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.UInt64Type,
			ByteLength:   sema.UInt64TypeSize,
			Converter:    NewUInt64ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.UInt128Type,
			ByteLength:   sema.UInt128TypeSize,
			Converter:    NewUInt128ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.UInt256Type,
			ByteLength:   sema.UInt256TypeSize,
			Converter:    NewUInt256ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.UIntType,
			Converter:    NewUIntValueFromBigEndianBytes,
		},

		// Word*
		{
			ReceiverType: sema.Word8Type,
			ByteLength:   sema.Word8TypeSize,
			Converter:    NewWord8ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Word16Type,
			ByteLength:   sema.Word16TypeSize,
			Converter:    NewWord16ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Word32Type,
			ByteLength:   sema.Word32TypeSize,
			Converter:    NewWord32ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Word64Type,
			ByteLength:   sema.Word64TypeSize,
			Converter:    NewWord64ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Word128Type,
			ByteLength:   sema.Word128TypeSize,
			Converter:    NewWord128ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Word256Type,
			ByteLength:   sema.Word256TypeSize,
			Converter:    NewWord256ValueFromBigEndianBytes,
		},

		// Fix*
		{
			ReceiverType: sema.Fix64Type,
			ByteLength:   sema.Fix64TypeSize,
			Converter:    NewFix64ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.Fix128Type,
			ByteLength:   sema.Fix128TypeSize,
			Converter:    NewFix128ValueFromBigEndianBytes,
		},

		// UFix*
		{
			ReceiverType: sema.UFix64Type,
			ByteLength:   sema.UFix64TypeSize,
			Converter:    NewUFix64ValueFromBigEndianBytes,
		},
		{
			ReceiverType: sema.UFix128Type,
			ByteLength:   sema.UFix128TypeSize,
			Converter:    NewUFix128ValueFromBigEndianBytes,
		},
	} {
		// index by type name
		typeName := converter.ReceiverType.String()
		if _, ok := converters[typeName]; ok {
			panic(errors.NewUnexpectedError("duplicate from big-endian bytes converter for type %s", typeName))
		}
		converters[typeName] = converter
	}

	return converters
}()

type ValueConverterDeclaration struct {
	Min             Value
	Max             Value
	Convert         func(common.MemoryGauge, Value) Value
	nestedVariables []struct {
		Name  string
		Value Value
	}
	Name string
}

// It would be nice if return types in Go's function types would be covariant
var ConverterDeclarations = []ValueConverterDeclaration{
	{
		Name: sema.IntTypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertInt(gauge, value)
		},
	},
	{
		Name: sema.UIntTypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertUInt(gauge, value)
		},
		Min: NewUnmeteredUIntValueFromBigInt(sema.UIntTypeMin),
	},
	{
		Name: sema.Int8TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertInt8(gauge, value)
		},
		Min: NewUnmeteredInt8Value(math.MinInt8),
		Max: NewUnmeteredInt8Value(math.MaxInt8),
	},
	{
		Name: sema.Int16TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertInt16(gauge, value)
		},
		Min: NewUnmeteredInt16Value(math.MinInt16),
		Max: NewUnmeteredInt16Value(math.MaxInt16),
	},
	{
		Name: sema.Int32TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertInt32(gauge, value)
		},
		Min: NewUnmeteredInt32Value(math.MinInt32),
		Max: NewUnmeteredInt32Value(math.MaxInt32),
	},
	{
		Name: sema.Int64TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertInt64(gauge, value)
		},
		Min: NewUnmeteredInt64Value(math.MinInt64),
		Max: NewUnmeteredInt64Value(math.MaxInt64),
	},
	{
		Name: sema.Int128TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertInt128(gauge, value)
		},
		Min: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
		Max: NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
	},
	{
		Name: sema.Int256TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertInt256(gauge, value)
		},
		Min: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
		Max: NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
	},
	{
		Name: sema.UInt8TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertUInt8(gauge, value)
		},
		Min: NewUnmeteredUInt8Value(0),
		Max: NewUnmeteredUInt8Value(math.MaxUint8),
	},
	{
		Name: sema.UInt16TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertUInt16(gauge, value)
		},
		Min: NewUnmeteredUInt16Value(0),
		Max: NewUnmeteredUInt16Value(math.MaxUint16),
	},
	{
		Name: sema.UInt32TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertUInt32(gauge, value)
		},
		Min: NewUnmeteredUInt32Value(0),
		Max: NewUnmeteredUInt32Value(math.MaxUint32),
	},
	{
		Name: sema.UInt64TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertUInt64(gauge, value)
		},
		Min: NewUnmeteredUInt64Value(0),
		Max: NewUnmeteredUInt64Value(math.MaxUint64),
	},
	{
		Name:    sema.UInt128TypeName,
		Convert: ConvertUInt128,
		Min:     NewUnmeteredUInt128ValueFromUint64(0),
		Max:     NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
	},
	{
		Name: sema.UInt256TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertUInt256(gauge, value)
		},
		Min: NewUnmeteredUInt256ValueFromUint64(0),
		Max: NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
	},
	{
		Name: sema.Word8TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertWord8(gauge, value)
		},
		Min: NewUnmeteredWord8Value(0),
		Max: NewUnmeteredWord8Value(math.MaxUint8),
	},
	{
		Name: sema.Word16TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertWord16(gauge, value)
		},
		Min: NewUnmeteredWord16Value(0),
		Max: NewUnmeteredWord16Value(math.MaxUint16),
	},
	{
		Name: sema.Word32TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertWord32(gauge, value)
		},
		Min: NewUnmeteredWord32Value(0),
		Max: NewUnmeteredWord32Value(math.MaxUint32),
	},
	{
		Name: sema.Word64TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertWord64(gauge, value)
		},
		Min: NewUnmeteredWord64Value(0),
		Max: NewUnmeteredWord64Value(math.MaxUint64),
	},
	{
		Name:    sema.Word128TypeName,
		Convert: ConvertWord128,
		Min:     NewUnmeteredWord128ValueFromUint64(0),
		Max:     NewUnmeteredWord128ValueFromBigInt(sema.Word128TypeMaxIntBig),
	},
	{
		Name:    sema.Word256TypeName,
		Convert: ConvertWord256,
		Min:     NewUnmeteredWord256ValueFromUint64(0),
		Max:     NewUnmeteredWord256ValueFromBigInt(sema.Word256TypeMaxIntBig),
	},
	{
		Name: sema.Fix64TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertFix64(gauge, value)
		},
		Min: NewUnmeteredFix64Value(math.MinInt64),
		Max: NewUnmeteredFix64Value(math.MaxInt64),
	},
	{
		Name: sema.Fix128TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertFix128(gauge, value)
		},
		Min: NewUnmeteredFix128Value(fixedpoint.Fix128TypeMin),
		Max: NewUnmeteredFix128Value(fixedpoint.Fix128TypeMax),
	},
	{
		Name: sema.UFix64TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertUFix64(gauge, value)
		},
		Min: NewUnmeteredUFix64Value(0),
		Max: NewUnmeteredUFix64Value(math.MaxUint64),
	},
	{
		Name: sema.UFix128TypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertUFix128(gauge, value)
		},
		Min: NewUnmeteredUFix128Value(fixedpoint.UFix128TypeMin),
		Max: NewUnmeteredUFix128Value(fixedpoint.UFix128TypeMax),
	},
	{
		Name: sema.AddressTypeName,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return ConvertAddress(gauge, value)
		},
		nestedVariables: []struct {
			Name  string
			Value Value
		}{
			// Converter functions are static functions.
			{
				Name: sema.AddressTypeFromBytesFunctionName,
				Value: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
					sema.AddressTypeFromBytesFunctionType,
					NativeAddressFromBytesFunction,
				),
			},
			{
				Name: sema.AddressTypeFromStringFunctionName,
				Value: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
					sema.AddressTypeFromStringFunctionType,
					NativeAddressFromStringFunction,
				),
			},
		},
	},
	{
		Name: sema.PublicPathType.Name,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return newPathFromStringValue(gauge, common.PathDomainPublic, value)
		},
	},
	{
		Name: sema.PrivatePathType.Name,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
			return newPathFromStringValue(gauge, common.PathDomainPrivate, value)
		},
	},
	{
		Name: sema.StoragePathType.Name,
		Convert: func(gauge common.MemoryGauge, value Value) Value {
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

		if _, ok := StringValueParsers[typeName]; !ok {
			panic(fmt.Sprintf("missing fromString implementation for number type: %s", numberType))
		}

		if _, ok := BigEndianBytesConverters[typeName]; !ok {
			panic(fmt.Sprintf("missing fromBigEndianBytes implementation for number type: %s", numberType))
		}
	}
}

func ConstructOptionalTypeValue(context InvocationContext, typeValue TypeValue) Value {
	return NewTypeValue(
		context,
		NewOptionalStaticType(
			context,
			typeValue.Type,
		),
	)
}

func ConstructVariableSizedArrayTypeValue(context InvocationContext, typeValue TypeValue) TypeValue {
	return NewTypeValue(
		context,
		NewVariableSizedStaticType(
			context,
			typeValue.Type,
		),
	)
}

func ConstructConstantSizedArrayTypeValue(
	context InvocationContext,
	typeValue TypeValue,
	sizeValue IntValue,
) TypeValue {
	return NewTypeValue(
		context,
		NewConstantSizedStaticType(
			context,
			typeValue.Type,
			int64(sizeValue.ToInt()),
		),
	)
}

func ConstructDictionaryTypeValue(
	context InvocationContext,
	keyTypeValue TypeValue,
	valueTypeValue TypeValue,
) Value {
	keyType := keyTypeValue.Type
	valueType := valueTypeValue.Type

	// if the given key is not a valid dictionary key, it wouldn't make sense to create this type
	if keyType == nil ||
		!sema.IsSubType(
			context.SemaTypeFromStaticType(keyType),
			sema.HashableStructType,
		) {
		return Nil
	}

	return NewSomeValueNonCopying(
		context,
		NewTypeValue(
			context,
			NewDictionaryStaticType(
				context,
				keyType,
				valueType,
			),
		),
	)
}

func ConstructCompositeTypeValue(
	context InvocationContext,
	typeIDValue *StringValue,
) Value {
	typeID := typeIDValue.Str

	composite, err := lookupComposite(context, typeID)
	if err != nil {
		return Nil
	}

	return NewSomeValueNonCopying(
		context,
		NewTypeValue(
			context,
			ConvertSemaToStaticType(context, composite),
		),
	)
}

func ConstructFunctionTypeValue(
	invocationContext InvocationContext,
	parameterTypeValues *ArrayValue,
	returnTypeValue TypeValue,
) Value {
	returnType := invocationContext.SemaTypeFromStaticType(returnTypeValue.Type)

	var parameterTypes []sema.Parameter
	parameterCount := parameterTypeValues.Count()
	if parameterCount > 0 {
		parameterTypes = make([]sema.Parameter, 0, parameterCount)
		parameterTypeValues.Iterate(
			invocationContext,
			func(param Value) bool {
				semaType := invocationContext.SemaTypeFromStaticType(param.(TypeValue).Type)
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
		)
	}
	functionStaticType := NewFunctionStaticType(
		invocationContext,
		sema.NewSimpleFunctionType(
			sema.FunctionPurityImpure,
			parameterTypes,
			sema.NewTypeAnnotation(returnType),
		),
	)
	return NewUnmeteredTypeValue(functionStaticType)
}

func ConstructReferenceTypeValue(
	invocationContext InvocationContext,
	entitlementValues *ArrayValue,
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

func ConstructIntersectionTypeValue(
	context InvocationContext,
	intersectionIDs *ArrayValue,
) Value {
	var staticIntersections []*InterfaceStaticType
	var semaIntersections []*sema.InterfaceType

	count := intersectionIDs.Count()
	if count > 0 {
		staticIntersections = make([]*InterfaceStaticType, 0, count)
		semaIntersections = make([]*sema.InterfaceType, 0, count)

		var invalidIntersectionID bool
		intersectionIDs.Iterate(
			context,
			func(typeID Value) bool {
				typeIDValue, ok := typeID.(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				intersectedInterface, err := lookupInterface(context, typeIDValue.Str)
				if err != nil {
					invalidIntersectionID = true
					return true
				}

				staticIntersections = append(
					staticIntersections,
					ConvertSemaToStaticType(context, intersectedInterface).(*InterfaceStaticType),
				)
				semaIntersections = append(semaIntersections, intersectedInterface)

				// Continue iteration
				return true
			},
			false,
		)

		// If there are any invalid interfaces,
		// then return nil
		if invalidIntersectionID {
			return Nil
		}
	}

	var invalidIntersectionType bool
	sema.CheckIntersectionType(
		context,
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
		context,
		NewTypeValue(
			context,
			NewIntersectionStaticType(
				context,
				staticIntersections,
			),
		),
	)
}

func ConstructCapabilityTypeValue(
	context InvocationContext,
	typeValue TypeValue,
) Value {

	ty := typeValue.Type

	// Capabilities must hold references
	_, ok := ty.(*ReferenceStaticType)
	if !ok {
		return Nil
	}

	return NewSomeValueNonCopying(
		context,
		NewTypeValue(
			context,
			NewCapabilityStaticType(
				context,
				ty,
			),
		),
	)
}

func ConstructInclusiveRangeTypeValue(
	context InvocationContext,
	typeValue TypeValue,
) Value {

	ty := typeValue.Type

	// InclusiveRanges must hold integers
	elemSemaTy := context.SemaTypeFromStaticType(ty)
	if !sema.IsSameTypeKind(elemSemaTy, sema.IntegerType) {
		return Nil
	}

	return NewSomeValueNonCopying(
		context,
		NewTypeValue(
			context,
			NewInclusiveRangeStaticType(
				context,
				ty,
			),
		),
	)
}

func defineBaseFunctions(activation *VariableActivation) {
	defineConverterFunctions(activation)
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

		converterFunctionType := sema.BaseValueActivation.Find(declaration.Name).Type.(*sema.FunctionType)

		converterFunctionValue := NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			converterFunctionType,
			NativeConverterFunction(convert),
		)

		addMember := func(name string, value Value) {
			if converterFunctionValue.NestedVariables == nil {
				converterFunctionValue.NestedVariables = map[string]Variable{}
			}
			// these variables are not needed to be metered as they are only ever declared once,
			// and can be considered base interpreter overhead
			converterFunctionValue.NestedVariables[name] = NewVariableWithValue(nil, value)
		}

		if declaration.Min != nil {
			addMember(sema.NumberTypeMinFieldName, declaration.Min)
		}

		if declaration.Max != nil {
			addMember(sema.NumberTypeMaxFieldName, declaration.Max)
		}

		if stringValueParser, ok := StringValueParsers[declaration.Name]; ok {
			addMember(sema.FromStringFunctionName, newFromStringFunction(stringValueParser))
		}

		if bigEndianBytesConverter, ok := BigEndianBytesConverters[declaration.Name]; ok {
			addMember(sema.FromBigEndianBytesFunctionName, newFromBigEndianBytesFunction(bigEndianBytesConverter))
		}

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
	name        string
	constructor *HostFunctionValue
}

var NativeMetaTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		_ []Value,
	) Value {
		staticType := typeArguments.NextStatic()

		return NewTypeValue(context, staticType)
	},
)

var NativeOptionalTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		typeValue := AssertValueOfType[TypeValue](args[0])

		return ConstructOptionalTypeValue(context, typeValue)
	},
)

var NativeVariableSizedArrayTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		typeValue := AssertValueOfType[TypeValue](args[0])

		return ConstructVariableSizedArrayTypeValue(context, typeValue)
	},
)

var NativeConstantSizedArrayTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		typeValue := AssertValueOfType[TypeValue](args[0])
		sizeValue := AssertValueOfType[IntValue](args[1])

		return ConstructConstantSizedArrayTypeValue(
			context,
			typeValue,
			sizeValue,
		)
	},
)

var NativeDictionaryTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		keyTypeValue := AssertValueOfType[TypeValue](args[0])
		valueTypeValue := AssertValueOfType[TypeValue](args[1])

		return ConstructDictionaryTypeValue(
			context,
			keyTypeValue,
			valueTypeValue,
		)
	},
)

var NativeCompositeTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		typeIDValue := AssertValueOfType[*StringValue](args[0])

		return ConstructCompositeTypeValue(context, typeIDValue)
	},
)

var NativeFunctionTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		parameterTypeValues := AssertValueOfType[*ArrayValue](args[0])
		returnTypeValue := AssertValueOfType[TypeValue](args[1])

		return ConstructFunctionTypeValue(
			context,
			parameterTypeValues,
			returnTypeValue,
		)
	},
)

var NativeReferenceTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		entitlementValues := AssertValueOfType[*ArrayValue](args[0])
		typeValue := AssertValueOfType[TypeValue](args[1])

		return ConstructReferenceTypeValue(
			context,
			entitlementValues,
			typeValue,
		)
	},
)

var NativeIntersectionTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		intersectionIDs := AssertValueOfType[*ArrayValue](args[0])

		return ConstructIntersectionTypeValue(
			context,
			intersectionIDs,
		)
	},
)

var NativeCapabilityTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		typeValue := AssertValueOfType[TypeValue](args[0])

		return ConstructCapabilityTypeValue(context, typeValue)
	},
)

var NativeInclusiveRangeTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		typeValue := AssertValueOfType[TypeValue](args[0])

		return ConstructInclusiveRangeTypeValue(context, typeValue)
	},
)

var NativeAddressFromBytesFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		byteArray := AssertValueOfType[*ArrayValue](args[0])

		return AddressValueFromByteArray(context, byteArray)
	},
)

var NativeAddressFromStringFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		string := AssertValueOfType[*StringValue](args[0])

		return AddressValueFromString(context, string)
	},
)

func NativeConverterFunction(convert func(memoryGauge common.MemoryGauge, value Value) Value) NativeFunction {
	return func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		return convert(context, args[0])
	}
}

func NativeFromStringFunction(parser StringValueParser) NativeFunction {
	return func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		argument := AssertValueOfType[*StringValue](args[0])
		return parser(context, argument.Str)
	}
}

func NativeFromBigEndianBytesFunction(byteLength uint, converter func(memoryGauge common.MemoryGauge, bytes []byte) Value) NativeFunction {
	return func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		args []Value,
	) Value {
		argument := AssertValueOfType[*ArrayValue](args[0])

		bytes, err := ByteArrayValueToByteSlice(context, argument)
		if err != nil {
			return Nil
		}

		// overflow
		if byteLength != 0 && uint(len(bytes)) > byteLength {
			return Nil
		}

		return NewSomeValueNonCopying(context, converter(context, bytes))
	}
}

var NativeStringFunction = NativeFunction(
	func(
		_ NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		_ Value,
		_ []Value,
	) Value {
		return EmptyString
	},
)

// Constructor functions are stateless functions. Hence they can be re-used across interpreters.
// They are also static functions.
var runtimeTypeConstructors = []runtimeTypeConstructor{
	{
		name: sema.MetaTypeName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.MetaTypeFunctionType,
			NativeMetaTypeFunction,
		),
	},
	{
		name: sema.OptionalTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.OptionalTypeFunctionType,
			NativeOptionalTypeFunction,
		),
	},
	{
		name: sema.VariableSizedArrayTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.VariableSizedArrayTypeFunctionType,
			NativeVariableSizedArrayTypeFunction,
		),
	},
	{
		name: sema.ConstantSizedArrayTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.ConstantSizedArrayTypeFunctionType,
			NativeConstantSizedArrayTypeFunction,
		),
	},
	{
		name: sema.DictionaryTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.DictionaryTypeFunctionType,
			NativeDictionaryTypeFunction,
		),
	},
	{
		name: sema.CompositeTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.CompositeTypeFunctionType,
			NativeCompositeTypeFunction,
		),
	},
	{
		name: sema.FunctionTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.FunctionTypeFunctionType,
			NativeFunctionTypeFunction,
		),
	},

	{
		name: sema.ReferenceTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.ReferenceTypeFunctionType,
			NativeReferenceTypeFunction,
		),
	},
	{
		name: sema.IntersectionTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.IntersectionTypeFunctionType,
			NativeIntersectionTypeFunction,
		),
	},
	{
		name: sema.CapabilityTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.CapabilityTypeFunctionType,
			NativeCapabilityTypeFunction,
		),
	},
	{
		name: sema.InclusiveRangeTypeFunctionName,
		constructor: NewUnmeteredStaticHostFunctionValueFromNativeFunction(
			sema.InclusiveRangeTypeFunctionType,
			NativeInclusiveRangeTypeFunction,
		),
	},
}

func defineRuntimeTypeConstructorFunctions(activation *VariableActivation) {
	for _, constructorFunc := range runtimeTypeConstructors {
		defineBaseValue(activation, constructorFunc.name, constructorFunc.constructor)
	}
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

	semaType := typeConverter.SemaTypeFromStaticType(superType)

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

	semaSubType := typeConverter.SemaTypeFromStaticType(staticSubType)

	return sema.IsSubType(semaSubType, superType)
}

func domainPaths(context StorageContext, address common.Address, domain common.PathDomain) []Value {
	storageMap := context.Storage().GetDomainStorageMap(context, address, domain.StorageDomain(), false)
	if storageMap == nil {
		return []Value{}
	}
	iterator := storageMap.Iterator()
	var paths []Value

	count := storageMap.Count()
	if count > 0 {
		paths = make([]Value, 0, count)
		for key := iterator.NextKey(context); key != nil; key = iterator.NextKey(context) {
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
	domain common.PathDomain,
	pathType StaticType,
) *ArrayValue {
	address := addressValue.ToAddress()
	values := domainPaths(context, address, domain)
	return NewArrayValue(
		context,
		NewVariableSizedStaticType(context, pathType),
		common.ZeroAddress,
		values...,
	)
}

func publicAccountPaths(
	context ArrayCreationContext,
	addressValue AddressValue,
) *ArrayValue {
	return accountPaths(
		context,
		addressValue,
		common.PathDomainPublic,
		PrimitiveStaticTypePublicPath,
	)
}

func storageAccountPaths(
	context ArrayCreationContext,
	addressValue AddressValue,
) *ArrayValue {
	return accountPaths(
		context,
		addressValue,
		common.PathDomainStorage,
		PrimitiveStaticTypeStoragePath,
	)
}

func (interpreter *Interpreter) RecordStorageMutation() {
	if interpreter.SharedState.inStorageIteration {
		interpreter.SharedState.storageMutatedDuringIteration = true
	}
}

func NativeAccountStorageIterateFunction(
	addressPointer *AddressValue,
	domain common.PathDomain,
	pathType sema.Type,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		address := GetAddressValue(receiver, addressPointer).ToAddress()

		return AccountStorageIterate(
			context,
			args,
			address,
			domain,
			pathType,
		)
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

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		functionType,
		NativeAccountStorageIterateFunction(&addressValue, domain, pathType),
	)
}

func AccountStorageIterate(
	invocationContext InvocationContext,
	arguments []Value,
	address common.Address,
	domain common.PathDomain,
	pathType sema.Type,
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
	storageIterator := storageMap.Iterator()

	wasInIteration := invocationContext.InStorageIteration()
	invocationContext.SetInStorageIteration(true)
	defer invocationContext.SetInStorageIteration(wasInIteration)

	for key, value := storageIterator.Next(invocationContext); key != nil && value != nil; key, value = storageIterator.Next(invocationContext) {

		staticType := value.StaticType(invocationContext)

		// Perform a forced value de-referencing to see if the associated type is not broken.
		// If broken, skip this value from the iteration.
		valueError := checkValue(
			invocationContext,
			value,
			staticType,
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

		result := invokeIteratorFunction(
			invocationContext,
			fn,
			arguments,
			invocationArgumentTypes,
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
			panic(&StorageMutatedDuringIterationError{})
		}

	}

	return Void
}

func (interpreter *Interpreter) InvokeFunction(
	_ FunctionValue,
	_ []Value,
	_ sema.Type,
) Value {
	// Interpreter's function values shouldn't/doesn't use `InvocationContext.InvokeFunction`.
	// They directly use the methods of `Interpreter`.
	// This indirection is only needed in VM.
	panic(errors.NewUnreachableError())
}

func invokeIteratorFunction(
	context InvocationContext,
	fn FunctionValue,
	arguments []Value,
	invocationArgumentTypes []sema.Type,
) Value {
	fnType := fn.FunctionType(context)
	parameterTypes := fnType.ParameterTypes()
	returnType := fnType.ReturnTypeAnnotation.Type

	result := invokeFunctionValue(
		context,
		fn,
		arguments,
		invocationArgumentTypes,
		parameterTypes,
		returnType,
		nil,
	)
	return result
}

func checkValue(
	context StoredValueCheckContext,
	value Value,
	staticType StaticType,
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

func NativeAccountStorageSaveFunction(
	addressPointer *AddressValue,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		addressValue := GetAddressValue(receiver, addressPointer)

		return AccountStorageSave(
			context,
			args,
			addressValue,
		)
	}
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
		NativeAccountStorageSaveFunction(&addressValue),
	)
}

func AccountStorageSave(
	context InvocationContext,
	arguments []Value,
	addressValue AddressValue,
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
		panic(&OverwriteError{
			Address: addressValue,
			Path:    path,
		})
	}

	value = value.Transfer(
		context,
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

func NativeAccountStorageTypeFunction(
	addressPointer *AddressValue,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		address := GetAddressValue(receiver, addressPointer).ToAddress()

		return AccountStorageType(
			context,
			args,
			address,
		)
	}
}

func authAccountStorageTypeFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		sema.Account_StorageTypeTypeFunctionType,
		NativeAccountStorageTypeFunction(&addressValue),
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
	return authAccountLoadFunction(
		context,
		storageValue,
		addressValue,
		sema.Account_StorageTypeLoadFunctionType,
	)
}

func authAccountStorageCopyFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {
	return authAccountCopyFunction(
		context,
		storageValue,
		addressValue,
		sema.Account_StorageTypeCopyFunctionType,
	)
}

func NativeAccountStorageCopyFunction(
	addressPointer *AddressValue,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		address := GetAddressValue(receiver, addressPointer).ToAddress()
		semaBorrowType := typeArguments.NextSema()

		return AccountStorageCopy(
			context,
			args,
			semaBorrowType,
			address,
		)
	}
}

func NativeAccountStorageLoadFunction(
	addressPointer *AddressValue,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		address := GetAddressValue(receiver, addressPointer).ToAddress()
		semaBorrowType := typeArguments.NextSema()

		return AccountStorageLoad(
			context,
			args,
			semaBorrowType,
			address,
		)
	}
}

func authAccountCopyFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
	functionType *sema.FunctionType,
) BoundFunctionValue {

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		functionType,
		NativeAccountStorageCopyFunction(&addressValue),
	)
}

func authAccountLoadFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
	functionType *sema.FunctionType,
) BoundFunctionValue {

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		functionType,
		NativeAccountStorageLoadFunction(&addressValue),
	)
}

func AccountStorageCopy(
	invocationContext InvocationContext,
	arguments []Value,
	typeParameter sema.Type,
	address common.Address,
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
		valueSemaType := invocationContext.SemaTypeFromStaticType(valueStaticType)

		panic(&StoredValueTypeMismatchError{
			ExpectedType: typeParameter,
			ActualType:   valueSemaType,
		})
	}

	transferredValue := value.Transfer(
		invocationContext,
		atree.Address{},
		false,
		nil,
		nil,
		false, // value is an element in storage map because it is from "ReadStored".
	)

	return NewSomeValueNonCopying(invocationContext, transferredValue)
}

func AccountStorageLoad(
	invocationContext InvocationContext,
	arguments []Value,
	typeParameter sema.Type,
	address common.Address,
) Value {
	path, ok := arguments[0].(PathValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	domain := path.Domain.StorageDomain()
	identifier := path.Identifier

	storageMapKey := StringStorageMapKey(identifier)

	storable := invocationContext.RemoveStored(address, domain, storageMapKey)

	if storable == nil {
		return Nil
	}

	transferedValue := StoredValue(invocationContext, storable, invocationContext.Storage()).
		Transfer(
			invocationContext,
			atree.Address{},
			true,
			storable,
			nil,
			true, // value is standalone because it was removed from parent container.
		)

	// If there is value stored for the given path,
	// check that it satisfies the type given as the type argument.

	valueStaticType := transferedValue.StaticType(invocationContext)

	if !IsSubTypeOfSemaType(invocationContext, valueStaticType, typeParameter) {
		valueSemaType := invocationContext.SemaTypeFromStaticType(valueStaticType)

		panic(&StoredValueTypeMismatchError{
			ExpectedType: typeParameter,
			ActualType:   valueSemaType,
		})
	}

	return NewSomeValueNonCopying(invocationContext, transferedValue)
}

func NativeAccountStorageBorrowFunction(
	addressPointer *AddressValue,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		address := GetAddressValue(receiver, addressPointer).ToAddress()
		typeParameter := typeArguments.NextSema()

		return AccountStorageBorrow(
			context,
			args,
			typeParameter,
			address,
		)
	}
}

func authAccountStorageBorrowFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		sema.Account_StorageTypeBorrowFunctionType,
		NativeAccountStorageBorrowFunction(&addressValue),
	)
}

func AccountStorageBorrow(
	invocationContext InvocationContext,
	arguments []Value,
	typeParameter sema.Type,
	address common.Address,
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

	value, err := reference.dereference(invocationContext)
	if err != nil {
		panic(err)
	}
	if value == nil {
		return Nil
	}

	return NewSomeValueNonCopying(invocationContext, reference)
}

func NativeAccountStorageCheckFunction(
	addressPointer *AddressValue,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		address := GetAddressValue(receiver, addressPointer).ToAddress()
		typeArgument := typeArguments.NextSema()

		return AccountStorageCheck(
			context,
			address,
			args,
			typeArgument,
		)
	}
}

func authAccountStorageCheckFunction(
	context FunctionCreationContext,
	storageValue *SimpleCompositeValue,
	addressValue AddressValue,
) BoundFunctionValue {

	return NewBoundHostFunctionValue(
		context,
		storageValue,
		sema.Account_StorageTypeCheckFunctionType,
		NativeAccountStorageCheckFunction(&addressValue),
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

func (interpreter *Interpreter) GetContractValue(contractLocation common.AddressLocation) *CompositeValue {
	inter := interpreter.EnsureLoaded(contractLocation)
	return inter.GetContractComposite(contractLocation)
}

// GetContractComposite gets the composite value of the contract at the address location.
func (interpreter *Interpreter) GetContractComposite(contractLocation common.AddressLocation) *CompositeValue {
	contractGlobal := interpreter.Globals.Get(contractLocation.Name)
	if contractGlobal == nil {
		return nil
	}

	contractValue, ok := contractGlobal.GetValue(interpreter).(*CompositeValue)
	if !ok {
		return nil
	}

	return contractValue
}

func GetNativeCompositeValueComputedFields(qualifiedIdentifier string) map[string]ComputedField {
	switch qualifiedIdentifier {
	case sema.PublicKeyType.Identifier:
		return map[string]ComputedField{
			sema.PublicKeyTypePublicKeyFieldName: func(
				context ValueTransferContext,
				v *CompositeValue,
			) Value {
				publicKeyValue := v.GetField(context, sema.PublicKeyTypePublicKeyFieldName)
				return publicKeyValue.Transfer(
					context,
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
	injectedCompositeFieldsHandler := context.GetInjectedCompositeFieldsHandler()
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

func (interpreter *Interpreter) GetCompositeValueFunctions(v *CompositeValue) *FunctionOrderedMap {

	var functions *FunctionOrderedMap

	typeID := v.TypeID()

	sharedState := interpreter.SharedState

	compositeValueFunctionsHandler := sharedState.Config.CompositeValueFunctionsHandler
	if compositeValueFunctionsHandler != nil {
		functions = compositeValueFunctionsHandler(interpreter, v)
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
		return nil, &InterfaceMissingLocationError{
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

	common.UseComputation(interpreter, common.LoopComputationUsage)

	onLoopIteration := interpreter.SharedState.Config.OnLoopIteration
	if onLoopIteration != nil {
		line := pos.StartPosition().Line
		onLoopIteration(interpreter, line)
	}
}

func (interpreter *Interpreter) reportFunctionInvocation() {

	common.UseComputation(interpreter, common.FunctionInvocationComputationUsage)

	onFunctionInvocation := interpreter.SharedState.Config.OnFunctionInvocation
	if onFunctionInvocation != nil {
		onFunctionInvocation(interpreter)
	}
}

func (interpreter *Interpreter) reportInvokedFunctionReturn() {
	onInvokedFunctionReturn := interpreter.SharedState.Config.OnInvokedFunctionReturn
	if onInvokedFunctionReturn == nil {
		return
	}

	onInvokedFunctionReturn(interpreter)
}

func GetAccessOfMember(context ValueStaticTypeContext, self Value, identifier string) sema.Access {
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
func getMember(
	context MemberAccessibleContext,
	self Value,
	identifier string,
	memberKind common.DeclarationKind,
) Value {
	var result Value
	// When the accessed value has a type that supports the declaration of members
	// or is a built-in type that has members (`MemberAccessibleValue`),
	// then try to get the member for the given identifier.
	// For example, the built-in type `String` has a member "length",
	// and composite declarations may contain member declarations
	if memberAccessibleValue, ok := self.(MemberAccessibleValue); ok {
		result = memberAccessibleValue.GetMember(context, identifier, memberKind)
	}
	if result == nil {
		result = getBuiltinFunctionMember(context, self, identifier)
	}

	// NOTE: do not panic if the member is nil. This is a valid state.
	// For example, when a composite field is initialized with a force-assignment, the field's value is read.

	return result
}

func GetMember(
	context MemberAccessibleContext,
	value MemberAccessibleValue,
	memberName string,
	memberKind common.DeclarationKind,
	nonFunctionMemberGetter func() Value,
) Value {
	switch memberKind {
	case common.DeclarationKindFunction:
		return context.GetMethod(value, memberName)
	default:
		if nonFunctionMemberGetter == nil {
			return nil
		}
		return nonFunctionMemberGetter()
	}
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

var NativeIsInstanceFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		// Retrieve the type from the end of arguments without using a fixed offset.
		// This is because this function can be invoked on attachments
		// which will have `base` as the first argument instead of the type.
		typeValue := AssertValueOfType[TypeValue](args[len(args)-1])
		return IsInstance(context, receiver, typeValue)
	},
)

func isInstanceFunction(context FunctionCreationContext, self Value) FunctionValue {
	return NewBoundHostFunctionValue(
		context,
		self,
		sema.IsInstanceFunctionType,
		NativeIsInstanceFunction,
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

var NativeGetTypeFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		return ValueGetType(context, receiver)
	},
)

func getTypeFunction(context FunctionCreationContext, self Value) FunctionValue {
	return NewBoundHostFunctionValue(
		context,
		self,
		sema.GetTypeFunctionType,
		NativeGetTypeFunction,
	)
}

func ValueGetType(context InvocationContext, self Value) Value {
	staticType := self.StaticType(context)
	return NewTypeValue(context, staticType)
}

func setMember(
	context ValueTransferContext,
	self Value,
	identifier string,
	value Value,
) bool {
	return self.(MemberAccessibleValue).SetMember(context, identifier, value)
}

func ExpectType(
	context ValueStaticTypeContext,
	value Value,
	expectedType sema.Type,
) {
	valueStaticType := value.StaticType(context)

	if !IsSubTypeOfSemaType(context, valueStaticType, expectedType) {
		valueSemaType := context.SemaTypeFromStaticType(valueStaticType)

		panic(&TypeMismatchError{
			ExpectedType: expectedType,
			ActualType:   valueSemaType,
		})
	}
}

func checkContainerMutation(
	context ValueStaticTypeContext,
	elementType StaticType,
	element Value,
) {
	actualElementType := element.StaticType(context)

	if !IsSubType(context, actualElementType, elementType) {
		panic(&ContainerMutationError{
			ExpectedType: context.SemaTypeFromStaticType(elementType),
			ActualType:   MustSemaTypeOfValue(element, context),
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

	defaultHIP := newHashInputProvider(interpreter)

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
			return value.Equal(interpreter, otherValue)

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
			var nonStorableValueErr *NonStorableValueError
			var nonStorableStaticTypeErr *NonStorableStaticTypeError

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
			var nonStorableValueErr *NonStorableValueError
			var nonStorableStaticTypeErr *NonStorableStaticTypeError

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
				InvalidateReferencedResources(context, fieldValue)
				// continue iteration
				return true
			},
		)
		valueID = value.ValueID()

	case *DictionaryValue:
		value.IterateReadOnlyLoaded(
			context,
			func(_, value Value) (resume bool) {
				InvalidateReferencedResources(context, value)
				return true
			},
		)
		valueID = value.ValueID()

	case *ArrayValue:
		value.IterateReadOnlyLoaded(
			context,
			func(element Value) (resume bool) {
				InvalidateReferencedResources(context, element)
				return true
			},
		)
		valueID = value.ValueID()

	case *SomeValue:
		InvalidateReferencedResources(context, value.value)
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
		panic(&InvalidatedResourceError{})
	}

	interpreter.SharedState.resourceVariables[resourceKindedValue] = variable
}

// checkInvalidatedResourceUse checks whether a resource variable is used after invalidation.
func (interpreter *Interpreter) checkInvalidatedResourceUse(
	value Value,
	variable Variable,
	identifier string,
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
		panic(&InvalidatedResourceError{})
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

// MeterComputation delegates the computation usage to the interpreter's computation gauge, if any.
func (interpreter *Interpreter) MeterComputation(usage common.ComputationUsage) error {
	if interpreter != nil {
		config := interpreter.SharedState.Config
		common.UseComputation(config.ComputationGauge, usage)
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

func NativeCapabilityBorrowFunction(
	addressValuePointer *AddressValue,
	capabilityIDPointer *UInt64Value,
	capabilityBorrowTypePointer *sema.ReferenceType,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		var capabilityBorrowType *sema.ReferenceType
		var capabilityID UInt64Value
		var addressValue AddressValue

		if capabilityBorrowTypePointer == nil {
			// VM does not provide the borrow type
			var idCapabilityValue *IDCapabilityValue

			switch capabilityValue := receiver.(type) {
			case *PathCapabilityValue: //nolint:staticcheck
				// Borrowing of path values is never allowed
				return Nil

			case *IDCapabilityValue:
				idCapabilityValue = capabilityValue

			default:
				panic(errors.NewUnreachableError())
			}
			capabilityID = idCapabilityValue.ID

			if capabilityID == InvalidCapabilityID {
				return Nil
			}

			capabilityBorrowType = context.SemaTypeFromStaticType(idCapabilityValue.BorrowType).(*sema.ReferenceType)
			addressValue = idCapabilityValue.Address()
		} else {
			capabilityBorrowType = capabilityBorrowTypePointer
			capabilityID = *capabilityIDPointer
			addressValue = *addressValuePointer
		}

		typeArgument := typeArguments.NextSema()

		return CapabilityBorrow(
			context,
			typeArgument,
			addressValue,
			capabilityID,
			capabilityBorrowType,
		)
	}
}

// Use function types with type-erased borrow types,
// as we cannot know the borrow types at function creation time.
// DO NOT use the borrow type of the capability value!

var CapabilityTypeBorrowFunctionType = sema.CapabilityTypeBorrowFunctionType(sema.AnyStructType)
var CapabilityTypeCheckFunctionType = sema.CapabilityTypeCheckFunctionType(sema.AnyStructType)

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
		CapabilityTypeBorrowFunctionType,
		NativeCapabilityBorrowFunction(&addressValue, &capabilityID, capabilityBorrowType),
	)
}

func CapabilityBorrow(
	invocationContext InvocationContext,
	typeArgument sema.Type,
	addressValue AddressValue,
	capabilityID UInt64Value,
	capabilityBorrowType *sema.ReferenceType,
) Value {
	if capabilityID == InvalidCapabilityID {
		return Nil
	}

	var wantedBorrowType *sema.ReferenceType
	if typeArgument != nil {
		var ok bool
		wantedBorrowType, ok = typeArgument.(*sema.ReferenceType)
		if !ok {
			panic(errors.NewUnreachableError())
		}
	}

	borrowHandler := invocationContext.GetCapabilityBorrowHandler()

	referenceValue := borrowHandler(
		invocationContext,
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

func NativeCapabilityCheckFunction(
	addressValuePointer *AddressValue,
	capabilityIDPointer *UInt64Value,
	capabilityBorrowTypePointer *sema.ReferenceType,
) NativeFunction {
	return func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		var capabilityBorrowType *sema.ReferenceType
		var capabilityID UInt64Value
		var addressValue AddressValue

		if capabilityBorrowTypePointer == nil {
			// VM does not provide the borrow type
			var idCapabilityValue *IDCapabilityValue

			switch capabilityValue := receiver.(type) {
			case *PathCapabilityValue: //nolint:staticcheck
				// Borrowing of path values is never allowed
				return FalseValue

			case *IDCapabilityValue:
				idCapabilityValue = capabilityValue

			default:
				panic(errors.NewUnreachableError())
			}

			capabilityID = idCapabilityValue.ID

			if capabilityID == InvalidCapabilityID {
				return FalseValue
			}

			capabilityBorrowType = context.SemaTypeFromStaticType(idCapabilityValue.BorrowType).(*sema.ReferenceType)
			addressValue = idCapabilityValue.Address()
		} else {
			capabilityBorrowType = capabilityBorrowTypePointer
			capabilityID = *capabilityIDPointer
			addressValue = *addressValuePointer
		}

		typeArgument := typeArguments.NextSema()

		return CapabilityCheck(
			context,
			typeArgument,
			addressValue,
			capabilityID,
			capabilityBorrowType,
		)
	}
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
		CapabilityTypeCheckFunctionType,
		NativeCapabilityCheckFunction(&addressValue, &capabilityID, capabilityBorrowType),
	)
}

func CapabilityCheck(
	invocationContext InvocationContext,
	typeArgument sema.Type,
	addressValue AddressValue,
	capabilityID UInt64Value,
	capabilityBorrowType *sema.ReferenceType,
) Value {

	if capabilityID == InvalidCapabilityID {
		return FalseValue
	}

	var wantedBorrowType *sema.ReferenceType
	if typeArgument != nil {
		var ok bool
		wantedBorrowType, ok = typeArgument.(*sema.ReferenceType)
		if !ok {
			panic(errors.NewUnreachableError())
		}
	}

	checkHandler := invocationContext.GetCapabilityCheckHandler()

	return checkHandler(
		invocationContext,
		addressValue,
		capabilityID,
		wantedBorrowType,
		capabilityBorrowType,
	)
}

func (interpreter *Interpreter) ValidateContainerMutation(valueID atree.ValueID) {
	_, present := interpreter.SharedState.containerValueIteration[valueID]
	if !present {
		return
	}
	panic(&ContainerMutatedDuringIterationError{})
}

func (interpreter *Interpreter) WithContainerMutationPrevention(valueID atree.ValueID, f func()) {
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

func (interpreter *Interpreter) EnforceNotResourceDestruction(valueID atree.ValueID) {
	_, exists := interpreter.SharedState.destroyedResources[valueID]
	if exists {
		panic(&DestroyedResourceError{})
	}
}

func (interpreter *Interpreter) WithResourceDestruction(valueID atree.ValueID, f func()) {
	interpreter.EnforceNotResourceDestruction(valueID)

	interpreter.SharedState.destroyedResources[valueID] = struct{}{}

	f()
}

func CheckResourceLoss(context ValueStaticTypeContext, value Value) {
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
		panic(&ResourceLossError{})
	}
}

func (interpreter *Interpreter) OnResourceOwnerChange(resource *CompositeValue, oldOwner common.Address, newOwner common.Address) {
	onResourceOwnerChange := interpreter.SharedState.Config.OnResourceOwnerChange
	if onResourceOwnerChange == nil {
		return
	}

	onResourceOwnerChange(interpreter, resource, oldOwner, newOwner)
}

func (interpreter *Interpreter) IsTypeInfoRecovered(location common.Location) bool {
	elaboration := interpreter.getElaboration(location)
	if elaboration == nil {
		return false
	}

	return elaboration.IsRecovered
}

func (interpreter *Interpreter) GetAccountHandlerFunc() AccountHandlerFunc {
	return interpreter.SharedState.Config.AccountHandler
}

func (interpreter *Interpreter) GetInjectedCompositeFieldsHandler() InjectedCompositeFieldsHandlerFunc {
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

func (interpreter *Interpreter) GetValidateAccountCapabilitiesGetHandler() ValidateAccountCapabilitiesGetHandlerFunc {
	return interpreter.SharedState.Config.ValidateAccountCapabilitiesGetHandler
}

func (interpreter *Interpreter) GetValidateAccountCapabilitiesPublishHandler() ValidateAccountCapabilitiesPublishHandlerFunc {
	return interpreter.SharedState.Config.ValidateAccountCapabilitiesPublishHandler
}

func (interpreter *Interpreter) GetCapabilityBorrowHandler() CapabilityBorrowHandlerFunc {
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

func (interpreter *Interpreter) GetMethod(value MemberAccessibleValue, name string) FunctionValue {
	return value.GetMethod(interpreter, name)
}

func (interpreter *Interpreter) GetGlobal(name string) Value {
	return interpreter.Globals.Get(name).GetValue(interpreter)
}

func (interpreter *Interpreter) GetGlobalType(name string) (*sema.Variable, bool) {
	return interpreter.Program.Elaboration.GetGlobalType(name)
}

func (interpreter *Interpreter) DefaultDestroyEvents(resourceValue *CompositeValue) []*CompositeValue {
	return resourceValue.DefaultDestroyEvents(interpreter)
}

func (interpreter *Interpreter) SemaTypeFromStaticType(staticType StaticType) sema.Type {
	return MustConvertStaticToSemaType(staticType, interpreter)
}

func (interpreter *Interpreter) MaybeUpdateStorageReferenceMemberReceiver(
	storageReference *StorageReferenceValue,
	referencedValue Value,
	member Value,
) Value {
	if boundFunction, isBoundFunction := member.(BoundFunctionValue); isBoundFunction {
		boundFunction.SelfReference = StorageReference(
			interpreter,
			storageReference,
			referencedValue,
		)
		return boundFunction
	}

	return member
}

func (interpreter *Interpreter) SemaAccessFromStaticAuthorization(auth Authorization) (sema.Access, error) {
	return ConvertStaticAuthorizationToSemaAccess(auth, interpreter)
}

func StorageReference(
	context ValueStaticTypeContext,
	storageReference *StorageReferenceValue,
	referencedValue Value,
) *StorageReferenceValue {

	// As also mentioned in `(StorageReference).GetMember` method,
	// we cannot use the storage reference as-is here.
	// This is because since we look up the member on the referenced value,
	// we also must use its type as the borrowed type for the `SelfReference` type,
	// because during invocation the bound function can only be invoked
	// if the type of the dereferenced value at that time still matches
	// the type of the dereferenced value at the time of binding (here).
	//
	// For example, imagine storing a value of type T (e.g. `String`),
	// creating a reference with a supertype (e.g. `AnyStruct`),
	// and then creating a bound function on it.
	// Then, if we change the storage location to store a value of unrelated type U instead (e.g. `Int`),
	// and invoke the bound function, the bound function is potentially invalid.

	referencedValueStaticType := referencedValue.StaticType(context)
	return NewStorageReferenceValue(
		context,
		storageReference.Authorization,
		storageReference.TargetStorageAddress,
		storageReference.TargetPath,
		context.SemaTypeFromStaticType(referencedValueStaticType),
	)
}
