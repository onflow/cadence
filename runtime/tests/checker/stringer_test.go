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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckStringer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		let a: {Stringer} = 1
		let b: {Stringer} = false
		let c: {Stringer} = "hey"
		access(all) 
		struct Foo: Stringer {
			view fun toString():String {
				return "foo"
			}
		}
		let d: {Stringer} = Foo()
	  `)

	assert.NoError(t, err)
}

func TestCheckInvalidStringer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		resource R {}
  
		let a: {Stringer} = <-create R()
		let b: {Stringer} = [<-create R()]
		let c: {Stringer} = {1: true}
	  `)

	errs := RequireCheckerErrors(t, err, 3)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
}
