package useraccess

import "context"

// Observer is implemented by any component that must react when user host-access
// grants change (e.g. policy.Service refreshing its cache).
type Observer interface {
	OnHostAccessChanged(ctx context.Context)
}
