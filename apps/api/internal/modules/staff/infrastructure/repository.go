// Package infrastructure holds the staff module's persistence (sqlc + database/sql).
package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"github.com/elkasir/api/internal/modules/staff/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func (r *Repo) List(ctx context.Context, f domain.ListFilter) ([]domain.Staff, error) {
	rows, err := r.q.ListStaff(ctx, f.StoreID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Staff, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntity(row))
	}
	return out, nil
}

// Get fetches one staff member scoped to the store (sql.ErrNoRows when absent).
func (r *Repo) Get(ctx context.Context, storeID, id string) (domain.Staff, error) {
	row, err := r.q.GetStaffScoped(ctx, sqlcgen.GetStaffScopedParams{ID: id, StoreID: storeID})
	if err != nil {
		return domain.Staff{}, err
	}
	return toEntity(row), nil
}

func (r *Repo) Create(ctx context.Context, storeID, id, passwordHash string, in domain.CreateInput) error {
	role, err := mapRole(in.Role)
	if err != nil {
		return err
	}
	status, err := mapStatus(in.Status)
	if err != nil {
		return err
	}
	return r.q.CreateStaff(ctx, sqlcgen.CreateStaffParams{
		ID:           id,
		StoreID:      storeID,
		Name:         strings.TrimSpace(in.Name),
		Username:     strings.TrimSpace(in.Username),
		Email:        nullStr(in.Email),
		PasswordHash: passwordHash,
		Role:         role,
		Status:       status,
	})
}

func (r *Repo) Update(ctx context.Context, storeID, id string, in domain.UpdateInput) error {
	role, err := mapRole(in.Role)
	if err != nil {
		return err
	}
	status, err := mapStatus(in.Status)
	if err != nil {
		return err
	}
	return r.q.UpdateStaff(ctx, sqlcgen.UpdateStaffParams{
		Name:     strings.TrimSpace(in.Name),
		Username: strings.TrimSpace(in.Username),
		Email:    nullStr(in.Email),
		Role:     role,
		Status:   status,
		ID:       id,
		StoreID:  storeID,
	})
}

func (r *Repo) UpdatePassword(ctx context.Context, storeID, id, passwordHash string) error {
	return r.q.UpdateStaffPassword(ctx, sqlcgen.UpdateStaffPasswordParams{PasswordHash: passwordHash, ID: id, StoreID: storeID})
}

// SetPin menyimpan hash PIN supervisor (atau mengosongkannya bila pinHash kosong → NULL).
func (r *Repo) SetPin(ctx context.Context, storeID, id, pinHash string) error {
	return r.q.UpdateStaffPin(ctx, sqlcgen.UpdateStaffPinParams{PinHash: nullStr(pinHash), ID: id, StoreID: storeID})
}

// SupervisorPin adalah baris supervisor aktif yang punya PIN (hash dipakai untuk verifikasi).
type SupervisorPin struct {
	ID      string
	Name    string
	PinHash string
}

// ListSupervisorPins mengembalikan supervisor aktif yang sudah menyetel PIN (untuk verifikasi).
func (r *Repo) ListSupervisorPins(ctx context.Context, storeID string) ([]SupervisorPin, error) {
	rows, err := r.q.ListSupervisorPins(ctx, storeID)
	if err != nil {
		return nil, err
	}
	out := make([]SupervisorPin, 0, len(rows))
	for _, row := range rows {
		out = append(out, SupervisorPin{ID: row.ID, Name: row.Name, PinHash: row.PinHash.String})
	}
	return out, nil
}

func (r *Repo) Delete(ctx context.Context, storeID, id string) error {
	return r.q.DeleteStaff(ctx, sqlcgen.DeleteStaffParams{ID: id, StoreID: storeID})
}

func toEntity(s sqlcgen.Staff) domain.Staff {
	return domain.Staff{
		ID:        s.ID,
		Name:      s.Name,
		Username:  s.Username,
		Email:     s.Email.String,
		Role:      string(s.Role),
		Status:    string(s.Status),
		HasPin:    s.PinHash.Valid,
		CreatedAt: s.CreatedAt,
	}
}

// mapRole maps a role string to the sqlcgen enum (Validation when invalid).
func mapRole(role string) (sqlcgen.StaffRole, error) {
	switch role {
	case string(sqlcgen.StaffRoleCashier):
		return sqlcgen.StaffRoleCashier, nil
	case string(sqlcgen.StaffRoleSupervisor):
		return sqlcgen.StaffRoleSupervisor, nil
	default:
		return "", httpx.Validation("Peran harus 'cashier' atau 'supervisor'.")
	}
}

// mapStatus maps a status string to the sqlcgen enum; empty defaults to 'active'.
func mapStatus(status string) (sqlcgen.StaffStatus, error) {
	switch status {
	case "", string(sqlcgen.StaffStatusActive):
		return sqlcgen.StaffStatusActive, nil
	case string(sqlcgen.StaffStatusInactive):
		return sqlcgen.StaffStatusInactive, nil
	default:
		return "", httpx.Validation("Status harus 'active' atau 'inactive'.")
	}
}

func nullStr(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
