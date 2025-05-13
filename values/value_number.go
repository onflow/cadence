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

type NumberValue[T Value] interface {
	ComparableValue[T]
	ToInt() (int, error)
	Negate(gauge common.Gauge) T
	Plus(gauge common.Gauge, other T) (T, error)
	SaturatingPlus(gauge common.Gauge, other T) (T, error)
	Minus(gauge common.Gauge, other T) (T, error)
	SaturatingMinus(gauge common.Gauge, other T) (T, error)
	Mod(gauge common.Gauge, other T) (T, error)
	Mul(gauge common.Gauge, other T) (T, error)
	SaturatingMul(gauge common.Gauge, other T) (T, error)
	Div(gauge common.Gauge, other T) (T, error)
	SaturatingDiv(gauge common.Gauge, other T) (T, error)
	ToBigEndianBytes() []byte
}
