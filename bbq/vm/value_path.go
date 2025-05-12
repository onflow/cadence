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

package vm

import (
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {

	for _, pathType := range []sema.Type{
		sema.PathType,
		sema.StoragePathType,
		sema.CapabilityPathType,
		sema.PublicPathType,
		sema.PrivatePathType,
	} {
		typeName := commons.TypeQualifier(pathType)

		RegisterTypeBoundFunction(
			typeName,
			NewNativeFunctionValue(
				sema.ToStringFunctionName,
				sema.ToStringFunctionType,
				func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
					address := arguments[receiverIndex].(interpreter.PathValue)
					return interpreter.PathValueToStringFunction(
						context,
						address,
						EmptyLocationRange,
					)
				},
			),
		)

	}
}
