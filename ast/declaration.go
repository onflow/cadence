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

package ast

import (
	"fmt"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/common"
)

type Declaration interface {
	Element
	fmt.Stringer
	isDeclaration()
	DeclarationIdentifier() *Identifier
	DeclarationKind() common.DeclarationKind
	DeclarationAccess() Access
	DeclarationMembers() *Members
	DeclarationDocString() string
	Doc() prettier.Doc
}
