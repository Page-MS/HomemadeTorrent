package networkcontroler

import (
	"HomemadeTorrent/pkg/parser"
	"log"
	"os"
)

const (
	START_ASKING_TO_JOIN_NETWORK = "START_ASKING_TO_JOIN_NETWORK"
)

func (nc *NetworkControler) AskPeerToJoinNetwork(pMsg parser.Message) string {
	fifoPath := "/tmp/network_fifos/in_" + pMsg.Payload

	// Ouvrir le FIFO en écriture
	fifo, err := os.OpenFile(fifoPath, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		log.Printf("[NETWORK_CONTROLER] Impossible d'ouvrir le fifo %s : %v\n", fifoPath, err)
	}
	defer fifo.Close()

	// Écrire le string
	message := "hello\n"
	_, err = fifo.WriteString(message)
	if err != nil {
		log.Fatal(err)
	}
}
