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

package bbq

import "github.com/onflow/cadence/ast"

// LineNumberTable holds the instruction-index to source-position mapping.
// It only maintains an entry for an instruction only if the position info
// got changed during that instruction.
// i.e: If multiple consecutive instructions were emitted for the same source-code position,
// then only the first instruction of that instruction-set would be available in the table.
type LineNumberTable struct {
	positions []PositionInfo
}

func (t *LineNumberTable) AddPositionInfo(bytecodeIndex uint16, position ast.Position) {
	t.positions = append(
		t.positions,
		PositionInfo{
			instructionIndex: bytecodeIndex,
			position:         position,
		},
	)
}

func (t *LineNumberTable) GetSourcePosition(instructionIndex uint16) ast.Position {
	var lastChangedPosition ast.Position

	for _, positionInfo := range t.positions {
		if instructionIndex < positionInfo.instructionIndex {
			break
		}
		lastChangedPosition = positionInfo.position
	}

	return lastChangedPosition
}

type PositionInfo struct {
	instructionIndex uint16
	position         ast.Position
}
