package ipc

import (
	"encoding/binary"
	"net"

	"github.com/golang/protobuf/proto"
	"github.com/onflow/cadence/runtime/ipc/bridge"
)

const (
	UnixNetwork   = "unix"
	SocketAddress = "/tmp/cadence.socket"
)

func ReadMessage(conn net.Conn) *bridge.Message {
	var messageLength int32

	// First 4 bytes is the size of message_content
	err := binary.Read(conn, binary.BigEndian, &messageLength)
	HandleError(err)

	buf := make([]byte, messageLength)
	err = binary.Read(conn, binary.BigEndian, buf)
	HandleError(err)

	message := &bridge.Message{}
	err = proto.Unmarshal(buf, message)
	HandleError(err)

	return message
}

func WriteMessage(conn net.Conn, msg *bridge.Message) {
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
