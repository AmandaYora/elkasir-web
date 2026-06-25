// Package application holds the cashmovement module's use cases.
package application

import (
	"context"
	"database/sql"
	"errors"
	"time"

	businessrules "github.com/elkasir/api/internal/domain"
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/cashmovement/domain"
	"github.com/elkasir/api/internal/modules/cashmovement/infrastructure"
	shiftclient "github.com/elkasir/api/internal/modules/shift/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
	"github.com/elkasir/api/internal/platform/id"
)

// Service mengandalkan shiftClient untuk atribusi shift terbuka (lintas-modul via contract).
type Service struct {
	repo   *infrastructure.Repo
	shifts shiftclient.Client
}

func NewService(repo *infrastructure.Repo, shiftClient shiftclient.Client) *Service {
	return &Service{repo: repo, shifts: shiftClient}
}

// DTO is the API representation of a cash movement (camelCase).
type DTO struct {
	ID         string    `json:"id"`
	ShiftID    string    `json:"shiftId,omitempty"`
	Type       string    `json:"type"`
	Amount     int64     `json:"amount"`
	Notes      string    `json:"notes,omitempty"`
	CreatedBy  string    `json:"createdBy,omitempty"`
	ApprovedBy string    `json:"approvedBy,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func toDTO(m sqlcgen.CashMovement) DTO {
	return DTO{
		ID: m.ID, ShiftID: m.ShiftID.String, Type: string(m.Type), Amount: m.Amount,
		Notes: m.Notes.String, CreatedBy: m.CreatedBy.String, ApprovedBy: m.ApprovedBy.String,
		CreatedAt: m.CreatedAt,
	}
}

func (s *Service) List(ctx context.Context, f domain.ListFilter) ([]DTO, int64, error) {
	rows, err := s.repo.List(ctx, f.StoreID, int32(f.Limit), int32(f.Offset))
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, f.StoreID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, toDTO(m))
	}
	return out, total, nil
}

func (s *Service) Create(ctx context.Context, p authcontract.Principal, in domain.Input) (DTO, error) {
	storeID := p.StoreID

	if err := in.Validate(); err != nil {
		return DTO{}, err
	}

	// Kebijakan kontrol biaya operasional (di server, bukan klien). Supervisor/admin yang
	// menjalankan langsung memenuhi syarat persetujuan (override otomatis).
	if in.Type == domain.TypeExpense {
		policy := s.controlPolicy(ctx, storeID)
		if policy.ExpenseNeedsApproval(in.Amount) && !p.IsSupervisorOrAdmin() && in.TrimmedApprovedBy() == "" {
			return DTO{}, httpx.Forbidden("Biaya melebihi plafon; butuh persetujuan supervisor (PIN).")
		}
	}

	openShift, err := s.shifts.CurrentOpenID(ctx, storeID)
	if err != nil {
		return DTO{}, err
	}

	createdBy := ""
	if p.Actor == authcontract.ActorStaff {
		createdBy = p.SubjectID
	}

	cmID := id.New()
	if err := s.repo.Create(ctx, infrastructure.CreateInput{
		ID: cmID, StoreID: storeID, ShiftID: openShift, Type: in.Type, Amount: in.Amount,
		Notes: in.Notes, CreatedBy: createdBy, ApprovedBy: in.ApprovedBy,
	}); err != nil {
		return DTO{}, err
	}

	m, err := s.repo.Get(ctx, storeID, cmID)
	if errors.Is(err, sql.ErrNoRows) {
		return DTO{}, httpx.NotFound("Kas tidak ditemukan.")
	}
	if err != nil {
		return DTO{}, err
	}
	return toDTO(m), nil
}

func (s *Service) controlPolicy(ctx context.Context, storeID string) businessrules.ControlPolicy {
	st, err := s.repo.Settings(ctx, storeID)
	if err != nil {
		// default aman bila settings belum ada
		return businessrules.ControlPolicy{MaxOperationalExpense: 200000}
	}
	return businessrules.ControlPolicy{MaxOperationalExpense: st.MaxOperationalExpense}
}
