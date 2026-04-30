package control

import (
	"HomemadeTorrent/pkg/distributed_file"
	"HomemadeTorrent/pkg/parser"
	"fmt"
)

func (c *Controller) ParserMessageToFileMessage(pMsg parser.Message) (distributed_file.Message, error) {
	fileMsgType, err := distributed_file.ParseFileMessageType(pMsg.Action)
	if err != nil {
		return distributed_file.Message{}, fmt.Errorf("[MAPPER] Type de message inconnu pour la file répartie: %v\n", err)
	}

	return distributed_file.Message{
		Type:        fileMsgType,
		IndexSender: c.getSiteIndexFromID(pMsg.Id),
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
