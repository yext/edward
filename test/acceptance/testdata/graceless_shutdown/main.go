// A simple executable that stays running until an interrupt is received
// Based on: https://gobyexample.com/signals
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	fmt.Println("Graceless shutdown service")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello: %s!", r.URL.Path[1:])
	})

	go func() {
		log.Fatal(http.ListenAndServe(":51234", nil))
	}()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	fmt.Println("Waiting for signal")
	<-done
	fmt.Println("Pretending to do some cleanup")
	time.Sleep(1 * time.Minute)
	fmt.Println("exiting")
}
