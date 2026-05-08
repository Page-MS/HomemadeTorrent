package main

import (
	"HomemadeTorrent/pkg/registre"
	torrentlogic "HomemadeTorrent/pkg/torrentLogic"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal("Usage:\n" +
			"  program <siteID> <allSiteIDs...>\n\n" +
			"Example:\n" +
			"  go run main.go Site1 Site1 Site2 Site3")
	}

	siteID := args[0]
	reg := registre.Registre{}
	registre.MakeInitialHardcodedRegister(&reg, "./bin/baseFiles", "./bin/parts")
	registre.InitialiseRegistre(siteID, &reg)

	MessageChannelForTransfer := make(chan torrentlogic.Message)
	outputChannel := make(chan torrentlogic.Message)

	// Launch a goroutine to receive outgoing messages
	go func() {
		for msg := range outputChannel {
			fmt.Printf("\nOutgoing message received: %+v\n", msg)
		}
	}()

	// Launch transfers in separate goroutines
	go torrentlogic.StartOutgoingTransfer("1", "3628364a96f7d5dc7b383b9ea5c5415c20cbdfd5fa437c92b7d0038f456e25f2", siteID, &reg, MessageChannelForTransfer, outputChannel)

	torrentlogic.HandlePeerAskingIfWeHavePart(siteID, "Alexis", "3628364a96f7d5dc7b383b9ea5c5415c20cbdfd5fa437c92b7d0038f456e25f2", 2, &reg, outputChannel)

	// Keep main running
	time.Sleep(50 * time.Second)
}
