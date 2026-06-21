// Contract implementation for productclient.Client. Tx-aware: every query goes through
// uow.Q(ctx) so Decrease joins the transaction opened by an orchestrator (atomic with
// transaction creation).
package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	productclient "github.com/elkasir/api/internal/modules/product/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	"github.com/elkasir/api/internal/platform/uow"
)

type apiClient struct{ uow *uow.Manager }

// NewClient builds an implementation of productclient.Client.
func NewClient(m *uow.Manager) productclient.Client { return &apiClient{uow: m} }

var _ productclient.Client = (*apiClient)(nil)

func (c *apiClient) GetForSale(ctx context.Context, storeID, productID string) (productclient.ProductSale, error) {
	row, err := c.uow.Q(ctx).GetProductForSale(ctx, sqlcgen.GetProductForSaleParams{ID: productID, StoreID: storeID})
	if errors.Is(err, sql.ErrNoRows) {
		return productclient.ProductSale{}, productclient.ErrNotFound
	}
	if err != nil {
		return productclient.ProductSale{}, err
	}
	return productclient.ProductSale{
		ID: row.ID, Name: row.Name, Category: row.Category, Price: row.Price,
		Active: row.Status == sqlcgen.ProductsStatusActive,
	}, nil
}

func (c *apiClient) ListActive(ctx context.Context, storeID string) ([]productclient.ProductSale, error) {
	rows, err := c.uow.Q(ctx).ListActiveProducts(ctx, storeID)
	if err != nil {
		return nil, err
	}
	out := make([]productclient.ProductSale, 0, len(rows))
	for _, p := range rows {
		out = append(out, productclient.ProductSale{
			ID: p.ID, Name: p.Name, Category: p.Category, Price: p.Price, ImageURL: p.ImageUrl, Active: true,
		})
	}
	return out, nil
}

// Decrease atomically reduces stock (UPDATE ... WHERE stock >= qty). 0 rows affected =
// product missing or insufficient stock → ErrInsufficientStock.
func (c *apiClient) Decrease(ctx context.Context, storeID, productID string, qty int32) error {
	n, err := c.uow.Q(ctx).DecrementStock(ctx, sqlcgen.DecrementStockParams{Qty: qty, ID: productID, StoreID: storeID})
	if err != nil {
		return err
	}
	if n == 0 {
		return productclient.ErrInsufficientStock
	}
	return nil
}
