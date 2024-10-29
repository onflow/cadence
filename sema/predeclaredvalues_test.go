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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckPredeclaredValues(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)

	valueDeclaration := stdlib.NewStandardLibraryStaticFunction(
		"foo",
		&sema.FunctionType{
			ReturnTypeAnnotation: sema.VoidTypeAnnotation,
		},
		"",
		nil,
	)
	baseValueActivation.DeclareValue(valueDeclaration)

	_, err := ParseAndCheckWithOptions(t,
		`
            access(all) fun test() {
                foo()
            }
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
