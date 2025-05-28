package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/cmd/util/ledger/util"
	"github.com/onflow/flow-go/cmd/util/ledger/util/registers"
	"github.com/onflow/flow-go/engine/execution/computation"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/storage/snapshot"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

var ErrNotImplemented = errors.New("not implemented")

var (
	flagState           = flag.String("state", "", "Path to state snapshot (.sp file)")
	flagStateCommitment = flag.String("state-commitment", "", "State commitment (64 hex chars)")
	flagChain           = flag.String("chain", "testnet", "Flow chain ID")
	flagScript          = flag.String("script", "", "Cadence script path")
)

func main() {

	flag.Parse()

	if *flagState == "" || *flagStateCommitment == "" || *flagScript == "" {
		fmt.Println("Usage: go run main.go --state <file.sp> --state-commitment <hex> --chain <testnet|mainnet> --script <file.cdc>")
		os.Exit(1)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.TimeOnly,
	})

	chainID := flow.ChainID(*flagChain)

	log.Info().Msg("loading state ...")

	var (
		err      error
		payloads []*ledger.Payload
	)
	log.Info().Msg("reading trie")

	stateCommitment := util.ParseStateCommitment(*flagStateCommitment)
	payloads, err = util.ReadTrieForPayloads(*flagState, stateCommitment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read payloads")
	}

	log.Info().Msgf("creating registers from payloads (%d)", len(payloads))

	registersByAccount, err := registers.NewByAccountFromPayloads(payloads)
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

	// Loop over all account registers with their owner string
	registersByAccount.ForEachAccount(func(accountRegisters *registers.AccountRegisters) error {
		ownerStr := accountRegisters.Owner()
		address := flow.HexToAddress(ownerStr)
		cadenceAddr := cadence.NewAddress(address)

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

		fmt.Printf("Address: %s, Result: %s\n", address.Hex(), string(result))
		return nil
	})
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
