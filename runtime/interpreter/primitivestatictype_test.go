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

package interpreter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrimitiveStaticTypeSemaTypeConversion(t *testing.T) {

	t.Parallel()

	test := func(ty PrimitiveStaticType) {
		if ty.IsDeprecated() { //nolint:staticcheck
			return
		}

		t.Run(ty.String(), func(t *testing.T) {
			t.Parallel()

			semaType := ty.SemaType()

			ty2 := ConvertSemaToPrimitiveStaticType(nil, semaType)
			require.True(t, ty2.Equal(ty))
		})
	}

	for ty := PrimitiveStaticType(1); ty < PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() || ty.IsDeprecated() { //nolint:staticcheck
			continue
		}
		test(ty)
	}
}

func TestPrimitiveStaticType_elementSize(t *testing.T) {

	t.Parallel()

	test := func(ty PrimitiveStaticType) {
		t.Run(ty.String(), func(t *testing.T) {
			t.Parallel()

			_ = ty.elementSize()
		})
	}

	for ty := PrimitiveStaticType(1); ty < PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() || ty.IsDeprecated() { //nolint:staticcheck
			continue
		}
		test(ty)
	}
}
