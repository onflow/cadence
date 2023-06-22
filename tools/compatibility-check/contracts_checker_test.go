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

package compatibility_check

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCyclicImport(t *testing.T) {

	var output bytes.Buffer
	var input bytes.Buffer

	checker := NewContractChecker(&output)

	input.Write([]byte(`location,code
A.0000000000000001.Foo,"import Bar from 0x0000000000000001
access(all) contract Foo {}"
A.0000000000000001.Bar,"import Baz from 0x0000000000000001
access(all) contract Foo {}"
A.0000000000000001.Baz,"import Foo from 0x0000000000000001
access(all) contract Foo {}"
`))

	checker.CheckCSV(&input)

	outputStr := output.String()

	assert.Contains(t, outputStr, "Foo:16(1:16):*sema.ImportedProgramError")
	assert.Contains(t, outputStr, "Bar:16(1:16):*sema.ImportedProgramError")
	assert.Contains(t, outputStr, "Baz:16(1:16):*sema.ImportedProgramError")
}
