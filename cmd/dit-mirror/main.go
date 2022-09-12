package main

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TheVoxcraft/dit/pkg/ditmaster"
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
	defer c.Close()

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
	} else if msg.MessageType == ditnet.MSG_GET_PARCEL {
		fmt.Println("GET_PARCEL", color.YellowString("@"+msg.OriginAuthor)+msg.ParcelPath)
		netparcel, err := GetParcelFiles(db, msg.OriginAuthor, msg.ParcelPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "db error:", err)
			return
		}

		// gob encode nparcel to bytes
		var parcelBytes bytes.Buffer
		enc := gob.NewEncoder(&parcelBytes)
		err = enc.Encode(netparcel)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gob encode error:", err)
			return
		}

		parcel_msg := ditnet.ServerMessage{
			MessageType: ditnet.MSG_PARCEL,
			Message:     "@" + msg.OriginAuthor + msg.ParcelPath,
			Data:        parcelBytes.Bytes(),
		}
		enc = gob.NewEncoder(c)
		err = enc.Encode(parcel_msg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "send error:", err)
			return
		}

	} else if msg.MessageType == ditnet.MSG_GET_FILE {
		fmt.Println("GET_FILE", color.YellowString("@"+msg.OriginAuthor)+msg.ParcelPath, "["+msg.Message+"]")
		filedata, gzip, err := GetFile(db, msg.OriginAuthor, msg.ParcelPath, msg.Message)
		if err != nil {
			fmt.Fprintln(os.Stderr, "db error:", err)
			return
		}

		file_msg := ditnet.ServerMessage{
			MessageType: ditnet.MSG_FILE,
			Message:     msg.Message,
			Data:        filedata,
			IsGZIP:      gzip,
		}
		enc := gob.NewEncoder(c)
		err = enc.Encode(file_msg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "send error:", err)
			return
		}
	} else if msg.MessageType == ditnet.MSG_SYNC_MASTER {
		fmt.Println("SYNC_MASTER", color.YellowString("@"+msg.OriginAuthor)+msg.ParcelPath)
		// decode to netmaster
		var netmaster ditnet.NetMaster
		dec := gob.NewDecoder(bytes.NewReader(msg.Data))
		err := dec.Decode(&netmaster)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gob decode error:", err)
			return
		}

		removed := RemoveFilesNotInMaster(db, msg.OriginAuthor, msg.ParcelPath, netmaster.Master)
		removed_str := strconv.Itoa(removed)
		success := ditnet.ServerMessage{
			MessageType: ditnet.MSG_SUCCESS,
			Message:     removed_str,
		}
		enc := gob.NewEncoder(c)
		err = enc.Encode(success)
		if err != nil {
			fmt.Fprintln(os.Stderr, "send error:", err)
			return
		}

	} else if msg.MessageType == ditnet.MSG_REGISTER {
		fmt.Println("REGISTER", color.YellowString("@"+msg.OriginAuthor)+msg.ParcelPath)

	} else {
		fmt.Println("unknown message type:", msg.MessageType)
	}
}

func RemoveFilesNotInMaster(db *sql.DB, author string, parcelpath string, master map[string]string) int {
	author = strings.TrimPrefix(author, "@")
	// for each row in the database, check if it is in the master
	// if it doesn't, delete it
	rows, err := db.Query("SELECT path, checksum FROM files WHERE author=? AND parcel=?", author, parcelpath)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	defer rows.Close()

	removed := 0

	del_tx, err := db.Begin()
	if err != nil {
		fmt.Println("del_tx begin error:", err)
		return 0
	}

	for rows.Next() {
		var path string
		var checksum string
		err = rows.Scan(&path, &checksum)
		if err != nil {
			fmt.Println(err)
			return removed
		}

		// check if the file is in the master
		if master[path] == "" {
			// if not, delete it
			fmt.Println("DEL", path)
			_, err = del_tx.Exec("DELETE FROM files WHERE path=? AND checksum=?", path, checksum)
			if err != nil {
				fmt.Println("DEL_TX ERROR", err)
				return removed
			}
			removed++
		}
	}
	err = del_tx.Commit()
	if err != nil {
		fmt.Println("del_tx commit error:", err)
	}
	return removed
}

func GetParcelFiles(db *sql.DB, author string, parcel string) (ditnet.NetParcel, error) {
	author = strings.TrimPrefix(author, "@")
	rows, err := db.Query("SELECT * FROM files WHERE author=? AND parcel=?", author, parcel)
	if err != nil {
		return ditnet.NetParcel{}, err
	}
	defer rows.Close()

	filePaths := make([]string, 0)

	for rows.Next() {
		var id int
		var author string
		var repopath string
		var filepath string
		var checksum string
		var data []byte
		var isGZIP bool
		var created string
		var last_sync string
		err = rows.Scan(&id, &author, &repopath, &filepath, &checksum, &data, &isGZIP, &created, &last_sync)
		if err != nil {
			return ditnet.NetParcel{}, err
		}
		filePaths = append(filePaths, filepath)
	}

	netparcel := ditnet.NetParcel{
		Info:      ditmaster.ParcelInfo{Author: author, RepoPath: parcel},
		FilePaths: filePaths,
	}

	return netparcel, nil
}

func GetFile(db *sql.DB, author string, parcel string, file string) ([]byte, bool, error) {
	author = strings.TrimPrefix(author, "@")
	row := db.QueryRow("SELECT data, isGZIP FROM files WHERE author=? AND parcel=? AND path=?", author, parcel, file)
	if row.Err() != nil {
		return nil, false, row.Err()
	}
	var data []byte
	var isGZIP bool

	err := row.Scan(&data, &isGZIP)
	if err != nil {
		return nil, false, err
	}

	return data, isGZIP, nil
}

func SyncFileToDB(db *sql.DB, author string, parcel string, path string, checksum string, data []byte, isGZIP bool) {
	author = strings.TrimPrefix(author, "@")
	var id int
	err := db.QueryRow("SELECT id FROM files WHERE author = ? AND parcel = ? AND path = ?", author, parcel, path).Scan(&id)
	// if err is sql.ErrNoRows
	timestamp := time.Now().String()

	if errors.Is(err, sql.ErrNoRows) { // TODO: check if this works as expected
		// insert
		//fmt.Println("insert")
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
		//fmt.Println("update")
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
