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

package wasm

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func WASM2WAT(binary []byte) string {
	f, err := ioutil.TempFile("", "wasm")
	if err != nil {
		panic(err)
	}

	defer os.Remove(f.Name())

	_, err = f.Write(binary)
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("wasm2wat", f.Name())
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			panic(fmt.Errorf("wasm2wat failed: %w:\n%s", err, ee.Stderr))
		} else {
			panic(fmt.Errorf("wasm2wat failed: %w", err))
		}
	}

	return string(out)
}
