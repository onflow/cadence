package main

import (
	"fmt"
	"net"
	"time"

	"github.com/onflow/cadence/runtime/poc/uds/utils"
)

func main() {
	conn, err := net.Dial(utils.UnixNetwork, utils.Address)
	utils.HandleError(err)

	start := time.Now()
	defer func() {
		fmt.Println("Time consumed:", time.Since(start))
	}()

	utils.WriteMessage(conn, "parse")

	listen(conn)
}

func listen(conn net.Conn) {
	// Keep listening until the final response is received.
	//
	// Rationale:
	// Once the initial request is sent to cadence, it may respond back
	// with requests (i.e: 'Interface' method calls). Hence, keep listening
	// to those requests and respond back. Terminate once the final ack
	// is received.

	for {
		message := utils.ReadMessage(conn)
		fmt.Println("Cadence message:", message)

		var fvmResponse string

		// TODO: switch on message header/meta_info
		switch message {
		// All 'Interface' methods goes here
		case "get_code":
			fvmResponse = "pub fun foo() {}"
		case "get_value":
			fvmResponse = "some value"
		case "OK":
			// This is the final ack. The round-trip is over.
			// Don't need to respond back.
			return
		default:
			// Properly handle: add a case to listen ERROR header
			fmt.Println(fmt.Sprintf("error occured '%s'", message))
			return
		}

		utils.WriteMessage(conn, fvmResponse)
	}
}
