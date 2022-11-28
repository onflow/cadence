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

package opcode

import (
	"github.com/onflow/cadence/runtime/bbq/registers"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=Opcode

type Opcode interface{}

type Return struct{}

type ReturnValue struct {
	Index uint16
}

type Jump struct {
	Target uint16
}

type JumpIfFalse struct {
	Condition, Target uint16
}

type IntAdd struct {
	LeftOperand, RightOperand, Result uint16
}

type IntSubtract struct {
	LeftOperand, RightOperand, Result uint16
}

type IntEqual struct {
	LeftOperand, RightOperand, Result uint16
}

type IntNotEqual struct {
	LeftOperand, RightOperand, Result uint16
}

type IntLess struct {
	LeftOperand, RightOperand, Result uint16
}

type IntGreater struct {
	LeftOperand, RightOperand, Result uint16
}

type IntLessOrEqual struct {
	LeftOperand, RightOperand, Result uint16
}

type IntGreaterOrEqual struct {
	LeftOperand, RightOperand, Result uint16
}

type GetIntConstant struct {
	Index  uint16
	Target uint16
}

type True struct {
	Index uint16
}

type False struct {
	Index uint16
}

//type GetLocal struct{}

type MoveInt struct {
	From, To uint16
}

type GetGlobalFunc struct {
	Index, Result uint16
}

type Call struct {
	FuncIndex uint16
	Arguments []Argument
	Result    uint16
}

type Argument struct {
	Type  registers.RegistryType
	Index uint16
}
