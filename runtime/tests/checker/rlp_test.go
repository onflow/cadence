/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

func TestCheckRLPDecodeString(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.RLPContract)

	_, err := ParseAndCheckWithOptions(t,
		`
           let l: [UInt8] = RLP.decodeString([0, 1, 2])
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithBaseValueActivation(baseValueActivation),
			},
		},
	)
	require.NoError(t, err)
}

func TestCheckInvalidRLPDecodeString(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.RLPContract)

	_, err := ParseAndCheckWithOptions(t,
		`
           let l: String = RLP.decodeString("string")
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithBaseValueActivation(baseValueActivation),
			},
		},
	)

	errs := ExpectCheckerErrors(t, err, 2)
	var mismatch *sema.TypeMismatchError
	require.IsType(t, mismatch, errs[0])
	require.IsType(t, mismatch, errs[1])
}

func TestCheckRLPDecodeList(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.RLPContract)

	_, err := ParseAndCheckWithOptions(t,
		`
           let l: [[UInt8]] = RLP.decodeList([0, 1, 2])
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithBaseValueActivation(baseValueActivation),
			},
		},
	)
	require.NoError(t, err)
}

func TestCheckInvalidRLPDecodeList(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.RLPContract)

	_, err := ParseAndCheckWithOptions(t,
		`
           let l: String = RLP.decodeList("string")
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithBaseValueActivation(baseValueActivation),
			},
		},
	)

	errs := ExpectCheckerErrors(t, err, 2)
	var mismatch *sema.TypeMismatchError
	require.IsType(t, mismatch, errs[0])
	require.IsType(t, mismatch, errs[1])
}
