package useraccess

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/collate"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	sq "github.com/Masterminds/squirrel"
)

// SubjectRow is one access subject's display identity: the role and
// bypass-host-check badge and the host groups granted to them. Role and bypass
// are rendered, never interpreted — the deciding domains are auth and the
// policy engine.
type SubjectRow struct {
	ID              ids.UserID
	DisplayName     string
	Role            httpapi.UserRole
	BypassHostCheck bool
	HostGroups      []httpapi.GroupSummary
}

// SubjectRows returns every non-deleted user as an access subject, optionally
// narrowed to one. It carries no device aggregates: a caller composing a fleet
// already holds the device rows those counts derive from.
func (r *Repository) SubjectRows(ctx context.Context, userID *ids.UserID) ([]SubjectRow, error) {
	type row struct {
		ID              ids.UserID       `db:"id"`
		DisplayName     string           `db:"display_name"`
		Role            auth.Role        `db:"role"`
		BypassHostCheck bool             `db:"bypass_host_check"`
		GroupID         *ids.HostGroupID `db:"group_id"`
		GroupName       *string          `db:"group_name"`
		GroupColor      *string          `db:"group_color"`
		GroupIcon       *string          `db:"group_icon"`
	}

	builder := sq.Select(
		"u.id",
		"u.display_name",
		"u.role",
		"COALESCE(uhs.bypass_host_check, 0) AS bypass_host_check",
		"hg.id    AS group_id",
		"hg.name  AS group_name",
		"hg.color AS group_color",
		"hg.icon  AS group_icon",
	).
		From("users u").
		LeftJoin("user_host_settings uhs ON uhs.user_id = u.id").
		LeftJoin("user_allowed_host_groups uahg ON uahg.user_id = u.id").
		LeftJoin("host_groups hg ON hg.id = uahg.host_group_id").
		Where(sq.Eq{"u.deleted_at": nil})
	if userID != nil {
		builder = builder.Where(sq.Eq{"u.id": *userID})
	}
	// u.id breaks display-name ties: display names are not unique, so without it
	// two subjects sharing one can swap places between refetches.
	query, args, err := builder.OrderBy("u.display_name", "u.id", "hg.name").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build subject rows query: %w", err)
	}

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("get subject rows: %w", err)
	}

	subjects := collate.Collapse(rows,
		func(rw row) ids.UserID { return rw.ID },
		func(rw row) SubjectRow {
			return SubjectRow{
				ID:              rw.ID,
				DisplayName:     rw.DisplayName,
				Role:            auth.RoleToAPI(rw.Role),
				BypassHostCheck: rw.BypassHostCheck,
				HostGroups:      []httpapi.GroupSummary{},
			}
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
		func(s *SubjectRow, g httpapi.GroupSummary) { s.HostGroups = append(s.HostGroups, g) },
	)
	return subjects, nil
}
