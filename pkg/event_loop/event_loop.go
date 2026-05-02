package event_loop

import (
	"HomemadeTorrent/pkg/control"
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// Definitions des evenements
type EventType int

const (
	ReadMessage EventType = iota
	WriteMessage
)

type EventSource int

const (
	FromNetwork EventSource = iota
	FromLocalUser
)

// Un evenement contient un message à lire/ecrire
type Event struct {
	Type   EventType
	Source EventSource
	Data   string
}

func Start(allSiteIDs []string, siteID string) {
	logFile, _ := os.Create(siteID + ".log")
	log.SetOutput(logFile)

	// Channels
	eventQueue := make(chan Event, 100)
	processingChan := make(chan Event, 100)
	localUserChan := make(chan string, 100)

	// Controler et app torrent
	controler := control.NewController(siteID, allSiteIDs)

	go listenStdEntry(eventQueue)
	go listenUserInput(localUserChan, eventQueue)
	go siteLogic(processingChan, eventQueue, controler)

	log.Printf("[EVENT_LOOP] START\n")

	// Event loop (bloquante)
	for {
		event := <-eventQueue

		switch event.Type {
		case ReadMessage:
			// Passer le message à la go-routine contenant la logique du site
			processingChan <- event

		case WriteMessage:
			write(event.Data)
		}
	}
}

func listenStdEntry(queue chan<- Event) {
	reader := bufio.NewReader(os.Stdin)
	var buffer strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("[EVENT_LOOP] Lecture impossible, entrée ignorée")
			continue
		}

		if strings.TrimSpace(line) == "" {
			// fin de message
			msg := buffer.String()
			buffer.Reset()

			log.Printf("[EVENT_LOOP] Message lu en entrée: %s\n", msg)

			queue <- Event{
				Type:   ReadMessage,
				Source: FromNetwork,
				Data:   msg,
			}
			continue
		}

		buffer.WriteString(line)
	}
}

func listenUserInput(userInputChan <-chan string, queue chan<- Event) {
	for msg := range userInputChan {
		queue <- Event{
			Type:   ReadMessage,
			Source: FromLocalUser,
			Data:   msg,
		}
	}
}

func write(msg string) {
	log.Println("[EVENT_LOOP] Message ecrit en sortie:", msg)
	_, err := fmt.Fprintf(os.Stdout, msg+"\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur écriture stdout: %v\n", err)
	}
}

func siteLogic(input <-chan Event, eventQueue chan<- Event, c *control.Controller) {
	for msg := range input {

		var responses []string

		switch msg.Source {

		case FromNetwork:
			responses = c.HandleIncomingFromNetwork(msg.Data)

		case FromLocalUser:
			responses = c.HandleIncomingFromLocal(msg.Data)
		}

		for _, r := range responses {
			eventQueue <- Event{
				Type: WriteMessage,
				Data: r,
			}
		}
	}
}
