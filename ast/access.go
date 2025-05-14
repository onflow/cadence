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

package ast

import (
	"encoding/json"
	"strings"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

//go:generate stringer -type=PrimitiveAccess

type Access interface {
	isAccess()
	Keyword() string
	Description() string
	String() string
	MarshalJSON() ([]byte, error)
}

type Separator uint8

const (
	Disjunction Separator = iota
	Conjunction
)

func (s Separator) String() string {
	switch s {
	case Disjunction:
		return " |"
	case Conjunction:
		return ","
	}
	panic(errors.NewUnreachableError())
}

type EntitlementSet interface {
	Authorization
	Entitlements() []*NominalType
	Separator() Separator
}

type ConjunctiveEntitlementSet struct {
	Elements []*NominalType `json:"ConjunctiveElements"`
}

var _ EntitlementSet = &ConjunctiveEntitlementSet{}

func (ConjunctiveEntitlementSet) isAuthorization() {}

func (s *ConjunctiveEntitlementSet) Entitlements() []*NominalType {
	return s.Elements
}

func (s *ConjunctiveEntitlementSet) Separator() Separator {
	return Conjunction
}

func NewConjunctiveEntitlementSet(entitlements []*NominalType) *ConjunctiveEntitlementSet {
	return &ConjunctiveEntitlementSet{Elements: entitlements}
}

type DisjunctiveEntitlementSet struct {
	Elements []*NominalType `json:"DisjunctiveElements"`
}

var _ EntitlementSet = &DisjunctiveEntitlementSet{}

func (DisjunctiveEntitlementSet) isAuthorization() {}

func (s *DisjunctiveEntitlementSet) Entitlements() []*NominalType {
	return s.Elements
}

func (s *DisjunctiveEntitlementSet) Separator() Separator {
	return Disjunction
}

func NewDisjunctiveEntitlementSet(entitlements []*NominalType) *DisjunctiveEntitlementSet {
	return &DisjunctiveEntitlementSet{Elements: entitlements}
}

type Authorization interface {
	isAuthorization()
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

func (e EntitlementAccess) entitlementsString(prefix *strings.Builder) {
	for i, entitlement := range e.EntitlementSet.Entitlements() {
		prefix.WriteString(entitlement.String())
		if i < len(e.EntitlementSet.Entitlements())-1 {
			prefix.WriteString(e.EntitlementSet.Separator().String())
		}
	}
}

func (e EntitlementAccess) String() string {
	var sb strings.Builder
	sb.WriteString("EntitlementAccess ")
	e.entitlementsString(&sb)
	return sb.String()
}

func (e EntitlementAccess) Keyword() string {
	var sb strings.Builder
	sb.WriteString("access(")
	e.entitlementsString(&sb)
	sb.WriteString(")")
	return sb.String()
}

func (e EntitlementAccess) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

type MappedAccess struct {
	EntitlementMap *NominalType
	StartPos       Position
}

var _ Access = &MappedAccess{}

func (*MappedAccess) isAccess()        {}
func (*MappedAccess) isAuthorization() {}

func (*MappedAccess) Description() string {
	return "entitlement-mapped access"
}

func NewMappedAccess(
	typ *NominalType,
	startPos Position,
) *MappedAccess {
	return &MappedAccess{
		EntitlementMap: typ,
		StartPos:       startPos,
	}
}

func (a *MappedAccess) StartPosition() Position {
	return a.StartPos
}

func (a *MappedAccess) EndPosition(memoryGauge common.MemoryGauge) Position {
	return a.EntitlementMap.EndPosition(memoryGauge)
}

func (a *MappedAccess) String() string {
	var str strings.Builder
	str.WriteString("mapping ")
	str.WriteString(a.EntitlementMap.String())
	return str.String()
}

func (a *MappedAccess) Keyword() string {
	var str strings.Builder
	str.WriteString("access(")
	str.WriteString(a.String())
	str.WriteString(")")
	return str.String()
}

func (a *MappedAccess) MarshalJSON() ([]byte, error) {
	type Alias MappedAccess
	return json.Marshal(&struct {
		*Alias
		Range
	}{
		Range: NewUnmeteredRangeFromPositioned(a),
		Alias: (*Alias)(a),
	})
}

type PrimitiveAccess uint8

// NOTE: order indicates permissiveness: from least to most permissive!

const (
	AccessNotSpecified PrimitiveAccess = iota
	AccessNone                         // "top" access, only used for mapping operations, not actually expressible in the language
	AccessSelf
	AccessContract
	AccessAccount
	AccessAll
	AccessPubSettableLegacy // Deprecated
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
	AccessSelf,
	AccessAll,
}

var AllAccesses = common.Concat(
	BasicAccesses,
	[]PrimitiveAccess{
		AccessContract,
		AccessAccount,
	},
)

func (a PrimitiveAccess) Keyword() string {
	switch a {
	case AccessNotSpecified:
		return ""
	case AccessSelf:
		return "access(self)"
	case AccessAll:
		return "access(all)"
	case AccessAccount:
		return "access(account)"
	case AccessContract:
		return "access(contract)"
	case AccessPubSettableLegacy:
		return "pub(set)"
	case AccessNone:
		return "inaccessible"
	}

	panic(errors.NewUnreachableError())
}

func (a PrimitiveAccess) Description() string {
	switch a {
	case AccessNotSpecified:
		return "not specified"
	case AccessSelf:
		return "self"
	case AccessAll:
		return "all"
	case AccessAccount:
		return "account"
	case AccessContract:
		return "contract"
	case AccessPubSettableLegacy:
		return "legacy public settable"
	case AccessNone:
		return "inaccessible"
	}

	panic(errors.NewUnreachableError())
}

func (a PrimitiveAccess) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}
