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
    StateNotStarted: "not started",
    StateError: "error",
    StateRetrying: "retrying",
    StateAskedForAvailability: "asked for availability",
    StateReceivedShasum: "received shasum",
    StateAskedForContent: "asked for content",
    StateReceivedContent: "received content",
    StateCompleted: "completed"
}

// This file contains the guards of the torrent logic, most of the utility functions are in registre.go

type PartTransfer struct{
	partID uint
	peerID string
	receiving bool
	state PartTransferStatus
}

type ongoingTransfer struct{
	file File
	partsToAskIDs []uint
	partsCompletedIDs []uint
	numberOfPartsCompleted int
	receiving bool
	partTransfers []PartTransfer
}

