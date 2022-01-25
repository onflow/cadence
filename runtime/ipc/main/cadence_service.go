package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/onflow/cadence/runtime/ipc"
	"github.com/onflow/cadence/runtime/ipc/bridge"
)

var signalsToWatch = []os.Signal{
	syscall.SIGINT, // same as `os.Interrupt`
	syscall.SIGTERM,
	//syscall.SIGKILL,
}

var runtimeInterface = ipc.NewProxyInterface()

func main() {
	listener := bridge.NewRuntimeListener()
	runtimeBridge := bridge.NewRuntimeBridge()

	// Handle interrupts
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, signalsToWatch...)
	go func() {
		_ = <-signals
		fmt.Printf("Shutting down")
		listener.Close()
		os.Exit(0)
	}()

	// Keep listening and serving the requests.
	for {
		conn, err := listener.Accept()
		bridge.HandleError(err)

		go func() {
			msg := bridge.ReadMessage(conn)

			switch msg := msg.(type) {
			case *bridge.Request:
				response := serveRequest(runtimeBridge, msg)
				bridge.WriteMessage(conn, response)
			case *bridge.Error:
				panic(fmt.Errorf(msg.GetErr()))
			default:
				panic(fmt.Errorf("unsupported message"))
			}
		}()
	}
}

func serveRequest(
	runtimeBridge *bridge.RuntimeBridge,
	request *bridge.Request,
) bridge.Message {

	var response bridge.Message

	switch request.Name {
	case ipc.RuntimeMethodExecuteScript:
		response = runtimeBridge.ExecuteScript(runtimeInterface, request.Params)

	case ipc.RuntimeMethodExecuteTransaction:
		response = runtimeBridge.ExecuteTransaction(runtimeInterface, request.Params)

	case ipc.RuntimeMethodInvokeContractFunction:
		response = runtimeBridge.InvokeContractFunction(request.Params)

	default:
		response = bridge.NewErrorMessage(
			fmt.Sprintf("unsupported request '%s'", request.Name),
		)
	}

	return response
}
