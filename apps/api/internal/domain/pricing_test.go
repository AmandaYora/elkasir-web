package domain

import "testing"

func TestRoundUpService(t *testing.T) {
	cases := []struct{ in, want int64 }{
		{0, 0},
		{1, 500},
		{200, 500},
		{480, 500},
		{500, 500},
		{540, 1000},
		{900, 1000},
		{1000, 1000},
		{1350, 1500}, // contoh pemilik
		{1500, 1500},
		{1650, 2000}, // contoh pemilik
		{2000, 2000},
	}
	for _, c := range cases {
		if got := RoundUpService(c.in); got != c.want {
			t.Errorf("RoundUpService(%d)=%d want %d", c.in, got, c.want)
		}
	}
}

func TestServiceCharge(t *testing.T) {
	// 2% dari 67.500 = 1.350 → 1.500
	if got := ServiceCharge(67_500, 2); got != 1_500 {
		t.Errorf("ServiceCharge(67500,2)=%d want 1500", got)
	}
	// 2% dari 82.500 = 1.650 → 2.000
	if got := ServiceCharge(82_500, 2); got != 2_000 {
		t.Errorf("ServiceCharge(82500,2)=%d want 2000", got)
	}
	if got := ServiceCharge(100_000, 0); got != 0 {
		t.Errorf("ServiceCharge percent 0 must be 0, got %d", got)
	}
}

func TestTax(t *testing.T) {
	if got := Tax(100_000, 11, true); got != 11_000 {
		t.Errorf("Tax(100000,11,true)=%d want 11000", got)
	}
	if got := Tax(100_000, 11, false); got != 0 {
		t.Errorf("Tax disabled must be 0, got %d", got)
	}
}

func TestComputeBreakdown(t *testing.T) {
	// Subtotal 100.000, no discount, gatewayFee 1.450, service 2% → 2.000, PPN 11% → 11.000.
	b := ComputeBreakdown(100_000, 0, 1_450, 2, 11, true)
	if b.Service != 2_000 || b.Tax != 11_000 || b.GatewayFee != 1_450 {
		t.Fatalf("unexpected components: %+v", b)
	}
	wantTotal := int64(100_000 - 0 + 2_000 + 1_450 + 11_000)
	if b.Total != wantTotal {
		t.Errorf("Total=%d want %d", b.Total, wantTotal)
	}
	if b.ServiceLine() != 2_000+1_450 {
		t.Errorf("ServiceLine=%d want %d", b.ServiceLine(), 2_000+1_450)
	}
	// PreGatewayBase = subtotal - disc + service + tax = 100000 + 2000 + 11000
	if got := PreGatewayBase(100_000, 0, 2, 11, true); got != 113_000 {
		t.Errorf("PreGatewayBase=%d want 113000", got)
	}
}
