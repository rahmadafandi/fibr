// Copyright 2026 Rahmad Afandi. MIT License.

package featureflag_test

import (
	"context"
	"fmt"

	"github.com/rahmadafandi/fibr/featureflag"
)

// Evaluate a flag with a percentage rollout and a user allowlist.
func ExampleFlags_Enabled() {
	f := featureflag.New(featureflag.Rules(map[string]featureflag.Rule{
		"new_checkout": {Percentage: 50, Users: []string{"vip"}},
	}))
	ctx := context.Background()

	// An allowlisted user is always in.
	fmt.Println("vip:", f.Enabled(ctx, "new_checkout", featureflag.Eval{UserID: "vip"}))
	// An unknown flag is off.
	fmt.Println("unknown:", f.Enabled(ctx, "nope", featureflag.Eval{UserID: "vip"}))
	// Output:
	// vip: true
	// unknown: false
}
