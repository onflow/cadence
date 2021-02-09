/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
)

type ResourceUse struct {
	UseAfterInvalidationReported bool
}

type ResourceUses struct {
	positions map[ast.Position]ResourceUse
}

// ForEach calls the given function for each resource use in the set.
// It can be used to iterate over all uses.
//
func (rus ResourceUses) ForEach(cb func(pos ast.Position, use ResourceUse) error) error {
	for pos, use := range rus.positions {
		err := cb(pos, use)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rus ResourceUses) Contains(pos ast.Position) bool {
	if rus.positions == nil {
		return false
	}
	_, ok := rus.positions[pos]
	return ok
}

// Insert adds the given resource use to this set.
//
func (rus *ResourceUses) Insert(pos ast.Position) {
	if rus.Contains(pos) {
		return
	}
	if rus.positions == nil {
		rus.positions = map[ast.Position]ResourceUse{}
	}
	rus.positions[pos] = ResourceUse{}
}

// MarkUseAfterInvalidationReported marks the use after invalidation
// of the resource at the given position as reported.
//
func (rus *ResourceUses) MarkUseAfterInvalidationReported(pos ast.Position) {
	if rus.positions == nil {
		rus.positions = map[ast.Position]ResourceUse{}
	}
	use := rus.positions[pos]
	use.UseAfterInvalidationReported = true
	rus.positions[pos] = use
}

// IsUseAfterInvalidationReported returns true if the use after invalidation
// of the resource at the given position is reported.
//
func (rus ResourceUses) IsUseAfterInvalidationReported(pos ast.Position) bool {
	if rus.positions == nil {
		return false
	}
	use := rus.positions[pos]
	return use.UseAfterInvalidationReported
}

// Merge adds the resource uses of the given set to this set.
//
func (rus *ResourceUses) Merge(other ResourceUses) {
	if rus.positions == nil {
		rus.positions = map[ast.Position]ResourceUse{}
	}

	_ = other.ForEach(func(pos ast.Position, use ResourceUse) error {
		// TODO: really overwrite potential existing use with UseAfterInvalidationReported == true
		//   with new UseAfterInvalidationReported == false ?
		rus.positions[pos] = use

		// NOTE: when changing this function to return an error,
		// also return it from the outer function,
		// as the outer error is currently ignored!
		return nil
	})
}

// Size returns the number of resource uses in this set.
//
func (rus ResourceUses) Size() int {
	return len(rus.positions)
}
