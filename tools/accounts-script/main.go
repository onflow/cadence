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
	"github.com/onflow/flow-go/utils/debug"
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
)

func ProcessAndRunScriptOnTrie(chainID flow.ChainID, tries []*trie.MTrie) {
	log.Info().Msgf("Processing %d tries", len(tries))
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

		storageSnapshot := registers.StorageSnapshot{
			Registers: registersByAccount,
		}

		vm := fvm.NewVirtualMachine()

		// Read the script from file
		code, err := os.ReadFile(*flagScript)
		if err != nil {
			log.Fatal().Msgf("failed to read script file: %s", err)
		}

		logAccount := moduleUtil.LogProgress(
			log.Logger,
			moduleUtil.DefaultLogProgressConfig(
				"processing account group",
				registersByAccount.AccountCount(),
			),
		)

		// Loop over all account registers with their owner string
		registersByAccount.ForEachAccount(
			func(accountRegisters *registers.AccountRegisters) error {
				defer logAccount(1)
				owner := accountRegisters.Owner()
				address := common.BytesToAddress([]byte(owner))
				cadenceAddr := cadence.NewAddress([8]byte(address.Bytes()))

				argBytes, err := jsoncdc.Encode(cadenceAddr)
				if err != nil {
					log.Error().Err(err).Str("address", address.Hex()).Msg("failed to encode argument")
					return err
				}

				result, err := runScript(vm, ctx, storageSnapshot, code, [][]byte{argBytes})
				if err != nil {
					log.Error().Err(err).Str("address", address.Hex()).Msg("script execution failed")
					return err
				}

				log.Info().Msgf("Address: %s, Result: %s", owner, string(result))
				return nil
			},
		)
	}
}

func getTriesFromCheckpoint() []*trie.MTrie {
	memAllocBefore := debug.GetHeapAllocsBytes()
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

	memAllocAfter := debug.GetHeapAllocsBytes()
	log.Info().Msgf("the checkpoint is loaded, mem usage: %d", memAllocAfter-memAllocBefore)

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

	if *flagCheckpointDir == "" || *flagScript == "" {
		fmt.Println("Usage: go run main.go --checkpoint-dir <dir> --chain <chain> --script <file.cdc>")
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

	ProcessAndRunScriptOnTrie(chainID, tries)
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
