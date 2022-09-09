package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/TheVoxcraft/dit/pkg/ditnet"
	"github.com/akamensky/argparse"
	"github.com/fatih/color"
)

func main() {
	parser := argparse.NewParser("dit-mirror", "Mirror server for dit clients")

	port := parser.Int("p", "port", &argparse.Options{Required: false, Help: "Port to listen on", Default: 3216})
	bind := parser.String("b", "bind", &argparse.Options{Required: false, Help: "Address to bind to", Default: "127.0.0.1"})
	err := parser.Parse(os.Args)
	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		fmt.Print(parser.Usage(err))
		return
	}

	l, err := net.Listen("tcp", *bind+":"+strconv.Itoa(*port))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	color.Green("Serving dit-mirror on port: %d", *port)

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	fmt.Printf("connection from %s\n", c.RemoteAddr().String())
	dec := gob.NewDecoder(c)
	msg := &ditnet.ClientMessage{}
	err := dec.Decode(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	if msg.MessageType == ditnet.MSG_SYNC_FILE {
		fmt.Println("sync", msg.Message)
	}
}
