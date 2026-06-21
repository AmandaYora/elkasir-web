// Package domain holds the selforder module's pure value objects, line-item snapshot
// value type, and the place-order input + validation. It is free of transport (JSON)
// and persistence (sqlc) concerns.
//
// The self-order read-model itself is the sqlc-generated row (sqlcgen.SelfOrder); the
// JSON DTOs (Menu/Order/Place result/Status/Checkout) live in the application layer.
package domain

import "errors"

// OrderItem adalah snapshot baris pesanan (harga & kategori dibekukan saat pemesanan).
// Dipakai repo saat menyimpan self_order_items.
type OrderItem struct {
	ProductID   string
	ProductName string
	Category    string
	Price       int64
	Quantity    int32
	LineTotal   int64
	Note        string
}

// PlaceItem adalah satu baris permintaan pada place-order (kuantitas dari pelanggan).
type PlaceItem struct {
	ProductID string
	Quantity  int32
	Note      string
}

// PlaceInput adalah masukan pembuatan self-order (sudah bebas transport).
type PlaceInput struct {
	Items         []PlaceItem
	PaymentMethod string
	CustomerNote  string
}

// Validate memeriksa aturan dasar place-order yang tak butuh data lintas-modul.
// Validasi yang butuh produk/meja (mis. harga, status meja) dilakukan di service.
func (in PlaceInput) Validate() error {
	if in.PaymentMethod != "qris" && in.PaymentMethod != "cash" {
		return ErrInvalidPaymentMethod
	}
	if len(in.Items) == 0 {
		return ErrEmptyOrder
	}
	for _, it := range in.Items {
		if it.Quantity <= 0 {
			return ErrInvalidQuantity
		}
	}
	return nil
}

// ListFilter untuk daftar self-order masuk (filter status opsional).
type ListFilter struct {
	StoreID string
	Status  string
	Limit   int
	Offset  int
}

// Sentinel validation errors (business rules). Service memetakan ke httpx.* sesuai konteks.
var (
	// ErrInvalidPaymentMethod: metode pembayaran bukan 'qris' atau 'cash'.
	ErrInvalidPaymentMethod = errors.New("metode pembayaran harus 'qris' atau 'cash'")
	// ErrEmptyOrder: pesanan tanpa item.
	ErrEmptyOrder = errors.New("pesanan tidak boleh kosong")
	// ErrInvalidQuantity: kuantitas item <= 0.
	ErrInvalidQuantity = errors.New("kuantitas item harus lebih dari 0")
)
