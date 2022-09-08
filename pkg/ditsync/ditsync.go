package ditsync

import (
	"crypto/sha256"
	"encoding/base32"
	"log"
	"os"
	"path/filepath"
)

var IgnoreList = []string{".git", ".gitignore", ".dit"}

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
