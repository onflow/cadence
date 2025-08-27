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

package ast

import (
	"fmt"

	"github.com/onflow/cadence/common"
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

func NewPositionAtCodeOffset(memoryGauge common.MemoryGauge, code string, offset int) Position {
	line := 1
	column := 0

	for i := 0; i < offset; i++ {
		if code[i] == '\n' {
			line++
			column = 0
		} else {
			column++
		}
	}

	return NewPosition(
		memoryGauge,
		offset,
		line,
		column,
	)
}

func (p Position) Shifted(memoryGauge common.MemoryGauge, length int) Position {
	return NewPosition(
		memoryGauge,
		p.Offset+length,
		p.Line,
		p.Column+length,
	)
}

// AttachLeft moves the position left until it reaches a non-whitespace character.
func (p Position) AttachLeft(code string) Position {

	newOffset := p.Offset - 1
	for ; newOffset >= 0; newOffset-- {
		switch code[newOffset] {
		case ' ', '\t', '\r', '\n':
			continue
		}
		break
	}

	newOffset++

	if newOffset == p.Offset {
		return p
	}

	return NewPositionAtCodeOffset(nil, code, newOffset)
}

func (p Position) SlurpWhitespaceSuffix(code string) Position {
	var length int
	for offset := p.Offset + 1; offset < len(code); offset++ {
		if code[offset] == ' ' {
			length++
		} else {
			break
		}
	}

	if length == 0 {
		return p
	}
	return p.Shifted(nil, length)
}

func (p Position) String() string {
	return fmt.Sprintf(
		"%d(%d:%d)",
		p.Offset,
		p.Line,
		p.Column,
	)
}

func (p Position) Compare(other Position) int {
	switch {
	case p.Offset < other.Offset:
		return -1
	case p.Offset > other.Offset:
		return 1
	default:
		return 0
	}
}

func EarlierPosition(p1, p2 *Position) *Position {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	if p1.Compare(*p2) < 0 {
		return p1
	}
	return p2
}

func EndPosition(memoryGauge common.MemoryGauge, startPosition Position, end int) Position {
	length := end - startPosition.Offset
	return startPosition.Shifted(memoryGauge, length)
}

func EarliestPosition(p Position, ps ...*Position) (earliest Position) {
	earliest = p
	for _, pos := range ps {
		if pos != nil && pos.Compare(earliest) < 0 {
			earliest = *pos
		}
	}
	return
}

// HasPosition

type HasPosition interface {
	StartPosition() Position
	EndPosition(memoryGauge common.MemoryGauge) Position
}

func RangeContains(memoryGauge common.MemoryGauge, a, b HasPosition) bool {
	return a.StartPosition().Compare(b.StartPosition()) <= 0 &&
		a.EndPosition(memoryGauge).Compare(b.EndPosition(memoryGauge)) >= 0
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

func (r Range) StartPosition() Position {
	return r.StartPos
}

func (r Range) EndPosition(common.MemoryGauge) Position {
	return r.EndPos
}

// NewRangeFromPositioned

func NewRangeFromPositioned(memoryGauge common.MemoryGauge, hasPosition HasPosition) Range {
	if hasPosition == nil {
		return EmptyRange
	}

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
