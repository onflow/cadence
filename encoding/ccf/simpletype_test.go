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

package ccf

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func TestTypeConversion(t *testing.T) {
	t.Parallel()

	// Missing types are not problematic,
	// just a missed optimization (more compact encoding)

	test := func(ty interpreter.PrimitiveStaticType, semaType sema.Type) {

		t.Run(semaType.QualifiedString(), func(t *testing.T) {

			t.Parallel()

			cadenceType := runtime.ExportType(semaType, map[sema.TypeID]cadence.Type{})

			simpleTypeID, ok := simpleTypeIDByType(cadenceType)
			require.True(t, ok)

			ty2 := typeBySimpleTypeID(simpleTypeID)
			require.Equal(t, cadence.PrimitiveType(ty), ty2)
		})
	}

	for ty := interpreter.PrimitiveStaticType(1); ty < interpreter.PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() {
			continue
		}

		semaType := ty.SemaType()

		// Some primitive static types are deprecated,
		// and only exist for migration purposes,
		// so do not have an equivalent sema type
		if semaType == nil {
			continue
		}

		if ty.IsDeprecated() {
			continue
		}

		if _, ok := semaType.(*sema.CapabilityType); ok {
			continue
		}

		test(ty, semaType)
	}
}
