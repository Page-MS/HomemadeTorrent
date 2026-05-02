package main

import (
	event_loop "HomemadeTorrent/pkg/event_loop"
)

func main() {
	/*
		// Lectures des arguments
		args := os.Args[1:]

		if len(args) < 2 {
			log.Fatal("Usage:\n" +
				"  program <siteID> <allSiteIDs...>\n\n" +
				"Example:\n" +
				"  go run main.go Site1 Site1 Site2 Site3")
		}

		siteID := args[0]
		allSiteIDs := args[1:]

		// Initialisations des composants du site
	*/

	// Lancement boucle
	allSiteIDs := []string{"Test"}
	event_loop.Start(allSiteIDs, "Test")

	// TODO: comprendre pourquoi ca ne passe pas le decodeur
}
