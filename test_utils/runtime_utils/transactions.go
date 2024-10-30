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

package runtime_utils

import (
	"encoding/hex"
	"fmt"
)

func DeploymentTransaction(name string, contract []byte) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: auth(Contracts) &Account) {
                  signer.contracts.add(name: "%s", code: "%s".decodeHex())
              }
          }
        `,
		name,
		hex.EncodeToString(contract),
	))
}

func RemovalTransaction(name string) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: auth(Contracts) &Account) {
                  signer.contracts.remove(name: "%s")
              }
          }
        `,
		name,
	))
}

func UpdateTransaction(name string, contract []byte) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: auth(Contracts) &Account) {
                  signer.contracts.update(name: "%s", code: "%s".decodeHex())
              }
          }
        `,
		name,
		hex.EncodeToString(contract),
	))
}
