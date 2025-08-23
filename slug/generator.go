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
