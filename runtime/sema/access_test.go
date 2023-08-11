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

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"neither map entitlement nor set entitlements given",
			func() {
				newEntitlementAccess(nil, Conjunction)
			},
		)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"invalid entitlement type: Void",
			func() {
				newEntitlementAccess([]Type{VoidType}, Conjunction)
			},
		)
	})

	t.Run("map + entitlement", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"mixed entitlement types",
			func() {
				newEntitlementAccess(
					[]Type{
						IdentityMappingType,
						MutateEntitlement,
					},
					Conjunction,
				)
			},
		)
	})

	t.Run("entitlement + map", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"mixed entitlement types",
			func() {
				newEntitlementAccess(
					[]Type{
						MutateEntitlement,
						IdentityMappingType,
					},
					Conjunction,
				)
			},
		)
	})

	t.Run("single entitlement", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			NewEntitlementSetAccess(
				[]*EntitlementType{
					MutateEntitlement,
				},
				Conjunction,
			),
			newEntitlementAccess(
				[]Type{
					MutateEntitlement,
				},
				Conjunction,
			),
		)
	})

	t.Run("two entitlements, conjunction", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			NewEntitlementSetAccess(
				[]*EntitlementType{
					MutateEntitlement,
					InsertEntitlement,
				},
				Conjunction,
			),
			newEntitlementAccess(
				[]Type{
					MutateEntitlement,
					InsertEntitlement,
				},
				Conjunction,
			),
		)
	})

	t.Run("two entitlements, disjunction", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			NewEntitlementSetAccess(
				[]*EntitlementType{
					MutateEntitlement,
					InsertEntitlement,
				},
				Disjunction,
			),
			newEntitlementAccess(
				[]Type{
					MutateEntitlement,
					InsertEntitlement,
				},
				Disjunction,
			),
		)
	})

	t.Run("single map", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("two maps", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"extra entitlement map type",
			func() {
				newEntitlementAccess(
					[]Type{
						IdentityMappingType,
						AccountMappingType,
					},
					Conjunction,
				)
			},
		)
	})
}
