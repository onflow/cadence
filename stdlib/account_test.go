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

package stdlib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/tests/utils"
)

func TestSemaCheckPathLiteralForInternalStorageDomains(t *testing.T) {

	t.Parallel()

	internalStorageDomains := []string{
		InboxStorageDomain,
		AccountCapabilityStorageDomain,
		CapabilityControllerStorageDomain,
		PathCapabilityStorageDomain,
		CapabilityControllerTagStorageDomain,
	}

	test := func(domain string) {

		t.Run(domain, func(t *testing.T) {
			t.Parallel()

			_, err := sema.CheckPathLiteral(nil, domain, "test", nil, nil)
			var invalidPathDomainError *sema.InvalidPathDomainError
			require.ErrorAs(t, err, &invalidPathDomainError)
		})
	}

	for _, domain := range internalStorageDomains {
		test(domain)
	}
}

func TestCanBorrow(t *testing.T) {

	t.Parallel()

	inter := newInterpreter(t, `
        access(all)
        entitlement E

        access(all)
        entitlement F

        access(all)
        resource interface RI {}

        access(all)
        resource R: RI {}
    `)

	typeID := func(qualifiedIdentifier string) sema.TypeID {
		return utils.TestLocation.TypeID(nil, qualifiedIdentifier)
	}

	entitlementE := inter.Program.Elaboration.EntitlementType(typeID("E"))
	require.NotNil(t, entitlementE)

	entitlementF := inter.Program.Elaboration.EntitlementType(typeID("F"))
	require.NotNil(t, entitlementF)

	test := func(
		t *testing.T,
		instantiate func(a, b sema.Type) (a2, b2 *sema.ReferenceType),
		expected bool,
	) {
		rType := inter.Program.Elaboration.CompositeType(typeID("R"))
		require.NotNil(t, rType)

		riType := inter.Program.Elaboration.InterfaceType(typeID("RI"))
		require.NotNil(t, riType)

		riIntersectionType := sema.NewIntersectionType(
			nil,
			nil,
			[]*sema.InterfaceType{
				riType,
			},
		)

		types := []sema.Type{
			rType,
			riIntersectionType,
			sema.AnyResourceType,
		}

		for _, a := range types {
			for _, b := range types {
				a2, b2 := instantiate(a, b)

				t.Run(fmt.Sprintf("%s / %s", a2, b2), func(t *testing.T) {
					t.Parallel()

					require.Equal(t, expected, canBorrow(a2, b2))
				})
			}
		}
	}

	t.Run("&T / &T", func(t *testing.T) {

		t.Parallel()

		// Borrowing with the same type and no entitlements is allowed

		test(t,
			func(a, b sema.Type) (a2, b2 *sema.ReferenceType) {
				return sema.NewReferenceType(nil, sema.UnauthorizedAccess, a),
					sema.NewReferenceType(nil, sema.UnauthorizedAccess, b)
			},
			true,
		)
	})

	t.Run("auth(E) &T / &T", func(t *testing.T) {

		t.Parallel()

		// Borrowing with more permissions is NOT allowed

		test(t,
			func(a, b sema.Type) (a2, b2 *sema.ReferenceType) {
				return sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
							},
							sema.Conjunction,
						),
						a,
					),
					sema.NewReferenceType(nil, sema.UnauthorizedAccess, b)
			},
			false,
		)
	})

	t.Run("auth(E) &T / auth(E) &T", func(t *testing.T) {

		t.Parallel()

		// Borrowing with same permissions is allowed

		test(t,
			func(a, b sema.Type) (a2, b2 *sema.ReferenceType) {
				return sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
							},
							sema.Conjunction,
						),
						a,
					),
					sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
							},
							sema.Conjunction,
						),
						b,
					)
			},
			true,
		)
	})

	t.Run("auth(E, F) &T / auth(E) &T", func(t *testing.T) {

		t.Parallel()

		// Borrowing with more permissions is NOT allowed

		test(t,
			func(a, b sema.Type) (a2, b2 *sema.ReferenceType) {
				return sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
								entitlementF,
							},
							sema.Conjunction,
						),
						a,
					),
					sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
							},
							sema.Conjunction,
						),
						b,
					)
			},
			false,
		)
	})

	t.Run("auth(E) &T / auth(E, F) &T", func(t *testing.T) {

		t.Parallel()

		// Borrowing with fewer permissions is allowed

		test(t,
			func(a, b sema.Type) (a2, b2 *sema.ReferenceType) {

				return sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
							},
							sema.Conjunction,
						),
						a,
					),
					sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
								entitlementF,
							},
							sema.Conjunction,
						),
						b,
					)
			},
			true,
		)
	})

	t.Run("auth(E | F) &T / auth(E) &T", func(t *testing.T) {

		t.Parallel()

		// Borrowing with fewer permissions is allowed

		test(t,
			func(a, b sema.Type) (a2, b2 *sema.ReferenceType) {
				return sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
								entitlementF,
							},
							sema.Disjunction,
						),
						a,
					),
					sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
							},
							sema.Conjunction,
						),
						b,
					)
			},
			true,
		)
	})

	t.Run("auth(F) &T / auth(E) &T", func(t *testing.T) {

		t.Parallel()

		// Borrowing with unrelated permission is NOT allowed

		test(t,
			func(a, b sema.Type) (a2, b2 *sema.ReferenceType) {
				return sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementF,
							},
							sema.Conjunction,
						),
						a,
					),
					sema.NewReferenceType(
						nil,
						sema.NewEntitlementSetAccess(
							[]*sema.EntitlementType{
								entitlementE,
							},
							sema.Conjunction,
						),
						b,
					)
			},
			false,
		)
	})
}
