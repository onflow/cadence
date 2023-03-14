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

package compiler

import (
	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/runtime/bbq/commons"
)

var indexedNativeFunctions map[string]*global
var nativeFunctions []*global

var builtinTypes = []sema.Type{
	sema.StringType,
}

// TODO: Maybe
var stdlibFunctions = []string{
	commons.LogFunctionName,
}

func init() {
	indexedNativeFunctions = make(map[string]*global)

	// Here the order isn't really important.
	// Because the native functions used by a program are also
	// added to the imports section of the compiled program.
	// Then the VM will link the imports (native functions) by the name.
	for _, builtinType := range builtinTypes {
		for name, _ := range builtinType.GetMembers() {
			funcName := commons.TypeQualifiedName(builtinType.QualifiedString(), name)
			addNativeFunction(funcName)
		}
	}

	for _, funcName := range stdlibFunctions {
		addNativeFunction(funcName)
	}
}

func addNativeFunction(name string) {
	global := &global{
		name: name,
	}
	nativeFunctions = append(nativeFunctions, global)
	indexedNativeFunctions[name] = global
}
