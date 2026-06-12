package networkcontroler

import (
	"HomemadeTorrent/pkg/distributed_file"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
	"log"
	"strings"
)

const (
	ADD_USER_CONFIRM = "ADD_USER_CONFIRM"
	UPDATE_REGISTRE  = "UPDATE_REGISTRE"
	UPDATE_LISTE     = "UPDATE_LISTE"
)

// AddUser intègre un nouveau site dans les structures locales.
// isLeader doit être true unqiuement si le site courant est l'élu qui a géré l'ajout.
func (nc *NetworkControler) AddUser(newSiteID string, isLeader bool) []string {
	oldDirectory := nc.Controler.NetworkDirectory
	oldVector := nc.Controler.Vector.GetCopy()
	oldTab := nc.Controler.DistFile.GetCopy()

	// Création de la nouvelle liste d'IDs
	newIDs := make([]string, len(oldDirectory.IndexToID))
	copy(newIDs, oldDirectory.IndexToID)
	newIDs = append(newIDs, newSiteID)

	// Reconstitution de la map d'indexation
	newIDToIndex := make(map[string]int)
	for i, id := range newIDs {
		newIDToIndex[id] = i
	}

	// Réalignement des anciennes valeurs de l'horloge vectorielle
	newVector := make([]int, len(newIDs))
	for _, id := range oldDirectory.IndexToID {
		oldIdx := oldDirectory.IDToIndex[id]
		newIdx := newIDToIndex[id]
		newVector[newIdx] = oldVector[oldIdx]
	}

	newTab := make([]distributed_file.TabEntry, len(newIDs))

	// On initialise toutes les cases par défaut (comme dans GetNewDistributedFile)
	for i := range newTab {
		newTab[i] = distributed_file.TabEntry{
			Type: distributed_file.SC_LIBERATION,
			Date: 0,
		}
	}

	// On replace les anciennes valeurs aux bons index
	for _, id := range oldDirectory.IndexToID {
		oldIdx := oldDirectory.IDToIndex[id]
		newIdx := newIDToIndex[id]
		newTab[newIdx] = oldTab[oldIdx]
	}

	// Injection des nouvelles structures
	newMyIndex := newIDToIndex[nc.SiteID]
	nc.Controler.Vector.UpdateLayout(newVector, newMyIndex)
	nc.Controler.DistFile.UpdateLayout(newTab, newMyIndex)

	nc.Controler.NetworkDirectory.IndexToID = newIDs
	nc.Controler.NetworkDirectory.IDToIndex = newIDToIndex
	nc.Controler.SiteIndex = newMyIndex
	//nc.Controler.Reg.AddNewUserToRegister(newSiteID, newSiteID)

	log.Printf("[ADD_USER] Annuaire mis à jour. Le site '%s' est officiellement reconnu.\n", newSiteID)

	if isLeader {
		// L'élu devient le voisin direct du nouveau site
		nc.NbNeighbors++

		log.Printf("[ADD_USER] ACTION LEADER Le site %s devient mon nouveau voisin direct. Total voisins : %d\n", newSiteID, nc.NbNeighbors)
		var responses []string

		// TODO: Envoyer la configuration de base au nouveau site (Registre, liste des pairs)
		jsonReg, err := nc.Controler.Reg.ToJSON()
		if err != nil {
			log.Printf("[ADD_USER] Erreur serialisation registre : %v\n", err)
			return nil
		}
		msgRegistre := parser.Message{
			Sender:  nc.SiteID,
			Dest:    newSiteID,
			Action:  UPDATE_REGISTRE,
			Payload: jsonReg,
		}
		encodedMsg, err := parser.Encode(msgRegistre)
		if err != nil {
			log.Printf("[ADD_USER] Erreur encodage UPDATE_REGISTRE : %v\n", err)
			return nil
		}
		responses = append(responses, encodedMsg)

		msgListe := parser.Message{
			Sender:  nc.SiteID,
			Dest:    newSiteID,
			Action:  UPDATE_LISTE,
			Payload: strings.Join(nc.Controler.NetworkDirectory.IndexToID, ","),
		}
		encodedMsg, err = parser.Encode(msgListe)
		if err != nil {
			log.Printf("[ADD_USER] Erreur encodage UPDATE_LISTE : %v\n", err)
			return nil
		}
		responses = append(responses, encodedMsg)

		// Création du message de confirmation pour prévenir le reste du réseau
		msg := parser.Message{
			Sender:  nc.SiteID,
			Dest:    BROADCAST,
			Action:  ADD_USER_CONFIRM,
			Payload: newSiteID,
		}

		encodedMsg, err = parser.Encode(msg)
		if err != nil {
			log.Printf("[ADD_USER] Erreur encodage ADD_USER_CONFIRM : %v\n", err)
			return nil
		}

		return append(responses, encodedMsg)
	}

	// Si on n'est pas le leader, on n'a rien à envoyer sur le réseau
	return nil
}

func (nc *NetworkControler) UpdateRegistre(pMsg parser.Message) {
	reg := &registre.Registre{}
	err := reg.FromJSON(pMsg.Payload)
	if err != nil {
		log.Printf("[UPDATE_REGISTRE] Erreur deserialisatuion du registre : %v\n", err)
		return
	}
	nc.Controler.Reg.Merge(reg)
}

func (nc *NetworkControler) UpdateListe(pMsg parser.Message) {
	peersList := strings.Split(pMsg.Payload, ",")

	// Création de la nouvelle liste d'IDs
	nc.Controler.NetworkDirectory.IndexToID = peersList

	// Reconstitution de la map d'indexation
	newIDToIndex := make(map[string]int)
	for i, id := range peersList {
		newIDToIndex[id] = i
	}
	nc.Controler.NetworkDirectory.IDToIndex = newIDToIndex

	// Horloge vectorielle
	newVector := make([]int, len(peersList))
	nc.Controler.Vector.UpdateLayout(newVector, newIDToIndex[nc.SiteID])

	// File repartie
	newTab := make([]distributed_file.TabEntry, len(peersList))
	// On initialise toutes les cases par défaut (comme dans GetNewDistributedFile)
	for i := range newTab {
		newTab[i] = distributed_file.TabEntry{
			Type: distributed_file.SC_LIBERATION,
			Date: 0,
		}
	}
	nc.Controler.DistFile.UpdateLayout(newTab, newIDToIndex[nc.SiteID])

	nc.Controler.SiteIndex = newIDToIndex[nc.SiteID]
}
