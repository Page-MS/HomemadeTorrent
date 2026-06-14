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
	log.Printf("Arguments: %v\n", args)
	siteID := args[0]
	nbNeighborsStr := args[1]
	bootstrapStr := args[2]
	allSiteIDs := args[3:]

	bootstrap, err := strconv.Atoi(bootstrapStr)
	if err != nil {
		log.Fatalf("bootstrap doit être 0 ou 1 : %v", err)
	}

	nbNeighbors, err := strconv.Atoi(nbNeighborsStr)
	if err != nil {
		log.Fatalf("nbNeighbors doit être un entier : %v", err)
	}

	// Lancement boucle
	event_loop.Start(allSiteIDs, siteID, nbNeighbors, bootstrap)
}
