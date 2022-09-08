package ditmaster

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nightlyone/lockfile"
)

const (
	DitPath      = "/.dit/"
	ManifestPath = "/.dit/manifest"
	MasterPath   = "/.dit/master"
	LockFilePath = "/.dit/dit.lock"
)

type DitMaster struct {
	Manifest map[string]string
	Master   map[string]string
}

var diskStores = DitMaster{
	Manifest: make(map[string]string),
	Master:   make(map[string]string),
}

var Stores = DitMaster{
	Manifest: make(map[string]string),
	Master:   make(map[string]string),
}

func SyncStores() error {
	// sync the stores to disk
	err := syncStoreToDisk(diskStores.Manifest, Stores.Manifest, ManifestPath)
	if err != nil {
		return err
	}
	err = syncStoreToDisk(diskStores.Master, Stores.Master, MasterPath)
	if err != nil {
		return err
	}
	return nil
}

func syncStoreToDisk(disk map[string]string, store map[string]string, path string) error {
	dirty := false
	for key, value := range store {
		disk[key] = value
		dirty = true
	}
	if dirty {
		KVSave(path, disk)
	}
	return nil
}

func CleanDitFolder(path string) error {
	ditFolderPath := filepath.Join(path, DitPath)
	// delete the folder
	err := os.RemoveAll(ditFolderPath)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func InitDitFolder(path string) error {
	ditFolderPath := filepath.Join(path, DitPath)

	// create a folder called .dit
	err := os.Mkdir(ditFolderPath, 0755)
	if err != nil {
		log.Fatal(err)
	}
	// create Lock file
	lockPath, err := filepath.Abs(filepath.Join(path, LockFilePath))
	if err != nil {
		return err
	}
	lock, err := lockfile.New(lockPath)
	if err != nil {
		return err
	}

	if err = lock.TryLock(); err != nil { // Try to lock the file
		return err
	}
	defer lock.Unlock() // Unlock the file when we're done

	// create manifest file
	err = newManifestFile(ditFolderPath)
	if err != nil {
		return err
	}
	err = newMasterRecord(ditFolderPath)
	if err != nil {
		return err
	}

	fmt.Println("Initialized .dit")
	return nil
}

func newManifestFile(path string) error {
	// create a manifest file
	_, err := os.Create(filepath.Join(path, ManifestPath))
	if err != nil {
		return err
	}
	Stores.Manifest["author"] = "@jonaslsa"
	Stores.Manifest["repo_path"] = "/my-project/v1.0/"
	Stores.Manifest["trackers"] = "dit.jonaslsa.com"
	Stores.Manifest["public_key"] = "(public key)"
	err = KVSave(filepath.Join(path, ManifestPath), Stores.Manifest)
	return err
}

func newMasterRecord(path string) error {
	// create a master record file
	_, err := os.Create(filepath.Join(path, MasterPath))
	if err != nil {
		return err
	}

	Stores.Master[".dit/master"] = "1"
	Stores.Master[".dit/manifest"] = "1"
	err = KVSave(filepath.Join(path, MasterPath), Stores.Master)
	if err != nil {
		return err
	}
	return nil
}
