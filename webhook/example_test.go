// Copyright 2026 Rahmad Afandi. MIT License.

package webhook_test

import (
	"fmt"
	"time"

	"github.com/rahmadafandi/fibr/webhook"
)

// Sign a payload, then verify it with the same secret.
func ExampleVerify() {
	secret := "whsec_example"
	payload := []byte(`{"event":"ok"}`)

	header := webhook.Sign(payload, secret, time.Now())
	err := webhook.Verify(payload, header, secret, webhook.DefaultTolerance)

	fmt.Println("valid:", err == nil)
	// Output:
	// valid: true
}
