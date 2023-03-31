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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common/orderedmap"
)

type Access interface {
	isAccess()
	// returns whether receiver access is less permissive than argument access
	IsLessPermissiveThan(Access) bool
	// returns whether receiver access permits argument access
	PermitsAccess(Access) bool
	Access() ast.Access
}

type EntitlementSetKind uint8

const (
	Conjunction EntitlementSetKind = iota
	Disjunction
)

type EntitlementAccess struct {
	astAccess    ast.EntitlementAccess
	Entitlements *EntitlementOrderedSet
	SetKind      EntitlementSetKind
}

var _ Access = EntitlementAccess{}

func NewEntitlementAccess(
	astAccess ast.EntitlementAccess,
	entitlements []*EntitlementType,
	setKind EntitlementSetKind,
) EntitlementAccess {
	set := orderedmap.New[EntitlementOrderedSet](len(entitlements))
	for _, entitlement := range entitlements {
		set.Set(entitlement, struct{}{})
	}
	return EntitlementAccess{
		Entitlements: set,
		SetKind:      setKind,
		astAccess:    astAccess,
	}
}

func (EntitlementAccess) isAccess() {}

func (a EntitlementAccess) Access() ast.Access {
	return a.astAccess
}

func (e EntitlementAccess) PermitsAccess(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return otherAccess == PrimitiveAccess(ast.AccessPrivate)
	case EntitlementAccess:
		switch otherAccess.SetKind {
		case Disjunction:
			var innerPredicate func(eKey *EntitlementType) bool
			switch e.SetKind {
			case Disjunction:
				// `e` permits `other` if `e` is a superset of `other` when both are disjunctions,
				// or equivalently if `other` is a subset of `e`, i.e. whichever entitlement `other` has,
				// it is guaranteed to be a valid entitlement for `e`.
				//
				// For example, given some `access(X | Y | Z) fun foo()` member on `R`, `foo` is callable by a value `ref` of type `auth(X | Y) &R`
				// because regardless of whether `ref` actually possesses an `X` or `Y`, that is one of the entitlements accepted by `foo`.
				//
				// Concretely: `auth (U1 | U2 | ... ) &X` <: `auth (T1 | T2 | ... ) &X` whenever `{U1, U2, ...}` is a subset of `{T1, T2, ...}`,
				// or equivalently `∀U ∈ {U1, U2, ...}, ∃T ∈ {T1, T2, ...}, T = U`
				innerPredicate = e.Entitlements.Contains
			case Conjunction:
				// when `e` is a conjunction and `other` is a disjunction, `e` permits other only when the two sets contain
				// exactly the same elements, or in practice when each set contains exactly one equivalent element
				//
				// Concretely: `auth (U1 | U2 | ... ) &X <: auth (T1, T2,  ... ) &X` whenever `∀U ∈ {U1, U2, ...}, ∀T ∈ {T1, T2, ...}, T = U`.
				innerPredicate = func(eKey *EntitlementType) bool {
					return e.Entitlements.ForAllKeys(func(otherKey *EntitlementType) bool {
						return eKey.Equal(otherKey)
					})
				}
			}
			return otherAccess.Entitlements.ForAllKeys(innerPredicate)
		case Conjunction:
			var outerPredicate func(func(eKey *EntitlementType) bool) bool
			switch e.SetKind {
			case Conjunction:
				// `e` permits other whenever `e` is a subset of `other` (when `other` possesses more entitlements than `e`)
				// when both are conjunctions.
				//
				// For example given some `access(X, Y) fun foo()` member on `R`, `foo` is callable by a value `ref` of type `auth(X, Y, Z) &R`
				// because `ref` possesses all the entitlements required by `foo` (and more)
				//
				// Concretely: `auth (U1, U2, ... ) &X <: auth (T1, T2, ... ) &X` whenever `{U1, U2, ...}` is a superset of `{T1, T2, ...}`,
				// or equivalently `∀T ∈ {T1, T2, ...}, ∃U ∈ {U1, U2, ...}, T = U`
				outerPredicate = e.Entitlements.ForAllKeys
			case Disjunction:
				// when `e` is a disjunction and `other` is a conjunction, `e` permits other when any of `other`'s entitlements appear in `e`,
				// or equivalently, when the two sets are not disjoint
				//
				// For example, given some `access(X | Y) fun foo()` member on `R`, `foo` is callable by a value `ref` of type `auth(X, Z) &R`
				// because `ref` possesses one the entitlements required by `foo`
				//
				// Concretely: `auth (U1, U2, ... ) &X <: auth (T1 | T2 | ... ) &X` whenever `{U1, U2, ...}` is not disjoint from `{T1, T2, ...}`,
				// or equivalently `∃U ∈ {U1, U2, ...}, ∃T ∈ {T1, T2, ...}, T = U`
				outerPredicate = e.Entitlements.ForAnyKey
			}
			return outerPredicate(otherAccess.Entitlements.Contains)
		}
		return false
	default:
		return false
	}
}

func (e EntitlementAccess) IsLessPermissiveThan(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return ast.PrimitiveAccess(otherAccess) != ast.AccessPrivate
	case EntitlementAccess:
		// subset check returns true on equality, and we want this function to be false on equality, so invert the >= check
		return !e.PermitsAccess(otherAccess)
	default:
		return true
	}
}

type EntitlementMapAccess struct {
	astAccess ast.EntitlementAccess
	Type      *EntitlementMapType
}

var _ Access = EntitlementMapAccess{}

func NewEntitlementMapAccess(astAccess ast.EntitlementAccess, mapType *EntitlementMapType) EntitlementMapAccess {
	return EntitlementMapAccess{astAccess: astAccess, Type: mapType}
}

func (EntitlementMapAccess) isAccess() {}

func (a EntitlementMapAccess) Access() ast.Access {
	return a.astAccess
}

func (e EntitlementMapAccess) PermitsAccess(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return otherAccess == PrimitiveAccess(ast.AccessPrivate)
	case EntitlementMapAccess:
		return e.Type.Equal(otherAccess.Type)
	default:
		return false
	}
}

func (e EntitlementMapAccess) IsLessPermissiveThan(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return ast.PrimitiveAccess(otherAccess) != ast.AccessPrivate
	case EntitlementMapAccess:
		// this should be false on equality
		return !e.Type.Equal(otherAccess.Type)
	default:
		return true
	}
}

type PrimitiveAccess ast.PrimitiveAccess

func (PrimitiveAccess) isAccess() {}

func (a PrimitiveAccess) Access() ast.Access {
	return ast.PrimitiveAccess(a)
}

func (a PrimitiveAccess) IsLessPermissiveThan(otherAccess Access) bool {
	if otherPrimitive, ok := otherAccess.(PrimitiveAccess); ok {
		return ast.PrimitiveAccess(a) < ast.PrimitiveAccess(otherPrimitive)
	}
	// primitive and entitlement access should never mix in interface conformance checks
	return true
}

func (a PrimitiveAccess) PermitsAccess(otherAccess Access) bool {
	if otherPrimitive, ok := otherAccess.(PrimitiveAccess); ok {
		return ast.PrimitiveAccess(a) >= ast.PrimitiveAccess(otherPrimitive)
	}
	// only priv access is guaranteed to be less permissive than entitlement-based access, but cannot appear in interfaces
	return ast.PrimitiveAccess(a) != ast.AccessPrivate
}
