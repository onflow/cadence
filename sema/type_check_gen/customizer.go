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

package main

import (
	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

func Update(decls []dst.Decl) []dst.Decl {
	for i, decl := range decls {
		for _, updater := range updaters {
			decls[i] = updater(decl)
		}
	}

	return decls
}

var updaters = []CodeUpdater{
	IsSubtypeFunctionUpdater,
}

type CodeUpdater func(decl dst.Decl) dst.Decl

// IsSubtypeFunctionUpdater replaces the usages of `IsSubtype` function with `IsSubTypeWithoutComparison`.
func IsSubtypeFunctionUpdater(decl dst.Decl) dst.Decl {

	return dstutil.Apply(
		decl,

		// Pre-order traversal: called before visiting children
		func(cursor *dstutil.Cursor) bool {
			currentNode := cursor.Node()

			switch currentNode := currentNode.(type) {

			case *dst.CallExpr:
				identifier, ok := currentNode.Fun.(*dst.Ident)
				if ok && identifier.Name == "IsSubType" {
					identifier.Name = "IsSubTypeWithoutComparison"
				}
			}

			// Return true to continue visiting children
			return true
		},

		// Post-order traversal: called after visiting children
		nil,
	).(dst.Decl)
}
