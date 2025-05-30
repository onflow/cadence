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

package interpreter

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

// LocationRange defines a range in the source of the import tree.
// The Position defines the script within the import tree, the Range
// defines the start/end position within the source of that script.
type LocationRange struct {
	Location common.Location
	ast.HasPosition
}

var _ ast.HasPosition = LocationRange{}

func (r LocationRange) StartPosition() ast.Position {
	if r.HasPosition == nil {
		return ast.EmptyPosition
	}
	return r.HasPosition.StartPosition()
}

func (r LocationRange) EndPosition(memoryGauge common.MemoryGauge) ast.Position {
	if r.HasPosition == nil {
		return ast.EmptyPosition
	}
	return r.HasPosition.EndPosition(memoryGauge)
}

func (r LocationRange) ImportLocation() common.Location {
	return r.Location
}

var EmptyLocationRange = LocationRange{}

func ReturnEmptyRange() ast.Range {
	return ast.EmptyRange
}
