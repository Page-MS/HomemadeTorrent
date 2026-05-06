package torrentlogic

import (
	"HomemadeTorrent/pkg/registre"
	"fmt"
	"math/rand"
	"os"
	"sync"
)

type MessageType string

type PartTransferStatus int

type TransferRelatedEvent int

// 1. On reçoit de notre site (UI) une demande de transfert
// 2. On reçoit AskingForShasum (vas être supprimée après réponse)
// 2. On reçoit AskingForContent (vas être supprimée après réponse)
const (
	AskingFromSC           MessageType = "AskingFromSC"
	DoneWithSC             MessageType = "DoneWithSC"
	TransferRelatedMessage MessageType = "TransferRelatedMessage"
	AskingForShasum        MessageType = "AskingForShasum"
	AskingForContent       MessageType = "AskingForContent"
)

const (
	None TransferRelatedEvent = iota
	ReceivingShasum
	ReceivingContent
)

const (
	StateNotStarted PartTransferStatus = iota
	StateError
	StateRetrying
	StateAskedForAvailability
	StateReceivedShasum
	StateAskedForContent
	StateReceivedContent
	StateCompleted
)

var messageName = map[MessageType]string{
	AskingFromSC:           "Asking for a critical section",
	DoneWithSC:             "Done with critical section",
	TransferRelatedMessage: "Message related to a transfer", // the controller doesn't have to look into it when sending it
	AskingForShasum:        "Asking for the shasum of a part",
	AskingForContent:       "Asking for the content of a part",
}

var stateName = map[PartTransferStatus]string{
	StateNotStarted:           "not started",
	StateError:                "error",
	StateRetrying:             "retrying",
	StateAskedForAvailability: "asked for availability",
	StateReceivedShasum:       "received shasum",
	StateAskedForContent:      "asked for content",
	StateReceivedContent:      "received content",
	StateCompleted:            "completed",
}

// This file contains the guards of the torrent logic, most of the utility functions are in registre.go

type PartTransfer struct {
	partID    uint
	peerID    string
	receiving bool
	state     PartTransferStatus
}

type ongoingTransfer struct {
	file                   registre.File
	partsToAskIDs          []uint
	partsCompletedIDs      []uint
	numberOfPartsCompleted int
	receiving              bool
}

// TODO type for message

type Message struct {
	MessageType          MessageType
	DeleteMe             bool
	SenderID             string
	TransferID           string
	TargetID             string
	TransferRelatedEvent TransferRelatedEvent
	FileID               string
	PartID               uint
	Content              string
}

// Main function to start a transfer
// It will then autonomously handle it until it's finished
func StartOutgoingTransfer(transferID string, fileID string, currentSite string, reg *registre.Registre, incomingMessagesChannel <-chan int, outputMessagesChannel <-chan int) (success bool, error error) {
	fmt.Print("\nStarting transfer for file ID: ", fileID)
	file := reg.GetFileByID(fileID)
	if file == nil {
		return false, fmt.Errorf("file with ID %s not found in register", fileID)
	}
	// We check that the current site is in the register
	if !reg.IsPeerInRegister(currentSite) {
		return false, fmt.Errorf("current site %s not found in register", currentSite)
	}
	// If we already have the file, we don't need to start the transfer
	for _, peer := range file.PeersThatHaveFileID {
		if peer == currentSite {
			fmt.Printf("\nCurrent site %s already has file %s, no need to start transfer", currentSite, file.Name)
			return true, nil
		}
	}
	transfer := &ongoingTransfer{
		file:                   *file,
		partsToAskIDs:          make([]uint, file.NumberOfParts),
		partsCompletedIDs:      make([]uint, 0),
		numberOfPartsCompleted: 0,
		receiving:              true,
	}
	// channel for communication between the transfers goroutine
	transfersResultsChannel := make(chan int)
	// WaitGroup for synchronizing the transfers goroutines
	var wg sync.WaitGroup
	// We make a tab of channels to transmit the messages to the goroutines
	channelsForGoroutines := make([]chan int, file.NumberOfParts)

	for i := uint(0); i < file.NumberOfParts; i++ {
		transfer.partsToAskIDs[i] = i
		go StartTransferForPart(fileID, i, currentSite, reg, transfersResultsChannel, &wg, channelsForGoroutines[i])
		wg.Add(1)
	}
	PrintTransferStatus(transfer)
	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		for n := range transfersResultsChannel {
			transfer.partsCompletedIDs = append(transfer.partsCompletedIDs, uint(n))
			transfer.numberOfPartsCompleted++
			transfer.partsToAskIDs = removeElementFromIntSlice(transfer.partsToAskIDs, uint(n))
			if transfer.numberOfPartsCompleted == int(file.NumberOfParts) {
				close(transfersResultsChannel)
				wg.Done()
				return
			}
		}
	}(&wg)
	wg.Wait()
	if len(transfer.partsToAskIDs) == 0 && len(transfer.partsCompletedIDs) == int(file.NumberOfParts) {
		fmt.Printf("\nTransfer for file %s completed successfully !", file.Name)
	} else {
		fmt.Printf("\nTransfer for file %s completed with errors, parts not received: %v", file.Name, transfer.partsToAskIDs)
		return false, nil
	}
	error = registre.ReassembleFileFromParts(file.Name, "bin/"+currentSite+"/parts", "bin/"+currentSite+"/reassembled", reg)
	if error != nil {
		fmt.Printf("\nError while reassembling file %s: %v", file.Name, error)
		return false, error
	}
	fmt.Printf("\nFile %s reassembled successfully !", file.Name)
	return true, nil
}

func removeElementFromIntSlice(slice []uint, element uint) []uint {
	newSlice := make([]uint, 0)
	for _, e := range slice {
		if e != element {
			newSlice = append(newSlice, e)
		}
	}
	return newSlice

}

func removeElementFromStringSlice(slice []string, element string) []string {
	newSlice := make([]string, 0)
	for _, e := range slice {
		if e != element {
			newSlice = append(newSlice, e)
		}
	}
	return newSlice

}
func PrintTransferStatus(transfer *ongoingTransfer) {
	fmt.Printf("\nTransfer status for file %s\n Number of parts to send: %d\n Number of parts completed: %d\n Is receving ? : %t", transfer.file.Name, len(transfer.partsToAskIDs), transfer.numberOfPartsCompleted, transfer.receiving)
}

func StartTransferForPart(fileID string, partID uint, currentSite string, registre *registre.Registre, channelFin chan<- int, wg *sync.WaitGroup, incomingMessageChannel <-chan int) (err error) {
	// number of peers having the part
	numberOfPeersWithFilePart := len(registre.GetPeersHavingPart(fileID, partID))
	// If none have it, we cannot start the transfer for this part, we log an error and return
	if numberOfPeersWithFilePart == 0 {
		fmt.Printf("\nNo peer has part %d of file %s, cannot start transfer for this part", partID, fileID)
		err = fmt.Errorf("no peer has part %d of file %s", partID, fileID)
		channelFin <- int(partID)
		wg.Done()
		return err
	}
	fmt.Printf("\nStarting transfer for part %d of file %s, number of peers having this part: %d", partID, fileID, numberOfPeersWithFilePart)
	// We take a random peer in the list of peers to not always ask the same peer first
	peersWithPart := registre.GetPeersHavingPart(fileID, partID)
	peerToAsk := peersWithPart[rand.Intn(len(peersWithPart))]
	fmt.Printf("\nAsking peer %s for part %d of file %s", peerToAsk, partID, fileID)
	var partTransferWg sync.WaitGroup
	transferSuccess, err := AskPeerForPart(peerToAsk, fileID, partID, &partTransferWg)
	if err != nil {
		fmt.Printf("\nError while asking peer %s for part %d of file %s: %v", peerToAsk, partID, fileID, err)
		channelFin <- int(partID)
		wg.Done()
		return err
	}
	for !transferSuccess {
		fmt.Printf("\nTransfer for part %d of file %s from peer %s failed, retrying with another peer if available", partID, fileID, peerToAsk)
		// We remove the peer that failed from the list of peers to ask and we try again with another peer if available
		peersWithPart = removeElementFromStringSlice(peersWithPart, peerToAsk)
		if len(peersWithPart) == 0 {
			fmt.Printf("\nNo more peer to ask for part %d of file %s, transfer failed for this part", partID, fileID)
			err = fmt.Errorf("no more peer to ask for part %d of file %s, transfer failed for this part", partID, fileID)
			channelFin <- int(partID)
			wg.Done()
			return err
		}
		peerToAsk := peersWithPart[rand.Intn(len(peersWithPart))]
		transferSuccess, err = AskPeerForPart(peerToAsk, fileID, partID, &partTransferWg)
		if err != nil {
			return err
		}
	}
	fmt.Printf("\nTransfer for part %d of file %s from peer %s succeeded !", partID, fileID, peerToAsk)

	channelFin <- int(partID)
	wg.Done()

	return err

}

func AskPeerForPart(peerID string, fileID string, partID uint, wg *sync.WaitGroup) (success bool, err error) {
	fmt.Print("\nAsking peer ", peerID, " for part ", partID, " of file ", fileID)
	/* transfer := PartTransfer{
		partID:    partID,
		peerID:    peerID,
		receiving: false,
		state:     StateNotStarted,
	} */
	// We send a message asking the peer for its shasum of the part
	// If we have a valid response, we check if the shasum is correct, if not we log an error and return false
	// If the shasum is correct, we ask the peer for the content of the part
	// If we have a valid response, we check if the content is correct by comparing its shasum with the one we received before, if not we log an error and return false
	// If the content is correct, we save it in the right folder and return true
	return true, nil
}

func HandlePeerAskingIfWeHavePart(currentSiteID string, peerID string, fileID string, partID uint, reg *registre.Registre) (err error) {
	fmt.Printf("\nPeer %s is asking if we have part %d of file %s", peerID, partID, fileID)
	// We check locally if we can find the part in our local storage
	// We get the file name

	filePath, err := reg.CheckIfWeHavePartInOurStorage(currentSiteID, fileID, partID, "./bin")
	if err != nil {
		return err
	}
	// We check for the shasum
	shasum := registre.CalculateShasum(filePath)
	SendMessageToPeer(peerID, shasum)

	return nil
}

func SendMessageToPeer(peerID string, message string) {
	fmt.Printf("\nSending message to peer %s: %s", peerID, message)
	//TODO
}

func HandlePeerAskingForPartContent(currentSiteID string, peerID string, fileID string, partID uint, reg *registre.Registre) (err error) {
	fmt.Printf("\nPeer %s is asking for the content of part %d of file %s", peerID, partID, fileID)
	// We check locally if we can find the part in our local storage
	// We get the file name

	filePath, err := reg.CheckIfWeHavePartInOurStorage(currentSiteID, fileID, partID, "./bin")
	if err != nil {
		return err
	}
	// We get the file content in a string
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("could not get file info: %v", err)
	}
	fileSize := uint(fileInfo.Size())
	filePartContent := make([]byte, fileSize)
	file.Read(filePartContent)
	// We send the content of the part to the peer
	SendMessageToPeer(peerID, string(filePartContent))

	return nil
}

func HandlePeerRespondingWithShasum(currentSiteID string, peerID string, fileID string, partID uint, shasum string, reg *registre.Registre) (err error) {
	fmt.Printf("\nPeer %s is responding with shasum for part %d of file %s: %s", peerID, partID, fileID, shasum)
	// We check if the shasum is correct by comparing it with the shasum of the part we have in in the register
	// We get the file name
	shasumFromRegister, err := reg.GetShasumOfPart(fileID, partID)
	if err != nil {
		return fmt.Errorf("could not get shasum of part %d of file %s from register: %v", partID, fileID, err)
	}
	fmt.Printf("\nShasum from register for part %d of file %s: %s", partID, fileID, shasumFromRegister)

	if shasumFromRegister != shasum {
		return fmt.Errorf("shasum calculated %s does not match shasum received %s for part %d of file %s", shasumFromRegister, shasum, partID, fileID)
	}
	fmt.Printf("\nShasum for part %d of file %s is correct", partID, fileID)
	return nil
}
