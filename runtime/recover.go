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

package runtime

import (
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

func Recover(onError func(Error), location Location, codesAndPrograms CodesAndPrograms) {
	recovered := recover()
	if recovered == nil {
		return
	}

	err := GetWrappedError(recovered, location, codesAndPrograms)
	onError(err)
}

func GetWrappedError(recovered any, location Location, codesAndPrograms CodesAndPrograms) Error {
	switch recovered := recovered.(type) {

	// If the error is already a `runtime.Error`, then avoid redundant wrapping.
	case Error:
		return recovered

	// Wrap with `runtime.Error` to include meta info.
	//
	// The following set of errors are the only known types of errors that would reach this point.
	// `interpreter.Error` is a generic wrapper for any error. Hence, it doesn't belong to any of the
	// three types: `UserError`, `InternalError`, `ExternalError`.
	// So it needs to be specially handled here
	case errors.InternalError,
		errors.UserError,
		errors.ExternalError,
		interpreter.Error:
		return newError(recovered.(error), location, codesAndPrograms)

	// Wrap any other unhandled error with a generic internal error first.
	// And then wrap with `runtime.Error` to include meta info.
	case error:
		err := errors.NewUnexpectedErrorFromCause(recovered)
		return newError(err, location, codesAndPrograms)
	default:
		err := errors.NewUnexpectedError("%s", recovered)
		return newError(err, location, codesAndPrograms)
	}
}
