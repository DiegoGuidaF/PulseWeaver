package hostaccess

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type HostGroup struct {
	ID          ids.HostGroupID   `db:"id"`
	Name        string            `db:"name"`
	Color       string            `db:"color"`
	Icon        string            `db:"icon"`
	Description *string           `db:"description"`
	UpdatedAt   time.Time         `db:"updated_at"`
	CreatedAt   time.Time         `db:"created_at"`
	HostIDs     []ids.KnownHostID `db:"-"`
}

// SameDefinitionAs reports whether two groups would produce identical rows
// in host_groups + host_group_members. Used by the reconciler to skip no-op
// updates and avoid spurious updated_at bumps.
func (g HostGroup) SameDefinitionAs(other HostGroup) bool {
	if g.Name != other.Name {
		return false
	}
	if g.Color != other.Color {
		return false
	}
	if g.Icon != other.Icon {
		return false
	}
	if !equalStringPtr(g.Description, other.Description) {
		return false
	}
	return sameKnownHostIDs(g.HostIDs, other.HostIDs)
}

func sameKnownHostIDs(a, b []ids.KnownHostID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
