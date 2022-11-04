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

package execute

import (
	"github.com/logrusorgru/aurora"

	"github.com/onflow/cadence/runtime/interpreter"
)

func colorizeResult(value interpreter.Value) string {
	str := value.String()
	return aurora.Colorize(str, aurora.YellowFg|aurora.BrightFg).String()
}

func formatValue(value interpreter.Value) string {
	if value == nil {
		return ""
	}

	return colorizeResult(value)
}

func colorizeError(message string) string {
	return aurora.Colorize(message, aurora.RedFg|aurora.BrightFg|aurora.BoldFm).String()
}
