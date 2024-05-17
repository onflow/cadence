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
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// members

type CapabilityValue struct {
	Address    AddressValue
	Path       PathValue
	BorrowType StaticType
}

var _ Value = CapabilityValue{}

func NewCapabilityValue(address AddressValue, path PathValue, borrowType StaticType) CapabilityValue {
	return CapabilityValue{
		Address:    address,
		Path:       path,
		BorrowType: borrowType,
	}
}

func (CapabilityValue) isValue() {}

func (v CapabilityValue) StaticType(gauge common.MemoryGauge) StaticType {
	return interpreter.NewCapabilityStaticType(gauge, v.BorrowType)
}

func (v CapabilityValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v CapabilityValue) String() string {
	var borrowType string
	if v.BorrowType != nil {
		borrowType = v.BorrowType.String()
	}
	return format.Capability(
		borrowType,
		v.Address.String(),
		v.Path.String(),
	)
}

func init() {
	typeName := interpreter.PrimitiveStaticTypeCapability.String()

	// Capability.borrow
	RegisterTypeBoundFunction(typeName, sema.CapabilityTypeBorrowField, NativeFunctionValue{
		ParameterCount: len(sema.StringTypeConcatFunctionType.Parameters),
		Function: func(config *Config, typeArguments []StaticType, value ...Value) Value {
			// TODO:
			return NilValue{}
		},
	})
}
