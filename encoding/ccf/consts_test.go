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

package ccf

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCCFCBORTagValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the CCF CBORTag enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[CBORTag]byte{
		// Root objects
		CBORTagTypeDef:         128,
		CBORTagTypeDefAndValue: 129,
		CBORTagTypeAndValue:    130,
		// Inline types
		CBORTagTypeRef:                              136,
		CBORTagSimpleType:                           137,
		CBORTagOptionalType:                         138,
		CBORTagVarsizedArrayType:                    139,
		CBORTagConstsizedArrayType:                  140,
		CBORTagDictType:                             141,
		CBORTagReferenceType:                        142,
		CBORTagIntersectionType:                     143,
		CBORTagCapabilityType:                       144,
		CBORTagInclusiveRangeType:                   145,
		CBORTagEntitlementSetAuthorizationAccessType: 146,
		CBORTagEntitlementMapAuthorizationAccessType: 147,
		// Composite types
		CBORTagStructType:     160,
		CBORTagResourceType:   161,
		CBORTagEventType:      162,
		CBORTagContractType:   163,
		CBORTagEnumType:       164,
		CBORTagAttachmentType: 165,
		// Interface types
		CBORTagStructInterfaceType:   176,
		CBORTagResourceInterfaceType: 177,
		CBORTagContractInterfaceType: 178,
		// Non-composite/non-interface type values
		CBORTagTypeValueRef:                                184,
		CBORTagSimpleTypeValue:                             185,
		CBORTagOptionalTypeValue:                           186,
		CBORTagVarsizedArrayTypeValue:                      187,
		CBORTagConstsizedArrayTypeValue:                    188,
		CBORTagDictTypeValue:                               189,
		CBORTagReferenceTypeValue:                          190,
		CBORTagIntersectionTypeValue:                       191,
		CBORTagCapabilityTypeValue:                         192,
		CBORTagFunctionTypeValue:                           193,
		CBORTagInclusiveRangeTypeValue:                     194,
		CBORTagEntitlementSetAuthorizationAccessTypeValue:  195,
		CBORTagEntitlementMapAuthorizationAccessTypeValue:  196,
		// Composite type values
		CBORTagStructTypeValue:     208,
		CBORTagResourceTypeValue:   209,
		CBORTagEventTypeValue:      210,
		CBORTagContractTypeValue:   211,
		CBORTagEnumTypeValue:       212,
		CBORTagAttachmentTypeValue: 213,
		// Interface type values
		CBORTagStructInterfaceTypeValue:   224,
		CBORTagResourceInterfaceTypeValue: 225,
		CBORTagContractInterfaceTypeValue: 226,
		CBORTag_Count:                     232,
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
