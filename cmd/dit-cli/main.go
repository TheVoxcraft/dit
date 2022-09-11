package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/TheVoxcraft/dit/pkg/ditclient"
	"github.com/TheVoxcraft/dit/pkg/ditmaster"
	"github.com/TheVoxcraft/dit/pkg/ditsync"
	"github.com/akamensky/argparse"
	"github.com/fatih/color"
)

func main() {
	parser := argparse.NewParser("dit", "A tool to sync directories")

	// Actions
	status := parser.NewCommand("status", "Show the status of the directory")

	config := parser.NewCommand("config", "Configure dit")
	config_set := config.NewCommand("set", "Set a config value")
	config_print := config.NewCommand("print", "Print the config to stdout")
	configSetAuthor := config_set.String("a", "author", &argparse.Options{Required: true, Help: "Author for parcels.", Default: ""})
	configSetMirror := config_set.String("m", "mirror", &argparse.Options{Required: true, Help: "Default mirror to use.", Default: ""})
	//configPublicKey := config.String("p", "public-key", &argparse.Options{Required: true, Help: "Path to the public key.", Default: ""})

	sync := parser.NewCommand("sync", "Sync the directory")
	sync_up := sync.NewCommand("up", "Sync the directory to the server")
	sync_down := sync.NewCommand("down", "Sync the directory from the server")

	init := parser.NewCommand("init", "Initialize a directory")
	initClean := init.Flag("c", "clean", &argparse.Options{Required: false, Help: "Clean initialization, removes all files in .dit"})
	initRepoPath := init.String("r", "repo", &argparse.Options{Required: true, Help: "Path to the repository, used to identify the parcel."})
	initMirror := init.String("m", "mirror", &argparse.Options{Required: false, Help: "Mirror to use for the parcel, overrides the default mirror."})

	err := parser.Parse(os.Args)
	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		fmt.Print(parser.Usage(err))
		return
	}

	hasDitParcel := ditmaster.HasDitParcel(".") // check if the current directory has a .dit folder

	// try to load parcel
	if hasDitParcel {
		err = ditmaster.LoadStoresFromDisk(".")
		if err != nil {
			log.Fatal(err)
		}
	}
	parcel := ditmaster.GetParcelInfo(".")
	parcel_files, err := ditsync.GetFileList(".")
	if err != nil {
		log.Fatal(err)
	}

	switch {
	case status.Happened():
		if !hasDitParcel {
			color.HiYellow("This directory is not a dit parcel.")
			return
		}
		PrintPreStatus(parcel, "status")

		for _, file := range parcel_files {
			checksum, err := ditsync.GetFileChecksum(file)
			if err != nil {
				log.Fatal(err)
			}
			// try to get from master store
			if ditmaster.Stores.Master[file] == "" {
				color.Red("\tN %s", file)
			} else if ditmaster.Stores.Master[file] == checksum {
				color.White("\t  %s", file)
			} else {
				color.HiYellow("\tM %s", file)
			}
		}
	case config.Happened():
		if config_set.Happened() {
			author := *configSetAuthor
			mirror := *configSetMirror
			//publicKey := *configPublicKey

			if author == "" {
				log.Fatal("Author cannot be empty")
			}
			if mirror == "" {
				log.Fatal("Mirror cannot be empty")
			}
			//if publicKey == "" {
			//	log.Fatal("Public key cannot be empty")
			//}

			ditclient.SetDitConfig(author, mirror, "")
			fmt.Println(color.CyanString("[-]"), "Config set.")
		} else if config_print.Happened() {
			fmt.Println(color.CyanString("[-]"), "Dit config")
			ditclient.PrintDitConfig()
		}
	case sync.Happened():
		if !hasDitParcel {
			color.HiYellow("This directory is not a dit parcel.")
			return
		}
		PrintPreStatus(parcel, "sync")

		if ditmaster.Stores.Manifest["author"] == "" {
			color.HiYellow("Author not set, please use 'dit config -a <author>'")
			return
		}

		if sync_up.Happened() {
			sync_files := make([]ditsync.SyncFile, 0)
			for _, file := range parcel_files {
				checksum, err := ditsync.GetFileChecksum(file)
				if err != nil {
					// warn and continue
					log.Println(err)
				}

				curr := ditsync.SyncFile{
					FilePath:     file,
					FileChecksum: checksum,
					IsDirty:      false,
					IsNew:        false,
				}

				// try to get from master store
				if ditmaster.Stores.Master[file] == "" {
					curr.IsNew = true
				} else if ditmaster.Stores.Master[file] != checksum {
					curr.IsDirty = true
				}
				sync_files = append(sync_files, curr)
			}
			ditclient.SyncFilesUp(sync_files, parcel)
		} else if sync_down.Happened() {
			ditclient.SyncFilesDown(parcel, "test/") // TODO: change to . for current directory
			ditmaster.SyncStoresToDisk(".")          // save stores to disk
		} else {
			fmt.Println(parser.Usage(err))
		}

	case init.Happened():
		if *initClean {
			ditmaster.CleanDitFolder(".")
			time.Sleep(1000 * time.Millisecond) // wait for the folder to be deleted before writing TODO: fix this
		} else {
			if hasDitParcel {
				color.HiYellow("This directory is already a dit parcel. Use --clean to reinitialize.")
				return
			}
		}

		if *initRepoPath == "" {
			log.Fatal("Repo path cannot be empty")
		}

		// try to get author and mirror from config
		author := ditclient.GetDitFromConfig("author")
		mirror := ditclient.GetDitFromConfig("mirror")
		if author == "" || mirror == "" {
			color.HiYellow("Author and/or mirror not set, please use 'dit config set'")
			return
		}

		if *initMirror != "" { // override mirror for this parcel
			mirror = *initMirror
		}

		canonicalRepoPath := ditclient.CanonicalizeRepoPath(*initRepoPath)

		parcel_info := ditmaster.ParcelInfo{
			Author:   strings.TrimSpace(author),
			RepoPath: canonicalRepoPath,
			Mirror:   strings.TrimSpace(mirror),
		}

		err := ditmaster.InitDitFolder(".", parcel_info)
		if err != nil {
			ditmaster.CleanDitFolder(".") // clean up as init failed
			log.Fatal("Failed to initialize dit folder: ", err)
		}
	}

}

func PrintPreStatus(parcel ditmaster.ParcelInfo, action string) {
	fmt.Println(color.CyanString("[-]"), "Parcel", color.YellowString(parcel.Author)+color.GreenString(parcel.RepoPath))
	color.Blue("           Mirror %s", parcel.Mirror)
	fmt.Println("\n    " + action + ":")
}
