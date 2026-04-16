package main

import (
	registre "HomemadeTorrent/pkg/registre"
	"fmt"
)

func main() {

	fmt.Print("Hello, World!")
	registre.InitialiseRegistre()
	// Test de la fonction de shasum
	shasum := registre.CalculateShasum("bin/baseFiles/babiesducks.png")
	fmt.Printf("Shasum of 'Hello, World!': %s\n", shasum)
}
