package networkcontroler

import (
	"HomemadeTorrent/pkg/control"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
	"log"
)

type NetworkControler struct {
	Controler    *control.Controller
	SeenMessages map[string]bool // Messages déjà vu par le site
	SiteID       string
}

func NewNetworkControler(siteID string, allSiteIDs []string, r *registre.Registre) *NetworkControler {
	return &NetworkControler{
		Controler:    control.NewController(siteID, allSiteIDs, r),
		SeenMessages: make(map[string]bool),
		SiteID:       siteID,
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

	// ================ Routage anneau ====================
	if nc.SeenMessages[pMsg.Id] {
		log.Printf("[NETWORK CONTROLER] Message déjà vu, ignoré\n")
		return nil
	}

	processLocal, forward := nc.routeMessage(pMsg)
	if processLocal {
		nc.SeenMessages[pMsg.Id] = true
	}
	if forward {
		responses = append(responses, raw)
	}
	if !processLocal {
		return responses
	}

	// ============= Logique Network Controler =============
	// Si le message concerne le reseau
	// TODO: vague, arrivé, depart

	// Si le message ne nous concerne pas (le default du switch)
	controlerResponse := nc.Controler.HandleIncomingFromNetwork(raw)

	return append(responses, controlerResponse...)
}

func (nc *NetworkControler) HandleIncomingFromLocal(raw string) []string {
	// Le message ne nous concerne forcement pas
	return nc.Controler.HandleIncomingFromLocal(raw)
}

// routeMessage gère le routage d'un anneau
func (nc *NetworkControler) routeMessage(pMsg parser.Message) (processLocal bool, forward bool) {
	// Vérifier que les informations pour le routage sont présentes
	if len(pMsg.Sender) == 0 || len(pMsg.Dest) == 0 {
		log.Printf("[ROUTAGE] Impossible de router ce message incomplet (pas de destinataire ou d'envoyeur), ignoré\n")
		return false, false
	}

	// Cas Message pour soi meme
	if pMsg.Sender == nc.SiteID {
		log.Printf("[ROUTAGE] Message envoyé par soi-même, ignoré\n")
		return false, false
	}

	// Cas broadcast
	if pMsg.Dest == control.BROADCAST {
		log.Printf("[ROUTAGE] Broadcast reçu sur site %s", nc.SiteID)
		return true, true
	}

	// Cas message pour ce site
	if pMsg.Dest == nc.SiteID {
		log.Printf("[ROUTAGE] Message pour ce site (%s)", nc.SiteID)
		return true, false
	}

	// Sinon → forward uniquement
	log.Printf("[ROUTAGE] Message pour %s, forward depuis %s", pMsg.Dest, nc.SiteID)
	return false, true
}
