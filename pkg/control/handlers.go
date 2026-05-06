package control

import (
	"HomemadeTorrent/pkg/snapshot"
	torrentlogic "HomemadeTorrent/pkg/torrentLogic"
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
	case distributed_file.LOCAL_SC_REQUEST:
		responseMsg = c.DistFile.SCRequestFromBaseApp()
	case distributed_file.LOCAL_SC_LIBERATION:
		responseMsg = c.DistFile.SCStopFromBaseApp()
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

	if pMsg.Action == snapshot.RESET_SNAPSHOT {
		if c.Snapshot.MyColor == snapshot.White {
			log.Printf("[SNAPSHOT] Reset reçu mais site déjà BLANC. Fin de boucle.")
			return parser.Message{Action: ""}
		}

		c.Snapshot.MyColor = snapshot.White
		c.Snapshot.IsInitiator = false
		log.Printf("[SNAPSHOT] Reset reçu de %s : Retour au BLANC.", pMsg.Sender)

		pMsg.Dest = c.getIdFromSIteIndex(c.getSuccessorIndex())
		pMsg.Vect = c.Vector.GetCopy()
		pMsg.Stamp = c.Lamport.GetValue()
		return pMsg
	}

	// Si on reçoit un MARKER et qu'on est blanc, on devient initiateur
	if pMsg.Action == snapshot.MARKER && c.Snapshot.MyColor == snapshot.White {
		log.Printf("[SNAPSHOT] Déclenchement initié par MARKER réseau.")
		c.triggerLocalSnapshot(true)
		pMsg.Sender = c.SiteID
		pMsg.Dest = c.getIdFromSIteIndex(c.getSuccessorIndex())
		pMsg.Color = string(snapshot.Red)
		pMsg.Vect = c.Vector.GetCopy()
		pMsg.Stamp = c.Lamport.GetValue()
		return pMsg
	}

	if !c.Snapshot.IsInitiator {
		pMsg.Dest = c.getIdFromSIteIndex(c.getSuccessorIndex())
		pMsg.Vect = c.Vector.GetCopy()
		pMsg.Stamp = c.Lamport.GetValue()
		return pMsg
	}

	switch pMsg.Action {
	case snapshot.STATE_COLLECT:
		c.Snapshot.NbEtatsAttendus--
		c.Snapshot.NbMsgAttendus += pMsg.Bilan
		// TODO : c.Snapshot.CollectedStates = append(registre serialiser dans payload)
		log.Printf("[SNAPSHOT] État reçu de %s (Bilan: %d). Attente de %d messages restants.", pMsg.Sender, pMsg.Bilan, c.Snapshot.NbMsgAttendus)

	case snapshot.PREPOST_COLLECT:
		if c.Snapshot.NbEtatsAttendus > 0 || c.Snapshot.NbMsgAttendus > 0 {
			c.Snapshot.NbMsgAttendus--
			c.Snapshot.CollectedPreposts = append(c.Snapshot.CollectedPreposts, pMsg.Payload)
			log.Printf("[SNAPSHOT] Message en vol archivé. Restant : %d", c.Snapshot.NbMsgAttendus)

		} else {
			log.Printf("[SNAPSHOT] Prepost reçu hors session, ignoré.")
		}
		return parser.Message{Action: ""}
	}

	// terminaison
	if c.Snapshot.NbEtatsAttendus == 0 && c.Snapshot.NbMsgAttendus == 0 {
		resetMsg := c.finalizeSnapshot()
		return resetMsg
	}

	return parser.Message{}
}

// handleTorrent pour les messages de fichiers
func (c *Controller) handleTorrent(pMsg parser.Message) parser.Message {
	// Si action en lien avec la file répartie alors pas besoin de check le payload
	switch pMsg.Action {
	case string(torrentlogic.AskingFromSC):
		log.Printf("[CONTROLLER] Redirection vers file repartie")
		pMsg.Action = string(distributed_file.LOCAL_SC_REQUEST)
		return c.handleDistributedFile(pMsg)
	case string(torrentlogic.DoneWithSC):
		log.Printf("[CONTROLLER] Redirection vers file repartie")
		pMsg.Action = string(distributed_file.LOCAL_SC_LIBERATION)
		return c.handleDistributedFile(pMsg)
	}

	// conversion du message Controle vers message torrent
	msgTorrent, err := c.ParserMessageToTorrentMessage(pMsg)
	if err != nil {
		log.Printf("[CONTROLLER] Conversion message controler vers message torrent impossible: %v\n", err)
		return parser.Message{}
	}

	switch pMsg.Action {
	case string(torrentlogic.TransferRelatedMessage):
		{
			input, exist := c.InputTorrentTransfers[msgTorrent.TransferID]
			if !exist {
				inputChan := make(chan torrentlogic.Message, 100)
				go torrentlogic.StartOutgoingTransfer(msgTorrent.TransferID, msgTorrent.FileID, c.SiteID, c.Register, inputChan, c.OutputTorrentChan)
			}
			input <- msgTorrent
			return parser.Message{}
		}

	// TODO: voir avec Page que faire lors de la reception des messages de type AskingForShasum et AskingForContent
	case string(torrentlogic.AskingForShasum):
	case string(torrentlogic.AskingForContent):
	}

	return parser.Message{}
}
