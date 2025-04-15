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

package bbq

type Function[E any] struct {
	Name               string
	QualifiedName      string
	Code               []E
	ParameterCount     uint16
	TypeParameterCount uint16
	LocalCount         uint16
	TypeIndex          uint16
}

func (f Function[E]) IsAnonymous() bool {
	return f.QualifiedName == ""
}

func (f Function[E]) IsNative() bool {
	return f.Code == nil
}
