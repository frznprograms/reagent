package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// tokenEntry represents the metadata associated with a generated
// token. Each token is valid only for a single installation call, which comes
// with a package manager and packages. The token expires after 5 mins.
type tokenEntry struct {
	packages  []string
	manager   string
	expiresAt time.Time
}

// tokenMu is a mutex, which guards the tokenStore from being modified
// by multiple goroutines at once. tokenStore itself only stores one token
// at a time.
// NOTE: mutex may be removed, as there are no goroutines yet.
var (
	tokenMu    sync.Mutex
	tokenStore = make(map[string]tokenEntry)
)

// generateToken generates a secure token to use by the installation tool.
// By generating a token, it ensures the LLM cannot fabricate authorisation
// to skip checking if the installations are safe before actually doing them.
func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
