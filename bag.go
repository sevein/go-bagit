package go_bagit

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

var manifestPtn = regexp.MustCompile("manifest-.*\\.txt$")
var tagmanifestPtn = regexp.MustCompile("tagmanifest-.*\\.txt$")

func ValidateBag(bagLocation string, fast bool, complete bool) error {
	errs := []error{}
	storedOxum, err := GetOxum(bagLocation)
	if err != nil {
		log.Printf("- ERROR - %s", err.Error())
		return err
	}

	err = ValidateOxum(bagLocation, storedOxum)
	if err != nil {
		log.Printf("- ERROR - %s", err.Error())
		return err
	}

	if fast == true {
		log.Printf("- INFO - %s valid according to Payload Oxum", bagLocation)
		return nil
	}

	//validate ant manifest files
	bagFiles, err := os.ReadDir(bagLocation)
	if err != nil {
		return err
	}

	dataFiles := map[string]bool{}
	for _, bagFile := range bagFiles {
		if tagmanifestPtn.MatchString(bagFile.Name()) {
			manifestLoc := filepath.Join(bagLocation, bagFile.Name())
			_, e := ValidateManifest(manifestLoc, complete)
			if len(e) > 0 {
				errs = append(errs, e...)
				errorMsgs := gatherErrors(errs, bagLocation)
				return errors.New(errorMsgs)
			}
		}

		if manifestPtn.MatchString(bagFile.Name()) == true {
			manifestLoc := filepath.Join(bagLocation, bagFile.Name())
			entries, e := ValidateManifest(manifestLoc, complete)
			if len(e) > 0 {
				errs = append(errs, e...)
			}
			for path := range entries {
				dataFiles[path] = true
			}
		}

	}

	dataDirName := filepath.Join(bagLocation, "data")
	if err := filepath.WalkDir(dataDirName, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || dataDirName == path {
			return nil
		}
		rel, err := filepath.Rel(bagLocation, path)
		if err != nil {
			return err
		}
		if _, ok := dataFiles[rel]; !ok {
			return fmt.Errorf("%s exists on filesystem but is not in the manifest", rel)
		}
		return nil
	}); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		log.Printf("- INFO - %s is valid", bagLocation)
		return nil
	}

	errorMsgs := gatherErrors(errs, bagLocation)
	return errors.New(errorMsgs)
}

func gatherErrors(errs []error, bagLocation string) string {
	errorMsgs := fmt.Sprintf("- ERROR - %s is invalid: Bag validation failed: ", bagLocation)
	for i, e := range errs {
		errorMsgs = errorMsgs + e.Error()
		if i < len(errs)-1 {
			errorMsgs = errorMsgs + "; "
		}
	}
	log.Println(errorMsgs)
	return errorMsgs
}

func CreateBag(inputDir string, algorithm string, numProcesses int) error {
	//check that input exists and is a directory
	if err := directoryExists(inputDir); err != nil {
		return err
	}

	log.Printf("- INFO - Creating Bag for directory %s", inputDir)

	//create a slice of files
	filesToBag, err := os.ReadDir(inputDir)
	if err != nil {
		return err
	}

	//check there is at least one file to be bagged.
	if len(filesToBag) < 1 {
		errMsg := fmt.Errorf("Could not create a bag, no files present in %s", inputDir)
		log.Println("- ERROR -", errMsg)
		return errMsg
	}

	//create a data directory for payload
	log.Println("- INFO - Creating data directory")
	dataDirName := filepath.Join(inputDir, "data")
	if err := os.Mkdir(dataDirName, 0777); err != nil {
		log.Println("- ERROR -", err)
		return err
	}

	//move the payload files into data dir
	for _, file := range filesToBag {
		originalLocation := filepath.Join(inputDir, file.Name())
		newLocation := filepath.Join(dataDirName, file.Name())
		log.Printf("- INFO - Moving %s to %s", originalLocation, newLocation)
		if err := os.Rename(originalLocation, newLocation); err != nil {
			log.Println("- ERROR -", err.Error())
			return err
		}
	}

	//Generate the manifest
	if err := CreateManifest("manifest", inputDir, algorithm, numProcesses); err != nil {
		return err
	}

	//Generate bagit.txt
	log.Println("- INFO - Creating bagit.txt")
	bagit := CreateBagit()
	bagit.Path = inputDir
	if err := bagit.Serialize(); err != nil {
		return err
	}

	//Generate bag-info.txt
	log.Println("- INFO - Creating bag-info.txt")

	//get the oxum
	oxum, err := CalculateOxum(inputDir)
	if err != nil {
		return err
	}
	bagInfo := CreateBagInfo()
	bagInfo.Tags[StandardTags.PayloadOxum] = oxum.String()
	bagInfo.Path = inputDir
	if err := bagInfo.Serialize(); err != nil {
		return err
	}

	//Generate TagManifest
	if err := CreateTagManifest(inputDir, algorithm, numProcesses); err != nil {
		return err
	}

	//you are done
	return nil
}

func AddFileToBag(bagLocation string, file string) error {
	//check if bag location is valid
	if err := directoryExists(bagLocation); err != nil {
		return err
	}

	//check if source file is valid
	if err := fileExists(file); err != nil {
		return err
	}

	//check if there is already a source file with the same name in the bag
	sourceFileInfo, err := os.Stat(file)
	if err != nil {
		return err
	}
	targetFilePath := filepath.Join(bagLocation, sourceFileInfo.Name())
	log.Println(targetFilePath)
	err = fileExists(targetFilePath)
	if err == nil {
		return fmt.Errorf("- ERROR - cannot create target file %s already exists", targetFilePath)
	}

	//create the target file
	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	//read the source file
	sourceFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	//write the contents of the source file to the target file
	log.Printf("- INFO - copying file %s to %s", file, bagLocation)
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}

	//ensure the new file exists
	if err := fileExists(targetFilePath); err != nil {
		return err
	}
	targetFile.Close()

	//locate any tag manifest files
	bagFiles, err := os.ReadDir(bagLocation)
	if err != nil {
		return err
	}

	for _, bagFile := range bagFiles {
		if tagmanifestPtn.MatchString(bagFile.Name()) {
			//add the file to the tag-manifest
			err := appendToTagManifest(targetFilePath, bagLocation, bagFile.Name())
			if err != nil {
				return err
			}
		}
	}

	//validate the bag
	if err := ValidateBag(bagLocation, false, false); err != nil {
		return err
	}

	return nil
}

func fileExists(file string) error {
	if _, err := os.Stat(file); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		errorMsg := fmt.Errorf("file %s does not exist", file)
		return errorMsg
	} else {
		log.Println("- ERROR - unknown error:", err.Error())
		return err
	}
}

func directoryExists(inputDir string) error {
	if fi, err := os.Stat(inputDir); err == nil {
		if fi.IsDir() == true {
			return nil
		} else {
			errorMsg := fmt.Errorf("- ERROR - input directory %s is not a directory", inputDir)
			return errorMsg
		}
	} else if os.IsNotExist(err) {
		errorMsg := fmt.Errorf("- ERROR - input %s directory does not exist", inputDir)
		return errorMsg
	} else {
		return err
	}
}
