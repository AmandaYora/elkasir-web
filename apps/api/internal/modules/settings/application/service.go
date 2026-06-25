// Package application: use case modul settings — baca & perbarui konfigurasi toko
// (kontrol diskon, fitur, pajak & layanan). Hanya menyentuh tabel settings (via repo).
package application

import (
	"context"

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
	MaxDiscountPercent    int32 `json:"maxDiscountPercent"`
	MaxOperationalExpense int64 `json:"maxOperationalExpense"`
	CashVarianceTolerance int64 `json:"cashVarianceTolerance"`
	FeatureSelfOrder      bool  `json:"featureSelfOrder"`
	FeatureQris           bool  `json:"featureQris"`
	FeaturePayAtCashier   bool  `json:"featurePayAtCashier"`
	TaxEnabled            bool  `json:"taxEnabled"`
	TaxPercent            int32 `json:"taxPercent"`
	ServicePercent        int32 `json:"servicePercent"`
}

// Input adalah payload PATCH /settings (semua field wajib — admin mengirim objek penuh).
type Input struct {
	MaxDiscountPercent    int32 `json:"maxDiscountPercent"`
	MaxOperationalExpense int64 `json:"maxOperationalExpense"`
	CashVarianceTolerance int64 `json:"cashVarianceTolerance"`
	FeatureSelfOrder      bool  `json:"featureSelfOrder"`
	FeatureQris           bool  `json:"featureQris"`
	FeaturePayAtCashier   bool  `json:"featurePayAtCashier"`
	TaxEnabled            bool  `json:"taxEnabled"`
	TaxPercent            int32 `json:"taxPercent"`
	ServicePercent        int32 `json:"servicePercent"`
}

func (s *Service) Get(ctx context.Context, storeID string) (DTO, error) {
	st, err := s.repo.Get(ctx, storeID)
	if err != nil {
		return DTO{}, err
	}
	return toDTO(st), nil
}

func (s *Service) Update(ctx context.Context, storeID string, in Input) (DTO, error) {
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
	return s.Get(ctx, storeID)
}

func toDTO(st sqlcgen.Setting) DTO {
	return DTO{
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
