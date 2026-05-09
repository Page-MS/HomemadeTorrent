package main

import (
	"HomemadeTorrent/pkg/event_loop"
	"HomemadeTorrent/pkg/registre"
	"log"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal("Usage:\n" +
			"  program <siteID> <allSiteIDs...>\n\n" +
			"Example:\n" +
			"  go run main.go Site1 Site1 Site2 Site3")
	}

	siteID := args[0]
	allSiteIDs := args[1:]

	log.Printf("Shasum: %s\n", registre.CalculateShasum("../../bin/baseFiles/decoyduck.png"))

	// Lancement boucle
	event_loop.Start(allSiteIDs, siteID)
}
