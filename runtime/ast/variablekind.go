/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

//go:generate go run golang.org/x/tools/cmd/stringer -type=VariableKind

type VariableKind uint

const (
	VariableKindNotSpecified VariableKind = iota
	VariableKindVariable
	VariableKindConstant
)

func VariableKindCount() int {
	return len(_VariableKind_index) - 1
}

var VariableKinds = []VariableKind{
	VariableKindConstant,
	VariableKindVariable,
}

func (k VariableKind) Name() string {
	switch k {
	case VariableKindNotSpecified:
		return ""
	case VariableKindVariable:
		return "variable"
	case VariableKindConstant:
		return "constant"
	}

	panic(errors.NewUnreachableError())
}

func (k VariableKind) Keyword() string {
	switch k {
	case VariableKindNotSpecified:
		return ""
	case VariableKindVariable:
		return "var"
	case VariableKindConstant:
		return "let"
	}

	panic(errors.NewUnreachableError())
}

func (k VariableKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}
