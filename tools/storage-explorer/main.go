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
	"github.com/onflow/flow-go/cmd/util/ledger/util"
	"github.com/onflow/flow-go/fvm/environment"
	"github.com/onflow/flow-go/fvm/storage/derived"
	"github.com/onflow/flow-go/fvm/storage/state"
	"github.com/onflow/flow-go/fvm/tracing"
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type StorageMapResponse struct {
	Keys []string `json:"keys"`
}

type Value struct {
	Type any `json:"type"`
}

type BoolValue struct {
	Type  any  `json:"type"`
	Value bool `json:"value"`
}

type NumberValue struct {
	Type  any    `json:"type"`
	Value string `json:"value"`
}

type StringValue struct {
	Type  any    `json:"type"`
	Value string `json:"value"`
}

type DictionaryValue struct {
	Type any   `json:"type"`
	Keys []any `json:"keys"`
}

type CompositeValue struct {
	Type   any      `json:"type"`
	Fields []string `json:"fields"`
}

type migrationTransactionPreparer struct {
	state.NestedTransactionPreparer
	derived.DerivedTransactionPreparer
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

	transactionState := state.NewTransactionState(payloadSnapshot, state.DefaultParameters())
	accounts := environment.NewAccounts(transactionState)

	accountsAtreeLedger := util.NewAccountsAtreeLedger(accounts)
	runtimeStorage := runtime.NewStorage(accountsAtreeLedger, util.NopMemoryGauge{})

	derivedChainData, err := derived.NewDerivedChainData(derived.DefaultDerivedDataCacheSize)
	if err != nil {
		log.Fatal().Err(err)
	}

	// The current block ID does not matter here, it is only for keeping a cross-block cache, which is not needed here.
	derivedTransactionData := derivedChainData.
		NewDerivedBlockDataForScript(flow.Identifier{}).
		NewSnapshotReadDerivedTransactionData()

	runtimeInterface := &util.MigrationRuntimeInterface{
		Accounts: accounts,
		Programs: environment.NewPrograms(
			tracing.NewTracerSpan(),
			util.NopMeter{},
			environment.NoopMetricsReporter{},
			migrationTransactionPreparer{
				NestedTransactionPreparer:  transactionState,
				DerivedTransactionPreparer: derivedTransactionData,
			},
			accounts,
		),
		ProgramErrors: map[common.Location]error{},
	}

	env := runtime.NewBaseInterpreterEnvironment(runtime.Config{
		// Attachments are enabled everywhere except for Mainnet
		AttachmentsEnabled: true,
	})

	env.Configure(
		runtimeInterface,
		runtime.NewCodesAndPrograms(),
		runtimeStorage,
		nil,
	)

	inter, err := interpreter.NewInterpreter(
		nil,
		nil,
		env.InterpreterConfig,
	)
	if err != nil {
		log.Fatal().Err(err)
	}

	r := mux.NewRouter()

	r.HandleFunc(
		"/accounts",
		NewAccountsHandler(payloadSnapshot, log),
	)

	r.HandleFunc(
		"/known_storage_maps",
		NewKnownStorageMapsHandler(log),
	)

	const accountDomainPattern = "/accounts/{address:[0-9A-Fa-f]{16}}/{domain:.+}"

	r.PathPrefix(accountDomainPattern + "/{identifier:.+}").
		HandlerFunc(NewAccountStorageMapIdentifierHandler(runtimeStorage, inter, log))

	r.HandleFunc(
		accountDomainPattern,
		NewAccountStorageMapHandler(runtimeStorage, log),
	)

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
		w.Header().Add("Content-Type", "application/json")
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
		w.Header().Add("Content-Type", "application/json")
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
			http.Error(
				w,
				fmt.Sprintf("unknown storage map domain: %s", storageMapDomain),
				http.StatusInternalServerError,
			)
			return
		}

		storageMap := storage.GetStorageMap(address, storageMapDomain, false)
		if storageMap == nil {
			http.Error(w, "storage map does not exist", http.StatusNotFound)
			return
		}

		keys := storageMapKeys(storageMap, knownStorageMap)

		response := StorageMapResponse{
			Keys: keys,
		}

		w.Header().Add("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Fatal().Err(err)
		}
	}
}

func NewAccountStorageMapIdentifierHandler(
	storage *runtime.Storage,
	inter *interpreter.Interpreter,
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
			http.Error(
				w,
				fmt.Sprintf("unknown storage map domain: %s", storageMapDomain),
				http.StatusInternalServerError,
			)
			return
		}

		storageMap := storage.GetStorageMap(address, storageMapDomain, false)
		if storageMap == nil {
			http.Error(w, "storage map does not exist", http.StatusNotFound)
			return
		}

		identifier := vars["identifier"]

		key, err := knownStorageMap.StringAsKey(identifier)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		value := storageMap.ReadValue(nil, key)
		if value == nil {
			http.Error(w, "value does not exist", http.StatusNotFound)
			return
		}

		preparedValue := prepareValue(value, inter)

		w.Header().Add("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(preparedValue)
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

func prepareType(value interpreter.Value, inter *interpreter.Interpreter) (result any) {
	staticType := value.StaticType(inter)

	defer func() {
		if recover() != nil {
			result = staticType.ID()
		}
	}()

	semaType, err := inter.ConvertStaticToSemaType(staticType)
	if err != nil {
		return staticType.ID()
	}

	cadenceType := runtime.ExportType(
		semaType,
		map[sema.TypeID]cadence.Type{},
	)

	return jsoncdc.PrepareType(
		cadenceType,
		jsoncdc.TypePreparationResults{},
	)
}

func prepareValue(value interpreter.Value, inter *interpreter.Interpreter) any {
	ty := prepareType(value, inter)

	switch value := value.(type) {
	case interpreter.BoolValue:
		return BoolValue{
			Type:  ty,
			Value: bool(value),
		}

	case interpreter.NumberValue:
		return NumberValue{
			Type:  ty,
			Value: value.String(),
		}

	case *interpreter.StringValue:
		return StringValue{
			Type:  ty,
			Value: value.Str,
		}

	case *interpreter.CharacterValue:
		return StringValue{
			Type:  ty,
			Value: value.Str,
		}

	case *interpreter.DictionaryValue:
		keys := make([]any, 0, value.Count())

		value.IterateKeys(inter, func(key interpreter.Value) (resume bool) {
			keys = append(keys, prepareValue(key, inter))

			return true
		})

		return DictionaryValue{
			Type: ty,
			Keys: keys,
		}

	case *interpreter.CompositeValue:
		fields := make([]string, 0, value.FieldCount())

		value.ForEachFieldName(func(field string) (resume bool) {
			fields = append(fields, field)

			return true
		})

		sort.Strings(fields)

		return CompositeValue{
			Type:   ty,
			Fields: fields,
		}

		// TODO:
		//   - AccountCapabilityControllerValue
		//   - AccountLinkValue
		//   - AddressValue
		//   - ArrayValue
		//   - CapabilityControllerValue
		//   - IDCapabilityValue
		//   - PathCapabilityValue
		//   - PathLinkValue
		//   - PathValue
		//   - PublishedValue
		//   - SimpleCompositeValue
		//   - SomeValue
		//   - StorageCapabilityControllerValue
		//   - TypeValue

	default:
		return Value{
			Type: ty,
		}
	}
}
