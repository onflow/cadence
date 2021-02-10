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

package runtime

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type Context struct {
	Interface         Interface
	Location          Location
	PredeclaredValues []ValueDeclaration
	codes             map[common.LocationID]string
	programs          map[common.LocationID]*ast.Program
}

func (c Context) SetCode(location common.Location, code string) {
	c.codes[location.ID()] = code
}

func (c Context) SetProgram(location common.Location, program *ast.Program) {
	c.programs[location.ID()] = program
}

func (c Context) WithLocation(location common.Location) Context {
	result := c
	result.Location = location
	return result
}

func (c *Context) InitializeCodesAndPrograms() {
	if c.codes == nil {
		c.codes = map[common.LocationID]string{}
	}

	if c.programs == nil {
		c.programs = map[common.LocationID]*ast.Program{}
	}
}
