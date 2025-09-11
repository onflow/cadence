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

type GlobalInfo struct {
	Name     string
	Location common.Location
	Index    uint16
}

type Global interface {
	GetGlobalInfo() GlobalInfo
}

type FunctionGlobal[E any] struct {
	GlobalInfo
	Function *Function[E]
}

type VariableGlobal[E any] struct {
	GlobalInfo
	Variable *Variable[E]
}

type ContractGlobal struct {
	GlobalInfo
	Contract *Contract
}

type ImportedGlobal struct {
	GlobalInfo
}

func NewFunctionGlobal[E any](
	memoryGauge common.MemoryGauge,
	name string,
	location common.Location,
	index uint16,
) *FunctionGlobal[E] {
	common.UseMemory(memoryGauge, common.CompilerGlobalMemoryUsage)
	return &FunctionGlobal[E]{
		GlobalInfo: GlobalInfo{
			Name:     name,
			Location: location,
			Index:    index,
		},
	}
}

func (g FunctionGlobal[E]) GetGlobalInfo() GlobalInfo {
	return g.GlobalInfo
}

func (g VariableGlobal[E]) GetGlobalInfo() GlobalInfo {
	return g.GlobalInfo
}

func (g ContractGlobal) GetGlobalInfo() GlobalInfo {
	return g.GlobalInfo
}

func (g ImportedGlobal) GetGlobalInfo() GlobalInfo {
	return g.GlobalInfo
}

func NewVariableGlobal[E any](
	memoryGauge common.MemoryGauge,
	name string,
	location common.Location,
	index uint16,
) *VariableGlobal[E] {
	common.UseMemory(memoryGauge, common.CompilerGlobalMemoryUsage)
	return &VariableGlobal[E]{
		GlobalInfo: GlobalInfo{
			Name:     name,
			Location: location,
			Index:    index,
		},
	}
}

func NewContractGlobal(
	memoryGauge common.MemoryGauge,
	name string,
	location common.Location,
	index uint16,
) *ContractGlobal {
	common.UseMemory(memoryGauge, common.CompilerGlobalMemoryUsage)
	return &ContractGlobal{
		GlobalInfo: GlobalInfo{
			Name:     name,
			Location: location,
			Index:    index,
		},
	}
}

func NewImportedGlobal(
	memoryGauge common.MemoryGauge,
	name string,
	location common.Location,
	index uint16,
) *ImportedGlobal {
	common.UseMemory(memoryGauge, common.CompilerGlobalMemoryUsage)
	return &ImportedGlobal{
		GlobalInfo: GlobalInfo{
			Name:     name,
			Location: location,
			Index:    index,
		},
	}
}
