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

package compiler

import (
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
)

type Global struct {
	Name     string
	Location common.Location
	Index    uint16
	Category commons.GlobalCategory
}

func NewGlobal(
	memoryGauge common.MemoryGauge,
	name string,
	location common.Location,
	index uint16,
	category commons.GlobalCategory,
) *Global {
	common.UseMemory(memoryGauge, common.CompilerGlobalMemoryUsage)
	return &Global{
		Name:     name,
		Location: location,
		Index:    index,
		Category: category,
	}
}

var _ commons.SortableGlobal = &Global{}

func (g *Global) GetCategory() commons.GlobalCategory {
	return g.Category
}

func (g *Global) GetName() string {
	return g.Name
}

func (g *Global) SetIndex(index uint16) {
	g.Index = index
}
