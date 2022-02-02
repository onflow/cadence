package ipc

//func StartInterfaceService(runtimeInterface runtime.Interface) error {
//	log := zlog.Logger
//
//	listener, err := bridge.NewInterfaceListener()
//	if err != nil {
//		return err
//	}
//
//	interfaceBridge := bridge.NewInterfaceBridge(runtimeInterface)
//
//	for {
//		conn, err := listener.Accept()
//		bridge.HandleError(err)
//
//		go func() {
//
//			// Gracefully handle all errors.
//			// Server shouldn't crash upon any errors.
//			defer func() {
//				if err, ok := recover().(error); ok {
//					errMsg := fmt.Sprintf("error occurred: %s", err.Error())
//					log.Error().Msg(errMsg)
//
//					// TODO: send an error response, only if the 'conn' is still alive
//					errResp := pb.NewErrorMessage(errMsg)
//					bridge.WriteMessage(conn, errResp)
//				}
//			}()
//
//			// Close the connection once everything is done.
//			defer bridge.CloseConnection(conn)
//
//			msg := bridge.ReadMessage(conn)
//
//			switch msg := msg.(type) {
//			case *pb.Request:
//				response := serveRequest(interfaceBridge, msg, log)
//				bridge.WriteMessage(conn, response)
//			case *pb.Error:
//				log.Error().Msg(msg.GetErr())
//			default:
//				log.Error().Msg("unsupported message")
//			}
//		}()
//	}
//}
