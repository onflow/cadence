package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/k0kubun/pp"

	jsoncdc "github.com/onflow/cadence/encoding/json"
)

func main() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintf(os.Stderr, "expected command\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "decode":
		var data bytes.Buffer
		reader := bufio.NewReader(os.Stdin)
		_, err := io.Copy(&data, reader)
		if err != nil {
			panic(err)
		}

		value, err := jsoncdc.Decode(nil, data.Bytes())
		if err != nil {
			panic(err)
		}

		_, _ = pp.Print(value)
	}
}
