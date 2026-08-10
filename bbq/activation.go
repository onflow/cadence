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

package bbq

import (
	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
)

// Activation is a map of CanonicalName keys to values.
type Activation[T any] = activations.KeyedActivation[CanonicalName, T]

func NewActivation[T any](memoryGauge common.MemoryGauge, parent *Activation[T]) *Activation[T] {
	return activations.NewKeyedActivation[CanonicalName, T](memoryGauge, parent)
}
