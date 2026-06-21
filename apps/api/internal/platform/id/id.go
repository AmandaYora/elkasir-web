// Package id menghasilkan ULID (CHAR(26)) urut-waktu, aman dipakai konkuren.
package id

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	mu      sync.Mutex
	entropy = ulid.Monotonic(rand.Reader, 0)
)

// New mengembalikan ULID baru sebagai string 26 karakter.
func New() string {
	mu.Lock()
	defer mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), entropy).String()
}

// Valid memeriksa apakah s adalah ULID yang sah.
func Valid(s string) bool {
	_, err := ulid.ParseStrict(s)
	return err == nil
}
