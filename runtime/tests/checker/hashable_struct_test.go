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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckHashableStruct(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	   let a: HashableStruct = 1
	   let b: HashableStruct = true
	 `)

	assert.NoError(t, err)
}

func TestCheckInvalidHashableStruct(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	   resource R {}
 
	   let a: HashableStruct = <-create R()
	   let b: HashableStruct = [<-create R()]
	   let c: HashableStruct = {1: true}
	 `)

	errs := RequireCheckerErrors(t, err, 3)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
}
