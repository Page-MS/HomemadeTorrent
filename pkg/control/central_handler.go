package control

import (
	"HomemadeTorrent/pkg/snapshot"
	torrentlogic "HomemadeTorrent/pkg/torrentLogic"
	"log"
	"sort"

	"HomemadeTorrent/pkg/clock"
	"HomemadeTorrent/pkg/distributed_file"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
)

type SiteDirectory struct {
	IDToIndex map[string]int
	IndexToID []string
}

type Controller struct {
	Lamport               *clock.LamportClock
	Vector                *clock.VectorClock
	DistFile              *distributed_file.DistributedFile
	Reg                   *registre.Registre
	SiteID                string        // nom du site
	SiteIndex             int           // index du site
	NetworkDirectory      SiteDirectory // Correspondance SiteId et index
	Snapshot              *snapshot.Snapshot
	InputTorrentTransfers map[string]chan torrentlogic.Message // Map des inputs des transfers torrent en cour
	OutputTorrentChan     chan torrentlogic.Message
	TorrentScChan         chan torrentlogic.Message
}

const BROADCAST string = "-1"

var torrentMessagesMap = map[torrentlogic.MessageType]struct{}{
	torrentlogic.TransferRelatedMessage: {},
	torrentlogic.AskingForShasum:        {},
	torrentlogic.AskingForContent:       {},
}

// NewController initialise un nouveau dispatcher central
func NewController(siteID string, allSiteIDs []string, r *registre.Registre) *Controller {
	clk := &clock.LamportClock{}
	dir := NewSiteDirectory(allSiteIDs)

	return &Controller{
		Lamport:          clk,
		Vector:           clock.NewVectorClock(len(allSiteIDs), dir.IDToIndex[siteID]),
		DistFile:         distributed_file.GetNewDistributedFile(len(allSiteIDs), dir.IDToIndex[siteID], clk),
		SiteID:           siteID,
		SiteIndex:        dir.IDToIndex[siteID],
		NetworkDirectory: dir,
		Snapshot: &snapshot.Snapshot{
			MyColor:     snapshot.White,
			Bilan:       0,
			IsInitiator: false,
		},
		InputTorrentTransfers: make(map[string]chan torrentlogic.Message),
		OutputTorrentChan:     make(chan torrentlogic.Message, 100), // Goulot d'étranglement sur la capacité d'envoi (augmenter si besoin)
		TorrentScChan:         make(chan torrentlogic.Message),
		Reg:                   r,
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

// HandleIncomingFromNetwork s'occupe de recevoir les message texte venant du réseau, synchronise les horloges et fait le routage.
func (c *Controller) HandleIncomingFromNetwork(raw string) []string {
	var responses []string

	// ============= Decodage str -> struct ===============
	pMsg, err := parser.Decode(raw)
	if err != nil {
		log.Printf("[CONTROLLER][NETWORK] Erreur decodage: %v\n", err)
		return responses
	}

	log.Printf("[CONTROLLER][NETWORK] Message reçut site %s | Sender: %s | Dest: %s\n", c.SiteID, pMsg.Sender, pMsg.Dest)

	// ============= synchro des horloges ==============
	c.Lamport.Update(pMsg.Stamp)
	if len(pMsg.Vect) > 0 {
		c.Vector.Update(pMsg.Vect)
	}

	//=============== Logique Snapshot =================

	// Chaque réception diminue le bilan local.
	// On ne décrémente le bilan que pour les messages torrent pas pour les autres messages
	if isApplicationMessage(pMsg.Action) {
		log.Printf("[SNAPSHOT] Message applicatif reçu pour traitement local. Bilan--")
		c.Snapshot.Bilan--
	}

	// Si on reçoit rouge alors qu'on est blanc on peut notre instantané avant de traiter le message.
	if pMsg.Color == string(snapshot.Red) && c.Snapshot.MyColor == snapshot.White && pMsg.Action == snapshot.MARKER {
		log.Printf("[SNAPSHOT] Lestage détecté (Msg ROUGE sur Site BLANC). Clic forcé.")
		initiatorID := pMsg.Sender

		msgSnapshot := c.triggerLocalSnapshot(false, initiatorID)
		if msgSnapshot != "" {
			responses = append(responses, msgSnapshot)
		}
	}

	// Détection des messages Prépost : Envoyé blanc, reçu rouge
	if pMsg.Color == string(snapshot.White) && c.Snapshot.MyColor == snapshot.Red && isApplicationMessage(pMsg.Action) {
		// si on est initiateur pas besoin de créer un message
		var responseMsg string
		if c.Snapshot.IsInitiator {
			log.Printf("[SNAPSHOT] Message Prépost identifié. Ajout à la liste.")
			responseMsg = c.addPrepostToSnapshot(pMsg)

		} else {
			log.Printf("[SNAPSHOT] Message Prépost identifié. Envoi à l'initiateur.")
			responseMsg = c.formatPrepostForInitiator(pMsg)
		}
		// On crée un message de contrôle pour envoyer ce contenu à l'initiateur
		responses = append(responses, responseMsg)
	}

	// ==================== Logique dispatcher ======================

	log.Printf("[CONTROLLER][NETWORK] Action: %s | de: %s | Lamport: %d\n", pMsg.Action, pMsg.Sender, c.Lamport.GetValue())

	// Redirection vers le service aproprié
	var returnMsg parser.Message
	switch pMsg.Action {

	// file répartie
	case string(distributed_file.SC_REQUEST), string(distributed_file.SC_LIBERATION), string(distributed_file.ACK):
		log.Printf("[CONTROLLER][NETWORK] Appel file répartie\n")
		returnMsg = c.handleDistributedFile(pMsg)

	// snapshot
	case snapshot.MARKER, snapshot.PREPOST_COLLECT, snapshot.STATE_COLLECT, snapshot.RESET_SNAPSHOT:
		log.Printf("[CONTROLLER][NETWORK] Appel snapshot\n")
		returnMsg = c.handleSnapshot(pMsg)

	// logique du torrent
	case string(torrentlogic.TransferRelatedMessage), string(torrentlogic.AskingForContent), string(torrentlogic.AskingForShasum), string(torrentlogic.RegisterUpdate):
		log.Printf("[CONTROLLER][NETWORK] Appel logique torrent\n")
		c.handleTorrent(pMsg)

	default:
		log.Printf("[CONTROLLER][NETWORK] Action inconnue, ignorée: %s\n", pMsg.Action)
		return responses
	}

	// ============= Encodage reponse =================
	// Si action vide alors c'est un message vide -> pas besoin d'encoder
	if returnMsg.Action == "" {
		return responses
	}
	pString, err := parser.Encode(returnMsg)
	if err != nil {
		log.Printf("[CONTROLLER][NETWORK] Erreur encodage pour réseau: %v\n", err)
		return responses
	}

	return append(responses, pString)
}

// HandleIncomingFromLocal gère la reception des messages venant du UI et de la couche applicative (donc pas de routage)
func (c *Controller) HandleIncomingFromLocal(raw string) []string {
	var responses []string
	// ============ Decodage ======================
	pMsg, err := parser.Decode(raw)
	if err != nil {
		log.Printf("[CONTROLLER][LOCAL] Erreur décodage commande locale: %v\n", err)
		return nil
	}

	// ============== Horloges =================
	c.Lamport.Tick()
	c.Vector.Tick()

	// ============ LOGIQUE SNAPSHOT ======================
	//Maj du bilan
	if isApplicationMessage(pMsg.Action) {
		c.Snapshot.Bilan++
		log.Printf("[SNAPSHOT][LOCAL] Création d'un message applicatif | Bilan: %d\n", c.Snapshot.Bilan)
	}
	// Maj couleur
	pMsg.Color = string(c.Snapshot.MyColor)

	// ========== REDIRECTION HANDLERS ===================
	var returnMsg parser.Message
	switch pMsg.Action {
	case string(distributed_file.LOCAL_SC_REQUEST), string(distributed_file.LOCAL_SC_LIBERATION):
		log.Printf("[CONTROLLER][LOCAL] Appel file répartie\n")
		returnMsg = c.handleDistributedFile(pMsg)
	case snapshot.MARKER:
		log.Printf("[CONTROLLER][LOCAL] Appel snapshot\n")
		returnMsg = c.handleSnapshot(pMsg)
	case string(torrentlogic.AskingFromSC), string(torrentlogic.DoneWithSC), string(torrentlogic.StartTransfers):
		log.Printf("[CONTROLLER][LOCAL] Appel logique torrent\n")
		returnMsg = c.handleTorrent(pMsg)
	default:
		if isApplicationMessage(pMsg.Action) || pMsg.Action == string(torrentlogic.RegisterUpdate) {
			// Message destiné à l'extérieur
			returnMsg = pMsg
		} else {
			log.Printf("[CONTROLLER][LOCAL] Action inconnue, ignorée: %s\n", pMsg.Action)
			return nil
		}
	}

	if len(returnMsg.Vect) == 0 {
		returnMsg.Vect = make([]int, len(c.NetworkDirectory.IndexToID))
	}

	// ================= Encodage ===============
	// Si pas d'actions alors pas besoin d'encoder
	if returnMsg.Action == "" {
		return responses
	}
	encodedMsg, err := parser.Encode(returnMsg)
	if err != nil {
		log.Printf("[CONTROLLER][LOCAL] Erreur encodage pour réseau: %v\n", err)
		return nil
	}

	responses = append(responses, encodedMsg)
	return responses
}

// getSiteIndexFromID fais la correspondance entre nom de site et index
func (c *Controller) getSiteIndexFromID(id string) int {
	return c.NetworkDirectory.IDToIndex[id]
}

// getIdFromSIteIndex fais la correspondance entre nom de site et index
func (c *Controller) getIdFromSIteIndex(index int) string {
	if index == -1 { // Index de broadcast
		return BROADCAST
	}
	return c.NetworkDirectory.IndexToID[index]
}
