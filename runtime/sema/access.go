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
)

type Access interface {
	isAccess()
	// returns whether receiver access < argument access
	IsLessPermissiveThan(Access) bool
	// returns whether receiver access >= argument access
	IsMorePermissiveThan(Access) bool
	Access() ast.Access
}

type EntitlementAccess struct {
	astAccess    ast.EntitlementAccess
	Entitlements []*EntitlementType
}

var _ Access = EntitlementAccess{}

func NewEntitlementAccess(entitlements []*EntitlementType) EntitlementAccess {
	return EntitlementAccess{Entitlements: entitlements}
}

func (EntitlementAccess) isAccess() {}

func (a EntitlementAccess) Access() ast.Access {
	return a.astAccess
}

func (e EntitlementAccess) subset(other EntitlementAccess) bool {
	otherSet := make(map[*EntitlementType]struct{})
	for _, entitlement := range other.Entitlements {
		otherSet[entitlement] = struct{}{}
	}

	for _, entitlement := range e.Entitlements {
		if _, found := otherSet[entitlement]; !found {
			return false
		}
	}

	return true
}

func (e EntitlementAccess) IsMorePermissiveThan(other Access) bool {
	if _, isPrimitive := other.(PrimitiveAccess); isPrimitive {
		return true
	}
	// e >= other if e is a subset of other, as entitlement sets are unions rather than intersections
	return e.subset(other.(EntitlementAccess))
}

func (e EntitlementAccess) IsLessPermissiveThan(other Access) bool {
	if primitive, isPrimitive := other.(PrimitiveAccess); isPrimitive {
		return ast.PrimitiveAccess(primitive) != ast.AccessPrivate
	}
	// subset check returns true on equality, and we want this function to be false on equality, so invert the >= check
	return !other.IsMorePermissiveThan(e)
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

func (a PrimitiveAccess) IsMorePermissiveThan(otherAccess Access) bool {
	if otherPrimitive, ok := otherAccess.(PrimitiveAccess); ok {
		return ast.PrimitiveAccess(a) >= ast.PrimitiveAccess(otherPrimitive)
	}
	// only priv access is guaranteed to be less permissive than entitlement-based access, but cannot appear in interfaces
	return ast.PrimitiveAccess(a) != ast.AccessPrivate
}
