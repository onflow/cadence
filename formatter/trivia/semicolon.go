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

package trivia

import "github.com/onflow/cadence/ast"

// ScanSemicolons walks the AST and checks the original source bytes after
// each statement/declaration's end position for a trailing semicolon.
// Returns a set of elements that had trailing semicolons in the source.
func ScanSemicolons(source []byte, prog *ast.Program) map[ast.Element]bool {
	result := make(map[ast.Element]bool)
	for _, decl := range prog.Declarations() {
		checkSemicolon(source, decl, result)
		decl.Walk(func(child ast.Element) {
			if child != nil {
				checkSemicolon(source, child, result)
			}
		})
	}
	return result
}

func checkSemicolon(source []byte, elem ast.Element, result map[ast.Element]bool) {
	end := elem.EndPosition(nil)
	if end.Offset < 0 || end.Offset >= len(source) {
		return
	}
	// Scan forward from end position, skipping spaces/tabs (not newlines).
	i := end.Offset + 1
	for i < len(source) && (source[i] == ' ' || source[i] == '\t') {
		i++
	}
	if i < len(source) && source[i] == ';' {
		result[elem] = true
	}
}
