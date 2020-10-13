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
	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

// diagnostics gets extra non-error diagnostics based on a checker.
//
// For example, this function will return diagnostics for declarations that are
// syntactically and semantically valid, but unsupported by the extension.
//
func (i *FlowIntegration) diagnostics(
	_ protocol.DocumentUri,
	checker *sema.Checker,
) (
	diagnostics []protocol.Diagnostic,
	err error,
) {

	// Warn if there are more than 1 transaction declarations as deployment will fail
	transactionDeclarations := checker.Program.TransactionDeclarations()

	if len(transactionDeclarations) > 1 {

		isFirst := true

		for _, declaration := range transactionDeclarations[1:] {

			position := declaration.StartPosition()

			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: conversion.ASTToProtocolRange(
					position,
					position,
				),
				Severity: protocol.SeverityWarning,
				Message:  "Cannot declare more than one transaction per file",
			})
		}
	}

	// Warn if there are more than one composite or interface declaration,
	// as deployment will fail

	var compositeAndInterfaceDeclarations []ast.Declaration

	for _, compositeDeclaration := range checker.Program.CompositeDeclarations() {
		compositeAndInterfaceDeclarations = append(compositeAndInterfaceDeclarations, compositeDeclaration)
	}

	for _, interfaceDeclaration := range checker.Program.InterfaceDeclarations() {
		compositeAndInterfaceDeclarations = append(compositeAndInterfaceDeclarations, interfaceDeclaration)
	}

	if len(compositeAndInterfaceDeclarations) > 1 {


		for _, declaration := range compositeAndInterfaceDeclarations {

			// Skip the first declaration
			if isFirst {
				isFirst = false
				continue
			}

			position := declaration.DeclarationIdentifier().StartPosition()

			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: conversion.ASTToProtocolRange(
					position,
					position,
				),
				Severity: protocol.SeverityWarning,
				Message:  "Cannot declare more than one top-level type per file",
			})
		}
	}

	return
}
