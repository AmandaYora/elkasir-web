// Package application holds the withdrawal module's use cases.
package application

import (
	"context"
	"time"

	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/withdrawal/domain"
	"github.com/elkasir/api/internal/modules/withdrawal/infrastructure"
	"github.com/elkasir/api/internal/platform/id"
)

type Service struct{ repo *infrastructure.Repo }

func NewService(repo *infrastructure.Repo) *Service { return &Service{repo: repo} }

// DTO is the API representation of a withdrawal (camelCase).
type DTO struct {
	ID          string    `json:"id"`
	Amount      int64     `json:"amount"`
	Bank        string    `json:"bank"`
	Account     string    `json:"account"`
	Holder      string    `json:"holder"`
	Status      string    `json:"status"`
	Reference   string    `json:"reference,omitempty"`
	RequestedBy string    `json:"requestedBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

func toDTO(w domain.Withdrawal) DTO {
	return DTO{
		ID: w.ID, Amount: w.Amount, Bank: w.Bank, Account: w.Account, Holder: w.Holder,
		Status: w.Status, Reference: w.Reference, RequestedBy: w.RequestedBy,
		CreatedAt: w.CreatedAt,
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

func (s *Service) Create(ctx context.Context, p authcontract.Principal, in domain.Input) (DTO, error) {
	if err := in.Validate(); err != nil {
		return DTO{}, err
	}
	wid := id.New()
	w, err := s.repo.Create(ctx, p.StoreID, wid, p.SubjectID, in)
	if err != nil {
		return DTO{}, err
	}
	return toDTO(w), nil
}
