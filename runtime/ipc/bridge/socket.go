package bridge

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	UnixNetwork            = "unix"
	RuntimeSocketAddress   = "/tmp/runtime.socket"
	InterfaceSocketAddress = "/tmp/interface.socket"
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

func ReadMessage(conn net.Conn) Message {
	var messageLength int32

	// First 4 bytes is the message length
	err := binary.Read(conn, binary.BigEndian, &messageLength)
	HandleError(err)

	buf := make([]byte, messageLength)
	err = binary.Read(conn, binary.BigEndian, buf)
	HandleError(err)

	message := &anypb.Any{}
	err = proto.Unmarshal(buf, message)
	HandleError(err)

	// Unwrap `Any` to get the specific type of message.
	typedMessage, err := message.UnmarshalNew()
	HandleError(err)

	fmt.Println("<--- received to", conn, " | message:", message)

	return typedMessage
}

func ReadResponse(conn net.Conn) (*Response, error) {
	msg := ReadMessage(conn)

	switch msg := msg.(type) {
	case *Response:
		return msg, nil
	case *Error:
		return nil, fmt.Errorf(msg.GetErr())
	default:
		return nil, fmt.Errorf("unsupported message")
	}
}

func WriteMessage(conn net.Conn, msg Message) {
	fmt.Println("---> sent to ", conn, " | message:", msg)

	// Wrap with `Any` to enrich type information for unmarshalling.
	typedMessage, err := anypb.New(msg)

	serialized, err := proto.Marshal(typedMessage)
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
