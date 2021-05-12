/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

// A utility program that parses a state dump in JSON Lines format and decodes all values

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/schollz/progressbar/v3"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

var roundtripFlag = flag.Bool("roundtrip", false, "encode and decode the decoded value and ensure equality")
var gzipFlag = flag.Bool("gzip", false, "set true if input file is gzipped")

type keyPart struct {
	Value string
}

type key struct {
	KeyParts []keyPart
}

type entry struct {
	Value string
	Key   key
}

func worker(jobs <-chan entry, wg *sync.WaitGroup, decoded *uint64) {
	defer wg.Done()

	var err error
	var data []byte

	for e := range jobs {

		data, err = hex.DecodeString(e.Value)
		if err != nil {
			log.Fatal(err)
		}

		var version uint16
		data, version = interpreter.StripMagic(data)
		if version == 0 {
			continue
		}

		rawOwner, err := hex.DecodeString(e.Key.KeyParts[1].Value)
		if err != nil {
			log.Fatal(err)
		}

		owner := common.BytesToAddress(rawOwner)

		decodeFunction := interpreter.DecodeValue
		if version <= 3 {
			decodeFunction = interpreter.DecodeValueV3
		}

		var value interpreter.Value
		value, err = decodeFunction(data, &owner, nil, version, nil)
		if err != nil {
			log.Fatalf("failed to decode value: %s\n%s\n", err, e.Value)
		}

		atomic.AddUint64(decoded, 1)

		if *roundtripFlag {
			reEncodeDecode(value, owner)
		}
	}
}

func reEncodeDecode(value interpreter.Value, owner common.Address) {
	data, deferrals, err := interpreter.EncodeValue(value, nil, true, nil)
	if err != nil {
		log.Fatalf("failed to encode value: %s\n%s\n", err, value)
	}

	if len(deferrals.Values) > 0 {
		log.Fatalf("re-encoding produced deferred values:\n%s\n", value)
	}

	if len(deferrals.Moves) > 0 {
		log.Fatalf("re-encoding produced deferred moves:\n%s\n", value)
	}

	newValue, err := interpreter.DecodeValue(data, &owner, nil, interpreter.CurrentEncodingVersion, nil)
	if err != nil {
		log.Fatalf("failed to decode re-encoded value: %s\n%s\n", err, value)
	}

	equatableValue, ok := value.(interpreter.EquatableValue)
	if !ok {
		log.Fatalf("cannot compare unequatable %[1]T\n%[1]s\n", value)
	}

	if !equatableValue.Equal(newValue, nil, false) {
		log.Fatalf("values are unequal:\n%s\n%s\n", value, newValue)
	}
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		panic("missing path argument")
	}

	file, err := os.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	jobs := make(chan entry)

	var decoded uint64

	var wg sync.WaitGroup

	workerCount := runtime.NumCPU()

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(jobs, &wg, &decoded)
	}

	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	fileSize := stat.Size()

	bar := progressbar.DefaultBytes(fileSize, "(processed JSON bytes)")

	progressReader := progressbar.NewReader(file, bar)

	var inputReader io.Reader = &progressReader
	if *gzipFlag {
		gzipReader, err := gzip.NewReader(inputReader)
		if err != nil {
			log.Fatal(err)
		}
		defer gzipReader.Close()
		inputReader = gzipReader
	}

	reader := bufio.NewReader(inputReader)

	decoder := json.NewDecoder(reader)
	for {
		var e entry

		err = decoder.Decode(&e)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		jobs <- e
	}

	close(jobs)

	wg.Wait()

	println()

	log.Printf("successfully decoded %d values\n", decoded)
}
