package control

import (
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/snapshot"
	"log"
)

// triggerLocalSnapshot effectue l'action de "clic"
func (c *Controller) triggerLocalSnapshot(isInitiator bool) {
	// passage au rouge
	c.Snapshot.MyColor = snapshot.Red
	c.Snapshot.IsInitiator = isInitiator

	// Sauvegarde de l'état local -> copie du registre pour que la snapshot ne change plus
	c.Snapshot.SavedRegister = *c.Reg

	// datation avec horloge vectorielle
	c.Snapshot.SavedVector = c.Vector.GetCopy()

	log.Printf("[SNAPSHOT] Site %s est ROUGE. Bilan: %d | Horloge: %v\n",
		c.SiteID, c.Snapshot.Bilan, c.Snapshot.SavedVector)

	// Si on est l'initiateur, on initialise le comptage
	if isInitiator {
		c.Snapshot.NbEtatsAttendus = len(c.NetworkDirectory.IndexToID) - 1
		c.Snapshot.NbMsgAttendus = c.Snapshot.Bilan
		// On ajoute notre propre état à la collecte
		c.Snapshot.CollectedStates = append(c.Snapshot.CollectedStates, c.Snapshot.SavedRegister)
	} else {
		// Sinon, on envoie notre état et notre bilan
		c.sendStateOnRing()
	}
}

// formatPrepostForInitiator prépare le transfert d'un message prépost
func (c *Controller) formatPrepostForInitiator(pMsg parser.Message) string {
	// Un message prépost est un message envoyé blanc reçu rouge
	prepost := parser.Message{
		Action:  "PREPOST_COLLECT",
		Sender:  c.SiteID,
		Dest:    c.getIdFromSIteIndex(c.getSuccessorIndex()), // forward sur l'anneau
		Payload: pMsg.Action,                                 // contenu du message d'origine
		Color:   string(snapshot.Red),                        // message de controles sont rouges
	}

	res, err := parser.Encode(prepost)
	if err != nil {
		log.Printf("[SNAPSHOT] Erreur encodage prépost: %v\n", err)
		return ""
	}
	return res
}

// sendStateOnRing envoie l'état local et le bilan au successeur
func (c *Controller) sendStateOnRing() {
	stateMsg := parser.Message{
		Action: "STATE_COLLECT",
		Sender: c.SiteID,
		Dest:   c.getIdFromSIteIndex(c.getSuccessorIndex()),
		Bilan:  c.Snapshot.Bilan, // transmet notre bilan à l'initiateur
		Color:  string(snapshot.Red),
		// TODO : serialiser le registre dans le payload
	}

	res, _ := parser.Encode(stateMsg)
	// Simule lenvoi au controlleur
	log.Printf("[SNAPSHOT] État local envoyé au successeur sur l'anneau.\n")
	_ = res
	// TODO : a envoyer au réseau
}

// getSuccessorIndex trouve l'index du site suivant sur l'anneau
func (c *Controller) getSuccessorIndex() int {
	return (c.SiteIndex + 1) % len(c.NetworkDirectory.IndexToID)
}

// Fonction utilitaire pour savoir si le message impacte le bilan pour snapshot
func isApplicationMessage(action string) bool {
	// Les messages Torrent impactent le bilan, les messages de contrôle non
	return action == "GET_PART" || action == "SEND_PART"
}

// finalizeSnapshot conclut l'algorithme de lestage
func (c *Controller) finalizeSnapshot() {
	log.Printf("[SNAPSHOT] TERMINAISON : État global cohérent reconstitué sur %s !", c.SiteID)
	log.Printf("[SNAPSHOT] Heure vectorielle de la sauvegarde : %v", c.Snapshot.SavedVector)

	// Réinitialisation de l'état pour permettre un futur snapshot
	c.Snapshot.IsInitiator = false
	c.Snapshot.MyColor = snapshot.White // site redevient blanc

	// Nettoyage des compteurs
	c.Snapshot.NbEtatsAttendus = 0
	c.Snapshot.NbMsgAttendus = 0

	// TODO : sauvegarder c.Snapshot.CollectedStates dans un fichier JSON par exemple
	log.Println("[SNAPSHOT] Système prêt pour une nouvelle sauvegarde.")
}
