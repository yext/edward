package edward_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/yext/edward/common"
	"github.com/yext/edward/home"

	"github.com/yext/edward/edward"
)

func createClient(configFile, testName, testPath string) (*edward.Client, string, func(), error) {
	// Copy test content into a temp dir on the GOPATH & defer deletion
	wd, cleanup, err := createWorkingDir(testName, testPath)
	if err != nil {
		return nil, "", func() {}, err
	}

	dirConfig := &home.EdwardConfiguration{}
	err = dirConfig.InitializeWithDir(path.Join(wd, "edwardHome"))
	if err != nil {
		return nil, "", func() {}, err
	}

	client, err := edward.NewClientWithConfig(
		path.Join(wd, configFile),
		common.EdwardVersion,
	)
	if err != nil {
		return nil, "", func() {}, err
	}

	client.DirConfig = dirConfig
	client.EdwardExecutable = edwardExecutable
	client.DisableConcurrentPhases = true
	client.WorkingDir = wd
	client.Tags = []string{fmt.Sprintf("test.%d", time.Now().UnixNano())}

	return client, wd, cleanup, nil
}

// createWorkingDir creates a directory to work in and changes into that directory.
// Returns a cleanup function.
func createWorkingDir(testName, testPath string) (string, func(), error) {
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
		return "", func() {}, err
	}
	copy_folder(testPath, workingDir)
	return workingDir, func() {
		os.RemoveAll(workingDir)
	}, nil
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
				fmt.Println("Copying folder:", err)
			}
		} else {
			err = copy_file(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println("Copying file:", err)
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
