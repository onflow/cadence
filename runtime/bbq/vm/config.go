/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/bbq/commons"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type Config struct {
	interpreter.Storage
	common.MemoryGauge
	commons.ImportHandler
	ContractValueHandler
	stdlib.AccountHandler

	// TODO: Move these to a 'shared state'?
	CapabilityControllerIterations              map[AddressPath]int
	MutationDuringCapabilityControllerIteration bool

	// TODO: These are temporary. Remove once storing/reading is supported for VM values.
	inter      *interpreter.Interpreter
	TypeLoader func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType
}

// TODO: This is temporary. Remove once storing/reading is supported for VM values.
func (c *Config) interpreter() *interpreter.Interpreter {
	if c.inter == nil {
		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage:               c.Storage,
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
