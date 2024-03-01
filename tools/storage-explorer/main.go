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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/onflow/flow-go/cmd/util/ledger/util"
	"github.com/rs/zerolog"

	"github.com/onflow/cadence/runtime/common"
)

func main() {

	portFlag := flag.Int("port", 3000, "port")
	payloadsFlag := flag.String("payloads", "", "payloads file")
	flag.Parse()

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.DateTime,
	}
	log := zerolog.New(consoleWriter).With().Timestamp().Logger()

	payloadsPath := *payloadsFlag
	if payloadsPath == "" {
		panic("missing payloads")
	}

	_, payloads, err := util.ReadPayloadFile(log, payloadsPath)
	if err != nil {
		panic(err)
	}

	log.Info().Msgf("read %d payloads", len(payloads))

	log.Info().Msg("building payload snapshot ...")

	payloadSnapshot, err := util.NewPayloadSnapshot(payloads)
	if err != nil {
		panic(err)
	}

	log.Info().Msg("formatting addresses ...")

	addressesJSON, err := addressesJSON(payloadSnapshot)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/accounts", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write(addressesJSON)
		if err != nil {
			panic(err)
		}
	})

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *portFlag))
	if err != nil {
		panic(err)
	}
	log.Info().Msgf("Listening on http://%s/", ln.Addr().String())
	var srv http.Server
	_ = srv.Serve(ln)
}

func addressesJSON(payloadSnapshot *util.PayloadSnapshot) ([]byte, error) {
	addressSet := map[string]struct{}{}
	for registerID := range payloadSnapshot.Payloads {
		owner := registerID.Owner
		if len(owner) > 0 {
			address := common.Address([]byte(owner)).HexWithPrefix()
			addressSet[address] = struct{}{}
		}
	}

	addresses := make([]string, 0, len(addressSet))
	for address := range addressSet {
		addresses = append(addresses, address)
	}

	sort.Strings(addresses)

	encoded, err := json.Marshal(addresses)
	if err != nil {
		return nil, err
	}

	return encoded, nil
}
