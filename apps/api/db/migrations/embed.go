// Package migrations menyematkan berkas migrasi SQL (golang-migrate) ke dalam
// binary lewat go:embed, sehingga image distroless yang sama bisa menjalankan
// `api migrate up` tanpa butuh tool/CLI eksternal di runtime.
package migrations

import "embed"

// FS berisi seluruh berkas migrasi *.up.sql / *.down.sql di direktori ini.
//
//go:embed *.sql
var FS embed.FS
