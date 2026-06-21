// Package media wires the media (image upload) module and exposes its HTTP handler.
package media

import (
	authcontract "github.com/elkasir/api/internal/modules/auth/contracts"
	"github.com/elkasir/api/internal/modules/media/application"
	"github.com/elkasir/api/internal/modules/media/presentation"
	"github.com/elkasir/api/internal/platform/storage"
)

// Module is the assembled media module.
type Module struct {
	Handler *presentation.Handler
}

// New assembles the media module: service (over object storage) → handler.
// store boleh nil → upload nonaktif (handler mengembalikan error yang jelas).
func New(store *storage.Client, auth authcontract.Authenticator) *Module {
	svc := application.NewService(store)
	return &Module{Handler: presentation.NewHandler(svc, auth)}
}
