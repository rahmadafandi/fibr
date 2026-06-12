// Copyright 2026 Rahmad Afandi. MIT License.

// Package sse serves Server-Sent Events: a one-way text/event-stream from
// server to client. Handler wraps a streaming writer; the supplied function
// owns the send loop and returns to end the stream.
package sse

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Stream writes SSE frames to a single client connection.
type Stream struct {
	w *bufio.Writer
}

// Event is a fully specified SSE event. Data is JSON-encoded unless it is a
// string or []byte, which are written verbatim.
type Event struct {
	ID    string
	Name  string
	Data  any
	Retry int // reconnection hint, milliseconds
}

// Handler returns a Fiber handler that streams events. fn is called with a
// Stream; when fn returns, the stream closes. fn owns its own loop and should
// stop when a Send returns an error (client gone) or the request context is
// done.
func Handler(fn func(c *fiber.Ctx, s *Stream)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/event-stream")
		c.Set(fiber.HeaderCacheControl, "no-cache")
		c.Set(fiber.HeaderConnection, "keep-alive")
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			fn(c, &Stream{w: w})
		})
		return nil
	}
}

// Send writes a named event with JSON-encoded data and flushes.
func (s *Stream) Send(event string, data any) error {
	return s.Event(Event{Name: event, Data: data})
}

// SendRaw writes a named event with a pre-formatted string payload.
func (s *Stream) SendRaw(event, data string) error {
	return s.Event(Event{Name: event, Data: raw(data)})
}

// Comment writes an SSE comment line (": text"), commonly used as a keepalive.
func (s *Stream) Comment(text string) error {
	if _, err := fmt.Fprintf(s.w, ": %s\n", text); err != nil {
		return err
	}
	return s.w.Flush()
}

// Event writes a fully specified event and flushes. A flush error indicates the
// client disconnected.
func (s *Stream) Event(e Event) error {
	if e.ID != "" {
		if _, err := fmt.Fprintf(s.w, "id: %s\n", e.ID); err != nil {
			return err
		}
	}
	if e.Name != "" {
		if _, err := fmt.Fprintf(s.w, "event: %s\n", e.Name); err != nil {
			return err
		}
	}
	if e.Retry > 0 {
		if _, err := fmt.Fprintf(s.w, "retry: %d\n", e.Retry); err != nil {
			return err
		}
	}
	payload, err := encode(e.Data)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(payload, "\n") {
		if _, err := fmt.Fprintf(s.w, "data: %s\n", line); err != nil {
			return err
		}
	}
	if _, err := s.w.WriteString("\n"); err != nil {
		return err
	}
	return s.w.Flush()
}

type raw string

func encode(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", nil
	case raw:
		return string(x), nil
	case string:
		return x, nil
	case []byte:
		return string(x), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
