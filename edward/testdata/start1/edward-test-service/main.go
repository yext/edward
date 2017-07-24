// A simple executable that stays runnning until an interrupt is received
// Based on: https://gobyexample.com/signals
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello: %s!", r.URL.Path[1:])
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Starting to listen on port 51936")
	go func() {
		log.Fatal(http.ListenAndServe(":51936", nil))
	}()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	select {
	case <-stop:
		fmt.Println("Terminated")
	case <-timer.C:
		fmt.Println("Timed out")
	}

	fmt.Println("Exiting")
}
