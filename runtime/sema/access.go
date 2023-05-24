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

	"golang.org/x/exp/maps"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
)

type Access interface {
	isAccess()
	// IsLessPermissiveThan returns whether receiver access is less permissive than argument access
	IsLessPermissiveThan(Access) bool
	// PermitsAccess returns whether receiver access permits argument access
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
	return a.AccessKeyword()
}

func (a EntitlementSetAccess) AccessKeyword() string {
	return a.string(func(ty Type) string { return ty.QualifiedString() })
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
		if i < e.Entitlements.Len()-1 {
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
						return eKey == otherKey
					})
				}
			default:
				panic(errors.NewUnreachableError())
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
			default:
				panic(errors.NewUnreachableError())
			}
			return outerPredicate(otherAccess.Entitlements.Contains)
		default:
			panic(errors.NewUnreachableError())
		}
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
	Type     *EntitlementMapType
	domain   EntitlementSetAccess
	codomain EntitlementSetAccess
	images   map[*EntitlementType]*EntitlementOrderedSet
}

var _ Access = EntitlementMapAccess{}

func NewEntitlementMapAccess(mapType *EntitlementMapType) EntitlementMapAccess {
	return EntitlementMapAccess{
		Type:   mapType,
		images: make(map[*EntitlementType]*EntitlementOrderedSet),
	}
}

func (EntitlementMapAccess) isAccess() {}

func (e EntitlementMapAccess) string(typeFormatter func(ty Type) string) string {
	return typeFormatter(e.Type)
}

func (a EntitlementMapAccess) Description() string {
	return a.AccessKeyword()
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
	// if we are initializing a field that was declared with an entitlement-mapped reference type,
	// the type we are using to initialize that member must be fully authorized for the entire codomain
	// of the map. That is, for some field declared `access(M) let x: auth(M) &T`, when `x` is intialized
	// by `self.x = y`, `y` must be a reference of type `auth(X, Y, Z, ...) &T` where `{X, Y, Z, ...}` is
	// a superset of all the possible output types of `M` (all the possible entitlements `x` may have)
	//
	// as an example:
	//
	// entitlement mapping M {
	//    X -> Y
	//    E -> F
	// }
	// resource R {
	//    access(M) let x: auth(M) &T
	//    init(tref: auth(Y, F) &T) {
	//        self.x = tref
	//    }
	// }
	//
	// the tref value used to initialize `x` must be entitled to the full output of `M` (in this case)
	// `(Y, F)`, because the mapped access of `x` may provide either (or both) `Y` and `F` depending on
	// the input entitlement. It is only safe for `R` to give out these entitlements if it actually
	// possesses them, so we require the initializing value to have every possible entitlement that may
	// be produced by the map
	case EntitlementSetAccess:
		return e.Codomain().PermitsAccess(otherAccess)
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

func (e EntitlementMapAccess) Domain() EntitlementSetAccess {
	if e.domain.Entitlements != nil {
		return e.domain
	}

	domain := make(map[*EntitlementType]struct{})
	for _, relation := range e.Type.Relations {
		domain[relation.Input] = struct{}{}
	}
	e.domain = NewEntitlementSetAccess(maps.Keys(domain), Conjunction)
	return e.domain
}

func (e EntitlementMapAccess) Codomain() EntitlementSetAccess {
	if e.codomain.Entitlements != nil {
		return e.codomain
	}

	codomain := make(map[*EntitlementType]struct{})
	for _, relation := range e.Type.Relations {
		codomain[relation.Output] = struct{}{}
	}
	e.codomain = NewEntitlementSetAccess(maps.Keys(codomain), Conjunction)
	return e.codomain
}

// produces the image set of a single entitlement through a map
// the image set of one element is always a conjunction
func (e EntitlementMapAccess) entitlementImage(entitlement *EntitlementType) (output *EntitlementOrderedSet) {
	image := e.images[entitlement]
	if image != nil {
		return image
	}

	output = orderedmap.New[EntitlementOrderedSet](0)
	for _, relation := range e.Type.Relations {
		if relation.Input.Equal(entitlement) {
			output.Set(relation.Output, struct{}{})
		}
	}

	e.images[entitlement] = output
	return
}

// Image applies all the entitlements in the `argumentAccess` to the function
// defined by the map in `e`, producing a new entitlement set of the image of the
// arguments.
func (e EntitlementMapAccess) Image(inputs Access, astRange ast.Range) (Access, error) {
	switch inputs := inputs.(type) {
	// primitive access always passes trivially through the map
	case PrimitiveAccess:
		return inputs, nil
	case EntitlementSetAccess:
		output := orderedmap.New[EntitlementOrderedSet](inputs.Entitlements.Len())
		var err error = nil
		inputs.Entitlements.Foreach(func(entitlement *EntitlementType, _ struct{}) {
			entitlementImage := e.entitlementImage(entitlement)
			// the image of a single element is always a conjunctive set; consider a mapping
			// M defined as X -> Y, X -> Z, A -> B, A -> C. M(X) = Y & Z and M(A) = B & C.
			// Thus M(X | A) would be ((Y & Z) | (B & C)), which is a disjunction of two conjunctions,
			// which is too complex to be represented in Cadence as a type. Thus whenever such a type
			// would arise, we raise an error instead
			if inputs.SetKind == Disjunction && entitlementImage.Len() > 1 {
				err = &UnrepresentableEntitlementMapOutputError{
					Input: inputs,
					Map:   e.Type,
					Range: astRange,
				}
			}
			output.SetAll(entitlementImage)
		})
		if err != nil {
			return nil, err
		}
		// the image of a set through a map is the conjunction of all the output sets
		if output.Len() == 0 {
			return UnauthorizedAccess, nil
		}
		return EntitlementSetAccess{
			Entitlements: output,
			SetKind:      inputs.SetKind,
		}, nil
	}
	return UnauthorizedAccess, nil
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
