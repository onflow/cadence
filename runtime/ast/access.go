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

package ast

import (
	"encoding/json"
	"strings"

	"github.com/onflow/cadence/runtime/errors"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=PrimitiveAccess

type Access interface {
	isAccess()
	Keyword() string
	Description() string
	String() string
	MarshalJSON() ([]byte, error)
}

type EntitlementSet interface {
	Entitlements() []*NominalType
	Separator() string
}

type ConjunctiveEntitlementSet struct {
	Elements []*NominalType `json:"ConjunctiveElements"`
}

func (s ConjunctiveEntitlementSet) Entitlements() []*NominalType {
	return s.Elements
}

func (s ConjunctiveEntitlementSet) Separator() string {
	return ","
}

func NewConjunctiveEntitlementSet(entitlements []*NominalType) ConjunctiveEntitlementSet {
	return ConjunctiveEntitlementSet{Elements: entitlements}
}

type DisjunctiveEntitlementSet struct {
	Elements []*NominalType `json:"DisjunctiveElements"`
}

func (s DisjunctiveEntitlementSet) Entitlements() []*NominalType {
	return s.Elements
}

func (s DisjunctiveEntitlementSet) Separator() string {
	return " |"
}

func NewDisjunctiveEntitlementSet(entitlements []*NominalType) DisjunctiveEntitlementSet {
	return DisjunctiveEntitlementSet{Elements: entitlements}
}

type EntitlementAccess struct {
	EntitlementSet EntitlementSet
}

var _ Access = EntitlementAccess{}

func NewEntitlementAccess(entitlements EntitlementSet) EntitlementAccess {
	return EntitlementAccess{EntitlementSet: entitlements}
}

func (EntitlementAccess) isAccess() {}

func (EntitlementAccess) Description() string {
	return "entitled access"
}

func (e EntitlementAccess) entitlementsString() string {
	str := strings.Builder{}
	for i, entitlement := range e.EntitlementSet.Entitlements() {
		str.Write([]byte(entitlement.String()))
		if i < len(e.EntitlementSet.Entitlements())-1 {
			str.Write([]byte(e.EntitlementSet.Separator()))
		}
	}
	return str.String()
}

func (e EntitlementAccess) String() string {
	return "ConjunctiveEntitlementAccess " + e.entitlementsString()
}

func (e EntitlementAccess) Keyword() string {
	return "access(" + e.entitlementsString() + ")"
}

func (e EntitlementAccess) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

func (e EntitlementAccess) subset(other EntitlementAccess) bool {
	otherSet := make(map[*NominalType]struct{})
	for _, entitlement := range other.EntitlementSet.Entitlements() {
		otherSet[entitlement] = struct{}{}
	}

	for _, entitlement := range e.EntitlementSet.Entitlements() {
		if _, found := otherSet[entitlement]; !found {
			return false
		}
	}

	return true
}

func (e EntitlementAccess) IsLessPermissiveThan(other Access) bool {
	if primitive, isPrimitive := other.(PrimitiveAccess); isPrimitive {
		return primitive == AccessPublic || primitive == AccessPublicSettable
	}
	conjunctiveEntitlementAccess, ok := other.(EntitlementAccess)
	if !ok {
		return false
	}
	return e.subset(conjunctiveEntitlementAccess)
}

type PrimitiveAccess uint8

// NOTE: order indicates permissiveness: from least to most permissive!

const (
	AccessNotSpecified PrimitiveAccess = iota
	AccessPrivate
	AccessContract
	AccessAccount
	AccessPublic
	AccessPublicSettable
)

func PrimitiveAccessCount() int {
	return len(_PrimitiveAccess_index) - 1
}

func (PrimitiveAccess) isAccess() {}

// TODO: remove.
//   only used by tests which are not updated yet
//   to include contract and account access

var BasicAccesses = []PrimitiveAccess{
	AccessNotSpecified,
	AccessPrivate,
	AccessPublic,
	AccessPublicSettable,
}

var AllAccesses = append(BasicAccesses[:],
	AccessContract,
	AccessAccount,
)

func (a PrimitiveAccess) Keyword() string {
	switch a {
	case AccessNotSpecified:
		return ""
	case AccessPrivate:
		return "priv"
	case AccessPublic:
		return "pub"
	case AccessPublicSettable:
		return "pub(set)"
	case AccessAccount:
		return "access(account)"
	case AccessContract:
		return "access(contract)"
	}

	panic(errors.NewUnreachableError())
}

func (a PrimitiveAccess) Description() string {
	switch a {
	case AccessNotSpecified:
		return "not specified"
	case AccessPrivate:
		return "private"
	case AccessPublic:
		return "public"
	case AccessPublicSettable:
		return "public settable"
	case AccessAccount:
		return "account"
	case AccessContract:
		return "contract"
	}

	panic(errors.NewUnreachableError())
}

func (a PrimitiveAccess) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}
