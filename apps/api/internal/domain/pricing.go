package domain

// Pricing — perhitungan biaya layanan, PPN, dan total. Semua dalam rupiah penuh (integer).
//
// Kebijakan (dikonfirmasi pemilik):
//   - Service  = 2% × Subtotal, DIBULATKAN KE ATAS (lihat RoundUpService). Dikenakan ke semua
//     transaksi (cash/QRIS, kasir/self-order).
//   - GatewayFee = biaya provider (Tripay/Midtrans) untuk QRIS; 0 untuk kasir/cash. Diquote
//     dari provider (live), bukan dihitung di sini.
//   - Tax (PPN) = persen × Subtotal bila diaktifkan (tidak dibulatkan).
//   - Total    = Subtotal − Discount + Service + GatewayFee + Tax.
//
// "Layanan" yang ditampilkan ke pelanggan = Service + GatewayFee (satu baris).

// RoundUpService membulatkan biaya layanan KE ATAS berdasarkan sisa terhadap ribuan:
// sisa = 0 → tetap; 0 < sisa ≤ 500 → naik ke kelipatan .500; sisa > 500 → naik ke ribuan berikutnya.
// Contoh: 1.350→1.500 · 1.650→2.000 · 480→500 · 540→1.000 · 1.500→1.500.
func RoundUpService(n int64) int64 {
	if n <= 0 {
		return 0
	}
	rem := n % 1000
	if rem == 0 {
		return n
	}
	base := n - rem
	if rem <= 500 {
		return base + 500
	}
	return base + 1000
}

// ServiceCharge = RoundUpService(percent% × subtotal). percent integer (mis. 2 = 2%).
func ServiceCharge(subtotal int64, percent int32) int64 {
	if percent <= 0 || subtotal <= 0 {
		return 0
	}
	return RoundUpService(subtotal * int64(percent) / 100)
}

// Tax = percent% × subtotal bila enabled (tidak dibulatkan).
func Tax(subtotal int64, percent int32, enabled bool) int64 {
	if !enabled || percent <= 0 || subtotal <= 0 {
		return 0
	}
	return subtotal * int64(percent) / 100
}

// Breakdown adalah rincian biaya satu transaksi.
type Breakdown struct {
	Subtotal   int64
	Discount   int64
	Service    int64 // 2% (rounded) — margin merchant
	GatewayFee int64 // biaya provider (QRIS) — pass-through ke gateway
	Tax        int64 // PPN
	Total      int64 // yang dibayar pelanggan
}

// ServiceLine = baris "Layanan" yang ditampilkan ke pelanggan (service + gateway fee).
func (b Breakdown) ServiceLine() int64 { return b.Service + b.GatewayFee }

// PreGatewayBase = jumlah yang menjadi dasar quote fee gateway (sebelum fee ditambahkan):
// Subtotal − Discount + Service + Tax. Provider menghitung fee atas nilai ini.
func PreGatewayBase(subtotal, discount int64, servicePercent, taxPercent int32, taxEnabled bool) int64 {
	base := subtotal - discount + ServiceCharge(subtotal, servicePercent) + Tax(subtotal, taxPercent, taxEnabled)
	if base < 0 {
		return 0
	}
	return base
}

// ComputeBreakdown menyusun rincian penuh. gatewayFee diisi caller (0 bila non-QRIS / simulasi).
func ComputeBreakdown(subtotal, discount, gatewayFee int64, servicePercent, taxPercent int32, taxEnabled bool) Breakdown {
	service := ServiceCharge(subtotal, servicePercent)
	tax := Tax(subtotal, taxPercent, taxEnabled)
	if gatewayFee < 0 {
		gatewayFee = 0
	}
	total := subtotal - discount + service + gatewayFee + tax
	if total < 0 {
		total = 0
	}
	return Breakdown{
		Subtotal: subtotal, Discount: discount, Service: service,
		GatewayFee: gatewayFee, Tax: tax, Total: total,
	}
}
