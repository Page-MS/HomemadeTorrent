package control

import (
	"HomemadeTorrent/pkg/registre"
	"HomemadeTorrent/pkg/snapshot"
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

	if pMsg.Action == snapshot.MARKER {

		// on est BLANC (Premier Marker reçu)
		if c.Snapshot.MyColor == snapshot.White {
			log.Printf("[SNAPSHOT] Premier MARKER reçu de %s. Clic !", pMsg.Sender)

			// On détermine si on est l'initiateur global (reçu de l'extérieur)
			isGlobalInitiator := pMsg.Sender == "USER"

			// On définit l'ID de l'initiateur (soit nous, soit celui qui nous l'envoie)
			initiatorID := pMsg.Sender
			if isGlobalInitiator {
				initiatorID = c.SiteID
			}

			// Action de Snapshot (Sauvegarde + Envoi de l'état si on n'est pas l'initiateur)
			c.triggerLocalSnapshot(isGlobalInitiator, c.getSiteIndexFromID(initiatorID))

			// On prépare le Marker pour le voisin suivant
			pMsg.Sender = initiatorID // On garde l'ID du vrai initiateur
			pMsg.Dest = c.getIdFromSIteIndex(c.getSuccessorIndex())
			pMsg.Color = string(snapshot.Red)

			return pMsg // On envoie le Marker au suivant
		}

		// On est déjà ROUGE (Le Marker a fait le tour ou arrive par un autre canal)
		if c.Snapshot.MyColor == snapshot.Red {
			if c.Snapshot.IsInitiator {
				log.Printf("[SNAPSHOT] Marker revenu à l'initiateur. Fin.")
				return parser.Message{} // L'initiateur arrête la boucle
			} else {
				log.Printf("[SNAPSHOT] Marker reçu alors que déjà rouge. Propagation simple.")
				pMsg.Dest = c.getIdFromSIteIndex(c.getSuccessorIndex())
				return pMsg // On laisse le Marker finir son tour d'anneau
			}
		}
	}

	switch pMsg.Action {
	case snapshot.STATE_COLLECT:
		c.Snapshot.NbEtatsAttendus--
		c.Snapshot.NbMsgAttendus += pMsg.Bilan
		distantReg := &registre.Registre{}

		// 2. Décoder la string JSON reçue dans le payload
		err := distantReg.FromJSON(pMsg.Payload)
		if err != nil {
			log.Printf("[SNAPSHOT][ERROR] Désérialisation de %s: %v", pMsg.Sender, err)
		}
		remoteState := snapshot.SiteState{
			SiteID:   pMsg.Sender,
			Register: *distantReg,
			Vector:   pMsg.Vect,
		}
		c.Snapshot.CollectedStates = append(c.Snapshot.CollectedStates, remoteState)
		log.Printf("[SNAPSHOT] État reçu de %s (Bilan: %d). Attente de %d messages restants.", pMsg.Sender, pMsg.Bilan, c.Snapshot.NbMsgAttendus)

	case snapshot.PREPOST_COLLECT:
		if c.Snapshot.NbEtatsAttendus > 0 || c.Snapshot.NbMsgAttendus > 0 {
			c.Snapshot.NbMsgAttendus--
			raw, err := parser.Encode(pMsg)
			if err != nil {
				log.Printf("[SNPASHOT][PREPOST_COLLECT] Erreur encodage du message PREPOST_COLLECT: %v\n", err)
			}
			c.Snapshot.CollectedPreposts = append(c.Snapshot.CollectedPreposts, raw)
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

// TODO: handleTorrent pour les messages de fichiers
func (c *Controller) handleTorrent(pMsg parser.Message) {
	log.Printf("[TORRENT] Traitement de la pièce %d pour l'objet %s", pMsg.Chunk, pMsg.Object)
}
