package main

import (
	"HomemadeTorrent/pkg/event_loop"
	"log"
	"os"
	"strconv"
)

func main() {
	args := os.Args[1:]

	if len(args) < 4 || (args[0] == "-h" || args[0] == "--help") {
		log.Fatal("Usage:\n" +
			"  program <siteID> <nbNeighbors> <bootrstrapbool> <allSiteIDs...>\n\n" +
			"Example:\n" +
			"  go run main.go Site1 2 1 Site1 Site2 Site3")

	}

	siteID := args[0]
	nbNeighborsStr := args[1]
	bootstrap := args[2]
	allSiteIDs := args[3:]

	nbNeighbors, _ := strconv.Atoi(nbNeighborsStr)

	// Lancement boucle
	if bootstrap == "1" {
		log.Printf("Bootstrap pour le site %s\n", siteID)
		event_loop.StartBootstrap(siteID)
	} else {
		event_loop.Start(allSiteIDs, siteID, nbNeighbors)

	}
}
