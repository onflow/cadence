/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// LocationPosition defines a position in the source of the import tree.
// The Location defines the script within the import tree, the Position
// defines the row/colum within the source of that script.
type LocationPosition struct {
	Location common.Location
	Position ast.Position
}

// LocationRange defines a range in the source of the import tree.
// The Position defines the script within the import tree, the Range
// defines the start/end position within the source of that script.
type LocationRange struct {
	Location common.Location
	ast.Range
}

func (r LocationRange) ImportLocation() common.Location {
	return r.Location
}
