package registre

import (
	"fmt"
	"testing"
)

func TestCalculateShasum(t *testing.T) {
	//Test de la fonction de shasum
	shasum := CalculateShasum("../../bin/baseFiles/babiesducks.png")
	fmt.Printf("Shasum of 'Hello, World!': %s\n\n\n\n", shasum)
}

func TestSplitFile(t *testing.T) {
	_, err := SplitFile("../../bin/baseFiles/babiesducks.png", "../../bin/parts")
	if err != nil {
		fmt.Printf("In TestSplitFile Error splitting file: %v\n", err)
		return
	}
	//fmt.Printf("File parts: %v\n", fileparts)
}

func TestPutAllFilesFromDirectoryInRegister(t *testing.T) {
	reg := Registre{}
	reg.PutAllFilesFromDirectoryInRegister("../../bin/baseFiles", "../../bin/parts")
	file := reg.GetFileByID(CalculateShasum("../../bin/baseFiles/babiesducks.png"))
	if file == nil {
		t.Fatal("File not found in register")
	}
	fmt.Printf("File found: %s\n", file.Name)
	if file.ID != CalculateShasum("../../bin/baseFiles/babiesducks.png") {
		t.Errorf("Expected ID %s, got %s", CalculateShasum("../../bin/baseFiles/babiesducks.png"), file.ID)
	}
	if file.size != 1124038 { // babiesducks.png actual size (1.1M)
		t.Errorf("Expected size %d, got %d", 1124038, file.size)
	}
	if file.NumberOfParts != 69 { // Number of 16KiB parts for a 1.1M file
		t.Errorf("Expected number of parts %d, got %d", 69, file.NumberOfParts)
	}
	if len(file.FileParts) != int(file.NumberOfParts) {
		t.Errorf("Expected file parts length %d, got %d", file.NumberOfParts, len(file.FileParts))
	}
	for i, part := range file.FileParts {
		expectedPartName := fmt.Sprintf("../../bin/parts/babiesducks_part%d", i)
		if part.parentFileID != expectedPartName {
			t.Errorf("Expected part name %s, got %s", expectedPartName, part.parentFileID)
		}
		if part.filePartSize <= 0 {
			t.Errorf("Expected part size greater than 0, got %d", part.filePartSize)
		}
		if part.filePartShasum == "" {
			t.Error("Expected non-empty shasum for file part")
		}
	}
}

func TestMakeInitialHardcodedRegister(t *testing.T) {
	reg := Registre{}
	MakeInitialHardcodedRegister(&reg, "../../bin/baseFiles", "../../bin/parts")
	reg.PrintRegister()
	if len(reg.files) == 0 {
		t.Error("Expected at least one file in the register, got 0")
	}
	for _, file := range reg.files {
		if file.Name == "" {
			t.Error("Expected non-empty file name, got empty string")
		}
		if file.ID == "" {
			t.Error("Expected non-empty file ID, got empty string")
		}
		if file.size <= 0 {
			t.Errorf("Expected file size greater than 0, got %d", file.size)
		}
		if file.NumberOfParts <= 0 {
			t.Errorf("Expected number of parts greater than 0, got %d", file.NumberOfParts)
		}
		if len(file.FileParts) != int(file.NumberOfParts) {
			t.Errorf("Expected file parts length %d, got %d", file.NumberOfParts, len(file.FileParts))
		}
		for _, part := range file.FileParts {
			if part.filePartSize <= 0 {
				t.Errorf("Expected part size greater than 0, got %d", part.filePartSize)
			}
			if part.filePartShasum == "" {
				t.Error("Expected non-empty shasum for file part")
			}
		}
	}
}

func TestInitialiseRegistre(t *testing.T) {
	reg := Registre{}
	MakeInitialHardcodedRegister(&reg, "../../bin/baseFiles", "../../bin/parts")
	InitialiseRegistre("Page", &reg)
	if len(reg.files) == 0 {
		t.Error("Expected at least one file in the register, got 0")
	}
	for _, file := range reg.files {
		if file.Name == "" {
			t.Error("Expected non-empty file name, got empty string")
		}
		if file.ID == "" {
			t.Error("Expected non-empty file ID, got empty string")
		}
		if file.size <= 0 {
			t.Errorf("Expected file size greater than 0, got %d", file.size)
		}
		if file.NumberOfParts <= 0 {
			t.Errorf("Expected number of parts greater than 0, got %d", file.NumberOfParts)
		}
		if len(file.FileParts) != int(file.NumberOfParts) {
			t.Errorf("Expected file parts length %d, got %d", file.NumberOfParts, len(file.FileParts))
		}
		for _, part := range file.FileParts {
			if part.filePartSize <= 0 {
				t.Errorf("Expected part size greater than 0, got %d", part.filePartSize)
			}
			if part.filePartShasum == "" {
				t.Error("Expected non-empty shasum for file part")
			}
		}
	}
}
