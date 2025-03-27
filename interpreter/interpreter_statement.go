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
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

func (interpreter *Interpreter) evalStatement(statement ast.Statement) StatementResult {

	// Recover and re-throw a panic, so that this interpreter's location and statement are used,
	// instead of a potentially calling interpreter's location and statement

	defer interpreter.RecoverErrors(func(internalErr error) {
		panic(internalErr)
	})

	interpreter.statement = statement

	config := interpreter.SharedState.Config

	onMeterComputation := config.OnMeterComputation
	if onMeterComputation != nil {
		onMeterComputation(common.ComputationKindStatement, 1)
	}

	debugger := config.Debugger
	if debugger != nil {
		debugger.onStatement(interpreter, statement)
	}

	onStatement := config.OnStatement
	if onStatement != nil {
		onStatement(interpreter, statement)
	}

	return ast.AcceptStatement[StatementResult](statement, interpreter)
}

func (interpreter *Interpreter) visitStatements(statements []ast.Statement) StatementResult {

	for _, statement := range statements {
		result := interpreter.evalStatement(statement)
		if result, ok := result.(controlResult); ok {
			return result
		}
	}

	return nil
}

func (interpreter *Interpreter) VisitReturnStatement(statement *ast.ReturnStatement) StatementResult {
	// NOTE: returning result

	var value Value
	if statement.Expression == nil {
		value = Void
	} else {
		value = interpreter.evalExpression(statement.Expression)

		returnStatementTypes := interpreter.Program.Elaboration.ReturnStatementTypes(statement)
		valueType := returnStatementTypes.ValueType
		returnType := returnStatementTypes.ReturnType

		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: statement.Expression,
		}

		// NOTE: copy on return
		value = transferAndConvert(interpreter, value, valueType, returnType, locationRange)
	}

	return ReturnResult{Value: value}
}

var theBreakResult StatementResult = BreakResult{}

func (interpreter *Interpreter) VisitBreakStatement(_ *ast.BreakStatement) StatementResult {
	return theBreakResult
}

var theContinueResult StatementResult = ContinueResult{}

func (interpreter *Interpreter) VisitContinueStatement(_ *ast.ContinueStatement) StatementResult {
	return theContinueResult
}

func (interpreter *Interpreter) VisitEntitlementDeclaration(_ *ast.EntitlementDeclaration) StatementResult {
	// TODO
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) VisitEntitlementMappingDeclaration(_ *ast.EntitlementMappingDeclaration) StatementResult {
	// TODO
	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) VisitIfStatement(statement *ast.IfStatement) StatementResult {
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
) StatementResult {

	value, ok := interpreter.evalExpression(test).(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	if value {
		return interpreter.visitBlock(thenBlock)
	} else if elseBlock != nil {
		return interpreter.visitBlock(elseBlock)
	}

	return nil
}

func (interpreter *Interpreter) visitIfStatementWithVariableDeclaration(
	declaration *ast.VariableDeclaration,
	thenBlock, elseBlock *ast.Block,
) StatementResult {

	value := interpreter.visitVariableDeclaration(declaration, true)

	if someValue, ok := value.(*SomeValue); ok {

		innerValue := someValue.InnerValue()

		interpreter.activations.PushNewWithCurrent()
		defer interpreter.activations.Pop()

		interpreter.declareVariable(
			declaration.Identifier.Identifier,
			innerValue,
		)

		return interpreter.visitBlock(thenBlock)
	} else if elseBlock != nil {
		return interpreter.visitBlock(elseBlock)
	}

	return nil
}

func (interpreter *Interpreter) VisitSwitchStatement(switchStatement *ast.SwitchStatement) StatementResult {

	testValue, ok := interpreter.evalExpression(switchStatement.Expression).(EquatableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	for _, switchCase := range switchStatement.Cases {

		runStatements := func() StatementResult {
			// NOTE: the new block ensures that a new scope is introduced

			block := ast.NewBlock(
				interpreter,
				switchCase.Statements,
				ast.EmptyRange,
			)

			result := interpreter.visitBlock(block)

			if _, ok := result.(BreakResult); ok {
				return nil
			}

			return result
		}

		// If the case has no expression it is the default case.
		// Evaluate it, i.e. all statements

		if switchCase.Expression == nil {
			return runStatements()
		}

		// The case has an expression.
		// Evaluate it and compare it to the test value

		result := interpreter.evalExpression(switchCase.Expression)

		caseValue, ok := result.(EquatableValue)

		if !ok {
			continue
		}

		// If the test value and case values are equal,
		// evaluate the case's statements

		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: switchCase.Expression,
		}

		if testValue.Equal(interpreter, locationRange, caseValue) {
			return runStatements()
		}

		// If the test value and the case values are unequal,
		// then try the next case
	}

	return nil
}

func (interpreter *Interpreter) VisitWhileStatement(statement *ast.WhileStatement) StatementResult {

	for {

		value, ok := interpreter.evalExpression(statement.Test).(BoolValue)
		if !ok || !bool(value) {
			return nil
		}

		interpreter.reportLoopIteration(statement)

		result := interpreter.visitBlock(statement.Block)

		switch result.(type) {
		case BreakResult:
			return nil

		case ContinueResult:
			// NO-OP

		case ReturnResult:
			return result
		}
	}
}

var intOne = NewUnmeteredIntValueFromInt64(1)

func (interpreter *Interpreter) VisitForStatement(statement *ast.ForStatement) (result StatementResult) {

	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: statement,
	}

	value := interpreter.evalExpression(statement.Value)

	// Do not transfer the iterable value.
	// Instead, transfer each iterating element.
	// This is done in `ForEach` method.

	iterable, ok := value.(IterableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	forStmtTypes := interpreter.Program.Elaboration.ForStatementType(statement)

	var index IntValue
	if statement.Index != nil {
		index = NewIntValueFromInt64(interpreter, 0)
	}

	executeBody := func(value Value) (resume bool) {
		statementResult, done := interpreter.visitForStatementBody(statement, index, value)
		if done {
			result = statementResult
		}

		resume = !done

		if statement.Index != nil {
			index = index.Plus(interpreter, intOne, locationRange).(IntValue)
		}

		return
	}

	// Transfer the elements before pass onto the loop-body.
	const transferElements = true

	iterable.ForEach(
		interpreter,
		forStmtTypes.ValueVariableType,
		executeBody,
		transferElements,
		locationRange,
	)

	return
}

func (interpreter *Interpreter) visitForStatementBody(
	statement *ast.ForStatement,
	index IntValue,
	value Value,
) (
	result StatementResult,
	done bool,
) {
	interpreter.reportLoopIteration(statement)

	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	if index.BigInt != nil {
		interpreter.declareVariable(
			statement.Index.Identifier,
			index,
		)
	}

	interpreter.declareVariable(
		statement.Identifier.Identifier,
		value,
	)

	result = interpreter.visitBlock(statement.Block)

	switch result.(type) {
	case BreakResult:
		return nil, true

	case ContinueResult:
		// NO-OP

	case ReturnResult:
		return result, true
	}

	return nil, false
}

func (interpreter *Interpreter) EmitEventValue(event *CompositeValue, eventType *sema.CompositeType, locationRange LocationRange) {

	config := interpreter.SharedState.Config

	onEventEmitted := config.OnEventEmitted
	if onEventEmitted == nil {
		panic(EventEmissionUnavailableError{
			LocationRange: locationRange,
		})
	}

	err := onEventEmitted(interpreter, locationRange, event, eventType)
	if err != nil {
		panic(err)
	}
}

func (interpreter *Interpreter) VisitEmitStatement(statement *ast.EmitStatement) StatementResult {

	event, ok := interpreter.evalExpression(statement.InvocationExpression).(*CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	eventType := interpreter.Program.Elaboration.EmitStatementEventType(statement)

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: statement,
	}

	interpreter.EmitEventValue(event, eventType, locationRange)

	return nil
}

func (interpreter *Interpreter) VisitRemoveStatement(removeStatement *ast.RemoveStatement) StatementResult {

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: removeStatement,
	}

	removeTarget := interpreter.evalExpression(removeStatement.Value)
	base, ok := removeTarget.(*CompositeValue)

	// we enforce this in the checker, but check defensively anyways
	if !ok || !base.Kind.SupportsAttachments() {
		panic(InvalidAttachmentOperationTargetError{
			Value:         removeTarget,
			LocationRange: locationRange,
		})
	}

	if inIteration := interpreter.SharedState.inAttachmentIteration(base); inIteration {
		panic(AttachmentIterationMutationError{
			Value:         base,
			LocationRange: locationRange,
		})
	}

	nominalType := interpreter.Program.Elaboration.AttachmentRemoveTypes(removeStatement)

	removed := base.RemoveTypeKey(interpreter, locationRange, nominalType)

	// attachment not present on this base
	if removed == nil {
		return nil
	}

	attachment, ok := removed.(*CompositeValue)
	// we enforce this in the checker
	if !ok {
		panic(errors.NewUnreachableError())
	}

	if attachment.IsResourceKinded(interpreter) {
		// this attachment is no longer attached to its base, but the `base` variable is still available in the destructor
		attachment.setBaseValue(base)
		attachment.Destroy(interpreter, locationRange)
	}

	return nil
}

func (interpreter *Interpreter) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) StatementResult {
	return nil
}

// VisitVariableDeclaration first visits the declaration's value,
// then declares the variable with the name bound to the value
func (interpreter *Interpreter) VisitVariableDeclaration(declaration *ast.VariableDeclaration) StatementResult {

	value := interpreter.visitVariableDeclaration(declaration, false)

	// NOTE: lexical scope, always declare a new variable.
	// Do not find an existing variable and assign the value!

	_ = interpreter.declareVariable(
		declaration.Identifier.Identifier,
		value,
	)

	return nil
}

func (interpreter *Interpreter) visitVariableDeclaration(
	declaration *ast.VariableDeclaration,
	isOptionalBinding bool,
) Value {

	variableDeclarationTypes := interpreter.Program.Elaboration.VariableDeclarationTypes(declaration)
	targetType := variableDeclarationTypes.TargetType
	valueType := variableDeclarationTypes.ValueType
	secondValueType := variableDeclarationTypes.SecondValueType

	// NOTE: It is *REQUIRED* that the getter for the value is used
	// instead of just evaluating value expression,
	// as the value may be an access expression (member access, index access),
	// which implicitly removes a resource.
	//
	// Performing the removal from the container is essential
	// (and just evaluating the expression does not perform the removal),
	// because if there is a second value,
	// the assignment to the value will cause an overwrite of the value.
	// If the resource was not moved out of the container,
	// its contents get deleted.

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: declaration.Value,
	}

	getterSetter := interpreter.assignmentGetterSetter(declaration.Value, locationRange)

	const allowMissing = false
	result := getterSetter.get(allowMissing)
	if result == nil {
		panic(errors.NewUnreachableError())
	}

	if isOptionalBinding {
		targetType = &sema.OptionalType{
			Type: targetType,
		}
	}

	transferredValue := transferAndConvert(
		interpreter,
		result,
		valueType,
		targetType,
		locationRange,
	)

	// Assignment is a potential resource move.
	interpreter.invalidateResource(result)

	if declaration.SecondValue != nil {
		interpreter.visitAssignment(
			declaration.Transfer.Operation,
			getterSetter,
			valueType,
			declaration.SecondValue,
			secondValueType,
			declaration,
		)
	}

	return transferredValue
}

func (interpreter *Interpreter) VisitAssignmentStatement(assignment *ast.AssignmentStatement) StatementResult {
	assignmentStatementTypes := interpreter.Program.Elaboration.AssignmentStatementTypes(assignment)
	targetType := assignmentStatementTypes.TargetType
	valueType := assignmentStatementTypes.ValueType

	target := assignment.Target
	value := assignment.Value

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: target,
	}

	getterSetter := interpreter.assignmentGetterSetter(target, locationRange)

	interpreter.visitAssignment(
		assignment.Transfer.Operation,
		getterSetter, targetType,
		value, valueType,
		assignment,
	)

	return nil
}

func (interpreter *Interpreter) VisitSwapStatement(swap *ast.SwapStatement) StatementResult {

	// Get type information

	swapStatementTypes := interpreter.Program.Elaboration.SwapStatementTypes(swap)
	leftType := swapStatementTypes.LeftType
	rightType := swapStatementTypes.RightType

	// Evaluate the left side (target and key)

	leftLocationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: swap.Left,
	}

	leftGetterSetter := interpreter.assignmentGetterSetter(swap.Left, leftLocationRange)

	// Evaluate the right side (target and key)

	rightLocationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: swap.Right,
	}

	rightGetterSetter := interpreter.assignmentGetterSetter(swap.Right, rightLocationRange)

	// Get left and right values

	const allowMissing = false

	leftValue := leftGetterSetter.get(allowMissing)
	interpreter.checkSwapValue(leftValue, swap.Left)

	rightValue := rightGetterSetter.get(allowMissing)
	interpreter.checkSwapValue(rightValue, swap.Right)

	// Set right value to left target,
	// and left value to right target

	checkInvalidatedResourceOrResourceReference(rightValue, rightLocationRange, interpreter)
	transferredRightValue := transferAndConvert(interpreter, rightValue, rightType, leftType, rightLocationRange)

	checkInvalidatedResourceOrResourceReference(leftValue, leftLocationRange, interpreter)
	transferredLeftValue := transferAndConvert(interpreter, leftValue, leftType, rightType, leftLocationRange)

	leftGetterSetter.set(transferredRightValue)
	rightGetterSetter.set(transferredLeftValue)

	return nil
}

func (interpreter *Interpreter) checkSwapValue(value Value, expression ast.Expression) {
	if value != nil {
		return
	}

	if expression, ok := expression.(*ast.MemberExpression); ok {
		panic(UseBeforeInitializationError{
			Name: expression.Identifier.Identifier,
			LocationRange: LocationRange{
				Location:    interpreter.Location,
				HasPosition: expression,
			},
		})
	}

	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) VisitExpressionStatement(statement *ast.ExpressionStatement) StatementResult {
	result := interpreter.evalExpression(statement.Expression)
	return ExpressionResult{Value: result}
}
