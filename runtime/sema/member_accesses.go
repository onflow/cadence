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
	"github.com/onflow/cadence/runtime/common/intervalst"
)

type MemberAccess struct {
	StartPos     Position
	EndPos       Position
	AccessedType Type
}

type MemberAccesses struct {
	tree *intervalst.IntervalST
}

func NewMemberAccesses() *MemberAccesses {
	return &MemberAccesses{
		tree: &intervalst.IntervalST{},
	}
}

func (m *MemberAccesses) Put(startPos, endPos ast.Position, accessedType Type) {
	access := MemberAccess{
		StartPos:     ASTToSemaPosition(startPos),
		EndPos:       ASTToSemaPosition(endPos),
		AccessedType: accessedType,
	}
	interval := intervalst.NewInterval(
		access.StartPos,
		access.EndPos,
	)
	m.tree.Put(interval, access)
}

func (m *MemberAccesses) Find(pos Position) *MemberAccess {
	interval, value := m.tree.Search(pos)
	if interval == nil {
		return nil
	}
	access := value.(MemberAccess)
	return &access
}

func (m *MemberAccesses) All() []MemberAccess {
	values := m.tree.Values()
	accesses := make([]MemberAccess, len(values))
	for i, value := range values {
		accesses[i] = value.(MemberAccess)
	}
	return accesses
}
