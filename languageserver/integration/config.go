//go:build !js
// +build !js

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

package integration

import (
	"errors"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
)

// Config defines configuration for the Language Server. These options are
// determined by the client and passed to the server at initialization.
type Config struct {
	// The address where the emulator is running.
	EmulatorAddr string

	// Current emulator state
	emulatorState EmulatorState

	// Active account
	activeAccount ClientAccount

	// path to flow.json
	configPath string
}

type AccountPrivateKey struct {
	PrivateKey crypto.PrivateKey
	SigAlgo    crypto.SignatureAlgorithm
	HashAlgo   crypto.HashAlgorithm
}

// configFromInitializationOptions creates a new config instance from the
// initialization options field passed from the client at startup.
//
// Returns an error if any fields are missing or malformed.
//
func configFromInitializationOptions(opts interface{}) (conf Config, err error) {
	optsMap, ok := opts.(map[string]interface{})
	if !ok {
		return Config{}, errors.New("invalid initialization options")
	}

	emulatorState, ok := optsMap["emulatorState"].(float64)
	if !ok {
		return Config{}, errors.New("initialization options: invalid emulator state")
	}
	conf.emulatorState = EmulatorState(emulatorState)

	activeAccountName, ok := optsMap["activeAccountName"].(string)
	if !ok {
		return Config{}, errors.New("initialization options: invalid active account name")
	}
	activeAccountAddress, ok := optsMap["activeAccountAddress"].(string)
	if !ok {
		return Config{}, errors.New("initialization options: invalid active account address")
	}

	conf.activeAccount = ClientAccount{
		Name:    activeAccountName,
		Address: flow.HexToAddress(activeAccountAddress),
	}

	configPath, ok := optsMap["configPath"].(string)
	if !ok || configPath == "" {
		return Config{}, errors.New("initialization options: invalid config path")
	}

	conf.configPath = configPath

	return
}
