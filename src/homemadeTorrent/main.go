package main

import (
	"HomemadeTorrent/pkg/event_loop"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) < 3 {
		log.Fatal("Usage:\n" +
			"  program <siteID> <nbNeighbors> <allSiteIDs...>\n\n" +
			"Example:\n" +
			"  go run main.go Site1 2 Site1 Site2 Site3")
	}

	siteID := args[0]
	nbNeighborsStr := os.Args[2]
	allSiteIDs := args[2:]

	parts := strings.Fields(nbNeighborsStr)
	nbNeighbors := make([]int, len(parts))
	for i, p := range parts {
		nbNeighbors[i], _ = strconv.Atoi(p)
	}

	// Lancement boucle
	event_loop.Start(allSiteIDs, siteID, nbNeighbors)
}
