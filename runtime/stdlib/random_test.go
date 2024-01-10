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

package stdlib

import (
	"crypto/rand"
	"strconv"
	"testing"

	"github.com/onflow/crypto/random"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type testCryptRandomGenerator struct{}

var _ RandomGenerator = testCryptRandomGenerator{}

func (t testCryptRandomGenerator) ReadRandom(buffer []byte) error {
	_, err := rand.Read(buffer)
	return err
}

// TestRandomBasicUniformityWithModulo is a sanity statistical test
// to make sure the random numbers less than modulo are uniform in [0,modulo-1].
// The test requires the original random source (here `crypto/rand`) to be uniform.
// The test uses the same small values for all types:
// one is a power of 2 and the other is not.
func TestRandomBasicUniformityWithModulo(t *testing.T) {

	t.Parallel()

	if testing.Short() {
		// skipped because the test is slow
		t.Skip()
	}

	testTypes := func(t *testing.T, testType func(*testing.T, sema.Type)) {
		for _, ty := range sema.AllFixedSizeUnsignedIntegerTypes {
			tyCopy := ty
			t.Run(ty.String(), func(t *testing.T) {
				t.Parallel()

				testType(t, tyCopy)
			})
		}
	}

	// dummy interpreter, just use for ConvertAndBox
	inter := newInterpreter(t, ``)

	runStatisticsWithModulo := func(modulo int) func(*testing.T, sema.Type) {
		return func(t *testing.T, ty sema.Type) {
			// make sure modulo fits in 8 bits
			require.Less(t, modulo, 1<<8)

			moduloValue := inter.ConvertAndBox(
				interpreter.EmptyLocationRange,
				interpreter.NewUnmeteredUIntValueFromUint64(uint64(modulo)),
				sema.UIntType,
				ty,
			)

			f := func() (uint64, error) {

				value := RevertibleRandom(
					testCryptRandomGenerator{},
					nil,
					ty,
					moduloValue,
				)

				return strconv.ParseUint(value.String(), 10, 8)
			}

			random.BasicDistributionTest(t, uint64(modulo), 1, f)
		}
	}

	t.Run("power of 2 (that fits in 8 bits)", func(t *testing.T) {
		t.Parallel()

		testTypes(t, runStatisticsWithModulo(64))
	})

	t.Run("non-power of 2", func(t *testing.T) {
		t.Parallel()

		testTypes(t, runStatisticsWithModulo(71))
	})
}
