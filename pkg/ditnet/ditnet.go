package ditnet

import (
	"encoding/gob"
	"log"
	"net"

	"github.com/TheVoxcraft/dit/pkg/ditmaster"
)

const (
	/* MessageTypes */

	// Client -> Server
	MSG_NEW_PARCEL  = iota // unused
	MSG_SYNC_FILE   = iota
	MSG_SYNC_MASTER = iota
	MSG_GET_PARCEL  = iota
	MSG_GET_FILE    = iota

	// Server -> Client
	MSG_REGISTER = iota // unused
	MSG_SUCCESS  = iota
	MSG_FAILURE  = iota // unused
	MSG_PARCEL   = iota
	MSG_FILE     = iota
)

type ClientMessage struct {
	OriginAuthor string
	ParcelPath   string
	MessageType  int
	Message      string
	Message2     string
	Data         []byte
	IsGZIP       bool
	Secret       string
}

type ServerMessage struct {
	MessageType int
	Message     string
	Data        []byte
	IsGZIP      bool
}

type NetParcel struct {
	Info      ditmaster.ParcelInfo
	FilePaths []string
}

type NetMaster struct { // Used to sync local master with remote master (removing deleted files)
	Master map[string]string
}

func SendMessageToServer(msg ClientMessage, mirror_addr string) ServerMessage {
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

	// Read RESPONSE from server
	dec := gob.NewDecoder(conn)
	server_msg := &ServerMessage{}
	err = dec.Decode(server_msg)
	if err != nil {
		log.Fatal("Failed to read message from mirror: ", err)
	}

	return *server_msg
}
