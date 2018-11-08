package servicelogs_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yext/edward/instance/servicelogs"
)

func TestFollowLogsExisting(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up

	tmpfn := filepath.Join(dir, "tmpfile")
	f, err := os.Create(tmpfn)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		lineData := servicelogs.LogLine{
			Name:    "MyService",
			Time:    time.Now(),
			Stream:  "stream",
			Message: fmt.Sprint(i),
		}

		jsonContent, err := json.Marshal(lineData)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Fprintln(f, string(jsonContent))
	}

	lf := servicelogs.NewLogFollower(tmpfn)
	lc := lf.Start()
	defer lf.Stop()

	success := make(chan struct{})
	var count int
	go func() {
		for range lc {
			fmt.Println(count)
			count++
			if count == 20 {
				close(success)
			}
		}
	}()

	for i := 10; i < 20; i++ {
		lineData := servicelogs.LogLine{
			Name:    "MyService",
			Time:    time.Now(),
			Stream:  "stream",
			Message: fmt.Sprint(i),
		}

		jsonContent, err := json.Marshal(lineData)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Fprintln(f, string(jsonContent))
	}

	select {
	case <-success:
		return
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting for results")
	}

	if t.Failed() {
		t.Logf("Got %d results", count)
	}

}

func TestFollowLogsWaitForCreation(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	tmpfn := filepath.Join(dir, "tmpfile")

	lf := servicelogs.NewLogFollower(tmpfn)
	lc := lf.Start()
	defer lf.Stop()

	f, err := os.Create(tmpfn)
	if err != nil {
		t.Fatal(err)
	}

	success := make(chan struct{})
	var count int
	go func() {
		for range lc {
			fmt.Println(count)
			count++
			if count == 20 {
				close(success)
				return
			}
		}
	}()

	for i := 0; i < 20; i++ {
		lineData := servicelogs.LogLine{
			Name:    "MyService",
			Time:    time.Now(),
			Stream:  "stream",
			Message: fmt.Sprint(i),
		}

		jsonContent, err := json.Marshal(lineData)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Fprintln(f, string(jsonContent))
		time.Sleep(time.Millisecond)
	}

	select {
	case <-success:
		return
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting for results")
	}

	if t.Failed() {
		t.Logf("Got %d results", count)
	}

}
