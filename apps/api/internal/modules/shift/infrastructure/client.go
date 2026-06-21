// Contract implementation for shiftclient.Client. Tx-aware: every query goes through
// uow.Q(ctx) so reads can join a transaction opened by an orchestrator.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	shiftclient "github.com/elkasir/api/internal/modules/shift/contracts"
	"github.com/elkasir/api/internal/platform/uow"
)

type apiClient struct{ uow *uow.Manager }

// NewClient membuat implementasi shiftclient.Client.
func NewClient(m *uow.Manager) shiftclient.Client { return &apiClient{uow: m} }

var _ shiftclient.Client = (*apiClient)(nil)

func (c *apiClient) CurrentOpenID(ctx context.Context, storeID string) (string, error) {
	sh, err := c.uow.Q(ctx).GetOpenShift(ctx, storeID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil // tak ada shift terbuka → "" (penjualan online tetap tercatat)
	}
	if err != nil {
		return "", err
	}
	return sh.ID, nil
}
