package util

import (
	"fmt"
	"time"
)

func RunWithTimeout(fn func() error, timeout time.Duration) error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		errCh <- fn()
	}()

	select {
	case err, ok := <-errCh:
		if !ok {
			return fmt.Errorf("channel closed")
		}
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout after %v", timeout)
	}
}
