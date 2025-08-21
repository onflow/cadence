/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

// Package ast contains all AST nodes for Cadence.
// All AST nodes implement the Element interface,
// so have position information
// and can be traversed using the Visitor interface.
// Elements also implement the json.Marshaler interface
// so can be serialized to a standardized/stable JSON format.
package ast

import "github.com/onflow/cadence/errors"

type TextEdit struct {
	Replacement string
	Insertion   string
	Range
}

func (edit TextEdit) ApplyTo(code string) string {
	runes := []rune(code)
	start := edit.Range.StartPos.Offset
	end := edit.Range.EndPos.Offset

	if edit.Insertion != "" {
		if edit.Replacement != "" {
			panic(errors.NewUnexpectedError("TextEdit with Insertion should not have a Replacement"))
		}
		if start != end {
			panic(errors.NewUnexpectedError("TextEdit with Insertion should have a zero-length range"))
		}

		return string(runes[:start]) + edit.Insertion + string(runes[end:])
	}

	return string(runes[:start]) + edit.Replacement + string(runes[end+1:])
}
