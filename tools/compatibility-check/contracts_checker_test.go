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

package compatibility_check

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/model/flow"
)

func TestCyclicImport(t *testing.T) {

	t.Parallel()

	var output bytes.Buffer
	var input bytes.Buffer

	chain := flow.Testnet.Chain()

	checker := NewContractChecker(chain, &output)

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

func TestCryptoImport(t *testing.T) {

	t.Parallel()

	var output bytes.Buffer
	var input bytes.Buffer

	chainID := flow.Testnet

	checker := NewContractChecker(chainID.Chain(), &output)

	contractsCSV := `location,code
A.0000000000000001.Foo,"import Crypto
access(all) contract Foo {}"
A.0000000000000001.Bar,"import Crypto
access(all) contract Bar {}"
`

	input.Write([]byte(contractsCSV))

	checker.CheckCSV(&input)

	outputStr := output.String()

	assert.Empty(t, outputStr)
}
