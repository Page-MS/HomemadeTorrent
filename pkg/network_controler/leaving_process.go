package networkcontroler

import (
	"HomemadeTorrent/pkg/parser"
	"log"
)

func (nc *NetworkControler) GetChildrenIDsString() string {
	return ""
}

func (nc *NetworkControler) StartLeavingProcess() string {
	//à l'init
	// On obtient l'adresse de nos enfants BroadcastNeighbors
	// Les voisins nous répondent en faisant suivre leur adresse Fifo à leurs enfants, etc... jusqu'à ce que tous les descendants aient répondu
	//on attends autant de réponses que nbNeighbors

	// On contacte notre parent pour signaler notre départ et lui donnner les adresses de nos enfants(à la réception on ne traite le msg  que si on est parent du site sender)

	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    BROADCAST,
		Action:  WAVE_ELECTION,
		Payload: "LEAVING_PROCESS," + nc.SiteID + "," + nc.GetChildrenIDsString(),
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[ELECTION] Erreur encodage Wave : %v\n", err)
		return ""
	}
	return encoded
}
