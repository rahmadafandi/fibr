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
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

// GenerateUniqueSlug creates a unique slug for any table
func Generate(db *gorm.DB, tableName string, title string) (string, error) {
	baseSlug := slug.Make(title)

	// Keep trying until we get a unique slug
	for {
		// Generate random suffix
		randomBytes := make([]byte, 10)
		if _, err := rand.Read(randomBytes); err != nil {
			return "", err
		}

		// Encode using base32 for shorter, URL-safe string
		// Remove padding and convert to lowercase
		uniqueID := strings.ToLower(strings.TrimRight(base32.StdEncoding.EncodeToString(randomBytes), "="))

		slugCandidate := baseSlug + "-" + uniqueID

		// Check if slug exists
		var count int64
		err := db.Table(tableName).Where("slug = ?", slugCandidate).Count(&count).Error
		if err != nil {
			return "", fmt.Errorf("error checking slug uniqueness: %w", err)
		}

		if count == 0 {
			return slugCandidate, nil
		}
		// If exists, loop will continue and generate a new random suffix
	}
}