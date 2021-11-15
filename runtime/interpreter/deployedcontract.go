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
	"github.com/onflow/cadence/runtime/sema"
)

// DeployedContractValue

var deployedContractDynamicType DynamicType = DeployedContractDynamicType{}
var deployedContractStaticType StaticType = PrimitiveStaticTypeDeployedContract
var deployedContractFieldNames = []string{
	sema.DeployedContractTypeAddressFieldName,
	sema.DeployedContractTypeNameFieldName,
	sema.DeployedContractTypeCodeFieldName,
}

func NewDeployedContractValue(
	address AddressValue,
	name *StringValue,
	code *ArrayValue,
) *SimpleCompositeValue {
	return NewSimpleCompositeValue(
		sema.DeployedContractType.TypeID,
		deployedContractStaticType,
		deployedContractDynamicType,
		deployedContractFieldNames,
		map[string]Value{
			sema.DeployedContractTypeAddressFieldName: address,
			sema.DeployedContractTypeNameFieldName:    name,
			sema.DeployedContractTypeCodeFieldName:    code,
		},
		nil,
		nil,
		nil,
	)
}
