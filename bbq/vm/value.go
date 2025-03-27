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

package vm

import (
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/interpreter"
)

type Value = interpreter.Value

// ConvertAndBox converts a value to a target type, and boxes in optionals and any value, if necessary
func ConvertAndBox(
	config *Config,
	value Value,
	valueType, targetType bbq.StaticType,
) Value {
	valueSemaType := interpreter.MustConvertStaticToSemaType(valueType, config)
	targetSemaType := interpreter.MustConvertStaticToSemaType(targetType, config)

	return interpreter.ConvertAndBox(
		config,
		EmptyLocationRange,
		value,
		valueSemaType,
		targetSemaType,
	)
}
