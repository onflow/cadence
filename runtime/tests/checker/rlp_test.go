/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestCheckRLPDecodeStringError(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
           var l = DecodeRLPString(input: "string")  
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(stdlib.BuiltinFunctions.ToSemaValueDeclarations()),
				sema.WithPredeclaredValues(stdlib.BuiltinValues.ToSemaValueDeclarations()),
			},
		},
	)

	errs := ExpectCheckerErrors(t, err, 1)
	var mismatch *sema.TypeMismatchError
	require.IsType(t, mismatch, errs[0])
}

func TestCheckRLPDecodeListError(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
           var l = DecodeRLPList(input: "string")  
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(stdlib.BuiltinFunctions.ToSemaValueDeclarations()),
				sema.WithPredeclaredValues(stdlib.BuiltinValues.ToSemaValueDeclarations()),
			},
		},
	)

	errs := ExpectCheckerErrors(t, err, 1)
	var mismatch *sema.TypeMismatchError
	require.IsType(t, mismatch, errs[0])
}
