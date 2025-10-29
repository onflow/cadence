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

// EqualsPredicate represents an equality check using `==` operator.
type EqualsPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = EqualsPredicate{}

func (e EqualsPredicate) isPredicate() {}

// DeepEqualsPredicate represents a deep equality check, defined with `Equals` method.
type DeepEqualsPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = DeepEqualsPredicate{}

func (e DeepEqualsPredicate) isPredicate() {}

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

// ReturnCovariantPredicate represents a return covariant check.
type ReturnCovariantPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

var _ Predicate = ReturnCovariantPredicate{}

func (r ReturnCovariantPredicate) isPredicate() {}

// TypeAssertionPredicate represents a type assertion.
type TypeAssertionPredicate struct {
	Source Expression `yaml:"source"`
	Type   Type       `yaml:"type"`
}

var _ Predicate = TypeAssertionPredicate{}

func (e TypeAssertionPredicate) isPredicate() {}

type SetContainsPredicate struct {
	Set     Expression `yaml:"set"`
	Element Expression `yaml:"element"`
}

var _ Predicate = SetContainsPredicate{}

func (e SetContainsPredicate) isPredicate() {}

type IsIntersectionSubsetPredicate struct {
	Sub   Expression `yaml:"sub"`
	Super Expression `yaml:"super"`
}

var _ Predicate = IsIntersectionSubsetPredicate{}

func (p IsIntersectionSubsetPredicate) isPredicate() {}

type IsParameterizedSubtypePredicate struct {
	Sub   Expression `yaml:"sub"`
	Super Expression `yaml:"super"`
}

var _ Predicate = IsParameterizedSubtypePredicate{}

func (c IsParameterizedSubtypePredicate) isPredicate() {}

type ForAllPredicate struct {
	Source    Expression `yaml:"source"`
	Target    Expression `yaml:"target"`
	Predicate Predicate  `yaml:"predicate"`
}

var _ Predicate = ForAllPredicate{}

func (c ForAllPredicate) isPredicate() {}

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
