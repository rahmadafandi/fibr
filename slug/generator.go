// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package slug

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"github.com/uptrace/bun"
)

const maxSlugAttempts = 10

// Generate creates a unique slug for the given table using a Bun database.
func Generate(ctx context.Context, db *bun.DB, tableName string, title string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("slug: db must not be nil")
	}

	baseSlug := slug.Make(title)

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		randomBytes := make([]byte, 10)
		if _, err := rand.Read(randomBytes); err != nil {
			return "", fmt.Errorf("slug: reading random bytes: %w", err)
		}

		uniqueID := strings.ToLower(strings.TrimRight(base32.StdEncoding.EncodeToString(randomBytes), "="))
		slugCandidate := baseSlug + "-" + uniqueID

		count, err := db.NewSelect().Table(tableName).Where("slug = ?", slugCandidate).Count(ctx)
		if err != nil {
			return "", fmt.Errorf("error checking slug uniqueness: %w", err)
		}

		if count == 0 {
			return slugCandidate, nil
		}
	}

	return "", fmt.Errorf("could not generate a unique slug for %q after %d attempts", title, maxSlugAttempts)
}
