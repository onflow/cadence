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

package lexer

import (
	"github.com/onflow/cadence/ast"
)

type Token struct {
	SpaceOrError any
	Type         TokenType
	ast.Range
	// TODO(preserve-comments): This is currently not true (first comment),
	// 	as leading comments are just all comments after the trailing comments of the previous token.
	// Leading comments span up to and including the first contiguous sequence of newlines characters.
	// Trailing comments span up to, but not including, the next newline character.
	// Not tracked for space token, since those are usually ignored in the parser.
	ast.Comments
}

func (t Token) Is(ty TokenType) bool {
	return t.Type == ty
}

func (t Token) Source(input []byte) []byte {
	return t.Range.Source(input)
}

func (t Token) Equal(other Token) bool {
	// ignore comments, since they should not be treated as source code
	return t.Type == other.Type && t.Range == other.Range && t.SpaceOrError == other.SpaceOrError
}
