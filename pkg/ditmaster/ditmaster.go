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

type ParcelInfo struct {
	Author    string
	RepoPath  string
	Mirror    string
	publicKey string
}

var diskStores = DitMaster{ // these stores are supposed to be synced to disk data
	Manifest: make(map[string]string),
	Master:   make(map[string]string),
}

var Stores = DitMaster{
	Manifest: make(map[string]string),
	Master:   make(map[string]string),
}

func HasDitParcel(path string) bool {
	// check if the folder has a .dit folder, return true if it does
	_, err := os.Stat(filepath.Join(path, DitPath))
	if err != nil {
		return false
	}
	return true
}

func SyncStoresToDisk(path string) error {
	// sync the stores to disk
	err := syncStoreToDisk(diskStores.Manifest, Stores.Manifest, filepath.Join(path, ManifestPath))
	if err != nil {
		return err
	}
	err = syncStoreToDisk(diskStores.Master, Stores.Master, filepath.Join(path, MasterPath))
	if err != nil {
		return err
	}
	return nil
}

func LoadStoresFromDisk(path string) error {
	manifest, err := KVLoad(filepath.Join(path, ManifestPath))
	if err != nil {
		return err
	}
	master, err := KVLoad(filepath.Join(path, MasterPath))
	if err != nil {
		return err
	}
	ExtendKVStore(Stores.Manifest, manifest)
	ExtendKVStore(Stores.Master, master)
	// copy the stores to disk
	diskStores.Manifest = CopyKVStore(Stores.Manifest)
	diskStores.Master = CopyKVStore(Stores.Master)
	return nil
}

func syncStoreToDisk(disk map[string]string, store map[string]string, path string) error {
	dirty := false
	for key, value := range store {
		if disk[key] != value {
			disk[key] = value
			dirty = true
		}
	}
	if dirty {
		err := KVSave(path, disk)
		if err != nil {
			log.Fatal(err)
		}
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
	err = newManifestFile(path)
	if err != nil {
		return err
	}
	err = newMasterRecord(path)
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
	Stores.Manifest["author"] = "jonaslsa"
	Stores.Manifest["repo_path"] = "/my-project/v1.0/"
	Stores.Manifest["mirror"] = "localhost:3216"
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

func GetParcelInfo(path string) ParcelInfo {
	return ParcelInfo{
		Author:    Stores.Manifest["author"],
		RepoPath:  Stores.Manifest["repo_path"],
		Mirror:    Stores.Manifest["mirror"],
		publicKey: Stores.Manifest["public_key"],
	}
}
