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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
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
		value = interpreter.transferAndConvert(value, valueType, returnType, locationRange)
	}

	return ReturnResult{value}
}

var theBreakResult StatementResult = BreakResult{}

func (interpreter *Interpreter) VisitBreakStatement(_ *ast.BreakStatement) StatementResult {
	return theBreakResult
}

var theContinueResult StatementResult = ContinueResult{}

func (interpreter *Interpreter) VisitContinueStatement(_ *ast.ContinueStatement) StatementResult {
	return theContinueResult
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

	// NOTE: It is *REQUIRED* that the getter for the value is used
	// instead of just evaluating value expression,
	// as the value may be an access expression (member access, index access),
	// which implicitly removes a resource.
	//
	// Performing the removal from the container is essential
	// (and just evaluating the expression does not perform the removal),
	// because if there is a second value,
	// the assignment to the value will cause an overwrite of the value.
	// If the resource was not moved ou of the container,
	// its contents get deleted.

	const allowMissing = false
	value := interpreter.assignmentGetterSetter(declaration.Value).get(allowMissing)
	if value == nil {
		panic(errors.NewUnreachableError())
	}

	variableDeclarationTypes := interpreter.Program.Elaboration.VariableDeclarationTypes(declaration)
	valueType := variableDeclarationTypes.ValueType

	if declaration.SecondValue != nil {
		secondValueType := variableDeclarationTypes.SecondValueType

		interpreter.visitAssignment(
			declaration.Transfer.Operation,
			declaration.Value,
			valueType,
			declaration.SecondValue,
			secondValueType,
			declaration,
		)
	}

	if someValue, ok := value.(*SomeValue); ok {

		targetType := variableDeclarationTypes.TargetType
		locationRange := LocationRange{
			Location:    interpreter.Location,
			HasPosition: declaration.Value,
		}
		innerValue := someValue.InnerValue(interpreter, locationRange)
		transferredUnwrappedValue := interpreter.transferAndConvert(
			innerValue,
			valueType,
			targetType,
			locationRange,
		)

		interpreter.activations.PushNewWithCurrent()
		defer interpreter.activations.Pop()

		// Assignment can also be a resource move.
		interpreter.invalidateResource(innerValue)

		interpreter.declareVariable(
			declaration.Identifier.Identifier,
			transferredUnwrappedValue,
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

func (interpreter *Interpreter) VisitForStatement(statement *ast.ForStatement) StatementResult {

	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	variable := interpreter.declareVariable(
		statement.Identifier.Identifier,
		nil,
	)

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: statement,
	}

	value := interpreter.evalExpression(statement.Value)
	transferredValue := value.Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		false,
		nil,
	)

	iterable, ok := transferredValue.(IterableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	iterator := iterable.Iterator(interpreter)

	var indexVariable *Variable
	if statement.Index != nil {
		indexVariable = interpreter.declareVariable(
			statement.Index.Identifier,
			NewIntValueFromInt64(interpreter, 0),
		)
	}

	for {
		value := iterator.Next(interpreter)
		if value == nil {
			return nil
		}

		interpreter.reportLoopIteration(statement)

		variable.SetValue(value)

		result := interpreter.visitBlock(statement.Block)

		switch result.(type) {
		case BreakResult:
			return nil

		case ContinueResult:
			// NO-OP

		case ReturnResult:
			return result
		}

		if indexVariable != nil {
			currentIndex := indexVariable.GetValue().(IntValue)
			nextIndex := currentIndex.Plus(interpreter, intOne)
			indexVariable.SetValue(nextIndex)
		}
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

	return nil
}

func (interpreter *Interpreter) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) StatementResult {
	return nil
}

// VisitVariableDeclaration first visits the declaration's value,
// then declares the variable with the name bound to the value
func (interpreter *Interpreter) VisitVariableDeclaration(declaration *ast.VariableDeclaration) StatementResult {

	interpreter.visitVariableDeclaration(
		declaration,
		func(identifier string, value Value) {

			// NOTE: lexical scope, always declare a new variable.
			// Do not find an existing variable and assign the value!

			_ = interpreter.declareVariable(
				identifier,
				value,
			)
		},
	)

	return nil
}

func (interpreter *Interpreter) visitVariableDeclaration(
	declaration *ast.VariableDeclaration,
	valueCallback func(identifier string, value Value),
) {

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
	// If the resource was not moved ou of the container,
	// its contents get deleted.

	const allowMissing = false
	result := interpreter.assignmentGetterSetter(declaration.Value).get(allowMissing)
	if result == nil {
		panic(errors.NewUnreachableError())
	}

	// Assignment is a potential resource move.
	interpreter.invalidateResource(result)

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: declaration.Value,
	}

	transferredValue := interpreter.transferAndConvert(result, valueType, targetType, locationRange)

	valueCallback(
		declaration.Identifier.Identifier,
		transferredValue,
	)

	if declaration.SecondValue == nil {
		return
	}

	interpreter.visitAssignment(
		declaration.Transfer.Operation,
		declaration.Value,
		valueType,
		declaration.SecondValue,
		secondValueType,
		declaration,
	)
}

func (interpreter *Interpreter) VisitAssignmentStatement(assignment *ast.AssignmentStatement) StatementResult {
	assignmentStatementTypes := interpreter.Program.Elaboration.AssignmentStatementTypes(assignment)
	targetType := assignmentStatementTypes.TargetType
	valueType := assignmentStatementTypes.ValueType

	target := assignment.Target
	value := assignment.Value

	interpreter.visitAssignment(
		assignment.Transfer.Operation,
		target, targetType,
		value, valueType,
		assignment,
	)

	return nil
}

func (interpreter *Interpreter) VisitSwapStatement(swap *ast.SwapStatement) StatementResult {
	swapStatementTypes := interpreter.Program.Elaboration.SwapStatementTypes(swap)
	leftType := swapStatementTypes.LeftType
	rightType := swapStatementTypes.RightType

	const allowMissing = false

	// Evaluate the left expression
	leftGetterSetter := interpreter.assignmentGetterSetter(swap.Left)
	leftValue := leftGetterSetter.get(allowMissing)
	interpreter.checkSwapValue(leftValue, swap.Left)

	// Evaluate the right expression
	rightGetterSetter := interpreter.assignmentGetterSetter(swap.Right)
	rightValue := rightGetterSetter.get(allowMissing)
	interpreter.checkSwapValue(rightValue, swap.Right)

	// Set right value to left target
	// and left value to right target

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: swap.Right,
	}
	transferredRightValue := interpreter.transferAndConvert(rightValue, rightType, leftType, locationRange)

	locationRange = LocationRange{
		Location:    interpreter.Location,
		HasPosition: swap.Left,
	}
	transferredLeftValue := interpreter.transferAndConvert(leftValue, leftType, rightType, locationRange)

	leftGetterSetter.set(transferredRightValue)
	rightGetterSetter.set(transferredLeftValue)

	return nil
}

func (interpreter *Interpreter) checkSwapValue(value Value, expression ast.Expression) {
	if value != nil {
		return
	}

	if expression, ok := expression.(*ast.MemberExpression); ok {
		panic(MissingMemberValueError{
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
	return ExpressionResult{result}
}
