package control

import (
	"HomemadeTorrent/pkg/parser"
	"fmt"
)

func (c *Controller) ParserMessageToFileMessage(pMsg parser.Message) (Message, error) {
	fileMsgType, err := ParseFileMessageType(pMsg.Action)
	if err != nil {
		return Message{}, fmt.Errorf("[MAPPER] Type de message inconnue pour la file répartie: %v\n", err)
	}

	return Message{
		Type:        fileMsgType,
		IndexSender: c.getSiteIndexFromID(pMsg.Id),
		ClockValue:  pMsg.Stamp,
	}, nil
}

func (c *Controller) FileMessageToParserMessage(fMsg Message) (parser.Message, error) {
	return parser.Message{
		Action: string(fMsg.Type),
		Stamp:  fMsg.ClockValue,
		// A completer quand le msg type sera plus clair
	}, nil
}
