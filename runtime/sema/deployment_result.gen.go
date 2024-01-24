// Code generated from deployment_result.cdc. DO NOT EDIT.
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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

const DeploymentResultTypeDeployedContractFieldName = "deployedContract"

var DeploymentResultTypeDeployedContractFieldType = &OptionalType{
	Type: DeployedContractType,
}

const DeploymentResultTypeDeployedContractFieldDocString = `
The deployed contract.

If the the deployment was unsuccessful, this will be nil.
`

const DeploymentResultTypeName = "DeploymentResult"

var DeploymentResultType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         DeploymentResultTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			DeploymentResultType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			DeploymentResultTypeDeployedContractFieldName,
			DeploymentResultTypeDeployedContractFieldType,
			DeploymentResultTypeDeployedContractFieldDocString,
		),
	}

	DeploymentResultType.Members = MembersAsMap(members)
	DeploymentResultType.Fields = MembersFieldNames(members)
}
