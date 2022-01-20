package utils

import (
	"encoding/binary"
	"net"
)

const (
	UnixNetwork = "unix"
	Address     = "/tmp/cadence.socket"
)

func ReadMessage(conn net.Conn) string {
	var messageLength int32

	// First 4 bytes is the size of message_content
	err := binary.Read(conn, binary.BigEndian, &messageLength)
	HandleError(err)

	buf := make([]byte, messageLength)
	err = binary.Read(conn, binary.BigEndian, buf)
	HandleError(err)

	return string(buf)
}

func WriteMessage(conn net.Conn, content string) {
	serialized := []byte(content)
	err := binary.Write(conn, binary.BigEndian, int32(len(serialized)))
	HandleError(err)

	_, err = conn.Write(serialized)
	HandleError(err)
}

func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}
