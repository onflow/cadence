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

package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

type testMemoryGauge struct {
	meter map[common.MemoryKind]uint64
}

var _ common.MemoryGauge = &testMemoryGauge{}

func newTestMemoryGauge() *testMemoryGauge {
	return &testMemoryGauge{
		meter: make(map[common.MemoryKind]uint64),
	}
}

func (g *testMemoryGauge) MeterMemory(usage common.MemoryUsage) error {
	if g.meter == nil {
		g.meter = make(map[common.MemoryKind]uint64)
	}
	g.meter[usage.Kind] += usage.Amount

	return nil
}

func (g *testMemoryGauge) getMemory(kind common.MemoryKind) int {
	return int(g.meter[kind])
}

func assertMemoryIntensitiesContains(t *testing.T, actual, contains map[common.MemoryKind]uint64) {
	for containKind, containAmount := range contains {
		// If the memory kind doesn't exist, it is same as zero-intensity.
		actualAmount, _ := actual[containKind]

		assert.Equal(
			t,
			containAmount,
			actualAmount,
			"memory metering mismatch for %s. expected %d, found %d",
			containKind,
			int(containAmount),
			int(actualAmount),
		)
	}
}

func parseCheckAndCompile(t *testing.T, code string, meter common.MemoryGauge) *bbq.InstructionProgram {
	checker, err := ParseAndCheck(t, code)
	require.NoError(t, err)

	comp := NewInstructionCompilerWithConfig(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&Config{
			MemoryGauge: meter,
		},
	)

	return comp.Compile()
}

func TestCompilerMemoryMetering(t *testing.T) {

	t.Parallel()

	t.Run("variables", func(t *testing.T) {
		t.Parallel()

		script := `
          var a = 5
          var b = 6
          var c = a + b
        `

		meter := newTestMemoryGauge()
		_ = parseCheckAndCompile(t, script, meter)

		expected := map[common.MemoryKind]uint64{
			common.MemoryKindCompiler:            1,
			common.MemoryKindCompilerGlobal:      3,
			common.MemoryKindCompilerLocal:       0,
			common.MemoryKindCompilerConstant:    2,
			common.MemoryKindCompilerFunction:    3,
			common.MemoryKindCompilerInstruction: 11,

			common.MemoryKindCompilerBBQProgram:  1,
			common.MemoryKindCompilerBBQConstant: 2,
			common.MemoryKindCompilerBBQFunction: 3,
			common.MemoryKindCompilerBBQImport:   0,
			common.MemoryKindCompilerBBQVariable: 3,
			common.MemoryKindCompilerBBQContract: 0,

			common.MemoryKindNumberValue: 0,
			common.MemoryKindBigInt:      16,

			common.MemoryKindGoSliceLength: 8,
		}

		assertMemoryIntensitiesContains(t, meter.meter, expected)
	})

	t.Run("empty functions", func(t *testing.T) {
		t.Parallel()

		script := `
          fun foo() {}

          fun bar() {}
        `

		meter := newTestMemoryGauge()
		_ = parseCheckAndCompile(t, script, meter)

		expected := map[common.MemoryKind]uint64{
			common.MemoryKindCompiler:            1,
			common.MemoryKindCompilerGlobal:      2,
			common.MemoryKindCompilerLocal:       0,
			common.MemoryKindCompilerConstant:    0,
			common.MemoryKindCompilerFunction:    2,
			common.MemoryKindCompilerInstruction: 2,

			common.MemoryKindCompilerBBQProgram:  1,
			common.MemoryKindCompilerBBQConstant: 0,
			common.MemoryKindCompilerBBQFunction: 2,
			common.MemoryKindCompilerBBQImport:   0,
			common.MemoryKindCompilerBBQVariable: 0,
			common.MemoryKindCompilerBBQContract: 0,

			common.MemoryKindNumberValue: 0,
			common.MemoryKindBigInt:      0,

			common.MemoryKindGoSliceLength: 4,
		}

		assertMemoryIntensitiesContains(t, meter.meter, expected)
	})

	t.Run("non-empty functions", func(t *testing.T) {
		t.Parallel()

		script := `
          fun foo(s: String) {
              let a = 1
              let b: UInt8 = 2
          }
        `

		meter := newTestMemoryGauge()
		_ = parseCheckAndCompile(t, script, meter)

		expected := map[common.MemoryKind]uint64{
			common.MemoryKindCompiler:            1,
			common.MemoryKindCompilerGlobal:      1,
			common.MemoryKindCompilerLocal:       3,
			common.MemoryKindCompilerConstant:    2,
			common.MemoryKindCompilerFunction:    1,
			common.MemoryKindCompilerInstruction: 9,

			common.MemoryKindCompilerBBQProgram:  1,
			common.MemoryKindCompilerBBQConstant: 2,
			common.MemoryKindCompilerBBQFunction: 1,
			common.MemoryKindCompilerBBQImport:   0,
			common.MemoryKindCompilerBBQVariable: 0,
			common.MemoryKindCompilerBBQContract: 0,

			common.MemoryKindNumberValue: 1,
			common.MemoryKindBigInt:      8,

			common.MemoryKindGoSliceLength: 4,
		}

		assertMemoryIntensitiesContains(t, meter.meter, expected)
	})

	t.Run("contract", func(t *testing.T) {
		t.Parallel()

		script := `
          contract Foo{}
        `

		meter := newTestMemoryGauge()
		_ = parseCheckAndCompile(t, script, meter)

		expected := map[common.MemoryKind]uint64{
			common.MemoryKindCompiler:            1,
			common.MemoryKindCompilerGlobal:      4,
			common.MemoryKindCompilerLocal:       1,
			common.MemoryKindCompilerConstant:    0,
			common.MemoryKindCompilerFunction:    3,
			common.MemoryKindCompilerInstruction: 6,

			common.MemoryKindCompilerBBQProgram:  1,
			common.MemoryKindCompilerBBQConstant: 0,
			common.MemoryKindCompilerBBQFunction: 3,
			common.MemoryKindCompilerBBQImport:   0,
			common.MemoryKindCompilerBBQVariable: 0,
			common.MemoryKindCompilerBBQContract: 1,

			common.MemoryKindNumberValue: 0,
			common.MemoryKindBigInt:      0,

			common.MemoryKindGoSliceLength: 7,
		}

		assertMemoryIntensitiesContains(t, meter.meter, expected)
	})

	t.Run("transaction", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction {

              prepare(signer: &Account) {}

              pre {
                  true
              }

              post {
                  false
              }

              execute {}
          }
        `

		meter := newTestMemoryGauge()
		_ = parseCheckAndCompile(t, script, meter)

		expected := map[common.MemoryKind]uint64{
			common.MemoryKindCompiler:            1,
			common.MemoryKindCompilerGlobal:      8,
			common.MemoryKindCompilerLocal:       4,
			common.MemoryKindCompilerConstant:    1,
			common.MemoryKindCompilerFunction:    6,
			common.MemoryKindCompilerInstruction: 26,

			common.MemoryKindCompilerBBQProgram:  1,
			common.MemoryKindCompilerBBQConstant: 1,
			common.MemoryKindCompilerBBQFunction: 6,
			common.MemoryKindCompilerBBQImport:   2,
			common.MemoryKindCompilerBBQVariable: 0,
			common.MemoryKindCompilerBBQContract: 0,

			common.MemoryKindNumberValue: 0,
			common.MemoryKindBigInt:      0,

			common.MemoryKindGoSliceLength: 17,
		}

		assertMemoryIntensitiesContains(t, meter.meter, expected)
	})

}
