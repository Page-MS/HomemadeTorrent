package control

import (
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
	"HomemadeTorrent/pkg/snapshot"
	"log"

	"github.com/google/uuid"
)

// triggerLocalSnapshot effectue l'action de "clic"
func (c *Controller) triggerLocalSnapshot(isInitiator bool) string {
	// passage au rouge
	c.Snapshot.MyColor = snapshot.Red
	c.Snapshot.IsInitiator = isInitiator

	// Sauvegarde de l'état local -> copie du registre pour que la snapshot ne change plus
	if c.Reg != nil {
		c.Snapshot.SavedRegister = *c.Reg
	} else {
		log.Printf("[WARNING] Registre inexistant lors du snapshot !")
		c.Snapshot.SavedRegister = registre.Registre{}
	}

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
		return ""
	} else {
		// Sinon, on envoie notre état et notre bilan
		return c.sendStateOnRing()
	}
}

// formatPrepostForInitiator prépare le transfert d'un message prépost
func (c *Controller) formatPrepostForInitiator(pMsg parser.Message) string {
	// Un message prépost est un message envoyé blanc reçu rouge
	prepost := parser.Message{
		Id:      uuid.New().String(),
		Action:  snapshot.PREPOST_COLLECT,
		Sender:  c.SiteID,
		Dest:    c.getIdFromSIteIndex(c.getSuccessorIndex()),                     // forward sur l'anneau
		Payload: pMsg.Sender + " a envoyé " + pMsg.Action + " : " + pMsg.Payload, // contenu du message d'origine
		Color:   string(snapshot.Red),                                            // message de controles sont rouges
		Stamp:   c.Lamport.GetValue(),
		Vect:    c.Vector.GetCopy(),
	}

	res, err := parser.Encode(prepost)
	if err != nil {
		log.Printf("[SNAPSHOT] Erreur encodage prépost: %v\n", err)
		return ""
	}
	return res
}

// sendStateOnRing envoie l'état local et le bilan au successeur
func (c *Controller) sendStateOnRing() string {
	log.Printf("[DEBUG-SEND] Mon bilan actuel au moment de l'envoi : %d", c.Snapshot.Bilan)
	stateMsg := parser.Message{
		Id:      uuid.New().String(),
		Action:  snapshot.STATE_COLLECT,
		Sender:  c.SiteID,
		Stamp:   c.Lamport.GetValue(),
		Vect:    c.Vector.GetCopy(),
		Dest:    c.getIdFromSIteIndex(c.getSuccessorIndex()),
		Bilan:   c.Snapshot.Bilan, // transmet notre bilan à l'initiateur
		Color:   string(snapshot.Red),
		Payload: "Registre Site " + c.SiteID,
		// TODO : serialiser le registre dans le payload
	}

	res, err := parser.Encode(stateMsg)
	if err != nil {
		log.Printf("[ERROR] Erreur encodage message: %v\n", err)
		return ""
	}

	log.Printf("[SNAPSHOT] État local préparé pour le successeur (Bilan: %d).\n", c.Snapshot.Bilan)
	return res
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
func (c *Controller) finalizeSnapshot() parser.Message {
	log.Printf("[SNAPSHOT] TERMINAISON : État global cohérent reconstitué sur %s !", c.SiteID)
	log.Printf("[SNAPSHOT] Heure vectorielle de la sauvegarde : %v", c.Snapshot.SavedVector)

	// Réinitialisation de l'état pour permettre un futur snapshot
	c.Snapshot.IsInitiator = false
	c.Snapshot.MyColor = snapshot.White // site redevient blanc

	// Nettoyage des compteurs
	c.Snapshot.NbEtatsAttendus = 0
	c.Snapshot.NbMsgAttendus = 0

	// TODO : sauvegarder c.Snapshot.CollectedStates et c.Snapshot.CollectedPreposts dans un fichier JSON par exemple

	resetMsg := parser.Message{
		Action: snapshot.RESET_SNAPSHOT,
		Id:     uuid.New().String(),
		Sender: c.SiteID,
		Stamp:  c.Lamport.GetValue(),
		Vect:   c.Vector.GetCopy(),
		Dest:   c.getIdFromSIteIndex(c.getSuccessorIndex()),
	}
	log.Println("[SNAPSHOT] Système prêt pour une nouvelle sauvegarde.")
	return resetMsg
}
