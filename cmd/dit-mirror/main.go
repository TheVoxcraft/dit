package main

import (
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/TheVoxcraft/dit/pkg/ditnet"
	"github.com/akamensky/argparse"
	"github.com/fatih/color"
	"github.com/mattn/go-sqlite3"
)

const (
	DITMIRROR_VERSION = "0.1.0"
)

func main() {
	parser := argparse.NewParser("dit-mirror", "Mirror server for dit clients")

	port := parser.Int("p", "port", &argparse.Options{Required: false, Help: "Port to listen on", Default: 3216})
	bind := parser.String("b", "bind", &argparse.Options{Required: false, Help: "Address to bind to", Default: "127.0.0.1"})
	db_path := parser.String("d", "db", &argparse.Options{Required: false, Help: "Path to the database", Default: "./dit.db"})
	err := parser.Parse(os.Args)
	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		fmt.Print(parser.Usage(err))
		return
	}

	fmt.Println("dit-mirror version:", DITMIRROR_VERSION)
	sqlite_version, _, _ := sqlite3.Version() // this is needed to import and initialize the sqlite3 package
	fmt.Println("SQLite version:", sqlite_version)
	ensureSQLiteDB(*db_path)

	l, err := net.Listen("tcp", *bind+":"+strconv.Itoa(*port))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	fmt.Println("Loading database:", *db_path)
	db, err := sql.Open("sqlite3", *db_path)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	color.Green("\n * Serving dit-mirror on port: %d", *port)

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c, db)
	}
}

func handleConnection(c net.Conn, db *sql.DB) {
	fmt.Printf("connection from %s\n", c.RemoteAddr().String())
	dec := gob.NewDecoder(c)
	msg := &ditnet.ClientMessage{}
	err := dec.Decode(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "recv error:", err)
		return
	}
	if msg.MessageType == ditnet.MSG_SYNC_FILE {
		fmt.Println(color.YellowString("@"+msg.OriginAuthor)+msg.ParcelPath, "~", msg.Message)

		SyncFileToDB(db, msg.OriginAuthor, msg.ParcelPath, msg.Message, msg.Message2, msg.Data, msg.IsGZIP)

		success := ditnet.ServerMessage{
			MessageType: ditnet.MSG_SUCCESS,
			Message:     "OK",
		}
		enc := gob.NewEncoder(c)
		err = enc.Encode(success)
		if err != nil {
			fmt.Fprintln(os.Stderr, "senc error:", err)
			return
		}
	}
}

func SyncFileToDB(db *sql.DB, author string, parcel string, path string, checksum string, data []byte, isGZIP bool) {
	var id int
	err := db.QueryRow("SELECT id FROM files WHERE author = ? AND parcel = ? AND path = ?", author, parcel, path).Scan(&id)
	// if err is sql.ErrNoRows
	timestamp := time.Now().String()

	if errors.Is(err, sql.ErrNoRows) {
		// insert
		fmt.Println("insert")
		stmt, err := db.Prepare("INSERT INTO files (author, parcel, path, checksum, data, isGZIP, created, last_sync) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			panic(err)
		}
		defer stmt.Close()
		_, err = stmt.Exec(author, parcel, path, checksum, data, isGZIP, timestamp, timestamp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "insert error: %v", err)
		}
	} else {
		// update
		fmt.Println("update")
		stmt, err := db.Prepare("UPDATE files SET checksum = ?, data = ?, isGZIP = ?, last_sync = ? WHERE id = ?")
		if err != nil {
			panic(err)
		}
		defer stmt.Close()
		_, err = stmt.Exec(checksum, data, isGZIP, timestamp, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "update error: %v", err)
		}
	}
}

func ensureSQLiteDB(db_path string) {
	// open or create database
	db, err := sql.Open("sqlite3", db_path)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// create table if not exists, id, file path, checksum, data blob, isGZIP bool, created timestamp, last_sync timestamp
	sqlStmt := `
	create table if not exists files (id integer not null primary key, author text, parcel text, path text, checksum text, data blob, isGZIP bool, created timestamp, last_sync timestamp);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		panic(err)
	}
}
