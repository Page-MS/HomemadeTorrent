// PARSER BIDON POUR QUE HANDLER PUISSE COMPILER EN ATTENDANT

package parser

import (
	"strconv"
	"strings"
)

type Message struct {
	Action     string
	Id         string
	Object     string
	Chunk      int
	PayloadLen int
	Payload    string
	Stamp      int
	Vector     []int
}

func Decode(raw_data string) (Message, error) {
	lines := strings.Split(raw_data, "\n")
	msg := Message{Chunk: -1}

	for _, l := range lines {
		parts := strings.Split(l, ":")
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		val := parts[1]

		switch key {
		case "ACTION":
			msg.Action = val
		case "ID":
			msg.Id = val
		case "STAMP":
			msg.Stamp, _ = strconv.Atoi(val)
		case "VECTOR":
			vParts := strings.Split(val, ",")
			for _, v := range vParts {
				i, _ := strconv.Atoi(v)
				msg.Vector = append(msg.Vector, i)
			}
		}
	}
	return msg, nil
}

func Encode(msg Message) (string, error) {
	return "", nil
}
