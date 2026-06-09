package networkcontroler

import (
	"HomemadeTorrent/pkg/parser"
	"log"
	"strings"
)

// Actions utilisées pour la diffusion en vague
const (
	WAVE        = "WAVE"
	RETURN_WAVE = "RETURN_WAVE"
	START_WAVE  = "START_WAVE"
)

// WaveState contient l'état local de l'algorithme de vague pour une vague donnée.
type WaveState struct {
	WaveID      string // ID unique de cette vague (= ID du message initiateur)
	Initiator   string // SiteID de l'initiateur
	Parent      string // Voisin dont on a reçu le premier WAVE ("" si on est l'initiateur)
	EchoPending int    // Nombre d'échos encore attendus
	Done        bool   // Vague terminée (écho final reçu par l'initiateur)
}

// InitWave est appelé quand CE site veut démarrer une nouvelle vague.
// Retourne le message Vague à envoyer en broadcast vers tous les voisins.
func (nc *NetworkControler) InitWave(waveID string) string {
	_, exists := nc.Waves[waveID]
	if exists {
		log.Printf("[WAVE] Vague %s déjà initialisée\n", waveID)
		return ""
	}

	state := &WaveState{
		WaveID:      waveID,
		Initiator:   nc.SiteID,
		Parent:      "", // pas de parent : on est l'initiateur
		EchoPending: nc.NbNeighbors,
		Done:        false,
	}
	nc.Waves[waveID] = state

	log.Printf("[WAVE] Site %s initie la vague %s vers %d voisins\n", nc.SiteID, waveID, state.EchoPending)

	return nc.buildWaveMessages(waveID, nc.SiteID, state.Parent)
}

// HandleWave est appelé quand on reçoit un message d'action WAVE.
// Retourne :
//   - le message à réémettre (nouveau WAVE vers les voisins, ou ECHO vers le parent)
func (nc *NetworkControler) HandleWave(pMsg parser.Message) string {
	sender := pMsg.Sender
	parts := strings.Split(pMsg.Payload, ",")
	initiator := parts[0]
	parent := parts[1]
	waveID := parts[2]

	state, exists := nc.Waves[waveID]
	if !exists {
		// Première fois qu'on voit cette vague : on s'initialise
		state = &WaveState{
			WaveID:    waveID,
			Initiator: initiator,
			Parent:    sender,
			Done:      false,
		}

		// On attend les échos de tous les voisins sauf le parent
		state.EchoPending = nc.NbNeighbors - 1
		nc.Waves[waveID] = state

		log.Printf("[WAVE] Site %s reçoit vague %s pour la première fois (parent=%s, pending=%d)\n", nc.SiteID, waveID, sender, state.EchoPending)

		// Cas feuille : aucun voisin à qui propager → écho immédiat
		if state.EchoPending == 0 {
			log.Printf("[WAVE] Site %s est une feuille, écho immédiat vers %s\n", nc.SiteID, sender)
			delete(nc.Waves, waveID)
			return nc.buildEchoMessages(waveID, sender)
		}

		// Propager la Wave à tous les voisins
		return nc.buildWaveMessages(waveID, initiator, state.Parent)

	} else if parent == nc.SiteID {
		// Mon descendant me repropage ma vague
		log.Printf("[WAVE] Propagation vague venant du descendant %s, ignoré\n", pMsg.Sender)
		return ""
	}

	// On a déjà vu cette vague : un de mes voisins a reçut la vague autrement que part moi
	log.Printf("[WAVE] Site %s a déjà traité la vague %s, pending - 1 : %d\n", nc.SiteID, waveID, state.EchoPending-1)
	state.EchoPending -= 1
	// Si aucun voisin à qui propager → écho
	if state.EchoPending == 0 {
		log.Printf("[WAVE] Site %s n'a plus de voisins à qui propager, écho vers %s\n", nc.SiteID, state.Parent)
		delete(nc.Waves, waveID)
		return nc.buildEchoMessages(waveID, state.Parent)
	}
	return ""
}

// HandleEcho est appelé quand on reçoit un message d'action ECHO.
// Retourne le message à envoyer (écho vers le parent, ou signal de fin si initiateur).
func (nc *NetworkControler) HandleEcho(pMsg parser.Message) string {
	waveID := pMsg.Payload

	state, exists := nc.Waves[waveID]
	if !exists {
		log.Printf("[ECHO] Reçu écho pour vague inconnue %s, ignoré\n", waveID)
		return ""
	}
	if state.Done {
		log.Printf("[ECHO] Vague %s déjà terminée, écho ignoré\n", waveID)
		return ""
	}

	state.EchoPending -= 1
	log.Printf("[ECHO] Site %s reçoit écho pour vague %s (restant=%d)\n", nc.SiteID, waveID, state.EchoPending)

	if state.EchoPending > 0 {
		// On attend encore d'autres échos
		return ""
	}

	// Tous les échos reçus
	state.Done = true

	if state.Parent == "" {
		// On est l'initiateur : la vague est terminée globalement
		log.Printf("[ECHO] Site %s (initiateur) : vague %s TERMINÉE\n", nc.SiteID, waveID)
		delete(nc.Waves, waveID)
		nc.onWaveComplete(waveID)
		return ""
	}

	// On renvoie l'écho à notre parent
	log.Printf("[ECHO] Site %s renvoie écho vers parent %s pour vague %s\n", nc.SiteID, state.Parent, waveID)
	return nc.buildEchoMessages(waveID, state.Parent)
}

// onWaveComplete est le callback déclenché sur l'initiateur quand la vague est terminée.
func (nc *NetworkControler) onWaveComplete(waveID string) {
	log.Printf("[WAVE] Vague %s complète sur le réseau entier\n", waveID)
	// TODO: déclencher l'action post-vague (ex: début du téléchargement, snapshot, election…)
}

// ─── Helpers de construction de messages ─────────────────────────────────────

func (nc *NetworkControler) buildWaveMessages(waveID string, initiator string, parent string) string {
	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    BROADCAST_NEIGHBORS,
		Action:  WAVE,
		Payload: initiator + "," + parent + "," + waveID,
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[WAVE] Erreur encodage WAVE : %v\n", err)
		return ""
	}
	return encoded
}

func (nc *NetworkControler) buildEchoMessages(waveID string, parent string) string {
	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    parent,
		Action:  RETURN_WAVE,
		Payload: waveID,
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[WAVE] Erreur encodage ECHO vers %s : %v\n", parent, err)
		return ""
	}
	return encoded
}
