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
	"regexp"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitPathExpression(expression *ast.PathExpression) ast.Repr {

	ty := CheckPathLiteral(
		expression.Domain.Identifier,
		expression.Identifier.Identifier,
		func(errFunc func(err *ast.PathExpression) error) {
			checker.report(errFunc(expression))
		},
	)

	return ty
}

var isValidIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString

func CheckPathLiteral(domainString, identifier string, report func(func(err *ast.PathExpression) error)) Type {

	// Check that the domain is valid
	domain, ok := common.AllPathDomainsByIdentifier[domainString]
	if !ok {
		report(func(expression *ast.PathExpression) error {
			return &InvalidPathDomainError{
				ActualDomain: domainString,
				Range:        ast.NewRangeFromPositioned(expression.Domain),
			}
		})
		return PathType
	}

	// Check that the identifier is valid
	if !isValidIdentifier(identifier) {
		report(func(expression *ast.PathExpression) error {
			return &InvalidPathIdentifierError{
				ActualIdentifier: identifier,
				Range:            ast.NewRangeFromPositioned(expression.Identifier),
			}
		})
		return PathType
	}

	switch domain {
	case common.PathDomainStorage:
		return StoragePathType
	case common.PathDomainPublic:
		return PublicPathType
	case common.PathDomainPrivate:
		return PrivatePathType
	default:
		return PathType
	}
}
