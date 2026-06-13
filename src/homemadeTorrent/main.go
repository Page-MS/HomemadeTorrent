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
			"  program <siteID> <siteAddress> <nbNeighbors> <bootrstrapbool> <allSiteIDs...>\n\n" +
			"Example:\n" +
			"  go run main.go Site1 /tmp/network_fifos/in_node1 2 1 Site1 Site2 Site3")

	}
	log.Printf("Arguments: %v\n", args)
	siteID := args[0]
	siteAddress := args[1]
	nbNeighborsStr := args[2]
	bootstrapStr := args[3]
	allSiteIDs := args[4:]

	bootstrap, err := strconv.Atoi(bootstrapStr)
	if err != nil {
		log.Fatalf("bootstrap doit être 0 ou 1 : %v", err)
	}

	nbNeighbors, err := strconv.Atoi(nbNeighborsStr)
	if err != nil {
		log.Fatalf("nbNeighbors doit être un entier : %v", err)
	}

	// Lancement boucle
	if bootstrap == 1 {
		event_loop.StartBootstrap(siteID)
	} else {
		event_loop.Start(allSiteIDs, siteID, nbNeighbors, siteAddress, bootstrap)
	}
}
