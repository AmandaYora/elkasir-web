package application

import "sync"

// PaymentNotifier adalah pub/sub in-memory (per orderID) untuk event status pembayaran
// self-order. Dipakai endpoint SSE agar layar pelanggan maju OTOMATIS saat callback
// pembayaran masuk — tanpa polling. In-memory sudah cukup karena Elkasir berjalan dalam
// satu proses/satu kontainer; tidak perlu broker eksternal (Redis dsb.).
//
// Catatan modularitas: notifier ini MILIK modul selforder dan hanya dipakai di dalamnya
// (service mem-publish saat lunas, handler men-subscribe untuk streaming). Tidak ada modul
// lain yang menyentuhnya.
type PaymentNotifier struct {
	mu     sync.Mutex
	nextID int
	subs   map[string]map[int]chan StatusDTO
}

func newPaymentNotifier() *PaymentNotifier {
	return &PaymentNotifier{subs: make(map[string]map[int]chan StatusDTO)}
}

// subscribe mendaftarkan listener untuk satu orderID. Mengembalikan channel event dan
// fungsi unsubscribe yang WAJIB dipanggil saat koneksi ditutup (mencegah kebocoran).
func (n *PaymentNotifier) subscribe(orderID string) (<-chan StatusDTO, func()) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.subs[orderID] == nil {
		n.subs[orderID] = make(map[int]chan StatusDTO)
	}
	id := n.nextID
	n.nextID++
	ch := make(chan StatusDTO, 1)
	n.subs[orderID][id] = ch

	return ch, func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		conns := n.subs[orderID]
		if conns == nil {
			return
		}
		if c, ok := conns[id]; ok {
			delete(conns, id)
			close(c)
		}
		if len(conns) == 0 {
			delete(n.subs, orderID)
		}
	}
}

// publish mengirim status terkini ke semua listener orderID. Non-blocking: bila buffer
// penuh / tak ada pembaca, event dilewati (channel ber-buffer 1; "paid" bersifat terminal,
// dan handler selalu mengirim snapshot saat koneksi dibuka sehingga tak ada yang terlewat).
func (n *PaymentNotifier) publish(s StatusDTO) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, ch := range n.subs[s.ID] {
		select {
		case ch <- s:
		default:
		}
	}
}
