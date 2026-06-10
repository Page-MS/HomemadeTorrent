package networkcontroler

import (
	"HomemadeTorrent/pkg/distributed_file"
	"HomemadeTorrent/pkg/parser"
	"log"
	"sort"
)

// AddUser intègre un nouveau site dans les structures locales.
// isLeader doit être true unqiuement si le site courant est l'élu qui a géré l'ajout.
func (nc *NetworkControler) AddUser(newSiteID string, isLeader bool) string {
	oldDirectory := nc.Controler.NetworkDirectory
	oldVector := nc.Controler.Vector.GetCopy()
	oldTab := nc.Controler.DistFile.GetCopy()

	// Création de la nouvelle liste d'IDs
	newIDs := make([]string, len(oldDirectory.IndexToID))
	copy(newIDs, oldDirectory.IndexToID)
	newIDs = append(newIDs, newSiteID)

	// Tri déterministe pour aligner les index de tout le réseau
	sort.Strings(newIDs)

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

	log.Printf("[ADD_USER] Annuaire mis à jour. Le site '%s' est officiellement reconnu.\n", newSiteID)

	if isLeader {
		// L'élu devient le voisin direct du nouveau site
		nc.NbNeighbors++

		log.Printf("[ADD_USER] ACTION LEADER Le site %s devient mon nouveau voisin direct. Total voisins : %d\n", newSiteID, nc.NbNeighbors)

		// TODO: Envoyer la configuration de base au nouveau site (Registre, liste des pairs, etc.)

		// Création du message de confirmation pour prévenir le reste du réseau
		msg := parser.Message{
			Sender:  nc.SiteID,
			Dest:    BROADCAST,
			Action:  "ADD_USER_CONFIRM",
			Payload: newSiteID,
		}

		encodedMsg, err := parser.Encode(msg)
		if err != nil {
			log.Printf("[ADD_USER] Erreur encodage ADD_USER_CONFIRM : %v\n", err)
			return ""
		}

		return encodedMsg
	}

	// Si on n'est pas le leader, on n'a rien à envoyer sur le réseau
	return ""
}
