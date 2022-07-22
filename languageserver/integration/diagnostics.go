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

package integration

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
)

// diagnostics gets extra non-error diagnostics based on a checker.
//
// For example, this function will return diagnostics for declarations that are
// syntactically and semantically valid, but unsupported by the extension.
//
func (i *FlowIntegration) diagnostics(
	_ protocol.DocumentURI,
	_ int32,
	checker *sema.Checker,
) (
	diagnostics []protocol.Diagnostic,
	err error,
) {
	diagnostics = append(diagnostics, i.transactionDeclarationCountDiagnostics(checker)...)
	diagnostics = append(diagnostics, i.compositeOrInterfaceDeclarationCountDiagnostics(checker)...)

	return
}

// transactionDeclarationCountDiagnostics reports diagnostics
// if there are more than 1 transaction declarations, as deployment will fail
//
func (i *FlowIntegration) transactionDeclarationCountDiagnostics(checker *sema.Checker) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	transactionDeclarations := checker.Program.TransactionDeclarations()

	if len(transactionDeclarations) > 1 {

		for _, declaration := range transactionDeclarations[1:] {

			position := declaration.StartPosition()

			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: conversion.ASTToProtocolRange(
					position,
					position.Shifted(nil, len(parser.KeywordTransaction)-1),
				),
				Severity: protocol.SeverityWarning,
				Message:  "Cannot declare more than one transaction per file",
			})
		}
	}

	return diagnostics
}

// compositeOrInterfaceDeclarationCountDiagnostics reports diagnostics
// if there are more than one composite or interface declaration, as deployment will fail
//
func (i *FlowIntegration) compositeOrInterfaceDeclarationCountDiagnostics(checker *sema.Checker) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	var compositeAndInterfaceDeclarations []ast.Declaration

	for _, compositeDeclaration := range checker.Program.CompositeDeclarations() {
		compositeAndInterfaceDeclarations = append(compositeAndInterfaceDeclarations, compositeDeclaration)
	}

	for _, interfaceDeclaration := range checker.Program.InterfaceDeclarations() {
		compositeAndInterfaceDeclarations = append(compositeAndInterfaceDeclarations, interfaceDeclaration)
	}

	if len(compositeAndInterfaceDeclarations) > 1 {

		for _, declaration := range compositeAndInterfaceDeclarations[1:] {

			identifier := declaration.DeclarationIdentifier()

			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: conversion.ASTToProtocolRange(
					identifier.StartPosition(),
					identifier.EndPosition(nil),
				),
				Severity: protocol.SeverityWarning,
				Message:  "Cannot declare more than one top-level type per file",
			})
		}
	}

	return diagnostics
}
