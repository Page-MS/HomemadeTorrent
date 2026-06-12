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
	bootstrapStr := args[2]
	allSiteIDs := args[3:]

	nbNeighbors, _ := strconv.Atoi(nbNeighborsStr)
	bootstrap, _ := strconv.Atoi(bootstrapStr)

	event_loop.Start(allSiteIDs, siteID, nbNeighbors, bootstrap)
}
