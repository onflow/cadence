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

package address_scan

import (
	"context"
	_ "embed"
	"fmt"
	"sync"
	"time"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-emulator/emulator"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/access/grpc"
	"github.com/onflow/flow-go/fvm/evm/stdlib"
	"github.com/rs/zerolog"
	grpcOpts "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// getFlowClient initializes and returns a flow client
func getFlowClient(flowClientUrl string) *grpc.BaseClient {
	flowClient, err := grpc.NewBaseClient(
		flowClientUrl,
		grpcOpts.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	return flowClient
}

// Config defines that application's config
type Config struct {
	// BachSize is the number of addresses for which to run each script
	BatchSize int
	// 0 is treated as the latest block height.
	AtBlockHeight     uint64
	FlowAccessNodeURL string
	Chain             flow.ChainID
	ConcurrentClients int
	Pause             time.Duration
}

var DefaultConfig = Config{
	BatchSize:         1000,
	AtBlockHeight:     0,
	FlowAccessNodeURL: "access.mainnet.nodes.onflow.org:9000",
	Chain:             flow.Mainnet,
	ConcurrentClients: 10, // should be a good number to not produce too much traffic
	Pause:             500 * time.Millisecond,
}

func Scanner(
	ctx context.Context,
	log zerolog.Logger,
	conf Config,
	script string,
	handler func(bool),
	retryOnErrors bool,
) error {

	code := []byte(script)

	flowClient := getFlowClient(conf.FlowAccessNodeURL)

	currentBlock, err := getBlockHeight(ctx, conf, flowClient)
	if err != nil {
		return err
	}
	log.Info().Uint64("blockHeight", currentBlock.Height).Msg("Fetched block info")

	ap, err := InitAddressProvider(ctx, log, conf.Chain, currentBlock.ID, flowClient, conf.Pause)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	addressChan := make(chan []flow.Address)

	for i := 0; i < conf.ConcurrentClients; i++ {
		wg.Add(1)
		go func() {
			// Each worker has a separate Flow client
			client := getFlowClient(conf.FlowAccessNodeURL)
			defer func() {
				err = client.Close()
				if err != nil {
					log.Warn().
						Err(err).
						Msg("error closing client")
				}
			}()

			// Get the batches of address through addressChan,
			// run the script with that batch of addresses,
			// and pass the result to the handler

			for accountAddresses := range addressChan {
				accountsCadenceValues := convertAddresses(accountAddresses)
				arguments := []cadence.Value{cadence.NewArray(accountsCadenceValues)}
				success := runScriptOnEmulator(
					code,
					arguments,
				)

				if !success {
					continue
				}

				handler(success)
			}

			wg.Done()
		}()
	}

	ap.GenerateAddressBatches(addressChan, conf.BatchSize)

	// Close the addressChan and wait for the workers to finish
	close(addressChan)
	wg.Wait()

	return nil
}

func getBlockHeight(ctx context.Context, conf Config, flowClient *grpc.BaseClient) (*flow.BlockHeader, error) {
	if conf.AtBlockHeight != 0 {
		blk, err := flowClient.GetBlockByHeight(ctx, conf.AtBlockHeight)
		if err != nil {
			return nil, fmt.Errorf("failed to get block at the specified height: %w", err)
		}
		return &blk.BlockHeader, nil
	} else {
		block, err := flowClient.GetLatestBlockHeader(ctx, true)
		if err != nil {
			return nil, fmt.Errorf("failed to get the latest block header: %w", err)
		}
		return block, nil
	}
}

// run script with emulator
func runScriptOnEmulator(
	script []byte,
	arguments []cadence.Value,
) (success bool) {
	b, _ := emulator.New()

	addressBytesArray := cadence.NewArray(arguments).WithType(stdlib.EVMAddressBytesCadenceType)

	_, err2 := b.ExecuteScript(script, [][]byte{jsoncdc.MustEncode(addressBytesArray)})

	return err2 == nil
}

// convertAddresses generates an array of cadence.Value from an array of flow.Address
func convertAddresses(addresses []flow.Address) []cadence.Value {
	var accounts []cadence.Value
	for _, address := range addresses {
		accounts = append(accounts, cadence.Address(address))
	}
	return accounts
}
