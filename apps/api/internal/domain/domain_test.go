package domain

import "testing"

func TestExpectedCash(t *testing.T) {
	// Selaras formula lama (mock.ts): initial + cashSales + additional - expenses - withdrawals + adjustments.
	s := ShiftCash{
		InitialCash:       500_000,
		CashSales:         1_200_000,
		AdditionalCapital: 300_000,
		Expenses:          120_000,
		Withdrawals:       2_000_000,
		Adjustments:       -8_000,
	}
	got := s.ExpectedCash()
	want := int64(500_000 + 1_200_000 + 300_000 - 120_000 - 2_000_000 - 8_000)
	if got != want {
		t.Fatalf("ExpectedCash = %d, want %d", got, want)
	}
}

func TestVarianceAndApproval(t *testing.T) {
	p := ControlPolicy{CashVarianceTolerance: 5_000}
	cases := []struct {
		actual, expected int64
		wantVar          int64
		wantApproval     bool
	}{
		{100_000, 100_000, 0, false},
		{100_000, 104_000, -4_000, false}, // dalam toleransi
		{100_000, 106_000, -6_000, true},  // di luar toleransi
		{112_000, 100_000, 12_000, true},
	}
	for _, c := range cases {
		v := Variance(c.actual, c.expected)
		if v != c.wantVar {
			t.Errorf("Variance(%d,%d)=%d want %d", c.actual, c.expected, v, c.wantVar)
		}
		if got := p.VarianceNeedsApproval(v); got != c.wantApproval {
			t.Errorf("VarianceNeedsApproval(%d)=%v want %v", v, got, c.wantApproval)
		}
	}
}

func TestDiscountNeedsApproval(t *testing.T) {
	p := ControlPolicy{MaxDiscountPercent: 10}
	cases := []struct {
		subtotal, discount int64
		want               bool
	}{
		{100_000, 0, false},
		{100_000, 10_000, false}, // tepat 10% → tidak butuh approval
		{100_000, 10_001, true},  // > 10%
		{0, 1, true},             // subtotal 0 tapi ada diskon → butuh approval
		{200_000, 15_000, false}, // 7.5%
		{200_000, 25_000, true},  // 12.5%
	}
	for _, c := range cases {
		if got := p.DiscountNeedsApproval(c.subtotal, c.discount); got != c.want {
			t.Errorf("DiscountNeedsApproval(%d,%d)=%v want %v", c.subtotal, c.discount, got, c.want)
		}
	}
}

func TestExpenseNeedsApproval(t *testing.T) {
	p := ControlPolicy{MaxOperationalExpense: 200_000}
	if p.ExpenseNeedsApproval(200_000) {
		t.Error("200000 tepat plafon seharusnya tidak butuh approval")
	}
	if !p.ExpenseNeedsApproval(200_001) {
		t.Error("200001 di atas plafon seharusnya butuh approval")
	}
}

func TestSalesMath(t *testing.T) {
	lines := []OrderLine{{Price: 18_000, Quantity: 2}, {Price: 35_000, Quantity: 1}}
	if got := Subtotal(lines); got != 71_000 {
		t.Fatalf("Subtotal=%d want 71000", got)
	}
	if got := Total(71_000, 11_000, 0); got != 60_000 {
		t.Fatalf("Total=%d want 60000", got)
	}
	if got := Total(10_000, 50_000, 0); got != 0 {
		t.Fatalf("Total negatif harus 0, got %d", got)
	}
	change, err := CashChange(100_000, 60_000)
	if err != nil || change != 40_000 {
		t.Fatalf("CashChange=%d err=%v want 40000,nil", change, err)
	}
	if _, err := CashChange(50_000, 60_000); err == nil {
		t.Fatal("CashChange harus error saat uang kurang")
	}
}
