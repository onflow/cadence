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

package subtype_gen

type Expression interface {
	isExpression()
}

type TypeExpression struct {
	Type Type
}

var _ Expression = TypeExpression{}

func (t TypeExpression) isExpression() {}

type IdentifierExpression struct {
	Name string
}

var _ Expression = IdentifierExpression{}

func (t IdentifierExpression) isExpression() {}

type MemberExpression struct {
	Parent     Expression
	MemberName string
}

var _ Expression = MemberExpression{}

func (t MemberExpression) isExpression() {}

type OneOfExpression struct {
	Expressions []Expression
}

var _ Expression = OneOfExpression{}

func (t OneOfExpression) isExpression() {}
