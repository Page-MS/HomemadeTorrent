package control

import (
	"log"
	"strconv"
	"strings"

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
		log.Printf("[CONTROLLER] Erreur décodage message: %v", err)
		return "", false
	}

	// synchro des horloges
	c.Lamport.Update(pMsg.Stamp)
	if len(pMsg.Vector) > 0 {
		c.Vector.Update(pMsg.Vector)
	}

	log.Printf("[CONTROLLER] Action: %s | de: %s | Lamport: %d", pMsg.Action, pMsg.Id, c.Lamport.GetValue())

	// routage
	switch pMsg.Action {

	// exclusion mutuelle
	case string(SC_REQUEST), string(SC_LIBERATION), string(ACK):
		return c.processDistributedFile(pMsg)

	// snapshot
	case "MARKER":
		return c.handleSnapshot(pMsg)

	// logique du torrent
	case "GET_PART", "SEND_PART":
		return c.handleTorrent(pMsg)

	default:
		log.Printf("[CONTROLLER] Action inconnue, ignorée: %s", pMsg.Action)
		return "", false
	}
}

// processDistributedFile fait le lien avec distributed_file.go
func (c *Controller) processDistributedFile(pMsg parser.Message) (string, bool) {
	// conversion du message Parser vers message de control interne
	msgCtrl := Message{
		Type:        MessageType(pMsg.Action),
		IndexSender: c.getSiteIndexFromID(pMsg.Id),
		ClockValue:  pMsg.Stamp,
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
		log.Println("[CONTROLLER] >>> SECTION CRITIQUE ACCORDÉE")

		// TODO : faire les actions adaptés en fonctions du type de message sur le registre

		// Modif finie donc on sort de SC
		liberationMsg := c.DistFile.SCStopFromBaseApp()
		return c.formatResponseFromAlgo(liberationMsg), true
	}

	if responseMsg.Type != "" {
		return c.formatResponseFromAlgo(responseMsg), (responseMsg.IndexDest == -1)
	}

	return "", false
}

// TODO : handleSnapshot qui appelera le package de snapshot
func (c *Controller) handleSnapshot(pMsg parser.Message) (string, bool) {
	log.Printf("[SNAPSHOT] Déclenchement via marker de %s", pMsg.Id)

	return "", false
}

// handleTorrent pour les messages de fichiers
func (c *Controller) handleTorrent(pMsg parser.Message) (string, bool) {
	log.Printf("[TORRENT] Traitement de la pièce %d pour l'objet %s", pMsg.Chunk, pMsg.Object)
	return "", false
}

// buildRawString prépare une réponse compatible avec le parser
// TODO : a virer quand le parser sera fini
func (c *Controller) buildRawString(action string, stamp int) string {
	var sb strings.Builder
	sb.WriteString("ACTION:" + action + "\n")
	sb.WriteString("ID:" + c.SiteID + "\n")
	sb.WriteString("STAMP:" + strconv.Itoa(stamp) + "\n")
	sb.WriteString("VECTOR:" + c.vectorToString(c.Vector.GetCopy()) + "\n")
	return sb.String()
}

// formatResponseFromAlgo utilise la clock déjà calculée par lalgo de file répartie
func (c *Controller) formatResponseFromAlgo(m Message) string {
	return c.buildRawString(string(m.Type), m.ClockValue) // TODO a remplacer par la fonction encode() du parser
}

// formatNewRequest pour les messages initiés par le controlleur
func (c *Controller) formatNewRequest(action string) string {
	c.Lamport.Tick()
	c.Vector.Tick()
	return c.buildRawString(action, c.Lamport.GetValue()) // TODO a remplacer par la fonction encode() du parser
}

// vectorToString convertie vector en string
func (c *Controller) vectorToString(v []int) string {
	strs := make([]string, len(v))
	for i, val := range v {
		strs[i] = strconv.Itoa(val)
	}
	return strings.Join(strs, ",")
}

// getSiteIndexFromID fais la correspondance entre nom de site et index
func (c *Controller) getSiteIndexFromID(id string) int {
	// TODO: Implémenter une vraie table de correspondance
	return 0
}
