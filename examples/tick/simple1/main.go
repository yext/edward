// A simple executable that stays runnning until an interrupt is received
// Based on: https://gobyexample.com/signals
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello: %s!", r.URL.Path[1:])
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Starting to listen on port", os.Args[1])

	go func() {
		c := time.Tick(1 * time.Second)
		for now := range c {
			fmt.Printf("%v %s\n", now, "Tick")
		}
	}()

	http.ListenAndServe(":"+os.Args[1], nil)
}
