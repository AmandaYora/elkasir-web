// Package application holds the transaction module's use cases — the cross-module cashier
// sale orchestrator.
package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	shareddomain "github.com/elkasir/api/internal/domain"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	productclient "github.com/elkasir/api/internal/modules/product/contracts"
	settingsclient "github.com/elkasir/api/internal/modules/settings/contracts"
	shiftclient "github.com/elkasir/api/internal/modules/shift/contracts"
	staffclient "github.com/elkasir/api/internal/modules/staff/contracts"
	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	"github.com/elkasir/api/internal/modules/transaction/domain"
	"github.com/elkasir/api/internal/modules/transaction/infrastructure"
	"github.com/elkasir/api/internal/platform/db"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/uow"
)

// Service merangkai master-data lintas-modul HANYA lewat contract (productclient,
// shiftclient, salesclient) + uow untuk atomik. Tidak menyentuh tabel modul lain langsung.
type Service struct {
	repo     *infrastructure.Repo
	products productclient.Client
	shifts   shiftclient.Client
	sales    salesclient.Client
	settings settingsclient.Client
	staff    staffclient.Client
	uow      *uow.Manager
}

func NewService(repo *infrastructure.Repo, productClient productclient.Client, shiftClient shiftclient.Client, salesClient salesclient.Client, settingsClient settingsclient.Client, staffClient staffclient.Client, uowMgr *uow.Manager) *Service {
	return &Service{repo: repo, products: productClient, shifts: shiftClient, sales: salesClient, settings: settingsClient, staff: staffClient, uow: uowMgr}
}

type ItemDTO struct {
	ProductID   string `json:"productId,omitempty"`
	ProductName string `json:"productName"`
	Category    string `json:"category"`
	Price       int64  `json:"price"`
	Quantity    int32  `json:"quantity"`
	LineTotal   int64  `json:"lineTotal"`
	Note        string `json:"note,omitempty"`
}

type DTO struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	ShiftID        string    `json:"shiftId,omitempty"`
	TableID        string    `json:"tableId,omitempty"`
	SelfOrderID    string    `json:"selfOrderId,omitempty"`
	CashierID      string    `json:"cashierId,omitempty"`
	OrderType      string    `json:"orderType"`
	Source         string    `json:"source"`
	PaymentMethod  string    `json:"paymentMethod"`
	Status         string    `json:"status"`
	Subtotal       int64     `json:"subtotal"`
	Discount       int64     `json:"discount"`
	Tax            int64     `json:"tax"`
	ServiceCharge  int64     `json:"serviceCharge"`
	GatewayFee     int64     `json:"gatewayFee"`
	ServiceLine    int64     `json:"serviceLine"`
	Total          int64     `json:"total"`
	AmountReceived int64      `json:"amountReceived"`
	ChangeAmount   int64      `json:"changeAmount"`
	CustomerNote   string     `json:"customerNote,omitempty"`
	VoidedAt       *time.Time `json:"voidedAt,omitempty"`
	VoidReason     string     `json:"voidReason,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	Items          []ItemDTO  `json:"items"`
}

type CreateItemInput struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
	Note      string `json:"note"`
}

type CreateInput struct {
	Items              []CreateItemInput `json:"items"`
	Discount           int64             `json:"discount"`
	PaymentMethod      string            `json:"paymentMethod"`
	AmountReceived     int64             `json:"amountReceived"`
	TableID            string            `json:"tableId"`
	OrderType          string            `json:"orderType"`
	DiscountApprovedBy string            `json:"discountApprovedBy"`
	SupervisorPin      string            `json:"supervisorPin"`
	CustomerNote       string            `json:"customerNote"`
}

// VoidInput membatalkan transaksi tunai pada shift berjalan (restock + reversal status).
type VoidInput struct {
	Reason        string `json:"reason"`
	SupervisorPin string `json:"supervisorPin"` // wajib untuk kasir; supervisor/admin override
}

// errVoidConflict: balapan — transaksi sudah dibatalkan di antara baca & tulis.
var errVoidConflict = errors.New("transaksi sudah dibatalkan")

// Create membuat transaksi kasir (Kondisi 1) atomik & idempoten.
// Mengembalikan (dto, created). created=false berarti hasil replay idempotency.
func (s *Service) Create(ctx context.Context, p authcontract.Principal, idemKey, reqHash string, in CreateInput) (DTO, bool, error) {
	storeID := p.StoreID

	// Idempotency: kunci sama + body sama → kembalikan hasil lama (tanpa duplikasi).
	if idemKey != "" {
		existing, err := s.repo.Idempotency(ctx, storeID, idemKey)
		switch {
		case err == nil:
			if existing.RequestHash != reqHash {
				return DTO{}, false, httpx.Conflict("Idempotency-Key sudah dipakai untuk permintaan berbeda.")
			}
			dto, err := s.getDTO(ctx, storeID, existing.ResponseBody.String)
			return dto, false, err
		case errors.Is(err, sql.ErrNoRows):
			// lanjut buat baru
		default:
			return DTO{}, false, err
		}
	}

	if len(in.Items) == 0 {
		return DTO{}, false, httpx.Validation("Pesanan tidak boleh kosong.")
	}
	if in.PaymentMethod != "cash" && in.PaymentMethod != "qris" {
		return DTO{}, false, httpx.Validation("Metode pembayaran harus 'cash' atau 'qris'.")
	}
	if in.Discount < 0 {
		return DTO{}, false, httpx.Validation("Diskon tidak boleh negatif.")
	}

	// Snapshot harga & kategori dari produk (via productclient) saat penjualan.
	var items []salesclient.SaleItem
	var subtotal int64
	for _, ci := range in.Items {
		if ci.Quantity <= 0 {
			return DTO{}, false, httpx.Validation("Kuantitas item harus lebih dari 0.")
		}
		pr, err := s.products.GetForSale(ctx, storeID, ci.ProductID)
		if errors.Is(err, productclient.ErrNotFound) {
			return DTO{}, false, httpx.Validation("Produk tidak ditemukan: " + ci.ProductID)
		}
		if err != nil {
			return DTO{}, false, err
		}
		if !pr.Active {
			return DTO{}, false, httpx.Validation("Produk nonaktif: " + pr.Name)
		}
		lt := pr.Price * int64(ci.Quantity)
		subtotal += lt
		items = append(items, salesclient.SaleItem{
			ProductID: pr.ID, ProductName: pr.Name, Category: pr.Category,
			Price: pr.Price, Quantity: ci.Quantity, LineTotal: lt, Note: strings.TrimSpace(ci.Note),
		})
	}

	// Settings toko (sekali ambil) → kebijakan kontrol diskon + biaya layanan/PPN.
	cfg, err := s.loadSettings(ctx, storeID)
	if err != nil {
		return DTO{}, false, err
	}

	// Backstop QRIS: bila admin menonaktifkan QRIS, tolak meski klien menembus UI.
	if in.PaymentMethod == "qris" && !cfg.FeatureQris {
		return DTO{}, false, httpx.Unprocessable("Pembayaran QRIS sedang tidak tersedia.")
	}

	policy := controlPolicyFrom(cfg)
	// Diskon di atas batas butuh persetujuan supervisor — kecuali yang menjalankan SUDAH
	// supervisor/admin (override otomatis terpenuhi). Untuk kasir, PIN supervisor diverifikasi
	// DI SERVER (anti-spoof): nama supervisor yang tercatat di approvedBy berasal dari hasil
	// resolusi PIN, bukan dari string yang dikirim klien.
	approvedBy := strings.TrimSpace(in.DiscountApprovedBy)
	if policy.DiscountNeedsApproval(subtotal, in.Discount) && !p.IsSupervisorOrAdmin() {
		sup, ok, verr := s.staff.ResolveSupervisorByPIN(ctx, storeID, in.SupervisorPin)
		if verr != nil {
			return DTO{}, false, verr
		}
		if !ok {
			return DTO{}, false, httpx.Forbidden("Diskon melebihi batas; butuh PIN supervisor yang valid.")
		}
		approvedBy = sup.Name
	}

	// Kasir tidak memakai payment gateway → gateway fee 0; service 2% + PPN tetap berlaku.
	bd := shareddomain.ComputeBreakdown(subtotal, in.Discount, 0, cfg.ServicePercent, cfg.TaxPercent, cfg.TaxEnabled)
	total := bd.Total

	var amountReceived, change int64
	if in.PaymentMethod == "cash" {
		amountReceived = in.AmountReceived
		ch, err := shareddomain.CashChange(amountReceived, total)
		if err != nil {
			return DTO{}, false, httpx.Validation("Uang diterima kurang dari total.")
		}
		change = ch
	} else {
		amountReceived = total
	}

	shiftID, err := s.shifts.CurrentOpenID(ctx, storeID)
	if err != nil {
		return DTO{}, false, err
	}

	cashierID := ""
	if p.Actor == authcontract.ActorStaff {
		cashierID = p.SubjectID
	}

	// Atomik lintas-modul: kurangi stok (productclient) + catat penjualan (salesclient)
	// dalam SATU transaksi DB via uow. Gagal di tengah → seluruhnya rollback.
	var txID string
	err = s.uow.Run(ctx, func(ctx context.Context) error {
		for _, it := range items {
			if derr := s.products.Decrease(ctx, storeID, it.ProductID, it.Quantity); derr != nil {
				if errors.Is(derr, productclient.ErrInsufficientStock) {
					return fmt.Errorf("%w: %s", productclient.ErrInsufficientStock, it.ProductName)
				}
				return derr
			}
		}
		newID, rerr := s.sales.RecordSale(ctx, salesclient.RecordSaleInput{
			StoreID: storeID, Source: "cashier", PaymentMethod: in.PaymentMethod, OrderType: in.OrderType,
			TableID: strings.TrimSpace(in.TableID), CashierID: cashierID, ShiftID: shiftID,
			DiscountApprovedBy: approvedBy,
			CustomerNote:       strings.TrimSpace(in.CustomerNote),
			Items:              items,
			Subtotal:           subtotal,
			Discount:           in.Discount,
			Tax:                bd.Tax,
			ServiceCharge:      bd.Service,
			GatewayFee:         0,
			Total:              total,
			AmountReceived:     amountReceived,
			Change:             change,
			IdempotencyKey:     idemKey,
			RequestHash:        reqHash,
		})
		if rerr != nil {
			return rerr
		}
		txID = newID
		return nil
	})
	if errors.Is(err, productclient.ErrInsufficientStock) {
		return DTO{}, false, httpx.Unprocessable("Stok tidak cukup (" + strings.TrimPrefix(err.Error(), "stok tidak cukup: ") + ").")
	}
	if db.IsDuplicate(err) {
		return DTO{}, false, httpx.Conflict("Transaksi duplikat (retry bersamaan).")
	}
	if err != nil {
		return DTO{}, false, err
	}

	dto, err := s.getDTO(ctx, storeID, txID)
	return dto, true, err
}

// Void membatalkan transaksi: restock item + tandai 'voided' secara atomik. Hanya transaksi
// TUNAI pada SHIFT BERJALAN yang bisa dibatalkan (agar shift tertutup tidak berubah). Kasir
// butuh PIN supervisor (diverifikasi server); supervisor/admin override otomatis. Begitu status
// jadi 'voided', transaksi otomatis keluar dari rekonsiliasi shift & laporan (query memfilter
// status='completed').
func (s *Service) Void(ctx context.Context, p authcontract.Principal, txID string, in VoidInput) (DTO, error) {
	storeID := p.StoreID

	t, err := s.repo.Get(ctx, storeID, txID)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Transaksi tidak ditemukan.")
	}
	if err != nil {
		return DTO{}, err
	}
	if t.Status != sqlcgen.TransactionsStatusCompleted {
		return DTO{}, httpx.Conflict("Transaksi ini sudah dibatalkan atau tidak dapat dibatalkan.")
	}
	if t.PaymentMethod != sqlcgen.TransactionsPaymentMethodCash {
		return DTO{}, httpx.Unprocessable("Hanya transaksi tunai yang bisa dibatalkan di kasir. Untuk QRIS ajukan refund lewat admin.")
	}

	// Hanya transaksi pada shift yang masih BERJALAN yang dapat dibatalkan.
	openShiftID, err := s.shifts.CurrentOpenID(ctx, storeID)
	if err != nil {
		return DTO{}, err
	}
	if !t.ShiftID.Valid || t.ShiftID.String == "" || t.ShiftID.String != openShiftID {
		return DTO{}, httpx.Conflict("Hanya transaksi pada shift berjalan yang dapat dibatalkan.")
	}

	// Otorisasi: supervisor/admin langsung (tercatat sebagai pelaku); kasir butuh PIN supervisor
	// (diverifikasi server, dan supervisor itulah yang tercatat sebagai pemberi otorisasi).
	voidedBy := p.SubjectID
	if !p.IsSupervisorOrAdmin() {
		sup, ok, verr := s.staff.ResolveSupervisorByPIN(ctx, storeID, in.SupervisorPin)
		if verr != nil {
			return DTO{}, verr
		}
		if !ok {
			return DTO{}, httpx.Forbidden("Pembatalan butuh PIN supervisor yang valid.")
		}
		voidedBy = sup.ID
	}

	items, err := s.repo.Items(ctx, txID)
	if err != nil {
		return DTO{}, err
	}

	// Atomik: kembalikan stok tiap item (yang masih punya produk) + tandai voided.
	err = s.uow.Run(ctx, func(ctx context.Context) error {
		for _, it := range items {
			if !it.ProductID.Valid || it.ProductID.String == "" {
				continue
			}
			if rerr := s.products.Increase(ctx, storeID, it.ProductID.String, it.Quantity); rerr != nil {
				return rerr
			}
		}
		ok, verr := s.sales.VoidSale(ctx, salesclient.VoidSaleInput{
			StoreID: storeID, TxID: txID, VoidedBy: voidedBy, Reason: strings.TrimSpace(in.Reason),
		})
		if verr != nil {
			return verr
		}
		if !ok {
			return errVoidConflict
		}
		return nil
	})
	if errors.Is(err, errVoidConflict) {
		return DTO{}, httpx.Conflict("Transaksi sudah dibatalkan.")
	}
	if err != nil {
		return DTO{}, err
	}

	return s.getDTO(ctx, storeID, txID)
}

func (s *Service) Get(ctx context.Context, storeID, txID string) (DTO, error) {
	return s.getDTO(ctx, storeID, txID)
}

func (s *Service) List(ctx context.Context, f domain.ListFilter) ([]DTO, int64, error) {
	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DTO, 0, len(rows))
	for _, t := range rows {
		out = append(out, toDTO(t, nil))
	}
	return out, total, nil
}

func (s *Service) getDTO(ctx context.Context, storeID, txID string) (DTO, error) {
	t, err := s.repo.Get(ctx, storeID, txID)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Transaksi tidak ditemukan.")
	}
	if err != nil {
		return DTO{}, err
	}
	items, err := s.repo.Items(ctx, txID)
	if err != nil {
		return DTO{}, err
	}
	return toDTO(t, items), nil
}

// loadSettings membaca settings via kontrak settingsclient. settingsclient.Get SUDAH
// mengembalikan default aman saat baris settings belum ada (toko baru) TANPA error, jadi error di
// sini berarti kegagalan DB sungguhan. Dalam kasus itu kita TIDAK fail-open: melanjutkan dengan
// default bisa diam-diam menjatuhkan PPN ke 0 atau melewati backstop QRIS (salah tagih). Lebih baik
// gagalkan penjualan — toh tulisan UoW berikutnya ke DB yang sama akan gagal juga.
func (s *Service) loadSettings(ctx context.Context, storeID string) (settingsclient.Settings, error) {
	return s.settings.Get(ctx, storeID)
}

func controlPolicyFrom(cfg settingsclient.Settings) shareddomain.ControlPolicy {
	return shareddomain.ControlPolicy{
		MaxDiscountPercent:    int64(cfg.MaxDiscountPercent),
		MaxOperationalExpense: cfg.MaxOperationalExpense,
		CashVarianceTolerance: cfg.CashVarianceTolerance,
	}
}

func toDTO(t sqlcgen.Transaction, items []sqlcgen.TransactionItem) DTO {
	d := DTO{
		ID: t.ID, Code: t.Code, ShiftID: t.ShiftID.String, TableID: t.TableID.String,
		SelfOrderID: t.SelfOrderID.String, CashierID: t.CashierID.String,
		OrderType: string(t.OrderType), Source: string(t.Source), PaymentMethod: string(t.PaymentMethod),
		Status: string(t.Status), Subtotal: t.Subtotal, Discount: t.Discount, Tax: t.Tax,
		ServiceCharge: t.ServiceCharge, GatewayFee: t.GatewayFee, ServiceLine: t.ServiceCharge + t.GatewayFee,
		Total:          t.Total,
		AmountReceived: t.AmountReceived, ChangeAmount: t.ChangeAmount, CustomerNote: t.CustomerNote.String,
		VoidReason: t.VoidReason.String,
		CreatedAt:  t.CreatedAt, Items: []ItemDTO{},
	}
	if t.VoidedAt.Valid {
		d.VoidedAt = &t.VoidedAt.Time
	}
	for _, it := range items {
		d.Items = append(d.Items, ItemDTO{
			ProductID: it.ProductID.String, ProductName: it.ProductName, Category: it.Category,
			Price: it.Price, Quantity: it.Quantity, LineTotal: it.LineTotal, Note: it.Note.String,
		})
	}
	return d
}
