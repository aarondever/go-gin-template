package util

import "github.com/google/uuid"

// NewID mints an identifier for a stored entity. Ids double as sort keys, so
// UUIDv7 is used: its timestamp prefix makes them ordered by creation time.
func NewID() string { return uuid.Must(uuid.NewV7()).String() }
