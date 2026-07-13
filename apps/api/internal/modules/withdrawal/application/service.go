// Package application holds the withdrawal module's use cases — tenant-facing request/list, and
// (implementing withdrawalclient.Client) the superadmin claim/complete flow + balance
// reconciliation (PLAN.md §2.6/§2.7).
package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	platformuserclient "github.com/elkasir/api/internal/modules/platformuser/contracts"
	salesclient "github.com/elkasir/api/internal/modules/transaction/contracts"
	withdrawalclient "github.com/elkasir/api/internal/modules/withdrawal/contracts"
	"github.com/elkasir/api/internal/modules/withdrawal/domain"
	"github.com/elkasir/api/internal/modules/withdrawal/infrastructure"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
	"github.com/elkasir/api/internal/platform/mail"
)

type Service struct {
	repo          *infrastructure.Repo
	sales         salesclient.Client
	platformUsers platformuserclient.Client
	mailer        *mail.Sender
	publicBaseURL string
}

func NewService(repo *infrastructure.Repo, salesClient salesclient.Client, platformUsers platformuserclient.Client, mailer *mail.Sender, publicBaseURL string) *Service {
	return &Service{repo: repo, sales: salesClient, platformUsers: platformUsers, mailer: mailer, publicBaseURL: publicBaseURL}
}

// DTO is the tenant-facing API representation of a withdrawal (camelCase). RejectedReason is
// included so the tenant sees WHY a request failed (§2.11/F5); processedBy/claimedAt/processedAt
// are superadmin-side audit detail and deliberately not exposed here.
type DTO struct {
	ID             string    `json:"id"`
	Amount         int64     `json:"amount"`
	Bank           string    `json:"bank"`
	Account        string    `json:"account"`
	Holder         string    `json:"holder"`
	Status         string    `json:"status"`
	Reference      string    `json:"reference,omitempty"`
	RequestedBy    string    `json:"requestedBy,omitempty"`
	RejectedReason string    `json:"rejectedReason,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

func toDTO(w domain.Withdrawal) DTO {
	return DTO{
		ID: w.ID, Amount: w.Amount, Bank: w.Bank, Account: w.Account, Holder: w.Holder,
		Status: w.Status, Reference: w.Reference, RequestedBy: w.RequestedBy,
		RejectedReason: w.RejectedReason, CreatedAt: w.CreatedAt,
	}
}

func (s *Service) List(ctx context.Context, f domain.ListFilter) ([]DTO, int64, error) {
	rows, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, f.StoreID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DTO, 0, len(rows))
	for _, w := range rows {
		out = append(out, toDTO(w))
	}
	return out, total, nil
}

// Create validates the requested amount against the claimable balance (§2.6) before inserting
// the pending row.
func (s *Service) Create(ctx context.Context, p authcontract.Principal, in domain.Input) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	claimable, err := s.claimableBalance(ctx, p.StoreID)
	if err != nil {
		return DTO{}, err
	}
	if in.Amount > claimable {
		return DTO{}, httpx.Unprocessable("Jumlah pencairan melebihi saldo yang dapat dicairkan.")
	}
	wid := id.New()
	w, err := s.repo.Create(ctx, p.StoreID, wid, p.SubjectID, in)
	if err != nil {
		return DTO{}, err
	}
	// Best-effort, fire-and-forget (§2.10) — must never block, slow down, or fail this
	// response, and must never be visible to the tenant in any way.
	go s.notifyPlatformUsers(context.WithoutCancel(ctx))
	return toDTO(w), nil
}

// notifyPlatformUsers pings every active superadmin that a new withdrawal request is waiting —
// generic content only, no tenant name/amount/bank details (§2.10). Runs in its own goroutine;
// any failure here is logged, never surfaced to the tenant.
func (s *Service) notifyPlatformUsers(ctx context.Context) {
	users, err := s.platformUsers.List(ctx)
	if err != nil {
		slog.Warn("withdrawal: notify platform users", "err", err)
		return
	}
	link := s.publicBaseURL + "/platform/withdrawals"
	subject := "Permintaan pencairan baru menunggu ditinjau"
	body := fmt.Sprintf("Ada permintaan pencairan baru yang menunggu ditinjau. Buka Konsol Platform untuk meninjau: %s", link)
	for _, u := range users {
		if u.Status != "active" {
			continue
		}
		if err := s.mailer.Send(ctx, []string{u.Email}, subject, body); err != nil {
			slog.Warn("withdrawal: send notification failed", "to", u.Email, "err", err)
		}
	}
}

// Balance returns the tenant's own AvailableBalance (§2.6 — reconciliation-accurate, the figure
// shown on the tenant's Withdrawals page) for GET /withdrawals/balance.
func (s *Service) Balance(ctx context.Context, storeID string) (int64, error) {
	return s.AvailableBalance(ctx, storeID)
}

// ── withdrawalclient.Client implementation (superadmin claim/complete flow) ──────────────────

var _ withdrawalclient.Client = (*Service)(nil)

// AvailableBalance implements withdrawalclient.Client — §2.6: money hasn't left the gateway
// until a withdrawal is 'success', so this is what should sum against the real gateway balance.
func (s *Service) AvailableBalance(ctx context.Context, storeID string) (int64, error) {
	qris, err := s.sales.SelfOrderQrisRevenueForStore(ctx, storeID)
	if err != nil {
		return 0, err
	}
	success, err := s.repo.SumSuccessfulByStore(ctx, storeID)
	if err != nil {
		return 0, err
	}
	return qris - success, nil
}

// AvailableBalanceByTenant implements withdrawalclient.Client — same basis, all tenants,
// merged in Go (not SQL) since salesclient/transaction and withdrawal own different tables.
func (s *Service) AvailableBalanceByTenant(ctx context.Context) ([]withdrawalclient.TenantBalance, error) {
	qrisByTenant, err := s.sales.PlatformSelfOrderQrisRevenueByTenant(ctx)
	if err != nil {
		return nil, err
	}
	successByStore, err := s.repo.SumSuccessfulGroupedByStore(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]withdrawalclient.TenantBalance, 0, len(qrisByTenant))
	for _, t := range qrisByTenant {
		out = append(out, withdrawalclient.TenantBalance{StoreID: t.StoreID, Balance: t.Amount - successByStore[t.StoreID]})
	}
	return out, nil
}

// TotalSuccessfulWithdrawals implements withdrawalclient.Client.
func (s *Service) TotalSuccessfulWithdrawals(ctx context.Context) (int64, error) {
	return s.repo.SumSuccessfulAll(ctx)
}

// ListActive implements withdrawalclient.Client.
func (s *Service) ListActive(ctx context.Context) ([]withdrawalclient.Withdrawal, error) {
	rows, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return toClientList(rows), nil
}

// ListAll implements withdrawalclient.Client.
func (s *Service) ListAll(ctx context.Context, filter withdrawalclient.ListFilter) ([]withdrawalclient.Withdrawal, int64, error) {
	rows, total, err := s.repo.ListAll(ctx, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, err
	}
	return toClientList(rows), total, nil
}

// Claim implements withdrawalclient.Client — pending -> processing (§2.7). Runs the claimable
// check (§2.6) and the tenant-suspension check (§2.14) before the atomic status transition.
func (s *Service) Claim(ctx context.Context, id, actorID string) error {
	wd, err := s.repo.Get(ctx, id)
	if err != nil {
		return httpx.NotFound("Permintaan pencairan tidak ditemukan.")
	}
	suspended, err := s.repo.StoreSuspended(ctx, wd.StoreID)
	if err != nil {
		return err
	}
	if suspended {
		return httpx.Forbidden("Toko ini sedang dinonaktifkan; klaim tidak dapat diproses.")
	}
	claimable, err := s.claimableBalance(ctx, wd.StoreID)
	if err != nil {
		return err
	}
	if wd.Amount > claimable {
		return httpx.Unprocessable("Saldo tenant tidak lagi mencukupi untuk klaim ini.")
	}
	n, err := s.repo.Claim(ctx, id, actorID)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpx.Conflict("Permintaan ini sudah diklaim atau diproses pihak lain.")
	}
	return nil
}

// MarkSuccess implements withdrawalclient.Client — processing -> success (§2.7); actorID must
// be who claimed it (enforced by the atomic conditional UPDATE itself).
func (s *Service) MarkSuccess(ctx context.Context, id, actorID string) error {
	n, err := s.repo.MarkSuccess(ctx, id, actorID)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpx.Conflict("Permintaan ini bukan milik Anda, atau sudah diproses.")
	}
	return nil
}

// MarkRejected implements withdrawalclient.Client — pending|processing -> failed (§2.7); any
// active superadmin, not gated by tenant-suspension status.
func (s *Service) MarkRejected(ctx context.Context, id, actorID, reason string) error {
	if reason == "" {
		return httpx.Validation("Alasan penolakan wajib diisi.")
	}
	n, err := s.repo.MarkRejected(ctx, id, actorID, reason)
	if err != nil {
		return err
	}
	if n == 0 {
		return httpx.Conflict("Permintaan ini sudah mencapai status akhir.")
	}
	return nil
}

// claimableBalance (§2.6, internal validation only — never its own UI figure) = AvailableBalance
// minus this store's currently-processing withdrawals ("spoken for" but not yet gateway-debited).
func (s *Service) claimableBalance(ctx context.Context, storeID string) (int64, error) {
	avail, err := s.AvailableBalance(ctx, storeID)
	if err != nil {
		return 0, err
	}
	processing, err := s.repo.SumProcessingByStore(ctx, storeID)
	if err != nil {
		return 0, err
	}
	return avail - processing, nil
}

func toClientList(rows []domain.Withdrawal) []withdrawalclient.Withdrawal {
	out := make([]withdrawalclient.Withdrawal, 0, len(rows))
	for _, w := range rows {
		out = append(out, withdrawalclient.Withdrawal{
			ID: w.ID, StoreID: w.StoreID, Amount: w.Amount, Bank: w.Bank, Account: w.Account, Holder: w.Holder,
			Status: w.Status, RequestedBy: w.RequestedBy, ProcessedBy: w.ProcessedBy,
			ClaimedAt: w.ClaimedAt, ProcessedAt: w.ProcessedAt, RejectedReason: w.RejectedReason,
			CreatedAt: w.CreatedAt,
		})
	}
	return out
}
