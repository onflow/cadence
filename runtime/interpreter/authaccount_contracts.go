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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/trampoline"
)

// AuthAccountContractsValue

type AuthAccountContractsValue struct {
	Address        AddressValue
	AddFunction    FunctionValue
	UpdateFunction FunctionValue
	GetFunction    FunctionValue
	RemoveFunction FunctionValue
}

func (AuthAccountContractsValue) IsValue() {}

func (v AuthAccountContractsValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitAuthAccountContractsValue(interpreter, v)
}

func (AuthAccountContractsValue) DynamicType(_ *Interpreter) DynamicType {
	return AuthAccountContractsDynamicType{}
}

func (v AuthAccountContractsValue) Copy() Value {
	return v
}

func (AuthAccountContractsValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (AuthAccountContractsValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (AuthAccountContractsValue) IsModified() bool {
	return false
}

func (AuthAccountContractsValue) SetModified(_ bool) {
	// NO-OP
}

func (v AuthAccountContractsValue) Destroy(_ *Interpreter, _ LocationRange) trampoline.Trampoline {
	return trampoline.Done{}
}

func (v AuthAccountContractsValue) String() string {
	return fmt.Sprintf("AuthAccount.Contracts(%s)", v.Address)
}

func (v AuthAccountContractsValue) GetMember(_ *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.AuthAccountContractsTypeAddFunctionName:
		return v.AddFunction
	case sema.AuthAccountContractsTypeGetFunctionName:
		return v.GetFunction
	case sema.AuthAccountContractsTypeRemoveFunctionName:
		return v.RemoveFunction
	case sema.AuthAccountContractsTypeUpdateExperimentalFunctionName:
		return v.UpdateFunction
	}

	return nil
}

func (AuthAccountContractsValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}
