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

package examples

const FungibleTokenContractInterface = `
  access(all) contract interface FungibleToken {

      access(all) resource interface Provider {

          access(all) fun withdraw(amount: Int): @Vault
      }

      access(all) resource interface Receiver {

          access(all) fun deposit(vault: @Vault)
      }

      access(all) resource Vault: Provider, Receiver {

          access(all) balance: Int

          init(balance: Int)
      }

      access(all) fun absorb(vault: @Vault)

      access(all) fun sprout(balance: Int): @Vault
  }
`

const ExampleFungibleTokenContract = `
  access(all) contract ExampleToken: FungibleToken {

     access(all) resource Vault: FungibleToken.Receiver, FungibleToken.Provider {

         access(all) var balance: Int

         init(balance: Int) {
             self.balance = balance
         }

         access(all) fun withdraw(amount: Int): @FungibleToken.Vault {
             self.balance = self.balance - amount
             return <-create Vault(balance: amount)
         }

         access(all) fun deposit(vault: @FungibleToken.Vault) {
            if let exampleVault <- vault as? @Vault {
                self.balance = self.balance + exampleVault.balance
                destroy exampleVault
            } else {
               destroy vault
               panic("deposited vault is not an ExampleToken.Vault")
            }
         }
     }

     access(all) fun absorb(vault: @FungibleToken.Vault) {
         destroy vault
     }

     access(all) fun sprout(balance: Int): @FungibleToken.Vault {
         return <-create Vault(balance: balance)
     }
  }
`
