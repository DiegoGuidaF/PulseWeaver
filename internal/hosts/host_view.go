package hosts

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/collate"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// GetAllHostsWithGroups returns every host with all of its group memberships,
// for the host list page.
func (r *Repository) GetAllHostsWithGroups(ctx context.Context) (httpapi.HostListResponse, error) {
	type row struct {
		ID         ids.HostID       `db:"id"`
		FQDN       string           `db:"fqdn"`
		GroupID    *ids.HostGroupID `db:"group_id"`
		GroupName  *string          `db:"group_name"`
		GroupColor *string          `db:"group_color"`
		GroupIcon  *string          `db:"group_icon"`
	}
	const query = `
		SELECT
			kh.id    AS id,
			kh.fqdn  AS fqdn,
			hg.id    AS group_id,
			hg.name  AS group_name,
			hg.color AS group_color,
			hg.icon  AS group_icon
		FROM hosts kh
		LEFT JOIN host_group_members hgm ON hgm.host_id = kh.id
		LEFT JOIN host_groups hg ON hg.id = hgm.host_group_id
		ORDER BY kh.fqdn, hg.name
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return httpapi.HostListResponse{}, fmt.Errorf("get hosts with groups: %w", err)
	}

	hostsList := collate.Collapse(rows,
		func(rw row) ids.HostID { return rw.ID },
		func(rw row) httpapi.Host {
			return httpapi.Host{Id: rw.ID.Int64(), Fqdn: rw.FQDN, Groups: []httpapi.GroupSummary{}}
		},
		func(rw row) (httpapi.GroupSummary, bool) {
			if rw.GroupID == nil {
				return httpapi.GroupSummary{}, false
			}
			return httpapi.GroupSummary{
				Id:    (*rw.GroupID).Int64(),
				Name:  *rw.GroupName,
				Color: *rw.GroupColor,
				Icon:  *rw.GroupIcon,
			}, true
		},
		func(h *httpapi.Host, g httpapi.GroupSummary) { h.Groups = append(h.Groups, g) },
	)
	return httpapi.HostListResponse{Hosts: hostsList}, nil
}
