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

package sema_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckHashAlgorithmCases(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, value := range stdlib.InterpreterDefaultScriptStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(value)
	}

	test := func(algorithm sema.CryptoAlgorithm) {

		_, err := ParseAndCheckWithOptions(t,
			fmt.Sprintf(
				`
               let algo: HashAlgorithm = HashAlgorithm.%s
            `,
				algorithm.Name(),
			),
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
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

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.NewInterpreterHashAlgorithmConstructor(nil))

	_, err := ParseAndCheckWithOptions(t,
		`
           let algo = HashAlgorithm(rawValue: 0)
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckHashAlgorithmHashFunctions(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.NewInterpreterHashAlgorithmConstructor(nil))

	_, err := ParseAndCheckWithOptions(t,
		`
           let data: [UInt8] = [1, 2, 3]
           let result: [UInt8] = HashAlgorithm.SHA2_256.hash(data)
           let result2: [UInt8] = HashAlgorithm.SHA2_256.hashWithTag(data, tag: "tag")
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckSignatureAlgorithmCases(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InterpreterSignatureAlgorithmConstructor)

	test := func(algorithm sema.CryptoAlgorithm) {

		_, err := ParseAndCheckWithOptions(t,
			fmt.Sprintf(
				`
               let algo: SignatureAlgorithm = SignatureAlgorithm.%s
            `,
				algorithm.Name(),
			),
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
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

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InterpreterSignatureAlgorithmConstructor)

	_, err := ParseAndCheckWithOptions(t,
		`
           let algo = SignatureAlgorithm(rawValue: 0)
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckVerifyPoP(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, valueDeclaration := range stdlib.InterpreterDefaultStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	_, err := ParseAndCheckWithOptions(t,
		`
           let key = PublicKey(
              publicKey: "".decodeHex(),
              signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381)

           let x: Bool = key.verifyPoP([1, 2, 3])
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckVerifyPoPInvalidArgument(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, valueDeclaration := range stdlib.InterpreterDefaultStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	_, err := ParseAndCheckWithOptions(t,
		`
           let key = PublicKey(
              publicKey: "".decodeHex(),
              signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381)

           let x: Int = key.verifyPoP([1 as Int32, 2, 3])
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 2)
	var mismatch *sema.TypeMismatchError
	require.IsType(t, mismatch, errs[0])
	require.IsType(t, mismatch, errs[1])
}

func TestCheckBLSAggregateSignatures(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.NewBLSContract(nil, nil))

	_, err := ParseAndCheckWithOptions(t,
		`
           let r: [UInt8] = BLS.aggregateSignatures([[1 as UInt8, 2, 3], []])!
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckInvalidBLSAggregateSignatures(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.NewBLSContract(nil, nil))

	_, err := ParseAndCheckWithOptions(t,
		`
           let r: [UInt16] = BLS.aggregateSignatures([[1 as UInt32, 2, 3], []])!
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 2)
	var mismatch *sema.TypeMismatchError
	require.IsType(t, mismatch, errs[0])
	require.IsType(t, mismatch, errs[1])
}

func TestCheckBLSAggregatePublicKeys(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, valueDeclaration := range stdlib.InterpreterDefaultStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	_, err := ParseAndCheckWithOptions(t,
		`
           let r: PublicKey = BLS.aggregatePublicKeys([
               PublicKey(
                   publicKey: [],
                   signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381
               )
           ])!
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckInvalidBLSAggregatePublicKeys(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, valueDeclaration := range stdlib.InterpreterDefaultStandardLibraryValues(nil) {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	_, err := ParseAndCheckWithOptions(t,
		`
           let r: [PublicKey] = BLS.aggregatePublicKeys([1])!
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 2)
	var mismatch *sema.TypeMismatchError
	require.IsType(t, mismatch, errs[0])
	require.IsType(t, mismatch, errs[1])
}
