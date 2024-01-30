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

// SubtypeTypeBound

type SubtypeTypeBound struct {
	Type Type
}

var _ TypeBound = SubtypeTypeBound{}

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

// StrictSubtypeTypeBound

type StrictSubtypeTypeBound struct {
	Type Type
}

var _ TypeBound = StrictSubtypeTypeBound{}

func (StrictSubtypeTypeBound) isTypeBound() {}

func (b StrictSubtypeTypeBound) Satisfies(ty Type) bool {
	return IsStrictSubType(ty, b.Type)
}

func (b StrictSubtypeTypeBound) HasInvalidType() bool {
	return b.Type.IsInvalidType()
}

func (b StrictSubtypeTypeBound) Equal(bound TypeBound) bool {
	other, ok := bound.(StrictSubtypeTypeBound)
	if !ok {
		return false
	}
	return b.Type.Equal(other.Type)
}

func (b StrictSubtypeTypeBound) CheckInstantiated(
	pos ast.HasPosition,
	memoryGauge common.MemoryGauge,
	report func(err error),
) {
	b.Type.CheckInstantiated(pos, memoryGauge, report)
}

func (b StrictSubtypeTypeBound) Map(
	gauge common.MemoryGauge,
	typeParamMap map[*TypeParameter]*TypeParameter,
	f func(Type) Type,
) TypeBound {
	return SubtypeTypeBound{
		Type: b.Type.Map(gauge, typeParamMap, f),
	}
}

func (b StrictSubtypeTypeBound) TypeAnnotationState() TypeAnnotationState {
	return b.Type.TypeAnnotationState()
}

func (b StrictSubtypeTypeBound) RewriteWithIntersectionTypes() (result TypeBound, rewritten bool) {
	rewrittenType, rewritten := b.Type.RewriteWithIntersectionTypes()
	if rewritten {
		return StrictSubtypeTypeBound{
			Type: rewrittenType,
		}, true
	}
	return b, false
}

// ConjunctionTypeBound

type ConjunctionTypeBound struct {
	TypeBounds []TypeBound
}

var _ TypeBound = ConjunctionTypeBound{}

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

// SupertypeTypeBound

type SupertypeTypeBound struct {
	Type Type
}

var _ TypeBound = SupertypeTypeBound{}

func (SupertypeTypeBound) isTypeBound() {}

func (b SupertypeTypeBound) Satisfies(ty Type) bool {
	return IsSubType(b.Type, ty)
}

func (b SupertypeTypeBound) HasInvalidType() bool {
	return b.Type.IsInvalidType()
}

func (b SupertypeTypeBound) Equal(bound TypeBound) bool {
	other, ok := bound.(SupertypeTypeBound)
	if !ok {
		return false
	}
	return b.Type.Equal(other.Type)
}

func (b SupertypeTypeBound) CheckInstantiated(
	pos ast.HasPosition,
	memoryGauge common.MemoryGauge,
	report func(err error),
) {
	b.Type.CheckInstantiated(pos, memoryGauge, report)
}

func (b SupertypeTypeBound) Map(
	gauge common.MemoryGauge,
	typeParamMap map[*TypeParameter]*TypeParameter,
	f func(Type) Type,
) TypeBound {
	return SupertypeTypeBound{
		Type: b.Type.Map(gauge, typeParamMap, f),
	}
}

func (b SupertypeTypeBound) TypeAnnotationState() TypeAnnotationState {
	return b.Type.TypeAnnotationState()
}

func (b SupertypeTypeBound) RewriteWithIntersectionTypes() (result TypeBound, rewritten bool) {
	rewrittenType, rewritten := b.Type.RewriteWithIntersectionTypes()
	if rewritten {
		return SupertypeTypeBound{
			Type: rewrittenType,
		}, true
	}
	return b, false
}

// StrictSupertypeTypeBound

type StrictSupertypeTypeBound struct {
	Type Type
}

var _ TypeBound = StrictSupertypeTypeBound{}

func (StrictSupertypeTypeBound) isTypeBound() {}

func (b StrictSupertypeTypeBound) Satisfies(ty Type) bool {
	return IsStrictSubType(b.Type, ty)
}

func (b StrictSupertypeTypeBound) HasInvalidType() bool {
	return b.Type.IsInvalidType()
}

func (b StrictSupertypeTypeBound) Equal(bound TypeBound) bool {
	other, ok := bound.(StrictSupertypeTypeBound)
	if !ok {
		return false
	}
	return b.Type.Equal(other.Type)
}

func (b StrictSupertypeTypeBound) CheckInstantiated(
	pos ast.HasPosition,
	memoryGauge common.MemoryGauge,
	report func(err error),
) {
	b.Type.CheckInstantiated(pos, memoryGauge, report)
}

func (b StrictSupertypeTypeBound) Map(
	gauge common.MemoryGauge,
	typeParamMap map[*TypeParameter]*TypeParameter,
	f func(Type) Type,
) TypeBound {
	return StrictSupertypeTypeBound{
		Type: b.Type.Map(gauge, typeParamMap, f),
	}
}

func (b StrictSupertypeTypeBound) TypeAnnotationState() TypeAnnotationState {
	return b.Type.TypeAnnotationState()
}

func (b StrictSupertypeTypeBound) RewriteWithIntersectionTypes() (result TypeBound, rewritten bool) {
	rewrittenType, rewritten := b.Type.RewriteWithIntersectionTypes()
	if rewritten {
		return StrictSupertypeTypeBound{
			Type: rewrittenType,
		}, true
	}
	return b, false
}
