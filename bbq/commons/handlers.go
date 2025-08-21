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

package commons

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

type ImportHandler func(location common.Location) *bbq.InstructionProgram

type LocationHandler = sema.LocationHandlerFunc

type ResolvedLocation = sema.ResolvedLocation

func ResolveLocation(
	locationHandler LocationHandler,
	identifiers []ast.Identifier,
	location common.Location,
) ([]ResolvedLocation, error) {

	// If no location handler is available,
	// default to resolving to a single location that declares all identifiers

	if locationHandler == nil {
		return []ResolvedLocation{
			{
				Location:    location,
				Identifiers: identifiers,
			},
		}, nil
	}

	// A location handler is available,
	// use it to resolve the location / identifiers
	return locationHandler(identifiers, location)
}
