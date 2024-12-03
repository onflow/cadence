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
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils/common_utils"
)

type Config struct {
	*Storage
	common.MemoryGauge
	commons.ImportHandler
	ContractValueHandler
	stdlib.AccountHandler

	// TODO: Move these to a 'shared state'?
	CapabilityControllerIterations              map[AddressPath]int
	MutationDuringCapabilityControllerIteration bool
	referencedResourceKindedValues              ReferencedResourceKindedValues

	// TODO: These are temporary. Remove once storing/reading is supported for VM values.
	inter      *interpreter.Interpreter
	TypeLoader func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType
}

func NewConfig(storage *Storage) *Config {
	return &Config{
		Storage:              storage,
		MemoryGauge:          nil,
		ImportHandler:        nil,
		ContractValueHandler: nil,
		AccountHandler:       nil,

		CapabilityControllerIterations:              make(map[AddressPath]int),
		MutationDuringCapabilityControllerIteration: false,
		referencedResourceKindedValues:              ReferencedResourceKindedValues{},
	}
}

func (c *Config) WithAccountHandler(handler stdlib.AccountHandler) *Config {
	c.AccountHandler = handler
	return c
}

// TODO: This is temporary. Remove once storing/reading is supported for VM values.
func (c *Config) interpreter() *interpreter.Interpreter {
	if c.inter == nil {
		inter, err := interpreter.NewInterpreter(
			nil,
			common_utils.TestLocation,
			&interpreter.Config{
				//Storage:               c.Storage,
				ImportLocationHandler: nil,
				CompositeTypeHandler: func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
					return c.TypeLoader(location, typeID).(*sema.CompositeType)
				},
				InterfaceTypeHandler: func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
					return c.TypeLoader(location, typeID).(*sema.InterfaceType)
				},
			},
		)

		if err != nil {
			panic(err)
		}

		c.inter = inter
	}

	return c.inter
}

type ContractValueHandler func(conf *Config, location common.Location) *CompositeValue

type AddressPath struct {
	Address common.Address
	Path    PathValue
}
