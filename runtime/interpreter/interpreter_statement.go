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
	"github.com/onflow/cadence/runtime/trampoline"
)

func (interpreter *Interpreter) visitStatements(statements []ast.Statement) trampoline.Trampoline {
	count := len(statements)

	// no statements? stop
	if count == 0 {
		// NOTE: no result, so it does *not* act like a return-statement
		return trampoline.Done{}
	}

	statement := statements[0]

	// interpret the first statement, then the remaining ones
	return StatementTrampoline{
		F: func() trampoline.Trampoline {
			return statement.Accept(interpreter).(trampoline.Trampoline)
		},
		Interpreter: interpreter,
		Statement:   statement,
	}.FlatMap(func(returnValue interface{}) trampoline.Trampoline {
		if _, isReturn := returnValue.(controlReturn); isReturn {
			return trampoline.Done{
				Result: returnValue,
			}
		}
		return interpreter.visitStatements(statements[1:])
	})
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

		// NOTE: copy on return
		value = interpreter.copyAndConvert(value, valueType, returnType)
	}
	return trampoline.Done{
		Result: functionReturn{value},
	}
}

func (interpreter *Interpreter) VisitBreakStatement(_ *ast.BreakStatement) ast.Repr {
	return trampoline.Done{
		Result: controlBreak{},
	}
}

func (interpreter *Interpreter) VisitContinueStatement(_ *ast.ContinueStatement) ast.Repr {
	return trampoline.Done{
		Result: controlContinue{},
	}
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
) trampoline.Trampoline {

	value := interpreter.evalExpression(test).(BoolValue)
	if value {
		return thenBlock.Accept(interpreter).(trampoline.Trampoline)
	} else if elseBlock != nil {
		return elseBlock.Accept(interpreter).(trampoline.Trampoline)
	}

	// NOTE: no result, so it does *not* act like a return-statement
	return trampoline.Done{}
}

func (interpreter *Interpreter) visitIfStatementWithVariableDeclaration(
	declaration *ast.VariableDeclaration,
	thenBlock, elseBlock *ast.Block,
) trampoline.Trampoline {

	result := interpreter.evalExpression(declaration.Value)

	if someValue, ok := result.(*SomeValue); ok {

		targetType := interpreter.Program.Elaboration.VariableDeclarationTargetTypes[declaration]
		valueType := interpreter.Program.Elaboration.VariableDeclarationValueTypes[declaration]
		unwrappedValueCopy := interpreter.copyAndConvert(someValue.Value, valueType, targetType)

		interpreter.activations.PushNewWithCurrent()
		interpreter.declareVariable(
			declaration.Identifier.Identifier,
			unwrappedValueCopy,
		)

		return thenBlock.Accept(interpreter).(trampoline.Trampoline).
			Then(func(_ interface{}) {
				interpreter.activations.Pop()
			})
	} else if elseBlock != nil {
		return elseBlock.Accept(interpreter).(trampoline.Trampoline)
	}

	// NOTE: ignore result, so it does *not* act like a return-statement
	return trampoline.Done{}
}

func (interpreter *Interpreter) VisitSwitchStatement(switchStatement *ast.SwitchStatement) ast.Repr {

	var visitCase func(i int, testValue EquatableValue) trampoline.Trampoline
	visitCase = func(i int, testValue EquatableValue) trampoline.Trampoline {

		// If no cases are left to evaluate, return (base case)

		if i >= len(switchStatement.Cases) {
			// NOTE: no result, so it does *not* act like a return-statement
			return trampoline.Done{}
		}

		switchCase := switchStatement.Cases[i]

		runStatements := func() trampoline.Trampoline {
			// NOTE: the new block ensures that a new block is introduced

			block := &ast.Block{
				Statements: switchCase.Statements,
			}

			return block.Accept(interpreter).(trampoline.Trampoline).
				FlatMap(func(value interface{}) trampoline.Trampoline {

					if _, ok := value.(controlBreak); ok {
						return trampoline.Done{}
					}

					return trampoline.Done{Result: value}
				})
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

		if testValue.Equal(interpreter, caseValue) {
			return runStatements()
		}

		// If the test value and the case values are unequal,
		// try the next case (recurse)

		return visitCase(i+1, testValue)
	}

	testValue := interpreter.evalExpression(switchStatement.Expression).(EquatableValue)
	return visitCase(0, testValue)
}

func (interpreter *Interpreter) VisitWhileStatement(statement *ast.WhileStatement) ast.Repr {

	value := interpreter.evalExpression(statement.Test).(BoolValue)
	if !value {
		return trampoline.Done{}
	}

	interpreter.reportLoopIteration(statement)

	return statement.Block.Accept(interpreter).(trampoline.Trampoline).
		FlatMap(func(value interface{}) trampoline.Trampoline {

			switch value.(type) {
			case controlBreak:
				return trampoline.Done{}

			case controlContinue:
				// NO-OP

			case functionReturn:
				return trampoline.Done{
					Result: value,
				}
			}

			// recurse
			return statement.Accept(interpreter).(trampoline.Trampoline)
		})
}

func (interpreter *Interpreter) VisitForStatement(statement *ast.ForStatement) ast.Repr {
	interpreter.activations.PushNewWithCurrent()

	variable := interpreter.declareVariable(
		statement.Identifier.Identifier,
		nil,
	)

	var loop func(i, count int, values []Value) trampoline.Trampoline
	loop = func(i, count int, values []Value) trampoline.Trampoline {

		if i == count {
			return trampoline.Done{}
		}

		interpreter.reportLoopIteration(statement)

		variable.Value = values[i]

		return statement.Block.Accept(interpreter).(trampoline.Trampoline).
			FlatMap(func(value interface{}) trampoline.Trampoline {

				switch value.(type) {
				case controlBreak:
					return trampoline.Done{}

				case controlContinue:
					// NO-OP

				case functionReturn:
					return trampoline.Done{
						Result: value,
					}
				}

				// recurse
				if i == count {
					return trampoline.Done{}
				}
				return loop(i+1, count, values)
			})
	}

	values := interpreter.evalExpression(statement.Value).(*ArrayValue).Values[:]
	count := len(values)

	return loop(0, count, values).
		Then(func(_ interface{}) {
			interpreter.activations.Pop()
		})
}

func (interpreter *Interpreter) VisitEmitStatement(statement *ast.EmitStatement) ast.Repr {
	event := interpreter.evalExpression(statement.InvocationExpression).(*CompositeValue)

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
	return trampoline.Done{}
}

func (interpreter *Interpreter) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) ast.Repr {
	return trampoline.Done{}
}

// VisitVariableDeclaration first visits the declaration's value,
// then declares the variable with the name bound to the value
func (interpreter *Interpreter) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Repr {

	targetType := interpreter.Program.Elaboration.VariableDeclarationTargetTypes[declaration]
	valueType := interpreter.Program.Elaboration.VariableDeclarationValueTypes[declaration]
	secondValueType := interpreter.Program.Elaboration.VariableDeclarationSecondValueTypes[declaration]

	result := interpreter.visitPotentialStorageRemoval(declaration.Value)

	valueCopy := interpreter.copyAndConvert(result, valueType, targetType)

	interpreter.declareVariable(
		declaration.Identifier.Identifier,
		valueCopy,
	)

	if declaration.SecondValue == nil {
		// NOTE: ignore result, so it does *not* act like a return-statement
		return trampoline.Done{}
	}

	return interpreter.visitAssignment(
		declaration.Transfer.Operation,
		declaration.Value,
		valueType,
		declaration.SecondValue,
		secondValueType,
		declaration,
	)
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

func (interpreter *Interpreter) VisitSwapStatement(swap *ast.SwapStatement) ast.Repr {

	leftType := interpreter.Program.Elaboration.SwapStatementLeftTypes[swap]
	rightType := interpreter.Program.Elaboration.SwapStatementRightTypes[swap]

	// Evaluate the left expression
	leftGetterSetter := interpreter.assignmentGetterSetter(swap.Left)
	leftValue := leftGetterSetter.get()
	if interpreter.movingStorageIndexExpression(swap.Left) != nil {
		leftGetterSetter.set(NilValue{})
	}

	// Evaluate the right expression
	rightGetterSetter := interpreter.assignmentGetterSetter(swap.Right)
	rightValue := rightGetterSetter.get()
	if interpreter.movingStorageIndexExpression(swap.Right) != nil {
		rightGetterSetter.set(NilValue{})
	}

	// Add right value to left target
	// and left value to right target

	rightValueCopy := interpreter.copyAndConvert(rightValue, rightType, leftType)
	leftValueCopy := interpreter.copyAndConvert(leftValue, leftType, rightType)

	leftGetterSetter.set(rightValueCopy)
	rightGetterSetter.set(leftValueCopy)

	return trampoline.Done{}
}

func (interpreter *Interpreter) VisitExpressionStatement(statement *ast.ExpressionStatement) ast.Repr {
	result := interpreter.evalExpression(statement.Expression)
	return trampoline.Done{
		Result: ExpressionStatementResult{result},
	}
}
