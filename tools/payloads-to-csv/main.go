package main

import (
	"encoding/csv"
	"encoding/hex"
	"flag"
	"os"
	"time"

	"github.com/onflow/flow-go/cmd/util/ledger/util"
	"github.com/onflow/flow-go/ledger/common/convert"
	"github.com/rs/zerolog"
)

var header = []string{"owner", "key", "value"}

func main() {

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

	log.Info().Msgf("read %d payloads. writing as CSV ...", len(payloads))

	writer := csv.NewWriter(os.Stdout)

	if err := writer.Write(header); err != nil {
		log.Fatal().Err(err).Msg("failed to write CSV header")
		return
	}

	for _, payload := range payloads {

		registerID, value, err := convert.PayloadToRegister(payload)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to write decode payload into register")
		}

		err = writer.Write([]string{
			hex.EncodeToString([]byte(registerID.Owner)),
			hex.EncodeToString([]byte(registerID.Key)),
			hex.EncodeToString(value),
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to write contract to CSV")
			return
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		log.Fatal().Err(err).Msg("failed to write CSV")
	}

	log.Info().Msgf("wrote %d payloads as CSV", len(payloads))
}
