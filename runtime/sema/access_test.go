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

package sema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEntitlementAccess(t *testing.T) {

	assert.PanicsWithError(t,
		"neither map entitlement nor set entitlements given",
		func() {
			newEntitlementAccess(nil, Conjunction)
		},
	)

	assert.PanicsWithError(t,
		"mixed entitlement types",
		func() {
			newEntitlementAccess(
				[]Type{
					IdentityMappingType,
					MutableEntitlement,
				},
				Conjunction,
			)
		},
	)

	assert.PanicsWithError(t,
		"extra entitlement map type",
		func() {
			newEntitlementAccess(
				[]Type{
					MutableEntitlement,
					IdentityMappingType,
				},
				Conjunction,
			)
		},
	)

	assert.Equal(t,
		NewEntitlementSetAccess(
			[]*EntitlementType{
				MutableEntitlement,
			},
			Conjunction,
		),
		newEntitlementAccess(
			[]Type{
				MutableEntitlement,
			},
			Conjunction,
		),
	)

	assert.Equal(t,
		NewEntitlementMapAccess(
			IdentityMappingType,
		),
		newEntitlementAccess(
			[]Type{
				IdentityMappingType,
			},
			Conjunction,
		),
	)
}
