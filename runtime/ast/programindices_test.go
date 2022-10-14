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

func TestProgramIndices(t *testing.T) {

	functionA := &FunctionDeclaration{
		Identifier: Identifier{Identifier: "A"},
	}
	functionB := &FunctionDeclaration{
		Identifier: Identifier{Identifier: "B"},
	}
	functionC := &FunctionDeclaration{
		Identifier: Identifier{Identifier: "C"},
	}

	compositeA := &CompositeDeclaration{
		Identifier: Identifier{Identifier: "A"},
	}
	compositeB := &CompositeDeclaration{
		Identifier: Identifier{Identifier: "B"},
	}
	compositeC := &CompositeDeclaration{
		Identifier: Identifier{Identifier: "C"},
	}

	attachmentA := &AttachmentDeclaration{
		Identifier: Identifier{Identifier: "A"},
	}
	attachmentB := &AttachmentDeclaration{
		Identifier: Identifier{Identifier: "B"},
	}
	attachmentC := &AttachmentDeclaration{
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

	transactionA := &TransactionDeclaration{
		ParameterList: &ParameterList{
			Parameters: []*Parameter{
				{
					Identifier: Identifier{Identifier: "A"},
				},
			},
		},
	}
	transactionB := &TransactionDeclaration{
		ParameterList: &ParameterList{
			Parameters: []*Parameter{
				{
					Identifier: Identifier{Identifier: "B"},
				},
			},
		},
	}
	transactionC := &TransactionDeclaration{
		ParameterList: &ParameterList{
			Parameters: []*Parameter{
				{
					Identifier: Identifier{Identifier: "C"},
				},
			},
		},
	}

	importA := &ImportDeclaration{
		Location: common.StringLocation("A"),
	}
	importB := &ImportDeclaration{
		Location: common.StringLocation("B"),
	}
	importC := &ImportDeclaration{
		Location: common.StringLocation("C"),
	}

	pragmaA := &PragmaDeclaration{
		Expression: &IdentifierExpression{
			Identifier: Identifier{Identifier: "A"},
		},
	}
	pragmaB := &PragmaDeclaration{
		Expression: &IdentifierExpression{
			Identifier: Identifier{Identifier: "B"},
		},
	}
	pragmaC := &PragmaDeclaration{
		Expression: &IdentifierExpression{
			Identifier: Identifier{Identifier: "C"},
		},
	}

	program := NewProgram(
		nil,
		[]Declaration{
			importB,
			pragmaA,
			transactionC,
			attachmentC,
			functionC,
			interfaceB,
			transactionA,
			compositeB,
			importC,
			transactionB,
			importA,
			interfaceA,
			attachmentA,
			pragmaB,
			functionA,
			compositeC,
			functionB,
			interfaceC,
			attachmentB,
			pragmaC,
			compositeA,
		},
	)

	var wg sync.WaitGroup
	const parallelExecutionCount = 10

	for i := 0; i < parallelExecutionCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			require.Equal(t,
				[]*FunctionDeclaration{
					functionC,
					functionA,
					functionB,
				},
				program.FunctionDeclarations(),
			)

			require.Equal(t,
				[]*CompositeDeclaration{
					compositeB,
					compositeC,
					compositeA,
				},
				program.CompositeDeclarations(),
			)

			require.Equal(t,
				[]*AttachmentDeclaration{
					attachmentC,
					attachmentA,
					attachmentB,
				},
				program.AttachmentDeclarations(),
			)

			require.Equal(t,
				[]*InterfaceDeclaration{
					interfaceB,
					interfaceA,
					interfaceC,
				},
				program.InterfaceDeclarations(),
			)

			require.Equal(t,
				[]*TransactionDeclaration{
					transactionC,
					transactionA,
					transactionB,
				},
				program.TransactionDeclarations(),
			)

			require.Equal(t,
				[]*ImportDeclaration{
					importB,
					importC,
					importA,
				},
				program.ImportDeclarations(),
			)

			require.Equal(t,
				[]*PragmaDeclaration{
					pragmaA,
					pragmaB,
					pragmaC,
				},
				program.PragmaDeclarations(),
			)
		}()
	}

	wg.Wait()
}
