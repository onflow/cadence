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
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/logrusorgru/aurora/v4"

	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/interpreter"
)

func PrintBytecode(
	builder *strings.Builder,
	code []byte,
	resolve bool,
	constants []constant.DecodedConstant,
	types [][]byte,
	functionNames []string,
	colorize bool,
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
		colorize,
	)
}

func PrintBytecodeWithFlow(
	builder *strings.Builder,
	code []byte,
	resolve bool,
	constants []constant.DecodedConstant,
	types [][]byte,
	functionNames []string,
	colorize bool,
) error {
	instructions := DecodeInstructions(code)
	staticTypes := DecodeStaticTypes(types)
	return PrintInstructionsWithFlow(
		builder,
		instructions,
		resolve,
		constants,
		staticTypes,
		functionNames,
		colorize,
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
	constants []constant.DecodedConstant,
	types []interpreter.StaticType,
	functionNames []string,
	colorize bool,
) error {

	tabWriter := tabwriter.NewWriter(builder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for offset, instruction := range instructions {

		var operandsBuilder strings.Builder
		if resolve {
			instruction.ResolvedOperandsString(
				&operandsBuilder,
				constants,
				types,
				functionNames,
				colorize,
			)
		} else {
			instruction.OperandsString(&operandsBuilder, colorize)
		}

		var formattedOffset string
		if colorize {
			formattedOffset = ColorizeOffset(offset)
		} else {
			formattedOffset = fmt.Sprint(offset)
		}

		var formattedOpcode string
		if colorize {
			formattedOpcode = ColorizeOpcode(instruction.Opcode())
		} else {
			formattedOpcode = fmt.Sprint(instruction.Opcode())
		}

		_, _ = fmt.Fprintf(
			tabWriter,
			"%s |\t%s |\t%s\n",
			formattedOffset,
			formattedOpcode,
			operandsBuilder.String(),
		)
	}

	_ = tabWriter.Flush()
	_, _ = fmt.Fprintln(builder)

	return nil
}

// PrintInstructionsWithFlowMode prints instructions in block format with flow visualization
func PrintInstructionsWithFlow(
	builder *strings.Builder,
	instructions []Instruction,
	resolve bool,
	constants []constant.DecodedConstant,
	types []interpreter.StaticType,
	functionNames []string,
	colorize bool,
) error {

	flowAnalysis := analyzeControlFlow(instructions)

	// Render as basic blocks
	blockRenderer := &BlockRenderer{
		analysis:     flowAnalysis,
		colorize:     colorize,
		instructions: instructions,
	}
	blockOutput := blockRenderer.renderBasicBlocks(constants, types, functionNames, resolve)
	builder.WriteString(blockOutput)

	return nil
}

func ColorizeOffset(offset int) string {
	return aurora.Gray(12, fmt.Sprintf("%3d", offset)).String()
}

func ColorizeOpcode(opcode Opcode) string {
	if opcode.IsControlFlow() {
		return aurora.Red(opcode).String()
	}

	return aurora.Blue(opcode).String()
}

// Flow visualization types and constants

// JumpType categorizes different kinds of control flow
type JumpType int

const (
	JumpTypeUnconditional JumpType = iota
	JumpTypeConditional
	JumpTypeCall
	JumpTypeReturn
)

// FlowAnalysis contains control flow information for a sequence of instructions
type FlowAnalysis struct {
	JumpInfoMap  map[int]JumpInfo // instruction index -> info for jump from this instruction
	BlockLeaders []int            // instruction indexes which are start instructions of basic blocks
	BasicBlocks  []BasicBlock     // identified basic blocks
}

// JumpInfo describes a jump from one instruction to another
type JumpInfo struct {
	Target    int      // target instruction index
	JumpType  JumpType // conditional, unconditional, call, return
	Condition string   // condition description (e.g., "if false", "if nil")
}

// BasicBlock represents a sequence of instructions with single entry/exit
type BasicBlock struct {
	Start        int   // first instruction index
	End          int   // last instruction index
	Successors   []int // indices of blocks that can follow this one
	Predecessors []int // indices of blocks that can precede this one
}

// analyzeControlFlow performs control flow analysis on a sequence of instructions
func analyzeControlFlow(instructions []Instruction) *FlowAnalysis {
	analysis := &FlowAnalysis{
		JumpInfoMap:  make(map[int]JumpInfo),
		BlockLeaders: []int{},
	}

	// First instruction is always a leader
	analysis.BlockLeaders = append(analysis.BlockLeaders, 0)

	// First pass: identify all jumps
	for i, instr := range instructions {
		switch instr := instr.(type) {
		case InstructionJump:
			target := int(instr.Target)
			jumpInfo := JumpInfo{
				Target:   target,
				JumpType: JumpTypeUnconditional,
			}
			analysis.JumpInfoMap[i] = jumpInfo
			// instructions that are jump targets are leaders and instructions immediately after jumps are leaders
			analysis.BlockLeaders = append(analysis.BlockLeaders, target, i+1)
		case InstructionJumpIfFalse:
			target := int(instr.Target)
			jumpInfo := JumpInfo{
				Target:    target,
				JumpType:  JumpTypeConditional,
				Condition: "if false",
			}
			analysis.JumpInfoMap[i] = jumpInfo
			// instructions that are jump targets are leaders and instructions immediately after jumps are leaders
			analysis.BlockLeaders = append(analysis.BlockLeaders, target, i+1)

		case InstructionJumpIfTrue:
			target := int(instr.Target)
			jumpInfo := JumpInfo{
				Target:    target,
				JumpType:  JumpTypeConditional,
				Condition: "if true",
			}
			analysis.JumpInfoMap[i] = jumpInfo
			// instructions that are jump targets are leaders and instructions immediately after jumps are leaders
			analysis.BlockLeaders = append(analysis.BlockLeaders, target, i+1)
		case InstructionJumpIfNil:
			target := int(instr.Target)
			jumpInfo := JumpInfo{
				Target:    target,
				JumpType:  JumpTypeConditional,
				Condition: "if nil",
			}
			analysis.JumpInfoMap[i] = jumpInfo
			// instructions that are jump targets are leaders and instructions immediately after jumps are leaders
			analysis.BlockLeaders = append(analysis.BlockLeaders, target, i+1)

		case InstructionInvoke, InstructionInvokeDynamic:
			// Function calls
			jumpInfo := JumpInfo{
				Target:   -1, // unknown target by default
				JumpType: JumpTypeCall,
			}

			// could analyze for call targets here, complicated

			analysis.JumpInfoMap[i] = jumpInfo
			// instructions immediately after jumps are leaders
			analysis.BlockLeaders = append(analysis.BlockLeaders, i+1)
		case InstructionReturn, InstructionReturnValue:
			jumpInfo := JumpInfo{
				Target:   -1, // function exit
				JumpType: JumpTypeReturn,
			}
			analysis.JumpInfoMap[i] = jumpInfo
			// instructions immediately after jumps are leaders, potentially end of program as well
			if i+1 < len(instructions) {
				analysis.BlockLeaders = append(analysis.BlockLeaders, i+1)
			}
		}
	}

	// sort BlockLeaders and remove duplicates
	slices.Sort(analysis.BlockLeaders)
	analysis.BlockLeaders = slices.Compact(analysis.BlockLeaders)

	// Second pass: identify basic blocks
	analysis.BasicBlocks = identifyBasicBlocks(instructions, analysis)

	return analysis
}

// identifyBasicBlocks finds basic blocks in the instruction sequence
func identifyBasicBlocks(instructions []Instruction, analysis *FlowAnalysis) []BasicBlock {
	// Create blocks
	var blocks []BasicBlock
	for i, start := range analysis.BlockLeaders {
		end := len(instructions) - 1
		if i+1 < len(analysis.BlockLeaders) {
			end = analysis.BlockLeaders[i+1] - 1
		}

		block := BasicBlock{
			Start: start,
			End:   end,
		}
		blocks = append(blocks, block)
	}

	return blocks
}

// BlockRenderer handles basic block visualization
type BlockRenderer struct {
	analysis     *FlowAnalysis
	colorize     bool
	instructions []Instruction
}

// renderBasicBlocks creates a basic block visualization
func (r *BlockRenderer) renderBasicBlocks(
	constants []constant.DecodedConstant,
	types []interpreter.StaticType,
	functionNames []string,
	resolve bool,
) string {
	var builder strings.Builder

	// Build block connections map
	blockConnections := r.buildBlockConnections()

	// Render each basic block
	for i, block := range r.analysis.BasicBlocks {
		r.renderBlock(&builder, i, block, constants, types, functionNames, resolve)

		// Show connections to other blocks
		if connections, hasConnections := blockConnections[i]; hasConnections {
			r.renderBlockConnections(&builder, connections)
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// buildBlockConnections maps block indices to their successor blocks
func (r *BlockRenderer) buildBlockConnections() map[int][]BlockConnection {
	connections := make(map[int][]BlockConnection)

	for blockIndex, block := range r.analysis.BasicBlocks {
		var blockConnections []BlockConnection

		// Check jumps from the last instruction of this block
		lastInstrIndex := block.End
		if jump, hasJump := r.analysis.JumpInfoMap[lastInstrIndex]; hasJump {
			if jump.JumpType == JumpTypeCall {
				// Function calls are considered external jumps
				conn := BlockConnection{
					TargetBlock: -1,
					JumpType:    jump.JumpType,
					Condition:   jump.Condition,
					IsJump:      true,
				}
				blockConnections = append(blockConnections, conn)
			} else if jump.Target >= 0 {
				// Regular jumps within the function
				targetBlock := r.findBlockContaining(jump.Target)
				if targetBlock >= 0 && targetBlock != blockIndex {
					conn := BlockConnection{
						TargetBlock: targetBlock,
						JumpType:    jump.JumpType,
						Condition:   jump.Condition,
						IsJump:      true,
					}
					blockConnections = append(blockConnections, conn)
				}
			}
		}

		// Check for fall-through to next block
		if blockIndex+1 < len(r.analysis.BasicBlocks) {
			// If last instruction doesn't unconditionally jump or return or call, add fall-through
			if !r.hasUnconditionalExit(lastInstrIndex) {
				conn := BlockConnection{
					TargetBlock: blockIndex + 1,
					JumpType:    JumpTypeUnconditional,
					Condition:   "fall through",
					IsJump:      false,
				}
				blockConnections = append(blockConnections, conn)
			}
		}

		if len(blockConnections) > 0 {
			connections[blockIndex] = blockConnections
		}
	}

	return connections
}

// BlockConnection represents a connection between basic blocks
type BlockConnection struct {
	TargetBlock int
	JumpType    JumpType
	Condition   string
	IsJump      bool
}

// findBlockContaining finds which basic block contains the given instruction index
func (r *BlockRenderer) findBlockContaining(instrIndex int) int {
	for i, block := range r.analysis.BasicBlocks {
		if instrIndex >= block.Start && instrIndex <= block.End {
			return i
		}
	}
	return -1
}

// hasUnconditionalExit checks if an instruction unconditionally exits (jump/return)
func (r *BlockRenderer) hasUnconditionalExit(instrIndex int) bool {
	if jump, hasJump := r.analysis.JumpInfoMap[instrIndex]; hasJump {
		if jump.JumpType == JumpTypeUnconditional || jump.JumpType == JumpTypeReturn || jump.JumpType == JumpTypeCall {
			return true
		}
	}
	return false
}

// renderBlock renders a single basic block
func (r *BlockRenderer) renderBlock(
	builder *strings.Builder,
	blockIndex int,
	block BasicBlock,
	constants []constant.DecodedConstant,
	types []interpreter.StaticType,
	functionNames []string,
	resolve bool,
) {
	// Block header
	header := fmt.Sprintf("Block %d (%d-%d)", blockIndex, block.Start, block.End)
	if r.colorize {
		header = aurora.Bold(aurora.Cyan(header)).String()
	}

	builder.WriteString("┌─ ")
	builder.WriteString(header)
	builder.WriteString(" ")

	// Calculate padding for box drawing
	maxWidth := 70
	headerLen := len(fmt.Sprintf("Block %d (%d-%d)", blockIndex, block.Start, block.End))
	padding := maxWidth - headerLen - 4
	if padding > 0 {
		builder.WriteString(strings.Repeat("─", padding))
	}
	builder.WriteString("┐\n")

	opcodePadding := maxOpcodeLen
	if r.colorize {
		opcodePadding += len(aurora.White("").String())
	}

	// Block instructions
	for instrIndex := block.Start; instrIndex <= block.End; instrIndex++ {
		instruction := r.instructions[instrIndex]

		var operandsBuilder strings.Builder
		if resolve {
			instruction.ResolvedOperandsString(
				&operandsBuilder,
				constants,
				types,
				functionNames,
				r.colorize,
			)
		} else {
			instruction.OperandsString(&operandsBuilder, r.colorize)
		}

		var formattedOffset string
		if r.colorize {
			formattedOffset = ColorizeOffset(instrIndex)
		} else {
			formattedOffset = fmt.Sprintf("%3d", instrIndex)
		}

		var formattedOpcode string
		if r.colorize {
			formattedOpcode = ColorizeOpcode(instruction.Opcode())
		} else {
			formattedOpcode = fmt.Sprint(instruction.Opcode())
		}

		// Format instruction line
		builder.WriteString("│ ")

		fmt.Fprintf(builder, "%4s | %-*s | %s",
			formattedOffset,
			opcodePadding,
			formattedOpcode,
			operandsBuilder.String(),
		)
		builder.WriteString("\n")
	}

	// Block footer
	builder.WriteString("└")
	builder.WriteString(strings.Repeat("─", maxWidth-1))
	builder.WriteString("┘\n")
}

var maxOpcodeLen int

func init() {
	for opcode := range OpcodeMax {
		l := len(opcode.String())
		if l > maxOpcodeLen {
			maxOpcodeLen = l
		}
	}
}

// renderBlockConnections shows connections from a block to other blocks
func (r *BlockRenderer) renderBlockConnections(
	builder *strings.Builder,
	connections []BlockConnection,
) {
	if len(connections) == 0 {
		return
	}

	for i, conn := range connections {
		var arrow, description string

		switch conn.JumpType {
		case JumpTypeUnconditional:
			if conn.IsJump {
				arrow = "──→"
				description = "jump"
			} else {
				arrow = "──→"
				description = "fall through"
			}
		case JumpTypeConditional:
			arrow = "─?→"
			description = fmt.Sprintf("jump %s", conn.Condition)
		case JumpTypeReturn:
			arrow = "──↩"
			description = "return"
		case JumpTypeCall:
			arrow = "──→"
			description = "function_call"
		}

		connectionText := fmt.Sprintf("    %s Block %d", arrow, conn.TargetBlock)
		if conn.TargetBlock == -1 {
			connectionText = fmt.Sprintf("    %s Unknown target", arrow)
		}
		if description != "" && description != "jump" {
			connectionText += fmt.Sprintf(" (%s)", description)
		}

		if r.colorize {
			switch conn.JumpType {
			case JumpTypeUnconditional:
				connectionText = aurora.Green(connectionText).String()
			case JumpTypeConditional:
				connectionText = aurora.Yellow(connectionText).String()
			case JumpTypeReturn:
				connectionText = aurora.Red(connectionText).String()
			case JumpTypeCall:
				connectionText = aurora.Blue(connectionText).String()
			}
		}

		builder.WriteString(connectionText)
		builder.WriteString("\n")

		// Add spacing between multiple connections
		if i < len(connections)-1 {
			builder.WriteString("\n")
		}
	}
}
