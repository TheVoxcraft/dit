package ditmaster

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

var kvDELIMITER string = "|"

func KVLoad(path string) (map[string]string, error) {
	// for each line in the file
	// split the line into key and value
	// store the key and value in the map
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	store := make(map[string]string)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		key, value := splitLineOnDelimiter(line)
		if key != "" {
			store[key] = value
		}
	}
	return store, nil
}

func KVSave(path string, store map[string]string) error { // save keys to exisiting file
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	if len(store) == 0 {
		os.Truncate(path, 0)
	} else {
		for key, value := range store {
			if strings.Contains(key+value, kvDELIMITER) {
				return errors.New("key or value contains delimiter")
			} else if strings.Contains(key+value, "\n") {
				return errors.New("key or value contains newline")
			}
			line := key + kvDELIMITER + value
			writer.WriteString(line + "\n")
			if err != nil {
				return err
			}
		}
	}
	writer.Flush()

	return nil
}

func splitLineOnDelimiter(line []byte) (string, string) {
	for i, char := range string(line) {
		if string(char) == kvDELIMITER {
			return string(line[:i]), string(line[i+1:])
		}
	}
	return "", ""
}

func ExtendKVStore(dst, src map[string]string) {
	for key, value := range src {
		dst[key] = value
	}
}

func CopyKVStore(src map[string]string) map[string]string {
	dst := make(map[string]string)
	ExtendKVStore(dst, src)
	return dst
}
