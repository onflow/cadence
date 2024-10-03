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
		let a: {StructStringer} = 1
		let b: {StructStringer} = false
		let c: {StructStringer} = "hey"
		access(all) 
		struct Foo: StructStringer {
			view fun toString():String {
				return "foo"
			}
		}
		let d: {StructStringer} = Foo()
		let e: {StructStringer} = /public/foo
	  `)

	assert.NoError(t, err)
}

func TestCheckInvalidStringer(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
		resource R {}
  
		let a: {StructStringer} = <-create R()
		let b: {StructStringer} = [<-create R()]
		let c: {StructStringer} = {1: true}
		struct Foo: StructStringer {}
	  `)

	errs := RequireCheckerErrors(t, err, 4)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	assert.IsType(t, &sema.ConformanceError{}, errs[3])
}
