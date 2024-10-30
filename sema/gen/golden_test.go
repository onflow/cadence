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

package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	_ "github.com/onflow/cadence/sema/gen/testdata/comparable"
	_ "github.com/onflow/cadence/sema/gen/testdata/composite_type_pragma"
	"github.com/onflow/cadence/sema/gen/testdata/constructor"
	"github.com/onflow/cadence/sema/gen/testdata/contract"
	_ "github.com/onflow/cadence/sema/gen/testdata/contract"
	_ "github.com/onflow/cadence/sema/gen/testdata/docstrings"
	_ "github.com/onflow/cadence/sema/gen/testdata/entitlement"
	_ "github.com/onflow/cadence/sema/gen/testdata/equatable"
	_ "github.com/onflow/cadence/sema/gen/testdata/exportable"
	_ "github.com/onflow/cadence/sema/gen/testdata/fields"
	_ "github.com/onflow/cadence/sema/gen/testdata/functions"
	_ "github.com/onflow/cadence/sema/gen/testdata/importable"
	_ "github.com/onflow/cadence/sema/gen/testdata/member_accessible"
	_ "github.com/onflow/cadence/sema/gen/testdata/nested"
	_ "github.com/onflow/cadence/sema/gen/testdata/simple_resource"
	_ "github.com/onflow/cadence/sema/gen/testdata/simple_struct"
	_ "github.com/onflow/cadence/sema/gen/testdata/storable"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestConstructor(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.StandardLibraryValue{
		Name: constructor.FooType.Identifier,
		Type: constructor.FooTypeConstructorType,
		Kind: common.DeclarationKindFunction,
	})

	_, err := ParseAndCheckWithOptions(t,
		`
          let x = Foo(bar: 1)
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)
	require.NoError(t, err)
}

func TestContract(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.StandardLibraryValue{
		Name: contract.TestType.Identifier,
		Type: contract.TestType,
		Kind: common.DeclarationKindContract,
	})

	_, err := ParseAndCheckWithOptions(t,
		`
          let x = Test.Foo(bar: 1)
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)
	require.NoError(t, err)
}
