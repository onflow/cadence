/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"math"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type PositionInfo struct {
	Occurrences         *Occurrences
	VariableOrigins     map[*Variable]*Origin
	MemberOrigins       map[Type]map[string]*Origin
	MemberAccesses      *MemberAccesses
	Ranges              *Ranges
	FunctionInvocations *FunctionInvocations
}

func NewPositionInfo() *PositionInfo {
	return &PositionInfo{
		MemberOrigins:       map[Type]map[string]*Origin{},
		VariableOrigins:     map[*Variable]*Origin{},
		Occurrences:         NewOccurrences(),
		MemberAccesses:      NewMemberAccesses(),
		Ranges:              NewRanges(),
		FunctionInvocations: NewFunctionInvocations(),
	}
}

func (i *PositionInfo) recordVariableReferenceOccurrence(
	memoryGauge common.MemoryGauge,
	startPos ast.Position,
	endPos ast.Position,
	variable *Variable,
) {
	origin, ok := i.VariableOrigins[variable]
	if !ok {
		originStartPos := variable.Pos
		var originEndPos *ast.Position
		if originStartPos != nil {
			pos := originStartPos.Shifted(memoryGauge, len(variable.Identifier)-1)
			originEndPos = &pos
		}
		origin = &Origin{
			Type:            variable.Type,
			DeclarationKind: variable.DeclarationKind,
			StartPos:        originStartPos,
			EndPos:          originEndPos,
			DocString:       variable.DocString,
		}
		i.VariableOrigins[variable] = origin
	}
	i.Occurrences.Put(startPos, endPos, origin)
}

func (i *PositionInfo) recordFieldDeclarationOrigin(
	memoryGauge common.MemoryGauge,
	identifier ast.Identifier,
	fieldType Type,
	docString string,
) *Origin {
	startPosition := identifier.StartPosition()
	endPosition := identifier.EndPosition(memoryGauge)

	origin := &Origin{
		Type:            fieldType,
		DeclarationKind: common.DeclarationKindField,
		StartPos:        &startPosition,
		EndPos:          &endPosition,
		DocString:       docString,
	}

	i.Occurrences.Put(
		startPosition,
		endPosition,
		origin,
	)

	return origin
}

func (i *PositionInfo) recordFunctionDeclarationOrigin(
	memoryGauge common.MemoryGauge,
	function *ast.FunctionDeclaration,
	functionType *FunctionType,
) *Origin {

	startPosition := function.Identifier.StartPosition()
	endPosition := function.Identifier.EndPosition(memoryGauge)

	origin := &Origin{
		Type:            functionType,
		DeclarationKind: common.DeclarationKindFunction,
		StartPos:        &startPosition,
		EndPos:          &endPosition,
		DocString:       function.DocString,
	}

	i.Occurrences.Put(
		startPosition,
		endPosition,
		origin,
	)
	return origin
}

func (i *PositionInfo) recordVariableDeclarationOccurrence(
	memoryGauge common.MemoryGauge,
	name string,
	variable *Variable,
) {
	if variable.Pos == nil {
		return
	}
	startPos := *variable.Pos
	endPos := variable.Pos.Shifted(memoryGauge, len(name)-1)
	i.recordVariableReferenceOccurrence(memoryGauge, startPos, endPos, variable)
}

func (i *PositionInfo) recordMemberOrigins(ty Type, origins map[string]*Origin) {
	i.MemberOrigins[ty] = origins
}

func (i *PositionInfo) recordGlobalRange(
	memoryGauge common.MemoryGauge,
	name string,
	variable *Variable,
) {
	i.Ranges.Put(
		ast.NewPosition(memoryGauge, 0, 1, 0),
		ast.NewPosition(memoryGauge, 0, math.MaxInt32, 0),
		Range{
			Identifier:      name,
			Type:            variable.Type,
			DeclarationKind: variable.DeclarationKind,
			DocString:       variable.DocString,
		},
	)
}

func (i *PositionInfo) recordParameterRange(
	startPos ast.Position,
	endPos ast.Position,
	parameter *Parameter,
) {
	i.Ranges.Put(
		startPos,
		endPos,
		Range{
			Identifier:      parameter.Identifier,
			Type:            parameter.TypeAnnotation.Type,
			DeclarationKind: common.DeclarationKindParameter,
		},
	)
}

func (i *PositionInfo) recordFunctionInvocation(
	invocationExpression *ast.InvocationExpression,
	functionType *FunctionType,
) {
	arguments := invocationExpression.Arguments

	trailingSeparatorPositions := make([]ast.Position, 0, len(arguments))

	for _, argument := range arguments {
		trailingSeparatorPositions = append(
			trailingSeparatorPositions,
			argument.TrailingSeparatorPos,
		)
	}

	i.FunctionInvocations.Put(
		invocationExpression.ArgumentsStartPos,
		invocationExpression.EndPos,
		functionType,
		trailingSeparatorPositions,
	)
}

func (i *PositionInfo) recordMemberAccess(
	memoryGauge common.MemoryGauge,
	expression *ast.MemberExpression,
	memberAccessType Type,
) {
	i.MemberAccesses.Put(
		expression.AccessPos,
		expression.EndPosition(memoryGauge),
		memberAccessType,
	)
}

func (i *PositionInfo) recordMemberOccurrence(
	accessedType Type,
	identifier string,
	identifierStartPosition ast.Position,
	identifierEndPosition ast.Position,
) {
	origins := i.MemberOrigins[accessedType]
	origin := origins[identifier]
	i.Occurrences.Put(
		identifierStartPosition,
		identifierEndPosition,
		origin,
	)
}

func (i *PositionInfo) recordVariableDeclarationRange(
	memoryGauge common.MemoryGauge,
	declaration *ast.VariableDeclaration,
	endPosition ast.Position,
	identifier string,
	declarationType Type,
) {
	// TODO: use the start position of the next statement
	//   after this variable declaration instead

	var startPosition ast.Position
	if declaration.SecondValue != nil {
		startPosition = declaration.SecondValue.EndPosition(memoryGauge)
	} else {
		startPosition = declaration.Value.EndPosition(memoryGauge)
	}

	i.Ranges.Put(
		startPosition,
		endPosition,
		Range{
			Identifier:      identifier,
			DeclarationKind: declaration.DeclarationKind(),
			Type:            declarationType,
			DocString:       declaration.DocString,
		},
	)
}
