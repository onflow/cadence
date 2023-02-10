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

package ast

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableKind_MarshalJSON(t *testing.T) {

	t.Parallel()

	for variableKind := VariableKind(0); variableKind < VariableKind(VariableKindCount()); variableKind++ {
		actual, err := json.Marshal(variableKind)
		require.NoError(t, err)

		assert.JSONEq(t, fmt.Sprintf(`"%s"`, variableKind), string(actual))
	}
}
