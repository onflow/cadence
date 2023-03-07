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

package vm

import (
	"github.com/onflow/cadence/runtime/bbq"
)

type Context struct {
	Program     *bbq.Program
	Globals     []Value
	Constants   []Value
	StaticTypes []StaticType
}

func NewContext(program *bbq.Program, globals []Value) *Context {
	return &Context{
		Program:     program,
		Globals:     globals,
		Constants:   make([]Value, len(program.Constants)),
		StaticTypes: make([]StaticType, len(program.Types)),
	}
}
