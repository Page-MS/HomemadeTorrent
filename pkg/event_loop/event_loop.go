package event_loop

import (
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

// Un evenement contient un message à lire/ecrire
type Event struct {
	Type EventType
	Data string
}

func Start() {
	eventQueue := make(chan Event, 100)
	processingChan := make(chan string, 100)

	// Configuration des "interruptions"
	go listenStdEntry(eventQueue)
	go siteLogic(processingChan, eventQueue)

	// Event loop (bloquante)
	for {
		event := <-eventQueue

		switch event.Type {
		case ReadMessage:
			read(event.Data, processingChan)

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
			Type: ReadMessage,
			Data: msg,
		}
	}
}

func siteLogic(input <-chan string, eventQueue chan<- Event) {
	for msg := range input {
		// Passage au controle (decode + algo réparti)
		log.Println("Message passé au contrôleur:", msg)

		// Envoie reponse dans la queue d'actions
		eventQueue <- Event{
			Type: WriteMessage,
			Data: "response de la logique du site",
		}
	}
}

func read(raw string, processingChan chan<- string) {
	// Passer le message à la go-routine de traitement pour traitement
	processingChan <- raw
}

func write(msg string) {
	log.Println("Message ecrit en sortie:", msg)
	_, err := fmt.Fprintln(os.Stdout, msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur écriture stdout: %v\n", err)
	}
}
