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
	"github.com/onflow/cadence/runtime/errors"
)

//go:generate stringer -type=Operation

type Operation int

const (
	OperationUnknown Operation = iota
	OperationOr
	OperationAnd
	OperationEqual
	OperationUnequal
	OperationLess
	OperationGreater
	OperationLessEqual
	OperationGreaterEqual
	OperationPlus
	OperationMinus
	OperationMul
	OperationDiv
	OperationMod
	OperationConcat
	OperationNegate
	OperationNilCoalesce
	OperationMove
	OperationCast
	OperationFailableCast
	OperationForceCast
)

func (s Operation) Symbol() string {
	switch s {
	case OperationOr:
		return "||"
	case OperationAnd:
		return "&&"
	case OperationEqual:
		return "=="
	case OperationUnequal:
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
	case OperationConcat:
		return "&"
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
	}

	panic(errors.NewUnreachableError())
}
