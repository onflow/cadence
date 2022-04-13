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

package ast

import (
	"fmt"

	"github.com/onflow/cadence/runtime/common"
)

var EmptyPosition = Position{}

// Position defines a row/column within a Cadence script.
type Position struct {
	// offset, starting at 0
	Offset int
	// line number, starting at 1
	Line int
	// column number, starting at 0 (byte count)
	Column int
}

func NewPosition(memoryGauge common.MemoryGauge, offset, line, column int) Position {
	common.UseMemory(memoryGauge, common.PositionMemoryUsage)
	return Position{
		Offset: offset,
		Line:   line,
		Column: column,
	}
}

func (position Position) Shifted(memoryGauge common.MemoryGauge, length int) Position {
	return NewPosition(
		memoryGauge,
		position.Offset+length,
		position.Line,
		position.Column+length,
	)
}

func (position Position) String() string {
	return fmt.Sprintf(
		"%d(%d:%d)",
		position.Offset,
		position.Line,
		position.Column,
	)
}

func (position Position) Compare(other Position) int {
	switch {
	case position.Offset < other.Offset:
		return -1
	case position.Offset > other.Offset:
		return 1
	default:
		return 0
	}
}

func EndPosition(memoryGauge common.MemoryGauge, startPosition Position, end int) Position {
	length := end - startPosition.Offset
	return startPosition.Shifted(memoryGauge, length)
}

// HasPosition

type HasPosition interface {
	StartPosition() Position
	EndPosition(memoryGauge common.MemoryGauge) Position
}

// Range

type Range struct {
	StartPos Position
	EndPos   Position
}

var EmptyRange = Range{}

func NewRange(memoryGauge common.MemoryGauge, startPos, endPos Position) Range {
	common.UseMemory(memoryGauge, common.RangeMemoryUsage)
	return NewUnmeteredRange(startPos, endPos)
}

func NewUnmeteredRange(startPos, endPos Position) Range {
	return Range{
		StartPos: startPos,
		EndPos:   endPos,
	}
}

func (e Range) StartPosition() Position {
	return e.StartPos
}

func (e Range) EndPosition(common.MemoryGauge) Position {
	return e.EndPos
}

// NewRangeFromPositioned

func NewRangeFromPositioned(memoryGauge common.MemoryGauge, hasPosition HasPosition) Range {
	return NewRange(
		memoryGauge,
		hasPosition.StartPosition(),
		hasPosition.EndPosition(memoryGauge),
	)
}

func NewUnmeteredRangeFromPositioned(hasPosition HasPosition) Range {
	return NewUnmeteredRange(
		hasPosition.StartPosition(),
		hasPosition.EndPosition(nil),
	)
}
