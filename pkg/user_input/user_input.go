package userInput

import (
	"log"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	START_SNAPSHOT string = "START_SNAPSHOT"
)

func CreateFIFOInput(siteID string) {
	path := strings.TrimSpace("/tmp/site_input_" + siteID)

	os.Remove(path)
	err := unix.Mkfifo(path, 0666)
	if err != nil {
		log.Fatalf("[USER_INPUT] Impossible de créer le fifo pour l'entrée utilisateur\n")
	}
}

func GetInputFile(siteID string) *os.File {
	f, err := os.OpenFile("/tmp/site_input_"+siteID, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		log.Fatal(err)
	}
	return f
}
