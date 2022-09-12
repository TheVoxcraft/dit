package ditclient

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TheVoxcraft/dit/pkg/ditmaster"
	"github.com/TheVoxcraft/dit/pkg/ditnet"
	"github.com/TheVoxcraft/dit/pkg/ditsync"
	"github.com/fatih/color"
)

func SyncFilesDown(parcel ditmaster.ParcelInfo, base_path string, get_files []string) {
	fpaths := get_files
	if len(fpaths) == 0 { // get all files if none are supplied
		req := ditnet.ClientMessage{
			OriginAuthor: parcel.Author,
			ParcelPath:   parcel.RepoPath,
			MessageType:  ditnet.MSG_GET_PARCEL,
		}

		resp := ditnet.SendMessageToServer(req, parcel.Mirror) // Get file paths from mirror

		var netparcel ditnet.NetParcel
		if resp.MessageType != ditnet.MSG_PARCEL {
			color.HiRed("ERROR: Failed to get file paths from", parcel.Mirror)
			fmt.Println("Got response type", resp.MessageType, "MSG: ", resp.Message)
			return
		}

		gob.NewDecoder(bytes.NewReader(resp.Data)).Decode(&netparcel)
		fmt.Println("   ", len(netparcel.FilePaths), "files from mirror")

		fpaths := netparcel.FilePaths
		for _, file := range fpaths {
			color.Blue("\tGot %s", file)
		}
	}

	// get files from mirror
	for _, fpath := range fpaths {
		req := ditnet.ClientMessage{
			OriginAuthor: parcel.Author,
			ParcelPath:   parcel.RepoPath,
			MessageType:  ditnet.MSG_GET_FILE,
			Message:      fpath,
		}
		resp := ditnet.SendMessageToServer(req, parcel.Mirror)
		if resp.MessageType != ditnet.MSG_FILE {
			color.HiRed("ERROR: Failed to get file", fpath, "from", parcel.Mirror)
			continue
		}

		data := resp.Data
		if resp.IsGZIP { // decompress if needed
			uncompressed, err := ditsync.GZIPDecompress(data)
			if err != nil {
				color.HiRed("ERROR: Failed to decompress", fpath, "from", parcel.Mirror)
				continue
			}
			data = uncompressed
		}

		// write file to disk using os
		err := WriteFileWithDir(filepath.Join(base_path, fpath), data)
		if err != nil {
			color.HiRed("ERROR: Failed to write", fpath, "to disk")
			continue
		}

		// update master store
		// get checksum of file
		checksum, err := ditsync.GetFileChecksum(filepath.Join(base_path, fpath))
		if err != nil {
			color.HiRed("ERROR: Failed to get checksum of", fpath)
			continue
		}
		ditmaster.Stores.Master[fpath] = checksum
	}
}

func WriteFileWithDir(path string, data []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func SyncMasterUp(parcel ditmaster.ParcelInfo) {
	netmaster := ditnet.NetMaster{
		Master: ditmaster.Stores.Master,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(netmaster)
	if err != nil {
		log.Fatal(err)
	}

	msg := ditnet.ClientMessage{
		OriginAuthor: parcel.Author,
		ParcelPath:   parcel.RepoPath,
		MessageType:  ditnet.MSG_SYNC_MASTER,
		Data:         buf.Bytes(),
		IsGZIP:       false,
	}

	resp := ditnet.SendMessageToServer(msg, parcel.Mirror)
	if resp.MessageType != ditnet.MSG_SUCCESS {
		color.HiRed("ERROR: Failed to sync master store to mirror")
	}
	// int parse resp.Message to get number of files synced
	count, err := strconv.Atoi(resp.Message)
	if err != nil {
		color.HiRed("ERROR: Failed to parse number of files synced")
		return
	}
	if count > 0 {
		color.HiGreen("Removed %d files from mirror", count)
	}
}

func SyncFilesUp(sync_files []ditsync.SyncFile, parcel ditmaster.ParcelInfo, save_to_master bool) {
	for _, file := range sync_files {
		if file.IsDirty || file.IsNew {
			file_data, is_gzip := ditsync.GetFileData(file.FilePath)
			m := ditnet.ClientMessage{
				OriginAuthor: parcel.Author,
				ParcelPath:   parcel.RepoPath,
				MessageType:  ditnet.MSG_SYNC_FILE,
				Message:      file.FilePath,
				Message2:     file.FileChecksum,
				Data:         file_data,
				IsGZIP:       is_gzip,
			}

			resp := ditnet.SendMessageToServer(m, parcel.Mirror)
			if resp.MessageType != ditnet.MSG_SUCCESS {
				color.HiRed("ERROR: Failed to sync file", file.FilePath, "to", parcel.Mirror)
			}
			if file.IsNew {
				color.Green("\tAdd: %s", file.FilePath)
			} else {
				color.HiYellow("\tModified: %s", file.FilePath)
			}

			if save_to_master {
				ditmaster.Stores.Master[file.FilePath] = file.FileChecksum
			}

		} else {
			color.White("\tSkipping: %s", file.FilePath)
		}
	}

	err := ditmaster.SyncStoresToDisk(".") // save stores to disk
	if err != nil {
		log.Fatal(err)
	}
}

func GetParcelInfoFromMirror(author string, repoPath string, mirror string) (ditnet.NetParcel, error) {
	author = strings.TrimSpace(strings.ToLower(author))
	repoPath = strings.TrimSpace(strings.ToLower(repoPath))
	mirror = strings.TrimSpace(strings.ToLower(mirror))

	req := ditnet.ClientMessage{
		OriginAuthor: author,
		ParcelPath:   repoPath,
		MessageType:  ditnet.MSG_GET_PARCEL,
		//TODO: Secret: secret,
	}

	resp := ditnet.SendMessageToServer(req, mirror)
	if resp.MessageType != ditnet.MSG_PARCEL {
		return ditnet.NetParcel{}, errors.New("failed to get parcel info from mirror")
	}

	var netparcel ditnet.NetParcel
	gob.NewDecoder(bytes.NewReader(resp.Data)).Decode(&netparcel)

	netparcel.Info.Mirror = mirror

	return netparcel, nil
}

func SetDitConfig(author string, mirror string, pub_key string) string {
	// home dir
	home, err := os.UserHomeDir()
	home_dit := filepath.Join(home, ".dit")

	if err != nil {
		log.Fatal(err)
	}
	_, err = os.Create(home_dit)
	if err != nil {
		log.Fatal(err)
	}

	config_map := map[string]string{
		"author": author,
		"mirror": mirror,
		"pubkey": pub_key,
	}

	err = ditmaster.KVSave(home_dit, config_map)
	if err != nil {
		log.Fatal(err)
	}
	return home_dit
}

func GetDitFromConfig(key string) string {
	// home dir
	home, err := os.UserHomeDir()
	home_dit := filepath.Join(home, ".dit")

	if err != nil {
		log.Fatal(err)
	}

	config_map, err := ditmaster.KVLoad(home_dit)
	if err != nil {
		log.Fatal(err)
	}

	return config_map[key]
}

func PrintDitConfig() {
	// home dir
	home, err := os.UserHomeDir()
	home_dit := filepath.Join(home, ".dit")

	if err != nil {
		log.Fatal(err)
	}

	config_map, err := ditmaster.KVLoad(home_dit)
	if err != nil {
		log.Fatal(err)
	}

	for key, value := range config_map {
		fmt.Println("   ", color.MagentaString(key), ":", value)
	}
}

func CanonicalizeRepoPath(repo string) string {
	forbidden := []string{":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range forbidden {
		if strings.Contains(repo, char) {
			log.Fatal("ERROR: Repo name cannot contain", char)
		}
	}
	new := strings.TrimSpace(repo)
	new = strings.ReplaceAll(new, " ", "-")
	new = strings.ReplaceAll(new, "//", "/")
	new = strings.ReplaceAll(new, "\\", "/")
	if new[0] != '/' {
		new = "/" + new
	}
	if new[len(new)-1] != '/' {
		new = new + "/"
	}
	return strings.ToLower(new)
}

func ParseFullRepoPath(full_repo string) (string, string) {
	full_repo = strings.TrimSpace(full_repo)
	if strings.HasPrefix(full_repo, "@") && strings.Contains(full_repo, "/") {
		split := strings.Split(full_repo, "/")
		return split[0][1:], strings.Join(split[1:], "/")
	} else {
		log.Fatal("ERROR: Invalid repo path. format: @author/repo/path")
	}
	return "", ""
}
