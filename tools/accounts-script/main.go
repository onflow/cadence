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

package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/cmd/util/ledger/util/registers"
	"github.com/onflow/flow-go/engine/execution/computation"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/storage/snapshot"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/metrics"
	moduleUtil "github.com/onflow/flow-go/module/util"
)

// references flow-go/cmd/util/cmd/checkpoint-collect-stats

type PayloadInfo struct {
	Address string `json:"address"`
	Key     string `json:"key"`
	Type    string `json:"type"`
	Size    uint64 `json:"size"`
}

var (
	flagCheckpointDir = flag.String("checkpoint-dir", "", "Path to directory containing checkpoint files")
	flagChain         = flag.String("chain", "flow-mainnet", "Flow chain ID")
	flagScript        = flag.String("script", "", "Cadence script path")
	flagBatchSize     = flag.Int("batch", 1000, "Batch size for addresses passed to script")
)

func ProcessAndRunScriptOnTrie(chainID flow.ChainID, tries []*trie.MTrie) error {
	log.Info().Msgf("Processing %d tries", len(tries))
	options := computation.DefaultFVMOptions(chainID, false, false)
	options = append(
		options,
		fvm.WithContractDeploymentRestricted(false),
		fvm.WithContractRemovalRestricted(false),
		fvm.WithAuthorizationChecksEnabled(false),
		fvm.WithSequenceNumberCheckAndIncrementEnabled(false),
		fvm.WithTransactionFeesEnabled(false),
	)
	ctx := fvm.NewContext(options...)
	vm := fvm.NewVirtualMachine()

	// Read the script from file
	code, err := os.ReadFile(*flagScript)
	if err != nil {
		log.Fatal().Msgf("failed to read script file: %s", err)
	}

	for _, trie := range tries {

		// create registers to view accounts and for storage snapshot
		registersByAccount, err := registers.NewByAccountFromPayloads(trie.AllPayloads())
		if err != nil {
			log.Fatal().Err(err)
		}
		log.Info().Msgf(
			"created %d registers from payloads (%d accounts)",
			registersByAccount.Count(),
			registersByAccount.AccountCount(),
		)

		storageSnapshot := registers.StorageSnapshot{
			Registers: registersByAccount,
		}

		logAccount := moduleUtil.LogProgress(
			log.Logger,
			moduleUtil.DefaultLogProgressConfig(
				"processing accounts",
				registersByAccount.AccountCount(),
			),
		)

		addresses := make([]cadence.Value, 0)

		// Loop over all account registers with their owner string
		err = registersByAccount.ForEachAccount(
			func(accountRegisters *registers.AccountRegisters) error {
				defer logAccount(1)
				owner := accountRegisters.Owner()
				address := common.BytesToAddress([]byte(owner))
				cadenceAddr := cadence.NewAddress([8]byte(address.Bytes()))

				addresses = append(addresses, cadenceAddr)

				if len(addresses) >= *flagBatchSize {
					argBytes, err := jsoncdc.Encode(cadence.NewArray(addresses))
					if err != nil {
						log.Error().Err(err).Str("address", address.Hex()).Msg("failed to encode argument")
						return err
					}

					_, err = runScript(vm, ctx, storageSnapshot, code, [][]byte{argBytes})
					if err != nil {
						log.Error().Err(err).Str("address batch failed with last address", address.Hex()).Msg("cadence error")
						return err
					}

					addresses = make([]cadence.Value, 0)
				}

				// log.Info().Msgf("Address: %s, Result: %s", address.Hex(), string(result))
				return nil
			},
		)

		if err != nil {
			return err
		}
	}
	return nil
}

func getTriesFromCheckpoint() []*trie.MTrie {
	log.Info().Msgf("loading checkpoint(s) from %v", *flagCheckpointDir)

	diskWal, err := wal.NewDiskWAL(zerolog.Nop(), nil, &metrics.NoopCollector{}, *flagCheckpointDir, complete.DefaultCacheSize, pathfinder.PathByteSize, wal.SegmentSize)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create WAL")
	}
	led, err := complete.NewLedger(diskWal, complete.DefaultCacheSize, &metrics.NoopCollector{}, log.Logger, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create ledger from write-a-head logs and checkpoints")
	}
	compactor, err := complete.NewCompactor(led, diskWal, zerolog.Nop(), complete.DefaultCacheSize, math.MaxInt, 1, atomic.NewBool(false), &metrics.NoopCollector{})
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create compactor")
	}
	<-compactor.Ready()
	defer func() {
		<-led.Done()
		<-compactor.Done()
	}()

	log.Info().Msg("the checkpoint is loaded")

	var tries []*trie.MTrie

	ts, err := led.Tries()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get tries")
	}

	tries = append(tries, ts...)

	return tries
}

func main() {

	flag.Parse()

	if *flagCheckpointDir == "" || *flagScript == "" || *flagChain == "" {
		fmt.Println("Usage: go run main.go --checkpoint-dir <dir> --chain <chain> --script <file.cdc> --batch <1000>")
		os.Exit(1)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.TimeOnly,
	})

	chainID := flow.ChainID(*flagChain)
	// Validate chain ID
	_ = chainID.Chain()

	// Load execution state and retrieve payloads
	tries := getTriesFromCheckpoint()

	err := ProcessAndRunScriptOnTrie(chainID, tries)

	if err != nil {
		log.Fatal().Err(err).Msg("storage iteration failed")
	}
	log.Info().Msgf("Success")
}

func runScript(
	vm *fvm.VirtualMachine,
	ctx fvm.Context,
	storageSnapshot snapshot.StorageSnapshot,
	code []byte,
	arguments [][]byte,
) (
	encodedResult []byte,
	err error,
) {
	_, res, err := vm.Run(
		ctx,
		fvm.Script(code).WithArguments(arguments...),
		storageSnapshot,
	)
	if err != nil {
		return nil, err
	}

	if res.Err != nil {
		return nil, res.Err
	}

	encoded, err := jsoncdc.Encode(res.Value)
	if err != nil {
		return nil, err
	}

	return encoded, nil
}
