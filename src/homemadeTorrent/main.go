package main

import (
	"HomemadeTorrent/pkg/event_loop"
	"log"
	"os"
)

func main() {
	args := os.Args[1:]
	println("Arguments:", args)
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		log.Fatal("Usage:\n" +
			"  program <siteID> <bootrstrapbool> <allSiteIDs...>\n\n" +
			"Example:\n" +
			"  go run main.go Site1 1 Site1 Site2 Site3")

	}
	if len(args) < 2 {
		log.Fatal("Usage:\n" +
			"  program <siteID> <bootrstrapbool> <allSiteIDs...>\n\n" +
			"Example:\n" +
			"  go run main.go Site1 1 Site1 Site2 Site3")
	}

	siteID := args[0]
	bootstrap := args[1]
	allSiteIDs := args[2:]

	// Lancement boucle
	if bootstrap == "1" {
		log.Printf("Bootstrap pour le site %s\n", siteID)
		event_loop.StartBootstrap(allSiteIDs, siteID)
	} else {
		event_loop.Start(allSiteIDs, siteID)

	}
}
