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
	"github.com/onflow/cadence/common"
)

type GlobalKind int

const (
	GlobalKindFunction GlobalKind = iota
	GlobalKindVariable
	GlobalKindContract
	GlobalKindImported
)

type Global[E any] struct {
	Name     string
	Location common.Location
	Index    uint16
	// used for linking
	Function *Function[E]
	Variable *Variable[E]
	Contract *Contract
	Kind     GlobalKind
}

func NewGlobal[E any](
	memoryGauge common.MemoryGauge,
	name string,
	location common.Location,
	index uint16,
	kind GlobalKind,
) *Global[E] {
	common.UseMemory(memoryGauge, common.CompilerGlobalMemoryUsage)
	return &Global[E]{
		Name:     name,
		Location: location,
		Index:    index,
		Kind:     kind,
	}
}
