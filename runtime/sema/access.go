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
	IsLessPermissiveThan(Access) bool
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

func (e EntitlementAccess) IsLessPermissiveThan(other Access) bool {
	if primitive, isPrimitive := other.(PrimitiveAccess); isPrimitive {
		return ast.PrimitiveAccess(primitive) == ast.AccessPublic || ast.PrimitiveAccess(primitive) == ast.AccessPublicSettable
	}
	return e.subset(other.(EntitlementAccess))
}

type PrimitiveAccess ast.PrimitiveAccess

func (PrimitiveAccess) isAccess() {}

func (a PrimitiveAccess) Access() ast.Access {
	return ast.PrimitiveAccess(a)
}

func (a PrimitiveAccess) IsLessPermissiveThan(otherAccess Access) bool {
	if otherPrimitive, ok := otherAccess.(PrimitiveAccess); ok {
		return ast.PrimitiveAccess(a) <= ast.PrimitiveAccess(otherPrimitive)
	}
	// only private access is guaranteed to be less permissive than entitlement-based access
	return ast.PrimitiveAccess(a) == ast.AccessPrivate
}
