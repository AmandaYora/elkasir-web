// Package salesclient adalah KONTRAK modul transaction (ledger penjualan) untuk
// dikonsumsi modul lain — terutama selforder saat menebus/checkout menjadi transaksi.
// (≈ "create()" pada diagram). RecordSale TIDAK mengubah stok; pengurangan stok adalah
// tanggung jawab productclient.Decrease yang dipanggil orchestrator dalam transaksi sama.
package salesclient

import "context"

// SaleItem adalah snapshot baris penjualan.
type SaleItem struct {
	ProductID   string
	ProductName string
	Category    string
	Price       int64
	Quantity    int32
	LineTotal   int64
	Note        string
}

// RecordSaleInput memuat seluruh data pembuatan satu transaksi penjualan.
// Field id (TableID/SelfOrderID/CashierID/ShiftID/...) adalah ID primitif lintas-modul
// ("" = NULL) — tanpa FK, sesuai prinsip "Bebas dari Penjara FK".
type RecordSaleInput struct {
	StoreID            string
	Source             string // "cashier" | "self_order"
	PaymentMethod      string // "cash" | "qris"
	OrderType          string // "dineIn" | "takeaway"
	TableID            string
	SelfOrderID        string
	CashierID          string
	ShiftID            string
	DiscountApprovedBy string
	CustomerNote       string
	Items              []SaleItem
	Subtotal           int64
	Discount           int64
	Tax                int64 // PPN
	ServiceCharge      int64 // biaya layanan 2% (rounded)
	GatewayFee         int64 // biaya gateway QRIS (0 utk cash/kasir)
	Total              int64
	AmountReceived     int64
	Change             int64
	IdempotencyKey     string
	RequestHash        string
}

// VoidSaleInput memuat data pembatalan satu transaksi (void).
type VoidSaleInput struct {
	StoreID  string
	TxID     string
	VoidedBy string // staf yang mengotorisasi (supervisor/admin) — "" = NULL
	Reason   string
}

// Client adalah kontrak yang dipublikasikan modul transaction.
type Client interface {
	// RecordSale menyisipkan transaksi + item (+ idempotency bila ada) dan mengembalikan
	// id transaksi. Harus dipanggil di dalam uow.Run agar atomik dengan langkah lain.
	RecordSale(ctx context.Context, in RecordSaleInput) (txID string, err error)
	// VoidSale menandai transaksi 'completed' menjadi 'voided' (status reversal). Harus
	// dipanggil di dalam uow.Run agar atomik dengan restock. ok=false bila tak ada baris
	// 'completed' yang cocok (sudah dibatalkan / tak ditemukan).
	VoidSale(ctx context.Context, in VoidSaleInput) (ok bool, err error)
}
