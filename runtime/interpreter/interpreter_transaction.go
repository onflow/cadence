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
	"github.com/onflow/cadence/runtime/sema"
)

func (interpreter *Interpreter) VisitTransactionDeclaration(declaration *ast.TransactionDeclaration) ast.Repr {
	interpreter.declareTransactionEntryPoint(declaration)

	return nil
}

func (interpreter *Interpreter) declareTransactionEntryPoint(declaration *ast.TransactionDeclaration) {
	transactionType := interpreter.Program.Elaboration.TransactionDeclarationTypes[declaration]

	lexicalScope := interpreter.activations.CurrentOrNew()

	var prepareFunction *ast.FunctionDeclaration
	var prepareFunctionType *sema.FunctionType
	if declaration.Prepare != nil {
		prepareFunction = declaration.Prepare.FunctionDeclaration
		prepareFunctionType = transactionType.PrepareFunctionType()
	}

	var executeFunction *ast.FunctionDeclaration
	var executeFunctionType *sema.FunctionType
	if declaration.Execute != nil {
		executeFunction = declaration.Execute.FunctionDeclaration
		executeFunctionType = transactionType.ExecuteFunctionType()
	}

	postConditionsRewrite :=
		interpreter.Program.Elaboration.PostConditionsRewrite[declaration.PostConditions]

	staticType := NewCompositeStaticType(interpreter.Location, "")

	self := NewSimpleCompositeValue(
		staticType.TypeID,
		staticType,
		nil,
		nil,
		map[string]Value{},
		nil,
		nil,
		nil,
	)

	transactionFunction := NewHostFunctionValue(
		func(invocation Invocation) Value {
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

			if prepareFunction != nil {
				prepare := interpreter.functionDeclarationValue(
					prepareFunction,
					prepareFunctionType,
					transactionScope,
				)

				prepare.invoke(invocation)
			}

			var body func() controlReturn
			if executeFunction != nil {
				execute := interpreter.functionDeclarationValue(
					executeFunction,
					executeFunctionType,
					transactionScope,
				)

				invocationWithoutArguments := invocation
				invocationWithoutArguments.Arguments = nil

				body = func() controlReturn {
					value := execute.invoke(invocationWithoutArguments)
					return functionReturn{
						Value: value,
					}
				}
			}

			var preConditions ast.Conditions
			if declaration.PreConditions != nil {
				preConditions = *declaration.PreConditions
			}

			return interpreter.visitFunctionBody(
				postConditionsRewrite.BeforeStatements,
				preConditions,
				body,
				postConditionsRewrite.RewrittenPostConditions,
				sema.VoidType,
			)
		},

		// This is an internally used function.
		// So ideally wouldn't need to perform type checks.
		nil,
	)

	interpreter.Transactions = append(
		interpreter.Transactions,
		transactionFunction,
	)
}
