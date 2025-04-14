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
)

func PrintBytecode(builder *strings.Builder, code []byte) error {
	instructions := DecodeInstructions(code)
	return PrintInstructions(builder, instructions)
}

func PrintInstructions(builder *strings.Builder, instructions []Instruction) error {

	tabWriter := tabwriter.NewWriter(builder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for offset, instruction := range instructions {

		var operandsBuilder strings.Builder
		instruction.OperandsString(&operandsBuilder)
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
