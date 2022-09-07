package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TheVoxcraft/dit/pkg/ditmaster"
	_ "github.com/TheVoxcraft/dit/pkg/ditsync"
	"github.com/akamensky/argparse"
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
	}

	switch {
	case status.Happened():
		fmt.Println("Status")
	case sync.Happened():
		fmt.Println("Sync")
	case init.Happened():
		if *initClean {
			ditmaster.CleanDitFolder(".")
		}
		err := ditmaster.InitDitFolder(".")
		if err != nil {
			ditmaster.CleanDitFolder(".") // clean up as init failed
			log.Fatal("Failed to initialize dit folder: ", err)
		}
	}

}
