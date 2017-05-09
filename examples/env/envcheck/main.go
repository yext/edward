package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello: %s!", r.URL.Path[1:])
}

func main() {
	if len(os.Getenv("MUST_SET")) == 0 {
		panic("MUST_SET was not set")
	}

	http.HandleFunc("/", handler)
	fmt.Println("Starting to listen on port 51936")
	err := http.ListenAndServe(":51936", nil)
	if err != nil {
		log.Fatal(err)
	}
}
