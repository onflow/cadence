/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"encoding/json"

	"github.com/onflow/cadence/runtime/errors"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=Operation

type Operation uint

const (
	OperationUnknown Operation = iota
	OperationOr
	OperationAnd
	OperationEqual
	OperationNotEqual
	OperationLess
	OperationGreater
	OperationLessEqual
	OperationGreaterEqual
	OperationPlus
	OperationMinus
	OperationMul
	OperationDiv
	OperationMod
	OperationNegate
	OperationNilCoalesce
	OperationMove
	OperationCast
	OperationFailableCast
	OperationForceCast
	OperationBitwiseOr
	OperationBitwiseXor
	OperationBitwiseAnd
	OperationBitwiseLeftShift
	OperationBitwiseRightShift
)

func OperationCount() int {
	return len(_Operation_index) - 1
}

func (s Operation) Symbol() string {
	switch s {
	case OperationOr:
		return "||"
	case OperationAnd:
		return "&&"
	case OperationEqual:
		return "=="
	case OperationNotEqual:
		return "!="
	case OperationLess:
		return "<"
	case OperationGreater:
		return ">"
	case OperationLessEqual:
		return "<="
	case OperationGreaterEqual:
		return ">="
	case OperationPlus:
		return "+"
	case OperationMinus:
		return "-"
	case OperationMul:
		return "*"
	case OperationDiv:
		return "/"
	case OperationMod:
		return "%"
	case OperationNegate:
		return "!"
	case OperationNilCoalesce:
		return "??"
	case OperationMove:
		return "<-"
	case OperationCast:
		return "as"
	case OperationFailableCast:
		return "as?"
	case OperationForceCast:
		return "as!"
	case OperationBitwiseOr:
		return "|"
	case OperationBitwiseXor:
		return "^"
	case OperationBitwiseAnd:
		return "&"
	case OperationBitwiseLeftShift:
		return "<<"
	case OperationBitwiseRightShift:
		return ">>"
	}

	panic(errors.NewUnreachableError())
}

func (s Operation) Category() string {
	switch s {
	case OperationOr,
		OperationAnd,
		OperationNegate:
		return "logical"

	case OperationEqual,
		OperationNotEqual,
		OperationLess,
		OperationGreater,
		OperationLessEqual,
		OperationGreaterEqual:
		return "comparison"

	case OperationPlus,
		OperationMinus,
		OperationMul,
		OperationDiv,
		OperationMod:
		return "arithmetic"

	case OperationNilCoalesce:
		return "nil-coalescing"

	case OperationMove:
		return "move"

	case OperationCast,
		OperationFailableCast,
		OperationForceCast:
		return "casting"

	case OperationBitwiseOr,
		OperationBitwiseXor,
		OperationBitwiseAnd,
		OperationBitwiseLeftShift,
		OperationBitwiseRightShift:
		return "bitwise"
	}

	panic(errors.NewUnreachableError())
}

func (s Operation) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}
