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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
)

const FunctionEntryPointName = "main"

// FunctionEntryPointDeclaration returns the entry point function declaration, if any.
//
// Returns nil if there are multiple function declarations with the same function entry point name, or a transaction declaration.
//
func FunctionEntryPointDeclaration(program *ast.Program) *ast.FunctionDeclaration {

	functionDeclarations := program.FunctionDeclarations()

	if len(program.TransactionDeclarations()) > 0 {

		return nil
	}

	var entryPointFunctionDeclaration *ast.FunctionDeclaration

	for _, declaration := range functionDeclarations {

		if declaration.Identifier.Identifier != FunctionEntryPointName {
			continue
		}

		if entryPointFunctionDeclaration != nil {
			return nil
		}

		entryPointFunctionDeclaration = declaration
	}

	return entryPointFunctionDeclaration
}

// EntryPointParameters returns the parameters of the transaction or script, if any.
//
// Returns nil if the program specifies both a valid transaction and entry point function declaration.
//
func (checker *Checker) EntryPointParameters() []*Parameter {
	transactionDeclaration := checker.Program.SoleTransactionDeclaration()
	if transactionDeclaration != nil {
		transactionType := checker.Elaboration.TransactionDeclarationTypes[transactionDeclaration]
		return transactionType.Parameters
	}

	functionDeclaration := FunctionEntryPointDeclaration(checker.Program)
	if functionDeclaration != nil {
		functionType := checker.Elaboration.FunctionDeclarationFunctionTypes[functionDeclaration]
		return functionType.Parameters
	}

	return nil
}
