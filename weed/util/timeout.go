package util

import (
	"context"
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

// RunWithTimeout chạy fn với thời gian chờ và hủy goroutine nếu timeout.
func RunWithContextTimeout(fn func(ctx context.Context) error, timeout time.Duration) error {
	// Tạo context với thời gian chờ
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // Đảm bảo hủy context để giải phóng tài nguyên

	errCh := make(chan error, 1)

	// Chạy fn trong goroutine với context
	go func() {
		defer close(errCh)
		errCh <- fn(ctx)
	}()

	select {
	case err, ok := <-errCh:
		if !ok {
			return fmt.Errorf("channel closed")
		}
		return err
	case <-ctx.Done():
		return fmt.Errorf("timeout after %v", timeout)
	}
}
