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
	"github.com/onflow/cadence/runtime/errors"
	"golang.org/x/exp/maps"
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
	return a.AccessKeyword()
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
	var domain map[*EntitlementType]struct{} = make(map[*EntitlementType]struct{})
	for _, relation := range e.Type.Relations {
		domain[relation.Input] = struct{}{}
	}
	return NewEntitlementSetAccess(maps.Keys(domain), Disjunction)
}

func (e EntitlementMapAccess) Codomain() EntitlementSetAccess {
	var codomain map[*EntitlementType]struct{} = make(map[*EntitlementType]struct{})
	for _, relation := range e.Type.Relations {
		codomain[relation.Output] = struct{}{}
	}
	return NewEntitlementSetAccess(maps.Keys(codomain), Conjunction)
}

func (e EntitlementMapAccess) entitlementImage(entitlement *EntitlementType) (output *EntitlementOrderedSet) {
	output = orderedmap.New[EntitlementOrderedSet](0)
	for _, relation := range e.Type.Relations {
		if relation.Input.Equal(entitlement) {
			output.Set(relation.Output, struct{}{})
		}
	}
	return
}

func (e EntitlementMapAccess) entitlementPreImage(entitlement *EntitlementType) (input *EntitlementOrderedSet) {
	input = orderedmap.New[EntitlementOrderedSet](0)
	for _, relation := range e.Type.Relations {
		if relation.Output.Equal(entitlement) {
			input.Set(relation.Input, struct{}{})
		}
	}
	return
}

// Image applies all the entitlements in the `argumentAccess` to the function
// defined by the map in `e`, producing a new entitlement set of the image of the
// arguments
func (e EntitlementMapAccess) Image(inputs Access, astRange ast.Range) (Access, error) {
	switch inputs := inputs.(type) {
	// primitive access always passes trivially through the map
	case PrimitiveAccess:
		return inputs, nil
	case EntitlementSetAccess:
		var output *EntitlementOrderedSet = orderedmap.New[EntitlementOrderedSet](inputs.Entitlements.Len())
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
	// it should be impossible to obtain a concrete reference with a mapped entitlement authorization
	panic(errors.NewUnreachableError())
}

// Preimage applies all the entitlements in the `argumentAccess` to the inverse of the function
// defined by the map in `e`, producing a new entitlement set of the preimage of the
// arguments
func (e EntitlementMapAccess) Preimage(outputs Access, astRange ast.Range) (Access, error) {
	switch outputs := outputs.(type) {
	// primitive access always passes trivially through the map
	case PrimitiveAccess:
		return outputs, nil
	case EntitlementSetAccess:
		var input *EntitlementOrderedSet = orderedmap.New[EntitlementOrderedSet](outputs.Entitlements.Len())
		var err error = nil
		outputs.Entitlements.Foreach(func(entitlement *EntitlementType, _ struct{}) {
			entitlementPreImage := e.entitlementPreImage(entitlement)
			// the preimage of a single element is always a disjunctive set; consider a mapping
			// M defined as Y -> X, Z -> X, B -> A, C -> A. M^-1(X) = Y | Z and M^-1(A) = B | C, since either an
			// Y or a Z can result in an X and either a B or a C can result in an A.
			// Thus M^-1(X | A) would be ((Y | Z) & (B | C)), which is a conjunction of two disjunctions,
			// which is too complex to be represented in Cadence as a type. Thus whenever such a type
			// would arise, we raise an error instead
			if (outputs.SetKind == Conjunction && outputs.Entitlements.Len() > 1) && entitlementPreImage.Len() > 1 {
				err = &UnrepresentableEntitlementMapOutputError{
					Input: outputs,
					Map:   e.Type,
					Range: astRange,
				}
			}
			input.SetAll(entitlementPreImage)
		})
		if err != nil {
			return nil, err
		}
		// the preimage of a set through a map is the disjunction of all the input sets
		if input.Len() == 0 {
			return UnauthorizedAccess, nil
		}
		setKind := outputs.SetKind
		if outputs.SetKind == Conjunction && outputs.Entitlements.Len() == 1 {
			setKind = Disjunction
		}
		return EntitlementSetAccess{
			Entitlements: input,
			SetKind:      setKind,
		}, nil
	}
	// it should be impossible to obtain a concrete reference with a mapped entitlement authorization
	panic(errors.NewUnreachableError())
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
