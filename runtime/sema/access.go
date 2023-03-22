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
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common/orderedmap"
)

type Access interface {
	isAccess()
	// returns whether receiver access is less permissive than argument access
	IsLessPermissiveThan(Access) bool
	// returns whether receiver access permits argument access
	PermitsAccess(Access) bool
	Equal(other Access) bool
	string(func(ty Type) string) string
	Description() string
	// string representation of this access when it is used as an access modifier
	AccessKeyword() string
	// string representation of this access when it is used as an auth modifier
	AuthKeyword() string
}

type EntitlementSetKind uint8

const (
	Conjunction EntitlementSetKind = iota
	Disjunction
)

type EntitlementSetAccess struct {
	Entitlements *EntitlementOrderedSet
	SetKind      EntitlementSetKind
}

var _ Access = EntitlementSetAccess{}

func NewEntitlementSetAccess(
	entitlements []*EntitlementType,
	setKind EntitlementSetKind,
) EntitlementSetAccess {
	set := orderedmap.New[EntitlementOrderedSet](len(entitlements))
	for _, entitlement := range entitlements {
		set.Set(entitlement, struct{}{})
	}
	return EntitlementSetAccess{
		Entitlements: set,
		SetKind:      setKind,
	}
}

func (EntitlementSetAccess) isAccess() {}

func (a EntitlementSetAccess) Description() string {
	return "entitlement set access"
}

func (a EntitlementSetAccess) AccessKeyword() string {
	return a.string(func(ty Type) string { return ty.String() })
}

func (a EntitlementSetAccess) AuthKeyword() string {
	return fmt.Sprintf("auth(%s)", a.AccessKeyword())
}

func (e EntitlementSetAccess) string(typeFormatter func(ty Type) string) string {
	var builder strings.Builder
	var separator string

	if e.SetKind == Conjunction {
		separator = ", "
	} else if e.SetKind == Disjunction {
		separator = " | "
	}

	e.Entitlements.ForeachWithIndex(func(i int, entitlement *EntitlementType, _ struct{}) {
		builder.WriteString(typeFormatter(entitlement))
		if i < e.Entitlements.Len() {
			builder.WriteString(separator)
		}
	})
	return builder.String()
}

func (e EntitlementSetAccess) Equal(other Access) bool {
	switch otherAccess := other.(type) {
	case EntitlementSetAccess:
		return e.SetKind == otherAccess.SetKind &&
			e.PermitsAccess(otherAccess) &&
			otherAccess.PermitsAccess(e)
	}
	return false
}

func (e EntitlementSetAccess) PermitsAccess(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return otherAccess == PrimitiveAccess(ast.AccessPrivate)
	case EntitlementSetAccess:
		switch otherAccess.SetKind {
		case Disjunction:
			var innerPredicate func(eKey *EntitlementType) bool
			switch e.SetKind {
			case Disjunction:
				// e permits other if e is a superset of other when both are disjunctions,
				// or equivalently if other is a subset of e, i.e. whichever entitlement other has,
				// it is guaranteed to be a valid entitlement for e
				innerPredicate = e.Entitlements.Contains
			case Conjunction:
				// when e is a conjunction and other is a disjunction, e permits other only when the two sets contain
				// exactly the same elements
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
				// e permits other whenever e is a subset of other (when other possesses more entitlements than e)
				// when both are conjunctions
				outerPredicate = e.Entitlements.ForAllKeys
			case Disjunction:
				// when e is a disjunction and other is a conjunction, e permits other when any of other's entitlements appear in e,
				// or equivalently, when the two sets are not disjoint
				outerPredicate = e.Entitlements.ForAnyKey
			}
			return outerPredicate(otherAccess.Entitlements.Contains)
		}
		return false
	default:
		return false
	}
}

func (e EntitlementSetAccess) IsLessPermissiveThan(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return ast.PrimitiveAccess(otherAccess) != ast.AccessPrivate
	case EntitlementSetAccess:
		// subset check returns true on equality, and we want this function to be false on equality, so invert the >= check
		return !e.PermitsAccess(otherAccess)
	default:
		return true
	}
}

type EntitlementMapAccess struct {
	Type *EntitlementMapType
}

var _ Access = EntitlementMapAccess{}

func NewEntitlementMapAccess(mapType *EntitlementMapType) EntitlementMapAccess {
	return EntitlementMapAccess{Type: mapType}
}

func (EntitlementMapAccess) isAccess() {}

func (e EntitlementMapAccess) string(typeFormatter func(ty Type) string) string {
	return typeFormatter(e.Type)
}

func (a EntitlementMapAccess) Description() string {
	return "entitlement map access"
}

func (a EntitlementMapAccess) AccessKeyword() string {
	return a.string(func(ty Type) string { return ty.String() })
}

func (a EntitlementMapAccess) AuthKeyword() string {
	return fmt.Sprintf("auth(%s)", a.AccessKeyword())
}

func (e EntitlementMapAccess) Equal(other Access) bool {
	switch otherAccess := other.(type) {
	case EntitlementMapAccess:
		return e.Type.Equal(otherAccess.Type)
	}
	return false
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

func (a PrimitiveAccess) string(_ func(_ Type) string) string {
	return ast.PrimitiveAccess(a).String()
}

func (a PrimitiveAccess) Description() string {
	return ast.PrimitiveAccess(a).Description()
}

func (a PrimitiveAccess) AccessKeyword() string {
	return ast.PrimitiveAccess(a).Keyword()
}

func (a PrimitiveAccess) AuthKeyword() string {
	return ""
}

func (a PrimitiveAccess) Equal(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return ast.PrimitiveAccess(a) == ast.PrimitiveAccess(otherAccess)
	}
	return false
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
