package domain

// ControlPolicy — ambang persetujuan supervisor (ditetapkan owner di settings).
type ControlPolicy struct {
	MaxDiscountPercent    int64 // diskon di atas % subtotal ini butuh approval
	MaxOperationalExpense int64 // biaya operasional di atas ini butuh approval
	CashVarianceTolerance int64 // selisih kas di atas ini butuh approval saat tutup shift
}

// DiscountNeedsApproval true bila diskon melebihi MaxDiscountPercent dari subtotal.
// Memakai aritmetika integer (diskon*100 > subtotal*persen) agar tanpa float.
func (p ControlPolicy) DiscountNeedsApproval(subtotal, discount int64) bool {
	if discount <= 0 {
		return false
	}
	if subtotal <= 0 {
		return true
	}
	return discount*100 > subtotal*p.MaxDiscountPercent
}

// ExpenseNeedsApproval true bila biaya operasional melebihi plafon.
func (p ControlPolicy) ExpenseNeedsApproval(amount int64) bool {
	return amount > p.MaxOperationalExpense
}

// VarianceNeedsApproval true bila |selisih kas| melebihi toleransi.
func (p ControlPolicy) VarianceNeedsApproval(variance int64) bool {
	return abs(variance) > p.CashVarianceTolerance
}
