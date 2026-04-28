package main

import (
	"fmt"
	"log"

	"HomemadeTorrent/pkg/control"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	nbSites := 3

	// Création des sites
	sites := make([]*control.Controller, nbSites)
	for i := 0; i < nbSites; i++ {
		siteID := fmt.Sprintf("%d", i)
		sites[i] = control.NewController(nbSites, i, siteID)
	}

	// Message initial simulé (site 0 -> broadcast)
	rawMsg := `ACTION:SC_REQUEST
ID:msg-1
DEST:-1
SENDER:0
STAMP:1
VECT:1,0,0`

	fmt.Println("=== Début simulation ===")

	// File de messages à traiter
	queue := []string{rawMsg}

	// Simulation
	for len(queue) > 0 {
		msg := queue[0]
		queue = queue[1:]

		fmt.Printf("\n===== Nouveau message dans le réseau =====\n%s\n", msg)

		for i, site := range sites {
			fmt.Printf("\n--- Site %d reçoit ---\n", i)

			responses := site.HandleIncoming(msg)

			// Ajouter toutes les réponses à la queue
			for _, resp := range responses {
				fmt.Printf("→ Site %d génère:\n%s\n", i, resp)
				queue = append(queue, resp)
			}
		}
	}

	fmt.Println("=== Fin simulation ===")
}
