// Copyright 2026 Rahmad Afandi. MIT License.

package i18n_test

import (
	"fmt"

	"github.com/rahmadafandi/fibr/i18n"
)

func ExampleBundle() {
	b := i18n.New(i18n.WithFallback("en"))
	b.LoadMap("en", map[string]any{
		"welcome": "Hello, {name}!",
		"cart":    map[string]any{"items": map[string]any{"one": "{n} item", "other": "{n} items"}},
	})
	fmt.Println(b.Translate("en", "welcome", i18n.M{"name": "Sam"}))
	fmt.Println(b.Plural("en", "cart.items", 3, nil))
	// Output:
	// Hello, Sam!
	// 3 items
}
