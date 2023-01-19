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

package ir

type Constant interface {
	isConstant()
	Accept(Visitor) Repr
}

type Int struct {
	Value []byte
}

func (Int) isConstant() {}

func (c Int) Accept(v Visitor) Repr {
	return v.VisitInt(c)
}

type String struct {
	Value string
}

func (String) isConstant() {}

func (c String) Accept(v Visitor) Repr {
	return v.VisitString(c)
}
