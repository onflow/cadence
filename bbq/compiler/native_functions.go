/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"github.com/onflow/cadence/sema"

	"github.com/onflow/cadence/bbq/commons"
)

var indexedNativeFunctions = make(map[string]*global)
var nativeFunctions []*global

var builtinTypes = []sema.Type{
	sema.StringType,
	sema.AccountType,
	&sema.CapabilityType{},
}

var stdlibFunctions = []string{
	commons.LogFunctionName,
	commons.PanicFunctionName,
	commons.GetAccountFunctionName,
}

func init() {
	// Here the order isn't really important.
	// Because the native functions used by a program are also
	// added to the imports section of the compiled program.
	// Then the VM will link the imports (native functions) by the name.
	for _, typ := range builtinTypes {
		registerBoundFunctions(typ)
	}

	for _, funcName := range stdlibFunctions {
		addNativeFunction(funcName)
	}
}

func registerBoundFunctions(typ sema.Type) {
	for name, _ := range typ.GetMembers() {
		funcName := commons.TypeQualifiedName(typ.QualifiedString(), name)
		addNativeFunction(funcName)
	}

	compositeType, ok := typ.(*sema.CompositeType)
	if ok && compositeType.NestedTypes != nil {
		compositeType.NestedTypes.Foreach(func(_ string, nestedType sema.Type) {
			registerBoundFunctions(nestedType)
		})
	}
}

func addNativeFunction(name string) {
	global := &global{
		name: name,
	}
	nativeFunctions = append(nativeFunctions, global)
	indexedNativeFunctions[name] = global
}
