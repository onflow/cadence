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
	"github.com/onflow/cadence/runtime/trampoline"
)

// ContractValue

type ContractValue struct {
	Name *StringValue
	Code *ArrayValue
}

func (ContractValue) IsValue() {}

func (ContractValue) DynamicType(_ *Interpreter) DynamicType {
	return ContractDynamicType{}
}

func (v ContractValue) Copy() Value {
	return v
}

func (ContractValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (ContractValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (ContractValue) IsModified() bool {
	return false
}

func (ContractValue) SetModified(_ bool) {
	// NO-OP
}

func (v ContractValue) Destroy(_ *Interpreter, _ LocationRange) trampoline.Trampoline {
	return trampoline.Done{}
}

func (v ContractValue) String() string {
	return fmt.Sprintf(
		"Contract(name: %s, code: %s)",
		v.Name,
		v.Code,
	)
}

func (v ContractValue) GetMember(_ *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case "name":
		return v.Name

	case "code":
		return v.Code
	}

	return nil
}

func (ContractValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}
