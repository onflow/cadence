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

package runtime

import (
	"math"

	"github.com/onflow/cadence/common"
)

type LimitingMemoryGauge struct {
	Weights map[common.MemoryKind]uint64
	Usage   uint64
	Limit   uint64
}

func NewLimitingMemoryGauge(
	weights map[common.MemoryKind]uint64,
	limit uint64,
) *LimitingMemoryGauge {
	if limit == 0 {
		limit = math.MaxUint64
	}
	return &LimitingMemoryGauge{
		Weights: weights,
		Limit:   limit,
		Usage:   0,
	}
}

func (g *LimitingMemoryGauge) MeterMemory(usage common.MemoryUsage) error {
	g.Usage += g.Weights[usage.Kind] * usage.Amount

	if g.Usage > g.Limit {
		return MemoryLimitExceededError{
			Limit: g.Limit,
			Usage: g.Usage,
		}
	}

	return nil
}
