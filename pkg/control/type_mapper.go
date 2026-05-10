package control

import (
	"HomemadeTorrent/pkg/distributed_file"
	"HomemadeTorrent/pkg/parser"
	torrentlogic "HomemadeTorrent/pkg/torrentLogic"
	"fmt"
)

func (c *Controller) ParserMessageToFileMessage(pMsg parser.Message) (distributed_file.Message, error) {
	fileMsgType, err := distributed_file.ParseFileMessageType(pMsg.Action)
	if err != nil {
		return distributed_file.Message{}, fmt.Errorf("[MAPPER] Type de message inconnu pour la file répartie: %v\n", err)
	}

	return distributed_file.Message{
		Type:        fileMsgType,
		IndexSender: c.getSiteIndexFromID(pMsg.Sender),
		ClockValue:  pMsg.Stamp,
	}, nil
}

func (c *Controller) FileMessageToParserMessage(fMsg distributed_file.Message) (parser.Message, error) {
	return parser.Message{
		Action: string(fMsg.Type),
		Stamp:  fMsg.ClockValue,
		Vect:   c.Vector.GetCopy(),
		Dest:   c.getIdFromSIteIndex(fMsg.IndexDest),
		Sender: c.getIdFromSIteIndex(fMsg.IndexSender),
	}, nil
}

func (c *Controller) ParserMessageToTorrentMessage(pMsg parser.Message) (torrentlogic.Message, error) {
	if pMsg.Payload == "" {
		return torrentlogic.Message{}, fmt.Errorf("[MAPPER] Payload vide, impossible de convertir en un message torrent\n")
	}

	if pMsg.Action == string(torrentlogic.StartTransfers) {
		msgTorrent, err := parser.DecodeStartTransferPayload(pMsg.Payload)
		if err != nil {
			return torrentlogic.Message{}, fmt.Errorf("[MAPPER] Impossible de décoder le payload StartTransfers: %v\n", err)
		}
		return msgTorrent, nil
	}

	torrentMsgType, err := parser.DecodeTorrentPayload(pMsg.Payload)
	if err != nil {
		return torrentlogic.Message{}, fmt.Errorf("[MAPPER] Impossible de décoder le payload torrent: %v\n", err)
	}
	return torrentMsgType, nil
}

func (c *Controller) TorrentMessageToParserMessage(tMsg torrentlogic.Message) (parser.Message, error) {
	payload, err := parser.EncodeTorrentPayload(tMsg)
	if err != nil {
		return parser.Message{}, fmt.Errorf("[MAPPER] Impossible d'encoder le payload torrent: %v", err)
	}

	return parser.Message{
		Action:  string(tMsg.MessageType),
		Stamp:   c.Lamport.GetValue(),
		Vect:    c.Vector.GetCopy(),
		Dest:    tMsg.TargetID,
		Sender:  c.SiteID,
		Payload: payload,
	}, nil
}
