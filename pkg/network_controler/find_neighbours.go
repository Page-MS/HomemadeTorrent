package networkcontroler

import (
	"HomemadeTorrent/pkg/parser"
	"log"
	"strconv"
)

// Actions utilisées pour l'initialisation de la recherche de voisins
const (
	INIT_FIND_NEIGHBORS    = "INIT_FIND_NEIGHBORS"
	I_M_NEIGHBOR           = "I_M_NEIGHBOR"
	START_NEIGHBORS_SEARCH = "START_NEIGHBORS_SEARCH"
)

func (nc *NetworkControler) InitFindNeighbors(isLeaver bool) []string {
	var response []string

	if isLeaver {
		msgBroadcast := parser.Message{
			Sender: nc.SiteID,
			Dest:   BROADCAST,
			Action: START_NEIGHBORS_SEARCH,
		}
		encodedBroadcast, err := parser.Encode(msgBroadcast)
		if err != nil {
			log.Printf("[NETWORK CONTROLER][FIND NEIGHBORS] Erreur encodage: %v\n", err)
			return nil
		}
		response = append(response, encodedBroadcast)
	}

	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    BROADCAST_NEIGHBORS,
		Action:  INIT_FIND_NEIGHBORS,
		Payload: strconv.FormatBool(isLeaver),
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[NETWORK CONTROLER][FIND NEIGHBORS] Erreur encodage: %v\n", err)
		return nil
	}
	return append(response, encoded)
}

func (nc *NetworkControler) HandleFindNeighbors(pMsg parser.Message) string {
	log.Printf("[NETWORK CONTROLER][HandleFindNeighbors] INIT_FIND_NEIGHBORS reçu de %s, envoi de notre adresse à ce voisin\n", pMsg.Sender)
	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    pMsg.Sender,
		Action:  I_M_NEIGHBOR,
		Payload: pMsg.Payload,
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[NETWORK CONTROLER][HandleFindNeighbors] Erreur encodage: %v\n", err)
		return ""
	}
	return encoded
}

func (nc *NetworkControler) HandleIMNeighborMessage(pMsg parser.Message) bool {
	voisinID := pMsg.Sender
	voisinAddress := "/tmp/network_fifos/in_" + voisinID
	if nc.NeighborIDsAndAdresses == nil {
		nc.NeighborIDsAndAdresses = make(map[string]string)
	}
	nc.NeighborIDsAndAdresses[voisinID] = voisinAddress
	log.Printf("[NETWORK CONTROLER][HandleIMNeighborMessage] Nouveau voisin ajouté : %s à l'adresse %s\n", voisinID, voisinAddress)
	nc.LogNeighbors()

	if len(nc.NeighborIDsAndAdresses) == nc.NbNeighbors {
		return true
	}
	return false
}

// Juste pour les tests
func (nc *NetworkControler) LogNeighbors() {
	log.Printf("[NETWORK CONTROLER][LogNeighbors] Voisins actuels de %s :\n", nc.SiteID)
	for id, address := range nc.NeighborIDsAndAdresses {
		log.Printf(" - Enfant: %s, Adresse: %s\n", id, address)
	}
}
