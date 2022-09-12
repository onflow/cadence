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
	"regexp"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitPathExpression(expression *ast.PathExpression) Type {

	ty, err := CheckPathLiteral(
		expression.Domain.Identifier,
		expression.Identifier.Identifier,
		func() ast.Range {
			return ast.NewRangeFromPositioned(checker.memoryGauge, expression.Domain)
		},
		func() ast.Range {
			return ast.NewRangeFromPositioned(checker.memoryGauge, expression.Identifier)
		},
	)

	checker.report(err)

	return ty
}

var isValidIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString

func CheckPathLiteral(domainString, identifier string, domainRangeThunk, idRangeThunk func() ast.Range) (Type, error) {

	// Check that the domain is valid
	domain, ok := common.AllPathDomainsByIdentifier[domainString]
	if !ok {
		return PathType, &InvalidPathDomainError{
			ActualDomain: domainString,
			Range:        domainRangeThunk(),
		}
	}

	// Check that the identifier is valid
	if !isValidIdentifier(identifier) {
		return PathType, &InvalidPathIdentifierError{
			ActualIdentifier: identifier,
			Range:            idRangeThunk(),
		}
	}

	switch domain {
	case common.PathDomainStorage:
		return StoragePathType, nil
	case common.PathDomainPublic:
		return PublicPathType, nil
	case common.PathDomainPrivate:
		return PrivatePathType, nil
	default:
		return PathType, nil
	}
}
