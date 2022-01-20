package main

import (
	"fmt"
	"net"

	"github.com/onflow/cadence/runtime/poc/uds/utils"
)

func main() {
	conn, err := net.Dial(utils.UnixNetwork, utils.Address)
	utils.HandleError(err)

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
		response := utils.ReadMessage(conn)
		println("Cadence response:", response)

		var fvmResponse string

		// TODO: switch on message header/meta_info
		switch response {
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
			fvmResponse = fmt.Sprintf("enexpected response '%s'", response)
		}

		utils.WriteMessage(conn, fvmResponse)
	}
}
