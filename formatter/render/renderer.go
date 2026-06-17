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

package render

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/formatter/trivia"
)

// renderer holds state shared across rendering of a single program: the
// CommentMap (drained as comments are emitted), the original source bytes
// (used for blank-line detection that can't rely on AST line numbers), and
// the optional explicit-semicolon set (populated only when StripSemicolons
// is false). All render functions are methods on *renderer so this state
// doesn't need to be threaded through every call.
type renderer struct {
	cm         *trivia.CommentMap
	source     []byte
	semicolons map[ast.Element]bool
}

// hasSemicolon reports whether elem had a trailing semicolon in the source.
// Returns false when semicolons is nil (StripSemicolons mode).
func (r *renderer) hasSemicolon(elem ast.Element) bool {
	return r.semicolons[elem]
}
