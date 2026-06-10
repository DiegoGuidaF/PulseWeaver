package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/collate"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type UserView struct {
	ID                 ids.UserID  `db:"id"`
	Username           string      `db:"username"`
	DisplayName        string      `db:"display_name"`
	Email              string      `db:"email"`
	Role               auth.Role   `db:"role"`
	MustChangePassword bool        `db:"must_change_password"`
	BypassHostCheck    bool        `db:"bypass_host_check"`
	CreatedBy          *ids.UserID `db:"created_by"`
	CreatedAt          time.Time   `db:"created_at"`
}

func (r *Repository) GetAllUsers(ctx context.Context) ([]UserView, error) {
	const query = `
		SELECT
			u.id, u.username, u.display_name, u.email, u.role,
			u.must_change_password, u.created_by, u.created_at,
			COALESCE(uhs.bypass_host_check, 0) AS bypass_host_check
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.created_at DESC
	`
	var users []UserView
	if err := r.db.SelectContext(ctx, &users, query); err != nil {
		return nil, fmt.Errorf("get all users: %w", err)
	}
	if users == nil {
		users = []UserView{}
	}
	return users, nil
}

func (r *Repository) ListUserAccessRows(ctx context.Context) ([]httpapi.UserListItem, error) {
	type userRow struct {
		ID              ids.UserID `db:"id"`
		DisplayName     string     `db:"display_name"`
		UserName        string     `db:"username"`
		Email           string     `db:"email"`
		Role            auth.Role  `db:"role"`
		BypassHostCheck bool       `db:"bypass_host_check"`
		HostCount       int        `db:"host_count"`
		DeviceCount     int        `db:"device_count"`
		LiveIPCount     int        `db:"live_ip_count"`
	}
	const userQuery = `
		SELECT
			u.id,
			u.display_name,
			u.username,
			u.email,
			u.role,
			uhs.bypass_host_check as bypass_host_check,
			CASE WHEN COALESCE(uhs.bypass_host_check, 0) = 1 THEN
				(SELECT COUNT(*) FROM hosts)
			ELSE
				(
					SELECT COUNT(DISTINCT h.id)
					FROM hosts h
					WHERE h.id IN (
						SELECT hgm.host_id FROM host_group_members hgm
						JOIN user_allowed_host_groups uahg ON uahg.host_group_id = hgm.host_group_id
						WHERE uahg.user_id = u.id
					)
				)
			END as host_count,
			(
				SELECT count(d.id) FROM devices d
				WHERE d.owner_id = u.id
				AND d.deleted_at is NULL
			) as device_count,
			(
				SELECT count(a.id) FROM addresses a
				JOIN devices d on a.device_id = d.id
				WHERE d.owner_id = u.id
				and a.device_id = d.id
				and a.is_enabled = 1
			) as live_ip_count
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.display_name
	`
	var userRows []userRow
	if err := r.db.SelectContext(ctx, &userRows, userQuery); err != nil {
		return nil, fmt.Errorf("list user access rows: %w", err)
	}

	type grantRow struct {
		UserID     ids.UserID      `db:"user_id"`
		GroupID    ids.HostGroupID `db:"group_id"`
		GroupName  string          `db:"group_name"`
		GroupColor string          `db:"group_color"`
		GroupIcon  string          `db:"group_icon"`
	}
	const grantQuery = `
		SELECT uahg.user_id, hg.id AS group_id, hg.name AS group_name,
		       hg.color AS group_color, COALESCE(hg.icon, '') AS group_icon
		FROM user_allowed_host_groups uahg
		JOIN host_groups hg ON hg.id = uahg.host_group_id
		ORDER BY uahg.user_id, hg.name
	`
	var grantRows []grantRow
	if err := r.db.SelectContext(ctx, &grantRows, grantQuery); err != nil {
		return nil, fmt.Errorf("list user access rows groups: %w", err)
	}

	grantsByUser := collate.GroupByMap(grantRows,
		func(gr grantRow) ids.UserID { return gr.UserID },
		func(gr grantRow) httpapi.GroupSummary {
			return httpapi.GroupSummary{
				Id:    gr.GroupID.Int64(),
				Name:  gr.GroupName,
				Color: gr.GroupColor,
				Icon:  gr.GroupIcon,
			}
		},
	)

	rows := make([]httpapi.UserListItem, len(userRows))
	for i, ur := range userRows {
		rows[i] = httpapi.UserListItem{
			Id:               ur.ID.Int64(),
			Username:         ur.UserName,
			DisplayName:      ur.DisplayName,
			Role:             httpapi.UserRole(ur.Role),
			BypassHostCheck:  ur.BypassHostCheck,
			DeviceCount:      ur.DeviceCount,
			HostCount:        ur.HostCount,
			LiveAddressCount: ur.LiveIPCount,
			Groups:           collate.OrEmpty(grantsByUser[ur.ID]),
		}
	}
	return rows, nil
}

func (r *Repository) GetUserAccessDetail(ctx context.Context, userID ids.UserID) (httpapi.UserAccessDetail, error) {
	// Q1: user base info + bypass flag
	type userRow struct {
		ID              ids.UserID `db:"id"`
		DisplayName     string     `db:"display_name"`
		Username        string     `db:"username"`
		Email           string     `db:"email"`
		Role            auth.Role  `db:"role"`
		BypassHostCheck bool       `db:"bypass_host_check"`
	}
	const userQuery = `
		SELECT u.id, u.display_name, u.username, u.email, u.role,
		       COALESCE(uhs.bypass_host_check, 0) AS bypass_host_check
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.id = ? AND u.deleted_at IS NULL
	`
	var ur userRow
	if err := r.db.GetContext(ctx, &ur, userQuery, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpapi.UserAccessDetail{}, auth.ErrUserNotFound
		}
		return httpapi.UserAccessDetail{}, fmt.Errorf("get user access detail user: %w", err)
	}

	// Q2: all groups with grant flag and their member hosts
	type groupRow struct {
		GroupID          ids.HostGroupID `db:"group_id"`
		GroupName        string          `db:"group_name"`
		GroupIcon        string          `db:"group_icon"`
		GroupColor       string          `db:"group_color"`
		GroupDescription *string         `db:"group_description"`
		GroupCreatedAt   time.Time       `db:"group_created_at"`
		GroupUpdatedAt   time.Time       `db:"group_updated_at"`
		Granted          bool            `db:"granted"`
		HostID           *ids.HostID     `db:"host_id"`
		HostFQDN         *string         `db:"host_fqdn"`
	}
	const groupQuery = `
		SELECT
			hg.id   AS group_id,
			hg.name AS group_name,
			hg.icon AS group_icon,
			hg.color AS group_color,
			hg.description AS group_description,
			hg.created_at AS group_created_at,
			hg.updated_at AS group_updated_at,
			CASE WHEN uahg.user_id IS NOT NULL THEN 1 ELSE 0 END AS granted,
			h.id   AS host_id,
			h.fqdn AS host_fqdn
		FROM host_groups hg
		LEFT JOIN user_allowed_host_groups uahg ON uahg.host_group_id = hg.id AND uahg.user_id = ?
		LEFT JOIN host_group_members hgm ON hgm.host_group_id = hg.id
		LEFT JOIN hosts h ON h.id = hgm.host_id
		ORDER BY hg.name, h.fqdn
	`
	var groupRows []groupRow
	if err := r.db.SelectContext(ctx, &groupRows, groupQuery, userID); err != nil {
		return httpapi.UserAccessDetail{}, fmt.Errorf("get user access detail groups: %w", err)
	}

	groups := collate.Collapse(groupRows,
		func(gr groupRow) ids.HostGroupID { return gr.GroupID },
		func(gr groupRow) httpapi.SubjectGroupDetail {
			return httpapi.SubjectGroupDetail{
				Id:              gr.GroupID.Int64(),
				Name:            gr.GroupName,
				Icon:            gr.GroupIcon,
				Color:           gr.GroupColor,
				Description:     gr.GroupDescription,
				CreatedAt:       httpapi.UTCTime(gr.GroupCreatedAt),
				UpdatedAt:       httpapi.UTCTime(gr.GroupUpdatedAt),
				Granted:         gr.Granted,
				Hosts:           []httpapi.HostSummary{},
			}
		},
		func(gr groupRow) (httpapi.HostSummary, bool) {
			if gr.HostID == nil || gr.HostFQDN == nil {
				return httpapi.HostSummary{}, false
			}
			return httpapi.HostSummary{
				Id:   (*gr.HostID).Int64(),
				Fqdn: *gr.HostFQDN,
			}, true
		},
		func(g *httpapi.SubjectGroupDetail, h httpapi.HostSummary) { g.Hosts = append(g.Hosts, h) },
	)

	// Q3: devices owned by the user
	deviceViews, err := r.GetDevicesByUser(ctx, userID)
	if err != nil {
		return httpapi.UserAccessDetail{}, fmt.Errorf("get user access detail devices: %w", err)
	}
	devices := make([]httpapi.DeviceListItem, len(deviceViews))
	for i := range deviceViews {
		devices[i] = httpapi.DeviceListItem{
			Id:               deviceViews[i].ID.Int64(),
			Name:             deviceViews[i].Name,
			ApiKeyPrefix:     deviceViews[i].KeyPrefix,
			Icon:             deviceViews[i].Icon,
			LiveAddressCount: deviceViews[i].LiveAddressCount,
		}
	}

	return httpapi.UserAccessDetail{
		Id:              ur.ID.Int64(),
		Username:        ur.Username,
		DisplayName:     ur.DisplayName,
		BypassHostCheck: ur.BypassHostCheck,
		Role:            httpapi.UserRole(ur.Role),
		Email:           new(openapi_types.Email(ur.Email)),
		Groups:          groups,
		Devices:         devices,
	}, nil
}
