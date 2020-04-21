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

	"github.com/onflow/flow-go-sdk/crypto"
)

// Config defines configuration for the Language Server. These options are
// determined by the client and passed to the server at initialization.
type Config struct {
	// The address where the emulator is running.
	EmulatorAddr string

	// The root account key information.
	RootAccountKey AccountPrivateKey
}

type AccountPrivateKey struct {
	PrivateKey crypto.PrivateKey
	SigAlgo crypto.SignatureAlgorithm
	HashAlgo crypto.HashAlgorithm
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

	emulatorAddr, ok := optsMap["emulatorAddress"].(string)
	if !ok {
		return Config{}, errors.New("missing emulatorAddress field")
	}

	rootPrivateKeyHex, ok := optsMap["rootPrivateKey"].(string)
	if !ok {
		return Config{}, errors.New("missing rootPrivateKey field")
	}

	rootKeySigAlgoStr, ok := optsMap["rootKeySignatureAlgorithm"].(string)
	if !ok {
		return Config{}, errors.New("missing rootKeySignatureAlgorithm field")
	}

	rootKeyHashAlgoStr, ok := optsMap["rootKeyHashAlgorithm"].(string)
	if !ok {
		return Config{}, errors.New("missing rootKeyHashAlgorithm field")
	}

	rootAccountKey := AccountPrivateKey{
		SigAlgo:    crypto.StringToSignatureAlgorithm(rootKeySigAlgoStr),
		HashAlgo:   crypto.StringToHashAlgorithm(rootKeyHashAlgoStr),
	}

	rootAccountKey.PrivateKey, err = crypto.DecodePrivateKeyHex(rootAccountKey.SigAlgo, rootPrivateKeyHex)
	if err != nil {
		return
	}

	conf.EmulatorAddr = emulatorAddr
	conf.RootAccountKey = rootAccountKey

	return
}
