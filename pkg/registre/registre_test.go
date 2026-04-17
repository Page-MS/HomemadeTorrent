package registre

import (
	"fmt"
	"testing"
)

func TestCalculateShasum(t *testing.T) {
	//Test de la fonction de shasum
	shasum := CalculateShasum("bin/baseFiles/babiesducks.png")
	fmt.Printf("Shasum of 'Hello, World!': %s\n", shasum)
}

func TestSplitFile(t *testing.T) {
	fileparts, err := SplitFile("bin/baseFiles/babiesducks.png", "bin/parts")
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		return
	}
	fmt.Printf("File parts: %v\n", fileparts)
}
