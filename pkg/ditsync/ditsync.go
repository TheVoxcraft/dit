package dirsync

import (
	"crypto/sha256"
	"log"
	"os"
)

func GetFileList(path string) ([]string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	// make a slice of strings
	var fileList []string

	for _, file := range files {
		fileList = append(fileList, file.Name())
	}

	return fileList, nil
}

func GetFileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	hash := sha256.Sum256(data)
	return string(hash[:]), nil
}
