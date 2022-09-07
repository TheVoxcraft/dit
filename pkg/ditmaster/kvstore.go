package ditmaster

import (
	"bufio"
	"os"
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

func KVSave(path string, store map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for key, value := range store {
		line := key + kvDELIMITER + value
		writer.WriteString(line + "\n")
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
