package bridge

import (
	"encoding/binary"
	"fmt"
	"github.com/onflow/cadence/runtime/ipc/protobuf"
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

func NewRuntimeListener() (net.Listener, error) {
	syscall.Unlink(RuntimeSocketAddress)

	listener, err := net.Listen(UnixNetwork, RuntimeSocketAddress)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func NewRuntimeConnection() (net.Conn, error) {
	conn, err := net.Dial(UnixNetwork, RuntimeSocketAddress)
	if err != nil {
		// Do not expose network info to the user.
		// Return a generic error instead.
		return nil, fmt.Errorf("cannot connect to cadence runtime")
	}

	return conn, nil
}

func NewInterfaceListener() (net.Listener, error) {
	syscall.Unlink(InterfaceSocketAddress)
	listener, err := net.Listen(UnixNetwork, InterfaceSocketAddress)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func NewInterfaceConnection() (net.Conn, error) {
	conn, err := net.Dial(UnixNetwork, InterfaceSocketAddress)
	if err != nil {
		// Do not expose network info to the user.
		// Return a generic error instead.
		return nil, fmt.Errorf("cannot connect to host-env")
	}

	return conn, nil
}

func ReadMessage(conn net.Conn) pb.Message {
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

func ReadResponse(conn net.Conn) (*pb.Response, error) {
	msg := ReadMessage(conn)

	switch msg := msg.(type) {
	case *pb.Response:
		return msg, nil
	case *pb.Error:
		return nil, fmt.Errorf(msg.GetErr())
	default:
		return nil, fmt.Errorf("unsupported message")
	}
}

func WriteMessage(conn net.Conn, msg pb.Message) {
	fmt.Println("---> sent to ", conn, " | message:", msg)

	// Wrap with `Any` to enrich type information for unmarshalling.
	typedMessage := pb.AsAny(msg)

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
	// TODO: handle EOF error
	if err != nil {
		panic(err)
	}
}
