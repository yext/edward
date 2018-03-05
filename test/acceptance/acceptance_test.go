package acceptance

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

// Path to the Edward executable as built
var edwardExecutable string

const workingPath = "testdata/.working"

func TestMain(m *testing.M) {
	// Enable acceptance tests with a flag
	acceptanceEnabled := flag.Bool("edward.acceptance", false, "Enable acceptance tests for Edward.")
	flag.Parse()
	if acceptanceEnabled == nil || !*acceptanceEnabled {
		log.Println("Acceptance tests disabled")
		return
	}

	buildDir, err := ioutil.TempDir("", "edwardTest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(buildDir)

	edwardExecutable = path.Join(buildDir, "edward")

	cmd := exec.Command("go", "build", "-o", edwardExecutable, "github.com/yext/edward")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Create working dir as needed
	if _, err := os.Stat(workingPath); os.IsNotExist(err) {
		os.Mkdir(workingPath, os.ModePerm)
	}
	defer func() {
		err := os.Remove(workingPath)
		if err != nil {
			log.Fatal(err)
		}
	}()

	os.Exit(m.Run())
}

func executeCommand(t *testing.T, workingDir string, command string, arg ...string) {
	t.Helper()

	cmd := exec.Command(command, arg...)
	cmd.Dir = workingDir
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Fatal(err)
	}
}

func expectErrorFromURL(t *testing.T, url string) {
	t.Helper()

	_, err := getFromURL(url)
	if err == nil {
		t.Error("expected an error when service stopped")
	}
}

func expectFromURL(t *testing.T, expected string, url string) {
	t.Helper()

	content, err := getFromURL(url)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(content, expected) {
		t.Errorf("Response incorrect. Expected '%s', got '%s'", expected, content)
	}
}

func getFromURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), nil
}

// createWorkingDir creates a directory to work in and changes into that directory.
// Returns a cleanup function.
func createWorkingDir(testName, testDataPath string) (string, func(), error) {
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
	copyFolder(testDataPath, workingDir)
	return workingDir, func() {
		os.RemoveAll(workingDir)
	}, nil
}

func copyFolder(source string, dest string) (err error) {

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
			err = copyFolder(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println("Copying folder:", err)
			}
		} else {
			err = copyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println("Copying file:", err)
			}
		}

	}
	return
}

func copyFile(source string, dest string) (err error) {
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
