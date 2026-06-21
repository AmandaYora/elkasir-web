// Package infrastructure: akses tabel settings (milik modul settings) + implementasi
// kontrak settingsclient untuk pembaca lintas-modul.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

// Default aman dipakai saat baris settings belum ada (toko baru / belum di-seed).
var defaultSettings = sqlcgen.Setting{
	MaxDiscountPercent:    10,
	MaxOperationalExpense: 200000,
	CashVarianceTolerance: 5000,
	FeatureSelfOrder:      true,
	FeatureQris:           true,
	TaxEnabled:            false,
	TaxPercent:            11,
	ServicePercent:        2,
}

// Repo menyentuh HANYA tabel settings.
type Repo struct {
	pool *sql.DB
	q    *sqlcgen.Queries
}

func NewRepo(pool *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{pool: pool, q: q} }

// Get mengembalikan settings toko, atau default (dengan StoreID terisi) bila belum ada.
func (r *Repo) Get(ctx context.Context, storeID string) (sqlcgen.Setting, error) {
	st, err := r.q.GetSettingsByStore(ctx, storeID)
	if errors.Is(err, sql.ErrNoRows) {
		d := defaultSettings
		d.StoreID = storeID
		return d, nil
	}
	return st, err
}

func (r *Repo) Upsert(ctx context.Context, p sqlcgen.UpsertSettingsParams) error {
	return r.q.UpsertSettings(ctx, p)
}
