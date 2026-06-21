package domain

import "errors"

// OrderLine adalah satu baris pesanan (harga satuan snapshot × kuantitas).
type OrderLine struct {
	Price    int64
	Quantity int
}

// LineTotal = price × quantity.
func (l OrderLine) LineTotal() int64 { return l.Price * int64(l.Quantity) }

// Subtotal menjumlahkan seluruh baris. Pajak = 0 (kebijakan Elkasir).
func Subtotal(lines []OrderLine) int64 {
	var sum int64
	for _, l := range lines {
		sum += l.LineTotal()
	}
	return sum
}

// Total = subtotal - discount + tax (tax selalu 0). Tidak pernah negatif.
func Total(subtotal, discount, tax int64) int64 {
	t := subtotal - discount + tax
	if t < 0 {
		return 0
	}
	return t
}

var (
	ErrInsufficientPayment = errors.New("uang diterima kurang dari total")
	ErrEmptyOrder          = errors.New("pesanan tidak boleh kosong")
)

// CashChange menghitung kembalian untuk pembayaran tunai. Error bila kurang.
func CashChange(amountReceived, total int64) (int64, error) {
	if amountReceived < total {
		return 0, ErrInsufficientPayment
	}
	return amountReceived - total, nil
}
