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

// Predicate represents different types of predicates in rules
type Predicate interface {
	GetType() string
}

// AlwaysPredicate represents an always-true condition
type AlwaysPredicate struct{}

func (a AlwaysPredicate) GetType() string { return "always" }

// IsResourcePredicate represents a resource type check
type IsResourcePredicate struct {
	Expression Expression `yaml:"isResource"`
}

func (i IsResourcePredicate) GetType() string { return "isResource" }

// IsAttachmentPredicate represents an attachment type check
type IsAttachmentPredicate struct {
	Expression Expression `yaml:"isAttachment"`
}

func (i IsAttachmentPredicate) GetType() string { return "isAttachment" }

// IsHashableStructPredicate represents a hashable struct type check
type IsHashableStructPredicate struct {
	Expression Expression `yaml:"isHashableStruct"`
}

func (i IsHashableStructPredicate) GetType() string { return "isHashableStruct" }

// IsStorablePredicate represents a storable type check
type IsStorablePredicate struct {
	Expression Expression `yaml:"isStorable"`
}

func (i IsStorablePredicate) GetType() string { return "isStorable" }

// EqualsPredicate represents an equality check
type EqualsPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

func (e EqualsPredicate) GetType() string { return "equals" }

// SubtypePredicate represents a subtype check
type SubtypePredicate struct {
	Sub   Expression `yaml:"sub"`
	Super Expression `yaml:"super"`
}

func (s SubtypePredicate) GetType() string { return "subtype" }

// AndPredicate represents a logical AND predicate
type AndPredicate struct {
	Predicates []Predicate `yaml:"and"`
}

func (a AndPredicate) GetType() string { return "and" }

// OrPredicate represents a logical OR predicate
type OrPredicate struct {
	Predicates []Predicate `yaml:"or"`
}

func (o OrPredicate) GetType() string { return "or" }

// NotPredicate represents a logical NOT predicate
type NotPredicate struct {
	Predicate Predicate `yaml:"not"`
}

func (n NotPredicate) GetType() string { return "not" }

// PermitsPredicate represents a permits check
type PermitsPredicate struct {
	Sub   Expression `yaml:"sub"`
	Super Expression `yaml:"super"`
}

func (p PermitsPredicate) GetType() string { return "permits" }

// PurityPredicate represents a purity check
type PurityPredicate struct {
	EqualsOrView bool `yaml:"equals_or_view"`
}

func (p PurityPredicate) GetType() string { return "purity" }

// TypeParamsEqualPredicate represents a type parameters equality check
type TypeParamsEqualPredicate struct{}

func (t TypeParamsEqualPredicate) GetType() string { return "typeParamsEqual" }

// ParamsContravariantPredicate represents a params contravariant check
type ParamsContravariantPredicate struct{}

func (p ParamsContravariantPredicate) GetType() string { return "paramsContravariant" }

// ReturnCovariantPredicate represents a return covariant check
type ReturnCovariantPredicate struct{}

func (r ReturnCovariantPredicate) GetType() string { return "returnCovariant" }

// ConstructorEqualPredicate represents a constructor equality check
type ConstructorEqualPredicate struct{}

func (c ConstructorEqualPredicate) GetType() string { return "constructorEqual" }

// TypeAssertionPredicate represents an equality check
type TypeAssertionPredicate struct {
	Source  Expression `yaml:"source"`
	Type    Type       `yaml:"type"`
	IfMatch *Predicate `yaml:"ifMatch"`
}

func (e TypeAssertionPredicate) GetType() string { return "mustType" }

type SetContainsPredicate struct {
	Source Expression `yaml:"source"`
	Target Expression `yaml:"target"`
}

func (e SetContainsPredicate) GetType() string { return "setContains" }
