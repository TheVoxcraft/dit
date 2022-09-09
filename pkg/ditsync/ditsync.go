package ditsync

import (
	"crypto/sha256"
	"encoding/base32"
	"log"
	"os"
	"path/filepath"
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

type SerializedFile struct {
	FilePath     string
	FileChecksum string
	File         []byte
}

func SerializeFiles(files []string) []SerializedFile {
	serialized_files := make([]SerializedFile, 0, len(files))
	for _, file := range files {
		checksum, err := GetFileChecksum(file)
		if err != nil {
			log.Fatal(err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}
		serialized_files = append(serialized_files, SerializedFile{
			FilePath:     file,
			FileChecksum: checksum,
			File:         data,
		})
	}
	return serialized_files
}
