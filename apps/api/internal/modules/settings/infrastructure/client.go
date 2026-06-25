package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	settingsclient "github.com/elkasir/api/internal/modules/settings/contracts"
	"github.com/elkasir/api/internal/platform/db/sqlcgen"
	uow "github.com/elkasir/api/internal/platform/uow"
)

// contractClient mengimplementasikan settingsclient.Client untuk pembaca lintas-modul.
// Membaca via uow.Q(ctx) agar konsisten dengan transaksi aktif (bila ada).
type contractClient struct {
	uow *uow.Manager
}

// NewClient membuat implementasi settingsclient.Client.
func NewClient(m *uow.Manager) settingsclient.Client { return &contractClient{uow: m} }

var _ settingsclient.Client = (*contractClient)(nil)

func (c *contractClient) Get(ctx context.Context, storeID string) (settingsclient.Settings, error) {
	st, err := c.uow.Q(ctx).GetSettingsByStore(ctx, storeID)
	if errors.Is(err, sql.ErrNoRows) {
		d := defaultSettings
		return mapSettings(d), nil
	}
	if err != nil {
		return settingsclient.Settings{}, err
	}
	return mapSettings(st), nil
}

func mapSettings(st sqlcgen.Setting) settingsclient.Settings {
	return settingsclient.Settings{
		MaxDiscountPercent:    st.MaxDiscountPercent,
		MaxOperationalExpense: st.MaxOperationalExpense,
		CashVarianceTolerance: st.CashVarianceTolerance,
		FeatureSelfOrder:      st.FeatureSelfOrder,
		FeatureQris:           st.FeatureQris,
		FeaturePayAtCashier:   st.FeaturePayAtCashier,
		TaxEnabled:            st.TaxEnabled,
		TaxPercent:            st.TaxPercent,
		ServicePercent:        st.ServicePercent,
	}
}
