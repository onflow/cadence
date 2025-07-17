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

package sema

import (
	"github.com/onflow/cadence/errors"
)

//go:generate stringer -type=ResourceInvalidationKind

type ResourceInvalidationKind uint

const (
	ResourceInvalidationKindUnknown ResourceInvalidationKind = iota
	ResourceInvalidationKindMoveDefinite
	ResourceInvalidationKindMovePotential
	ResourceInvalidationKindMoveTemporary
	ResourceInvalidationKindDestroyDefinite
	ResourceInvalidationKindDestroyPotential
)

func (i ResourceInvalidationKind) DetailedNoun() string {
	switch i {
	case ResourceInvalidationKindMoveDefinite:
		return "definite move"
	case ResourceInvalidationKindMoveTemporary:
		return "temporary move"
	case ResourceInvalidationKindMovePotential:
		return "potential move"
	case ResourceInvalidationKindDestroyDefinite:
		return "definite destruction"
	case ResourceInvalidationKindDestroyPotential:
		return "potential destruction"
	}

	panic(errors.NewUnreachableError())
}

func (i ResourceInvalidationKind) CoarseNoun() string {
	switch i {
	case ResourceInvalidationKindMoveDefinite,
		ResourceInvalidationKindMoveTemporary,
		ResourceInvalidationKindMovePotential:

		return "move"

	case ResourceInvalidationKindDestroyDefinite,
		ResourceInvalidationKindDestroyPotential:

		return "destruction"
	}

	panic(errors.NewUnreachableError())
}

func (i ResourceInvalidationKind) IsDefinite() bool {
	switch i {
	case ResourceInvalidationKindMoveDefinite,
		ResourceInvalidationKindDestroyDefinite:

		return true

	case ResourceInvalidationKindMovePotential,
		ResourceInvalidationKindDestroyPotential,
		ResourceInvalidationKindMoveTemporary,
		ResourceInvalidationKindUnknown:

		return false
	}

	panic(errors.NewUnreachableError())
}

func (i ResourceInvalidationKind) AsPotential() ResourceInvalidationKind {
	switch i {
	case ResourceInvalidationKindMoveDefinite:
		return ResourceInvalidationKindMovePotential
	case ResourceInvalidationKindDestroyDefinite:
		return ResourceInvalidationKindDestroyPotential
	}

	return i
}

func (i ResourceInvalidationKind) CoarsePassiveVerb() string {
	switch i {
	case ResourceInvalidationKindMoveDefinite,
		ResourceInvalidationKindMoveTemporary,
		ResourceInvalidationKindMovePotential:

		return "moved"

	case ResourceInvalidationKindDestroyDefinite,
		ResourceInvalidationKindDestroyPotential:

		return "destroyed"
	}

	panic(errors.NewUnreachableError())
}
