package ditsync

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base32"
	"log"
	"os"
	"path/filepath"
)

const (
	MINIMUM_GZIP_SIZE = 1024
)

var IgnoreList = []string{".git", ".gitignore", ".dit"}

type SyncFile struct {
	FilePath     string
	FileChecksum string
	IsDirty      bool
	IsNew        bool
}

func isIgnored(path string) bool {
	// TODO: iterate through path components and check if any of them are in the ignore list
	for _, ignore := range IgnoreList {
		ignoreLen := len(ignore)
		if len(path) >= ignoreLen {
			if path[:ignoreLen] == ignore {
				return true
			}
		}
	}
	return false
}

func GetFileList(path string) ([]string, error) {
	files := make([]string, 0, 10)
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if isIgnored(path) {
				return nil
			}
			rel_path, err := filepath.Rel(".", path)
			if err != nil {
				return err
			}
			files = append(files, rel_path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func GetFileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	hash := sha256.Sum256(data)

	friendly_string := base32.StdEncoding.EncodeToString(hash[:])
	return friendly_string, nil
}

/* SerializedFile: Unused for now
type SerializedFile struct {
	FilePath     string
	FileChecksum string
	File         []byte
}

func SerializeFile(path string) SerializedFile {
	checksum, err := GetFileChecksum(path)
	if err != nil {
		log.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return SerializedFile{
		FilePath:     path,
		FileChecksum: checksum,
		File:         data,
	}
}*/

func GetFileData(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	// if data size is larger than MINIMUM_GZIP_SIZE bytes compress it
	if len(data) >= MINIMUM_GZIP_SIZE {
		compressed, err := GZIPCompress(data)
		if err != nil {
			log.Fatal(err)
		}
		return compressed, true
	}
	return data, false
}

func GZIPCompress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func GZIPDecompress(data []byte) ([]byte, error) {
	b := bytes.NewBuffer(data)
	var out bytes.Buffer
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	if _, err := out.ReadFrom(r); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
