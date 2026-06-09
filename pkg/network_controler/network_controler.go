package networkcontroler

import (
	"HomemadeTorrent/pkg/control"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
	"log"

	"github.com/google/uuid"
)

const BROADCAST = control.BROADCAST

type NetworkControler struct {
	Controler    *control.Controller
	SeenMessages map[string]bool // Messages déjà vu par le site
	SiteID       string
	NbNeighbors  int
	Waves        map[string]*WaveState
}

func NewNetworkControler(siteID string, allSiteIDs []string, r *registre.Registre, nbNeighbors int) *NetworkControler {
	return &NetworkControler{
		Controler:    control.NewController(siteID, allSiteIDs, r),
		SeenMessages: make(map[string]bool),
		SiteID:       siteID,
		NbNeighbors:  nbNeighbors,
		Waves:        make(map[string]*WaveState),
	}
}

func (nc *NetworkControler) HandleIncomingFromNetwork(raw string) []string {
	var responses []string

	// ============= Decodage str -> struct ===============
	pMsg, err := parser.Decode(raw)
	if err != nil {
		log.Printf("[NETWORK CONTROLLER][NETWORK] Erreur decodage: %v\n", err)
		return responses
	}

	// ================ Routage ====================
	if nc.SeenMessages[pMsg.Id] {
		log.Printf("[NETWORK CONTROLER] Message déjà vu, ignoré\n")
		return nil
	}

	processLocal, forward := nc.routeMessage(pMsg)
	nc.SeenMessages[pMsg.Id] = true
	if forward {
		responses = append(responses, raw)
	}
	if !processLocal {
		return responses
	}

	// ============= Logique Network Controler =============
	// Si le message concerne le reseau
	// TODO: vague, arrivé, depart
	switch pMsg.Action {
	case WAVE:
		msg, _ := nc.HandleWave(pMsg)
		responses = append(responses, msg)

	case RETURN_WAVE:
		msg, _ := nc.HandleEcho(pMsg)
		responses = append(responses, msg)
	default:
		// Si le message ne nous concerne pas
		controlerResponse := nc.Controler.HandleIncomingFromNetwork(raw)
		responses = append(responses, controlerResponse...)
	}

	return responses
}

func (nc *NetworkControler) HandleIncomingFromLocal(raw string) []string {
	var responses []string

	// ============= Decodage str -> struct ===============
	pMsg, err := parser.Decode(raw)
	if err != nil {
		log.Printf("[NETWORK CONTROLLER][NETWORK] Erreur decodage: %v\n", err)
		return responses
	}

	// ============= Logique Network Controler =============
	switch pMsg.Action {
	case START_WAVE:
		msg := nc.InitWave(uuid.NewString())
		responses = append(responses, msg)
	default:
		// Si le message ne nous concerne pas
		controlerResponse := nc.Controler.HandleIncomingFromLocal(raw)
		responses = append(responses, controlerResponse...)
	}

	return responses
}

// routeMessage gère le routage
func (nc *NetworkControler) routeMessage(pMsg parser.Message) (processLocal bool, forward bool) {
	if len(pMsg.Sender) == 0 || len(pMsg.Dest) == 0 {
		return false, false
	}

	// Ne jamais re-émettre ses propres messages
	if pMsg.Sender == nc.SiteID {
		log.Printf("[ROUTAGE] Message envoyé par soi-même, ignoré\n")
		return false, false
	}

	// Broadcast : traiter ET re-émettre
	if pMsg.Dest == control.BROADCAST {
		// Exeception si c'est une propagation vague : on intercepte le broadcast
		if pMsg.Action == WAVE {
			log.Printf("[ROUTAGE] Broadcast vague reçu sur site %s, pas de propagation\n", nc.SiteID)
			return true, false
		}
		log.Printf("[ROUTAGE] Broadcast reçu sur site %s\n", nc.SiteID)
		return true, true
	}

	// Message pour ce site : traiter, ne pas re-émettre
	if pMsg.Dest == nc.SiteID {
		log.Printf("[ROUTAGE] Message pour ce site\n")
		return true, false
	}

	// Message pour quelqu'un d'autre : re-émettre sans traiter
	return false, true
}
