package control

import (
	"log"

	"HomemadeTorrent/pkg/clock"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
)

type Controller struct {
	Lamport      *clock.LamportClock
	Vector       *clock.VectorClock
	DistFile     *DistributedFile
	Reg          *registre.Registre
	SiteID       string // nom du site
	SiteIndex    int    // index du site
	SeenMessages map[string]bool
}

// Adapter cette valeur en focntion de la convention choisie
const BROADCAST string = "-1"

// NewController initialise un nouveau dispatcher central
func NewController(nbSites int, siteIndex int, siteID string) *Controller {
	clk := &clock.LamportClock{}
	return &Controller{
		Lamport:      clk,
		Vector:       clock.NewVectorClock(nbSites, siteIndex),
		DistFile:     GetNewDistributedFile(nbSites, siteIndex, clk),
		SiteID:       siteID,
		SiteIndex:    siteIndex,
		SeenMessages: make(map[string]bool),
	}
}

// HandleIncoming s'occupe de recevoir les message texte, synchronise les horloges et fait le routage.
func (c *Controller) HandleIncoming(raw string) []string {
	var responses []string

	// -------------- Decodage ------------------
	pMsg, err := parser.Decode(raw)
	if err != nil {
		log.Printf("[CONTROLLER] Erreur décodage message: %v\n", err)
		return responses
	}

	log.Printf("[CONTROLLER] Message reçut site %s | Sender: %s | Dest: %s\n", c.SiteID, pMsg.Sender, pMsg.Dest)

	// -------------- Routage ------------------------
	// Eviter la duplication des messages causé par BROADCAST
	if c.SeenMessages[pMsg.Id] {
		log.Printf("[ROUTAGE] Message déjà vu (%s), ignoré", pMsg.Id)
		return responses
	}
	c.SeenMessages[pMsg.Id] = true
	// verifier si le message est pour ce site
	processLocal, forward := c.routeMessage(pMsg)
	if !processLocal {
		if forward {
			return append(responses, raw)
		}
		return responses
	}

	// ------------- Logique controler --------------
	// synchro des horloges
	c.Lamport.Update(pMsg.Stamp)
	if len(pMsg.Vect) > 0 {
		c.Vector.Update(pMsg.Vect)
	}

	log.Printf("[CONTROLLER] Action: %s | de: %s | Lamport: %d\n", pMsg.Action, pMsg.Sender, c.Lamport.GetValue())

	// Redirection vers le service aproprié
	var returnMsg parser.Message
	switch pMsg.Action {

	// exclusion mutuelle
	case string(SC_REQUEST), string(SC_LIBERATION), string(ACK):
		returnMsg = c.processDistributedFile(pMsg)

	// snapshot
	// TODO: Remplacer par les constantes des actions de sauvegarde
	case "MARKER":
		returnMsg = c.handleSnapshot(pMsg)

	// logique du torrent
	// TODO: remplacer par les constantes des actions Torrent et logique de gestion
	case "GET_PART", "SEND_PART":
		c.handleTorrent(pMsg)

	default:
		log.Printf("[CONTROLLER] Action inconnue, ignorée: %s\n", pMsg.Action)
		return responses
	}

	// ---------- Encodage reponse ----------------
	pString, err := parser.Encode(returnMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Erreur encodage reponse: %v\n", err)
		return responses
	}

	return append(responses, pString)
}

// processDistributedFile fait le lien avec distributed_file.go
func (c *Controller) processDistributedFile(pMsg parser.Message) parser.Message {
	// conversion du message Parser vers message de control interne
	msgCtrl, err := c.ParserMessageToFileMessage(pMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Conversion message parser vers message file impossible: %v\n", err)
		return parser.Message{}
	}

	var responseMsg Message
	var isReady bool

	switch msgCtrl.Type {
	case SC_REQUEST:
		responseMsg, isReady = c.DistFile.SCRequestFromNetwork(msgCtrl)
	case SC_LIBERATION:
		isReady = c.DistFile.SCStopFromNetwork(msgCtrl)
	case ACK:
		isReady = c.DistFile.AckFromNetwork(msgCtrl)
	}

	if isReady {
		log.Printf("[CONTROLLER] >>> SECTION CRITIQUE ACCORDÉE SITE %s\n", c.SiteID)
		// TODO: informer l'app torrent
	}

	returnMsg, err := c.FileMessageToParserMessage(responseMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Conversion message file vers message parser impossible: %v\n", err)
		return parser.Message{}
	}
	return returnMsg
}

// TODO : handleSnapshot qui appelera le package de snapshot
func (c *Controller) handleSnapshot(pMsg parser.Message) parser.Message {
	log.Printf("[SNAPSHOT] Déclenchement via marker de %s", pMsg.Id)

	return parser.Message{}
}

// handleTorrent pour les messages de fichiers
func (c *Controller) handleTorrent(pMsg parser.Message) {
	log.Printf("[TORRENT] Traitement de la pièce %d pour l'objet %s", pMsg.Chunk, pMsg.Object)
}

// getSiteIndexFromID fais la correspondance entre nom de site et index
func (c *Controller) getSiteIndexFromID(id string) int {
	// TODO: Implémenter une vraie table de correspondance
	return 0
}

// getIdFromSIteIndex fais la correspondance entre nom de site et index
func (c *Controller) getIdFromSIteIndex(index int) string {
	// TODO: Implémenter une vraie table de correspondance
	return "0"
}

func (c *Controller) routeMessage(pMsg parser.Message) (processLocal bool, forward bool) {
	// Cas broadcast
	if pMsg.Dest == BROADCAST {
		log.Printf("[ROUTAGE] Broadcast reçu sur site %s", c.SiteID)
		return true, true
	}

	// Cas message pour ce site
	if pMsg.Dest == c.SiteID {
		log.Printf("[ROUTAGE] Message pour ce site (%s)", c.SiteID)
		return true, false
	}

	// Sinon → forward uniquement
	log.Printf("[ROUTAGE] Message pour %s, forward depuis %s", pMsg.Dest, c.SiteID)
	return false, true
}
