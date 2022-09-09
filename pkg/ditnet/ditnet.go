package ditnet

import (
	"encoding/gob"
	"net"
)

const (
	// MessageTypes
	MSG_PING = iota
	MSG_PONG = iota

	MSG_REGISTER = iota

	MSG_SUCCESS = iota
	MSG_FAILURE = iota

	MSG_NEW_PARCEL = iota
	MSG_SYNC_FILE  = iota
)

type ClientMessage struct {
	OriginAuthor string
	MessageType  int
	Message      string
	Data         []byte
	IsGZIP       bool
}

func SendMessageToServer(msg ClientMessage, mirror_addr string) {
	conn, err := net.Dial("tcp", mirror_addr)
	if err != nil {
		return
	}
	defer conn.Close()

	enc := gob.NewEncoder(conn)
	err = enc.Encode(msg)
	if err != nil {
		return
	}
}
