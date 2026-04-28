package control

import (
	"log"

	"HomemadeTorrent/pkg/clock"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
)

type Controller struct {
	Lamport   *clock.LamportClock
	Vector    *clock.VectorClock
	DistFile  *DistributedFile
	Reg       *registre.Registre
	SiteID    string // nom du site
	SiteIndex int    // index du site
}

// Adapter cette valeur en focntion de la convention choisie
const BROADCAST string = "-1"

// NewController initialise un nouveau dispatcher central
func NewController(nbSites int, siteIndex int, siteID string) *Controller {
	clk := &clock.LamportClock{}
	return &Controller{
		Lamport:   clk,
		Vector:    clock.NewVectorClock(nbSites, siteIndex),
		DistFile:  GetNewDistributedFile(nbSites, siteIndex, clk),
		SiteID:    siteID,
		SiteIndex: siteIndex,
	}
}

// HandleIncoming s'occupe de recevoir les message texte, synchronise les horloges et fait le routage.
func (c *Controller) HandleIncoming(raw string) (response string, toBroadcast bool) {
	pMsg, err := parser.Decode(raw)
	if err != nil {
		log.Printf("[CONTROLLER] Erreur décodage message: %v\n", err)
		return "", false
	}

	log.Printf("[CONTROLLER] Message reçut site %s | Sender: %s | Dest: %s\n", c.SiteID, pMsg.Sender, pMsg.Dest)

	// synchro des horloges
	c.Lamport.Update(pMsg.Stamp)
	if len(pMsg.Vect) > 0 {
		c.Vector.Update(pMsg.Vect)
	}

	// Verification que le message est pour nous sinon on le renvoie au suivant
	if pMsg.Dest != BROADCAST && pMsg.Dest != c.SiteID {
		log.Printf("[CONTROLLER] Message pas pour ce site => renvoie au suivant\n")
		return raw, (pMsg.Dest == BROADCAST)
	}

	// TODO: cas du broadcast -> renvoie a la fois le message recut et la réponse du site
	// Besoin de modifier le parser pour créer différents messages à l'encodage en focntion des \n

	log.Printf("[CONTROLLER] Action: %s | de: %s | Lamport: %d\n", pMsg.Action, pMsg.Sender, c.Lamport.GetValue())

	// Redirection vers le service aproprié
	var returnMsg parser.Message
	isBroadcast := false
	switch pMsg.Action {

	// exclusion mutuelle
	case string(SC_REQUEST), string(SC_LIBERATION), string(ACK):
		returnMsg, isBroadcast = c.processDistributedFile(pMsg)

	// snapshot
	// TODO: Remplacer par les constantes des actions de sauvegarde
	case "MARKER":
		returnMsg, isBroadcast = c.handleSnapshot(pMsg)

	// logique du torrent
	// TODO: remplacer par les constantes des actions Torrent et logique de gestion
	case "GET_PART", "SEND_PART":
		c.handleTorrent(pMsg)

	default:
		log.Printf("[CONTROLLER] Action inconnue, ignorée: %s\n", pMsg.Action)
		return "", false
	}

	pString, err := parser.Encode(returnMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Erreur encodage reponse: %v\n", err)
		return "", false
	}

	return pString, isBroadcast
}

// processDistributedFile fait le lien avec distributed_file.go
func (c *Controller) processDistributedFile(pMsg parser.Message) (parser.Message, bool) {
	// conversion du message Parser vers message de control interne
	msgCtrl, err := c.ParserMessageToFileMessage(pMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Conversion message parser vers message file impossible: %v\n", err)
		return parser.Message{}, false
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
		return parser.Message{}, false
	}
	return returnMsg, (responseMsg.IndexDest == -1)
}

// TODO : handleSnapshot qui appelera le package de snapshot
func (c *Controller) handleSnapshot(pMsg parser.Message) (parser.Message, bool) {
	log.Printf("[SNAPSHOT] Déclenchement via marker de %s", pMsg.Id)

	return parser.Message{}, false
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
