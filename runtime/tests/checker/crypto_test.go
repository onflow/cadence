/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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
	"fmt"
	"testing"

	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/stretchr/testify/require"
)

func TestCheckHashAlgorithmCases(t *testing.T) {

	t.Parallel()

	test := func(algorithm sema.CryptoAlgorithm) {

		_, err := ParseAndCheckWithOptions(t,
			fmt.Sprintf(
				`
               let algo: HashAlgorithm = HashAlgorithm.%s
            `,
				algorithm.Name(),
			),
			ParseAndCheckOptions{
				Options: []sema.Option{
					sema.WithPredeclaredValues(
						stdlib.BuiltinValues().ToSemaValueDeclarations(),
					),
				},
			},
		)

		require.NoError(t, err)
	}

	for _, algo := range sema.HashAlgorithms {
		test(algo)
	}
}

func TestCheckHashAlgorithmConstructor(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
           let algo = HashAlgorithm(rawValue: 0)
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(
					stdlib.BuiltinValues().ToSemaValueDeclarations(),
				),
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckHashAlgorithmHashFunctions(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
           let data: [UInt8] = [1, 2, 3]
           let result: [UInt8] = HashAlgorithm.SHA2_256.hash(data)
           let result2: [UInt8] = HashAlgorithm.SHA2_256.hashWithTag(data, tag: "tag")
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(
					stdlib.BuiltinValues().ToSemaValueDeclarations(),
				),
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckSignatureAlgorithmCases(t *testing.T) {

	t.Parallel()

	test := func(algorithm sema.CryptoAlgorithm) {

		_, err := ParseAndCheckWithOptions(t,
			fmt.Sprintf(
				`
               let algo: SignatureAlgorithm = SignatureAlgorithm.%s
            `,
				algorithm.Name(),
			),
			ParseAndCheckOptions{
				Options: []sema.Option{
					sema.WithPredeclaredValues(
						stdlib.BuiltinValues().ToSemaValueDeclarations(),
					),
				},
			},
		)

		require.NoError(t, err)
	}

	for _, algo := range sema.SignatureAlgorithms {
		test(algo)
	}
}

func TestCheckSignatureAlgorithmConstructor(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
           let algo = SignatureAlgorithm(rawValue: 0)
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(
					stdlib.BuiltinValues().ToSemaValueDeclarations(),
				),
			},
		},
	)

	require.NoError(t, err)
}
