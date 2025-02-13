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

package values

import "github.com/onflow/cadence/common"

type NumberValue interface {
	ComparableValue
	ToInt() (int, error)
	Negate(gauge common.MemoryGauge) NumberValue
	Plus(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	SaturatingPlus(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	Minus(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	SaturatingMinus(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	Mod(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	Mul(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	SaturatingMul(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	Div(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	SaturatingDiv(gauge common.MemoryGauge, other NumberValue) (NumberValue, error)
	ToBigEndianBytes() []byte
}
