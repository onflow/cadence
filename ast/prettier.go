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

package ast

import (
	"strings"

	"github.com/turbolent/prettier"
)

type Pretty interface {
	Doc() prettier.Doc
}

func Prettier(element Pretty) string {
	var builder strings.Builder
	doc := element.Doc().Flatten()
	prettier.Prettier(&builder, doc, 80, "    ")
	return builder.String()
}

// docOrEmpty returns the document of the given element,
// or an empty text document if the element is the zero value.
//
// NOTE: The function is generic because the version which takes an interface
// will not properly allow for a nil check: When passing a pointer typed value as an interface,
// a typed nil is created, which is not equal to nil.
func docOrEmpty[T interface {
	Pretty
	comparable
}](element T) prettier.Doc {
	var empty T
	if element == empty {
		return prettier.Text("")
	}
	return element.Doc()
}
