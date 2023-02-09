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

package errors

import (
	goRuntime "runtime"
)

func WrapPanic(f func()) {
	defer func() {
		if r := recover(); r != nil {
			// don't wrap Go errors and internal errors
			switch r := r.(type) {
			case goRuntime.Error, InternalError:
				panic(r)
			case error:
				panic(ExternalError{
					Recovered: r,
				})
			default:
				panic(ExternalNonError{
					Recovered: r,
				})
			}
		}
	}()
	f()
}
