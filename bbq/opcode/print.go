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

package opcode

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/interpreter"
)

func PrintBytecode(
	builder *strings.Builder,
	code []byte,
	resolve bool,
	constants []constant.Constant,
	types [][]byte,
	functionNames []string,
) error {
	instructions := DecodeInstructions(code)
	staticTypes := DecodeStaticTypes(types)
	return PrintInstructions(
		builder,
		instructions,
		resolve,
		constants,
		staticTypes,
		functionNames,
	)
}

func DecodeStaticTypes(types [][]byte) []interpreter.StaticType {
	var staticTypes []interpreter.StaticType
	if len(types) > 0 {
		staticTypes = make([]interpreter.StaticType, len(types))
		for i, typ := range types {
			staticType, err := interpreter.StaticTypeFromBytes(typ)
			if err != nil {
				panic(fmt.Sprintf("failed to decode static type: %v", err))
			}
			staticTypes[i] = staticType
		}
	}
	return staticTypes
}

func PrintInstructions(
	builder *strings.Builder,
	instructions []Instruction,
	resolve bool,
	constants []constant.Constant,
	types []interpreter.StaticType,
	functionNames []string,
) error {

	tabWriter := tabwriter.NewWriter(builder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for offset, instruction := range instructions {

		var operandsBuilder strings.Builder
		if resolve {
			instruction.ResolvedOperandsString(&operandsBuilder, constants, types, functionNames)
		} else {
			instruction.OperandsString(&operandsBuilder)
		}

		_, _ = fmt.Fprintf(
			tabWriter,
			"%d |\t%s |\t%s\n",
			offset,
			instruction.Opcode(),
			operandsBuilder.String(),
		)
	}

	_ = tabWriter.Flush()
	_, _ = fmt.Fprintln(builder)

	return nil
}
