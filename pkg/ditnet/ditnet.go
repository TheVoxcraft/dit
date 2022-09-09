package ditnet

import (
	"encoding/gob"
	"log"
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
	ParcelPath   string
	MessageType  int
	Message      string
	Message2     string
	Data         []byte
	IsGZIP       bool
}

type ServerMessage struct {
	MessageType int
	Message     string
}

func SendMessageToServer(msg ClientMessage, mirror_addr string) {
	conn, err := net.Dial("tcp", mirror_addr)
	if err != nil {
		log.Fatal("Failed to connect to mirror: ", err)
	}
	defer conn.Close()

	enc := gob.NewEncoder(conn)
	err = enc.Encode(msg)
	if err != nil {
		log.Fatal("Failed to send message to mirror: ", err)
	}

	// Read OK from server
	dec := gob.NewDecoder(conn)
	server_msg := &ServerMessage{}
	err = dec.Decode(server_msg)
	if err != nil {
		log.Fatal("Failed to read message from mirror: ", err)
	}

	if server_msg.MessageType == MSG_SUCCESS {
		return
	}
}
