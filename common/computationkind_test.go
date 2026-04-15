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

package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputationKindValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the ComputationKind enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[ComputationKind]uint{
		ComputationKindUnknown:                          0,
		ComputationKindStatement:                        1001,
		ComputationKindLoop:                             1002,
		ComputationKindFunctionInvocation:               1003,
		ComputationKindCreateCompositeValue:             1010,
		ComputationKindTransferCompositeValue:           1011,
		ComputationKindDestroyCompositeValue:            1012,
		ComputationKindCreateArrayValue:                 1025,
		ComputationKindTransferArrayValue:               1026,
		ComputationKindDestroyArrayValue:                1027,
		ComputationKindCreateDictionaryValue:            1040,
		ComputationKindTransferDictionaryValue:          1041,
		ComputationKindDestroyDictionaryValue:           1042,
		ComputationKindStringToLower:                    1055,
		ComputationKindStringDecodeHex:                  1056,
		ComputationKindGraphemesIteration:               1057,
		ComputationKindStringComparison:                 1058,
		ComputationKindEncodeValue:                      1080,
		ComputationKindWordSliceOperation:               1081,
		ComputationKindUintParse:                        1082,
		ComputationKindIntParse:                         1083,
		ComputationKindBigIntParse:                      1084,
		ComputationKindUfixParse:                        1085,
		ComputationKindFixParse:                         1086,
		ComputationKindSTDLIBPanic:                      1100,
		ComputationKindSTDLIBAssert:                     1101,
		ComputationKindSTDLIBRevertibleRandom:           1102,
		ComputationKindSTDLIBRLPDecodeString:            1108,
		ComputationKindSTDLIBRLPDecodeList:              1109,
		ComputationKindAtreeArraySingleSlabConstruction: 1200,
		ComputationKindAtreeArrayBatchConstruction:      1201,
		ComputationKindAtreeArrayGet:                    1202,
		ComputationKindAtreeArraySet:                    1203,
		ComputationKindAtreeArrayAppend:                 1204,
		ComputationKindAtreeArrayInsert:                 1205,
		ComputationKindAtreeArrayRemove:                 1206,
		ComputationKindAtreeArrayReadIteration:          1207,
		ComputationKindAtreeArrayPopIteration:           1208,
		ComputationKindAtreeMapConstruction:             1220,
		ComputationKindAtreeMapSingleSlabConstruction:   1221,
		ComputationKindAtreeMapBatchConstruction:        1222,
		ComputationKindAtreeMapHas:                      1223,
		ComputationKindAtreeMapGet:                      1224,
		ComputationKindAtreeMapSet:                      1225,
		ComputationKindAtreeMapRemove:                   1226,
		ComputationKindAtreeMapReadIteration:            1227,
		ComputationKindAtreeMapPopIteration:             1228,
		ComputationKind_Count:                           1229,
	}

	// Check all expected values.
	for kind, expectedValue := range expectedValues {
		require.Equal(t, expectedValue, uint(kind), "value mismatch for %s", kind)
	}

	// Check that no new named values have been added
	// without updating the expected values above.
	// If a placeholder `_` is replaced with a new named value,
	// its String() representation will no longer be a numeric fallback.
	// NOTE: This requires the stringer-generated file to be up to date (CI runs go generate).
	//
	// ComputationKindUnknown = 0, then the rest start at ComputationKindRangeStart (1000).
	for i := uint(0); i < uint(ComputationKind_Count); i++ {
		// Skip the gap between 0 and 1001.
		if i > 0 && i < ComputationKindRangeStart {
			continue
		}

		kind := ComputationKind(i)
		if _, ok := expectedValues[kind]; ok {
			continue
		}

		require.True(t,
			strings.HasPrefix(kind.String(), "ComputationKind("),
			"unexpected named value %s (%d): update expectedValues", kind, i,
		)
	}
}
