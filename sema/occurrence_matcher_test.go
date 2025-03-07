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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

type OccurrenceMatcher struct {
	OriginStartPos  *sema.Position
	OriginEndPos    *sema.Position
	StartPos        sema.Position
	EndPos          sema.Position
	DeclarationKind common.DeclarationKind
}

func (matcher *OccurrenceMatcher) Match(actual any) bool {
	occurrence, ok := actual.(sema.Occurrence)
	if !ok {
		return false
	}

	if occurrence.StartPos != matcher.StartPos {
		return false
	}

	if occurrence.EndPos != matcher.EndPos {
		return false
	}

	if occurrence.Origin.DeclarationKind != matcher.DeclarationKind {
		return false
	}

	if occurrence.Origin.StartPos != nil {
		if occurrence.Origin.StartPos.Line != matcher.OriginStartPos.Line ||
			occurrence.Origin.StartPos.Column != matcher.OriginStartPos.Column {
			return false
		}
	}

	if occurrence.Origin.EndPos != nil {
		if occurrence.Origin.EndPos.Line != matcher.OriginEndPos.Line ||
			occurrence.Origin.EndPos.Column != matcher.OriginEndPos.Column {
			return false
		}
	}

	return true
}
