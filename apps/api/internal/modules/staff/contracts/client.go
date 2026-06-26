// Package staffclient is the PUBLIC contract of the staff module: it lets orchestrators
// (transaction, shift) verify a supervisor approval PIN without touching the staff tables.
package staffclient

import "context"

// Supervisor identifies the active supervisor whose PIN matched (recorded as the approver).
type Supervisor struct {
	ID   string
	Name string
}

// Client is the read-only contract published by the staff module.
type Client interface {
	// ResolveSupervisorByPIN returns the active supervisor in the store whose approval PIN
	// matches `pin` (ok=false when none match / pin empty). Used to authorize a cashier's
	// over-threshold action with an in-place supervisor PIN.
	ResolveSupervisorByPIN(ctx context.Context, storeID, pin string) (Supervisor, bool, error)
}
