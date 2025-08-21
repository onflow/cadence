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

package commons

import "github.com/onflow/cadence/ast"

const (
	InitFunctionName                = "init"
	ExecuteFunctionName             = "execute"
	TransactionWrapperCompositeName = "transaction"
	TransactionExecuteFunctionName  = "transaction.execute"
	TransactionPrepareFunctionName  = "transaction.prepare"

	// FailPreConditionFunctionName is the name of the function which is used for failing pre-conditions
	FailPreConditionFunctionName = "$failPreCondition"
	// FailPostConditionFunctionName is the name  of the function which is used for failing post-conditions
	FailPostConditionFunctionName = "$failPostCondition"

	GeneratedNameQualifier              = "$"
	ResourceDestroyedEventsFunctionName = GeneratedNameQualifier + ast.ResourceDestructionDefaultEventName

	// Names used by generated constructs

	ProgramInitFunctionName         = "$_init_"
	TransactionGeneratedParamPrefix = "$_param_"

	// Type qualifiers for built-in member functions

	TypeQualifierArrayConstantSized = "$ArrayConstantSized"
	TypeQualifierArrayVariableSized = "$ArrayVariableSized"
	TypeQualifierDictionary         = "$Dictionary"
	TypeQualifierFunction           = "$Function"
	TypeQualifierOptional           = "$Optional"
	TypeQualifierInclusiveRange     = "$InclusiveRange"
)
