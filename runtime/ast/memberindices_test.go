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

package ast

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
)

func TestMemberIndices(t *testing.T) {

	fieldA := &FieldDeclaration{
		Identifier: Identifier{Identifier: "A"},
	}
	fieldB := &FieldDeclaration{
		Identifier: Identifier{Identifier: "B"},
	}
	fieldC := &FieldDeclaration{
		Identifier: Identifier{Identifier: "C"},
	}

	functionA := &FunctionDeclaration{
		Identifier: Identifier{Identifier: "A"},
	}
	functionB := &FunctionDeclaration{
		Identifier: Identifier{Identifier: "B"},
	}
	functionC := &FunctionDeclaration{
		Identifier: Identifier{Identifier: "C"},
	}

	specialFunctionA := &SpecialFunctionDeclaration{
		Kind: common.DeclarationKindInitializer,
	}
	specialFunctionB := &SpecialFunctionDeclaration{
		Kind: common.DeclarationKindDestructor,
	}
	specialFunctionC := &SpecialFunctionDeclaration{}

	compositeA := &CompositeDeclaration{
		Identifier: Identifier{Identifier: "A"},
	}
	compositeB := &CompositeDeclaration{
		Identifier: Identifier{Identifier: "B"},
	}
	compositeC := &CompositeDeclaration{
		Identifier: Identifier{Identifier: "C"},
	}

	interfaceA := &InterfaceDeclaration{
		Identifier: Identifier{Identifier: "A"},
	}
	interfaceB := &InterfaceDeclaration{
		Identifier: Identifier{Identifier: "B"},
	}
	interfaceC := &InterfaceDeclaration{
		Identifier: Identifier{Identifier: "C"},
	}

	enumCaseA := &EnumCaseDeclaration{
		Identifier: Identifier{Identifier: "A"},
	}
	enumCaseB := &EnumCaseDeclaration{
		Identifier: Identifier{Identifier: "B"},
	}
	enumCaseC := &EnumCaseDeclaration{
		Identifier: Identifier{Identifier: "C"},
	}

	members := NewUnmeteredMembers(
		[]Declaration{
			specialFunctionB,
			enumCaseA,
			compositeC,
			fieldC,
			interfaceB,
			compositeA,
			functionB,
			specialFunctionC,
			compositeB,
			specialFunctionA,
			interfaceA,
			enumCaseB,
			fieldA,
			functionC,
			fieldB,
			interfaceC,
			enumCaseC,
			functionA,
		},
	)

	var wg sync.WaitGroup
	const parallelExecutionCount = 10

	for i := 0; i < parallelExecutionCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			require.Equal(t,
				[]*FieldDeclaration{
					fieldC,
					fieldA,
					fieldB,
				},
				members.Fields(),
			)

			require.Equal(t,
				[]*FunctionDeclaration{
					functionB,
					functionC,
					functionA,
				},
				members.Functions(),
			)

			require.Equal(t,
				[]*SpecialFunctionDeclaration{
					specialFunctionB,
					specialFunctionC,
					specialFunctionA,
				},
				members.SpecialFunctions(),
			)

			require.Equal(t,
				[]*InterfaceDeclaration{
					interfaceB,
					interfaceA,
					interfaceC,
				},
				members.Interfaces(),
			)

			require.Equal(t,
				[]*CompositeDeclaration{
					compositeC,
					compositeA,
					compositeB,
				},
				members.Composites(),
			)

			require.Equal(t,
				[]*EnumCaseDeclaration{
					enumCaseA,
					enumCaseB,
					enumCaseC,
				},
				members.EnumCases(),
			)
		}()
	}

	wg.Wait()
}
