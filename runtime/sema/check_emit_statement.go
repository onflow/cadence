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
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitEmitStatement(statement *ast.EmitStatement) (_ struct{}) {
	invocation := statement.InvocationExpression

	ty := checker.checkInvocationExpression(invocation)

	if ty.IsInvalidType() {
		return
	}

	// Check that emitted expression is an event

	compositeType, isCompositeType := ty.(*CompositeType)
	if !isCompositeType || compositeType.Kind != common.CompositeKindEvent {
		checker.report(
			&EmitNonEventError{
				Type:  ty,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, statement.InvocationExpression),
			},
		)
		return
	}

	checker.Elaboration.EmitStatementEventTypes[statement] = compositeType

	// Check that the emitted event is declared in the same location

	if compositeType.Location != checker.Location {

		checker.report(
			&EmitImportedEventError{
				Type:  ty,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, statement.InvocationExpression),
			},
		)
	}

	// emitting an event is an impure operation, but it is redundant to enforce that here:
	// because the emit statement can only be used on events (as a result of the above checks),
	// and because an event can only be invoked in an emit statement, every emit corresponds
	// 1:1 to an event invocation, which is an impure function call. Thus we already report
	// emits as impure by virtue of the necessity to create the event being emitted
	return
}
