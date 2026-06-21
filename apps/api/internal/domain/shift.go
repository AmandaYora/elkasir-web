// Package domain memuat aturan bisnis murni (tanpa DB/HTTP) agar mudah diuji &
// menjadi satu-satunya sumber kebenaran logika — bukan terduplikasi di klien.
package domain

// ShiftCash adalah komponen kas sebuah shift untuk rekonsiliasi penutupan.
type ShiftCash struct {
	InitialCash       int64
	CashSales         int64
	AdditionalCapital int64
	Expenses          int64
	Withdrawals       int64
	Adjustments       int64
}

// ExpectedCash = initial + cashSales + additionalCapital - expenses - withdrawals + adjustments.
// Formula ini WAJIB identik dengan perhitungan lama (POS/web) — lihat mock.ts.
func (s ShiftCash) ExpectedCash() int64 {
	return s.InitialCash + s.CashSales + s.AdditionalCapital - s.Expenses - s.Withdrawals + s.Adjustments
}

// Variance = actualCash - expectedCash (positif = lebih, negatif = kurang).
func Variance(actualCash, expectedCash int64) int64 {
	return actualCash - expectedCash
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
