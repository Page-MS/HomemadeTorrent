package torrentlogic

import (
	"HomemadeTorrent/pkg/registre"
	"fmt"
	"sync"
)

type PartTransferStatus int

const (
	StateIdle PartTransferStatus = iota
	StateNotStarted
	StateError
	StateRetrying
	StateAskedForAvailability
	StateReceivedShasum
	StateAskedForContent
	StateReceivedContent
	StateCompleted
)

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
	partTransfers          []PartTransfer
}

// Main function to start a transfer
// It will then autonomously handle it until it's finished
func StartTransfer(fileID string, currentSite string, reg *registre.Registre) (success bool, error error) {
	fmt.Print("\nStarting transfer for file ID: ", fileID)
	file := reg.GetFileByID(fileID)
	if file == nil {
		return false, fmt.Errorf("file with ID %s not found in register", fileID)
	}
	// We check that the current site is in the register
	if !reg.IsPeerInRegister(currentSite) {
		return false, fmt.Errorf("current site %s not found in register", currentSite)
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

	for i := uint(0); i < file.NumberOfParts; i++ {
		transfer.partsToAskIDs[i] = i
		go StartTransferForPart(fileID, i, currentSite, reg, transfersResultsChannel, &wg)
		wg.Add(1)
	}
	PrintTransferStatus(transfer)
	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		for n := range transfersResultsChannel {
			transfer.partsCompletedIDs = append(transfer.partsCompletedIDs, uint(n))
			transfer.numberOfPartsCompleted++
			PrintTransferStatus(transfer)
			transfer.partsToAskIDs = removeElementFromSlice(transfer.partsToAskIDs, uint(n))
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

func removeElementFromSlice(slice []uint, element uint) []uint {
	newSlice := make([]uint, 0)
	for _, e := range slice {
		if e != element {
			newSlice = append(newSlice, e)
		}
	}
	return newSlice

}
func PrintTransferStatus(transfer *ongoingTransfer) {
	fmt.Printf("\nTransfer status for file %s\n Number of parts to send: %d\n Number of parts completed: %d\n Is receving ? : %t", transfer.file.Name, len(transfer.partsToAskIDs), transfer.numberOfPartsCompleted, transfer.receiving)
	for _, partTransfer := range transfer.partTransfers {
		fmt.Printf("Part %d: %s (peer: %s)\n", partTransfer.partID, stateName[partTransfer.state], partTransfer.peerID)
	}
}

func StartTransferForPart(fileID string, partID uint, currentSite string, registre *registre.Registre, channelFin chan<- int, wg *sync.WaitGroup) (err error) {
	fmt.Printf("\nStarting transfer for part %d", partID)
	channelFin <- int(partID)
	wg.Done()

	return err

}
