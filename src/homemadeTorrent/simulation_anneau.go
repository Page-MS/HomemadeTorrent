package main

import (
	"HomemadeTorrent/pkg/control"
	"fmt"
	"log"
)

type Msg struct {
	from int
	msg  string
}

func testAnneau() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	nbSites := 3

	// IDs globaux
	allSiteIDs := make([]string, nbSites)
	for i := 0; i < nbSites; i++ {
		allSiteIDs[i] = fmt.Sprintf("%d", i)
	}

	// Création des sites
	sites := make([]*control.Controller, nbSites)
	for i := 0; i < nbSites; i++ {
		sites[i] = control.NewController(allSiteIDs[i], allSiteIDs)
	}

	fmt.Println("=== Début simulation anneau ===")

	next := func(i int) int {
		return (i + 1) % nbSites
	}

	// Injection initiale : le message NE passe PAS par le sender
	startSender := 0
	entrySite := next(startSender)

	queue := []Msg{
		{
			from: entrySite,
			msg: `ACTION:SC_REQUEST
ID:msg-1
DEST:-1
SENDER:2
STAMP:1
VECT:1,0,0`,
		},
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		currentSite := sites[item.from]

		fmt.Printf("\n===== Message sur site %s =====\n%s\n", currentSite.SiteID, item.msg)

		fmt.Printf("\n--- Site %s traite ---\n", currentSite.SiteID)

		responses := currentSite.HandleIncomingFromNetwork(item.msg)
		fmt.Printf("Site %s a généré %d message(s)\n", currentSite.SiteID, len(responses))

		for _, resp := range responses {
			to := next(item.from)

			fmt.Printf("→ %s envoie vers %s:\n%s\n",
				currentSite.SiteID,
				sites[to].SiteID,
				resp,
			)

			queue = append(queue, Msg{
				from: to,
				msg:  resp,
			})
		}
	}

	fmt.Println("=== Fin simulation anneau ===")
}
