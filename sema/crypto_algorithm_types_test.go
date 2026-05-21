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

package sema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashAlgorithm_IsValid(t *testing.T) {
	for _, algorithm := range HashAlgorithms {
		require.True(t, algorithm.IsValid())
	}
}

func TestHashAlgorithmValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the HashAlgorithm enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[HashAlgorithm]uint8{
		HashAlgorithmUnknown:               0,
		HashAlgorithmSHA2_256:              1,
		HashAlgorithmSHA2_384:              2,
		HashAlgorithmSHA3_256:              3,
		HashAlgorithmSHA3_384:              4,
		HashAlgorithmKMAC128_BLS_BLS12_381: 5,
		HashAlgorithmKECCAK_256:            6,
		HashAlgorithm_Count:                7,
	}

	expectedRawValues := map[HashAlgorithm]uint8{
		HashAlgorithmUnknown:               0,
		HashAlgorithmSHA2_256:              1,
		HashAlgorithmSHA2_384:              2,
		HashAlgorithmSHA3_256:              3,
		HashAlgorithmSHA3_384:              4,
		HashAlgorithmKMAC128_BLS_BLS12_381: 5,
		HashAlgorithmKECCAK_256:            6,
	}

	// Check all expected values.
	for algo, expectedValue := range expectedValues {
		require.Equal(t, expectedValue, uint8(algo), "value mismatch for %s", algo)
	}

	// Check all expected raw values.
	for algo, expectedRawValue := range expectedRawValues {
		require.Equal(t, expectedRawValue, algo.RawValue(), "raw value mismatch for %s", algo)
	}

	// Check that no new named values have been added
	// without updating the expected values above.
	// NOTE: This requires the stringer-generated file to be up to date (CI runs go generate).
	for i := uint8(0); i < uint8(HashAlgorithm_Count); i++ {
		algo := HashAlgorithm(i)
		if _, ok := expectedValues[algo]; ok {
			continue
		}

		require.True(t,
			strings.HasPrefix(algo.String(), "HashAlgorithm("),
			"unexpected named value %s (%d): update expectedValues", algo, i,
		)
	}
}

func TestSignatureAlgorithmValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the SignatureAlgorithm enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[SignatureAlgorithm]uint8{
		SignatureAlgorithmUnknown:         0,
		SignatureAlgorithmECDSA_P256:      1,
		SignatureAlgorithmECDSA_secp256k1: 2,
		SignatureAlgorithmBLS_BLS12_381:   3,
		SignatureAlgorithm_Count:          4,
	}

	expectedRawValues := map[SignatureAlgorithm]uint8{
		SignatureAlgorithmUnknown:         0,
		SignatureAlgorithmECDSA_P256:      1,
		SignatureAlgorithmECDSA_secp256k1: 2,
		SignatureAlgorithmBLS_BLS12_381:   3,
	}

	// Check all expected values.
	for algo, expectedValue := range expectedValues {
		require.Equal(t, expectedValue, uint8(algo), "value mismatch for %s", algo)
	}

	// Check all expected raw values.
	for algo, expectedRawValue := range expectedRawValues {
		require.Equal(t, expectedRawValue, algo.RawValue(), "raw value mismatch for %s", algo)
	}

	// Check that no new named values have been added
	// without updating the expected values above.
	// NOTE: This requires the stringer-generated file to be up to date (CI runs go generate).
	for i := uint8(0); i < uint8(SignatureAlgorithm_Count); i++ {
		algo := SignatureAlgorithm(i)
		if _, ok := expectedValues[algo]; ok {
			continue
		}

		require.True(t,
			strings.HasPrefix(algo.String(), "SignatureAlgorithm("),
			"unexpected named value %s (%d): update expectedValues", algo, i,
		)
	}
}
