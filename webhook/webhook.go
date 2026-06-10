// Copyright 2026 Rahmad Afandi. MIT License.

// Package webhook signs and verifies webhook payloads with an HMAC-SHA256
// scheme: the signature header is "t=<unix>,v1=<hex>" where the MAC is computed
// over "<unix>.<payload>". The timestamp enables replay protection.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultHeader is the conventional signature header name.
	DefaultHeader = "Webhook-Signature"
	// DefaultTolerance is the default allowed clock skew for the timestamp.
	DefaultTolerance = 5 * time.Minute
)

var (
	// ErrMalformedHeader means the signature header could not be parsed.
	ErrMalformedHeader = errors.New("webhook: malformed signature header")
	// ErrExpired means the signature timestamp is outside the tolerance window.
	ErrExpired = errors.New("webhook: signature timestamp outside tolerance")
	// ErrInvalidSignature means the signature did not match the payload.
	ErrInvalidSignature = errors.New("webhook: signature mismatch")
)

func mac(payload []byte, secret string, unix int64) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	// signed string is "<unix>.<payload>"
	h.Write([]byte(strconv.FormatInt(unix, 10)))
	h.Write([]byte("."))
	h.Write(payload)
	return h.Sum(nil)
}

// Sign returns the signature header value for payload at time t.
func Sign(payload []byte, secret string, t time.Time) string {
	unix := t.Unix()
	return fmt.Sprintf("t=%d,v1=%s", unix, hex.EncodeToString(mac(payload, secret, unix)))
}

// Verify checks header against payload using secret. If tolerance > 0, the
// timestamp must be within tolerance of now. Returns nil when valid, otherwise
// ErrMalformedHeader, ErrExpired, or ErrInvalidSignature.
func Verify(payload []byte, header, secret string, tolerance time.Duration) error {
	var tsStr, sig string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			tsStr = v
		case "v1":
			sig = v
		}
	}
	if tsStr == "" || sig == "" {
		return ErrMalformedHeader
	}

	unix, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return ErrMalformedHeader
	}

	if tolerance > 0 {
		delta := time.Since(time.Unix(unix, 0))
		if delta < 0 {
			delta = -delta
		}
		if delta > tolerance {
			return ErrExpired
		}
	}

	got, err := hex.DecodeString(sig)
	if err != nil {
		return ErrInvalidSignature
	}

	if !hmac.Equal(mac(payload, secret, unix), got) {
		return ErrInvalidSignature
	}
	return nil
}
