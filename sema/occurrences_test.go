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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

// TODO: implement occurrences for type references
//  (e.g. conformances, failable casting expression)

func TestCheckOccurrencesVariableDeclarations(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheckWithOptions(t, `
        let x = 1
        var y = x
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				PositionInfoEnabled: true,
			},
		},
	)

	require.NoError(t, err)

	occurrences := checker.PositionInfo.Occurrences.All()

	matchers := []*OccurrenceMatcher{
		{
			StartPos:        sema.Position{Line: 2, Column: 12},
			EndPos:          sema.Position{Line: 2, Column: 12},
			OriginStartPos:  &sema.Position{Line: 2, Column: 12},
			OriginEndPos:    &sema.Position{Line: 2, Column: 12},
			DeclarationKind: common.DeclarationKindConstant,
		},
		{
			StartPos:        sema.Position{Line: 3, Column: 12},
			EndPos:          sema.Position{Line: 3, Column: 12},
			OriginStartPos:  &sema.Position{Line: 3, Column: 12},
			OriginEndPos:    &sema.Position{Line: 3, Column: 12},
			DeclarationKind: common.DeclarationKindVariable,
		},
		{
			StartPos:        sema.Position{Line: 3, Column: 16},
			EndPos:          sema.Position{Line: 3, Column: 16},
			OriginStartPos:  &sema.Position{Line: 2, Column: 12},
			OriginEndPos:    &sema.Position{Line: 2, Column: 12},
			DeclarationKind: common.DeclarationKindConstant,
		},
	}

nextMatcher:
	for _, matcher := range matchers {
		for _, occurrence := range occurrences {
			if matcher.Match(occurrence) {
				continue nextMatcher
			}
		}

		assert.Fail(t, "failed to find occurrence", "matcher: %#+v", matcher)
	}

	for _, matcher := range matchers {
		assert.NotNil(t, checker.PositionInfo.Occurrences.Find(matcher.StartPos))
		assert.NotNil(t, checker.PositionInfo.Occurrences.Find(matcher.EndPos))
	}
}

func TestCheckOccurrencesFunction(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheckWithOptions(t,
		`
		fun f1(paramX: Int, paramY: Bool) {
		   let x = 1
		   var y: Int? = x
		   fun f2() {
		       if let y = y {
		       }
		   }
           f1(paramX: 1, paramY: true)
		}

        fun f3() {
            f1(paramX: 2, paramY: false)
        }
		`,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				PositionInfoEnabled: true,
			},
		},
	)

	require.NoError(t, err)

	occurrences := checker.PositionInfo.Occurrences.All()

	matchers := []*OccurrenceMatcher{
		{
			StartPos:        sema.Position{Line: 2, Column: 6},
			EndPos:          sema.Position{Line: 2, Column: 7},
			OriginStartPos:  &sema.Position{Line: 2, Column: 6},
			OriginEndPos:    &sema.Position{Line: 2, Column: 7},
			DeclarationKind: common.DeclarationKindFunction,
		},
		{
			StartPos:        sema.Position{Line: 2, Column: 9},
			EndPos:          sema.Position{Line: 2, Column: 14},
			OriginStartPos:  &sema.Position{Line: 2, Column: 9},
			OriginEndPos:    &sema.Position{Line: 2, Column: 14},
			DeclarationKind: common.DeclarationKindParameter,
		},
		{
			StartPos:        sema.Position{Line: 2, Column: 22},
			EndPos:          sema.Position{Line: 2, Column: 27},
			OriginStartPos:  &sema.Position{Line: 2, Column: 22},
			OriginEndPos:    &sema.Position{Line: 2, Column: 27},
			DeclarationKind: common.DeclarationKindParameter,
		},
		{
			StartPos:        sema.Position{Line: 3, Column: 9},
			EndPos:          sema.Position{Line: 3, Column: 9},
			OriginStartPos:  &sema.Position{Line: 3, Column: 9},
			OriginEndPos:    &sema.Position{Line: 3, Column: 9},
			DeclarationKind: common.DeclarationKindConstant,
		},
		{
			StartPos:        sema.Position{Line: 4, Column: 9},
			EndPos:          sema.Position{Line: 4, Column: 9},
			OriginStartPos:  &sema.Position{Line: 4, Column: 9},
			OriginEndPos:    &sema.Position{Line: 4, Column: 9},
			DeclarationKind: common.DeclarationKindVariable,
		},
		{
			StartPos:        sema.Position{Line: 4, Column: 19},
			EndPos:          sema.Position{Line: 4, Column: 19},
			OriginStartPos:  &sema.Position{Line: 3, Column: 9},
			OriginEndPos:    &sema.Position{Line: 3, Column: 9},
			DeclarationKind: common.DeclarationKindConstant,
		},
		{
			StartPos:        sema.Position{Line: 5, Column: 9},
			EndPos:          sema.Position{Line: 5, Column: 10},
			OriginStartPos:  &sema.Position{Line: 5, Column: 9},
			OriginEndPos:    &sema.Position{Line: 5, Column: 10},
			DeclarationKind: common.DeclarationKindFunction,
		},
		{
			StartPos:        sema.Position{Line: 6, Column: 16},
			EndPos:          sema.Position{Line: 6, Column: 16},
			OriginStartPos:  &sema.Position{Line: 6, Column: 16},
			OriginEndPos:    &sema.Position{Line: 6, Column: 16},
			DeclarationKind: common.DeclarationKindConstant,
		},
		{
			StartPos:        sema.Position{Line: 6, Column: 20},
			EndPos:          sema.Position{Line: 6, Column: 20},
			OriginStartPos:  &sema.Position{Line: 4, Column: 9},
			OriginEndPos:    &sema.Position{Line: 4, Column: 9},
			DeclarationKind: common.DeclarationKindVariable,
		},
		{
			StartPos:        sema.Position{Line: 9, Column: 11},
			EndPos:          sema.Position{Line: 9, Column: 12},
			OriginStartPos:  &sema.Position{Line: 2, Column: 6},
			OriginEndPos:    &sema.Position{Line: 2, Column: 7},
			DeclarationKind: common.DeclarationKindFunction,
		},
		{
			StartPos:        sema.Position{Line: 12, Column: 12},
			EndPos:          sema.Position{Line: 12, Column: 13},
			OriginStartPos:  &sema.Position{Line: 12, Column: 12},
			OriginEndPos:    &sema.Position{Line: 12, Column: 13},
			DeclarationKind: common.DeclarationKindFunction,
		},
		{
			StartPos:        sema.Position{Line: 13, Column: 12},
			EndPos:          sema.Position{Line: 13, Column: 13},
			OriginStartPos:  &sema.Position{Line: 2, Column: 6},
			OriginEndPos:    &sema.Position{Line: 2, Column: 7},
			DeclarationKind: common.DeclarationKindFunction,
		},
	}

	ms := make([]any, len(matchers))
	for i := range matchers {
		ms[i] = matchers[i]
	}

nextMatcher:
	for _, matcher := range matchers {
		for _, occurrence := range occurrences {
			if matcher.Match(occurrence) {
				continue nextMatcher
			}
		}

		assert.Fail(t, "failed to find occurrence", "matcher: %#+v", matcher)
	}

	for _, matcher := range matchers {
		assert.NotNil(t, checker.PositionInfo.Occurrences.Find(matcher.StartPos))
		assert.NotNil(t, checker.PositionInfo.Occurrences.Find(matcher.EndPos))
	}
}

func TestCheckOccurrencesStructAndInterface(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheckWithOptions(t,
		`
		struct interface I1 {}

	    struct S1: I1 {
	       let x: Int
	       init() {
	          self.x = 1
	          self.test()
	       }
	       fun test() {}
	    }

	    fun f(): S1 {
	       return S1()
	    }
		`,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				PositionInfoEnabled: true,
			},
		},
	)

	require.NoError(t, err)

	occurrences := checker.PositionInfo.Occurrences.All()

	matchers := []*OccurrenceMatcher{
		{
			StartPos:        sema.Position{Line: 2, Column: 19},
			EndPos:          sema.Position{Line: 2, Column: 20},
			OriginStartPos:  &sema.Position{Line: 2, Column: 19},
			OriginEndPos:    &sema.Position{Line: 2, Column: 20},
			DeclarationKind: common.DeclarationKindStructureInterface,
		},
		{
			StartPos:        sema.Position{Line: 4, Column: 12},
			EndPos:          sema.Position{Line: 4, Column: 13},
			OriginStartPos:  &sema.Position{Line: 4, Column: 12},
			OriginEndPos:    &sema.Position{Line: 4, Column: 13},
			DeclarationKind: common.DeclarationKindStructure,
		},
		{
			StartPos:        sema.Position{Line: 5, Column: 12},
			EndPos:          sema.Position{Line: 5, Column: 12},
			OriginStartPos:  &sema.Position{Line: 5, Column: 12},
			OriginEndPos:    &sema.Position{Line: 5, Column: 12},
			DeclarationKind: common.DeclarationKindField,
		},
		{
			StartPos:        sema.Position{Line: 7, Column: 11},
			EndPos:          sema.Position{Line: 7, Column: 14},
			OriginStartPos:  nil,
			OriginEndPos:    nil,
			DeclarationKind: common.DeclarationKindSelf,
		},
		{
			StartPos:        sema.Position{Line: 7, Column: 16},
			EndPos:          sema.Position{Line: 7, Column: 16},
			OriginStartPos:  &sema.Position{Line: 5, Column: 12},
			OriginEndPos:    &sema.Position{Line: 5, Column: 12},
			DeclarationKind: common.DeclarationKindField,
		},
		{
			StartPos:        sema.Position{Line: 8, Column: 11},
			EndPos:          sema.Position{Line: 8, Column: 14},
			OriginStartPos:  nil,
			OriginEndPos:    nil,
			DeclarationKind: common.DeclarationKindSelf,
		},
		{
			StartPos:        sema.Position{Line: 8, Column: 16},
			EndPos:          sema.Position{Line: 8, Column: 19},
			OriginStartPos:  &sema.Position{Line: 10, Column: 12},
			OriginEndPos:    &sema.Position{Line: 10, Column: 15},
			DeclarationKind: common.DeclarationKindFunction,
		},
		// TODO: why the duplicate?
		{
			StartPos:        sema.Position{Line: 10, Column: 12},
			EndPos:          sema.Position{Line: 10, Column: 15},
			OriginStartPos:  &sema.Position{Line: 10, Column: 12},
			OriginEndPos:    &sema.Position{Line: 10, Column: 15},
			DeclarationKind: common.DeclarationKindFunction,
		},
		{
			StartPos:        sema.Position{Line: 13, Column: 9},
			EndPos:          sema.Position{Line: 13, Column: 9},
			OriginStartPos:  &sema.Position{Line: 13, Column: 9},
			OriginEndPos:    &sema.Position{Line: 13, Column: 9},
			DeclarationKind: common.DeclarationKindFunction,
		},
		{
			StartPos:        sema.Position{Line: 14, Column: 15},
			EndPos:          sema.Position{Line: 14, Column: 16},
			OriginStartPos:  &sema.Position{Line: 4, Column: 12},
			OriginEndPos:    &sema.Position{Line: 4, Column: 13},
			DeclarationKind: common.DeclarationKindStructure,
		},
	}

	ms := make([]any, len(matchers))
	for i := range matchers {
		ms[i] = matchers[i]
	}

nextMatcher:
	for _, matcher := range matchers {
		for _, occurrence := range occurrences {
			if matcher.Match(occurrence) {
				continue nextMatcher
			}
		}

		assert.Fail(t, "failed to find occurrence", "matcher: %#+v", matcher)
	}

	for _, matcher := range matchers {
		assert.NotNil(t, checker.PositionInfo.Occurrences.Find(matcher.StartPos))
		assert.NotNil(t, checker.PositionInfo.Occurrences.Find(matcher.EndPos))
	}
}
