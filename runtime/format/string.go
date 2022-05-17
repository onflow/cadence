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

package format

import (
	"github.com/onflow/cadence/runtime/ast"
)

func String(s string) string {
	return ast.QuoteString(s)
}

func FormattedStringLength(s string) int {
	asciiNonPrintableChars := 0
	escapedChars := 0
	for _, r := range s {
		switch r {
		case 0,
			'\n',
			'\r',
			'\t',
			'\\',
			'"':
			escapedChars++
		default:
			// ASCII non-printable characters (i.e: out of range from space through DEL-1)
			if 0x20 > r || r > 0x7E {
				asciiNonPrintableChars++
			}
		}
	}

	// len = printableChars + (escapedChars x 2) + (nonPrintableChars x 8) + (quote x 2)
	return (len(s) - asciiNonPrintableChars - escapedChars) + escapedChars*2 + asciiNonPrintableChars*8 + 2
}
