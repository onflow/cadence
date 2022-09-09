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

package parser

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/onflow/cadence/runtime/common"
)

func BenchmarkParseDeploy(b *testing.B) {

	b.Run("byte array", func(b *testing.B) {

		var builder strings.Builder
		for i := 0; i < 15000; i++ {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(strconv.Itoa(rand.Intn(math.MaxUint8)))
		}

		transaction := []byte(fmt.Sprintf(`
              transaction {
                execute {
                  AuthAccount(publicKeys: [], code: [%s])
                }
              }
            `,
			builder.String(),
		))

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := ParseProgram(transaction, nil)
			if err != nil {
				b.FailNow()
			}
		}
	})

	b.Run("decode hex", func(b *testing.B) {

		var builder strings.Builder
		for i := 0; i < 15000; i++ {
			builder.WriteString(fmt.Sprintf("%02x", i))
		}

		transaction := []byte(fmt.Sprintf(`
              transaction {
                execute {
                  AuthAccount(publicKeys: [], code: "%s".decodeHex())
                }
              }
            `,
			builder.String(),
		))

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := ParseProgram(transaction, nil)
			if err != nil {
				b.FailNow()
			}
		}
	})
}

const fungibleTokenContract = `
pub contract FungibleToken {

    pub resource interface Provider {
        pub fun withdraw(amount: Int): @Vault {
            pre {
                amount > 0:
                    "Withdrawal amount must be positive"
            }
            post {
                result.balance == amount:
                    "Incorrect amount returned"
            }
        }
    }

    pub resource interface Receiver {
        pub balance: Int

        init(balance: Int) {
            pre {
                balance >= 0:
                    "Initial balance must be non-negative"
            }
            post {
                self.balance == balance:
                    "Balance must be initialized to the initial balance"
            }
        }

        pub fun deposit(from: @Receiver) {
            pre {
                from.balance > 0:
                    "Deposit balance needs to be positive!"
            }
            post {
                self.balance == before(self.balance) + before(from.balance):
                    "Incorrect amount removed"
            }
        }
    }

    pub resource Vault: Provider, Receiver {

        pub var balance: Int

        init(balance: Int) {
            self.balance = balance
        }

        pub fun withdraw(amount: Int): @Vault {
            self.balance = self.balance - amount
            return <-create Vault(balance: amount)
        }

        // transfer combines withdraw and deposit into one function call
        pub fun transfer(to: &Receiver, amount: Int) {
            pre {
                amount <= self.balance:
                    "Insufficient funds"
            }
            post {
                self.balance == before(self.balance) - amount:
                    "Incorrect amount removed"
            }
            to.deposit(from: <-self.withdraw(amount: amount))
        }

        pub fun deposit(from: @Receiver) {
            self.balance = self.balance + from.balance
            destroy from
        }

        pub fun createEmptyVault(): @Vault {
            return <-create Vault(balance: 0)
        }
    }

    pub fun createEmptyVault(): @Vault {
        return <-create Vault(balance: 0)
    }

    pub resource VaultMinter {
        pub fun mintTokens(amount: Int, recipient: &Receiver) {
            recipient.deposit(from: <-create Vault(balance: amount))
        }
    }

    init() {
        let oldVault <- self.account.storage[Vault] <- create Vault(balance: 30)
        destroy oldVault

        let oldMinter <- self.account.storage[VaultMinter] <- create VaultMinter()
        destroy oldMinter
    }
}
`

type testMemoryGauge struct {
	meter map[common.MemoryKind]uint64
}

func (g *testMemoryGauge) MeterMemory(usage common.MemoryUsage) error {
	g.meter[usage.Kind] += usage.Amount
	return nil
}

func BenchmarkParseFungibleToken(b *testing.B) {

	code := []byte(fungibleTokenContract)

	b.Run("Without memory metering", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := ParseProgram(code, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("With memory metering", func(b *testing.B) {
		meter := &testMemoryGauge{
			meter: make(map[common.MemoryKind]uint64),
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := ParseProgram(code, meter)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
