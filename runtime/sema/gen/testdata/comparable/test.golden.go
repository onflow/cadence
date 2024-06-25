// Code generated from testdata/comparable/test.cdc. DO NOT EDIT.
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

package comparable

import "github.com/onflow/cadence/runtime/sema"

const TestTypeName = "Test"

var TestType = &sema.SimpleType{
	Name:          TestTypeName,
	QualifiedName: TestTypeName,
	TypeID:        TestTypeName,
	TypeTag:       TestTypeTag,
	IsResource:    false,
	Storable:      false,
	Primitive:     false,
	Equatable:     false,
	Comparable:    true,
	Exportable:    false,
	Importable:    false,
	ContainFields: false,
}
