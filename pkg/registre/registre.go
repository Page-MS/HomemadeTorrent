package registre

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
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
	peers []string
}

// Calculate the shasum of a file based on its path
//
// Parameters:
// - filePath: the path of the file to calculate the shasum of
//
// Returns:
// - the shasum of the file as a string, or an error if something went wrong
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

	//fmt.Printf("%x", h.Sum(nil))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Split a file into parts of size FILE_PART_SIZE and return the informations about the file parts
//
// Parameters:
// - filePath: the path of the file to split
// - destination: the path of the directory where the file parts will be created
//
// Returns:
// - a slice of filePart containing the informations about the file parts, or an error if something went wrong
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
				filePartID:     i + 1, // We start the file part ID at 1 for better readability
				filePartSize:   partSize,
				filePartShasum: filePartShasum,
			}
		}

	}

	return fileParts, nil
}

// Get the information about a file based on its ID
//
// Parameters:
// - fileID: the ID of the file to get
//
// Returns:
// - a pointer to the file if it is found in the register, nil otherwise
func (r *Registre) GetFileByID(fileID string) *file {
	for i, file := range r.files {
		if file.ID == fileID {
			return &r.files[i]
		}
	}
	fmt.Printf("File with ID %s not found\n", fileID)
	return nil
}

// Take all of the files of a directory and puts their informations in the register, and split them into parts in the destination folder
//
// Parameters:
//
// source: the path of the directory containing the files to put in the register
//
// destination: the path of the directory where the file parts will be created
//
// This is used at initialization
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
		}
	}
}

// Copy the files to the folder of the current site
//
// Parameters:
//
// - file: the informations of the file to copy
// - siteID: the ID of the current site, used to create the destination path of the file to copy
func initialisationFileCopy(fileInfos file, siteID string) {
	fileURL := "bin/baseFiles/" + fileInfos.name
	filecontent, err := os.ReadFile(fileURL)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	// We create the fullFiles folder for the site if it does not exist
	if _, err := os.Stat("bin/" + siteID); os.IsNotExist(err) {
		err = os.MkdirAll("bin/"+siteID, 0755)
		if err != nil {
			fmt.Printf("Error creating fullFiles folder: %v\n", err)
			return
		}
	}

	err = os.WriteFile("bin/"+siteID+"/"+"fullFiles"+siteID+"_"+fileInfos.name, filecontent, 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}
}

// add the informations of a file in the register
func (r *Registre) AddFile(fileInfos file) {
	r.files = append(r.files, fileInfos)
}

// Return the list of peers in the register
func (r *Registre) GetPeerList() []string {
	if len(r.peers) == 0 {
		fmt.Printf("No peers in the register\n")
		return nil
	}
	return r.peers
}

// Return the data structure of the files in the register
func (r *Registre) GetFileList() []file {
	if len(r.files) == 0 {
		fmt.Printf("No files in the register\n")
		return nil
	}
	return r.files
}

// Get the information about a file part based on the file ID and the file part ID
//
// Parameters:
// - fileID: the ID of the file to which the file part belongs
// - partID: the ID of the file part to get
//
// Returns:
// - a pointer to the file part if it is found in the register, nil otherwise
func (r *Registre) GetFilePart(fileID string, partID uint) *filePart {
	file := r.GetFileByID(fileID)
	if file == nil {
		fmt.Printf("File with ID %s not found\n", fileID)
		return nil
	}
	for i, part := range file.fileParts {
		if part.filePartID == partID {
			return &file.fileParts[i]
		}
	}
	fmt.Printf("File part with ID %d not found in file with ID %s\n", partID, fileID)
	return nil
}

// Print the register for debug purposes
func (r *Registre) PrintRegister() {
	if len(r.files) == 0 {
		fmt.Printf("No files in the register\n")
		return
	}
	for _, file := range r.files {
		fmt.Printf("File name: %s, File ID: %s, File size: %d, Number of parts: %d\n ", file.name, file.ID, file.size, file.numberOfParts)
		for _, peer := range file.peersThatHaveFileID {
			fmt.Printf("\tPeer that has the file: %s\n", peer)
		}
		for _, part := range file.fileParts {
			fmt.Printf("\tPart ID: %d, Part size: %d, Part shasum: %s\n", part.filePartID, part.filePartSize, part.filePartShasum)
		}
	}
}

// Create an empty register
func NewRegistre() *Registre {
	return &Registre{
		files: []file{},
		peers: []string{},
	}
}

// Initialize and return the initial hardcoded register
//
// Parameters:
// - registre: an empty register to override with the initial hardcoded register
// - sourcePath: the path to the directory containing the source files
// - destinationPath: the path to the directory where the file parts will be stored
func MakeInitialHardcodedRegister(registre *Registre, sourcePath string, destinationPath string) {
	peersList := []string{"Mathy", "Alexis", "Noah", "Page"}
	registre.peers = peersList
	registre.PutAllFilesFromDirectoryInRegister(sourcePath, destinationPath)
	CleanUpPartsDirectory()
	// We decide very arbitrary which peers have which files at the begining of the execution of the program
	// TODO: make this more dynamic and less hardcoded
	for i := range registre.GetFileList() {
		if i%4 == 0 {
			registre.files[i].peersThatHaveFileID = []string{"Mathy", "Alexis"}
		} else if i%4 == 1 {
			registre.files[i].peersThatHaveFileID = []string{"Noah", "Page"}
		} else if i%4 == 2 {
			registre.files[i].peersThatHaveFileID = []string{"Mathy", "Noah"}
		} else {
			registre.files[i].peersThatHaveFileID = []string{"Alexis", "Page"}
		}
	}

}

// Takes the siteID and intialize the files that the file should have at the beginning of the execution of the program based on the precreated common register
//
// Parameters:
// - currentSiteID: the ID of the current site
// - registre: the common register that contains the information about which files each site should have at the beginning of the execution of the program
func InitialiseRegistre(currentSiteID string, registre *Registre) {
	fmt.Printf("Initialisation du registre pour le site %s\n", currentSiteID)
	// If the site ID is not in the register, we return an error
	if !slices.Contains(registre.peers, currentSiteID) {
		fmt.Printf("Site ID %s not found in the register\n", currentSiteID)
		return
	}
	// We get the files that the site should have at the beginning of the execution of the program based on the precreated common register
	filesToHave := make([]file, 0)
	for _, file := range registre.GetFileList() {
		if slices.Contains(file.peersThatHaveFileID, currentSiteID) {
			filesToHave = append(filesToHave, file)
		}
	}
	if len(filesToHave) == 0 {
		fmt.Printf("No files to initialize for site ID %s\n", currentSiteID)
		return
	}
	// We copy the files that the site should have at the beginning of the execution of the program based on the precreated common register from the fullFiles folder to the site folder
	for _, file := range filesToHave {
		initialisationFileCopy(file, currentSiteID)
		SplitFile("bin/"+currentSiteID+"/"+"fullFiles"+currentSiteID+"_"+file.name, "bin/"+currentSiteID+"/parts")
	}

}

// CLean up the files in bin
//
// Is used between executions or after an intialization of the register to clean up the files in bin and avoid having old files that can interfere with the execution of the program
func CleanUpPartsDirectory() {
	files, err := os.ReadDir("bin/parts")
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}
	for _, file := range files {
		err := os.Remove("bin/parts/" + file.Name())
		if err != nil {
			fmt.Printf("Error removing file: %v\n", err)
			return
		}
	}
	// We delete the subfolder
	err = os.Remove("bin/parts")
	if err != nil {
		fmt.Printf("Error removing directory: %v\n", err)
		return
	}
	// We remove the subfolders for each site
	files, err = os.ReadDir("bin")
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}
	for _, file := range files {
		if file.IsDir() && file.Name() != "baseFiles" {
			err := os.RemoveAll("bin/" + file.Name())
			if err != nil {
				fmt.Printf("Error removing directory: %v\n", err)
				return
			}
		}
	}
}
