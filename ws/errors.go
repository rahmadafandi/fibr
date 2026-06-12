// Copyright 2026 Rahmad Afandi. MIT License.

package ws

import "errors"

// ErrClosed is returned when sending on a closed connection.
var ErrClosed = errors.New("ws: connection closed")

// ErrSlowClient is returned when a connection's send buffer is full; the
// connection is closed and dropped.
var ErrSlowClient = errors.New("ws: slow client dropped")
