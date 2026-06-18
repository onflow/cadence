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

package compiler_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

// TestPeepholePatternAtEndOfCode ensures the peephole optimizer does not panic
// (with a slice-out-of-range error) when an instruction sequence ends with an
// instruction that is the first opcode of a multi-instruction pattern, and there
// are not enough remaining instructions for the full pattern to match.
//
// The first opcodes of the existing patterns are: GetLocal, GetConstant, Nil, NewPath.
func TestPeepholePatternAtEndOfCode(t *testing.T) {
	t.Parallel()

	checker, err := ParseAndCheck(t, `fun test() {}`)
	require.NoError(t, err)

	newOptimizer := func() compiler.PeepholeOptimizer[opcode.Instruction] {
		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.PeepholeOptimizationsEnabled = true
		_ = comp.Compile()
		return compiler.NewPeepholeOptimizer(comp)
	}

	test := func(t *testing.T, code []opcode.Instruction) {
		// Should not panic, and a too-short window must be left unchanged.
		result := newOptimizer().Optimize(code)
		require.Equal(t, code, result)
	}

	t.Run("GetLocal at end", func(t *testing.T) {
		t.Parallel()
		test(t, []opcode.Instruction{
			opcode.InstructionGetLocal{Local: 0},
		})
	})

	t.Run("GetConstant at end", func(t *testing.T) {
		t.Parallel()
		test(t, []opcode.Instruction{
			opcode.InstructionGetConstant{Constant: 0},
		})
	})

	t.Run("Nil at end", func(t *testing.T) {
		t.Parallel()
		test(t, []opcode.Instruction{
			opcode.InstructionNil{},
		})
	})

	t.Run("partial pattern after another instruction", func(t *testing.T) {
		t.Parallel()
		// GetLocal is the start of the (GetLocal, GetField) pattern,
		// but there is no following instruction to complete the pattern.
		test(t, []opcode.Instruction{
			opcode.InstructionTrue{},
			opcode.InstructionGetLocal{Local: 0},
		})
	})
}
