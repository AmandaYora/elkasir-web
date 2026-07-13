// Package infrastructure holds the platformuser module's persistence (sqlc + database/sql).
package infrastructure

import (
	"context"
	"database/sql"

	"github.com/elkasir/api/internal/modules/platformuser/domain"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
)

type Repo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewRepo(db *sql.DB, q *sqlcgen.Queries) *Repo { return &Repo{db: db, q: q} }

func (r *Repo) List(ctx context.Context) ([]domain.PlatformUser, error) {
	rows, err := r.q.ListPlatformUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PlatformUser, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomain(u))
	}
	return out, nil
}

func (r *Repo) Get(ctx context.Context, id string) (domain.PlatformUser, error) {
	u, err := r.q.GetPlatformUser(ctx, id)
	if err != nil {
		return domain.PlatformUser{}, err
	}
	return toDomain(u), nil
}

func (r *Repo) Create(ctx context.Context, id, name, email, passwordHash string) error {
	return r.q.CreatePlatformUser(ctx, sqlcgen.CreatePlatformUserParams{
		ID: id, Name: name, Email: email, PasswordHash: passwordHash,
		Status: sqlcgen.PlatformUsersStatusActive,
	})
}

// SetStatus returns rows-affected (0 = not found).
func (r *Repo) SetStatus(ctx context.Context, id, status string) (int64, error) {
	return r.q.SetPlatformUserStatus(ctx, sqlcgen.SetPlatformUserStatusParams{
		ID: id, Status: sqlcgen.PlatformUsersStatus(status),
	})
}

func (r *Repo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	return r.q.UpdatePlatformUserPassword(ctx, sqlcgen.UpdatePlatformUserPasswordParams{
		ID: id, PasswordHash: passwordHash,
	})
}

func toDomain(u sqlcgen.PlatformUser) domain.PlatformUser {
	return domain.PlatformUser{
		ID: u.ID, Name: u.Name, Email: u.Email, Status: string(u.Status), CreatedAt: u.CreatedAt,
	}
}
