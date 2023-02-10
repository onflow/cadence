/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package ast

type Walker interface {
	Walk(element Element) Walker
}

// Walk traverses an AST in depth-first order:
// It starts by calling walker.Walk(element);
// If the returned walker is nil,
// child elements are not walked.
// If the returned walker is not-nil,
// then Walk is invoked recursively on this returned walker
// for each of the non-nil children of the element,
// followed by a call of Walk(nil) on the returned walker.
//
// The initial walker may not be nil.
func Walk(walker Walker, element Element) {
	if walker = walker.Walk(element); walker == nil {
		return
	}

	element.Walk(func(child Element) {
		Walk(walker, child)
	})

	walker.Walk(nil)
}

func walkExpressions(walkChild func(Element), expressions []Expression) {
	for _, expression := range expressions {
		walkChild(expression)
	}
}

func walkStatements(walkChild func(Element), statements []Statement) {
	for _, statement := range statements {
		walkChild(statement)
	}
}

func walkDeclarations(walkChild func(Element), declarations []Declaration) {
	for _, declaration := range declarations {
		walkChild(declaration)
	}
}
