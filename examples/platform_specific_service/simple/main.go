// A simple executable that stays runnning until an interrupt is received
// Based on: https://gobyexample.com/signals
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = <-sigs
		done <- true
	}()

	fmt.Println("Waiting for signal")
	<-done
	fmt.Println("Exiting")
}
