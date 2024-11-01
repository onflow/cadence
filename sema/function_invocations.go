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
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common/intervalst"
)

type FunctionInvocation struct {
	FunctionType               *FunctionType
	TrailingSeparatorPositions []ast.Position
	StartPos                   Position
	EndPos                     Position
}

type FunctionInvocations struct {
	tree *intervalst.IntervalST[FunctionInvocation]
}

func NewFunctionInvocations() *FunctionInvocations {
	return &FunctionInvocations{
		tree: &intervalst.IntervalST[FunctionInvocation]{},
	}
}

func (f *FunctionInvocations) Put(
	startPos, endPos ast.Position,
	functionType *FunctionType,
	trailingSeparatorPositions []ast.Position,
) {
	invocation := FunctionInvocation{
		StartPos:                   ASTToSemaPosition(startPos),
		EndPos:                     ASTToSemaPosition(endPos),
		FunctionType:               functionType,
		TrailingSeparatorPositions: trailingSeparatorPositions,
	}
	interval := intervalst.NewInterval(
		invocation.StartPos,
		invocation.EndPos,
	)
	f.tree.Put(interval, invocation)
}

func (f *FunctionInvocations) Find(pos Position) *FunctionInvocation {
	_, invocation, present := f.tree.Search(pos)
	if !present {
		return nil
	}
	return &invocation
}
