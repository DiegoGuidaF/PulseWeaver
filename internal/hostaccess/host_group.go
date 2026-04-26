package hostaccess

import (
	"strconv"
	"time"
)

type HostGroupID int64

func (id HostGroupID) Int64() int64   { return int64(id) }
func (id HostGroupID) String() string { return strconv.FormatInt(int64(id), 10) }

type HostGroup struct {
	ID          HostGroupID   `db:"id"`
	Name        string        `db:"name"`
	Color       *string       `db:"color"`
	Description *string       `db:"description"`
	Icon        *string       `db:"icon"`
	UpdatedAt   time.Time     `db:"updated_at"`
	CreatedAt   time.Time     `db:"created_at"`
	HostIDs     []KnownHostID `db:"-"`
}

// SameDefinitionAs reports whether two groups would produce identical rows
// in host_groups + host_group_members. Used by the reconciler to skip no-op
// updates and avoid spurious updated_at bumps.
func (g HostGroup) SameDefinitionAs(other HostGroup) bool {
	if g.Name != other.Name {
		return false
	}
	if !equalStringPtr(g.Color, other.Color) {
		return false
	}
	if !equalStringPtr(g.Description, other.Description) {
		return false
	}
	if !equalStringPtr(g.Icon, other.Icon) {
		return false
	}
	return sameKnownHostIDs(g.HostIDs, other.HostIDs)
}

func sameKnownHostIDs(a, b []KnownHostID) bool {
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

// KnownHostRef is a lightweight reference returned inside group member lists.
type KnownHostRef struct {
	ID   KnownHostID
	FQDN string
	Icon *string
}
