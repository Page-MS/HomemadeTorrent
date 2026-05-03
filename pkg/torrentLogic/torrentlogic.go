package torrentlogic

import (
	"HomemadeTorrent/pkg/registre"
	"fmt"
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
func StartTransfer(fileID string, currentSite string, registre *registre.Registre) (success bool, error error) {
	fmt.Print("\nStarting transfer for file ID: ", fileID)
	file := registre.GetFileByID(fileID)
	if file == nil {
		return false, fmt.Errorf("file with ID %s not found in register", fileID)
	}
	// We check that the current site is in the register
	if !registre.IsPeerInRegister(currentSite) {
		return false, fmt.Errorf("current site %s not found in register", currentSite)
	}
	transfer := &ongoingTransfer{
		file:                   *file,
		partsToAskIDs:          make([]uint, file.NumberOfParts),
		partsCompletedIDs:      make([]uint, 0),
		numberOfPartsCompleted: 0,
		receiving:              true,
	}
	for i := uint(0); i < file.NumberOfParts; i++ {
		transfer.partsToAskIDs[i] = i
	}
	PrintTransferStatus(transfer)
	return true, nil
}

func PrintTransferStatus(transfer *ongoingTransfer) {
	fmt.Printf("\nTransfer status for file %s\n Number of parts to send: %d\n Number of parts completed: %d\n Is receving ? : %t", transfer.file.Name, len(transfer.partsToAskIDs), transfer.numberOfPartsCompleted, transfer.receiving)
	for _, partTransfer := range transfer.partTransfers {
		fmt.Printf("Part %d: %s (peer: %s)\n", partTransfer.partID, stateName[partTransfer.state], partTransfer.peerID)
	}
}
