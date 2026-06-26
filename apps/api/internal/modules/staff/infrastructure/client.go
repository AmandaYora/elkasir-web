package infrastructure

import (
	"context"
	"strings"

	staffclient "github.com/elkasir/api/internal/modules/staff/contracts"
	"github.com/elkasir/api/internal/platform/security"
)

// contractClient implements staffclient.Client for cross-module supervisor-PIN verification.
type contractClient struct {
	repo *Repo
}

// NewClient builds the staff contract client (supervisor-PIN resolution).
func NewClient(repo *Repo) staffclient.Client { return &contractClient{repo: repo} }

var _ staffclient.Client = (*contractClient)(nil)

func (c *contractClient) ResolveSupervisorByPIN(ctx context.Context, storeID, pin string) (staffclient.Supervisor, bool, error) {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return staffclient.Supervisor{}, false, nil
	}
	rows, err := c.repo.ListSupervisorPins(ctx, storeID)
	if err != nil {
		return staffclient.Supervisor{}, false, err
	}
	for _, r := range rows {
		if security.VerifyPassword(r.PinHash, pin) {
			return staffclient.Supervisor{ID: r.ID, Name: r.Name}, true, nil
		}
	}
	return staffclient.Supervisor{}, false, nil
}
