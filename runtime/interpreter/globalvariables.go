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

package interpreter

// GlobalVariables represents global variables defined in a program.
type GlobalVariables struct {
	variables map[string]*Variable
}

func (g *GlobalVariables) Contains(name string) bool {
	if g.variables == nil {
		return false
	}
	_, ok := g.variables[name]
	return ok
}

func (g *GlobalVariables) Get(name string) *Variable {
	if g.variables == nil {
		return nil
	}
	return g.variables[name]
}

func (g *GlobalVariables) Set(name string, variable *Variable) {
	if g.variables == nil {
		g.variables = map[string]*Variable{}
	}
	g.variables[name] = variable
}
