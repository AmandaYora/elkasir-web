// Contract implementation for tableclient.Client. Tx-aware: every query goes through
// uow.Q(ctx) so reads can join a transaction opened by an orchestrator.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	tableclient "github.com/elkasir/api/internal/modules/table/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/uow"
)

type apiClient struct{ uow *uow.Manager }

// NewClient membuat implementasi tableclient.Client.
func NewClient(m *uow.Manager) tableclient.Client { return &apiClient{uow: m} }

var _ tableclient.Client = (*apiClient)(nil)

func toClientTable(t sqlcgen.DiningTable) tableclient.Table {
	return tableclient.Table{
		ID: t.ID, StoreID: t.StoreID, Code: t.Code, Name: t.Name, Area: t.Area, Status: string(t.Status),
	}
}

func (c *apiClient) FindByCode(ctx context.Context, storeSlug, code string) (tableclient.Table, error) {
	t, err := c.uow.Q(ctx).FindTableByStoreSlugAndCode(ctx, sqlcgen.FindTableByStoreSlugAndCodeParams{
		Slug: storeSlug, Code: code,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return tableclient.Table{}, tableclient.ErrNotFound
	}
	if err != nil {
		return tableclient.Table{}, err
	}
	return toClientTable(t), nil
}

func (c *apiClient) GetByID(ctx context.Context, storeID, id string) (tableclient.Table, error) {
	t, err := c.uow.Q(ctx).GetTable(ctx, sqlcgen.GetTableParams{ID: id, StoreID: storeID})
	if errors.Is(err, sql.ErrNoRows) {
		return tableclient.Table{}, tableclient.ErrNotFound
	}
	if err != nil {
		return tableclient.Table{}, err
	}
	return toClientTable(t), nil
}

func (c *apiClient) ListAll(ctx context.Context, storeID string) ([]tableclient.Table, error) {
	rows, err := c.uow.Q(ctx).ListTables(ctx, storeID)
	if err != nil {
		return nil, err
	}
	out := make([]tableclient.Table, 0, len(rows))
	for _, t := range rows {
		out = append(out, toClientTable(t))
	}
	return out, nil
}
