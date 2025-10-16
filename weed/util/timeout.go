package util

import (
	"fmt"
	"time"
)

// Execute executes a function that returns an error with a specified timeout.
func Execute(fn func() error, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("operation timed out after %v", timeout)
	}
}
