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

package runtime

import (
	"github.com/onflow/cadence/runtime/ast"
)

type Context struct {
	Interface      Interface
	Location       Location
	Environment    Environment
	CoverageReport *CoverageReport
}

// codesAndPrograms collects the source code and AST for each location.
// It is purely used for debugging: Both the codes and the programs
// are provided in runtime errors.
type codesAndPrograms struct {
	codes    map[Location][]byte
	programs map[Location]*ast.Program
}

func (c codesAndPrograms) setCode(location Location, code []byte) {
	c.codes[location] = code
}

func (c codesAndPrograms) setProgram(location Location, program *ast.Program) {
	c.programs[location] = program
}

func newCodesAndPrograms() codesAndPrograms {
	return codesAndPrograms{
		codes:    map[Location][]byte{},
		programs: map[Location]*ast.Program{},
	}
}
