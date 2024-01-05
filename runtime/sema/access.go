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
	"strings"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
)

type Access interface {
	isAccess()
	IsPrimitiveAccess() bool
	ID() TypeID
	String() string
	QualifiedString() string
	Equal(other Access) bool
	// IsLessPermissiveThan returns whether receiver access is less permissive than argument access
	IsLessPermissiveThan(Access) bool
	// PermitsAccess returns whether receiver access permits argument access
	PermitsAccess(Access) bool
}

type EntitlementSetKind uint8

const (
	Conjunction EntitlementSetKind = iota
	Disjunction
)

// EntitlementSetAccess

type EntitlementSetAccess struct {
	_            common.Incomparable
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

func NewAccessFromEntitlementSet(
	set *EntitlementOrderedSet,
	setKind EntitlementSetKind,
) Access {
	if set.Len() == 0 {
		return UnauthorizedAccess
	}

	return EntitlementSetAccess{
		Entitlements: set,
		SetKind:      setKind,
	}
}

func (EntitlementSetAccess) isAccess() {}

func (EntitlementSetAccess) IsPrimitiveAccess() bool {
	return false
}

func (e EntitlementSetAccess) ID() TypeID {
	entitlementTypeIDs := make([]TypeID, 0, e.Entitlements.Len())
	e.Entitlements.Foreach(func(entitlement *EntitlementType, _ struct{}) {
		entitlementTypeIDs = append(
			entitlementTypeIDs,
			entitlement.ID(),
		)
	})

	// FormatEntitlementSetTypeID sorts
	return FormatEntitlementSetTypeID(entitlementTypeIDs, e.SetKind)
}

func FormatEntitlementSetTypeID[T ~string](entitlementTypeIDs []T, kind EntitlementSetKind) T {
	var builder strings.Builder
	var separator string

	switch kind {
	case Conjunction:
		separator = ","
	case Disjunction:
		separator = "|"
	default:
		panic(errors.NewUnreachableError())
	}

	// Join entitlements' type IDs in increasing order (sorted)

	slices.Sort(entitlementTypeIDs)

	for i, entitlementTypeID := range entitlementTypeIDs {
		if i > 0 {
			builder.WriteString(separator)
		}
		builder.WriteString(string(entitlementTypeID))
	}

	return T(builder.String())
}

func (e EntitlementSetAccess) string(typeFormatter func(Type) string) string {
	var builder strings.Builder
	var separator string

	switch e.SetKind {
	case Conjunction:
		separator = ", "
	case Disjunction:
		separator = " | "
	default:
		panic(errors.NewUnreachableError())
	}

	// Join entitlements' string representation in given order (as-is)

	e.Entitlements.ForeachWithIndex(func(i int, entitlement *EntitlementType, _ struct{}) {
		if i > 0 {
			builder.WriteString(separator)
		}
		builder.WriteString(typeFormatter(entitlement))

	})

	return builder.String()
}

func (e EntitlementSetAccess) String() string {
	return e.string(func(t Type) string {
		return t.String()
	})
}

func (e EntitlementSetAccess) QualifiedString() string {
	return e.string(func(t Type) string {
		return t.QualifiedString()
	})
}

func (e EntitlementSetAccess) Equal(other Access) bool {
	otherAccess, ok := other.(EntitlementSetAccess)
	if !ok {
		return false
	}

	return e.SetKind == otherAccess.SetKind &&
		e.PermitsAccess(otherAccess) &&
		otherAccess.PermitsAccess(e)
}

func (e EntitlementSetAccess) PermitsAccess(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return otherAccess == PrimitiveAccess(ast.AccessSelf)
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
		return ast.PrimitiveAccess(otherAccess) != ast.AccessSelf
	case EntitlementSetAccess:
		// subset check returns true on equality, and we want this function to be false on equality, so invert the >= check
		return !e.PermitsAccess(otherAccess)
	default:
		return true
	}
}

// EntitlementMapAccess

type EntitlementMapAccess struct {
	Type         *EntitlementMapType
	domain       EntitlementSetAccess
	domainOnce   sync.Once
	codomain     EntitlementSetAccess
	codomainOnce sync.Once
	images       sync.Map
}

var _ Access = &EntitlementMapAccess{}

func NewEntitlementMapAccess(mapType *EntitlementMapType) *EntitlementMapAccess {
	return &EntitlementMapAccess{
		Type:   mapType,
		images: sync.Map{},
	}
}

func (*EntitlementMapAccess) isAccess() {}

func (*EntitlementMapAccess) IsPrimitiveAccess() bool {
	return false
}

func (e *EntitlementMapAccess) ID() TypeID {
	return e.Type.ID()
}

func (e *EntitlementMapAccess) String() string {
	return e.Type.String()
}

func (e *EntitlementMapAccess) QualifiedString() string {
	return e.Type.QualifiedString()
}

func (e *EntitlementMapAccess) Equal(other Access) bool {
	switch otherAccess := other.(type) {
	case *EntitlementMapAccess:
		return e.Type.Equal(otherAccess.Type)
	}
	return false
}

func (e *EntitlementMapAccess) PermitsAccess(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return otherAccess == PrimitiveAccess(ast.AccessSelf)
	case *EntitlementMapAccess:
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

func (e *EntitlementMapAccess) IsLessPermissiveThan(other Access) bool {
	switch otherAccess := other.(type) {
	case PrimitiveAccess:
		return ast.PrimitiveAccess(otherAccess) != ast.AccessSelf
	case *EntitlementMapAccess:
		// this should be false on equality
		return !e.Type.Equal(otherAccess.Type)
	default:
		return true
	}
}

func (e *EntitlementMapAccess) Domain() EntitlementSetAccess {
	e.domainOnce.Do(func() {
		domain := common.MappedSliceWithNoDuplicates(
			e.Type.Relations,
			func(r EntitlementRelation) *EntitlementType {
				return r.Input
			},
		)
		e.domain = NewEntitlementSetAccess(domain, Conjunction)
	})
	return e.domain
}

func (e *EntitlementMapAccess) Codomain() EntitlementSetAccess {
	e.codomainOnce.Do(func() {
		codomain := common.MappedSliceWithNoDuplicates(
			e.Type.Relations,
			func(r EntitlementRelation) *EntitlementType {
				return r.Output
			},
		)
		e.codomain = NewEntitlementSetAccess(codomain, Conjunction)
	})
	return e.codomain
}

// produces the image set of a single entitlement through a map
// the image set of one element is always a conjunction
func (e *EntitlementMapAccess) entitlementImage(entitlement *EntitlementType) *EntitlementOrderedSet {
	image, ok := e.images.Load(entitlement)

	if ok {
		return image.(*EntitlementOrderedSet)
	}

	imageMap := orderedmap.New[EntitlementOrderedSet](0)
	for _, relation := range e.Type.Relations {
		if relation.Input.Equal(entitlement) {
			imageMap.Set(relation.Output, struct{}{})
		}
	}
	if e.Type.IncludesIdentity {
		imageMap.Set(entitlement, struct{}{})
	}

	e.images.Store(entitlement, imageMap)
	return imageMap
}

// Image applies all the entitlements in the `argumentAccess` to the function
// defined by the map in `e`, producing a new entitlement set of the image of the
// arguments.
func (e *EntitlementMapAccess) Image(inputs Access, astRange func() ast.Range) (Access, error) {

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
					Range: astRange(),
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

// PrimitiveAccess

type PrimitiveAccess ast.PrimitiveAccess

var _ Access = PrimitiveAccess(0)

func (PrimitiveAccess) isAccess() {}

func (PrimitiveAccess) IsPrimitiveAccess() bool {
	return true
}

func (PrimitiveAccess) ID() TypeID {
	panic(errors.NewUnreachableError())
}

func (a PrimitiveAccess) String() string {
	return ast.PrimitiveAccess(a).Description()
}

func (a PrimitiveAccess) QualifiedString() string {
	return ast.PrimitiveAccess(a).Description()
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
	// only access(self) access is guaranteed to be less permissive than entitlement-based access, but cannot appear in interfaces
	return ast.PrimitiveAccess(a) != ast.AccessSelf
}

func newEntitlementAccess(
	entitlements []Type,
	setKind EntitlementSetKind,
) Access {

	var setEntitlements []*EntitlementType
	var mapEntitlement *EntitlementMapType

	for _, entitlement := range entitlements {
		switch entitlement := entitlement.(type) {
		case *EntitlementType:
			if mapEntitlement != nil {
				panic(errors.NewDefaultUserError("mixed entitlement types"))
			}

			setEntitlements = append(setEntitlements, entitlement)

		case *EntitlementMapType:
			if len(setEntitlements) > 0 {
				panic(errors.NewDefaultUserError("mixed entitlement types"))
			}

			if mapEntitlement != nil {
				panic(errors.NewDefaultUserError("extra entitlement map type"))
			}

			mapEntitlement = entitlement

		default:
			panic(errors.NewDefaultUserError("invalid entitlement type: %s", entitlement))
		}
	}

	if len(setEntitlements) > 0 {
		return NewEntitlementSetAccess(setEntitlements, setKind)
	}

	if mapEntitlement != nil {
		return NewEntitlementMapAccess(mapEntitlement)
	}

	panic(errors.NewDefaultUserError("neither map entitlement nor set entitlements given"))
}
