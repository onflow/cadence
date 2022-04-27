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

package sema

import (
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
)

type Hint interface {
	Hint() string
	ast.HasPosition
	isHint()
}

// ReplacementHint

type ReplacementHint struct {
	Expression ast.Expression
	ast.Range
}

func (h *ReplacementHint) Hint() string {
	return fmt.Sprintf(
		"consider replacing with: `%s`",
		h.Expression,
	)
}

func (*ReplacementHint) isHint() {}

// RemovalHint

type RemovalHint struct {
	Description string
	ast.Range
}

func (h *RemovalHint) Hint() string {
	description := h.Description
	if description == "" {
		description = "code"
	}
	return fmt.Sprintf("consider removing this %s", description)
}

func (*RemovalHint) isHint() {}

// AlwaysSucceedingFailableCastHint

type AlwaysSucceedingFailableCastHint struct {
	ValueType  Type
	TargetType Type
	ast.Range
}

func (h *AlwaysSucceedingFailableCastHint) Hint() string {
	return fmt.Sprintf(
		"failable cast ('%s') from `%s` to `%s` always succeeds",
		ast.OperationFailableCast.Symbol(),
		h.ValueType,
		h.TargetType,
	)
}

func (*AlwaysSucceedingFailableCastHint) isHint() {}

// AlwaysSucceedingForceCastHint

type AlwaysSucceedingForceCastHint struct {
	ValueType  Type
	TargetType Type
	ast.Range
}

func (h *AlwaysSucceedingForceCastHint) Hint() string {
	return fmt.Sprintf(
		"force cast ('%s') from `%s` to `%s` always succeeds",
		ast.OperationForceCast.Symbol(),
		h.ValueType,
		h.TargetType,
	)
}

func (*AlwaysSucceedingForceCastHint) isHint() {}

// UnnecessaryCastHint

type UnnecessaryCastHint struct {
	TargetType Type
	ast.Range
}

func (h *UnnecessaryCastHint) Hint() string {
	return fmt.Sprintf(
		"cast to `%s` is redundant",
		h.TargetType,
	)
}

func (*UnnecessaryCastHint) isHint() {}
