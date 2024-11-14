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

package vm

import (
	"github.com/onflow/atree"
)

type Value interface {
	isValue()
	StaticType(*Config) StaticType
	Transfer(
		config *Config,
		address atree.Address,
		remove bool,
		storable atree.Storable,
	) Value
	String() string
}

type MemberAccessibleValue interface {
	GetMember(config *Config, name string) Value
	SetMember(conf *Config, name string, value Value)
}

type ResourceKindedValue interface {
	Value
	//Destroy(interpreter *Interpreter, locationRange LocationRange)
	//IsDestroyed() bool
	//isInvalidatedResource(*Interpreter) bool
	IsResourceKinded() bool
}

// ReferenceTrackedResourceKindedValue is a resource-kinded value
// that must be tracked when a reference of it is taken.
type ReferenceTrackedResourceKindedValue interface {
	ResourceKindedValue
	IsReferenceTrackedResourceKindedValue()
	ValueID() atree.ValueID
	IsStaleResource() bool
}
