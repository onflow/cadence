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

package compiler_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils/contracts"
	"github.com/onflow/cadence/test_utils/sema_utils"
)

func BenchmarkCompileFungibleTokenTransferTransaction(b *testing.B) {

	address := common.MustBytesToAddress([]byte{0x1})

	burnerLocation := common.NewAddressLocation(nil, address, "Burner")
	viewResolverLocation := common.NewAddressLocation(nil, address, "ViewResolver")
	fungibleTokenLocation := common.NewAddressLocation(nil, address, "FungibleToken")
	metadataViewsLocation := common.NewAddressLocation(nil, address, "MetadataViews")
	fungibleTokenMetadataViewsLocation := common.NewAddressLocation(nil, address, "FungibleTokenMetadataViews")
	nonFungibleTokenLocation := common.NewAddressLocation(nil, address, "NonFungibleToken")
	flowTokenLocation := common.NewAddressLocation(nil, address, "FlowToken")

	contractCodes := []struct {
		location common.Location
		code     string
	}{
		{
			location: burnerLocation,
			code:     contracts.RealBurnerContract,
		},
		{
			location: viewResolverLocation,
			code:     contracts.RealViewResolverContract,
		},
		{
			location: fungibleTokenLocation,
			code:     contracts.RealFungibleTokenContract,
		},
		{
			location: nonFungibleTokenLocation,
			code:     contracts.RealNonFungibleTokenContract,
		},
		{
			location: metadataViewsLocation,
			code:     contracts.RealMetadataViewsContract,
		},
		{
			location: fungibleTokenMetadataViewsLocation,
			code:     contracts.RealFungibleTokenMetadataViewsContract,
		},
		{
			location: flowTokenLocation,
			code:     contracts.RealFlowContract,
		},
	}

	locationHandler := func(identifiers []ast.Identifier, location common.Location) ([]commons.ResolvedLocation, error) {
		var name string
		switch location := location.(type) {
		case common.AddressLocation:
			name = location.Name

		case common.StringLocation:
			name = string(location)

		default:
			return nil, fmt.Errorf("cannot resolve location %s", location)
		}

		require.GreaterOrEqual(b, 1, len(identifiers))
		if len(identifiers) > 0 {
			identifier := identifiers[0].Identifier
			require.Equal(b, name, identifier)
		}

		return []commons.ResolvedLocation{
			{
				Location: common.AddressLocation{
					Address: address,
					Name:    name,
				},
			},
		}, nil
	}

	programs := CompiledPrograms{}

	compilerConfig := &compiler.Config{
		LocationHandler: locationHandler,
		BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())

			activation.Set(
				stdlib.AssertFunctionName,
				compiler.NewGlobalImport(stdlib.AssertFunctionName),
			)

			activation.Set(
				stdlib.GetAccountFunctionName,
				compiler.NewGlobalImport(stdlib.GetAccountFunctionName),
			)

			activation.Set(
				stdlib.PanicFunctionName,
				compiler.NewGlobalImport(stdlib.PanicFunctionName),
			)

			return activation
		},
	}

	for _, contract := range contractCodes {

		location := contract.location

		// Only need to compile, to make the compiled program available.
		ParseCheckAndCompileCodeWithOptions(
			b,
			contract.code,
			location,
			ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &sema_utils.ParseAndCheckOptions{
					Location: location,
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: TestBaseValueActivation,
						LocationHandler:            locationHandler,
					},
				},
				CompilerConfig: compilerConfig,
			},
			programs,
		)
	}

	tokenTransferTx := contracts.RealFlowTokenTransferTokensTransaction
	transactionLocation := common.TransactionLocation{0x1}

	checker := ParseAndCheckWithOptionsForCompiling(
		b,
		tokenTransferTx,
		transactionLocation,
		&sema_utils.ParseAndCheckOptions{
			Location: transactionLocation,
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: TestBaseValueActivation,
				LocationHandler:            locationHandler,
			},
		},
		nil,
		programs,
	)

	programs[transactionLocation] = &CompiledProgram{
		DesugaredElaboration: compiler.NewDesugaredElaboration(checker.Elaboration),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		program, _ := Compile(
			b,
			compilerConfig,
			checker,
			programs,
		)

		require.NotNil(b, program)
	}
}
