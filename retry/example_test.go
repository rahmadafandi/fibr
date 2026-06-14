// Copyright 2026 Rahmad Afandi. MIT License.

package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rahmadafandi/fibr/retry"
)

// Retry a flaky operation a few times with exponential backoff.
func ExampleDo() {
	attempts := 0
	err := retry.Do(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}, retry.WithAttempts(5), retry.WithDelay(time.Millisecond))

	fmt.Println("attempts:", attempts, "err:", err)
	// Output: attempts: 3 err: <nil>
}
