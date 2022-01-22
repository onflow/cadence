package ipc

import (
	"encoding/binary"
	"net"
	"syscall"

	"github.com/golang/protobuf/proto"
	"github.com/onflow/cadence/runtime/ipc/bridge"
)

const (
	UnixNetwork          = "unix"
	RuntimeSocketAddress = "/tmp/cadence.socket"

	// TODO: rename FVM to something generic
	InterfaceSocketAddress = "/tmp/fvm.socket"
)

func NewRuntimeListener() net.Listener {
	syscall.Unlink(RuntimeSocketAddress)
	listener, err := net.Listen(UnixNetwork, RuntimeSocketAddress)
	HandleError(err)
	return listener
}

func NewRuntimeConnection() net.Conn {
	conn, err := net.Dial(UnixNetwork, RuntimeSocketAddress)
	HandleError(err)
	return conn
}

func NewInterfaceListener() net.Listener {
	syscall.Unlink(InterfaceSocketAddress)
	listener, err := net.Listen(UnixNetwork, InterfaceSocketAddress)
	HandleError(err)
	return listener
}

func NewInterfaceConnection() net.Conn {
	conn, err := net.Dial(UnixNetwork, InterfaceSocketAddress)
	HandleError(err)
	return conn
}

func ReadMessage(conn net.Conn) *bridge.Message {
	var messageLength int32

	// First 4 bytes is the message length
	err := binary.Read(conn, binary.BigEndian, &messageLength)
	HandleError(err)

	buf := make([]byte, messageLength)
	err = binary.Read(conn, binary.BigEndian, buf)
	HandleError(err)

	message := &bridge.Message{}
	err = proto.Unmarshal(buf, message)
	HandleError(err)

	//fmt.Println("<--- received to", conn, " | message:", message)

	return message
}

func WriteMessage(conn net.Conn, msg *bridge.Message) {
	//fmt.Println("---> sent to ", conn, " | message:", msg)

	serialized, err := proto.Marshal(msg)
	HandleError(err)

	// Write msg length
	err = binary.Write(conn, binary.BigEndian, int32(len(serialized)))
	HandleError(err)

	// Write msg
	err = binary.Write(conn, binary.BigEndian, serialized)
	HandleError(err)
}

func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}
