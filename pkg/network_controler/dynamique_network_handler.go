package networkcontroler

import (
	"HomemadeTorrent/pkg/parser"
	"log"
	"os"
	"slices"

	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type PipelineInfo struct {
	CatPID  int
	TeePID  int
	PGID    int
	OutFifo string   // le fifo out (out_nodeX)
	InFifos []string // les fifos destinations (in_nodeY...)
}

const (
	START_ASKING_TO_JOIN_NETWORK = "START_ASKING_TO_JOIN_NETWORK"
	ASKING_TO_JOIN_NETWORK       = "ASKING_TO_JOIN_NETWORK"
	NAME_ERROR                   = "NAME_ERROR"
)

func (nc *NetworkControler) AskPeerToJoinNetwork(pMsg parser.Message) {
	fifoPath := "/tmp/network_fifos/in_" + pMsg.Payload

	// Ouvrir le FIFO en écriture
	fifo, err := os.OpenFile(fifoPath, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		log.Printf("[NETWORK_CONTROLER] Impossible d'ouvrir le fifo %s : %v\n", fifoPath, err)
	}
	defer fifo.Close()

	// Écrire le string
	msg := parser.Message{
		Sender: nc.SiteID,
		Dest:   pMsg.Payload,
		Action: ASKING_TO_JOIN_NETWORK,
	}
	encoded, err := parser.Encode(msg)
	if err != nil {
		log.Printf("[NETWORK_CONTROLER] Erreur encodage : %v\n", err)
		return
	}

	_, err = fifo.WriteString(encoded + "\n")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[NETWORK_CONTROLER] Envoie à %s de message : %s\n", fifoPath, encoded)

	// Ajout du site comme voisin
	nc.NbNeighbors++
}

func (nc *NetworkControler) HandlePeerAskingToJoin(pMsg parser.Message) string {
	// On stocke pour traitement ultérieur ou post election
	nc.PeersWaitingToJoin = append(nc.PeersWaitingToJoin, pMsg.Sender)

	// Tenter une election pour etre en charge de l'ajout du site
	response := nc.StartElection()
	if response == "" {
		log.Printf("[ARRIVEE] Le site ne peux pas gerer l'arrivée d'un site, attente de la libération de l'election...\n")
	}
	return response
}

func (nc *NetworkControler) HandleElectionResult() []string {
	var response []string

	fifoOutPath := "/tmp/network_fifos/out_" + nc.SiteID
	fifoInPath := "/tmp/network_fifos/in_" + nc.SiteID

	result := nc.PeersWaitingToJoin[:0]
	for _, newSiteName := range nc.PeersWaitingToJoin {

		newSitefifoInPath := "/tmp/network_fifos/in_" + newSiteName
		newSitefifoOutPath := "/tmp/network_fifos/out_" + newSiteName

		// Verifier l'unicité du nom
		if slices.Contains(nc.Controler.NetworkDirectory.IndexToID, newSiteName) {
			// Renvoyer au site un message d'erreur
			// Ouvrir le FIFO en écriture
			fifo, err := os.OpenFile(newSitefifoInPath, os.O_WRONLY, os.ModeNamedPipe)
			if err != nil {
				log.Printf("[NETWORK_CONTROLER] Impossible d'ouvrir le fifo %s : %v\n", newSitefifoInPath, err)
				result = append(result, newSiteName)
				continue
			}
			defer fifo.Close()

			// Écrire le message
			msg := parser.Message{
				Sender: nc.SiteID,
				Dest:   newSiteName,
				Action: NAME_ERROR,
			}
			encoded, err := parser.Encode(msg)
			if err != nil {
				log.Printf("[NETWORK_CONTROLER] Erreur encodage : %v\n", err)
				result = append(result, newSiteName)
				continue
			}

			_, err = fifo.WriteString(encoded + "\n")
			if err != nil {
				log.Printf("[NETWORK_CONTROLER] Erreur encodage : %v\n", err)
				result = append(result, newSiteName)
				continue
			}
			log.Printf("[NETWORK_CONTROLER] Envoie à %s de message : %s\n", newSitefifoInPath, encoded)
		}

		// Link des nouveaux lien du shell
		// out site -> in new site
		err := AddFifoToLink(fifoOutPath, newSitefifoInPath, false)
		if err != nil {
			log.Printf("[ARRIVE] Erreur lors de link entre fifo out %s et fifo in %s : %v\n", fifoOutPath, newSitefifoInPath, err)
			result = append(result, newSiteName)
			continue
		}

		// out new site -> in site
		err = AddFifoToLink(newSitefifoOutPath, fifoInPath, false)
		if err != nil {
			log.Printf("[ARRIVE] Erreur lors de link entre fifo out %s et fifo in %s : %v\n", newSitefifoOutPath, fifoInPath, err)
			result = append(result, newSiteName)
			continue
		}

		// Appel de fonction update de ce site et des autres
		response = append(response, nc.AddUser(newSiteName, true)...)
	}
	nc.PeersWaitingToJoin = result
	return response
}

// Trouve le pipeline (cat + tee) qui lit un FIFO donné
func FindPipelineForFifo(fifoPath string) (*PipelineInfo, error) {
	// lsof pour trouver le cat qui lit ce fifo
	out, err := exec.Command("lsof", "-F", "pcf", fifoPath).Output()
	if err != nil {
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	// Parser la sortie lsof (-F pcf = fields: pid, command, fd)
	var catPID int
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "ccат") || strings.HasPrefix(line, "ccat") {
			// ligne précédente est le PID
			if i > 0 && strings.HasPrefix(lines[i-1], "p") {
				pid, _ := strconv.Atoi(lines[i-1][1:])
				catPID = pid
			}
		}
	}

	if catPID == 0 {
		return nil, fmt.Errorf("aucun processus cat trouvé pour %s", fifoPath)
	}

	// Récupérer le PGID du cat (= PGID du pipeline)
	pgid, err := syscall.Getpgid(catPID)
	if err != nil {
		return nil, fmt.Errorf("getpgid failed: %w", err)
	}

	// Trouver le tee dans le même PGID
	psOut, err := exec.Command("ps", "-eo", "pid,pgid,cmd").Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
	}

	info := &PipelineInfo{
		CatPID:  catPID,
		PGID:    pgid,
		OutFifo: fifoPath,
	}

	for _, line := range strings.Split(string(psOut), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		linePGID, _ := strconv.Atoi(fields[1])
		cmd := fields[2]

		if linePGID == pgid && strings.Contains(cmd, "tee") {
			info.TeePID = pid
			// Les arguments du tee = les fifos destinations
			info.InFifos = fields[3:]
			// Filtrer le "> /dev/null" éventuel
			filtered := []string{}
			for _, f := range info.InFifos {
				if f != "/dev/null" && f != ">" {
					filtered = append(filtered, f)
				}
			}
			info.InFifos = filtered
			break
		}
	}

	return info, nil
}

// Recrée le pipeline avec un fifo supplémentaire
func AddFifoToLink(fifoPath string, newFifo string, ignoreFirst bool) error {
	info, err := FindPipelineForFifo(fifoPath)
	if err != nil {
		return err
	}

	log.Printf("Pipeline trouvé:\n")
	log.Printf("  cat PID:  %d\n", info.CatPID)
	log.Printf("  tee PID:  %d\n", info.TeePID)
	log.Printf("  PGID:     %d\n", info.PGID)
	log.Printf("  source:   %s\n", info.OutFifo)
	log.Printf("  destinations: %v\n", info.InFifos)

	// Garder en lecture pour eviter que ca plante
	holder, err := os.OpenFile(fifoPath, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("open fifo holder failed: %w", err)
	}
	defer holder.Close()

	// Kill le process group entier (cat + tee)
	if err := syscall.Kill(-info.PGID, syscall.SIGTERM); err != nil {
		return fmt.Errorf("kill pgid failed: %w", err)
	}
	log.Printf("  Pipeline %d tué\n", info.PGID)

	newInFifos := make([]string, 0)
	if ignoreFirst {
		// Remplacer la boucle par le fifo in
		log.Printf("  [IGNORE_FIRST=true] Suppression du premier fifo %s\n", info.InFifos[0])
		newInFifos = append(info.InFifos[1:], newFifo)

	} else {
		// Recréer le pipeline avec le nouveau fifo en plus
		log.Printf("  [IGNORE_FIRST=false] Ajout de %s aux destinations existantes %v\n", newFifo, info.InFifos)
		newInFifos = append(info.InFifos, newFifo)
	}
	log.Printf("  Nouvelles destinations tee: %v\n", newInFifos)

	// Construire la commande: cat src | tee dst1 dst2 ... > /dev/null
	teeArgs := append(newInFifos, "> /dev/null") // on va gérer /dev/null via la commande
	_ = teeArgs

	// Lancer via sh pour gérer la redirection > /dev/null
	cmdStr := fmt.Sprintf("cat %s | tee %s > /dev/null &",
		info.OutFifo,
		strings.Join(newInFifos, " "),
	)

	log.Printf("  Relance: %s\n", cmdStr)

	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("relance failed: %w", err)
	}

	log.Printf("  Nouveau pipeline lancé (PID: %d)\n", cmd.Process.Pid)
	return nil
}
