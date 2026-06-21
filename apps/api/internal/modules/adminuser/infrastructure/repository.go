// Package infrastructure holds the adminuser module's persistence (sqlc + database/sql).
package infrastructure

import (
	"context"
	"database/sql"
	"strings"

	"github.com/elkasir/api/internal/modules/adminuser/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/httpx"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func (r *Repo) List(ctx context.Context, f domain.ListFilter) ([]domain.AdminUser, error) {
	rows, err := r.q.ListAdminUsers(ctx, f.StoreID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdminUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntity(row))
	}
	return out, nil
}

// Get fetches one admin user scoped to the store (sql.ErrNoRows when absent).
func (r *Repo) Get(ctx context.Context, storeID, id string) (domain.AdminUser, error) {
	row, err := r.q.GetAdminUserScoped(ctx, sqlcgen.GetAdminUserScopedParams{ID: id, StoreID: storeID})
	if err != nil {
		return domain.AdminUser{}, err
	}
	return toEntity(row), nil
}

func (r *Repo) Create(ctx context.Context, storeID, id, email, username, passwordHash string, in domain.CreateInput) error {
	role, err := parseRole(in.Role)
	if err != nil {
		return err
	}
	status, err := parseStatus(in.Status)
	if err != nil {
		return err
	}
	return r.q.CreateAdminUser(ctx, sqlcgen.CreateAdminUserParams{
		ID:           id,
		StoreID:      storeID,
		Name:         strings.TrimSpace(in.Name),
		Email:        email,
		Username:     sql.NullString{String: username, Valid: username != ""},
		PasswordHash: passwordHash,
		Role:         role,
		Status:       status,
	})
}

func (r *Repo) Update(ctx context.Context, storeID, id, email string, in domain.UpdateInput) error {
	role, err := parseRole(in.Role)
	if err != nil {
		return err
	}
	status, err := parseStatus(in.Status)
	if err != nil {
		return err
	}
	return r.q.UpdateAdminUser(ctx, sqlcgen.UpdateAdminUserParams{
		Name:    strings.TrimSpace(in.Name),
		Email:   email,
		Role:    role,
		Status:  status,
		ID:      id,
		StoreID: storeID,
	})
}

func (r *Repo) UpdatePassword(ctx context.Context, storeID, id, passwordHash string) error {
	return r.q.UpdateAdminUserPassword(ctx, sqlcgen.UpdateAdminUserPasswordParams{
		PasswordHash: passwordHash,
		ID:           id,
		StoreID:      storeID,
	})
}

func (r *Repo) Delete(ctx context.Context, storeID, id string) error {
	return r.q.DeleteAdminUser(ctx, sqlcgen.DeleteAdminUserParams{ID: id, StoreID: storeID})
}

func toEntity(u sqlcgen.AdminUser) domain.AdminUser {
	d := domain.AdminUser{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Username:  u.Username.String,
		Role:      string(u.Role),
		Status:    string(u.Status),
		CreatedAt: u.CreatedAt,
	}
	if u.LastActiveAt.Valid {
		t := u.LastActiveAt.Time
		d.LastActiveAt = &t
	}
	return d
}

func parseRole(s string) (sqlcgen.AdminUsersRole, error) {
	switch s {
	case "owner":
		return sqlcgen.AdminUsersRoleOwner, nil
	case "admin":
		return sqlcgen.AdminUsersRoleAdmin, nil
	case "manager":
		return sqlcgen.AdminUsersRoleManager, nil
	case "viewer":
		return sqlcgen.AdminUsersRoleViewer, nil
	default:
		return "", httpx.Validation("Role harus 'owner', 'admin', 'manager', atau 'viewer'.")
	}
}

func parseStatus(s string) (sqlcgen.AdminUsersStatus, error) {
	switch s {
	case "", "active":
		return sqlcgen.AdminUsersStatusActive, nil
	case "inactive":
		return sqlcgen.AdminUsersStatusInactive, nil
	default:
		return "", httpx.Validation("Status harus 'active' atau 'inactive'.")
	}
}
