package parser

import (
	torrentlogic "HomemadeTorrent/pkg/torrentLogic"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Message = struct {
	Action  string
	Id      string
	Stamp   int
	Vect    []int
	Dest    string
	Sender  string
	Payload string
	// pour snapshot
	Color string
	Bilan int
}

// ACTION:qlksdjfqmlsdfjmqsdlkf
// ID:slkdjfmqldskjf
// OBJECT:sldkfjsdlkf
// CHUNK:1
// SENDER:3
// DEST:1
// STAMP:123
// VECT:123,231,344
// PAYLOAD_LEN:123
// <payload_qlksdjfmlqksdjflkqsd>

// Mandatory fields: ACTION,ID
// if no PAYLOAD_LEN, then no payload
// if len(payload) == 0, then no payload & payload_len is send
// chunk is zero indexed, -1 indicates no chunk
// dest is one indexed, 0/-1 indicates broadcast

// string -> Message, \n is the sep
func Decode(raw_data string) (Message, error) {
	lines := strings.Split(raw_data, "\n")
	msg := Message{}

	for _, l := range lines {
		if l == "\n" || l == "" {
			continue
		}
		parts := strings.Split(l, ":")
		if parts[0] == "\n" || parts[0] == "" {
			continue
		}
		if len(parts) != 2 {
			return Message{}, errors.New("Message line must have exactly 2 component. Found: " + strings.Join(parts, " "))
		}
		key := parts[0]
		value := strings.TrimSpace(parts[1])

		switch key {
		case "ACTION":
			{
				msg.Action = value
			}
		case "ID":
			{
				msg.Id = value
			}

		case "DEST":
			{
				msg.Dest = value
			}

		case "SENDER":
			{
				msg.Sender = value
			}

		case "STAMP":
			{
				val, err := strconv.Atoi(value)
				if err != nil {
					log.Printf("[PARSER] Erreur: %v\n", err)
					return Message{}, errors.New("Impossible to convert STAMP value")
				}
				msg.Stamp = val
			}

		case "VECT":
			msg.Vect = make([]int, 0)
			for _, val := range strings.Split(value, ",") {
				trimmed := strings.TrimSpace(val)
				if trimmed == "" {
					continue
				}
				nb, err := strconv.Atoi(trimmed)
				if err != nil {
					log.Printf("[PARSER] Warning: Vect conversion failed for '%s'", trimmed)
					continue
				}
				msg.Vect = append(msg.Vect, nb)
			}

		case "COLOR":
			msg.Color = value

		case "BILAN":
			val, err := strconv.Atoi(value)
			if err != nil {
				return Message{}, errors.New("Impossible to BILAN")
			}
			msg.Bilan = val
		default:
			{
				return Message{}, errors.New("Found unknown field: " + key)
			}
		}
	}
	return msg, nil
}

func Encode(msg Message) (string, error) {
	data := make([]string, 0, 10)

	if msg.Action == "" {
		return "", errors.New("Empty action")
	} else {
		data = append(data, "ACTION:"+msg.Action)
	}

	if msg.Id == "" {
		msg.Id = uuid.New().String()
	}
	data = append(data, "ID:"+msg.Id)

	if msg.Dest == "" {
		return "", errors.New("Empty DEST")
	}
	if msg.Sender == "" {
		return "", errors.New("Empty SENDER")
	}
	data = append(data, "DEST:"+msg.Dest)
	data = append(data, "SENDER:"+msg.Sender)
	data = append(data, "STAMP:"+strconv.Itoa(msg.Stamp))

	str := make([]string, 0, 2)
	for _, v := range msg.Vect {
		str = append(str, strconv.Itoa(v))
	}
	data = append(data, "VECT:"+strings.Join(str, ","))

	payload_len := len(msg.Payload)
	if payload_len > 0 {
		data = append(data, "PAYLOAD_LEN:"+strconv.Itoa(payload_len))
		data = append(data, msg.Payload)
	}

	if msg.Color != "" {
		data = append(data, "COLOR:"+msg.Color)
	}

	if msg.Bilan != 0 || msg.Action == "STATE_COLLECT" {
		data = append(data, "BILAN:"+strconv.Itoa(msg.Bilan))
	}
	return strings.Join(data, "\n") + "\n", nil
}

func DecodeTorrentPayload(raw_payload string) (torrentlogic.Message, error) {
	lines := strings.Split(raw_payload, "\n")
	msg := torrentlogic.Message{}

	for _, l := range lines {
		if l == "\n" || l == "" {
			continue
		}
		parts := strings.Split(l, ":")
		if parts[0] == "\n" || parts[0] == "" {
			continue
		}
		if len(parts) != 2 {
			return torrentlogic.Message{}, errors.New("Payload line must have exactly 2 component. Found: " + strings.Join(parts, " "))
		}
		key := parts[0]
		value := strings.TrimSpace(parts[1])
		switch key {
		case "MessageType":
			msg.MessageType = torrentlogic.MessageType(value)
		case "DeleteMe":
			b, err := strconv.ParseBool(value)
			if err != nil {
				log.Printf("[PARSER] Erreur: %v\n", err)
				return torrentlogic.Message{}, errors.New("Impossible to convert DeleteMe value")
			}
			msg.DeleteMe = b
		case "SenderID":
			msg.SenderID = value
		case "TransferID":
			msg.TransferID = value
		case "TargetID":
			msg.TargetID = value
		case "TransferRelatedEvent":
			val, err := strconv.Atoi(value)
			if err != nil {
				log.Printf("[PARSER] Erreur: %v\n", err)
				return torrentlogic.Message{}, errors.New("Impossible to convert TransferRelatedEvent value")
			}
			msg.TransferRelatedEvent = torrentlogic.TransferRelatedEvent(val)
		case "FileID":
			msg.FileID = value
		case "PartID":
			val, err := strconv.Atoi(value)
			if err != nil {
				log.Printf("[PARSER] Erreur: %v\n", err)
				return torrentlogic.Message{}, errors.New("Impossible to convert PartID value")
			}
			msg.PartID = uint(val)
		case "Content":
			msg.Content = value
		default:
			return torrentlogic.Message{}, errors.New("Found unknown field: " + key)
		}
	}
	return msg, nil
}

func EncodeTorrentPayload(msg torrentlogic.Message) (string, error) {
	data := make([]string, 0, 10)

	if msg.MessageType == "" {
		return "", errors.New("Empty MessageType")
	}
	data = append(data, "MessageType:"+string(msg.MessageType))

	data = append(data, "DeleteMe:"+strconv.FormatBool(msg.DeleteMe))

	if msg.SenderID != "" {
		data = append(data, "SenderID:"+msg.SenderID)
	}

	if msg.TransferID != "" {
		data = append(data, "TransferID:"+msg.TransferID)
	}

	if msg.TargetID != "" {
		data = append(data, "TargetID:"+msg.TargetID)
	}

	data = append(data, "TransferRelatedEvent:"+strconv.Itoa(int(msg.TransferRelatedEvent)))

	if msg.FileID != "" {
		data = append(data, "FileID:"+msg.FileID)
	}

	data = append(data, "PartID:"+strconv.Itoa(int(msg.PartID)))

	if msg.Content != "" {
		data = append(data, "Content:"+msg.Content)
	}

	return strings.Join(data, "\n") + "\n", nil
}
