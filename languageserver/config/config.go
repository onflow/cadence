/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package config

import (
	"errors"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/keys"
)

// Config defines configuration for the Language Server. These options are
// determined by the client and passed to the server at initialization.
type Config struct {
	// The key of the root account.
	RootAccountKey flow.AccountPrivateKey `json:"rootAccountKey"`
	// The address where the emulator is running.
	EmulatorAddr string `json:"emulatorAddress"`
}

// FromInitializationOptions creates a new config instance from the
// initialization options field passed from the client at startup.
//
// Returns an error if any fields are missing or malformed.
func FromInitializationOptions(opts interface{}) (conf Config, err error) {
	optsMap, ok := opts.(map[string]interface{})
	if !ok {
		return Config{}, errors.New("")
	}

	rootAccountKeyHex, ok := optsMap["rootAccountKey"].(string)
	if !ok {
		return Config{}, errors.New("missing rootAccountKey field")
	}
	emulatorAddr, ok := optsMap["emulatorAddress"].(string)
	if !ok {
		return Config{}, errors.New("missing emulatorAddress field")
	}

	conf.RootAccountKey, err = keys.DecodePrivateKeyHex(rootAccountKeyHex)
	if err != nil {
		return
	}
	conf.EmulatorAddr = emulatorAddr

	return
}
