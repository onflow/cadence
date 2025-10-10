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

package subtype_gen

// Predicate represents different types of predicates in rules.
type Predicate interface {
	isPredicate()
}

// AlwaysPredicate represents an always-true condition.
type AlwaysPredicate struct{}

var _ Predicate = AlwaysPredicate{}

func (a AlwaysPredicate) isPredicate() {}

// NeverPredicate represents a never-true condition.
type NeverPredicate struct{}

var _ Predicate = NeverPredicate{}

func (a NeverPredicate) isPredicate() {}

// IsResourcePredicate represents a resource type check.
type IsResourcePredicate struct {
	Expression Expression `yaml:"isResource"`
}

var _ Predicate = IsResourcePredicate{}

func (i IsResourcePredicate) isPredicate() {}

// IsAttachmentPredicate represents an attachment type check.
type IsAttachmentPredicate struct {
	Expression Expression `yaml:"isAttachment"`
}

var _ Predicate = IsAttachmentPredicate{}

func (i IsAttachmentPredicate) isPredicate() {}

// IsHashableStructPredicate represents a hashable struct type check.
type IsHashableStructPredicate struct {
	Expression Expression `yaml:"isHashableStruct"`
}

var _ Predicate = IsHashableStructPredicate{}

func (i IsHashableStructPredicate) isPredicate() {}

// IsStorablePredicate represents a storable type check.
type IsStorablePredicate struct {
	Expression Expression `yaml:"isStorable"`
}

var _ Predicate = IsStorablePredicate{}

func (i IsStorablePredicate) isPredicate() {}

// EqualsPredicate represents an equality check.
type EqualsPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = EqualsPredicate{}

func (e EqualsPredicate) isPredicate() {}

// SubtypePredicate represents a subtype check.
type SubtypePredicate struct {
	Sub   Expression `yaml:"sub"`
	Super Expression `yaml:"super"`
}

var _ Predicate = SubtypePredicate{}

func (s SubtypePredicate) isPredicate() {}

// AndPredicate represents a logical AND predicate.
type AndPredicate struct {
	Predicates []Predicate `yaml:"and"`
}

var _ Predicate = AndPredicate{}

func (a AndPredicate) isPredicate() {}

// OrPredicate represents a logical OR predicate.
type OrPredicate struct {
	Predicates []Predicate `yaml:"or"`
}

var _ Predicate = OrPredicate{}

func (o OrPredicate) isPredicate() {}

// NotPredicate represents a logical NOT predicate.
type NotPredicate struct {
	Predicate Predicate `yaml:"not"`
}

var _ Predicate = NotPredicate{}

func (n NotPredicate) isPredicate() {}

// PermitsPredicate represents a permits check.
type PermitsPredicate struct {
	Sub   Expression `yaml:"sub"`
	Super Expression `yaml:"super"`
}

var _ Predicate = PermitsPredicate{}

func (p PermitsPredicate) isPredicate() {}

// TypeParamsEqualPredicate represents a type parameters equality check.
type TypeParamsEqualPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = TypeParamsEqualPredicate{}

func (t TypeParamsEqualPredicate) isPredicate() {}

// ParamsContravariantPredicate represents a params contravariant check.
type ParamsContravariantPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = ParamsContravariantPredicate{}

func (p ParamsContravariantPredicate) isPredicate() {}

// ReturnCovariantPredicate represents a return covariant check.
type ReturnCovariantPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = ReturnCovariantPredicate{}

func (r ReturnCovariantPredicate) isPredicate() {}

// ConstructorEqualPredicate represents a constructor equality check.
type ConstructorEqualPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = ConstructorEqualPredicate{}

func (c ConstructorEqualPredicate) isPredicate() {}

// TypeAssertionPredicate represents a type assertion.
type TypeAssertionPredicate struct {
	Source Expression `yaml:"source"`
	Type   Type       `yaml:"type"`
}

var _ Predicate = TypeAssertionPredicate{}

func (e TypeAssertionPredicate) isPredicate() {}

type SetContainsPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = SetContainsPredicate{}

func (e SetContainsPredicate) isPredicate() {}

type IsIntersectionSubsetPredicate struct {
	Sub   Expression `yaml:"sub"`
	Super Expression `yaml:"super"`
}

var _ Predicate = IsIntersectionSubsetPredicate{}

func (p IsIntersectionSubsetPredicate) isPredicate() {}

type TypeArgumentsEqualPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = TypeArgumentsEqualPredicate{}

func (c TypeArgumentsEqualPredicate) isPredicate() {}

type IsParameterizedSubtypePredicate struct {
	Sub   Expression `yaml:"sub"`
	Super Expression `yaml:"super"`
}

var _ Predicate = IsParameterizedSubtypePredicate{}

func (c IsParameterizedSubtypePredicate) isPredicate() {}

// Predicates is a collection of predicates.
type Predicates struct {
	size       int
	index      int
	predicates []Predicate
}

func NewPredicateChain(predicates []Predicate) *Predicates {
	return &Predicates{
		size:       len(predicates),
		index:      0,
		predicates: predicates,
	}
}

func (p *Predicates) hasMore() bool {
	return p.index < p.size
}

func (p *Predicates) next() Predicate {
	predicate := p.predicates[p.index]
	p.index++
	return predicate
}
