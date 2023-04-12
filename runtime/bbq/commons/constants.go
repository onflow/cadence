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

package commons

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
)

const (
	InitFunctionName                = "init"
	TransactionWrapperCompositeName = "transaction"
	TransactionExecuteFunctionName  = "transaction.execute"
	TransactionPrepareFunctionName  = "transaction.prepare"
	LogFunctionName                 = "log"
	PanicFunctionName               = "panic"
)

type CastType byte

const (
	SimpleCast CastType = iota
	FailableCast
	ForceCast
)

func CastTypeFrom(operation ast.Operation) CastType {
	switch operation {
	case ast.OperationCast:
		return SimpleCast
	case ast.OperationFailableCast:
		return FailableCast
	case ast.OperationForceCast:
		return ForceCast
	default:
		panic(errors.NewUnreachableError())
	}
}
