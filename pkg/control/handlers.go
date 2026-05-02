package control

import (
	"log"

	"HomemadeTorrent/pkg/distributed_file"
	"HomemadeTorrent/pkg/parser"
)

// handleDistributedFile fait le lien avec distributed_file.go
func (c *Controller) handleDistributedFile(pMsg parser.Message) parser.Message {
	// conversion du message Parser vers message de control interne
	msgCtrl, err := c.ParserMessageToFileMessage(pMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Conversion message parser vers message file impossible: %v\n", err)
		return parser.Message{}
	}

	var responseMsg distributed_file.Message
	var isReady bool

	switch msgCtrl.Type {
	case distributed_file.SC_REQUEST:
		responseMsg, isReady = c.DistFile.SCRequestFromNetwork(msgCtrl)
	case distributed_file.SC_LIBERATION:
		isReady = c.DistFile.SCStopFromNetwork(msgCtrl)
	case distributed_file.ACK:
		isReady = c.DistFile.AckFromNetwork(msgCtrl)
	}

	if isReady {
		log.Printf("[CONTROLLER] >>> SECTION CRITIQUE ACCORDÉE SITE %s\n", c.SiteID)
		// TODO: informer l'app torrent
	}

	returnMsg, err := c.FileMessageToParserMessage(responseMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Conversion message file vers message parser impossible | Message: %+v | Erreur: %v\n", returnMsg, err)
		return parser.Message{}
	}
	return returnMsg
}

func (c *Controller) handleSnapshot(pMsg parser.Message) parser.Message {
	if !c.Snapshot.IsInitiator {
		pMsg.Dest = c.getIdFromSIteIndex(c.getSuccessorIndex())
		return pMsg
	}

	switch pMsg.Action {
	case "STATE_COLLECT":
		c.Snapshot.NbEtatsAttendus--
		c.Snapshot.NbMsgAttendus += pMsg.Bilan
		// c.Snapshot.CollectedStates = append(...)

	case "PREPOST_COLLECT":
		c.Snapshot.NbMsgAttendus--
	}

	// terminaison
	if c.Snapshot.NbEtatsAttendus == 0 && c.Snapshot.NbMsgAttendus == 0 {
		c.finalizeSnapshot()
	}

	return parser.Message{}
}

// TODO: handleTorrent pour les messages de fichiers
func (c *Controller) handleTorrent(pMsg parser.Message) {
	log.Printf("[TORRENT] Traitement de la pièce %d pour l'objet %s", pMsg.Chunk, pMsg.Object)
}
