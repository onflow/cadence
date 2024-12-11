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
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
)

type CodeGen interface {
	Offset() int
	Code() interface{}
	EmitNil()
	EmitTrue()
	EmitFalse()
	EmitDup()
	EmitDrop()
	EmitGetConstant(index uint16)
	EmitJump(target uint16) int
	EmitJumpIfFalse(target uint16) int
	PatchJump(offset int, newTarget uint16)
	EmitReturnValue()
	EmitReturn()
	EmitGetLocal(index uint16)
	EmitSetLocal(index uint16)
	EmitGetGlobal(index uint16)
	EmitSetGlobal(index uint16)
	EmitGetField()
	EmitSetField()
	EmitGetIndex()
	EmitSetIndex()
	EmitNewArray(index uint16, size uint16, isResource bool)
	EmitIntAdd()
	EmitIntSubtract()
	EmitIntMultiply()
	EmitIntDivide()
	EmitIntMod()
	EmitEqual()
	EmitNotEqual()
	EmitIntLess()
	EmitIntLessOrEqual()
	EmitIntGreater()
	EmitIntGreaterOrEqual()
	EmitUnwrap()
	EmitCast(index uint16, kind opcode.CastKind)
	EmitDestroy()
	EmitTransfer(index uint16)
	EmitNewRef(index uint16)
	EmitPath(domain common.PathDomain, identifier string)
	EmitNew(kind uint16, index uint16)
	EmitInvoke(typeArgs []uint16)
	EmitInvokeDynamic(name string, typeArgs []uint16, argCount uint16)
}

type BytecodeGen struct {
	code []byte
}

var _ CodeGen = &BytecodeGen{}

func (g *BytecodeGen) Offset() int {
	return len(g.code)
}

func (g *BytecodeGen) Code() interface{} {
	return g.code
}

func (g *BytecodeGen) EmitNil() {
	opcode.EmitNil(&g.code)
}

func (g *BytecodeGen) EmitTrue() {
	opcode.EmitTrue(&g.code)
}

func (g *BytecodeGen) EmitFalse() {
	opcode.EmitFalse(&g.code)
}

func (g *BytecodeGen) EmitDup() {
	opcode.EmitDup(&g.code)
}

func (g *BytecodeGen) EmitDrop() {
	opcode.EmitDrop(&g.code)
}

func (g *BytecodeGen) EmitGetConstant(index uint16) {
	opcode.EmitGetConstant(&g.code, index)
}

func (g *BytecodeGen) EmitJump(target uint16) int {
	return opcode.EmitJump(&g.code, target)
}

func (g *BytecodeGen) EmitJumpIfFalse(target uint16) int {
	return opcode.EmitJumpIfFalse(&g.code, target)
}

func (g *BytecodeGen) PatchJump(offset int, newTarget uint16) {
	opcode.PatchJump(&g.code, offset, newTarget)
}

func (g *BytecodeGen) EmitReturnValue() {
	opcode.EmitReturnValue(&g.code)
}

func (g *BytecodeGen) EmitReturn() {
	opcode.EmitReturn(&g.code)
}

func (g *BytecodeGen) EmitGetLocal(index uint16) {
	opcode.EmitGetLocal(&g.code, index)
}
func (g *BytecodeGen) EmitSetLocal(index uint16) {
	opcode.EmitSetLocal(&g.code, index)
}

func (g *BytecodeGen) EmitGetGlobal(index uint16) {
	opcode.EmitGetGlobal(&g.code, index)
}

func (g *BytecodeGen) EmitSetGlobal(index uint16) {
	opcode.EmitSetGlobal(&g.code, index)
}

func (g *BytecodeGen) EmitGetField() {
	opcode.EmitGetField(&g.code)
}

func (g *BytecodeGen) EmitSetField() {
	opcode.EmitSetField(&g.code)
}

func (g *BytecodeGen) EmitGetIndex() {
	opcode.EmitGetIndex(&g.code)
}

func (g *BytecodeGen) EmitSetIndex() {
	opcode.EmitSetIndex(&g.code)
}

func (g *BytecodeGen) EmitNewArray(index uint16, size uint16, isResource bool) {
	opcode.EmitNewArray(&g.code, index, size, isResource)
}

func (g *BytecodeGen) EmitIntAdd() {
	opcode.EmitIntAdd(&g.code)
}

func (g *BytecodeGen) EmitIntSubtract() {
	opcode.EmitIntSubtract(&g.code)
}

func (g *BytecodeGen) EmitIntMultiply() {
	opcode.EmitIntMultiply(&g.code)
}

func (g *BytecodeGen) EmitIntDivide() {
	opcode.EmitIntDivide(&g.code)
}

func (g *BytecodeGen) EmitIntMod() {
	opcode.EmitIntMod(&g.code)
}

func (g *BytecodeGen) EmitEqual() {
	opcode.EmitEqual(&g.code)
}

func (g *BytecodeGen) EmitNotEqual() {
	opcode.EmitNotEqual(&g.code)
}

func (g *BytecodeGen) EmitIntLess() {
	opcode.EmitIntLess(&g.code)
}

func (g *BytecodeGen) EmitIntLessOrEqual() {
	opcode.EmitIntLessOrEqual(&g.code)
}

func (g *BytecodeGen) EmitIntGreater() {
	opcode.EmitIntGreater(&g.code)
}

func (g *BytecodeGen) EmitIntGreaterOrEqual() {
	opcode.EmitIntGreaterOrEqual(&g.code)
}

func (g *BytecodeGen) EmitUnwrap() {
	opcode.EmitUnwrap(&g.code)
}

func (g *BytecodeGen) EmitCast(index uint16, kind opcode.CastKind) {
	opcode.EmitCast(&g.code, index, kind)
}

func (g *BytecodeGen) EmitDestroy() {
	opcode.EmitDestroy(&g.code)
}

func (g *BytecodeGen) EmitTransfer(index uint16) {
	opcode.EmitTransfer(&g.code, index)
}

func (g *BytecodeGen) EmitNewRef(index uint16) {
	opcode.EmitNewRef(&g.code, index)
}

func (g *BytecodeGen) EmitPath(domain common.PathDomain, identifier string) {
	opcode.EmitPath(&g.code, domain, identifier)
}

func (g *BytecodeGen) EmitNew(kind uint16, index uint16) {
	opcode.EmitNew(&g.code, kind, index)
}

func (g *BytecodeGen) EmitInvoke(typeArgs []uint16) {
	opcode.EmitInvoke(&g.code, typeArgs)
}

func (g *BytecodeGen) EmitInvokeDynamic(name string, typeArgs []uint16, argCount uint16) {
	opcode.EmitInvokeDynamic(&g.code, name, typeArgs, argCount)
}
