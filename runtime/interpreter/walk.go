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

type ValueWalker interface {
	WalkValue(interpreter *Interpreter, value Value) ValueWalker
}

// WalkValue traverses a Value object graph in depth-first order:
// It starts by calling valueWalker.WalkValue(value);
// If the returned walker is nil,
// child values are not walked.
// If the returned walker is not-nil,
// then WalkValue is invoked recursively on this returned walker
// for each of the non-nil children of the value,
// followed by a call of WalkValue(nil) on the returned walker.
//
// The initial walker may not be nil.
func WalkValue(interpreter *Interpreter, walker ValueWalker, value Value) {
	if walker = walker.WalkValue(interpreter, value); walker == nil {
		return
	}

	value.Walk(interpreter, func(child Value) {
		WalkValue(interpreter, walker, child)
	})

	walker.WalkValue(interpreter, nil)
}
