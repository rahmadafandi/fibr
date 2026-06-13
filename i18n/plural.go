// Copyright 2026 Rahmad Afandi. MIT License.

package i18n

import "strings"

// pluralCategory returns the CLDR cardinal plural category
// (zero/one/two/few/many/other) for the integer n in locale's base language.
// Languages without a specific rule (or unknown) use the English rule
// (one for n==1, otherwise other). Only integer rules are implemented.
func pluralCategory(locale string, n int) string {
	if n < 0 {
		n = -n
	}
	lang := locale
	if i := strings.IndexByte(lang, '-'); i >= 0 {
		lang = lang[:i]
	}
	lang = strings.ToLower(lang)

	mod10, mod100 := n%10, n%100
	switch lang {
	case "id", "ja", "ko", "zh", "th", "vi": // no plural distinction
		return "other"
	case "fr":
		if n == 0 || n == 1 {
			return "one"
		}
		return "other"
	case "ru", "uk": // East Slavic
		switch {
		case mod10 == 1 && mod100 != 11:
			return "one"
		case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
			return "few"
		default:
			return "many"
		}
	case "pl":
		switch {
		case n == 1:
			return "one"
		case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
			return "few"
		default:
			return "many"
		}
	case "ar":
		switch {
		case n == 0:
			return "zero"
		case n == 1:
			return "one"
		case n == 2:
			return "two"
		case mod100 >= 3 && mod100 <= 10:
			return "few"
		case mod100 >= 11 && mod100 <= 99:
			return "many"
		default:
			return "other"
		}
	default: // en and everything else
		if n == 1 {
			return "one"
		}
		return "other"
	}
}
