// Package infrastructure: akses tabel settings + kolom profil toko di stores (milik modul
// settings) + implementasi kontrak settingsclient untuk pembaca lintas-modul.
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
	FeaturePayAtCashier:   true,
	TaxEnabled:            false,
	TaxPercent:            11,
	ServicePercent:        2,
}

// Repo menyentuh tabel settings, PLUS kolom profil (name/address/phone/logo_url) di tabel
// stores — pengecualian shared-kernel, lihat knowledge/MODULE_MAP.md. Kolom lain di stores
// (type/timezone/currency) tetap di luar tanggung jawab modul ini.
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

// GetStoreProfile mengembalikan identitas toko (name/address/phone/logo_url).
func (r *Repo) GetStoreProfile(ctx context.Context, storeID string) (sqlcgen.GetStoreProfileRow, error) {
	return r.q.GetStoreProfile(ctx, storeID)
}

func (r *Repo) UpdateStoreProfile(ctx context.Context, p sqlcgen.UpdateStoreProfileParams) error {
	return r.q.UpdateStoreProfile(ctx, p)
}
