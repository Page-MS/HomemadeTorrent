package event_loop

import (
	"HomemadeTorrent/pkg/control"
	networkcontroler "HomemadeTorrent/pkg/network_controler"
	"HomemadeTorrent/pkg/parser"
	"HomemadeTorrent/pkg/registre"
	"HomemadeTorrent/pkg/webui"
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

func StartBootstrap(siteID string) {
	// We contact the bootstrap node which we have the access to the fifo
	write("BOOTSTRAP " + siteID)
	//Start(allSiteIDs, siteID)
}

func Start(allSiteIDs []string, siteID string) {
	// Debug
	dir, _ := os.Getwd()
	log.Printf("working dir: %s", dir)

	// Channels
	eventQueue := make(chan Event, 100)
	processingChan := make(chan Event, 100)

	// Init Controler et Registre
	register := registre.Registre{}
	registre.MakeInitialHardcodedRegister(&register, "../../bin/baseFiles", "../../bin/parts", allSiteIDs)
	registre.InitialiseRegistre(siteID, &register)

	networkControler := networkcontroler.NewNetworkControler(siteID, allSiteIDs, &register)

	log.Printf("SiteID: %s, Index: %d, All sites: %s\n", networkControler.SiteID, networkControler.Controler.SiteIndex, allSiteIDs)

	go listenStdEntry(eventQueue)
	go listenUserUIInput(eventQueue, siteID, networkControler.Controler, &register)
	go listenLocalTorrentOutput(eventQueue, networkControler.Controler)
	go siteLogic(processingChan, eventQueue, networkControler)

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
	//fmt.Println("DEBUG: Le lecteur clavier est bien lancé")
	scanner := bufio.NewScanner(os.Stdin)
	const maxCapacity = 10 * 1024 * 1024 // 10 Mo pour pouvoir envoyer le registre
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)
	var buffer strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			// Si on a déjà accumulé des données, on traite le message
			if buffer.Len() > 0 {
				msg := buffer.String()
				buffer.Reset()

				log.Printf("[EVENT_LOOP] Message réseau lu en entrée: %s\n", msg)

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

// interface web
func listenUserUIInput(queue chan<- Event, siteID string, controler *control.Controller, register *registre.Registre) {
	onMsg := func(msg string) {
		queue <- Event{
			Type:   ReadMessage,
			Source: FromLocalUser,
			Data:   msg,
		}
	}
	webui.StartWebUI(siteID, controler.SiteIndex, onMsg, register)
}

func listenLocalTorrentOutput(queue chan<- Event, c *control.Controller) {
	for msg := range c.OutputTorrentChan {
		ctrlMsg, err := c.TorrentMessageToParserMessage(msg)
		if err != nil {
			log.Printf("[EVENT_LOOP] Erreur de lecture local torrent output: %v\n", err)
		}
		strMsg, err := parser.Encode(ctrlMsg)
		if err != nil {
			log.Printf("[EVENT_LOOP] Erreur de lecture local torrent output: %v\n", err)
		}

		queue <- Event{
			Type:   ReadMessage,
			Source: FromLocalUser,
			Data:   strMsg,
		}
	}
}

func write(msg string) {
	log.Println("[EVENT_LOOP] Message écrit en sortie:", msg)
	_, err := fmt.Fprintf(os.Stdout, "%s\n\n", msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur écriture stdout: %v\n", err)
	}
}

func siteLogic(input <-chan Event, eventQueue chan<- Event, nc *networkcontroler.NetworkControler) {
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
				responses = nc.HandleIncomingFromNetwork(cleanRaw)
			case FromLocalUser:
				responses = nc.HandleIncomingFromLocal(cleanRaw)
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
