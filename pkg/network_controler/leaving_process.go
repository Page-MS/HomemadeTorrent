package networkcontroler

import (
	"HomemadeTorrent/pkg/parser"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	RECEIVE_NODE_LEAVING  = "RECEIVE_NODE_LEAVING"
	START_LEAVING_PROCESS = "START_LEAVING_PROCESS"
	TEST_ENFANTS          = "TEST_ENFANTS"
)

func (nc *NetworkControler) GetChildrenIDsString() string {
	// On fait une string avec les IDs et adresses de nos enfants séparés par des virgules
	string := ""
	for id, address := range nc.NeighborIDsAndAdresses {
		string += id + ":" + address + ","
	}
	return string
}

func (nc *NetworkControler) StartLeavingProcess() string {
	//à l'init
	// On obtient l'adresse de nos enfants BroadcastNeighbors
	// Les voisins nous répondent en faisant suivre leur adresse Fifo à leurs enfants, etc... jusqu'à ce que tous les descendants aient répondu

	// On contacte notre parent pour signaler notre départ et lui donner les adresses de nos enfants(à la réception on ne traite le msg  que si on est parent du site sender)

	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    BROADCAST,
		Action:  RECEIVE_NODE_LEAVING,
		Payload: nc.GetChildrenIDsString(),
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[LEAVING] erreur d'encodage : %v\n", err)
		return ""
	}

	go endProgram() // On quitte le site après avoir envoyé le message à notre parent
	return encoded
}

func endProgram() {
	time.Sleep(10 * time.Second)
	log.Println("[LEAVING] Au revoir !")
	os.Exit(0) // On attend 5 seconde pour que le message soit bien envoyé avant de quitter le site
}

func (nc *NetworkControler) ReceiveLeavingProcess(pMsg parser.Message) {
	// Si  le sender est dans nos enfants on prends en compte sa demande de départ
	estParent := false
	for id := range nc.NeighborIDsAndAdresses {
		if id == pMsg.Sender {
			estParent = true
			log.Printf("[NETWORK CONTROLER][ReceiveLeavingProcess] Notre enfant %s part.\n", pMsg.Sender)
			// On le retire de nos enfants
			delete(nc.NeighborIDsAndAdresses, id)

			break
		}
	}
	if estParent {
		mapNouveauxEnfants := nc.UnparseLeavingProcessPayload(pMsg.Payload)
		// On ajoute les nouveaux enfants à notre liste d'enfants
		for id, address := range mapNouveauxEnfants {
			nc.NeighborIDsAndAdresses[id] = address
			cmd := exec.Command("tee", "-a", nc.NeighborIDsAndAdresses[id])
			err := cmd.Run()

			if err != nil {
				log.Fatal(err)
			}

		}
		log.Printf("[NETWORK CONTROLER][ReceiveLeavingProcess] Nouvelle list d'enfants\n")
		nc.LogNeighbors()
	}
}

func (nc *NetworkControler) UnparseLeavingProcessPayload(payload string) map[string]string {

	mapIds := make(map[string]string)

	// On split la payload pour obtenir les IDs et adresses de nos nouveaux enfants
	parts := strings.Split(payload, ",")
	for _, part := range parts {
		if len(part) == 0 {
			//Pour la fin
			continue
		}
		subParts := strings.Split(part, ":")
		if len(subParts) != 2 {
			log.Printf("[NETWORK CONTROLER][UnparseLeavingProcessPayload] Erreur de format dans la payload de départ d'un site: %s\n", part)
			continue
		}
		id := subParts[0]
		address := subParts[1]
		mapIds[id] = address
	}

	return mapIds
}
