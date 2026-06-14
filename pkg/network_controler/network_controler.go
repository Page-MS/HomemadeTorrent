package networkcontroler

import (
	"HomemadeTorrent/pkg/control"
	"HomemadeTorrent/pkg/delay"
	"HomemadeTorrent/pkg/distributed_file"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
	torrentlogic "HomemadeTorrent/pkg/torrentLogic"
	"log"
	"strconv"

	"github.com/google/uuid"
)

const BROADCAST = control.BROADCAST
const BROADCAST_NEIGHBORS = "-2"

type NetworkControler struct {
	Controler    *control.Controller
	SeenMessages map[string]bool // Messages déjà vu par le site
	SiteID       string

	NbNeighbors int
	Waves       map[string]*WaveState

	Election  *ElectionState
	ElectedID string

	PeersWaitingToJoin  []string
	PeersWaitingToLeave []string
	CurrentElectionGoal string

	NeighborIDsAndAdresses map[string]string // Map des enfants et leurs adresses (pour gérer le départ)
	NbTeeReceived          int
}

func NewNetworkControler(siteID string, allSiteIDs []string, r *registre.Registre, nbNeighbors int, delay delay.Delay) *NetworkControler {
	return &NetworkControler{
		Controler:              control.NewController(siteID, allSiteIDs, r, &delay),
		SeenMessages:           make(map[string]bool),
		SiteID:                 siteID,
		NbNeighbors:            nbNeighbors,
		Waves:                  make(map[string]*WaveState),
		PeersWaitingToJoin:     make([]string, 0),
		PeersWaitingToLeave:    make([]string, 0),
		NeighborIDsAndAdresses: make(map[string]string),
	}
}

func (nc *NetworkControler) HandleIncomingFromNetwork(raw string) ([]string, bool) {
	var responses []string
	isLeaving := false

	// ============= Decodage str -> struct ===============
	pMsg, err := parser.Decode(raw)
	if err != nil {
		log.Printf("[NETWORK CONTROLLER][NETWORK] Erreur decodage: %v\n", err)
		return responses, isLeaving
	}

	// ================ Routage ====================
	if nc.SeenMessages[pMsg.Id] {
		log.Printf("[NETWORK CONTROLER] Message déjà vu, ignoré (ID: %s)\n", pMsg.Id)
		return nil, isLeaving
	}

	processLocal, forward := nc.routeMessage(pMsg)
	nc.SeenMessages[pMsg.Id] = true
	if forward {
		responses = append(responses, raw)
	}
	if !processLocal {
		return responses, isLeaving
	}

	// ============= Logique Network Controler =============
	// Si le message concerne le reseau
	switch pMsg.Action {
	// ============= ELECTION ==============
	case WAVE:
		msg := nc.HandleWave(pMsg)
		responses = append(responses, msg)
	case RETURN_WAVE:
		msg := nc.HandleEcho(pMsg)
		responses = append(responses, msg)
	case WAVE_ELECTION:
		msg := nc.HandleElectionWave(pMsg)
		responses = append(responses, msg)
	case ECHO_ELECTION:
		msg := nc.HandleElectionEcho(pMsg)
		responses = append(responses, msg...)
	case ELECTED:
		nc.HandleElected(pMsg)
	case RELEASE_ELECTION:
		msg := nc.HandleReleaseElected()
		responses = append(responses, msg)
		// ============= DEPART ==============
	case START_NEIGHBORS_SEARCH:
		msg := nc.InitFindNeighbors(false)
		responses = append(responses, msg...)
	case I_M_NEIGHBOR:
		isFinished := nc.HandleIMNeighborMessage(pMsg)
		isLeaver, _ := strconv.ParseBool(pMsg.Payload)
		if isFinished && isLeaver {
			msg := nc.HandlePeerAskingToLeave(pMsg)
			responses = append(responses, msg)
		}
	case INIT_FIND_NEIGHBORS:
		msg := nc.HandleFindNeighbors(pMsg)
		responses = append(responses, msg)
	case RECEIVE_NODE_LEAVING:
		msg := nc.ReceiveLeavingProcess(pMsg)
		responses = append(responses, msg)
	case PARENT_TEE_UPDATE:
		msg := nc.HandleTeeUpdate(pMsg)
		responses = append(responses, msg)

		if nc.NbTeeReceived == nc.NbNeighbors {
			log.Printf("[LEAVING] ENVOIE SIGNAL ARRET\n")
			isLeaving = true
		}
	case PARENT_TEE_UPDATE_ACK:
		nc.RestartTee()
		// ============= ARRIVEE ==============
	case ASKING_TO_JOIN_NETWORK:
		msg := nc.HandlePeerAskingToJoin(pMsg)
		responses = append(responses, msg)
	case ADD_USER_CONFIRM:
		// Si on est le site ajouté, on doit pas se rajouter nous meme à nous meme
		if pMsg.Payload != nc.SiteID {
			nc.AddUser(pMsg.Payload, false)
		}
	case UPDATE_LISTE:
		nc.UpdateListe(pMsg)

	default:
		// Si le message ne nous concerne pas
		controlerResponse := nc.Controler.HandleIncomingFromNetwork(raw)
		responses = append(responses, controlerResponse...)
	}

	return responses, isLeaving
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
	case START_ELECTION:
		msg := nc.StartElection("")
		responses = append(responses, msg)
	case START_LEAVING_PROCESS:
		msg := nc.InitFindNeighbors(true)
		responses = append(responses, msg...)
	case START_ASKING_TO_JOIN_NETWORK:
		nc.AskPeerToJoinNetwork(pMsg)
	case string(torrentlogic.DoneWithSCLeaving):
		msg := nc.StartLeavingProcess(true)
		responses = append(responses, msg...)

		pMsg.Action = string(distributed_file.SC_LIBERATION)
		raw, _ = parser.Encode(pMsg)
		controlerResponse := nc.Controler.HandleIncomingFromLocal(raw)
		responses = append(responses, controlerResponse...)
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
		log.Printf("[ROUTAGE] Message envoyé par soi-même, ignoré (ID: %s)\n", pMsg.Id)
		return false, false
	}

	// Broadcast : traiter ET re-émettre
	if pMsg.Dest == BROADCAST {
		log.Printf("[ROUTAGE] Broadcast reçu sur site %s\n", nc.SiteID)
		return true, true
	}

	if pMsg.Dest == BROADCAST_NEIGHBORS {
		log.Printf("[ROUTAGE] Broadcast seulement aux voisins reçu sur site %s, pas de propagation\n", nc.SiteID)
		return true, false
	}

	// Message pour ce site : traiter, ne pas re-émettre
	if pMsg.Dest == nc.SiteID {
		log.Printf("[ROUTAGE] Message pour ce site\n")
		return true, false
	}

	// Message pour quelqu'un d'autre : re-émettre sans traiter
	return false, true
}
