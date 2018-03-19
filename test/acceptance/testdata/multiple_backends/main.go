// A simple executable that stays runnning until an interrupt is received
// Based on: https://gobyexample.com/signals
package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	fmt.Println("Success")

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
