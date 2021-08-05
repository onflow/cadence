/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package interpreter

import (
	"fmt"

	"github.com/fxamacker/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// DeployedContractValue

type DeployedContractValue struct {
	Address AddressValue
	Name    *StringValue
	Code    *ArrayValue
}

var _ Value = DeployedContractValue{}

func (DeployedContractValue) IsValue() {}

func (v DeployedContractValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitDeployedContractValue(interpreter, v)
}

func (v DeployedContractValue) Walk(walkChild func(Value)) {
	walkChild(v.Address)
	walkChild(v.Name)
	walkChild(v.Code)
}

var deployedContractDynamicType DynamicType = DeployedContractDynamicType{}

func (DeployedContractValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return deployedContractDynamicType
}

func (DeployedContractValue) StaticType() StaticType {
	return PrimitiveStaticTypeDeployedContract
}

func (DeployedContractValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (DeployedContractValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (v DeployedContractValue) Destroy(_ *Interpreter, _ func() LocationRange) {
	// NO-OP
}

func (v DeployedContractValue) String() string {
	return v.RecursiveString(StringResults{})
}

func (v DeployedContractValue) RecursiveString(results StringResults) string {
	return fmt.Sprintf(
		"DeployedContract(address: %s, name: %s, code: %s)",
		v.Address.RecursiveString(results),
		v.Name.RecursiveString(results),
		v.Code.RecursiveString(results),
	)
}

func (v DeployedContractValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case sema.DeployedContractTypeAddressFieldName:
		return v.Address

	case sema.DeployedContractTypeNameFieldName:
		return v.Name

	case sema.DeployedContractTypeCodeFieldName:
		return v.Code
	}

	return nil
}

func (DeployedContractValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v DeployedContractValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(DeployedContractDynamicType)
	return ok
}

func (DeployedContractValue) IsStorable() bool {
	return false
}

func (v DeployedContractValue) Storable(_ atree.SlabStorage) atree.Storable {
	return atree.NonStorable{Value: v}
}

func (v DeployedContractValue) DeepCopy(_ atree.SlabStorage, _ atree.Address) (atree.Value, error) {
	return v, nil
}
