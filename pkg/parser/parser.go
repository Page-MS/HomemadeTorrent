package parser

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Message = struct {
	Action      string
	Id          string
	Stamp       int
	Vect        []int
	Dest        string
	Sender      string
	Object      string
	Chunk       int
	payload_len int
	Payload     string
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

	for i, l := range lines {
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
		case "OBJECT":
			{
				msg.Object = value
			}

		case "DEST":
			{
				msg.Dest = value
			}

		case "SENDER":
			{
				msg.Sender = value
			}

		case "CHUNK":
			{
				val, err := strconv.Atoi(value)
				if err != nil {
					return Message{}, errors.New("Impossible to convert CHUNK nb")
				}
				msg.Chunk = val
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
			{
				msg.Vect = make([]int, 0)
				for _, val := range strings.Split(value, ",") {
					nb, err := strconv.Atoi(strings.TrimSpace(val))
					if err != nil {
						return Message{}, errors.New("Impossible to convert VECT value")
					}
					msg.Vect = append(msg.Vect, nb)
				}
			}

		case "PAYLOAD_LEN":
			{
				val, err := strconv.Atoi(value)
				if err != nil {
					return Message{}, errors.New("Impossible to payload_len")
				}
				msg.payload_len = val
				msg.Payload = lines[i+1]
				if len(msg.Payload) <= 0 {
					return Message{}, errors.New("Provided payload len but no payload")
				}

				return msg, nil // return now
			}

		default:
			{
				return Message{}, errors.New("Found unknonw field: " + key)
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

	if msg.Object != "" {
		data = append(data, "OBJECT:"+msg.Object)
	}

	if msg.Chunk != -1 {
		data = append(data, "CHUNK:"+strconv.Itoa(msg.Chunk))
	}

	payload_len := len(msg.Payload)
	if payload_len > 0 {
		data = append(data, "PAYLOAD_LEN:"+strconv.Itoa(payload_len))
		data = append(data, msg.Payload)
	}

	return strings.Join(data, "\n") + "\n", nil
}
