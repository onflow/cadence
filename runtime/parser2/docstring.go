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

package parser2

import (
	"regexp"
	"strings"
)

var pragmaArgumentRegexp = regexp.MustCompile(`^\s+pragma\s+arguments\s+(.*)(?:\n|$)`)

// ParseDocstringPragmaArguments parses the docstring and returns the values of all pragma arguments declarations.
//
// A pragma arguments declaration has the form `pragma arguments <argument-list>`,
// where <argument-list> is a Cadence argument list.
//
// The validity of the argument list is NOT checked by this function.
//
func ParseDocstringPragmaArguments(docString string) []string {
	var pragmaArguments []string

	for _, line := range strings.Split(docString, "\n") {
		match := pragmaArgumentRegexp.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		pragmaArguments = append(pragmaArguments, match[1])
	}

	return pragmaArguments
}
