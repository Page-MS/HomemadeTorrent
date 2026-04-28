package control

import (
	"log"
	"sort"

	"HomemadeTorrent/pkg/clock"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
)

type SiteDirectory struct {
	IDToIndex map[string]int
	IndexToID []string
}

type Controller struct {
	Lamport          *clock.LamportClock
	Vector           *clock.VectorClock
	DistFile         *DistributedFile
	Reg              *registre.Registre
	SiteID           string          // nom du site
	SiteIndex        int             // index du site
	SeenMessages     map[string]bool // Messages déjà vu par le site
	NetworkDirectory SiteDirectory   // Correspondance SiteId et index
}

// Adapter cette valeur en focntion de la convention choisie
const BROADCAST string = "-1"

// NewController initialise un nouveau dispatcher central
func NewController(siteID string, allSiteIDs []string) *Controller {
	clk := &clock.LamportClock{}
	dir := NewSiteDirectory(allSiteIDs)
	return &Controller{
		Lamport:          clk,
		Vector:           clock.NewVectorClock(len(allSiteIDs), dir.IDToIndex[siteID]),
		DistFile:         GetNewDistributedFile(len(allSiteIDs), dir.IDToIndex[siteID], clk),
		SiteID:           siteID,
		SiteIndex:        dir.IDToIndex[siteID],
		SeenMessages:     make(map[string]bool),
		NetworkDirectory: dir,
	}
}

// Initialise l'index de correspondace entre les SiteID et leurs Index
func NewSiteDirectory(siteIDs []string) SiteDirectory {
	// copie pour éviter effets de bord
	ids := make([]string, len(siteIDs))
	copy(ids, siteIDs)

	// tri déterministe
	sort.Strings(ids)

	idToIndex := make(map[string]int)
	for i, id := range ids {
		idToIndex[id] = i
	}

	return SiteDirectory{
		IDToIndex: idToIndex,
		IndexToID: ids,
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
	if forward {
		responses = append(responses, raw)
	}
	if !processLocal {

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
		log.Printf("[CONTROLLER] Appel file répartie\n")
		returnMsg = c.processDistributedFile(pMsg)

	// snapshot
	// TODO: Remplacer par les constantes des actions de sauvegarde
	case "MARKER":
		log.Printf("[CONTROLLER] Appel snapshot\n")
		returnMsg = c.handleSnapshot(pMsg)

	// logique du torrent
	// TODO: remplacer par les constantes des actions Torrent et logique de gestion
	case "GET_PART", "SEND_PART":
		log.Printf("[CONTROLLER] Appel logique torrent\n")
		c.handleTorrent(pMsg)

	default:
		log.Printf("[CONTROLLER] Action inconnue, ignorée: %s\n", pMsg.Action)
		return responses
	}

	// ---------- Encodage reponse ----------------
	pString, err := parser.Encode(returnMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Pas d'actions -> Pas de message à envoyer")
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
		log.Printf("[CONTROLLER] Conversion message file vers message parser impossible | Message: %s | Erreur: %v\n", returnMsg, err)
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
	return c.NetworkDirectory.IDToIndex[id]
}

// getIdFromSIteIndex fais la correspondance entre nom de site et index
func (c *Controller) getIdFromSIteIndex(index int) string {
	return c.NetworkDirectory.IndexToID[index]
}

func (c *Controller) routeMessage(pMsg parser.Message) (processLocal bool, forward bool) {
	// Cas Message pour soi meme
	if pMsg.Sender == c.SiteID {
		log.Printf("[ROUTAGE] Message envoyé par soi-même, ignoré\n")
		return false, false
	}

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
