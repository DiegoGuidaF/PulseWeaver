package hostaccess

import "context"

// Observer is implemented by any component that must react when host access
// configuration changes (e.g. policy.Service refreshing its cache).
type Observer interface {
	OnHostAccessChanged(ctx context.Context)
}
