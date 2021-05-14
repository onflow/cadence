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

package format

import (
	"strings"
)

func Composite(typeID string, fields []struct {
	Name  string
	Value string
}) string {
	var builder strings.Builder
	builder.WriteString(typeID)
	builder.WriteRune('(')
	for i, nameValuePair := range fields {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(nameValuePair.Name)
		builder.WriteString(": ")
		builder.WriteString(nameValuePair.Value)
	}
	builder.WriteRune(')')
	return builder.String()
}
