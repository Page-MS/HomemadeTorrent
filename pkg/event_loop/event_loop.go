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

func Start(allSiteIDs []string, siteID string, nbNeighbors int, isBootstrap int) {
	// Debug
	dir, _ := os.Getwd()
	log.Printf("working dir: %s", dir)

	// Channels
	eventQueue := make(chan Event, 10000)
	processingChan := make(chan Event, 10000)

	// Init Controler et Registre
	register := registre.Registre{}
	if isBootstrap == 0 {
		registre.MakeInitialHardcodedRegister(&register, "../../bin/baseFiles", "../../bin/parts", allSiteIDs)
	} else {
		log.Printf("Bootstrap pour le site %s\n", siteID)
	}

	registre.InitialiseRegistre(siteID, &register)

	networkControler := networkcontroler.NewNetworkControler(siteID, allSiteIDs, &register, nbNeighbors)

	log.Printf("SiteID: %s, Index: %d, All sites: %s, NbVoisins: %d\n", networkControler.SiteID, networkControler.Controler.SiteIndex, allSiteIDs, nbNeighbors)

	go listenStdEntry(eventQueue)
	go listenUserUIInput(eventQueue, siteID, networkControler.Controler, &register, isBootstrap)
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

		// Détection d'un ACTION: au milieu d'une ligne (fusion de deux messages)
		if buffer.Len() > 0 {
			if idx := strings.Index(line, "ACTION:"); idx > 0 {
				// On termine le message en cours avec ce qu'il y a avant ACTION:
				buffer.WriteString(line[:idx] + "\n")
				msg := buffer.String()
				buffer.Reset()
				log.Printf("[EVENT_LOOP] WARN: fusion détectée, envoi du message en cours: %s\n", msg)
				queue <- Event{
					Type:   ReadMessage,
					Source: FromNetwork,
					Data:   msg,
				}
				// Le nouveau message commence à ACTION:
				line = line[idx:]
			}
		} else {
			// Si le buffer est vide, c'est la première ligne d'un nouveau message
			if buffer.Len() == 0 {
				if idx := strings.Index(line, "ACTION:"); idx > 0 {
					garbage := line[:idx]
					line = line[idx:]
					log.Printf("[EVENT_LOOP] WARN: résidu détecté avant ACTION, supprimé: %q\n", garbage)
				}
			}
		}
		// On rajoute un \n manuellement pour reconstruire le message proprement
		buffer.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		log.Println("[EVENT_LOOP] Erreur de lecture Stdin:", err)
	}
}

// interface web
func listenUserUIInput(queue chan<- Event, siteID string, controler *control.Controller, register *registre.Registre, isBoostrap int) {
	onMsg := func(msg string) {
		queue <- Event{
			Type:   ReadMessage,
			Source: FromLocalUser,
			Data:   msg,
		}
	}
	webui.StartWebUI(siteID, controler.SiteIndex, onMsg, register, isBoostrap)
}

func listenLocalTorrentOutput(queue chan<- Event, c *control.Controller) {
	for msg := range c.OutputTorrentChan {
		log.Printf("[TORRENT_PART2] La part de partID %d envoyée est : binaire avec event %d!\n", msg.PartID, msg.TransferRelatedEvent)
		ctrlMsg, err := c.TorrentMessageToParserMessage(msg)
		if err != nil {
			log.Printf("[EVENT_LOOP] Erreur de lecture local torrent output: %v\n", err)
		}
		log.Printf("[TORRENT_PART3] La part de ID %s encodé est %s :  !\n", ctrlMsg.Id, ctrlMsg.Payload)
		strMsg, err := parser.Encode(ctrlMsg)
		if err != nil {
			log.Printf("[EVENT_LOOP] Erreur de lecture local torrent output: %v\n", err)
		}
		log.Printf("[TORRENT_PART4] La part de ID %s string est %s :  !\n", ctrlMsg.Id, strMsg)

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
