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

package values

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCBORTagValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the CBORTag enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[CBORTag]byte{
		CBORTagVoidValue:                        128,
		CBORTagSomeValue:                        130,
		CBORTagAddressValue:                     131,
		CBORTagCompositeValue:                   132,
		CBORTagTypeValue:                        133,
		CBORTagStringValue:                      135,
		CBORTagCharacterValue:                   136,
		CBORTagSomeValueWithNestedLevels:        137,
		CBORTagIntValue:                         152,
		CBORTagInt8Value:                        153,
		CBORTagInt16Value:                       154,
		CBORTagInt32Value:                       155,
		CBORTagInt64Value:                       156,
		CBORTagInt128Value:                      157,
		CBORTagInt256Value:                      158,
		CBORTagUIntValue:                        160,
		CBORTagUInt8Value:                       161,
		CBORTagUInt16Value:                      162,
		CBORTagUInt32Value:                      163,
		CBORTagUInt64Value:                      164,
		CBORTagUInt128Value:                     165,
		CBORTagUInt256Value:                     166,
		CBORTagWord8Value:                       169,
		CBORTagWord16Value:                      170,
		CBORTagWord32Value:                      171,
		CBORTagWord64Value:                      172,
		CBORTagWord128Value:                     173,
		CBORTagWord256Value:                     174,
		CBORTagFix64Value:                       180,
		CBORTagFix128Value:                      181,
		CBORTagUFix64Value:                      188,
		CBORTagUFix128Value:                     189,
		CBORTagAddressLocation:                  192,
		CBORTagStringLocation:                   193,
		CBORTagIdentifierLocation:               194,
		CBORTagTransactionLocation:              195,
		CBORTagScriptLocation:                   196,
		CBORTagPathValue:                        200,
		CBORTagPathCapabilityValue:              201,
		CBORTagPathLinkValue:                    203,
		CBORTagPublishedValue:                   204,
		CBORTagAccountLinkValue:                 205,
		CBORTagStorageCapabilityControllerValue: 206,
		CBORTagAccountCapabilityControllerValue: 207,
		CBORTagCapabilityValue:                  208,
		CBORTagPrimitiveStaticType:              212,
		CBORTagCompositeStaticType:              213,
		CBORTagInterfaceStaticType:              214,
		CBORTagVariableSizedStaticType:          215,
		CBORTagConstantSizedStaticType:          216,
		CBORTagDictionaryStaticType:             217,
		CBORTagOptionalStaticType:               218,
		CBORTagReferenceStaticType:              219,
		CBORTagIntersectionStaticType:           220,
		CBORTagCapabilityStaticType:             221,
		CBORTagUnauthorizedStaticAuthorization:  222,
		CBORTagEntitlementMapStaticAuthorization:  223,
		CBORTagEntitlementSetStaticAuthorization:  224,
		CBORTagInaccessibleStaticAuthorization:    225,
		CBORTagInclusiveRangeStaticType:           230,
		CBORTag_Count:                             231,
	}

	// Check all expected values.
	for tag, expectedValue := range expectedValues {
		require.Equal(t, expectedValue, byte(tag), "value mismatch for %s", tag)
	}

	// Check that no new named values have been added
	// without updating the expected values above.
	// If a placeholder `_` is replaced with a new named value,
	// its String() representation will no longer be a numeric fallback.
	// NOTE: This requires the stringer-generated file to be up to date (CI runs go generate).
	for i := byte(CBORTagBase); i < byte(CBORTag_Count); i++ {
		tag := CBORTag(i)
		if _, ok := expectedValues[tag]; ok {
			continue
		}
		if !strings.HasPrefix(tag.String(), "CBORTag(") {
			t.Errorf("unexpected named value %s (%d): update expectedValues", tag, i)
		}
	}
}
