package main

import (
	"fmt"
	"net"
	"syscall"

	"github.com/onflow/cadence/runtime/poc/uds/utils"
)

func main() {
	syscall.Unlink(utils.Address)
	listener, err := net.Listen(utils.UnixNetwork, utils.Address)
	utils.HandleError(err)

	// Keep listening and serving the requests from FVM.
	for {
		conn, err := listener.Accept()
		utils.HandleError(err)

		msg := utils.ReadMessage(conn)
		fmt.Println(msg)

		var response string
		switch msg {
		case "parse":
			parse(conn)
			response = "OK"
		default:
			// TODO: add a case to listen ERROR header

			response = fmt.Sprintf("unsupported operation '%s'", msg)
		}

		utils.WriteMessage(conn, response)
	}
}

func parse(conn net.Conn) {
	// do something

	// call FVM back and forth

	code := fvmGetCode(conn)
	fmt.Println(code)

	value := fvmGetValue(conn)
	fmt.Println(value)

	// do more stuff
}

// 'Interface' method invocations

func fvmGetCode(conn net.Conn) string {
	utils.WriteMessage(conn, "get_code")
	return utils.ReadMessage(conn)
}

func fvmGetValue(conn net.Conn) string {
	utils.WriteMessage(conn, "get_value")
	return utils.ReadMessage(conn)
}
