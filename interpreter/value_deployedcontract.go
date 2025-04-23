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

package interpreter

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

// DeployedContractValue

var deployedContractStaticType StaticType = PrimitiveStaticTypeDeployedContract // unmetered
var deployedContractFieldNames = []string{
	sema.DeployedContractTypeAddressFieldName,
	sema.DeployedContractTypeNameFieldName,
	sema.DeployedContractTypeCodeFieldName,
	sema.DeployedContractTypePublicTypesFunctionName,
}

func NewDeployedContractValue(
	inter *Interpreter,
	address AddressValue,
	name *StringValue,
	code *ArrayValue,
) *SimpleCompositeValue {
	deployedContract := NewSimpleCompositeValue(
		inter,
		sema.DeployedContractType.TypeID,
		deployedContractStaticType,
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

	publicTypesFuncValue := newPublicTypesFunctionValue(
		inter,
		deployedContract,
		address,
		name,
	)
	deployedContract.Fields[sema.DeployedContractTypePublicTypesFunctionName] = publicTypesFuncValue

	return deployedContract
}

func newPublicTypesFunctionValue(
	inter *Interpreter,
	self MemberAccessibleValue,
	addressValue AddressValue,
	name *StringValue,
) FunctionValue {
	// public types only need to be computed once per contract
	var publicTypes *ArrayValue

	address := addressValue.ToAddress()
	return NewBoundHostFunctionValue(
		inter,
		self,
		sema.DeployedContractTypePublicTypesFunctionType,
		func(_ MemberAccessibleValue, inv Invocation) Value {
			if publicTypes == nil {
				innerInter := inv.Interpreter
				contractLocation := common.NewAddressLocation(innerInter, address, name.Str)
				// we're only looking at the contract as a whole, so no need to construct a nested path
				qualifiedIdent := name.Str
				typeID := common.NewTypeIDFromQualifiedName(innerInter, contractLocation, qualifiedIdent)
				compositeType, err := innerInter.GetCompositeType(contractLocation, qualifiedIdent, typeID)
				if err != nil {
					panic(err)
				}

				nestedTypes := compositeType.NestedTypes
				pair := nestedTypes.Oldest()
				// all top-level type declarations in a contract must be public
				// no need to filter here for public visibility
				yieldNext := func() Value {
					if pair == nil {
						return nil
					}
					typeValue := NewTypeValue(innerInter, ConvertSemaToStaticType(innerInter, pair.Value))
					pair = pair.Next()
					return typeValue
				}

				publicTypes = NewArrayValueWithIterator(
					innerInter,
					NewVariableSizedStaticType(innerInter, PrimitiveStaticTypeMetaType),
					common.Address{},
					uint64(nestedTypes.Len()),
					yieldNext,
				)
			}

			return publicTypes
		})
}
