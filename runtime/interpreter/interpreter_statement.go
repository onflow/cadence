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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
)

func (interpreter *Interpreter) evalStatement(statement ast.Statement) interface{} {

	// Recover and re-throw a panic, so that this interpreter's location and statement are used,
	// instead of a potentially calling interpreter's location and statement

	defer interpreter.recoverErrors(func(internalErr error) {
		panic(internalErr)
	})

	interpreter.statement = statement

	if interpreter.onStatement != nil {
		interpreter.onStatement(interpreter, statement)
	}

	return statement.Accept(interpreter)
}

func (interpreter *Interpreter) visitStatements(statements []ast.Statement) controlReturn {

	for _, statement := range statements {
		result := interpreter.evalStatement(statement)
		if ret, ok := result.(controlReturn); ok {
			return ret
		}
	}

	return nil
}

func (interpreter *Interpreter) VisitReturnStatement(statement *ast.ReturnStatement) ast.Repr {
	// NOTE: returning result

	var value Value
	if statement.Expression == nil {
		value = VoidValue{}
	} else {
		value = interpreter.evalExpression(statement.Expression)

		valueType := interpreter.Program.Elaboration.ReturnStatementValueTypes[statement]
		returnType := interpreter.Program.Elaboration.ReturnStatementReturnTypes[statement]

		getLocationRange := locationRangeGetter(interpreter.Location, statement.Expression)

		// NOTE: copy on return
		value = interpreter.copyAndConvert(value, valueType, returnType, getLocationRange)
	}

	return functionReturn{value}
}

func (interpreter *Interpreter) VisitBreakStatement(_ *ast.BreakStatement) ast.Repr {
	return controlBreak{}
}

func (interpreter *Interpreter) VisitContinueStatement(_ *ast.ContinueStatement) ast.Repr {
	return controlContinue{}
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
) controlReturn {

	value := interpreter.evalExpression(test).(BoolValue)
	var result interface{}
	if value {
		result = thenBlock.Accept(interpreter)
	} else if elseBlock != nil {
		result = elseBlock.Accept(interpreter)
	}

	if ret, ok := result.(controlReturn); ok {
		return ret
	}
	return nil
}

func (interpreter *Interpreter) visitIfStatementWithVariableDeclaration(
	declaration *ast.VariableDeclaration,
	thenBlock, elseBlock *ast.Block,
) controlReturn {

	value := interpreter.evalExpression(declaration.Value)
	var result interface{}
	if someValue, ok := value.(*SomeValue); ok {

		targetType := interpreter.Program.Elaboration.VariableDeclarationTargetTypes[declaration]
		valueType := interpreter.Program.Elaboration.VariableDeclarationValueTypes[declaration]
		getLocationRange := locationRangeGetter(interpreter.Location, declaration.Value)
		unwrappedValueCopy := interpreter.copyAndConvert(someValue.Value, valueType, targetType, getLocationRange)

		interpreter.activations.PushNewWithCurrent()
		defer interpreter.activations.Pop()

		interpreter.declareVariable(
			declaration.Identifier.Identifier,
			unwrappedValueCopy,
		)

		result = thenBlock.Accept(interpreter)
	} else if elseBlock != nil {
		result = elseBlock.Accept(interpreter)
	}

	if ret, ok := result.(controlReturn); ok {
		return ret
	}
	return nil
}

func (interpreter *Interpreter) VisitSwitchStatement(switchStatement *ast.SwitchStatement) ast.Repr {

	testValue := interpreter.evalExpression(switchStatement.Expression).(EquatableValue)

	for _, switchCase := range switchStatement.Cases {

		runStatements := func() ast.Repr {
			// NOTE: the new block ensures that a new scope is introduced

			block := &ast.Block{
				Statements: switchCase.Statements,
			}

			result := block.Accept(interpreter)

			if _, ok := result.(controlBreak); ok {
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

		caseValue := result.(EquatableValue)

		// If the test value and case values are equal,
		// evaluate the case's statements

		if testValue.Equal(caseValue, interpreter, true) {
			return runStatements()
		}

		// If the test value and the case values are unequal,
		// then try the next case
	}

	return nil
}

func (interpreter *Interpreter) VisitWhileStatement(statement *ast.WhileStatement) ast.Repr {

	for {

		value := interpreter.evalExpression(statement.Test).(BoolValue)
		if !value {
			return nil
		}

		interpreter.reportLoopIteration(statement)

		result := statement.Block.Accept(interpreter)

		switch result.(type) {
		case controlBreak:
			return nil

		case controlContinue:
			// NO-OP

		case functionReturn:
			return result
		}
	}
}

func (interpreter *Interpreter) VisitForStatement(statement *ast.ForStatement) ast.Repr {

	interpreter.activations.PushNewWithCurrent()
	defer interpreter.activations.Pop()

	variable := interpreter.declareVariable(
		statement.Identifier.Identifier,
		nil,
	)

	values := interpreter.evalExpression(statement.Value).(*ArrayValue).Values[:]

	for _, value := range values {

		interpreter.reportLoopIteration(statement)

		variable.SetValue(value)

		result := statement.Block.Accept(interpreter)

		switch result.(type) {
		case controlBreak:
			return nil

		case controlContinue:
			// NO-OP

		case functionReturn:
			return result
		}
	}

	return nil
}

func (interpreter *Interpreter) VisitEmitStatement(statement *ast.EmitStatement) ast.Repr {
	event := interpreter.evalExpression(statement.InvocationExpression).(*CompositeValue)

	eventType := interpreter.Program.Elaboration.EmitStatementEventTypes[statement]

	if interpreter.onEventEmitted == nil {
		panic(EventEmissionUnavailableError{
			LocationRange: locationRangeGetter(interpreter.Location, statement)(),
		})
	}

	err := interpreter.onEventEmitted(interpreter, event, eventType)
	if err != nil {
		panic(err)
	}

	return nil
}

func (interpreter *Interpreter) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) ast.Repr {
	return nil
}

// VisitVariableDeclaration first visits the declaration's value,
// then declares the variable with the name bound to the value
func (interpreter *Interpreter) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Repr {

	targetType := interpreter.Program.Elaboration.VariableDeclarationTargetTypes[declaration]
	valueType := interpreter.Program.Elaboration.VariableDeclarationValueTypes[declaration]
	secondValueType := interpreter.Program.Elaboration.VariableDeclarationSecondValueTypes[declaration]

	result := interpreter.evalPotentialResourceMoveIndexExpression(declaration.Value)
	if result == nil {
		panic(errors.NewUnreachableError())
	}

	getLocationRange := locationRangeGetter(interpreter.Location, declaration.Value)

	valueCopy := interpreter.copyAndConvert(result, valueType, targetType, getLocationRange)

	interpreter.declareVariable(
		declaration.Identifier.Identifier,
		valueCopy,
	)

	if declaration.SecondValue == nil {
		return nil
	}

	interpreter.visitAssignment(
		declaration.Transfer.Operation,
		declaration.Value,
		valueType,
		declaration.SecondValue,
		secondValueType,
		declaration,
	)

	return nil
}

func (interpreter *Interpreter) VisitAssignmentStatement(assignment *ast.AssignmentStatement) ast.Repr {
	targetType := interpreter.Program.Elaboration.AssignmentStatementTargetTypes[assignment]
	valueType := interpreter.Program.Elaboration.AssignmentStatementValueTypes[assignment]

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

func (interpreter *Interpreter) VisitSwapStatement(swap *ast.SwapStatement) ast.Repr {

	leftType := interpreter.Program.Elaboration.SwapStatementLeftTypes[swap]
	rightType := interpreter.Program.Elaboration.SwapStatementRightTypes[swap]

	// Evaluate the left expression
	leftGetterSetter := interpreter.assignmentGetterSetter(swap.Left)
	leftValue := leftGetterSetter.get()
	interpreter.checkSwapValue(leftValue, swap.Left)
	if interpreter.resourceMoveIndexExpression(swap.Left) != nil {
		leftGetterSetter.set(NilValue{})
	}

	// Evaluate the right expression
	rightGetterSetter := interpreter.assignmentGetterSetter(swap.Right)
	rightValue := rightGetterSetter.get()
	interpreter.checkSwapValue(rightValue, swap.Right)
	if interpreter.resourceMoveIndexExpression(swap.Right) != nil {
		rightGetterSetter.set(NilValue{})
	}

	// Add right value to left target
	// and left value to right target

	getLocationRange := locationRangeGetter(interpreter.Location, swap.Right)
	rightValueCopy := interpreter.copyAndConvert(rightValue, rightType, leftType, getLocationRange)

	getLocationRange = locationRangeGetter(interpreter.Location, swap.Left)
	leftValueCopy := interpreter.copyAndConvert(leftValue, leftType, rightType, getLocationRange)

	leftGetterSetter.set(rightValueCopy)
	rightGetterSetter.set(leftValueCopy)

	return nil
}

func (interpreter *Interpreter) checkSwapValue(value Value, expression ast.Expression) {
	if value != nil {
		return
	}

	if expression, ok := expression.(*ast.MemberExpression); ok {
		panic(MissingMemberValueError{
			Name:          expression.Identifier.Identifier,
			LocationRange: locationRangeGetter(interpreter.Location, expression)(),
		})
	}

	panic(errors.NewUnreachableError())
}

func (interpreter *Interpreter) VisitExpressionStatement(statement *ast.ExpressionStatement) ast.Repr {
	result := interpreter.evalExpression(statement.Expression)
	return ExpressionStatementResult{result}
}
