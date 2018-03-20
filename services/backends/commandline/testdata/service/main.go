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

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello: %s!", r.URL.Path[1:])
	})

	go func() {
		log.Fatal(http.ListenAndServe(":0", nil))
	}()

	fmt.Println("Started")

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
