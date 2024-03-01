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

	"github.com/gorilla/mux"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/flow-go/cmd/util/ledger/util"
	"github.com/rs/zerolog"
)

type StorageMapResponse struct {
	Exists bool     `json:"exists"`
	Keys   []string `json:"keys"`
}

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
		log.Fatal().Msg("missing payloads")
	}

	_, payloads, err := util.ReadPayloadFile(log, payloadsPath)
	if err != nil {
		log.Fatal().Err(err)
	}

	log.Info().Msgf("read %d payloads", len(payloads))

	log.Info().Msg("building payload snapshot ...")

	payloadSnapshot, err := util.NewPayloadSnapshot(payloads)
	if err != nil {
		log.Fatal().Err(err)
	}

	log.Info().Msg("creating storage ...")

	ledger := PayloadSnapshotLedger{
		PayloadSnapshot: payloadSnapshot,
	}

	storage := runtime.NewStorage(ledger, nil)

	r := mux.NewRouter()

	r.HandleFunc(
		"/accounts",
		NewAccountsHandler(payloadSnapshot, log),
	)

	r.HandleFunc(
		"/accounts/{address:[0-9A-Fa-f]{16}}/{domain:.+}",
		NewAccountStorageMapHandler(storage, log),
	)

	r.HandleFunc("/known_storage_maps", NewKnownStorageMapsHandler(log))

	http.Handle("/", r)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *portFlag))
	if err != nil {
		log.Fatal().Err(err)
	}
	log.Info().Msgf("Listening on http://%s/", ln.Addr().String())
	var srv http.Server
	_ = srv.Serve(ln)
}

func NewKnownStorageMapsHandler(log zerolog.Logger) func(w http.ResponseWriter, r *http.Request) {
	knownStorageMapsJSON := knownStorageMapsJSON()

	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(knownStorageMapsJSON)
		if err != nil {
			log.Fatal().Err(err)
		}
	}
}

func NewAccountsHandler(
	payloadSnapshot *util.PayloadSnapshot,
	log zerolog.Logger,
) func(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("formatting addresses ...")

	addressesJSON, err := addressesJSON(payloadSnapshot)
	if err != nil {
		log.Fatal().Err(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(addressesJSON)
		if err != nil {
			log.Fatal().Err(err)
		}
	}
}

func NewAccountStorageMapHandler(
	storage *runtime.Storage,
	log zerolog.Logger,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		address, err := common.HexToAddress(vars["address"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		storageMapDomain := vars["domain"]
		knownStorageMap, ok := knownStorageMaps[storageMapDomain]
		if !ok {
			http.Error(w, "unknown storage map domain", http.StatusInternalServerError)
			return
		}

		storageMap := storage.GetStorageMap(address, storageMapDomain, false)

		var keys []string
		exists := storageMap != nil
		if exists {
			keys = storageMapKeys(storageMap, knownStorageMap)
		}

		response := StorageMapResponse{
			Exists: exists,
			Keys:   keys,
		}

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Fatal().Err(err)
		}
	}
}

func storageMapKeys(storageMap *interpreter.StorageMap, knownStorageMap KnownStorageMap) []string {
	keys := make([]string, 0, storageMap.Count())
	iterator := storageMap.Iterator(nil)
	for {
		key := iterator.NextKey()
		if key == nil {
			break
		}

		keys = append(
			keys,
			knownStorageMap.KeyAsString(key),
		)
	}

	sort.Strings(keys)

	return keys
}
