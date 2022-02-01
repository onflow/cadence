package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	zlog "github.com/rs/zerolog/log"

	"github.com/onflow/cadence/runtime/ipc"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/ipc/protobuf"
)

var signalsToWatch = []os.Signal{
	syscall.SIGINT, // same as `os.Interrupt`
	syscall.SIGTERM,
	//syscall.SIGKILL,
}

var runtimeInterface = ipc.NewProxyInterface()

func main() {
	log := zlog.Logger

	listener, err := bridge.NewRuntimeListener()
	if err != nil {
		log.Info().Msgf("cannot start cadence runtime: %s", err.Error())
		return
	}

	runtimeBridge := bridge.NewRuntimeBridge()

	log.Info().Msg("starting cadence runtime")

	// Handle interrupts
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, signalsToWatch...)
	go func() {
		_ = <-signals
		log.Info().Msg("shutting down cadence runtime")

		err := listener.Close()
		if err != nil {
			log.Info().Msgf("failed to close listener: %s", err.Error())
		}

		os.Exit(0)
	}()

	// Keep listening and serving the requests.
	for {
		conn, err := listener.Accept()
		bridge.HandleError(err)

		go func() {
			// Gracefully handle all errors.
			// Server shouldn't crash upon any errors.
			defer func() {
				if err, ok := recover().(error); ok {
					errMsg := fmt.Sprintf("error occurred: %s", err.Error())
					log.Error().Msg(errMsg)

					// TODO: send an error response, only if the 'conn' is still alive
					errResp := pb.NewErrorMessage(errMsg)
					bridge.WriteMessage(conn, errResp)
				}
			}()

			// Close the connection once everything is done.
			defer bridge.CloseConnection(conn)

			msg := bridge.ReadMessage(conn)

			switch msg := msg.(type) {
			case *pb.Request:
				response := serveRequest(runtimeBridge, msg)
				bridge.WriteMessage(conn, response)
			case *pb.Error:
				log.Error().Msg(msg.GetErr())
			default:
				log.Error().Msg("unsupported message")
			}
		}()
	}
}

func serveRequest(
	runtimeBridge *bridge.RuntimeBridge,
	request *pb.Request,
) pb.Message {

	var response pb.Message

	switch request.Name {
	case ipc.RuntimeMethodExecuteScript:
		response = runtimeBridge.ExecuteScript(runtimeInterface, request.Params)

	case ipc.RuntimeMethodExecuteTransaction:
		response = runtimeBridge.ExecuteTransaction(runtimeInterface, request.Params)

	case ipc.RuntimeMethodInvokeContractFunction:
		response = runtimeBridge.InvokeContractFunction(request.Params)

	default:
		response = pb.NewErrorMessage(
			fmt.Sprintf("unsupported request '%s'", request.Name),
		)
	}

	return response
}
