package torrentlogic

import (
	"HomemadeTorrent/pkg/registre"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
)

// This file contains the guards of the torrent logic, most of the utility functions are in registre.go

type MessageType string

type PartTransferStatus int

type TransferRelatedEvent int

// I considered adding a register field to the messages but this seemed more elegant
const (
	RegisterUpdate MessageType = "RegisterUpdate"
)

// 1. On reçoit de notre site (UI) une demande de transfert
// 2. On reçoit AskingForShasum (vas être supprimée après réponse)
// 2. On reçoit AskingForContent (vas être supprimée après réponse)

const BIN_PATH = registre.BIN_PATH_FROM_MAIN

const (
	AskingFromSC           MessageType = "AskingFromSC"
	StartSC                MessageType = "StartSC"
	DoneWithSC             MessageType = "DoneWithSC"
	TransferRelatedMessage MessageType = "TransferRelatedMessage"
	StartTransfers         MessageType = "StartTransfers"
	AskingForShasum        MessageType = "AskingForShasum"
	AskingForContent       MessageType = "AskingForContent"
)

// Possible types of messages that can be received that the controller should just pass to the transfer without looking into it
const (
	None TransferRelatedEvent = iota
	ReceivingShasum
	ReceivingContent
)

// Messages types
var messageName = map[MessageType]string{
	AskingFromSC:           "Asking for a critical section",
	StartSC:                "Authorised to start the critical section",
	DoneWithSC:             "Done with critical section",
	TransferRelatedMessage: "Message related to a transfer", // the controller doesn't have to look into it when sending it
	AskingForShasum:        "Asking for the shasum of a part",
	AskingForContent:       "Asking for the content of a part",
}

type ongoingTransfer struct {
	file                   registre.File
	partsToAskIDs          []uint
	partsCompletedIDs      []uint
	numberOfPartsCompleted int
	receiving              bool
}

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

// Main function to start a transfer (ask for a file)
// It will then autonomously handle it until it's finished
func StartOutgoingTransfer(transferID string, fileID string, currentSite string, reg *registre.Registre, incomingMessagesChannel <-chan Message, outputMessagesChannel chan<- Message, scChan <-chan Message) (success bool, error error) {
	log.Print("\n[TORRENT] Starting transfer for file ID: ", fileID)
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
			log.Printf("\n[TORRENT] Current site %s already has file %s, no need to start transfer", currentSite, file.Name)
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
	transfersResultsChannel := make(chan uint)
	// WaitGroup for synchronizing the transfers goroutines
	var wg sync.WaitGroup
	// We make a tab of channels to transmit the messages to the goroutines
	// +1 for handling the file parts starting at one
	partIncomingChannels := make([]chan Message, file.NumberOfParts+1)
	for i := uint(1); i <= file.NumberOfParts; i++ {
		transfer.partsToAskIDs[i-1] = i
		partIncomingChannels[i] = make(chan Message)
		wg.Add(1)
		go StartTransferForPart(transferID, fileID, i, currentSite, reg, transfersResultsChannel, &wg, partIncomingChannels[i], outputMessagesChannel)
	}
	// Goroutine to dispatch incoming messages to the corresponding part goroutine
	go func() {
		for msg := range incomingMessagesChannel {
			if msg.PartID >= 1 && msg.PartID <= file.NumberOfParts {
				partIncomingChannels[msg.PartID] <- msg
			}
		}
	}()
	PrintTransferStatus(transfer)
	// TODO: Debug prépost
	/*
		if currentSite == "3" {
			log.Printf("[TEST] snapshot enclenchée\n")
			cmd := exec.Command("../../pkg/user_input/ui_hooks/startSnapshot.sh", "1")
			_, err := cmd.CombinedOutput()
			if err != nil {
				log.Println("Erreur :", err)
			}
		}
	*/
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		for n := range transfersResultsChannel {
			transfer.partsCompletedIDs = append(transfer.partsCompletedIDs, n)
			transfer.numberOfPartsCompleted++
			transfer.partsToAskIDs = removeElementFromIntSlice(transfer.partsToAskIDs, n)
			if transfer.numberOfPartsCompleted == int(file.NumberOfParts) {
				close(transfersResultsChannel)
				wg.Done()
				return
			}
		}
	}(&wg)
	wg.Wait()
	if len(transfer.partsToAskIDs) == 0 && len(transfer.partsCompletedIDs) == int(file.NumberOfParts) {
		log.Printf("\n[TORRENT] Transfer for file %s completed successfully !", file.Name)
	} else {
		log.Printf("\n[TORRENT] Transfer for file %s completed with errors, parts not received: %v", file.Name, transfer.partsToAskIDs)
		return false, nil
	}
	error = registre.ReassembleFileFromParts(file.Name, BIN_PATH+"/"+currentSite+"/parts", BIN_PATH+"/"+currentSite+"/reassembled", reg)
	if error != nil {
		log.Printf("\n[TORRENT] Error while reassembling file %s: %v", file.Name, error)
		return false, error
	}
	log.Printf("\n[TORRENT] File %s reassembled successfully !", file.Name)

	// SC handling
	// We update our register to say we have the file
	log.Printf("\n[TORRENT] FileID : %s", fileID)
	err := reg.AddPeerHavingFile(currentSite, fileID)
	if err != nil {
		log.Printf("\n[TORRENT] Error while updating register to say we have the file %s: %v", file.Name, err)
		return false, err
	}

	// We ask for a SC
	SendMessageToPeer(AskingFromSC, false, currentSite, transferID, transferID, 0, fileID, 0, "", outputMessagesChannel)
	select {
	case message := <-scChan:
		log.Printf("\n[TORRENT] Message reçu %v: \n", message)
		// If this is the authorziation for a critical section
		if message.MessageType == StartSC {
			log.Printf("\n[TORRENT] Received authorization to start critical section for file %s, updating register of the others", file.Name)
			err = SendRegisterUpdateToPeer(currentSite, transferID, "-1", fileID, 0, outputMessagesChannel, reg)
			if err != nil {
				log.Printf("\n[TORRENT] Error while sending register update to peers for file %s: %v", file.Name, err)
				return false, err
			}
			// We send the message announcing we have finished with our critical section
			SendMessageToPeer(DoneWithSC, false, currentSite, transferID, "-1", 0, fileID, 0, "", outputMessagesChannel)

		} else {
			err = fmt.Errorf("ERROR: Unexpected message received while waiting for a SC authorization ")
			return false, err
		}
	}
	return true, nil
}

// Util function
func removeElementFromIntSlice(slice []uint, element uint) []uint {
	newSlice := make([]uint, 0)
	for _, e := range slice {
		if e != element {
			newSlice = append(newSlice, e)
		}
	}
	return newSlice

}

// Util function
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
	log.Printf("\n[TORRENT] Transfer status for file %s\n Number of parts to send: %d\n Number of parts completed: %d\n Is receving ? : %t", transfer.file.Name, len(transfer.partsToAskIDs), transfer.numberOfPartsCompleted, transfer.receiving)
}

// Handle the transfer for asking a single part of a file, use a channel to indicate its success and ID
func StartTransferForPart(transferID string, fileID string, partID uint, currentSite string, registre *registre.Registre, channelFin chan<- uint, wg *sync.WaitGroup, incomingMessagesChannel <-chan Message, outputMessagesChannel chan<- Message) (err error) {
	// number of peers having the part
	numberOfPeersWithFilePart := len(registre.GetPeersHavingPart(fileID, partID))
	// If none have it, we cannot start the transfer for this part, we log an error and return
	if numberOfPeersWithFilePart == 0 {
		log.Printf("\n[TORRENT] No peer has part %d of file %s, cannot start transfer for this part", partID, fileID)
		err = fmt.Errorf("no peer has part %d of file %s", partID, fileID)
		channelFin <- partID
		wg.Done()
		return err
	}
	log.Printf("\n[TORRENT] Starting transfer for part %d of file %s, number of peers having this part: %d", partID, fileID, numberOfPeersWithFilePart)
	// We take a random peer in the list of peers to not always ask the same peer first
	peersWithPart := registre.GetPeersHavingPart(fileID, partID)
	peerToAsk := peersWithPart[rand.Intn(len(peersWithPart))]
	log.Printf("\n[TORRENT] Asking peer %s for part %d of file %s", peerToAsk, partID, fileID)
	var partTransferWg sync.WaitGroup
	transferSuccess, err := AskPeerForPart(transferID, peerToAsk, fileID, partID, currentSite, registre, &partTransferWg, incomingMessagesChannel, outputMessagesChannel)
	if err != nil {
		log.Printf("\n[TORRENT] Error while asking peer %s for part %d of file %s: %v", peerToAsk, partID, fileID, err)
		channelFin <- partID
		wg.Done()
		return err
	}
	for !transferSuccess {
		log.Printf("\n[TORRENT] Transfer for part %d of file %s from peer %s failed, retrying with another peer if available", partID, fileID, peerToAsk)
		// We remove the peer that failed from the list of peers to ask and we try again with another peer if available
		peersWithPart = removeElementFromStringSlice(peersWithPart, peerToAsk)
		if len(peersWithPart) == 0 {
			log.Printf("\n[TORRENT] No more peer to ask for part %d of file %s, transfer failed for this part", partID, fileID)
			err = fmt.Errorf("no more peer to ask for part %d of file %s, transfer failed for this part", partID, fileID)
			channelFin <- partID
			wg.Done()
			return err
		}
		peerToAsk := peersWithPart[rand.Intn(len(peersWithPart))]
		transferSuccess, err = AskPeerForPart(transferID, peerToAsk, fileID, partID, currentSite, registre, &partTransferWg, incomingMessagesChannel, outputMessagesChannel)
		if err != nil {
			return err
		}
	}
	// SC handling
	// We update our register to say we have the file
	err = registre.AddPeerHavingPart(currentSite, fileID, partID)
	if err != nil {
		log.Printf("\n[TORRENT] Error while updating register to say we have the file part %s: %v", partID, err)
		return err
	}
	log.Printf("\n[TORRENT] Transfer for part %d of file %s from peer %s succeeded !", partID, fileID, peerToAsk)

	channelFin <- partID
	wg.Done()

	return err

}

// Ask a peer for a file part, timeout if unsuccessful
func AskPeerForPart(transferID string, peerID string, fileID string, partID uint, currentSite string, reg *registre.Registre, wg *sync.WaitGroup, incomingMessagesChannel <-chan Message, outputMessagesChannel chan<- Message) (success bool, err error) {
	log.Print("\n[TORRENT] Asking peer ", peerID, " for part ", partID, " of file ", fileID)
	// Send message asking for shasum
	SendMessageToPeer(AskingForShasum, false, currentSite, transferID, peerID, None, fileID, partID, "", outputMessagesChannel)
	// Wait for response (shasum) or timeout
	select {
	case msg := <-incomingMessagesChannel:
		if msg.TransferRelatedEvent != ReceivingShasum || msg.PartID != partID || msg.FileID != fileID {
			return false, fmt.Errorf("unexpected message: %+v", msg)
		}
		shasum := msg.Content
		// Check if the shasum match our register
		err = HandlePeerRespondingWithShasum(currentSite, peerID, fileID, partID, shasum, reg)
		if err != nil {
			return false, err
		}
	}
	// Send message asking for content
	SendMessageToPeer(AskingForContent, false, currentSite, transferID, peerID, None, fileID, partID, "", outputMessagesChannel)
	// We wait until we receive the content of the file (hopefully...)
	select {
	case msg := <-incomingMessagesChannel:
		if msg.TransferRelatedEvent != ReceivingContent || msg.PartID != partID || msg.FileID != fileID {
			return false, fmt.Errorf("unexpected message: %+v", msg)
		}
		content := msg.Content
		file := reg.GetFileByID(fileID)
		if file == nil {
			return false, fmt.Errorf("file with ID %s not found in register", fileID)
		}
		fileNameWithoutExt := file.Name
		if idx := strings.LastIndex(file.Name, "."); idx != -1 {
			fileNameWithoutExt = file.Name[:idx]
		}
		partFilePath := fmt.Sprintf("%s/%s/parts/%s_part%d", BIN_PATH, currentSite, fileNameWithoutExt, partID-1)
		if err := os.MkdirAll(fmt.Sprintf("%s/%s/parts", BIN_PATH, currentSite), 0755); err != nil {
			return false, fmt.Errorf("could not create parts directory: %v", err)
		}
		err = os.WriteFile(partFilePath, []byte(content), 0644)
		if err != nil {
			log.Printf("\n[TORRENT] Error while writing part: %v", err)
			return false, err
		}
		log.Printf("\n[TORRENT] Saved part file: %s\n", partFilePath)
		return true, nil
	}
}

// Function called by the controller to answer a request without launching a full transfer
func HandlePeerAskingIfWeHavePart(currentSiteID string, peerID string, fileID string, partID uint, transferID string, reg *registre.Registre, outputMessagesChannel chan<- Message) (err error) {
	log.Printf("\n[TORRENT] Peer %s is asking if we have part %d of file %s", peerID, partID, fileID)
	// We check locally if we can find the part in our local storage
	// We get the file name

	filePath, err := reg.CheckIfWeHavePartInOurStorage(currentSiteID, fileID, partID, BIN_PATH)
	if err != nil {
		return err
	}
	// We check for the shasum
	shasum := registre.CalculateShasum(filePath)
	SendMessageToPeer(TransferRelatedMessage, false, currentSiteID, transferID, peerID, ReceivingShasum, fileID, partID, shasum, outputMessagesChannel)

	return nil
}

// Util function to send a message
func SendMessageToPeer(messageType MessageType, deleteMe bool, senderID string, transferID string, targetID string, transferRelatedEvent TransferRelatedEvent, fileID string, partID uint, content string, outputMessagesChannel chan<- Message) {
	message := Message{
		MessageType:          messageType,
		DeleteMe:             deleteMe,
		SenderID:             senderID,
		TransferID:           transferID,
		TargetID:             targetID,
		TransferRelatedEvent: transferRelatedEvent,
		FileID:               fileID,
		PartID:               partID,
		Content:              content,
	}
	log.Printf("\n[TORRENT] Message details:\n Type: %s\n Sender: %s\n Target: %s\n TransferID: %s\n FileID: %s\n PartID: %d\n Content: %s\n", messageName[message.MessageType], message.SenderID, message.TargetID, message.TransferID, message.FileID, message.PartID, message.Content)
	outputMessagesChannel <- message

}

// Function called by the controller to answer a request without launching a full transfer
func HandlePeerAskingForPartContent(currentSiteID string, peerID string, fileID string, partID uint, transferID string, reg *registre.Registre, outputMessagesChannel chan<- Message) (err error) {
	log.Printf("\n[TORRENT] Peer %s is asking for the content of part %d of file %s", peerID, partID, fileID)
	// We check locally if we can find the part in our local storage
	// We get the file name
	filePath, err := reg.CheckIfWeHavePartInOurStorage(currentSiteID, fileID, partID, BIN_PATH)
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
	SendMessageToPeer(TransferRelatedMessage, false, currentSiteID, transferID, peerID, ReceivingContent, fileID, partID, string(filePartContent), outputMessagesChannel)

	return nil
}

// Compare the shasum of the file received and the one expected in the register
func HandlePeerRespondingWithShasum(currentSiteID string, peerID string, fileID string, partID uint, shasum string, reg *registre.Registre) (err error) {
	log.Printf("\n[TORRENT] Peer %s is responding with shasum for part %d of file %s: %s", peerID, partID, fileID, shasum)
	// We check if the shasum is correct by comparing it with the shasum of the part we have in in the register
	// We get the file name
	shasumFromRegister, err := reg.GetShasumOfPart(fileID, partID)
	if err != nil {
		return fmt.Errorf("could not get shasum of part %d of file %s from register: %v", partID, fileID, err)
	}
	log.Printf("\n[TORRENT] Shasum from register for part %d of file %s: %s", partID, fileID, shasumFromRegister)

	if shasumFromRegister != shasum {
		return fmt.Errorf("shasum calculated %s does not match shasum received %s for part %d of file %s", shasumFromRegister, shasum, partID, fileID)
	}
	log.Printf("\n[TORRENT] Shasum for part %d of file %s is correct", partID, fileID)
	return nil
}

// Used when we obtain a critical section to update the register of the others
func SendRegisterUpdateToPeer(senderID, transferID, targetID, fileID string, partID uint, outputMessagesChannel chan<- Message, reg *registre.Registre) error {
	jsonReg, err := reg.ToJSON()
	if err != nil {
		return fmt.Errorf("[TORRENT][ERROR] Erreur sur la transformation du regsitre en JSON : %v\n", err)
	}
	SendMessageToPeer(RegisterUpdate, false, senderID, transferID, targetID, None, fileID, partID, jsonReg, outputMessagesChannel)
	log.Printf("[TORRENT] Send local updated register successfuly\n")
	return nil
}

// Handling a register update message received from a peer, we update our register accordingly
func HandleRegisterUpdateMessage(msg Message, reg *registre.Registre) error {
	remoteReg := registre.NewRegistre()
	err := remoteReg.FromJSON(msg.Content)
	if err != nil {
		return fmt.Errorf("[TORRENT][ERROR] Update du registre: %v", err)
	}
	reg.Merge(remoteReg)
	log.Printf("[TORRENT] Update Register from remote successfuly\n")
	return nil
}
