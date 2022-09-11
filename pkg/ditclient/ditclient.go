package ditclient

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/TheVoxcraft/dit/pkg/ditmaster"
	"github.com/TheVoxcraft/dit/pkg/ditnet"
	"github.com/TheVoxcraft/dit/pkg/ditsync"
	"github.com/fatih/color"
)

func SyncFilesDown(parcel ditmaster.ParcelInfo, base_path string) {
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

	// get files from mirror
	for _, fpath := range fpaths {
		req = ditnet.ClientMessage{
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

func SyncFilesUp(sync_files []ditsync.SyncFile, parcel ditmaster.ParcelInfo) {
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

			ditmaster.Stores.Master[file.FilePath] = file.FileChecksum

		} else {
			color.White("\tSkipping: %s", file.FilePath)
		}
	}

	err := ditmaster.SyncStoresToDisk(".") // save stores to disk
	if err != nil {
		log.Fatal(err)
	}
}