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

package analysis

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

type SuggestedFix = errors.SuggestedFix[ast.TextEdit]

type TextEdit = ast.TextEdit

type Diagnostic struct {
	Location         common.Location
	Category         string
	Message          string
	SecondaryMessage string
	Code             string // Diagnostic code (e.g. "unused-variable")
	URL              string // URL to documentation
	SuggestedFixes   []SuggestedFix
	ast.Range
}

func (d Diagnostic) SuggestFixes(_ string) []SuggestedFix {
	return d.SuggestedFixes
}
