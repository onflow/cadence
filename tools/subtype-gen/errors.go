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
	"strings"

	yaml "github.com/goccy/go-yaml/ast"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/pretty"
)

type SubTypeGenError struct {
	Code []byte
	Err  error
}

var _ error = &SubTypeGenError{}

func (e *SubTypeGenError) Error() string {
	var sb strings.Builder
	sb.WriteString("Parsing failed:\n")
	printErr := pretty.NewErrorPrettyPrinter(&sb, false).
		PrettyPrintError(e.Err, nil, map[common.Location][]byte{nil: e.Code})
	if printErr != nil {
		panic(printErr)
	}
	sb.WriteString(errors.ErrorPrompt)
	return sb.String()
}

type ParsingError struct {
	Message string
	Path    string
	ast.Range
}

var _ error = ParsingError{}
var _ ast.HasPosition = ParsingError{}

func NewParsingError(message string, node yaml.Node) ParsingError {
	yamlPosition := node.GetToken().Position

	// yaml parser doesn't store end position.
	// So use the start position for both, for now.
	position := ast.Position{
		Offset: yamlPosition.Offset,
		Line:   yamlPosition.Line,
		Column: yamlPosition.Column,
	}

	return ParsingError{
		Message: message,
		Path:    node.GetPath(),
		Range:   ast.NewUnmeteredRange(position, position),
	}
}

func (p ParsingError) Error() string {
	return p.Message
}
