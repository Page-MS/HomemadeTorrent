package registre

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// 16 KiB file part size (except the last one)

const FILE_PART_SIZE uint = 16 * 1024

type filePart struct {
	parentFileID   string
	filePartID     uint
	filePartSize   uint
	filePartShasum string
}

type file struct {
	name                string
	ID                  string
	size                uint
	peersThatHaveFileID []string
	numberOfParts       uint
	fileParts           []filePart
}

type Registre struct {
	files []file
}

func CalculateShasum(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%x", h.Sum(nil))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func SplitFile(filePath string, destination string) ([]filePart, error) {

	// We get the name of the file
	fileName := filePath[strings.LastIndex(filePath, "/")+1:]
	// Remove file extension
	fileNameWithoutExt := fileName[:strings.LastIndex(fileName, ".")]
	// We read the size of the file
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not get file info: %v", err)
	}
	fileSize := uint(fileInfo.Size())

	// We calculate the number of parts
	numberOfParts := (fileSize / FILE_PART_SIZE) + 1
	// We create the file parts
	fileParts := make([]filePart, numberOfParts)

	// We read the file and split it into parts
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	//if the parts destination folder does not exist, we create it
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		err = os.Mkdir(destination, 0755)
	}
	if err != nil {
		return nil, fmt.Errorf("could not create destination folder: %v", err)
	}

	for i := uint(0); i < numberOfParts; i++ {
		partSize := FILE_PART_SIZE
		if i == numberOfParts-1 {
			partSize = fileSize - (i * FILE_PART_SIZE)
		}
		// get the content of the file part
		filePartContent := make([]byte, partSize)
		_, err := file.Read(filePartContent)
		if err != nil {
			return nil, fmt.Errorf("could not read file part: %v", err)
		}

		// We create a file in the subfolder destination with the content of the file part
		partFileName := fmt.Sprintf("%s/%s_part%d", destination, fileNameWithoutExt, i)
		err = os.WriteFile(partFileName, filePartContent, 0644)
		if err != nil {
			return nil, fmt.Errorf("could not write file part: %v", err)
		}
		//calculate the shasum of the file part
		filePartShasum := CalculateShasum(partFileName)
		if err != nil {
			return nil, fmt.Errorf("could not calculate shasum: %v", err)
		} else {
			fileParts[i] = filePart{
				parentFileID:   partFileName,
				filePartSize:   partSize,
				filePartShasum: filePartShasum,
			}
		}

	}

	return fileParts, nil
}

func (r *Registre) GetFileByID(fileID string) *file {
	for i, file := range r.files {
		if file.ID == fileID {
			return &r.files[i]
		}
	}
	fmt.Printf("File with ID %s not found\n", fileID)
	return nil
}

func (r *Registre) PutAllFilesFromDirectoryInRegister(source string, destination string) {
	files, err := os.ReadDir(source)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}
	for _, fileTreated := range files {
		if !fileTreated.IsDir() {
			filePath := source + "/" + fileTreated.Name()
			fileParts, err := SplitFile(filePath, destination)
			if err != nil {
				fmt.Printf("Error splitting file: %v\n", err)
				continue
			}
			fileInfo, _ := fileTreated.Info()
			fileSize := fileInfo.Size()
			newFile := file{
				name:          fileTreated.Name(),
				ID:            CalculateShasum(filePath),
				size:          uint(fileSize),
				numberOfParts: uint(len(fileParts)),
				fileParts:     fileParts,
			}
			r.AddFile(newFile)
			initialisationFileCopy(newFile, "site1")
		}
	}
}

func initialisationFileCopy(file file, siteID string) {
	fileURL := "bin/baseFiles/" + file.name
	filecontent, err := os.ReadFile(fileURL)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	err = os.WriteFile("bin/"+siteID+"-"+file.name, filecontent, 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}
}

func (r *Registre) AddFile(file file) {
	r.files = append(r.files, file)
}

func InitialiseRegistre() {
	fmt.Print("Initialise Registre")
}
