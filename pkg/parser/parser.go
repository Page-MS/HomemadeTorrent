package parser

import (
	"HomemadeTorrent/pkg/utils"
	"errors"
	"strconv"
	"strings"
)


type Message = struct {
	action string
	id string
	object string
	chunk int
	checksum string
	payload_len int
	payload string
}

// ACTION:qlksdjfqmlsdfjmqsdlkf
// ID:slkdjfmqldskjf
// OBJECT:sldkfjsdlkf
// CHUNK:1
// CHECKSUM:smldskqsdfmjlk
// PAYLOAD_LEN:123
// <payload_qlksdjfmlqksdjflkqsd>


// string -> Message, \n is the sep
func decode(raw_data string) (Message, error) {
	lines := strings.Split(raw_data, "\n")
	msg := Message{}

	for _, l := range lines {
		parts := strings.Split(l, ":")
		if (len(parts) != 2) {
			return Message{}, errors.New("Message line must have exactly 2 component.")
		}
		key   := parts[0]
		value := parts[1]

		switch (key) {
		case "ACTION": {
			msg.action = value
		}
		case "ID": {
			msg.id     = value
		}
		case "OBJECT": {
			msg.object = value
		}
		case "CHUNK": {
			val, err := strconv.Atoi(value)
			if err != nil {
				return Message{}, errors.New("Impossible to chunk nb")

			}
			msg.chunk = val
		}
		case "CHECKSUM": {
			msg.checksum = value
		}
		case "PAYLOAD_LEN": {
			val, err := strconv.Atoi(value)
			if err != nil {
				return Message{}, errors.New("Impossible to payload_len")
			}
			msg.payload_len = val
			utils.TODO("implement payload decode")
		}
		}
	}
	return msg, nil
}

func encode(msg Message) (string, error) {
	data := make([]string, 2)

	if msg.action == ""{
		return "", errors.New("Empty action")
	} else {
		data = append(data, "ACTION:" + msg.action)
	}

	if msg.id == ""{
		utils.TODO("Set uuid")
	}
	data = append(data, "ID:" + msg.id)

	if msg.object != ""{
		data = append(data, "OBJECT:" + msg.object)
	}

	if msg.chunk != -1{
		data = append(data, "CHUNK:" + strconv.Itoa(msg.chunk))
	}

	if msg.checksum != ""{
		utils.TODO("Calculate checksum")
	}

	payload_len := len(msg.payload)
	if payload_len > 0 {
		data = append(data, "PAYLOAD_LEN:" + strconv.Itoa(payload_len))
		data = append(data, msg.payload)
	}
		
	return strings.Join(data, "\n"), nil
}
