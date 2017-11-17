package edward_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
)

// createWorkingDir creates a directory to work in and changes into that directory.
// Returns a cleanup function.
func createWorkingDir(t *testing.T, testName, testPath string) (string, func()) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	workingPath := path.Join(wd, "testdata", ".working")
	if _, err := os.Stat(workingPath); os.IsNotExist(err) {
		os.Mkdir(workingPath, os.ModePerm)
	}
	workingDir, err := ioutil.TempDir(workingPath, testName)
	if err != nil {
		t.Fatal(err)
	}
	copy_folder(testPath, workingDir)
	return workingDir, func() {
		os.RemoveAll(workingDir)
	}
}

func copy_folder(source string, dest string) (err error) {

	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {

		sourcefilepointer := source + "/" + obj.Name()

		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			err = copy_folder(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			err = copy_file(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}

	}
	return
}

func copy_file(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	return
}
