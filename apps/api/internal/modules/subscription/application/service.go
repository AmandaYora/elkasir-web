// Package application holds the subscription module's use cases — tenant (store) billing to
// the elkasir platform. It reuses the SAME QRIS gateway as selforder (paymentclient.Client)
// but owns 100% of its own persistence; no table, row, or webhook state is shared with
// selforder's self_orders/payments.
package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	"github.com/elkasir/api/internal/modules/subscription/domain"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
)

// invoiceRepo is the subset of *infrastructure.Repo this service depends on — declared here
// (not in `infrastructure`) purely as a test seam: *infrastructure.Repo satisfies it
// structurally with zero changes on its side, and unit tests substitute a fake in-memory
// implementation instead of standing up a real MySQL instance (see service_test.go).
type invoiceRepo interface {
	ListActivePlans(ctx context.Context) ([]domain.Plan, error)
	ListAllPlans(ctx context.Context) ([]domain.Plan, error)
	CreatePlan(ctx context.Context, planID, code, name string, price int64, periodDays int32, isActive bool) error
	UpdatePlan(ctx context.Context, planID, name string, price int64, periodDays int32, isActive bool) (int64, error)
	GetPlan(ctx context.Context, planID string) (domain.Plan, error)
	PlatformRevenue(ctx context.Context) (int64, error)
	GetByStore(ctx context.Context, storeID string) (domain.Subscription, error)
	GetPendingInvoice(ctx context.Context, storeID string) (domain.Invoice, error)
	CreateInvoice(ctx context.Context, invID, storeID, planID, provider, providerRef string, amount int64) (domain.Invoice, error)
	MarkInvoiceTerminal(ctx context.Context, invoiceID, status string) error
	SetInvoiceProviderRef(ctx context.Context, invoiceID, providerRef string) error
	ListInvoices(ctx context.Context, storeID string, limit, offset int32) ([]domain.Invoice, int64, error)
	MarkInvoicePaidAndExtend(ctx context.Context, invoiceID string, now time.Time) error
	ListPendingElProof(ctx context.Context, limit int32) ([]domain.Invoice, error)
}

type Service struct {
	repo     invoiceRepo
	payments paymentclient.Client
}

func NewService(repo invoiceRepo, paymentClient paymentclient.Client) *Service {
	return &Service{repo: repo, payments: paymentClient}
}

// ── DTO ──────────────────────────────────────────────────────
// Plans use the subscriptionclient.Plan/PlanInput types directly (no separate presentation
// DTO) — this module's own tenant-facing routes and the platform contract need the identical
// shape, and duplicating it would just be two structs kept in sync by hand.

// SubscriptionDTO is an alias of subscriptionclient.Subscription (not a separate struct) —
// this makes Current's existing return type automatically satisfy subscriptionclient.Client's
// Current method (added in Phase B1.5) with no change to Current's body or to this module's
// own presentation handler that already consumes SubscriptionDTO.
type SubscriptionDTO = subscriptionclient.Subscription

type InvoiceDTO struct {
	ID          string     `json:"id"`
	PlanID      string     `json:"planId"`
	Amount      int64      `json:"amount"`
	Status      string     `json:"status"`
	PeriodStart *time.Time `json:"periodStart,omitempty"`
	PeriodEnd   *time.Time `json:"periodEnd,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// CheckoutResult mirrors selforder's PlaceResult shape (QR data for the customer/owner to scan).
type CheckoutResult struct {
	Invoice    InvoiceDTO `json:"invoice"`
	QRString   string     `json:"qrString,omitempty"`
	QRImageURL string     `json:"qrImageUrl,omitempty"`
	Simulated  bool       `json:"simulated,omitempty"`
}

func (s *Service) ListPlans(ctx context.Context) ([]subscriptionclient.Plan, error) {
	plans, err := s.repo.ListActivePlans(ctx)
	if err != nil {
		return nil, err
	}
	return toPlans(plans), nil
}

// ── Manajemen plan (platform/superadmin — implements subscriptionclient.Client) ──────────
func validatePlanInput(in subscriptionclient.PlanInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return httpx.Validation("Nama paket wajib diisi.")
	}
	if in.Price <= 0 {
		return httpx.Validation("Harga paket harus lebih dari 0.")
	}
	if in.PeriodDays <= 0 {
		return httpx.Validation("Periode paket (hari) harus lebih dari 0.")
	}
	return nil
}

// ListAllPlans returns every plan (termasuk nonaktif) — dashboard superadmin.
func (s *Service) ListAllPlans(ctx context.Context) ([]subscriptionclient.Plan, error) {
	plans, err := s.repo.ListAllPlans(ctx)
	if err != nil {
		return nil, err
	}
	return toPlans(plans), nil
}

// CreatePlan menambah paket baru ke katalog (platform/superadmin only).
func (s *Service) CreatePlan(ctx context.Context, in subscriptionclient.PlanInput) (subscriptionclient.Plan, error) {
	if err := validatePlanInput(in); err != nil {
		return subscriptionclient.Plan{}, err
	}
	if strings.TrimSpace(in.Code) == "" {
		return subscriptionclient.Plan{}, httpx.Validation("Kode paket wajib diisi.")
	}
	planID := id.New()
	if err := s.repo.CreatePlan(ctx, planID, in.Code, in.Name, in.Price, in.PeriodDays, in.IsActive); err != nil {
		return subscriptionclient.Plan{}, err
	}
	return subscriptionclient.Plan{ID: planID, Code: in.Code, Name: in.Name, Price: in.Price, PeriodDays: in.PeriodDays, IsActive: in.IsActive}, nil
}

// UpdatePlan mengubah harga/periode/status aktif sebuah paket (platform/superadmin only).
// Code sengaja tidak bisa diubah — itu identitas stabil paket, dipakai sebagai referensi jangka
// panjang (mis. laporan/integrasi lain di masa depan).
func (s *Service) UpdatePlan(ctx context.Context, planID string, in subscriptionclient.PlanInput) (subscriptionclient.Plan, error) {
	if err := validatePlanInput(in); err != nil {
		return subscriptionclient.Plan{}, err
	}
	n, err := s.repo.UpdatePlan(ctx, planID, in.Name, in.Price, in.PeriodDays, in.IsActive)
	if err != nil {
		return subscriptionclient.Plan{}, err
	}
	if n == 0 {
		return subscriptionclient.Plan{}, httpx.NotFound("Paket langganan tidak ditemukan.")
	}
	p, err := s.repo.GetPlan(ctx, planID)
	if err != nil {
		return subscriptionclient.Plan{}, err
	}
	return subscriptionclient.Plan{ID: p.ID, Code: p.Code, Name: p.Name, Price: p.Price, PeriodDays: p.PeriodDays, IsActive: p.IsActive}, nil
}

// PlatformRevenue implements subscriptionclient.Client.
func (s *Service) PlatformRevenue(ctx context.Context) (int64, error) {
	return s.repo.PlatformRevenue(ctx)
}

func toPlans(plans []domain.Plan) []subscriptionclient.Plan {
	out := make([]subscriptionclient.Plan, 0, len(plans))
	for _, p := range plans {
		out = append(out, subscriptionclient.Plan{
			ID: p.ID, Code: p.Code, Name: p.Name, Price: p.Price, PeriodDays: p.PeriodDays,
			IsActive: p.IsActive, RenewalOnly: p.RenewalOnly,
		})
	}
	return out
}

// Current returns the store's subscription status. A store that has never checked out
// returns status "none" (not an error) — it simply hasn't picked a plan yet. PlanName is
// resolved via GetPlan (not the active-only ListActivePlans a tenant's plan-picker uses) so a
// subscriber on a hidden/legacy plan (§2.15 backfill) still sees its name on their own page.
func (s *Service) Current(ctx context.Context, storeID string) (SubscriptionDTO, error) {
	sub, err := s.repo.GetByStore(ctx, storeID)
	if errors.Is(err, sql.ErrNoRows) {
		return SubscriptionDTO{Status: "none"}, nil
	}
	if err != nil {
		return SubscriptionDTO{}, err
	}
	dto := SubscriptionDTO{
		Status: sub.Status, PlanID: sub.PlanID,
		CurrentPeriodStart: sub.CurrentPeriodStart, CurrentPeriodEnd: sub.CurrentPeriodEnd,
	}
	if sub.PlanID != "" {
		if plan, err := s.repo.GetPlan(ctx, sub.PlanID); err == nil {
			dto.PlanName = plan.Name
			dto.PlanPrice = plan.Price
			dto.PlanPeriodDays = plan.PeriodDays
			dto.PlanRenewalOnly = plan.RenewalOnly
		}
	}
	return dto, nil
}

// Checkout starts a new billing charge for a plan: creates a pending invoice + a QRIS charge
// via the SAME gateway selforder uses, tagged with this module's app id (§9.1.4 — replacing
// the old "sub_" ref-prefix convention) so the payment module's registry-driven webhook
// dispatcher can route the callback back here without either module importing the other.
//
// Two plan-switching rules enforced here (not just UI-level hiding, since a client could call
// this endpoint directly with any planID): a hidden plan (IsActive=false, e.g. the "Premium
// Contributor" legacy-backfill plan) can only be checked out into by a store already ON it
// (i.e. a renewal, planID == current plan) — nobody can freshly opt into a hidden plan. And a
// RenewalOnly plan (same "Premium Contributor" plan today) can never be switched AWAY from —
// once assigned, the only valid checkout for that store is the SAME planID, forever.
func (s *Service) Checkout(ctx context.Context, storeID, planID string) (CheckoutResult, error) {
	plan, err := s.repo.GetPlan(ctx, planID)
	if errors.Is(err, sql.ErrNoRows) {
		return CheckoutResult{}, httpx.NotFound("Paket langganan tidak ditemukan.")
	}
	if err != nil {
		return CheckoutResult{}, err
	}
	if err := s.validatePlanSwitch(ctx, storeID, planID, plan); err != nil {
		return CheckoutResult{}, err
	}
	// Fast-path guard against double-submit (impatient owner double-tapping "Bayar", or a
	// client retry racing the first request): friendly, informative rejection in the common
	// (non-concurrent) case. The DB-level unique constraint below is what actually closes the
	// race for two requests landing at the same instant — this check alone would still have a
	// TOCTOU gap.
	if pending, err := s.repo.GetPendingInvoice(ctx, storeID); err == nil {
		return CheckoutResult{}, httpx.Conflict(fmt.Sprintf(
			"Anda sudah memiliki tagihan (%s) yang belum lunas. Selesaikan atau tunggu hingga kedaluwarsa sebelum membuat tagihan baru.", pending.ID))
	} else if !errors.Is(err, sql.ErrNoRows) {
		return CheckoutResult{}, err
	}

	invID := id.New()
	// Create the invoice locally FIRST, before the external gateway call. AppSubscribe always
	// routes to ElProof (payment/infrastructure/client.go hardcodes Provider="elproof" for this
	// appID in both simulation and live mode), so the provider is known upfront — no need to
	// wait for the charge response. This ordering matters: the old order (charge first, invoice
	// second) meant a transient failure persisting the invoice AFTER the ElProof charge already
	// succeeded left an orphaned external charge with no matching Elkasir record, forever. Now
	// the worst case is a local invoice with no charge behind it, which is closed out
	// immediately below instead of being left for the reconciler to poll forever.
	inv, err := s.repo.CreateInvoice(ctx, invID, storeID, planID, "elproof", "", plan.Price)
	if errors.Is(err, domain.ErrInvoiceAlreadyPending) {
		// Lost the race against a concurrent checkout that the pre-check above couldn't see yet.
		return CheckoutResult{}, httpx.Conflict("Anda sudah memiliki tagihan yang belum lunas. Selesaikan atau tunggu hingga kedaluwarsa sebelum membuat tagihan baru.")
	}
	if err != nil {
		return CheckoutResult{}, err
	}

	charge, err := s.payments.CreateCharge(ctx, paymentclient.AppSubscribe, storeID, invID, plan.Price)
	if err != nil {
		// Detached from ctx's cancellation on purpose: if CreateCharge failed BECAUSE ctx was
		// cancelled/timed out (client disconnected, request deadline exceeded — a realistic way
		// for an outbound gateway call to fail), reusing the same ctx here would make this
		// cleanup write fail too, leaving the invoice stuck 'pending' with no charge behind it
		// forever. ReconcilePending's force-fail deadline (see below) is the last-resort backstop
		// if this write somehow still fails for another reason.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		if merr := s.repo.MarkInvoiceTerminal(cleanupCtx, invID, "failed"); merr != nil {
			slog.Warn("subscription: gagal menandai invoice gagal setelah charge gateway gagal", "invoiceId", invID, "err", merr)
		}
		cancel()
		return CheckoutResult{}, httpx.Internal("Gagal membuat QR pembayaran: " + err.Error())
	}
	// providerRef is informational only for ElProof invoices (status checks key off the
	// invoice's own ID as orderRef — see payment/infrastructure/elproof.go's checkStatus) —
	// never block checkout success on persisting it.
	if charge.ProviderRef != "" {
		if err := s.repo.SetInvoiceProviderRef(ctx, invID, charge.ProviderRef); err != nil {
			slog.Warn("subscription: gagal menyimpan providerRef ElProof", "invoiceId", invID, "err", err)
		}
	}
	inv.ProviderRef = charge.ProviderRef
	return CheckoutResult{
		Invoice: toInvoiceDTO(inv), QRString: charge.QRString, QRImageURL: charge.QRImageURL, Simulated: charge.Simulated,
	}, nil
}

// validatePlanSwitch enforces Checkout's two plan-switching rules (see its doc comment) for
// every request that isn't a plain renewal (planID == the store's current plan, always allowed).
func (s *Service) validatePlanSwitch(ctx context.Context, storeID, planID string, plan domain.Plan) error {
	currentSub, err := s.repo.GetByStore(ctx, storeID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && currentSub.PlanID == planID {
		return nil // renewal — always allowed regardless of IsActive/RenewalOnly
	}
	if !plan.IsActive {
		return httpx.Unprocessable("Paket ini tidak tersedia untuk dipilih.")
	}
	if err == nil && currentSub.PlanID != "" {
		if currentPlan, cerr := s.repo.GetPlan(ctx, currentSub.PlanID); cerr == nil && currentPlan.RenewalOnly {
			return httpx.Unprocessable("Paket Anda saat ini hanya bisa diperpanjang, tidak bisa diganti ke paket lain.")
		}
	}
	return nil
}

func (s *Service) ListInvoices(ctx context.Context, storeID string, limit, offset int32) ([]InvoiceDTO, int64, error) {
	rows, total, err := s.repo.ListInvoices(ctx, storeID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]InvoiceDTO, 0, len(rows))
	for _, inv := range rows {
		out = append(out, toInvoiceDTO(inv))
	}
	return out, total, nil
}

// ApplyWebhookEvent applies an already-verified/parsed/idempotency-checked gateway event to
// this module's own invoice + subscription period. The payment module's registry-driven
// dispatcher (§9.1.4/§9.1.5) has already resolved that this event belongs to
// paymentclient.AppSubscribe before calling this — ev.OrderRef is the raw invoice id passed to
// CreateCharge at checkout time, no prefix stripping needed (replaces the old "sub_" convention).
//
// ev.Paid=false is NOT necessarily "still pending" — ElProof's relay fires on ANY status change
// (paid, expired, failed, refund), but its payload only ever carries a `paid` boolean (see
// docs/PAYMENT_INTEGRATION_GUIDE.md §8: "paid cuma boolean — untuk status lengkap, panggil §7").
// So on paid=false we look up the real status via CheckStatus to learn whether this is a genuine
// terminal outcome (expired/failed/refund) or just still-unpaid — otherwise an expired charge's
// webhook would silently no-op and its invoice would stay 'pending' forever.
func (s *Service) ApplyWebhookEvent(ctx context.Context, ev paymentclient.WebhookEvent) error {
	if ev.Paid {
		return s.repo.MarkInvoicePaidAndExtend(ctx, ev.OrderRef, time.Now().UTC())
	}
	status, err := s.payments.CheckStatus(ctx, paymentclient.AppSubscribe, ev.OrderRef)
	if err != nil {
		return err
	}
	_, err = s.markTerminalIfNeeded(ctx, ev.OrderRef, status.RawStatus)
	return err
}

// mapElProofTerminalStatus maps ElProof's raw status string (unpaid | paid | expired | failed |
// refund, per its own docs) to subscription_invoices' own status enum ('expired' | 'failed') —
// pure, no I/O, so it's unit-testable in isolation from the DB. ok=false for a still-"unpaid" raw
// status (deliberately left alone: this invoice hasn't resolved yet) or any value this mapping
// doesn't recognize — the latter fails safe (no-op) rather than guessing, in case ElProof ever
// adds a new status value.
func mapElProofTerminalStatus(rawStatus string) (target string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(rawStatus)) {
	case "expired":
		return "expired", true
	case "failed", "refund":
		return "failed", true
	default:
		return "", false
	}
}

// markTerminalIfNeeded applies mapElProofTerminalStatus's mapping, ONLY for genuinely terminal
// non-paid outcomes — returns whether it actually applied a terminal state (used by
// ReconcilePending to report an accurate resolved count).
func (s *Service) markTerminalIfNeeded(ctx context.Context, invoiceID, rawStatus string) (bool, error) {
	target, ok := mapElProofTerminalStatus(rawStatus)
	if !ok {
		return false, nil
	}
	if err := s.repo.MarkInvoiceTerminal(ctx, invoiceID, target); err != nil {
		return false, err
	}
	return true, nil
}

const (
	// reconcileBatchLimit caps how many invoices one tick re-checks against ElProof — mirrors
	// ElProof's own reconcileBatchLimit on its sweep — so a large backlog (e.g. after an
	// extended ElProof outage spanning many tenants) can't turn one tick into an unbounded burst
	// of outbound requests.
	reconcileBatchLimit = 50
	// reconcileConcurrency bounds how many CheckStatus calls run in parallel per tick. ElProof's
	// only documented rate limit (docs/PAYMENT_INTEGRATION_GUIDE.md §4) is 10/min on token
	// exchange specifically — irrelevant here since ensureToken caches the token per gateway
	// instance behind a mutex (payment/infrastructure/elproof.go), so concurrent CheckStatus
	// calls share one cached token regardless of how many run at once. A small worker pool still
	// keeps this polite to ElProof and bounds Elkasir's own outbound connection count.
	reconcileConcurrency = 8
	// reconcileForceFailAfter bounds how long an invoice can sit 'pending' while CheckStatus
	// keeps ERRORING (never once returning a definitive answer, paid or otherwise) before it's
	// force-resolved locally as 'failed' — mirrors ElProof's own sweep, which force-resolves a
	// dispatch past its charge's expiry+grace rather than polling a dead end forever (see the
	// elproof sibling repo's payment/infrastructure/client.go reconcileExpiryGrace/
	// reconcileMaxAgeFallback). Without this, an invoice whose ElProof charge was never actually
	// created (e.g. Checkout's own cleanup write above also failed) would retry every tick,
	// forever, with zero possible outcome. Comfortably past ElProof/Tripay's own QRIS expiry
	// window (routinely ≤24h) so a genuinely-still-open charge is never force-failed
	// prematurely — this only fires when CheckStatus has NEVER once succeeded for this invoice
	// across its entire lifetime.
	reconcileForceFailAfter = 25 * time.Hour
)

// ReconcilePending polls ElProof's status-check endpoint for every invoice still `pending`
// (PLAN.md §11 Part C) — a fallback for the rare case ElProof's webhook relay (best-effort,
// single-attempt, no retry) never arrives, AND the mechanism that actually closes out invoices
// whose charge expired/failed/refunded on ElProof's side (see markTerminalIfNeeded above — without
// this, an expired charge would leave its invoice stuck 'pending' forever). Uses the invoice's
// own ID as the ref, since that is exactly the orderRef sent to ElProof at Checkout time
// (CreateChannelCharge) — ElProof's status endpoint is keyed by orderRef, not its own providerRef
// (see infrastructure/elproof.go). Safe to call repeatedly/concurrently with a real webhook
// landing: both MarkInvoicePaidAndExtend and MarkInvoiceTerminal are guarded by status='pending'
// at the SQL level, so whichever resolution arrives first wins and the other becomes a no-op.
//
// Each invoice's check is an independent HTTP round-trip to ElProof, so they run concurrently
// (bounded by reconcileConcurrency) rather than one at a time — sequential polling would make one
// tick's latency scale linearly with backlog size for no correctness benefit. Returns
// (checked, resolved) so the caller (subscription.module.go's ticker) can log a useful summary,
// mirroring ElProof's own StartReconciler. Also mirrors ElProof's own sweep in having a deadline
// (reconcileForceFailAfter, see reconcileOne) past which a persistently-unreachable/unresolvable
// invoice is force-failed instead of retried forever with zero possible outcome.
func (s *Service) ReconcilePending(ctx context.Context) (checked, resolved int, err error) {
	pending, err := s.repo.ListPendingElProof(ctx, reconcileBatchLimit)
	if err != nil {
		return 0, 0, err
	}

	var (
		wg            sync.WaitGroup
		sem           = make(chan struct{}, reconcileConcurrency)
		resolvedCount int64
	)
	for _, inv := range pending {
		wg.Add(1)
		sem <- struct{}{}
		go func(inv domain.Invoice) {
			defer wg.Done()
			defer func() { <-sem }()
			if s.reconcileOne(ctx, inv) {
				atomic.AddInt64(&resolvedCount, 1)
			}
		}(inv)
	}
	wg.Wait()
	return len(pending), int(resolvedCount), nil
}

// reconcileOne checks and resolves a single invoice, returning whether it ended up resolved
// (paid, or a genuine terminal non-paid outcome) this round.
func (s *Service) reconcileOne(ctx context.Context, inv domain.Invoice) bool {
	status, err := s.payments.CheckStatus(ctx, paymentclient.AppSubscribe, inv.ID)
	if err != nil {
		// Satu invoice gagal diperiksa tidak boleh menghentikan reconciliasi invoice lain.
		slog.Warn("subscription: reconcile gagal memeriksa status ElProof", "invoiceId", inv.ID, "err", err)
		if time.Since(inv.CreatedAt) > reconcileForceFailAfter {
			if merr := s.repo.MarkInvoiceTerminal(ctx, inv.ID, "failed"); merr != nil {
				slog.Warn("subscription: reconcile gagal force-fail invoice macet", "invoiceId", inv.ID, "err", merr)
				return false
			}
			slog.Warn("subscription: invoice force-failed setelah gagal cek status berkepanjangan", "invoiceId", inv.ID, "age", time.Since(inv.CreatedAt).String())
			return true
		}
		return false
	}
	if status.Paid {
		if err := s.repo.MarkInvoicePaidAndExtend(ctx, inv.ID, time.Now().UTC()); err != nil {
			slog.Warn("subscription: reconcile gagal menandai lunas", "invoiceId", inv.ID, "err", err)
			return false
		}
		return true
	}
	wasTerminal, err := s.markTerminalIfNeeded(ctx, inv.ID, status.RawStatus)
	if err != nil {
		slog.Warn("subscription: reconcile gagal menandai status akhir", "invoiceId", inv.ID, "err", err)
		return false
	}
	return wasTerminal
}

func toInvoiceDTO(inv domain.Invoice) InvoiceDTO {
	return InvoiceDTO{
		ID: inv.ID, PlanID: inv.PlanID, Amount: inv.Amount, Status: inv.Status,
		PeriodStart: inv.PeriodStart, PeriodEnd: inv.PeriodEnd, CreatedAt: inv.CreatedAt,
	}
}
