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

package test

import (
	_ "embed"
)

//go:embed contracts/ViewResolver.cdc
var realViewResolverContract string

//go:embed contracts/Burner.cdc
var realBurnerContract string

//go:embed contracts/FungibleToken.cdc
var realFungibleTokenContract string

//go:embed contracts/MetadataViews.cdc
var realMetadataViewsContract string

//go:embed contracts/FungibleTokenMetadataViews.cdc
var realFungibleTokenMetadataViewsContract string

//go:embed contracts/NonFungibleToken.cdc
var realNonFungibleTokenContract string

//go:embed contracts/FlowToken.cdc
var realFlowContract string

//go:embed transactions/flowToken/setup_account.cdc
var realFlowTokenSetupAccountTransaction string

//go:embed transactions/flowToken/transfer_tokens.cdc
var realFlowTokenTransferTokensTransaction string

//go:embed transactions/flowToken/mint_tokens.cdc
var realFlowTokenMintTokensTransaction string

//go:embed scripts/flowToken/get_balance.cdc
var realFlowTokenGetBalanceScript string
