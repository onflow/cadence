package main

import (
	"fmt"
	"net"
	"syscall"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ipc"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func main() {
	syscall.Unlink(ipc.SocketAddress)
	listener, err := net.Listen(ipc.UnixNetwork, ipc.SocketAddress)
	ipc.HandleError(err)

	runtimeBridge := bridge.NewRuntimeBridge()

	// Keep listening and serving the requests from FVM.
	for {
		conn, err := listener.Accept()
		ipc.HandleError(err)

		context := runtime.Context{
			Interface: ipc.NewProxyInterface(conn),
			Location:  utils.TestLocation,
		}
		context.InitializeCodesAndPrograms()

		msg := ipc.ReadMessage(conn)

		var response bridge.Message

		// TODO: change to switch on message type + meta-info
		switch msg.String() {
		case ipc.InitInterpreterRuntimeMethod:

		case ipc.RuntimeMethodExecuteScript:
			script := runtime.Script{
				Source: []byte("pub fun main():String { return \"Hello, world!\" }"),
			}
			response = runtimeBridge.ExecuteScript(script, context)

		case ipc.RuntimeMethodExecuteTransaction:
			script := runtime.Script{
				Source: []byte("pub fun main() {}"),
			}
			response = runtimeBridge.ExecuteScript(script, context)

		case ipc.RuntimeMethodInvokeContractFunction:
			response = runtimeBridge.InvokeContractFunction()

		default:
			response = &bridge.Error{
				Content: fmt.Sprintf("unsupported operation '%s'", msg),
			}
		}

		ipc.WriteMessage(conn, response)
	}
}
