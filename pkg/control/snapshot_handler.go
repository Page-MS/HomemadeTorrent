package control

import (
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
	"HomemadeTorrent/pkg/snapshot"
	torrentlogic "HomemadeTorrent/pkg/torrentLogic"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// triggerLocalSnapshot effectue l'action de "clic"
func (c *Controller) triggerLocalSnapshot(isInitiator bool, initiatorID int) string {
	// passage au rouge
	c.Snapshot.MyColor = snapshot.Red
	c.Snapshot.IsInitiator = isInitiator
	c.Snapshot.InitiatorID = initiatorID

	// Sauvegarde de l'état local -> copie du registre pour que la snapshot ne change plus
	if c.Reg != nil {
		c.Snapshot.SavedRegister = *c.Reg
	} else {
		log.Printf("[SNAPSHOT][WARNING] Initialisation d'un registre vide pour le test")
		c.Reg = registre.NewRegistre()

		// TEST RAPIDE : On ajoute un fichier fictif pour voir s'il apparaît dans le JSON
		testFile := registre.File{Name: "TEST_SNAPSHOT.txt", ID: "1234"}
		c.Reg.Files = append(c.Reg.Files, testFile)

		c.Snapshot.SavedRegister = *c.Reg
		log.Printf("[SNAPSHOT] Registre sauvegardé avec %d fichiers", len(c.Snapshot.SavedRegister.Files))
	}

	// datation avec horloge vectorielle
	c.Snapshot.SavedVector = c.Vector.GetCopy()

	log.Printf("[SNAPSHOT] Site %s est ROUGE. Bilan: %d | Horloge: %v\n",
		c.SiteID, c.Snapshot.Bilan, c.Snapshot.SavedVector)

	// Si on est l'initiateur, on initialise le comptage
	if isInitiator {
		c.Snapshot.Bilan++
		c.Snapshot.NbEtatsAttendus = len(c.NetworkDirectory.IndexToID) - 1
		c.Snapshot.NbMsgAttendus = c.Snapshot.Bilan
		c.Snapshot.CollectedStates = []snapshot.SiteState{}
		c.Snapshot.CollectedPreposts = []string{}
		// On ajoute notre propre état à la collecte
		myState := snapshot.SiteState{
			SiteID:   c.SiteID,
			Register: c.Snapshot.SavedRegister,
			Vector:   c.Snapshot.SavedVector,
		}
		c.Snapshot.CollectedStates = append(c.Snapshot.CollectedStates, myState)
		return ""
	} else {
		// Sinon, on envoie notre état et notre bilan
		return c.sendStateOnRing()
	}
}

// formatPrepostForInitiator prépare le transfert d'un message prépost
func (c *Controller) formatPrepostForInitiator(pMsg parser.Message) string {
	// Un message prépost est un message envoyé blanc reçu rouge
	jsonMsg, err := json.Marshal(pMsg)
	if err != nil {
		log.Printf("[SNPASHOT][PREPOST] Erreur serialisation JSON du prepost: %v\n", err)
	}
	prepost := parser.Message{
		Id:      uuid.New().String(),
		Action:  snapshot.PREPOST_COLLECT,
		Sender:  c.SiteID,
		Dest:    c.getIdFromSIteIndex(c.Snapshot.InitiatorID),
		Payload: string(jsonMsg),      // message d'origine
		Color:   string(snapshot.Red), // message de controles sont rouges
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
	jsonReg, err := c.Reg.ToJSON()
	if err != nil {
		log.Printf("[SNAPSHOT][ERROR] Erreur sur la transformation du regsitre en JSON : %v\n", err)
	}
	stateMsg := parser.Message{
		Id:      uuid.New().String(),
		Action:  snapshot.STATE_COLLECT,
		Sender:  c.SiteID,
		Stamp:   c.Lamport.GetValue(),
		Vect:    c.Vector.GetCopy(),
		Dest:    c.getIdFromSIteIndex(c.Snapshot.InitiatorID),
		Bilan:   c.Snapshot.Bilan, // transmet notre bilan à l'initiateur
		Color:   string(snapshot.Red),
		Payload: jsonReg,
	}

	res, err := parser.Encode(stateMsg)
	if err != nil {
		log.Printf("[SNAPSHOT][ERROR] Erreur encodage message: %v\n", err)
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
	_, exists := torrentMessagesMap[torrentlogic.MessageType(action)]
	return exists
}

// finalizeSnapshot conclut l'algorithme de lestage
func (c *Controller) finalizeSnapshot() parser.Message {
	log.Printf("[SNAPSHOT] TERMINAISON : État global cohérent reconstitué sur %s !", c.SiteID)
	log.Printf("[SNAPSHOT] Heure vectorielle de la sauvegarde : %v", c.Snapshot.SavedVector)

	dirPath := "../../snapshots"

	// on crée le dossier snapshots s'il n'existe pas
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		log.Printf("[ERROR] Impossible de créer le dossier %s: %v", dirPath, err)
	}

	// on définie la structure qui sera exporté
	export := snapshot.GlobalSnapshot{
		SnapshotID:        fmt.Sprintf("snapshot_%s_%d", c.SiteID, time.Now().Unix()),
		CollectedStates:   c.Snapshot.CollectedStates,
		CollectedPreposts: c.Snapshot.CollectedPreposts,
	}

	// on la transforme en JSON
	jsonData, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		log.Printf("[SNAPSHOT][ERROR] Erreur de formatage JSON : %v", err)
	}

	// on écrit le fichier au bon endroit
	fileName := fmt.Sprintf("snapshot_global_%s_%d.json", c.SiteID, time.Now().Unix())
	fullPath := filepath.Join(dirPath, fileName)

	err = os.WriteFile(fullPath, jsonData, 0644)
	if err != nil {
		log.Printf("[SNAPSHOT][ERROR] Erreur d'écriture dans %s: %v", fullPath, err)
	} else {
		log.Printf("[SNAPSHOT] Fichier sauvegardé avec succès dans : %s", fullPath)
	}

	log.Printf("[SNAPSHOT] Sauvegarde réussie dans le fichier : %s", fileName)

	// Réinitialisation de l'état pour permettre un futur snapshot
	c.Snapshot.IsInitiator = false
	c.Snapshot.MyColor = snapshot.White // site redevient blanc

	// Nettoyage des compteurs
	c.Snapshot.NbEtatsAttendus = 0
	c.Snapshot.NbMsgAttendus = 0

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
