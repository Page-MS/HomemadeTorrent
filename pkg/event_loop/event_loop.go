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
			//fmt.Println("[DEBUG] EventLoop: Message reçu, envoi vers processingChan")
			// Passer le message à la go-routine contenant la logique du site
			processingChan <- event

		case WriteMessage:
			//fmt.Println("[DEBUG] EventLoop: Demande d'écriture détectée !")
			write(event.Data)
		}
	}
}

func listenStdEntry(queue chan<- Event) {
	//fmt.Println("DEBUG: Le lecteur clavier est bien lancé")
	scanner := bufio.NewScanner(os.Stdin)
	var buffer strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			//fmt.Println(">>> LIGNE VIDE DÉTECTÉE, ENVOI DANS LA QUEUE")
			// Si on a déjà accumulé des données, on traite le message
			if buffer.Len() > 0 {
				msg := buffer.String()
				buffer.Reset()

				log.Printf("[EVENT_LOOP] Message lu en entrée: %s\n", msg)

				queue <- Event{
					Type:   ReadMessage,
					Source: FromNetwork,
					Data:   msg,
				}
			}
			continue
		}
		// On rajoute un \n manuellement pour reconstruire le message proprement
		buffer.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		log.Println("[EVENT_LOOP] Erreur de lecture Stdin:", err)
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
	log.Println("[EVENT_LOOP] Message écrit en sortie:", msg)
	_, err := fmt.Fprintf(os.Stdout, msg+"\n\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur écriture stdout: %v\n", err)
	}
}

func siteLogic(input <-chan Event, eventQueue chan<- Event, c *control.Controller) {
	for event := range input {
		// Découpage par double saut de ligne pour séparer les messages collés
		rawMessages := strings.Split(event.Data, "\n\n")

		for _, raw := range rawMessages {
			cleanRaw := strings.TrimSpace(raw)
			if cleanRaw == "" {
				continue
			}

			var responses []string
			switch event.Source {
			case FromNetwork:
				responses = c.HandleIncomingFromNetwork(cleanRaw)
			case FromLocalUser:
				responses = c.HandleIncomingFromLocal(cleanRaw)
			}

			for _, r := range responses {
				eventQueue <- Event{
					Type: WriteMessage,
					Data: r,
				}
			}
		}
	}
}
