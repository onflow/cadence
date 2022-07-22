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
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/intervalst"
)

type Position struct {
	// line number, starting at 1
	Line int
	// column number, starting at 0 (byte count)
	Column int
}

func (pos Position) String() string {
	return fmt.Sprintf("Position{%d, %d}", pos.Line, pos.Column)
}

func (pos Position) Compare(other intervalst.Position) int {
	if _, ok := other.(intervalst.MinPosition); ok {
		return 1
	}

	otherPos, ok := other.(Position)
	if !ok {
		panic(fmt.Sprintf("not a sema.Position: %#+v", other))
	}
	if pos.Line < otherPos.Line {
		return -1
	}
	if pos.Line > otherPos.Line {
		return 1
	}
	if pos.Column < otherPos.Column {
		return -1
	}
	if pos.Column > otherPos.Column {
		return 1
	}
	return 0
}

type Origin struct {
	Type            Type
	DeclarationKind common.DeclarationKind
	StartPos        *ast.Position
	EndPos          *ast.Position
	Occurrences     []ast.Range
	DocString       string
}

type Occurrences struct {
	tree *intervalst.IntervalST[Occurrence]
}

func NewOccurrences() *Occurrences {
	return &Occurrences{
		tree: &intervalst.IntervalST[Occurrence]{},
	}
}

func ASTToSemaPosition(position ast.Position) Position {
	return Position{
		Line:   position.Line,
		Column: position.Column,
	}
}

func (o *Occurrences) Put(startPos, endPos ast.Position, origin *Origin) {
	occurrence := Occurrence{
		StartPos: ASTToSemaPosition(startPos),
		EndPos:   ASTToSemaPosition(endPos),
		Origin:   origin,
	}
	interval := intervalst.NewInterval(
		occurrence.StartPos,
		occurrence.EndPos,
	)
	o.tree.Put(interval, occurrence)
	if origin != nil {
		origin.Occurrences = append(
			origin.Occurrences,
			ast.NewUnmeteredRange(
				startPos,
				endPos,
			),
		)
	}
}

type Occurrence struct {
	StartPos Position
	EndPos   Position
	Origin   *Origin
}

func (o *Occurrences) All() []Occurrence {
	return o.tree.Values()
}

func (o *Occurrences) Find(pos Position) *Occurrence {
	_, occurrence, present := o.tree.Search(pos)
	if !present {
		return nil
	}
	return &occurrence
}

func (o *Occurrences) FindAll(pos Position) []Occurrence {
	entries := o.tree.SearchAll(pos)
	occurrences := make([]Occurrence, len(entries))
	for i, entry := range entries {
		occurrences[i] = entry.Value
	}
	return occurrences
}
