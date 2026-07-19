package application

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	"github.com/elkasir/api/internal/modules/subscription/domain"
	"github.com/elkasir/api/internal/platform/httpx"
)

// ── fakeRepo: in-memory stand-in for infrastructure.Repo ─────────────────────────────────────
//
// Mirrors the real repository's SQL-level guards (status='pending' before any mark-paid/
// mark-terminal write) so tests exercise the same invariants the production code relies on,
// without a real MySQL instance. Methods this package's tests never exercise panic loudly
// instead of silently returning zero values, so an accidental call is caught immediately.

type fakeRepo struct {
	mu    sync.Mutex
	plans map[string]domain.Plan
	subs  map[string]domain.Subscription
	invs  map[string]domain.Invoice

	// forceCreateInvoiceErr, if set, is returned exactly once by the next CreateInvoice call —
	// used to simulate losing a race against a concurrent checkout (the DB unique-index path).
	forceCreateInvoiceErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{plans: map[string]domain.Plan{}, subs: map[string]domain.Subscription{}, invs: map[string]domain.Invoice{}}
}

func (r *fakeRepo) ListActivePlans(ctx context.Context) ([]domain.Plan, error) {
	panic("not exercised by these tests")
}
func (r *fakeRepo) ListAllPlans(ctx context.Context) ([]domain.Plan, error) {
	panic("not exercised by these tests")
}
func (r *fakeRepo) CreatePlan(ctx context.Context, planID, code, name string, price int64, periodDays int32, isActive bool) error {
	panic("not exercised by these tests")
}
func (r *fakeRepo) UpdatePlan(ctx context.Context, planID, name string, price int64, periodDays int32, isActive bool) (int64, error) {
	panic("not exercised by these tests")
}
func (r *fakeRepo) PlatformRevenue(ctx context.Context) (int64, error) {
	panic("not exercised by these tests")
}
func (r *fakeRepo) ListInvoices(ctx context.Context, storeID string, limit, offset int32) ([]domain.Invoice, int64, error) {
	panic("not exercised by these tests")
}

func (r *fakeRepo) GetPlan(ctx context.Context, planID string) (domain.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.plans[planID]
	if !ok {
		return domain.Plan{}, sql.ErrNoRows
	}
	return p, nil
}

func (r *fakeRepo) GetByStore(ctx context.Context, storeID string) (domain.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.subs[storeID]
	if !ok {
		return domain.Subscription{}, sql.ErrNoRows
	}
	return s, nil
}

func (r *fakeRepo) GetPendingInvoice(ctx context.Context, storeID string) (domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var latest domain.Invoice
	found := false
	for _, inv := range r.invs {
		if inv.StoreID == storeID && inv.Status == "pending" && (!found || inv.CreatedAt.After(latest.CreatedAt)) {
			latest, found = inv, true
		}
	}
	if !found {
		return domain.Invoice{}, sql.ErrNoRows
	}
	return latest, nil
}

func (r *fakeRepo) CreateInvoice(ctx context.Context, invID, storeID, planID, provider, providerRef string, amount int64) (domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceCreateInvoiceErr != nil {
		err := r.forceCreateInvoiceErr
		r.forceCreateInvoiceErr = nil
		return domain.Invoice{}, err
	}
	inv := domain.Invoice{
		ID: invID, StoreID: storeID, PlanID: planID, Amount: amount,
		Status: "pending", Provider: provider, ProviderRef: providerRef, CreatedAt: time.Now(),
	}
	r.invs[invID] = inv
	return inv, nil
}

func (r *fakeRepo) MarkInvoiceTerminal(ctx context.Context, invoiceID, status string) error {
	if err := ctx.Err(); err != nil {
		// A real *sql.DB call would fail immediately against an already-cancelled/expired
		// context — simulate that here so tests can verify Checkout's cleanup write uses a
		// context detached from the inbound request (see
		// TestCheckoutCleanupSurvivesRequestContextCancellation).
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invs[invoiceID]
	if !ok || inv.Status != "pending" {
		return nil // matches real repo: guarded by status='pending', a no-op otherwise
	}
	inv.Status = status
	r.invs[invoiceID] = inv
	return nil
}

func (r *fakeRepo) SetInvoiceProviderRef(ctx context.Context, invoiceID, providerRef string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invs[invoiceID]
	if !ok {
		return nil
	}
	inv.ProviderRef = providerRef
	r.invs[invoiceID] = inv
	return nil
}

func (r *fakeRepo) MarkInvoicePaidAndExtend(ctx context.Context, invoiceID string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invs[invoiceID]
	if !ok || inv.Status != "pending" {
		return nil // matches real repo: guarded by status='pending'
	}
	end := now.Add(24 * time.Hour)
	inv.Status = "paid"
	inv.PeriodStart, inv.PeriodEnd = &now, &end
	r.invs[invoiceID] = inv

	sub := r.subs[inv.StoreID]
	sub.StoreID, sub.PlanID = inv.StoreID, inv.PlanID
	sub.CurrentPeriodStart, sub.CurrentPeriodEnd = &now, &end
	r.subs[inv.StoreID] = sub
	return nil
}

func (r *fakeRepo) ListPendingElProof(ctx context.Context, limit int32) ([]domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0, len(r.invs))
	for id := range r.invs {
		ids = append(ids, id)
	}
	sort.Strings(ids) // deterministic order for test assertions
	out := make([]domain.Invoice, 0, len(ids))
	for _, id := range ids {
		inv := r.invs[id]
		if inv.Status == "pending" && inv.Provider == "elproof" {
			out = append(out, inv)
			if int32(len(out)) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *fakeRepo) invoice(t *testing.T, id string) domain.Invoice {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invs[id]
	if !ok {
		t.Fatalf("invoice %q tidak ditemukan di fakeRepo", id)
	}
	return inv
}

// ── fakePaymentClient: stands in for paymentclient.Client ────────────────────────────────────
//
// Embeds a nil paymentclient.Client so any method these tests don't stub panics loudly on call
// (nil interface dereference) rather than silently returning a zero value.

type fakePaymentClient struct {
	paymentclient.Client
	mu                sync.Mutex
	createChargeCalls int
	createChargeFn    func(orderID string) (paymentclient.Charge, error)
	checkStatusFn     func(ref string) (paymentclient.ChargeStatus, error)
}

func (f *fakePaymentClient) CreateCharge(ctx context.Context, appID, storeID, orderID string, amount int64) (paymentclient.Charge, error) {
	f.mu.Lock()
	f.createChargeCalls++
	f.mu.Unlock()
	if f.createChargeFn != nil {
		return f.createChargeFn(orderID)
	}
	return paymentclient.Charge{Provider: "elproof", ProviderRef: "PRV-" + orderID, QRImageURL: "https://qr.example/" + orderID}, nil
}

func (f *fakePaymentClient) CheckStatus(ctx context.Context, appID, ref string) (paymentclient.ChargeStatus, error) {
	if f.checkStatusFn != nil {
		return f.checkStatusFn(ref)
	}
	return paymentclient.ChargeStatus{}, nil
}

func apiErrStatus(t *testing.T, err error) int {
	t.Helper()
	var apiErr *httpx.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %v bukan *httpx.APIError", err)
	}
	return apiErr.Status
}

// ── Checkout ──────────────────────────────────────────────────────────────────────────────────

func TestCheckoutSuccessPersistsPendingInvoiceThenProviderRef(t *testing.T) {
	repo := newFakeRepo()
	repo.plans["plan-1"] = domain.Plan{ID: "plan-1", Code: "BASIC", Name: "Basic", Price: 50_000, PeriodDays: 30, IsActive: true}
	pay := &fakePaymentClient{}
	svc := NewService(repo, pay)

	res, err := svc.Checkout(context.Background(), "store-1", "plan-1")
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if res.QRImageURL == "" || res.Invoice.Status != "pending" {
		t.Fatalf("CheckoutResult = %+v, want QRImageURL set and Status=pending", res)
	}
	if pay.createChargeCalls != 1 {
		t.Fatalf("CreateCharge dipanggil %d kali, want 1", pay.createChargeCalls)
	}
	stored := repo.invoice(t, res.Invoice.ID)
	if stored.Provider != "elproof" || stored.ProviderRef != "PRV-"+res.Invoice.ID || stored.Status != "pending" {
		t.Fatalf("invoice tersimpan = %+v, want Provider=elproof ProviderRef=PRV-<id> Status=pending", stored)
	}
}

func TestCheckoutMarksInvoiceFailedWhenGatewayCallFails(t *testing.T) {
	// This is the direct regression test for the write-ordering fix: the invoice must exist
	// locally BEFORE the gateway call, and a gateway failure must close it out as 'failed'
	// rather than leaving an orphaned invoice for the reconciler to poll forever.
	repo := newFakeRepo()
	repo.plans["plan-1"] = domain.Plan{ID: "plan-1", Code: "BASIC", Name: "Basic", Price: 50_000, PeriodDays: 30, IsActive: true}
	pay := &fakePaymentClient{createChargeFn: func(orderID string) (paymentclient.Charge, error) {
		return paymentclient.Charge{}, errors.New("elproof: charge gagal terkirim")
	}}
	svc := NewService(repo, pay)

	_, err := svc.Checkout(context.Background(), "store-1", "plan-1")
	if err == nil {
		t.Fatal("Checkout seharusnya gagal ketika gateway gagal")
	}
	if apiErrStatus(t, err) != 500 {
		t.Fatalf("status error = %d, want 500", apiErrStatus(t, err))
	}

	// Exactly one invoice should exist, and it must be 'failed' — not left 'pending' forever.
	repo.mu.Lock()
	n := len(repo.invs)
	var only domain.Invoice
	for _, inv := range repo.invs {
		only = inv
	}
	repo.mu.Unlock()
	if n != 1 {
		t.Fatalf("jumlah invoice = %d, want 1", n)
	}
	if only.Status != "failed" {
		t.Fatalf("invoice.Status = %q, want failed", only.Status)
	}
}

func TestCheckoutCleanupSurvivesRequestContextCancellation(t *testing.T) {
	// Regression test: CreateCharge can realistically fail BECAUSE the inbound request's context
	// was cancelled (client disconnected, deadline exceeded) — the cleanup write that marks the
	// invoice 'failed' must NOT reuse that same (now-cancelled) context, or it would fail too,
	// leaving the invoice stuck 'pending' forever with no charge behind it. fakeRepo's
	// MarkInvoiceTerminal returns ctx.Err() immediately when the context it's given is already
	// cancelled/expired, mimicking a real *sql.DB call.
	repo := newFakeRepo()
	repo.plans["plan-1"] = domain.Plan{ID: "plan-1", Code: "BASIC", Name: "Basic", Price: 50_000, PeriodDays: 30, IsActive: true}

	ctx, cancel := context.WithCancel(context.Background())
	pay := &fakePaymentClient{createChargeFn: func(orderID string) (paymentclient.Charge, error) {
		cancel() // simulate: CreateCharge's own HTTP call failed because ctx got cancelled
		return paymentclient.Charge{}, context.Canceled
	}}
	svc := NewService(repo, pay)

	if _, err := svc.Checkout(ctx, "store-1", "plan-1"); err == nil {
		t.Fatal("Checkout seharusnya gagal")
	}

	repo.mu.Lock()
	var only domain.Invoice
	for _, inv := range repo.invs {
		only = inv
	}
	repo.mu.Unlock()
	if only.Status != "failed" {
		t.Fatalf("invoice.Status = %q, want failed — cleanup write harus pakai context terpisah dari ctx yang sudah dibatalkan", only.Status)
	}
}

func TestCheckoutRejectsWhenInvoiceAlreadyPending(t *testing.T) {
	repo := newFakeRepo()
	repo.plans["plan-1"] = domain.Plan{ID: "plan-1", Code: "BASIC", Name: "Basic", Price: 50_000, PeriodDays: 30, IsActive: true}
	repo.invs["inv-existing"] = domain.Invoice{ID: "inv-existing", StoreID: "store-1", PlanID: "plan-1", Status: "pending", Provider: "elproof", CreatedAt: time.Now()}
	pay := &fakePaymentClient{}
	svc := NewService(repo, pay)

	_, err := svc.Checkout(context.Background(), "store-1", "plan-1")
	if err == nil {
		t.Fatal("Checkout seharusnya ditolak — sudah ada invoice pending")
	}
	if apiErrStatus(t, err) != 409 {
		t.Fatalf("status error = %d, want 409 (conflict)", apiErrStatus(t, err))
	}
	if pay.createChargeCalls != 0 {
		t.Fatalf("CreateCharge tidak boleh dipanggil sama sekali, got %d call(s)", pay.createChargeCalls)
	}
	if len(repo.invs) != 1 {
		t.Fatalf("tidak boleh ada invoice baru tercipta, got %d invoice(s)", len(repo.invs))
	}
}

func TestCheckoutLosesRaceAgainstConcurrentInvoice(t *testing.T) {
	// Simulates two checkout requests landing at nearly the same instant: the app-level
	// pre-check (GetPendingInvoice) sees nothing yet, but the DB-level unique constraint
	// (migration 000025) rejects the INSERT because the other request won first.
	repo := newFakeRepo()
	repo.plans["plan-1"] = domain.Plan{ID: "plan-1", Code: "BASIC", Name: "Basic", Price: 50_000, PeriodDays: 30, IsActive: true}
	repo.forceCreateInvoiceErr = domain.ErrInvoiceAlreadyPending
	pay := &fakePaymentClient{}
	svc := NewService(repo, pay)

	_, err := svc.Checkout(context.Background(), "store-1", "plan-1")
	if err == nil {
		t.Fatal("Checkout seharusnya gagal — kalah race melawan invoice yang baru tercipta")
	}
	if apiErrStatus(t, err) != 409 {
		t.Fatalf("status error = %d, want 409 (conflict)", apiErrStatus(t, err))
	}
	if pay.createChargeCalls != 0 {
		t.Fatalf("CreateCharge tidak boleh dipanggil ketika CreateInvoice sudah gagal, got %d call(s)", pay.createChargeCalls)
	}
}

func TestCheckoutRejectsHiddenPlanForFreshOptIn(t *testing.T) {
	repo := newFakeRepo()
	repo.plans["legacy"] = domain.Plan{ID: "legacy", Code: "LEGACY", Name: "Legacy", Price: 10_000, PeriodDays: 30, IsActive: false}
	svc := NewService(repo, &fakePaymentClient{})

	_, err := svc.Checkout(context.Background(), "store-1", "legacy")
	if err == nil || apiErrStatus(t, err) != 422 {
		t.Fatalf("checkout ke hidden plan tanpa langganan berjalan seharusnya 422, got err=%v", err)
	}
}

func TestCheckoutRejectsSwitchingAwayFromRenewalOnlyPlan(t *testing.T) {
	repo := newFakeRepo()
	repo.plans["legacy"] = domain.Plan{ID: "legacy", Code: "LEGACY", Name: "Legacy", Price: 10_000, PeriodDays: 30, IsActive: true, RenewalOnly: true}
	repo.plans["basic"] = domain.Plan{ID: "basic", Code: "BASIC", Name: "Basic", Price: 20_000, PeriodDays: 30, IsActive: true}
	repo.subs["store-1"] = domain.Subscription{StoreID: "store-1", PlanID: "legacy", Status: "active"}
	svc := NewService(repo, &fakePaymentClient{})

	_, err := svc.Checkout(context.Background(), "store-1", "basic")
	if err == nil || apiErrStatus(t, err) != 422 {
		t.Fatalf("checkout beralih dari plan RenewalOnly seharusnya 422, got err=%v", err)
	}
}

func TestCheckoutPlanNotFound(t *testing.T) {
	svc := NewService(newFakeRepo(), &fakePaymentClient{})
	_, err := svc.Checkout(context.Background(), "store-1", "missing")
	if err == nil || apiErrStatus(t, err) != 404 {
		t.Fatalf("checkout plan tidak ada seharusnya 404, got err=%v", err)
	}
}

// ── mapElProofTerminalStatus (pure) ──────────────────────────────────────────────────────────

func TestMapElProofTerminalStatus(t *testing.T) {
	cases := []struct {
		raw    string
		want   string
		wantOK bool
	}{
		{"expired", "expired", true},
		{"EXPIRED", "expired", true},
		{"  failed  ", "failed", true},
		{"refund", "failed", true},
		{"unpaid", "", false},
		{"paid", "", false}, // ApplyWebhookEvent never routes a "paid" raw status here (see ev.Paid branch)
		{"", "", false},
		{"something-new-elproof-adds-later", "", false},
	}
	for _, c := range cases {
		got, ok := mapElProofTerminalStatus(c.raw)
		if got != c.want || ok != c.wantOK {
			t.Errorf("mapElProofTerminalStatus(%q) = (%q, %v), want (%q, %v)", c.raw, got, ok, c.want, c.wantOK)
		}
	}
}

// ── ApplyWebhookEvent ─────────────────────────────────────────────────────────────────────────

func TestApplyWebhookEventPaidExtendsSubscription(t *testing.T) {
	repo := newFakeRepo()
	repo.plans["plan-1"] = domain.Plan{ID: "plan-1", PeriodDays: 30}
	repo.invs["inv-1"] = domain.Invoice{ID: "inv-1", StoreID: "store-1", PlanID: "plan-1", Status: "pending", Provider: "elproof"}
	svc := NewService(repo, &fakePaymentClient{})

	err := svc.ApplyWebhookEvent(context.Background(), paymentclient.WebhookEvent{EventID: "inv-1", OrderRef: "inv-1", Paid: true})
	if err != nil {
		t.Fatalf("ApplyWebhookEvent: %v", err)
	}
	if got := repo.invoice(t, "inv-1").Status; got != "paid" {
		t.Fatalf("invoice.Status = %q, want paid", got)
	}
	if _, ok := repo.subs["store-1"]; !ok {
		t.Fatal("store_subscriptions seharusnya ter-upsert setelah invoice lunas")
	}
}

func TestApplyWebhookEventUnpaidExpiredClosesInvoice(t *testing.T) {
	repo := newFakeRepo()
	repo.invs["inv-1"] = domain.Invoice{ID: "inv-1", StoreID: "store-1", Status: "pending", Provider: "elproof"}
	pay := &fakePaymentClient{checkStatusFn: func(ref string) (paymentclient.ChargeStatus, error) {
		return paymentclient.ChargeStatus{Paid: false, RawStatus: "expired"}, nil
	}}
	svc := NewService(repo, pay)

	if err := svc.ApplyWebhookEvent(context.Background(), paymentclient.WebhookEvent{EventID: "inv-1", OrderRef: "inv-1", Paid: false}); err != nil {
		t.Fatalf("ApplyWebhookEvent: %v", err)
	}
	if got := repo.invoice(t, "inv-1").Status; got != "expired" {
		t.Fatalf("invoice.Status = %q, want expired", got)
	}
}

func TestApplyWebhookEventUnpaidStillUnpaidLeavesInvoicePending(t *testing.T) {
	// The critical case the doc comment on ApplyWebhookEvent calls out: ElProof's payload only
	// carries a `paid` boolean, so paid=false must NOT be treated as a terminal outcome by
	// itself — only markTerminalIfNeeded's live CheckStatus lookup decides that.
	repo := newFakeRepo()
	repo.invs["inv-1"] = domain.Invoice{ID: "inv-1", StoreID: "store-1", Status: "pending", Provider: "elproof"}
	pay := &fakePaymentClient{checkStatusFn: func(ref string) (paymentclient.ChargeStatus, error) {
		return paymentclient.ChargeStatus{Paid: false, RawStatus: "unpaid"}, nil
	}}
	svc := NewService(repo, pay)

	if err := svc.ApplyWebhookEvent(context.Background(), paymentclient.WebhookEvent{EventID: "inv-1", OrderRef: "inv-1", Paid: false}); err != nil {
		t.Fatalf("ApplyWebhookEvent: %v", err)
	}
	if got := repo.invoice(t, "inv-1").Status; got != "pending" {
		t.Fatalf("invoice.Status = %q, want pending (masih menunggu)", got)
	}
}

// ── ReconcilePending ──────────────────────────────────────────────────────────────────────────

func TestReconcilePendingResolvesPaidExpiredAndSkipsErrors(t *testing.T) {
	repo := newFakeRepo()
	// CreatedAt must be recent (not the zero value) — otherwise reconcileOne's
	// reconcileForceFailAfter deadline would force-fail inv-check-fails within this same tick,
	// since time.Since(zero value) is enormous. See TestReconcilePendingForceFailsStuckOldInvoice
	// for that behavior specifically.
	now := time.Now()
	repo.invs["inv-paid"] = domain.Invoice{ID: "inv-paid", StoreID: "store-1", Status: "pending", Provider: "elproof", CreatedAt: now}
	repo.invs["inv-expired"] = domain.Invoice{ID: "inv-expired", StoreID: "store-2", Status: "pending", Provider: "elproof", CreatedAt: now}
	repo.invs["inv-still-unpaid"] = domain.Invoice{ID: "inv-still-unpaid", StoreID: "store-3", Status: "pending", Provider: "elproof", CreatedAt: now}
	repo.invs["inv-check-fails"] = domain.Invoice{ID: "inv-check-fails", StoreID: "store-4", Status: "pending", Provider: "elproof", CreatedAt: now}
	repo.invs["inv-not-elproof"] = domain.Invoice{ID: "inv-not-elproof", StoreID: "store-5", Status: "pending", Provider: "tripay", CreatedAt: now}
	repo.invs["inv-already-paid"] = domain.Invoice{ID: "inv-already-paid", StoreID: "store-6", Status: "paid", Provider: "elproof", CreatedAt: now}

	pay := &fakePaymentClient{checkStatusFn: func(ref string) (paymentclient.ChargeStatus, error) {
		switch {
		case strings.Contains(ref, "inv-paid"):
			return paymentclient.ChargeStatus{Paid: true}, nil
		case strings.Contains(ref, "inv-expired"):
			return paymentclient.ChargeStatus{Paid: false, RawStatus: "expired"}, nil
		case strings.Contains(ref, "inv-still-unpaid"):
			return paymentclient.ChargeStatus{Paid: false, RawStatus: "unpaid"}, nil
		case strings.Contains(ref, "inv-check-fails"):
			return paymentclient.ChargeStatus{}, errors.New("elproof: gagal memeriksa status (HTTP 500)")
		default:
			t.Fatalf("CheckStatus dipanggil untuk ref tak terduga: %q", ref)
			return paymentclient.ChargeStatus{}, nil
		}
	}}
	svc := NewService(repo, pay)

	checked, resolved, err := svc.ReconcilePending(context.Background())
	if err != nil {
		t.Fatalf("ReconcilePending: %v", err)
	}
	// Only the 4 provider=elproof + status=pending invoices are candidates — inv-not-elproof
	// and inv-already-paid must never even reach CheckStatus (enforced by the t.Fatalf above).
	if checked != 4 {
		t.Fatalf("checked = %d, want 4", checked)
	}
	if resolved != 2 { // inv-paid (paid) + inv-expired (terminal) resolved; still-unpaid and check-fails did not
		t.Fatalf("resolved = %d, want 2", resolved)
	}
	if got := repo.invoice(t, "inv-paid").Status; got != "paid" {
		t.Fatalf("inv-paid.Status = %q, want paid", got)
	}
	if got := repo.invoice(t, "inv-expired").Status; got != "expired" {
		t.Fatalf("inv-expired.Status = %q, want expired", got)
	}
	if got := repo.invoice(t, "inv-still-unpaid").Status; got != "pending" {
		t.Fatalf("inv-still-unpaid.Status = %q, want pending (masih menunggu)", got)
	}
	if got := repo.invoice(t, "inv-check-fails").Status; got != "pending" {
		t.Fatalf("inv-check-fails.Status = %q, want pending (gagal cek, coba lagi tick berikutnya)", got)
	}
}

func TestReconcilePendingRespectsBatchLimit(t *testing.T) {
	// Regression test for the performance fix: one tick must never re-check the ENTIRE backlog
	// unbounded — it should stop at reconcileBatchLimit even when more pending invoices exist.
	repo := newFakeRepo()
	for i := 0; i < reconcileBatchLimit+20; i++ {
		id := fakeInvoiceID(i)
		repo.invs[id] = domain.Invoice{ID: id, StoreID: id, Status: "pending", Provider: "elproof"}
	}
	pay := &fakePaymentClient{checkStatusFn: func(ref string) (paymentclient.ChargeStatus, error) {
		return paymentclient.ChargeStatus{Paid: false, RawStatus: "unpaid"}, nil
	}}
	svc := NewService(repo, pay)

	checked, _, err := svc.ReconcilePending(context.Background())
	if err != nil {
		t.Fatalf("ReconcilePending: %v", err)
	}
	if checked != reconcileBatchLimit {
		t.Fatalf("checked = %d, want exactly reconcileBatchLimit=%d", checked, reconcileBatchLimit)
	}
}

func TestReconcilePendingForceFailsStuckOldInvoice(t *testing.T) {
	// Regression test: an invoice whose ElProof charge was never actually created (e.g. both
	// CreateCharge AND Checkout's cleanup write failed) has no possible resolution — CheckStatus
	// will error forever. Past reconcileForceFailAfter it must be force-resolved locally instead
	// of retried every tick indefinitely with zero possible outcome.
	repo := newFakeRepo()
	repo.invs["inv-stuck"] = domain.Invoice{
		ID: "inv-stuck", StoreID: "store-1", Status: "pending", Provider: "elproof",
		CreatedAt: time.Now().Add(-(reconcileForceFailAfter + time.Hour)),
	}
	pay := &fakePaymentClient{checkStatusFn: func(ref string) (paymentclient.ChargeStatus, error) {
		return paymentclient.ChargeStatus{}, errors.New("elproof: order_ref tidak ditemukan (HTTP 404)")
	}}
	svc := NewService(repo, pay)

	checked, resolved, err := svc.ReconcilePending(context.Background())
	if err != nil {
		t.Fatalf("ReconcilePending: %v", err)
	}
	if checked != 1 || resolved != 1 {
		t.Fatalf("checked=%d resolved=%d, want 1, 1 (force-failed)", checked, resolved)
	}
	if got := repo.invoice(t, "inv-stuck").Status; got != "failed" {
		t.Fatalf("invoice.Status = %q, want failed", got)
	}
}

func TestReconcilePendingDoesNotForceFailRecentInvoiceOnCheckError(t *testing.T) {
	// The counterpart to the above: a RECENT invoice hitting a transient CheckStatus error
	// (network blip, ElProof briefly down) must NOT be force-failed — it still has a realistic
	// chance to resolve normally on a later tick.
	repo := newFakeRepo()
	repo.invs["inv-recent"] = domain.Invoice{ID: "inv-recent", StoreID: "store-1", Status: "pending", Provider: "elproof", CreatedAt: time.Now()}
	pay := &fakePaymentClient{checkStatusFn: func(ref string) (paymentclient.ChargeStatus, error) {
		return paymentclient.ChargeStatus{}, errors.New("elproof: request gagal terkirim (timeout)")
	}}
	svc := NewService(repo, pay)

	_, resolved, err := svc.ReconcilePending(context.Background())
	if err != nil {
		t.Fatalf("ReconcilePending: %v", err)
	}
	if resolved != 0 {
		t.Fatalf("resolved = %d, want 0 (invoice baru, belum boleh di-force-fail)", resolved)
	}
	if got := repo.invoice(t, "inv-recent").Status; got != "pending" {
		t.Fatalf("invoice.Status = %q, want pending", got)
	}
}

func fakeInvoiceID(i int) string {
	return "inv-" + strconv.Itoa(i)
}
