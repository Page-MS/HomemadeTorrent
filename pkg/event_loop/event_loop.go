package event_loop

import (
	"HomemadeTorrent/pkg/control"
	"bufio"
	"fmt"
	"log"
	"os"
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
	FromLocalApp
)

// Un evenement contient un message à lire/ecrire
type Event struct {
	Type   EventType
	Source EventSource
	Data   string
}

func Start(allSiteIDs []string, siteID string) {
	// Channels
	eventQueue := make(chan Event, 100)
	processingChan := make(chan Event, 100)
	localAppChan := make(chan string, 100)

	// Controler et app torrent
	controler := control.NewController(siteID, allSiteIDs)

	go listenStdEntry(eventQueue)
	go listenLocalApp(localAppChan, eventQueue)
	go siteLogic(processingChan, eventQueue, controler)

	// Event loop (bloquante)
	for {
		event := <-eventQueue

		switch event.Type {
		case ReadMessage:
			// Passer le message à la go-routine de traitement pour traitement
			processingChan <- event

		case WriteMessage:
			write(event.Data)
		}
	}
}

func listenStdEntry(queue chan<- Event) {
	reader := bufio.NewReader(os.Stdin)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			continue
		}
		log.Println("Message lu en entrée:", msg)
		queue <- Event{
			Type:   ReadMessage,
			Source: FromNetwork,
			Data:   msg,
		}
	}
}

func listenLocalApp(appChan <-chan string, queue chan<- Event) {
	for msg := range appChan {
		queue <- Event{
			Type:   ReadMessage,
			Source: FromLocalApp,
			Data:   msg,
		}
	}
}

func siteLogic(input <-chan Event, eventQueue chan<- Event, c *control.Controller) {
	for msg := range input {

		var responses []string

		switch msg.Source {

		case FromNetwork:
			responses = c.HandleIncomingFromNetwork(msg.Data)

		case FromLocalApp:
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

func write(msg string) {
	log.Println("Message ecrit en sortie:", msg)
	_, err := fmt.Fprintln(os.Stdout, msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur écriture stdout: %v\n", err)
	}
}
