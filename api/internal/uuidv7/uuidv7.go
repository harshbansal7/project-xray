package uuidv7

import (
	"github.com/google/uuid"
)

// New generates a new UUID v7 with temporal ordering
// UUID v7 embeds a 48-bit timestamp for natural time-based sorting
func New() string {
	return uuid.Must(uuid.NewV7()).String()
}
