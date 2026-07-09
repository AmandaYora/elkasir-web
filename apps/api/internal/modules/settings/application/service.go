// Package application: use case modul settings — baca & perbarui konfigurasi toko
// (kontrol diskon, fitur, pajak & layanan) dan profil identitas toko (nama/telepon/alamat/
// logo). Menyentuh tabel settings + kolom profil di stores (via repo).
package application

import (
	"context"
	"database/sql"
	"strings"

	"github.com/elkasir/api/internal/modules/settings/infrastructure"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
)

type Service struct {
	repo *infrastructure.Repo
}

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// DTO adalah representasi settings untuk admin (camelCase via JSON tags).
type DTO struct {
	StoreName             string `json:"storeName"`
	StorePhone            string `json:"storePhone"`
	StoreAddress          string `json:"storeAddress"`
	StoreLogoUrl          string `json:"storeLogoUrl"`
	MaxDiscountPercent    int32  `json:"maxDiscountPercent"`
	MaxOperationalExpense int64  `json:"maxOperationalExpense"`
	CashVarianceTolerance int64  `json:"cashVarianceTolerance"`
	FeatureSelfOrder      bool   `json:"featureSelfOrder"`
	FeatureQris           bool   `json:"featureQris"`
	FeaturePayAtCashier   bool   `json:"featurePayAtCashier"`
	TaxEnabled            bool   `json:"taxEnabled"`
	TaxPercent            int32  `json:"taxPercent"`
	ServicePercent        int32  `json:"servicePercent"`
}

// Input adalah payload PATCH /settings (semua field wajib — admin mengirim objek penuh).
type Input struct {
	StoreName             string `json:"storeName"`
	StorePhone            string `json:"storePhone"`
	StoreAddress          string `json:"storeAddress"`
	StoreLogoUrl          string `json:"storeLogoUrl"`
	MaxDiscountPercent    int32  `json:"maxDiscountPercent"`
	MaxOperationalExpense int64  `json:"maxOperationalExpense"`
	CashVarianceTolerance int64  `json:"cashVarianceTolerance"`
	FeatureSelfOrder      bool   `json:"featureSelfOrder"`
	FeatureQris           bool   `json:"featureQris"`
	FeaturePayAtCashier   bool   `json:"featurePayAtCashier"`
	TaxEnabled            bool   `json:"taxEnabled"`
	TaxPercent            int32  `json:"taxPercent"`
	ServicePercent        int32  `json:"servicePercent"`
}

func (s *Service) Get(ctx context.Context, storeID string) (DTO, error) {
	st, err := s.repo.Get(ctx, storeID)
	if err != nil {
		return DTO{}, err
	}
	profile, err := s.repo.GetStoreProfile(ctx, storeID)
	if err != nil {
		return DTO{}, err
	}
	return toDTO(st, profile), nil
}

func (s *Service) Update(ctx context.Context, storeID string, in Input) (DTO, error) {
	if strings.TrimSpace(in.StoreName) == "" {
		return DTO{}, httpx.Validation("Nama toko wajib diisi.")
	}
	if in.MaxDiscountPercent < 0 || in.MaxDiscountPercent > 100 {
		return DTO{}, httpx.Validation("Diskon maksimum harus 0–100%.")
	}
	if in.TaxPercent < 0 || in.TaxPercent > 100 {
		return DTO{}, httpx.Validation("PPN harus 0–100%.")
	}
	if in.ServicePercent < 0 || in.ServicePercent > 100 {
		return DTO{}, httpx.Validation("Biaya layanan harus 0–100%.")
	}
	if in.MaxOperationalExpense < 0 || in.CashVarianceTolerance < 0 {
		return DTO{}, httpx.Validation("Nilai ambang tidak boleh negatif.")
	}
	// Saat self-order aktif, minimal satu metode pembayaran harus aktif — kalau keduanya
	// mati, pelanggan tak punya cara membayar (self-order jadi mubazir).
	if in.FeatureSelfOrder && !in.FeatureQris && !in.FeaturePayAtCashier {
		return DTO{}, httpx.Validation("Minimal satu metode pembayaran (QRIS atau bayar di kasir) harus aktif saat self-order aktif.")
	}

	// Pertahankan id baris yang ada (upsert membuat baris baru bila belum ada).
	cur, err := s.repo.Get(ctx, storeID)
	if err != nil {
		return DTO{}, err
	}
	rowID := cur.ID
	if rowID == "" {
		rowID = id.New()
	}

	if err := s.repo.Upsert(ctx, sqlcgen.UpsertSettingsParams{
		ID: rowID, StoreID: storeID,
		MaxDiscountPercent:    in.MaxDiscountPercent,
		MaxOperationalExpense: in.MaxOperationalExpense,
		CashVarianceTolerance: in.CashVarianceTolerance,
		FeatureSelfOrder:      in.FeatureSelfOrder,
		FeatureQris:           in.FeatureQris,
		FeaturePayAtCashier:   in.FeaturePayAtCashier,
		TaxEnabled:            in.TaxEnabled,
		TaxPercent:            in.TaxPercent,
		ServicePercent:        in.ServicePercent,
	}); err != nil {
		return DTO{}, err
	}
	if err := s.repo.UpdateStoreProfile(ctx, sqlcgen.UpdateStoreProfileParams{
		ID:      storeID,
		Name:    strings.TrimSpace(in.StoreName),
		Address: toNullString(in.StoreAddress),
		Phone:   toNullString(in.StorePhone),
		LogoUrl: toNullString(in.StoreLogoUrl),
	}); err != nil {
		return DTO{}, err
	}
	return s.Get(ctx, storeID)
}

func toDTO(st sqlcgen.Setting, profile sqlcgen.GetStoreProfileRow) DTO {
	return DTO{
		StoreName:             profile.Name,
		StorePhone:            profile.Phone.String,
		StoreAddress:          profile.Address.String,
		StoreLogoUrl:          profile.LogoUrl.String,
		MaxDiscountPercent:    st.MaxDiscountPercent,
		MaxOperationalExpense: st.MaxOperationalExpense,
		CashVarianceTolerance: st.CashVarianceTolerance,
		FeatureSelfOrder:      st.FeatureSelfOrder,
		FeatureQris:           st.FeatureQris,
		FeaturePayAtCashier:   st.FeaturePayAtCashier,
		TaxEnabled:            st.TaxEnabled,
		TaxPercent:            st.TaxPercent,
		ServicePercent:        st.ServicePercent,
	}
}

func toNullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}
