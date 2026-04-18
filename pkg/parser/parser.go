package parser

import (
    "github.com/google/uuid"
	"errors"
	"strconv"
	"strings"
)


type Message = struct {
	Action string
	Id string
	Object string
	Chunk int
	payload_len int
	Payload string
}

// ACTION:qlksdjfqmlsdfjmqsdlkf
// ID:slkdjfmqldskjf
// OBJECT:sldkfjsdlkf
// CHUNK:1
// PAYLOAD_LEN:123
// <payload_qlksdjfmlqksdjflkqsd>

// Mandatory fields: ACTION,ID
// if no PAYLOAD_LEN, then no payload
// if len(payload) == 0, then no payload & payload_len is send
// chunk is zero indexed, -1 indicates no chunk


// string -> Message, \n is the sep
func Decode(raw_data string) (Message, error) {
	lines := strings.Split(raw_data, "\n")
	msg := Message{}

	for i, l := range lines {
		if (l == "\n" || l == "") {
			continue
		}
		parts := strings.Split(l, ":")
		if (parts[0] == "\n" || parts[0] == "") {
			continue
		}
		if (len(parts) != 2) {
			return Message{}, errors.New("Message line must have exactly 2 component. Found: " + strings.Join(parts, " "))
		}
		key   := parts[0]
		value := parts[1]

		switch (key) {
		case "ACTION": {
			msg.Action = value
		}
		case "ID": {
			msg.Id     = value
		}
		case "OBJECT": {
			msg.Object = value
		}
		case "CHUNK": {
			val, err := strconv.Atoi(value)
			if err != nil {
				return Message{}, errors.New("Impossible to chunk nb")
			}
			msg.Chunk = val
		}
		case "PAYLOAD_LEN": {
			val, err := strconv.Atoi(value)
			if err != nil {
				return Message{}, errors.New("Impossible to payload_len")
			}
			msg.payload_len = val
			msg.Payload = lines[i + 1]
			if len(msg.Payload) <= 0 {
				return Message{}, errors.New("Provided payload len but no payload")
			}

			return msg, nil // return now
		}
		default:{
			return Message{}, errors.New("Found unknonw field: " + key)
		}
		}
	}
	return msg, nil
}

func Encode(msg Message) (string, error) {
	data := make([]string, 0, 10)

	if msg.Action == ""{
		return "", errors.New("Empty action")
	} else {
		data = append(data, "ACTION:" + msg.Action)
	}

	if msg.Id == ""{
		msg.Id = uuid.New().String()
	}
	data = append(data, "ID:" + msg.Id)

	if msg.Object != ""{
		data = append(data, "OBJECT:" + msg.Object)
	}

	if msg.Chunk != -1{
		data = append(data, "CHUNK:" + strconv.Itoa(msg.Chunk))
	}

	payload_len := len(msg.Payload)
	if payload_len > 0 {
		data = append(data, "PAYLOAD_LEN:" + strconv.Itoa(payload_len))
		data = append(data, msg.Payload)
	}
		
	return strings.Join(data, "\n"), nil
}
