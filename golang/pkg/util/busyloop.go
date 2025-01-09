package util

import (
	"context"
	"time"

	"go.uber.org/zap"
)

func BusyLoop(ctx context.Context, busyloopSecs int) {
	zap.S().Infof("Busy Looping for %v seconds", busyloopSecs)
	done := make(chan bool)

	go func() {
		// start a timer in a separate thread that will signal the busyloop to exit
		time.Sleep(time.Duration(busyloopSecs) * time.Second)
		done <- true
	}()

	// run an infinite loop on this thread
	for {
		select {
		case <-done:
			return
		default:
			// do nothing
		}
	}



}