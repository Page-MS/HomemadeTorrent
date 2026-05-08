package registre

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strings"
)

// 16 KiB file part size (except the last one)

const FILE_PART_SIZE uint = 3 //16 * 1024

type FilePart struct {
	ParentFileID            string   `json:"parent_file_id"`
	FilePartID              uint     `json:"file_part_id"`
	FilePartSize            uint     `json:"file_part_size"`
	FilePartShasum          string   `json:"file_part_shasum"`
	PeersThatHaveFilePartID []string `json:"peers_with_part"`
}

type File struct {
	Name                string     `json:"name"`
	ID                  string     `json:"id"`
	Size                uint       `json:"size"`
	PeersThatHaveFileID []string   `json:"peers_with_file"`
	NumberOfParts       uint       `json:"number_of_parts"`
	FileParts           []FilePart `json:"file_parts"`
}

type Registre struct {
	Files []File   `json:"files"`
	Peers []string `json:"peers"`
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

func (r *Registre) IsPeerInRegister(peerID string) bool {
	for _, peer := range r.Peers {
		if peer == peerID {
			return true
		}
	}
	return false
}

// Split a file into parts of size FILE_PART_SIZE and return the informations about the file parts
//
// Parameters:
// - filePath: the path of the file to split
// - destination: the path of the directory where the file parts will be created
//
// Returns:
// - a slice of filePart containing the informations about the file parts, or an error if something went wrong
func SplitFile(filePath string, destination string) ([]FilePart, error) {

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
	NumberOfParts := (fileSize / FILE_PART_SIZE) + 1
	// We create the file parts
	FileParts := make([]FilePart, NumberOfParts)

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

	for i := uint(0); i < NumberOfParts; i++ {
		partSize := FILE_PART_SIZE
		if i == NumberOfParts-1 {
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
			FileParts[i] = FilePart{
				ParentFileID:   partFileName,
				FilePartID:     i + 1, // We start the file part ID at 1 for better readability
				FilePartSize:   partSize,
				FilePartShasum: filePartShasum,
			}
		}

	}

	return FileParts, nil
}

func (r *Registre) GetShasumOfPart(fileID string, partID uint) (string, error) {
	file := r.GetFileByID(fileID)
	if file == nil {
		return "", fmt.Errorf("file with ID %s not found", fileID)
	}

	for _, part := range file.FileParts {
		if part.FilePartID == partID {
			return part.FilePartShasum, nil
		}
	}

	return "", fmt.Errorf("part with ID %d not found in file %s", partID, fileID)
}

// Get the information about a file based on its ID
//
// Parameters:
// - fileID: the ID of the file to get
//
// Returns:
// - a pointer to the file if it is found in the register, nil otherwise
func (r *Registre) GetFileByID(fileID string) *File {
	for i, file := range r.Files {
		if file.ID == fileID {
			return &r.Files[i]
		}
	}
	log.Printf("[REGISTRE] File with ID %s not found\n", fileID)
	return nil
}

func (r *Registre) GetFileByName(fileName string) *File {
	for i, file := range r.Files {
		if file.Name == fileName {
			return &r.Files[i]
		}
	}
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
		log.Printf("[REGISTRE] Error reading directory: %v\n", err)
		return
	}
	for _, fileTreated := range files {
		if !fileTreated.IsDir() {
			filePath := source + "/" + fileTreated.Name()
			FileParts, err := SplitFile(filePath, destination)
			if err != nil {
				log.Printf("[REGISTRE] Error splitting file: %v\n", err)
				continue
			}
			fileInfo, _ := fileTreated.Info()
			fileSize := fileInfo.Size()
			newFile := File{
				Name:          fileTreated.Name(),
				ID:            CalculateShasum(filePath),
				Size:          uint(fileSize),
				NumberOfParts: uint(len(FileParts)),
				FileParts:     FileParts,
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
func initialisationFileCopy(fileInfos File, siteID string) {
	fileURL := "../../bin/baseFiles/" + fileInfos.Name
	filecontent, err := os.ReadFile(fileURL)
	if err != nil {
		log.Printf("[REGISTRE] Error reading file: %v\n", err)
		return
	}
	// We create the fullFiles folder for the site if it does not exist
	if _, err := os.Stat("../../bin/" + siteID); os.IsNotExist(err) {
		err = os.MkdirAll("../../bin/"+siteID, 0755)
		if err != nil {
			log.Printf("[REGISTRE] Error creating fullFiles folder: %v\n", err)
			return
		}
	}

	err = os.WriteFile("../../bin/"+siteID+"/"+fileInfos.Name, filecontent, 0644)
	if err != nil {
		log.Printf("[REGISTRE] Error writing file: %v\n", err)
		return
	}
}

// add the informations of a file in the register
func (r *Registre) AddFile(fileInfos File) {
	r.Files = append(r.Files, fileInfos)
}

// Return the list of peers in the register
func (r *Registre) GetPeerList() []string {
	if len(r.Peers) == 0 {
		log.Printf("[REGISTRE] No peers in the register\n")
		return nil
	}
	return r.Peers
}

// Return the data structure of the files in the register
func (r *Registre) GetFileList() []File {
	if len(r.Files) == 0 {
		log.Printf("[REGISTRE] No files in the register\n")
		return nil
	}
	return r.Files
}

// Get the information about a file part based on the file ID and the file part ID
//
// Parameters:
// - fileID: the ID of the file to which the file part belongs
// - partID: the ID of the file part to get
//
// Returns:
// - a pointer to the file part if it is found in the register, nil otherwise
func (r *Registre) GetFilePart(fileID string, partID uint) *FilePart {
	file := r.GetFileByID(fileID)
	if file == nil {
		log.Printf("[REGISTRE] File with ID %s not found\n", fileID)
		return nil
	}
	for i, part := range file.FileParts {
		if part.FilePartID == partID {
			return &file.FileParts[i]
		}
	}
	log.Printf("[REGISTRE] File part with ID %d not found in file with ID %s\n", partID, fileID)
	return nil
}

// Print the register for debug purposes
func (r *Registre) DetailedPrintRegister() {
	if len(r.Files) == 0 {
		log.Printf("[REGISTRE] No files in the register\n")
		return
	}
	for _, file := range r.Files {
		log.Printf("[REGISTRE] File name: %s, File ID: %s, File size: %d, Number of parts: %d\n ", file.Name, file.ID, file.Size, file.NumberOfParts)
		for _, peer := range file.PeersThatHaveFileID {
			log.Printf("\tPeer that has the file: %s\n", peer)
		}
		for _, part := range file.FileParts {
			log.Printf("\tPart ID: %d, Part size: %d, Part shasum: %s\n", part.FilePartID, part.FilePartSize, part.FilePartShasum)
			for _, peer := range part.PeersThatHaveFilePartID {
				log.Printf("\tPeer that has the file part: %s\n", peer)
			}
		}
	}
}

// Print the register for debug purposes
func (r *Registre) PrintRegister() {
	if len(r.Files) == 0 {
		log.Printf("[REGISTRE] No files in the register\n")
		return
	}
	for _, file := range r.Files {
		log.Printf("[REGISTRE] File name: %s, File ID: %s, File size: %d, Number of parts: %d\n ", file.Name, file.ID, file.Size, file.NumberOfParts)
		for _, peer := range file.PeersThatHaveFileID {
			log.Printf("\tPeer that has the file: %s\n", peer)
		}
	}
}

// Create an empty register
func NewRegistre() *Registre {
	return &Registre{
		Files: []File{},
		Peers: []string{},
	}
}

// Reassemble a file from its parts and save it in the specified path
//
// Parameters:
// - fileID: the ID of the file to reassemble
// - source: the path of the directory containing the file parts
// - destination: the path of the directory where the reassembled file will be created
// - registre: the register containing the file parts information
//
// Returns:
// - an error if something went wrong, nil otherwise
func ReassembleFileFromParts(fileID string, source string, destination string, registre *Registre) error {
	file := registre.GetFileByName(fileID)
	if file == nil {
		return fmt.Errorf("file with ID %s not found in register", fileID)
	}
	// We create the folder if it doesn't exist
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		err = os.Mkdir(destination, 0755)
		if err != nil {
			return fmt.Errorf("could not create destination folder: %v", err)
		}
	}
	outputFile, err := os.Create(destination + "/" + file.Name)
	if err != nil {
		return fmt.Errorf("could not create output file: %v", err)
	}
	defer outputFile.Close()
	// We use a buffer to write each parts in the output file
	// The parts are in the parts subfolder and are named following the convention fileName_partX
	for i := uint(0); i < file.NumberOfParts; i++ {
		partFilePath := fmt.Sprintf(source+"/%s_part%d", file.Name[:strings.LastIndex(file.Name, ".")], i)
		partFileContent, err := os.ReadFile(partFilePath)
		if err != nil {
			return fmt.Errorf("could not read file part: %v", err)
		}
		_, err = outputFile.Write(partFileContent)
		if err != nil {
			return fmt.Errorf("could not write to output file: %v", err)
		}
	}

	return nil
}

func (r *Registre) GetPeersHavingPart(fileID string, partID uint) []string {
	part := r.GetFilePart(fileID, partID)
	if part == nil {
		log.Printf("[REGISTRE] File part with ID %d not found in file with ID %s\n", partID, fileID)
		return nil
	}
	return part.PeersThatHaveFilePartID
}

func (r *Registre) GetFileNameByID(fileID string) string {
	for _, file := range r.Files {
		if file.ID == fileID {
			return file.Name
		}
	}
	log.Printf("[REGISTRE] File with ID %s not found in register\n", fileID)
	return ""
}

func (r *Registre) CheckIfWeHavePartInOurStorage(currentSiteID string, fileID string, partID uint, source string) (string, error) {
	fileName := r.GetFileNameByID(fileID)
	if fileName == "" {
		log.Printf("\n[REGISTRE] File with ID %s not found in register, cannot check if we have part %d", fileID, partID)
		return "", fmt.Errorf("\nfile with ID %s not found in register, cannot check if we have part %d", fileID, partID)
	}
	part := r.GetFilePart(fileID, partID)
	if part == nil {
		log.Printf("\n[REGISTRE] File part with ID %d not found in file with ID %s\n", partID, fileName)
		return "", fmt.Errorf("\nfile part with ID %d not found in file with ID %s", partID, fileName)
	}
	partFilePath := fmt.Sprintf(source+"/%s/parts/%s_part%d", currentSiteID, fileName[:strings.LastIndex(fileName, ".")], partID-1)
	log.Printf("\n[REGISTRE] Checking if we have part file %s\n", partFilePath)
	if _, err := os.Stat(partFilePath); os.IsNotExist(err) {
		return "", nil
	}
	return partFilePath, nil
}

// Initialize and return the initial hardcoded register
//
// Parameters:
// - registre: an empty register to override with the initial hardcoded register
// - sourcePath: the path to the directory containing the source files
// - destinationPath: the path to the directory where the file parts will be stored
func MakeInitialHardcodedRegister(registre *Registre, sourcePath string, destinationPath string, allSiteIDs []string) {
	//peersList := []string{"Mathy", "Alexis", "Noah", "Page"}
	peersList := allSiteIDs
	registre.Peers = peersList
	registre.PutAllFilesFromDirectoryInRegister(sourcePath, destinationPath)
	//CleanUpPartsDirectory()
	// We decide very arbitrary which peers have which files at the begining of the execution of the program
	// TODO: make this more dynamic and less hardcoded
	for i := range registre.GetFileList() {
		if i%4 == 0 {
			registre.Files[i].PeersThatHaveFileID = []string{"1", "2"}
			for part := range registre.Files[i].FileParts {
				registre.Files[i].FileParts[part].PeersThatHaveFilePartID = []string{"1", "2"}
			}
		} else if i%4 == 1 {
			registre.Files[i].PeersThatHaveFileID = []string{"3", "1"}
			for part := range registre.Files[i].FileParts {
				registre.Files[i].FileParts[part].PeersThatHaveFilePartID = []string{"3", "1"}
			}
		} else if i%4 == 2 {
			registre.Files[i].PeersThatHaveFileID = []string{"1", "3"}
			for part := range registre.Files[i].FileParts {
				registre.Files[i].FileParts[part].PeersThatHaveFilePartID = []string{"1", "3"}
			}
		} else {
			registre.Files[i].PeersThatHaveFileID = []string{"2", "1"}
			for part := range registre.Files[i].FileParts {
				registre.Files[i].FileParts[part].PeersThatHaveFilePartID = []string{"2", "1"}
			}
		}
	}

}

// Takes the siteID and intialize the files that the file should have at the beginning of the execution of the program based on the precreated common register
//
// Parameters:
// - currentSiteID: the ID of the current site
// - registre: the common register that contains the information about which files each site should have at the beginning of the execution of the program
func InitialiseRegistre(currentSiteID string, registre *Registre) {
	log.Printf("[REGISTRE] Initialisation du registre pour le site %s\n", currentSiteID)
	// If the site ID is not in the register, we return an error
	if !slices.Contains(registre.Peers, currentSiteID) {
		log.Printf("[REGISTRE] Site ID %s not found in the register\n", currentSiteID)
		return
	}
	// We get the files that the site should have at the beginning of the execution of the program based on the precreated common register
	filesToHave := make([]File, 0)
	for _, file := range registre.GetFileList() {
		if slices.Contains(file.PeersThatHaveFileID, currentSiteID) {
			filesToHave = append(filesToHave, file)
		}
	}
	if len(filesToHave) == 0 {
		log.Printf("[REGISTRE] No files to initialize for site ID %s\n", currentSiteID)
		return
	}
	// We copy the files that the site should have at the beginning of the execution of the program based on the precreated common register from the fullFiles folder to the site folder
	for _, file := range filesToHave {
		initialisationFileCopy(file, currentSiteID)
		SplitFile("../../bin/"+currentSiteID+"/"+file.Name, "../../bin/"+currentSiteID+"/parts")
	}

}

// CLean up the files in bin
//
// Is used between executions or after an intialization of the register to clean up the files in bin and avoid having old files that can interfere with the execution of the program
func CleanUpPartsDirectory() {
	files, err := os.ReadDir("../../bin/parts")
	if err != nil {
		log.Printf("[REGISTRE] Error reading directory: %v\n", err)
		return
	}
	for _, file := range files {
		err := os.Remove("../../bin/parts/" + file.Name())
		if err != nil {
			log.Printf("[REGISTRE] Error removing file: %v\n", err)
			return
		}
	}
	// We delete the subfolder
	err = os.Remove("../../bin/parts")
	if err != nil {
		log.Printf("[REGISTRE] Error removing directory: %v\n", err)
		return
	}
	// We remove the subfolders for each site
	files, err = os.ReadDir("../../bin")
	if err != nil {
		log.Printf("[REGISTRE] Error reading directory: %v\n", err)
		return
	}
	for _, file := range files {
		if file.IsDir() && file.Name() != "baseFiles" {
			err := os.RemoveAll("../../bin/" + file.Name())
			if err != nil {
				log.Printf("[REGISTRE] Error removing directory: %v\n", err)
				return
			}
		}
	}
}

// ToJSON transforme le registre en string (pour le Payload du message)
func (r *Registre) ToJSON() (string, error) {
	bytes, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("erreur lors de la sérialisation JSON : %v", err)
	}
	return string(bytes), nil
}

// FromJSON remplit le registre à partir d'une string JSON
func (r *Registre) FromJSON(data string) error {
	err := json.Unmarshal([]byte(data), r)
	if err != nil {
		return fmt.Errorf("erreur lors de la désérialisation JSON : %v", err)
	}
	return nil
}
