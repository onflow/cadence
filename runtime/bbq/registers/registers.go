/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package registers

import (
	"fmt"
	"github.com/onflow/cadence/runtime/sema"
)

type RegisterCounts struct {
	Ints  uint16
	Bools uint16
	Funcs uint16
}

func (c *RegisterCounts) NextIndex(registryType RegisterType) (index uint16) {
	switch registryType {
	case Int:
		index = c.Ints
		c.Ints++
	case Bool:
		index = c.Bools
		c.Bools++
	case Func:
		index = c.Funcs
		c.Funcs++
	default:
		panic(fmt.Errorf("unknown register type '%s'", registryType))
	}

	return
}

type RegisterType int8

const (
	Int = iota
	Bool
	Func
)

func RegistryTypeFromSemaType(semaType sema.Type) RegisterType {
	switch semaType := semaType.(type) {
	case *sema.NumericType:
		switch semaType {
		case sema.IntType:
			return Int
		}
	case *sema.SimpleType:
		switch semaType {
		case sema.BoolType:
			return Bool
		}
	case *sema.FunctionType:
		return Func
	}

	panic("Unknown registry type")
}
