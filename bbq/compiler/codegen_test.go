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
	"github.com/onflow/cadence/common"
)

func TestByteCodeGen(t *testing.T) {
	t.Parallel()

	codegen := &compiler.ByteCodeGen{}

	var target []byte

	codegen.SetTarget(&target)
	require.Equal(t, 0, codegen.Offset())
	require.Nil(t, codegen.LastInstruction())

	codegen.Emit(opcode.InstructionNil{})
	require.Equal(t, 1, codegen.Offset())
	require.Equal(t,
		opcode.InstructionNil{},
		codegen.LastInstruction(),
	)

	codegen.Emit(opcode.InstructionNewPath{
		Domain:     common.PathDomainStorage,
		Identifier: 2,
	})
	require.Equal(t, 5, codegen.Offset())
	require.Equal(t,
		opcode.InstructionNewPath{
			Domain:     common.PathDomainStorage,
			Identifier: 2,
		},
		codegen.LastInstruction(),
	)

	require.Equal(t,
		[]byte{
			byte(opcode.Nil),
			byte(opcode.NewPath), 0x01, 0x00, 0x02,
		},
		target,
	)
}

func TestInstructionCodeGen(t *testing.T) {
	t.Parallel()

	codegen := &compiler.InstructionCodeGen{}

	var target []opcode.Instruction

	codegen.SetTarget(&target)
	require.Equal(t, 0, codegen.Offset())
	require.Nil(t, codegen.LastInstruction())

	codegen.Emit(opcode.InstructionNil{})
	require.Equal(t, 1, codegen.Offset())
	require.Equal(t,
		opcode.InstructionNil{},
		codegen.LastInstruction(),
	)

	codegen.Emit(opcode.InstructionNewPath{
		Domain:     common.PathDomainStorage,
		Identifier: 2,
	})
	require.Equal(t, 2, codegen.Offset())
	require.Equal(t,
		opcode.InstructionNewPath{
			Domain:     common.PathDomainStorage,
			Identifier: 2,
		},
		codegen.LastInstruction(),
	)

	require.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionNil{},
			opcode.InstructionNewPath{
				Domain:     common.PathDomainStorage,
				Identifier: 2,
			},
		},
		target,
	)
}
