// Package application holds the subscription module's use cases — tenant (store) billing to
// the elkasir platform. It reuses the SAME QRIS gateway as selforder (paymentclient.Client)
// but owns 100% of its own persistence; no table, row, or webhook state is shared with
// selforder's self_orders/payments.
package application

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	paymentclient "github.com/elkasir/api/internal/modules/payment/contracts"
	subscriptionclient "github.com/elkasir/api/internal/modules/subscription/contracts"
	"github.com/elkasir/api/internal/modules/subscription/domain"
	"github.com/elkasir/api/internal/modules/subscription/infrastructure"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
)

type Service struct {
	repo     *infrastructure.Repo
	payments paymentclient.Client
}

func NewService(repo *infrastructure.Repo, paymentClient paymentclient.Client) *Service {
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

	invID := id.New()
	charge, err := s.payments.CreateCharge(ctx, paymentclient.AppSubscribe, storeID, invID, plan.Price)
	if err != nil {
		return CheckoutResult{}, httpx.Internal("Gagal membuat QR pembayaran: " + err.Error())
	}
	inv, err := s.repo.CreateInvoice(ctx, invID, storeID, planID, charge.Provider, charge.ProviderRef, plan.Price)
	if err != nil {
		return CheckoutResult{}, err
	}
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
func (s *Service) ApplyWebhookEvent(ctx context.Context, ev paymentclient.WebhookEvent) error {
	if !ev.Paid {
		return nil
	}
	return s.repo.MarkInvoicePaidAndExtend(ctx, ev.OrderRef, time.Now().UTC())
}

func toInvoiceDTO(inv domain.Invoice) InvoiceDTO {
	return InvoiceDTO{
		ID: inv.ID, PlanID: inv.PlanID, Amount: inv.Amount, Status: inv.Status,
		PeriodStart: inv.PeriodStart, PeriodEnd: inv.PeriodEnd, CreatedAt: inv.CreatedAt,
	}
}
