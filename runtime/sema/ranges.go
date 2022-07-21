/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/intervalst"
)

type Range struct {
	Identifier      string
	Type            Type
	DeclarationKind common.DeclarationKind
	DocString       string
}

type Ranges struct {
	tree *intervalst.IntervalST[Range]
}

func NewRanges() *Ranges {
	return &Ranges{
		tree: &intervalst.IntervalST[Range]{},
	}
}

func (r *Ranges) Put(startPos, endPos ast.Position, ra Range) {
	interval := intervalst.NewInterval(
		ASTToSemaPosition(startPos),
		ASTToSemaPosition(endPos),
	)
	r.tree.Put(interval, ra)
}

func (r *Ranges) All() []Range {
	return r.tree.Values()
}

func (r *Ranges) FindAll(pos Position) []Range {
	entries := r.tree.SearchAll(pos)
	ranges := make([]Range, len(entries))
	for i, entry := range entries {
		ranges[i] = entry.Value
	}
	return ranges
}
