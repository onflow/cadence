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

package stdlib

import (
	"fmt"

	"github.com/onflow/cadence/sema"
)

var StandardLibraryTypes = map[sema.TypeID]sema.ContainedType{}

func init() {
	stdlibTypes := []sema.Type{
		BLSType,
		RLPType,
	}

	extractTypes(
		stdlibTypes,
	)
}

func extractTypes(
	types []sema.Type,
) {
	for len(types) > 0 {
		lastIndex := len(types) - 1
		typ := types[lastIndex]
		types[lastIndex] = nil
		types = types[:lastIndex]

		var nestedTypes *sema.StringTypeOrderedMap

		switch typ := typ.(type) {
		case *sema.CompositeType:
			StandardLibraryTypes[typ.ID()] = typ
			nestedTypes = typ.NestedTypes
		case *sema.InterfaceType:
			StandardLibraryTypes[typ.ID()] = typ
			nestedTypes = typ.NestedTypes
		default:
			panic(fmt.Errorf("expected only composite or interface type, found %t", typ))
		}

		if nestedTypes == nil {
			continue
		}

		nestedTypes.Foreach(func(_ string, nestedType sema.Type) {
			types = append(types, nestedType)
		})
	}
}
