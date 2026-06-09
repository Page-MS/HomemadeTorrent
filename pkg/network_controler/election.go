package networkcontroler

import (
	"HomemadeTorrent/pkg/parser"
	"log"
	"strings"
)

const (
	START_ELECTION = "START_ELECTION"
	ELECTION_WAVE  = "ELECTION_WAVE"
	WAVE_ELECTION  = "WAVE_ELECTION" // message bleu
	ECHO_ELECTION  = "ECHO_ELECTION" // message rouge
	ELECTED        = "ELECTED"
)

// ElectionState représente l'état local de l'élection par extinction de vagues.
// Un seul état par site (pas par vague) : on écrase quand une meilleure vague arrive.
type ElectionState struct {
	EluID      string // Meilleur candidat connu (plus petit ID gagne)
	Parent     string // Voisin dont on a reçu la vague en cours ("" si on est candidat)
	NbAttendus int    // Nombre de messages rouges encore attendus
}

// StartElection démarre une élection : ce site se déclare candidat.
func (nc *NetworkControler) StartElection() string {
	if nc.Election != nil && nc.Election.EluID <= nc.SiteID {
		log.Printf("[ELECTION] Une meilleure élection est déjà en cours (élu courant=%s)\n", nc.Election.EluID)
		return ""
	}
	if nc.ElectedID != "" {
		log.Printf("[ELECTION] L'election est toujours detenue par l'élu %s\n", nc.ElectedID)
		return ""
	}

	nc.Election = &ElectionState{
		EluID:      nc.SiteID,
		Parent:     nc.SiteID, // on est l'initiateur
		NbAttendus: nc.NbNeighbors,
	}

	log.Printf("[ELECTION] Candidature, envoi vague bleue élu=%s\n", nc.SiteID)
	return nc.buildElectionWave(nc.SiteID, nc.Election.Parent)
}

// HandleElectionWave traite un message bleu (WAVE_ELECTION).
func (nc *NetworkControler) HandleElectionWave(pMsg parser.Message) string {
	parts := strings.Split(pMsg.Payload, ",")
	receivedElu := parts[0] // ID de l'élu porté par la vague bleue
	parent := parts[1]

	log.Printf("[ELECTION] Vague bleue reçue élu=%s de %s\n", receivedElu, pMsg.Sender)

	if nc.Election == nil || nc.Election.EluID > receivedElu {
		// Première vague reçue, ou meilleure vague (plus petit ID) : on adopte
		log.Printf("[ELECTION] Adoption de la vague élu=%s (ancienne=%v)\n", receivedElu, eluOuNil(nc.Election))
		nc.Election = &ElectionState{
			EluID:      receivedElu,
			Parent:     pMsg.Sender,
			NbAttendus: nc.NbNeighbors - 1,
		}

		// Feuille : renvoyer immédiatement un message rouge vers le parent
		if nc.Election.NbAttendus == 0 {
			log.Printf("[ELECTION] Feuille, écho rouge immédiat vers %s\n", pMsg.Sender)
			return nc.buildElectionEcho(receivedElu, pMsg.Sender)
		}

		// Propager la vague bleue aux autres voisins
		return nc.buildElectionWave(receivedElu, nc.Election.Parent)

	} else if nc.Election.EluID == receivedElu {
		if parent == nc.SiteID {
			// Mon descendant me repropage mon election
			log.Printf("[ELECTION] Propagation election venant du descendant %s, ignoré\n", pMsg.Sender)
			return ""
		}

		// Même vague, mais on est déjà au courant :
		log.Printf("[ELECTION] Site %s a déjà traité vague elu=%s, pending - 1 : %d\n", nc.SiteID, receivedElu, nc.Election.NbAttendus-1)
		nc.Election.NbAttendus -= 1
		// Si aucun voisin à qui propager → écho
		if nc.Election.NbAttendus == 0 {
			log.Printf("[ELECTION] Site %s n'a plus de voisins à qui propager, écho vers %s\n", nc.SiteID, nc.Election.Parent)
			return nc.buildElectionEcho(receivedElu, nc.Election.Parent)
		}
		return ""
	}

	// Notre vague courante est meilleure (plus petit ID) : on ignore
	log.Printf("[ELECTION] Vague ignorée élu=%s > élu courant=%s\n", receivedElu, nc.Election.EluID)
	return ""
}

// HandleElectionEcho traite un message rouge (ECHO_ELECTION).
func (nc *NetworkControler) HandleElectionEcho(pMsg parser.Message) string {
	receivedElu := pMsg.Payload

	if nc.Election == nil || nc.Election.EluID != receivedElu {
		// Message rouge d'une vague qu'on a abandonnée : ignorer
		log.Printf("[ELECTION] Rouge ignoré élu=%s (élu courant=%v)\n", receivedElu, eluOuNil(nc.Election))
		return ""
	}

	nc.Election.NbAttendus--
	log.Printf("[ELECTION] Rouge reçu élu=%s (restant=%d)\n", receivedElu, nc.Election.NbAttendus)

	if nc.Election.NbAttendus > 0 {
		return ""
	}

	// Tous les rouges reçus
	if nc.Election.EluID == nc.SiteID {
		// On est l'élu
		log.Printf("[ELECTION] *** ÉLU = %s ***\n", nc.SiteID)
		nc.ElectedID = nc.SiteID
		return nc.buildElectedBroadcast()
	}

	// Renvoyer le rouge vers notre parent
	log.Printf("[ELECTION] Rouge remonté vers parent %s élu=%s\n", nc.Election.Parent, receivedElu)
	return nc.buildElectionEcho(receivedElu, nc.Election.Parent)
}

// HandleElected traite la proclamation broadcast de l'élu.
func (nc *NetworkControler) HandleElected(pMsg parser.Message) {
	nc.ElectedID = pMsg.Payload
	nc.Election = nil
	log.Printf("[ELECTION] Élu proclamé : %s\n", nc.ElectedID)
}

// HandleReleaseElected traite la libération broadcast de l'élection.
func (nc *NetworkControler) HandleReleaseElected(pMsg parser.Message) {
	nc.ElectedID = ""
	log.Printf("[ELECTION] Libération, nouvelle election possible\n")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (nc *NetworkControler) buildElectionWave(eluID string, parent string) string {
	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    BROADCAST_NEIGHBORS,
		Action:  WAVE_ELECTION,
		Payload: eluID + "," + parent,
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[ELECTION] Erreur encodage Wave : %v\n", err)
		return ""
	}
	return encoded
}

func (nc *NetworkControler) buildElectionEcho(eluID string, dest string) string {
	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    dest,
		Action:  ECHO_ELECTION,
		Payload: eluID,
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[ELECTION] Erreur encodage ECHO : %v\n", err)
		return ""
	}
	return encoded
}

func (nc *NetworkControler) buildElectedBroadcast() string {
	msg := parser.Message{
		Sender:  nc.SiteID,
		Dest:    BROADCAST,
		Action:  ELECTED,
		Payload: nc.SiteID,
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[ELECTION] Erreur encodage broadcast : %v\n", err)
		return ""
	}
	return encoded
}

func eluOuNil(e *ElectionState) string {
	if e == nil {
		return "<nil>"
	}
	return e.EluID
}
