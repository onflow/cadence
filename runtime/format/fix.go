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

package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/onflow/cadence/fixedpoint"
)

func Fix64(v int64) string {
	integer := v / fixedpoint.Fix64Factor
	fraction := v % fixedpoint.Fix64Factor
	negative := fraction < 0
	var builder strings.Builder
	if negative {
		fraction = -fraction
		if integer == 0 {
			builder.WriteByte('-')
		}
	}
	builder.WriteString(fmt.Sprint(integer))
	builder.WriteByte('.')
	builder.WriteString(PadLeft(strconv.Itoa(int(fraction)), '0', fixedpoint.Fix64Scale))
	return builder.String()
}

func UFix64(v uint64) string {
	factor := uint64(fixedpoint.Fix64Factor)
	integer := v / factor
	fraction := v % factor
	return fmt.Sprintf(
		"%d.%s",
		integer,
		PadLeft(strconv.Itoa(int(fraction)), '0', fixedpoint.Fix64Scale),
	)
}
