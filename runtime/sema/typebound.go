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
	"github.com/onflow/cadence/runtime/common"
)

// TypeBound

type TypeBound interface {
	isTypeBound()
	Satisfies(ty Type) bool
	HasInvalidType() bool
	Equal(TypeBound) bool
	CheckInstantiated(
		pos ast.HasPosition,
		memoryGauge common.MemoryGauge,
		report func(err error),
	)
	Map(
		gauge common.MemoryGauge,
		typeParamMap map[*TypeParameter]*TypeParameter,
		f func(Type) Type,
	) TypeBound
	TypeAnnotationState() TypeAnnotationState
	RewriteWithIntersectionTypes() (result TypeBound, rewritten bool)
}

// SubtypeTypeBound(T) expresses the requirement that
// ∀U, U <= T

type SubtypeTypeBound struct {
	Type Type
}

var _ TypeBound = SubtypeTypeBound{}

func NewSubtypeTypeBound(ty Type) TypeBound {
	return SubtypeTypeBound{Type: ty}
}

func (SubtypeTypeBound) isTypeBound() {}

func (b SubtypeTypeBound) Satisfies(ty Type) bool {
	return IsSubType(ty, b.Type)
}

func (b SubtypeTypeBound) HasInvalidType() bool {
	return b.Type.IsInvalidType()
}

func (b SubtypeTypeBound) Equal(bound TypeBound) bool {
	other, ok := bound.(SubtypeTypeBound)
	if !ok {
		return false
	}
	return b.Type.Equal(other.Type)
}

func (b SubtypeTypeBound) CheckInstantiated(
	pos ast.HasPosition,
	memoryGauge common.MemoryGauge,
	report func(err error),
) {
	b.Type.CheckInstantiated(pos, memoryGauge, report)
}

func (b SubtypeTypeBound) Map(
	gauge common.MemoryGauge,
	typeParamMap map[*TypeParameter]*TypeParameter,
	f func(Type) Type,
) TypeBound {
	return SubtypeTypeBound{
		Type: b.Type.Map(gauge, typeParamMap, f),
	}
}

func (b SubtypeTypeBound) TypeAnnotationState() TypeAnnotationState {
	return b.Type.TypeAnnotationState()
}

func (b SubtypeTypeBound) RewriteWithIntersectionTypes() (result TypeBound, rewritten bool) {
	rewrittenType, rewritten := b.Type.RewriteWithIntersectionTypes()
	if rewritten {
		return SubtypeTypeBound{
			Type: rewrittenType,
		}, true
	}
	return b, false
}

// EqualTypeBound expresses the requirement that
// ∀U, U = T

type EqualTypeBound struct {
	Type Type
}

var _ TypeBound = EqualTypeBound{}

func NewEqualTypeBound(ty Type) TypeBound {
	return EqualTypeBound{Type: ty}
}

func (EqualTypeBound) isTypeBound() {}

func (b EqualTypeBound) Satisfies(ty Type) bool {
	return ty.Equal(b.Type)
}

func (b EqualTypeBound) HasInvalidType() bool {
	return b.Type.IsInvalidType()
}

func (b EqualTypeBound) Equal(bound TypeBound) bool {
	other, ok := bound.(EqualTypeBound)
	if !ok {
		return false
	}
	return b.Type.Equal(other.Type)
}

func (b EqualTypeBound) CheckInstantiated(
	pos ast.HasPosition,
	memoryGauge common.MemoryGauge,
	report func(err error),
) {
	b.Type.CheckInstantiated(pos, memoryGauge, report)
}

func (b EqualTypeBound) Map(
	gauge common.MemoryGauge,
	typeParamMap map[*TypeParameter]*TypeParameter,
	f func(Type) Type,
) TypeBound {
	return EqualTypeBound{
		Type: b.Type.Map(gauge, typeParamMap, f),
	}
}

func (b EqualTypeBound) TypeAnnotationState() TypeAnnotationState {
	return b.Type.TypeAnnotationState()
}

func (b EqualTypeBound) RewriteWithIntersectionTypes() (result TypeBound, rewritten bool) {
	rewrittenType, rewritten := b.Type.RewriteWithIntersectionTypes()
	if rewritten {
		return EqualTypeBound{
			Type: rewrittenType,
		}, true
	}
	return b, false
}

// NegationTypeBound(B) expresses the requirement that
// ∀U, !B(U)

type NegationTypeBound struct {
	NegatedBound TypeBound
}

var _ TypeBound = NegationTypeBound{}

func NewNegationTypeBound(bound TypeBound) TypeBound {
	return NegationTypeBound{NegatedBound: bound}
}

func (NegationTypeBound) isTypeBound() {}

func (b NegationTypeBound) Satisfies(ty Type) bool {
	return !b.NegatedBound.Satisfies(ty)
}

func (b NegationTypeBound) HasInvalidType() bool {
	return b.NegatedBound.HasInvalidType()
}

func (b NegationTypeBound) Equal(bound TypeBound) bool {
	other, ok := bound.(NegationTypeBound)
	if !ok {
		return false
	}
	return b.NegatedBound.Equal(other.NegatedBound)
}

func (b NegationTypeBound) CheckInstantiated(
	pos ast.HasPosition,
	memoryGauge common.MemoryGauge,
	report func(err error),
) {
	b.NegatedBound.CheckInstantiated(pos, memoryGauge, report)
}

func (b NegationTypeBound) Map(
	gauge common.MemoryGauge,
	typeParamMap map[*TypeParameter]*TypeParameter,
	f func(Type) Type,
) TypeBound {
	return NegationTypeBound{
		NegatedBound: b.NegatedBound.Map(gauge, typeParamMap, f),
	}
}

func (b NegationTypeBound) TypeAnnotationState() TypeAnnotationState {
	return b.NegatedBound.TypeAnnotationState()
}

func (b NegationTypeBound) RewriteWithIntersectionTypes() (result TypeBound, rewritten bool) {
	rewrittenBound, rewritten := b.NegatedBound.RewriteWithIntersectionTypes()
	if rewritten {
		return NegationTypeBound{
			NegatedBound: rewrittenBound,
		}, true
	}
	return b, false
}

// ConjunctionTypeBound(B1, ..., Bn) expresses the requirement that
// ∀U, B1(U) & ... & Bn(U)

type ConjunctionTypeBound struct {
	TypeBounds []TypeBound
}

var _ TypeBound = ConjunctionTypeBound{}

func NewConjunctionTypeBound(typeBounds []TypeBound) TypeBound {
	return ConjunctionTypeBound{TypeBounds: typeBounds}
}

func (ConjunctionTypeBound) isTypeBound() {}

func (b ConjunctionTypeBound) Satisfies(ty Type) bool {
	for _, typeBound := range b.TypeBounds {
		if !typeBound.Satisfies(ty) {
			return false
		}
	}
	return true
}

func (b ConjunctionTypeBound) HasInvalidType() bool {
	for _, typeBound := range b.TypeBounds {
		if typeBound.HasInvalidType() {
			return true
		}
	}
	return false
}

func (b ConjunctionTypeBound) Equal(bound TypeBound) bool {
	other, ok := bound.(ConjunctionTypeBound)
	if !ok {
		return false
	}

	if len(b.TypeBounds) != len(other.TypeBounds) {
		return false
	}

	for i, typeBound := range b.TypeBounds {
		otherTypeBound := other.TypeBounds[i]
		if !typeBound.Equal(otherTypeBound) {
			return false
		}
	}

	return true
}

func (b ConjunctionTypeBound) CheckInstantiated(
	pos ast.HasPosition,
	memoryGauge common.MemoryGauge,
	report func(err error),
) {
	for _, typeBound := range b.TypeBounds {
		typeBound.CheckInstantiated(pos, memoryGauge, report)
	}
}

func (b ConjunctionTypeBound) Map(
	gauge common.MemoryGauge,
	typeParamMap map[*TypeParameter]*TypeParameter,
	f func(Type) Type,
) TypeBound {
	newTypeBounds := make([]TypeBound, 0, len(b.TypeBounds))
	for _, typeBound := range b.TypeBounds {
		newTypeBounds = append(
			newTypeBounds,
			typeBound.Map(gauge, typeParamMap, f),
		)
	}
	return ConjunctionTypeBound{
		TypeBounds: newTypeBounds,
	}
}

func (b ConjunctionTypeBound) TypeAnnotationState() TypeAnnotationState {
	for _, typeBound := range b.TypeBounds {
		state := typeBound.TypeAnnotationState()
		if state != TypeAnnotationStateValid {
			return state
		}
	}
	return TypeAnnotationStateValid
}

func (b ConjunctionTypeBound) RewriteWithIntersectionTypes() (result TypeBound, rewritten bool) {
	rewrittenTypeBounds := make([]TypeBound, 0, len(b.TypeBounds))
	for _, typeBound := range b.TypeBounds {
		rewrittenTypeBound, currentRewritten := typeBound.RewriteWithIntersectionTypes()
		if currentRewritten {
			rewritten = true
			rewrittenTypeBounds = append(rewrittenTypeBounds, rewrittenTypeBound)
		} else {
			rewrittenTypeBounds = append(rewrittenTypeBounds, typeBound)
		}
	}
	if rewritten {
		return ConjunctionTypeBound{
			TypeBounds: rewrittenTypeBounds,
		}, true
	} else {
		return b, false
	}
}

// Any other kinds of type bounds we might wish to express can be
// written as the composition of `<=`, `=`, `!` and `&`. Technically, `=` is not
// really even necessary, as `U = T` is equivalent to `U <= T & T <= U`, but for
// performance reasons we give it its own basic bound

// `U <= T && !(T = U) ==> U < T`
func NewStrictSubtypeTypeBound(ty Type) TypeBound {
	subtypeBound := NewSubtypeTypeBound(ty)
	nonEqualBound := NewNegationTypeBound(NewEqualTypeBound(ty))
	return NewConjunctionTypeBound([]TypeBound{subtypeBound, nonEqualBound})
}

// `!(U <= T) ==> U > T`
func NewStrictSupertypeTypeBound(ty Type) TypeBound {
	return NewNegationTypeBound(NewSubtypeTypeBound(ty))
}

// `!(U < T) ==> U >= T`
func NewSupertypeTypeBound(ty Type) TypeBound {
	return NewNegationTypeBound(NewStrictSubtypeTypeBound(ty))
}

// `!(!B1 & ... & !Bn) ==> B1 || ... || Bn`
func NewDisjunctionTypeBound(typeBounds []TypeBound) TypeBound {
	var negatedTypeBounds []TypeBound
	for _, bound := range typeBounds {
		negatedTypeBounds = append(negatedTypeBounds, NewNegationTypeBound(bound))
	}
	return NewNegationTypeBound(NewConjunctionTypeBound(negatedTypeBounds))
}
