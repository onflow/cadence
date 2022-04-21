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
//
type GlobalVariables map[string]*Variable

func (globalVars GlobalVariables) Contains(name string) bool {
	_, ok := globalVars[name]
	return ok
}

func (globalVars GlobalVariables) Get(name string) (*Variable, bool) {
	variable, ok := globalVars[name]
	return variable, ok
}

func (globalVars GlobalVariables) Set(name string, variable *Variable) {
	globalVars[name] = variable
}
