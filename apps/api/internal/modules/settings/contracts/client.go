// Package settingsclient adalah KONTRAK modul settings (konfigurasi toko: kontrol diskon,
// fitur, pajak & layanan). Modul lain (selforder, transaction) MEMBACA setting lewat sini —
// tidak menyentuh tabel settings langsung.
package settingsclient

import "context"

// Settings adalah konfigurasi toko yang relevan lintas-modul. Persen disimpan integer
// (mis. TaxPercent=11 berarti 11%).
type Settings struct {
	MaxDiscountPercent    int32
	MaxOperationalExpense int64
	CashVarianceTolerance int64
	FeatureSelfOrder      bool
	FeatureQris           bool
	FeaturePayAtCashier   bool
	TaxEnabled            bool
	TaxPercent            int32
	ServicePercent        int32
}

// Client adalah kontrak baca yang dipublikasikan modul settings. Get selalu mengembalikan
// nilai (default aman bila baris belum ada), bukan error not-found.
type Client interface {
	Get(ctx context.Context, storeID string) (Settings, error)
}
