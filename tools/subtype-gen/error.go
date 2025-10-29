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

package subtype_gen

import (
	"fmt"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

type ParsingError struct {
	Message  string
	Path     string
	Position *token.Position
}

var _ error = ParsingError{}

func NewParsingError(message string, node ast.Node) ParsingError {
	return ParsingError{
		Message:  message,
		Path:     node.GetPath(),
		Position: node.GetToken().Position,
	}
}

func (p ParsingError) Error() string {
	return fmt.Sprintf("%s: %s", p.Path, p.Message)
}
