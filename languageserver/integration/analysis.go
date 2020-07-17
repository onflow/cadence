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

package integration

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// getTransactionDeclarations finds all transaction declarations.
//
func getTransactionDeclarations(
	txDeclarationMap map[*ast.TransactionDeclaration]*sema.TransactionType,
) (
	txDeclarations []*ast.TransactionDeclaration,
) {
	for decl := range txDeclarationMap {
		txDeclarations = append(txDeclarations, decl)
	}
	return
}

// getContractInterfaceDeclarations finds all interface declarations for contracts.
//
func getContractInterfaceDeclarations(
	interfaceDeclarationMap map[*ast.InterfaceDeclaration]*sema.InterfaceType,
) (
	contractInterfaceDeclarations []*ast.InterfaceDeclaration,
) {
	for decl := range interfaceDeclarationMap {
		if decl.CompositeKind == common.CompositeKindContract {
			contractInterfaceDeclarations = append(contractInterfaceDeclarations, decl)
		}
	}

	return
}

// getScriptDeclarations finds function declarations that are interpreted as scripts.
//
func getScriptDeclarations(
	funcDeclarationMap map[*ast.FunctionDeclaration]*sema.FunctionType,
) (
	scriptDeclarations []*ast.FunctionDeclaration,
) {
	for decl := range funcDeclarationMap {
		if decl.Identifier.String() == "main" && len(decl.ParameterList.Parameters) == 0 {
			scriptDeclarations = append(scriptDeclarations, decl)
		}
	}

	return
}

// getContractDeclarations returns a list of contract declarations based on
// the keys of the input map.
//
// Usage: `getContractDeclarations(checker.Elaboration.CompositeDeclarations)`
//
func getContractDeclarations(
	compositeDeclarations map[*ast.CompositeDeclaration]*sema.CompositeType,
) (
	contractDeclarations []*ast.CompositeDeclaration,
) {
	for decl := range compositeDeclarations {
		if decl.CompositeKind == common.CompositeKindContract {
			contractDeclarations = append(contractDeclarations, decl)
		}
	}

	return
}
