// A simple service that pauses before sending HTTP responses
package main

import (
	"fmt"
	"net/http"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// Pause to force the user to wait
	time.Sleep(time.Second * 5)
	fmt.Fprintf(w, "Hope that didn't time out!")
}

func main() {
	http.HandleFunc("/", handler)
	fmt.Println("Starting to listen on port 8080")
	http.ListenAndServe(":8080", nil)
}
