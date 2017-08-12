package edward_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

// createWorkingDir creates a directory to work in and changes into that directory.
// Returns a cleanup function.
func createWorkingDir(t *testing.T, testName, testPath string) func() {
	workingDir, err := ioutil.TempDir("", testName)
	if err != nil {
		t.Fatal(err)
	}
	copy_folder(testPath, workingDir)
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(workingDir)
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		err = os.Chdir(oldDir)
		if err != nil {
			t.Fatal(err)
		}
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
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}

	}

	return
}
