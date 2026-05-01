package control

import (
	"HomemadeTorrent/pkg/snapshot"
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
	Lamport          *clock.LamportClock
	Vector           *clock.VectorClock
	DistFile         *distributed_file.DistributedFile
	Reg              *registre.Registre
	SiteID           string          // nom du site
	SiteIndex        int             // index du site
	SeenMessages     map[string]bool // Messages déjà vu par le site
	NetworkDirectory SiteDirectory   // Correspondance SiteId et index
	Snapshot         *snapshot.Snapshot
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
		DistFile:         distributed_file.GetNewDistributedFile(len(allSiteIDs), dir.IDToIndex[siteID], clk),
		SiteID:           siteID,
		SiteIndex:        dir.IDToIndex[siteID],
		SeenMessages:     make(map[string]bool),
		NetworkDirectory: dir,
		Snapshot: &snapshot.Snapshot{
			MyColor:     snapshot.White,
			Bilan:       0,
			IsInitiator: false,
		},
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

// HandleIncomingFromNetwork s'occupe de recevoir les message texte, synchronise les horloges et fait le routage.
func (c *Controller) HandleIncomingFromNetwork(raw string) []string {
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

	// ------------- 3. Logique Snapshot (Lestage & Bilan) --------------

	// Chaque réception diminue le bilan local.
	// On ne décrémente le bilan que pour les messages torrent pas pour les autres messages
	if isApplicationMessage(pMsg.Action) {
		c.Snapshot.Bilan--
	}

	// Si on reçoit rouge alors qu'on est blanc on peut notre instantané avant de traiter le message.
	if pMsg.Color == "rouge" && c.Snapshot.MyColor == "blanc" {
		log.Printf("[SNAPSHOT] Lestage détecté (Msg ROUGE sur Site BLANC). Clic forcé.")
		c.triggerLocalSnapshot(false) // snapshot locale
	}

	// Détection des messages Prépost : Envoyé blanc, reçu rouge
	if pMsg.Color == "blanc" && c.Snapshot.MyColor == "rouge" {
		log.Printf("[SNAPSHOT] Message Prépost identifié. Envoi à l'initiateur.")
		// On crée un message de contrôle pour envoyer ce contenu à l'initiateur
		prepostMsg := c.formatPrepostForInitiator(pMsg)
		responses = append(responses, prepostMsg)
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
	case string(distributed_file.SC_REQUEST), string(distributed_file.SC_LIBERATION), string(distributed_file.ACK):
		log.Printf("[CONTROLLER] Appel file répartie\n")
		returnMsg = c.handleDistributedFile(pMsg)

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

// TODO: HandleIncomingFromLocal gère les demande venant de l'app Torrent
func (c *Controller) HandleIncomingFromLocal(raw string) []string {
	var responses []string
	pMsg, err := parser.Decode(raw)
	if err != nil {
		log.Printf("[LOCAL] Erreur décodage commande locale: %v\n", err)
		return nil
	}

	//Maj du bilan
	if isApplicationMessage(pMsg.Action) {
		c.Snapshot.Bilan++
	}

	// Maj couleur
	pMsg.Color = string(c.Snapshot.MyColor)

	pMsg.Sender = c.SiteID
	encodedMsg, err := parser.Encode(pMsg)
	if err != nil {
		log.Printf("[LOCAL] Erreur encodage pour réseau: %v\n", err)
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
