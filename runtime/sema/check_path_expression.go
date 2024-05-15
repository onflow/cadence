/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitPathExpression(expression *ast.PathExpression) Type {

	ty, err := CheckPathLiteral(
		checker.memoryGauge,
		expression.Domain.Identifier,
		expression.Identifier.Identifier,
		expression.Domain,
		expression.Identifier,
	)

	checker.report(err)

	return ty
}

var isValidIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString

func CheckPathLiteral(
	gauge common.MemoryGauge,
	domain string,
	identifier string,
	domainRange ast.HasPosition,
	identifierRange ast.HasPosition,
) (Type, error) {

	// Check that the domain is valid
	pathDomain := common.PathDomainFromIdentifier(domain)
	if pathDomain == common.PathDomainUnknown {
		return PathType, &InvalidPathDomainError{
			ActualDomain: domain,
			Range:        ast.NewRangeFromPositioned(gauge, domainRange),
		}
	}

	// Check that the identifier is valid
	if !isValidIdentifier(identifier) {
		return PathType, &InvalidPathIdentifierError{
			ActualIdentifier: identifier,
			Range:            ast.NewRangeFromPositioned(gauge, identifierRange),
		}
	}

	switch pathDomain {
	case common.PathDomainStorage:
		return StoragePathType, nil
	case common.PathDomainPublic:
		return PublicPathType, nil
	case common.PathDomainPrivate:
		return PrivatePathType, nil
	default:
		panic(errors.NewUnreachableError())
	}
}
