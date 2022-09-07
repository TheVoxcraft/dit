package ditmaster

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nightlyone/lockfile"
)

type DitMaster struct {
	Manifest map[string]string
	Master   map[string]string
}

var Stores = DitMaster{
	Manifest: make(map[string]string),
	Master:   make(map[string]string),
}

func CleanDitFolder(path string) error {
	ditFolder := path + "/.dit"
	// delete the folder
	err := os.RemoveAll(ditFolder)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func InitDitFolder(path string) error {
	ditFolder := path + "/.dit"

	// create a folder called .dit
	err := os.Mkdir(ditFolder, 0755)
	if err != nil {
		log.Fatal(err)
	}
	// create Lock file
	lockPath, err := filepath.Abs(ditFolder + "/dit.lock")
	if err != nil {
		return err
	}
	lock, err := lockfile.New(lockPath)
	if err != nil {
		return err
	}
	err = lock.TryLock()
	defer lock.Unlock()
	if err != nil {
		return err
	}
	// create manifest file
	err = newManifestFile(ditFolder)
	if err != nil {
		return err
	}
	err = newMasterRecord(ditFolder)
	if err != nil {
		return err
	}

	fmt.Println("Initialized .dit")
	return nil
}

func newManifestFile(path string) error {
	// create a manifest file
	_, err := os.Create(path + "/manifest")
	if err != nil {
		return err
	}
	Stores.Manifest["author"] = "@jonaslsa"
	Stores.Manifest["repo_path"] = "/my-project/v1.0/"
	Stores.Manifest["trackers"] = "dit.jonaslsa.com"
	Stores.Manifest["public_key"] = "(public key)"
	err = KVSave(path+"/manifest", Stores.Manifest)
	return err
}

func newMasterRecord(path string) error {
	// create a master record file
	_, err := os.Create(path + "/master")
	if err != nil {
		return err
	}

	Stores.Master[".dit/master"] = "1"
	Stores.Master[".dit/manifest"] = "1"
	err = KVSave(path+"/master", Stores.Master)
	if err != nil {
		return err
	}
	return nil
}
