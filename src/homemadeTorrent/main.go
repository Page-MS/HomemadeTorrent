package main

import (
	registre "HomemadeTorrent/pkg/registre"
	"fmt"
)

func main() {

	fmt.Print("Hello, World!")
	registre.InitialiseRegistre()
	// Test de la fonction de shasum
	//shasum := registre.CalculateShasum("bin/baseFiles/babiesducks.png")
	//fmt.Printf("Shasum of 'Hello, World!': %s\n", shasum)
	fileparts, err := registre.SplitFile("bin/baseFiles/babiesducks.png", "bin/parts")
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		return
	}
	fmt.Printf("File parts: %v\n", fileparts)
}
