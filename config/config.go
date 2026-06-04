// Copyright 2026 Rahmad Afandi. MIT License.

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var durationType = reflect.TypeOf(time.Duration(0))

// LoadConfig loads configuration from environment variables (and a .env file if
// present) into the struct pointed to by out, using "mapstructure" tags.
// Supported extra tags: "default" (fallback value) and "required" ("true").
// out must be a non-nil pointer to a struct.
func LoadConfig(out any) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}

	envPath := filepath.Join(wd, ".env")
	if err := godotenv.Load(envPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error loading .env file: %w", err)
		}
	}

	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("config: out must be a non-nil pointer to a struct")
	}
	v := rv.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("config: out must be a pointer to a struct")
	}
	t := v.Type()

	var missing []string
	var parseErrs []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			continue
		}

		fieldValue := v.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		key := strings.ToUpper(tag)
		raw := os.Getenv(key)
		if raw == "" {
			if def, ok := field.Tag.Lookup("default"); ok {
				raw = def
			}
		}

		if raw == "" {
			if field.Tag.Get("required") == "true" {
				missing = append(missing, key)
			}
			continue
		}

		if err := setField(fieldValue, raw); err != nil {
			parseErrs = append(parseErrs, fmt.Sprintf("%s: %v", key, err))
		}
	}

	var problems []string
	if len(missing) > 0 {
		problems = append(problems, "missing required config: "+strings.Join(missing, ", "))
	}
	if len(parseErrs) > 0 {
		problems = append(problems, "config parse errors: "+strings.Join(parseErrs, "; "))
	}
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}

	return nil
}

func intBitSize(k reflect.Kind) int {
	switch k {
	case reflect.Int8, reflect.Uint8:
		return 8
	case reflect.Int16, reflect.Uint16:
		return 16
	case reflect.Int32, reflect.Uint32:
		return 32
	case reflect.Int64, reflect.Uint64:
		return 64
	default: // reflect.Int / reflect.Uint -> platform size
		return 0
	}
}

func setField(fv reflect.Value, raw string) error {
	if fv.Type() == durationType {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}
		fv.SetInt(int64(d))
		return nil
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, intBitSize(fv.Kind()))
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, intBitSize(fv.Kind()))
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		bits := 64
		if fv.Kind() == reflect.Float32 {
			bits = 32
		}
		f, err := strconv.ParseFloat(raw, bits)
		if err != nil {
			return err
		}
		fv.SetFloat(f)
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice element type %s", fv.Type().Elem().Kind())
		}
		parts := strings.Split(raw, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		fv.Set(reflect.ValueOf(parts))
	default:
		return fmt.Errorf("unsupported type %s", fv.Kind())
	}

	return nil
}
