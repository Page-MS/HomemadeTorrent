package networkcontroler

import (
	"HomemadeTorrent/pkg/distributed_file"
	"HomemadeTorrent/pkg/parser"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"syscall"
)

const (
	RECEIVE_NODE_LEAVING  = "RECEIVE_NODE_LEAVING"
	START_LEAVING_PROCESS = "START_LEAVING_PROCESS"
	TEST_ENFANTS          = "TEST_ENFANTS"
	PARENT_TEE_UPDATE     = "PARENT_TEE_UPDATE"
	PARENT_TEE_UPDATE_ACK = "PARENT_TEE_UPDATE_ACK"
	GOODBYE               = "GOODBYE"
)

func (nc *NetworkControler) GetChildrenIDsString() string {
	// On fait une string avec les IDs et adresses de nos enfants séparés par des virgules
	string := ""
	for id, address := range nc.NeighborIDsAndAdresses {
		string += id + ":" + address + ","
	}
	return string
}

func (nc *NetworkControler) HandleTeeUpdate(pMsg parser.Message) string {
	msg := parser.Message{
		Sender: nc.SiteID,
		Dest:   pMsg.Sender,
		Action: PARENT_TEE_UPDATE_ACK,
	}

	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[LEAVING] erreur d'encodage : %v\n", err)
		return ""
	}

	nc.NbTeeReceived++

	return encoded
}

func (nc *NetworkControler) ReceiveLeavingProcess(pMsg parser.Message) string {
	var encoded string
	// Si  le sender est dans nos enfants on prends en compte sa demande de départ
	estParent := false
	for id := range nc.NeighborIDsAndAdresses {
		if id == pMsg.Sender {
			estParent = true
			log.Printf("[NETWORK CONTROLER][ReceiveLeavingProcess] Notre enfant %s part.\n", pMsg.Sender)
			// On le retire de nos enfants
			delete(nc.NeighborIDsAndAdresses, id)
			break
		}
	}
	if estParent {
		mapNouveauxEnfants := nc.UnparseLeavingProcessPayload(pMsg.Payload)
		// On ajoute les nouveaux enfants à notre liste d'enfants
		for id, address := range mapNouveauxEnfants {
			nc.NeighborIDsAndAdresses[id] = address
		}
		log.Printf("[NETWORK CONTROLER][ReceiveLeavingProcess] Nouvelle list d'enfants\n")
		nc.LogNeighbors()

		nc.NbNeighbors--

		msg := parser.Message{
			Sender: nc.SiteID,
			Dest:   pMsg.Sender,
			Action: PARENT_TEE_UPDATE,
		}
		var err error
		encoded, err = parser.Encode(msg)
		if err != nil {
			log.Printf("[LEAVING] erreur d'encodage : %v\n", err)
			return ""
		}
	}

	// Update
	nc.RemoveUser(pMsg.Sender)
	return encoded
}

func (nc *NetworkControler) RemoveUser(siteIDToRemove string) {
	log.Printf("[REMOVE_USER] ===== DEBUT suppression de %s =====\n", siteIDToRemove)
	log.Printf("[REMOVE_USER] IDs avant       : %v\n", nc.Controler.NetworkDirectory.IndexToID)
	log.Printf("[REMOVE_USER] IDToIndex avant  : %v\n", nc.Controler.NetworkDirectory.IDToIndex)
	log.Printf("[REMOVE_USER] Vector avant     : %v\n", nc.Controler.Vector.GetCopy())
	log.Printf("[REMOVE_USER] Tab avant        : %v\n", nc.Controler.DistFile.GetCopy())
	log.Printf("[REMOVE_USER] SiteIndex avant  : %d\n", nc.Controler.SiteIndex)

	oldDirectory := nc.Controler.NetworkDirectory
	oldVector := nc.Controler.Vector.GetCopy()
	oldTab := nc.Controler.DistFile.GetCopy()

	// Création de la nouvelle liste d'IDs sans le site à retirer
	newIDs := make([]string, 0, len(oldDirectory.IndexToID)-1)
	for _, id := range oldDirectory.IndexToID {
		if id != siteIDToRemove {
			newIDs = append(newIDs, id)
		}
	}

	// Reconstitution de la map d'indexation
	newIDToIndex := make(map[string]int)
	for i, id := range newIDs {
		newIDToIndex[id] = i
	}

	// Réalignement des anciennes valeurs de l'horloge vectorielle
	newVector := make([]int, len(newIDs))
	for _, id := range newIDs {
		oldIdx := oldDirectory.IDToIndex[id]
		newIdx := newIDToIndex[id]
		newVector[newIdx] = oldVector[oldIdx]
	}

	// Réalignement du tableau distribué
	newTab := make([]distributed_file.TabEntry, len(newIDs))
	for _, id := range newIDs {
		oldIdx := oldDirectory.IDToIndex[id]
		newIdx := newIDToIndex[id]
		newTab[newIdx] = oldTab[oldIdx]
	}

	// Injection des nouvelles structures
	newMyIndex := newIDToIndex[nc.SiteID]
	nc.Controler.Vector.UpdateLayout(newVector, newMyIndex)
	nc.Controler.DistFile.UpdateLayout(newTab, newMyIndex)

	nc.Controler.NetworkDirectory.IndexToID = newIDs
	nc.Controler.NetworkDirectory.IDToIndex = newIDToIndex
	nc.Controler.SiteIndex = newMyIndex

	log.Printf("[REMOVE_USER] ===== FIN suppression de %s =====\n", siteIDToRemove)
	log.Printf("[REMOVE_USER] IDs après        : %v\n", nc.Controler.NetworkDirectory.IndexToID)
	log.Printf("[REMOVE_USER] IDToIndex après  : %v\n", nc.Controler.NetworkDirectory.IDToIndex)
	log.Printf("[REMOVE_USER] Vector après     : %v\n", nc.Controler.Vector.GetCopy())
	log.Printf("[REMOVE_USER] Tab après        : %v\n", nc.Controler.DistFile.GetCopy())
	log.Printf("[REMOVE_USER] SiteIndex après  : %d\n", nc.Controler.SiteIndex)
}

func (nc *NetworkControler) RestartTee() {
	outFifo := fmt.Sprintf("/tmp/network_fifos/out_%s", nc.SiteID)

	// Construire la nouvelle liste de destinations
	inFifos := make([]string, 0, len(nc.NeighborIDsAndAdresses))
	for id, address := range nc.NeighborIDsAndAdresses {
		if id != nc.SiteID {
			inFifos = append(inFifos, address)
		}
	}

	if len(inFifos) == 0 {
		log.Printf("[DEPART_TEE] Aucun voisin, pas de tee à relancer\n")
		return
	}

	// Tuer l'ancien pipeline via PGID
	info, err := FindPipelineForFifo(outFifo)
	if err == nil {
		log.Printf("[DEPART_TEE] Pipeline trouvé:\n")
		log.Printf("[DEPART_TEE]   cat PID:      %d\n", info.CatPID)
		log.Printf("[DEPART_TEE]   tee PID:      %d\n", info.TeePID)
		log.Printf("[DEPART_TEE]   PGID:         %d\n", info.PGID)
		log.Printf("[DEPART_TEE]   source:       %s\n", info.OutFifo)
		log.Printf("[DEPART_TEE]   destinations: %v\n", info.InFifos)

		if err := syscall.Kill(-info.PGID, syscall.SIGTERM); err != nil {
			log.Printf("[DEPART_TEE]   Erreur kill PGID %d : %v\n", info.PGID, err)
		} else {
			log.Printf("[DEPART_TEE]   Pipeline PGID %d tué\n", info.PGID)
		}
	} else {
		log.Printf("[DEPART_TEE] Aucun pipeline trouvé pour %s : %v\n", outFifo, err)
	}

	log.Printf("[DEPART_TEE] Nouvelles destinations: %v\n", inFifos)

	// Relancer dans un nouveau process group (équivalent set -m / set +m)
	script := fmt.Sprintf("cat %s | tee %s > /dev/null", outFifo, strings.Join(inFifos, " "))
	log.Printf("[DEPART_TEE] Relance: %s\n", script)

	cmd := exec.Command("sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	if err := cmd.Start(); err != nil {
		log.Printf("[DEPART_TEE]   Erreur lors du lancement : %v\n", err)
	} else {
		log.Printf("[DEPART_TEE]   Nouveau pipeline lancé (PID: %d)\n", cmd.Process.Pid)
	}
}

func (nc *NetworkControler) UnparseLeavingProcessPayload(payload string) map[string]string {

	mapIds := make(map[string]string)

	// On split la payload pour obtenir les IDs et adresses de nos nouveaux enfants
	parts := strings.Split(payload, ",")
	for _, part := range parts {
		if len(part) == 0 {
			//Pour la fin
			continue
		}
		subParts := strings.Split(part, ":")
		if len(subParts) != 2 {
			log.Printf("[NETWORK CONTROLER][UnparseLeavingProcessPayload] Erreur de format dans la payload de départ d'un site: %s\n", part)
			continue
		}
		id := subParts[0]
		address := subParts[1]
		mapIds[id] = address
	}

	return mapIds
}

func (nc *NetworkControler) CreateLastMessage() string {
	msg := parser.Message{
		Sender: nc.SiteID,
		Dest:   nc.SiteID,
		Action: GOODBYE,
	}

	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[LEAVING] erreur d'encodage : %v\n", err)
		return ""
	}

	return encoded
}
