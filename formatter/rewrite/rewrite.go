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

package rewrite

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/formatter/trivia"
)

// Rewriter transforms an AST program in place. Rewriters run in a fixed
// order; changing the order may break idempotence.
type Rewriter interface {
	Name() string
	Rewrite(prog *ast.Program, cm *trivia.CommentMap) error
}

// Apply runs all rewriters in the canonical fixed order.
// If you change the pass order or add/remove passes,
// bump format.CurrentFormatVersion in options.go.
func Apply(prog *ast.Program, cm *trivia.CommentMap, sortImports bool) error {
	var rewriters []Rewriter
	if sortImports {
		rewriters = append(rewriters, &importsSorter{})
	}
	// modifiers: canonical ordering is enforced by the parser, so no rewrite needed
	// parens: conservative removal deferred to later phase
	for _, rw := range rewriters {
		if err := rw.Rewrite(prog, cm); err != nil {
			return err
		}
	}
	return nil
}
