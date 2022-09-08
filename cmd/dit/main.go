package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TheVoxcraft/dit/pkg/ditmaster"
	"github.com/TheVoxcraft/dit/pkg/ditsync"
	"github.com/akamensky/argparse"
	"github.com/fatih/color"
)

func main() {
	parser := argparse.NewParser("dit", "A tool to sync directories")

	// Actions
	status := parser.NewCommand("status", "Show the status of the directory")
	sync := parser.NewCommand("sync", "Sync the directory")
	init := parser.NewCommand("init", "Initialize a directory")
	initClean := init.Flag("c", "clean", &argparse.Options{Required: false, Help: "Clean initialization, removes all .dit files."})

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
	files, err := ditsync.GetFileList(".")
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

		for _, file := range files {
			checksum, err := ditsync.GetFileChecksum(file)
			if err != nil {
				log.Fatal(err)
			}
			// try to get from master store
			if ditmaster.Stores.Master[file] == "" {
				color.Red("\tN %s", file)
			} else if ditmaster.Stores.Master[file] == checksum {
				color.White("\t- %s", file)
			} else {
				color.Blue("\tM %s", file)
			}
		}
	case sync.Happened():
		if !hasDitParcel {
			color.HiYellow("This directory is not a dit parcel.")
			return
		}
		PrintPreStatus(parcel, "sync")

		for _, file := range files {
			checksum, err := ditsync.GetFileChecksum(file)
			if err != nil {
				// warn and continue
				log.Println(err)
			}

			// try to get from master store
			if ditmaster.Stores.Master[file] == "" {
				color.Green("\tAdd: %s", file)
				ditmaster.Stores.Master[file] = checksum
			} else if ditmaster.Stores.Master[file] == checksum {
				color.White("\tSkipping: %s", file)
			} else {
				color.Blue("\tModified: %s", file)
				ditmaster.Stores.Master[file] = checksum
			}
			err = ditmaster.SyncStoresToDisk(".")
			if err != nil {
				log.Fatal(err)
			}
		}
	case init.Happened():
		if *initClean {
			ditmaster.CleanDitFolder(".")
		} else {
			if hasDitParcel {
				color.HiYellow("This directory is already a dit parcel. Use --clean to reinitialize.")
				return
			}
		}
		err := ditmaster.InitDitFolder(".")
		if err != nil {
			ditmaster.CleanDitFolder(".") // clean up as init failed
			log.Fatal("Failed to initialize dit folder: ", err)
		}
	}

}

func PrintPreStatus(parcel ditmaster.ParcelInfo, action string) {
	fmt.Println("* Parcel", color.YellowString(parcel.Author)+color.GreenString(parcel.RepoPath))
	fmt.Println("\n   " + action + ":")
}
